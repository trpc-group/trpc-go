// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package admin provides management capabilities for trpc services,
// including but not limited to health checks, logging, performance monitoring, RPCZ, etc.
package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"trpc.group/trpc-go/trpc-go/internal/reuseport"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/healthcheck"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

// ServiceName is the service name of admin service.
const ServiceName = "admin"

// Patterns.
const (
	patternCmds          = "/cmds"
	patternVersion       = "/version"
	patternLoglevel      = "/cmds/loglevel"
	patternConfig        = "/cmds/config"
	patternHealthCheck   = "/is_healthy/"
	patternRPCZSpansList = "/cmds/rpcz/spans"
	patternRPCZSpanGet   = "/cmds/rpcz/spans/"
)

// Pprof patterns.
const (
	pprofPprof   = "/debug/pprof/"
	pprofCmdline = "/debug/pprof/cmdline"
	pprofProfile = "/debug/pprof/profile"
	pprofSymbol  = "/debug/pprof/symbol"
	pprofTrace   = "/debug/pprof/trace"
)

// Return parameters.
const (
	retErrCode    = "errorcode"
	retMessage    = "message"
	errCodeServer = 1
)

// Server structure provides utilities related to administration.
// It implements the server.Service interface.
type Server struct {
	config *configuration
	server *http.Server

	router      *router
	healthCheck *healthcheck.HealthCheck

	closeOnce sync.Once
	closeErr  error
}

// NewServer returns a new admin Server.
func NewServer(opts ...Option) *Server {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	s := &Server{
		config:      cfg,
		healthCheck: healthcheck.New(healthcheck.WithStatusWatchers(healthcheck.GetWatchers())),
	}
	if !cfg.skipServe {
		s.router = s.configRouter(newRouter())
	}
	return s
}

func (s *Server) configRouter(r *router) *router {
	r.add(patternCmds, s.handleCmds)         // Admin Command List.
	r.add(patternVersion, s.handleVersion)   // Framework version.
	r.add(patternLoglevel, s.handleLogLevel) // View/Set the log level of the framework.
	r.add(patternConfig, s.handleConfig)     // View framework configuration files.
	r.add(patternHealthCheck,
		http.StripPrefix(patternHealthCheck,
			http.HandlerFunc(s.handleHealthCheck),
		).ServeHTTP,
	) // Health check.

	r.add(patternRPCZSpansList, s.handleRPCZSpansList)
	r.add(patternRPCZSpanGet, s.handleRPCZSpanGet)

	r.add(pprofPprof, pprof.Index)
	r.add(pprofCmdline, pprof.Cmdline)
	r.add(pprofProfile, pprof.Profile)
	r.add(pprofSymbol, pprof.Symbol)
	r.add(pprofTrace, pprof.Trace)

	for pattern, handler := range pattern2Handler {
		r.add(pattern, handler)
	}

	// Delete the router registered with http.DefaultServeMux.
	// Avoid causing security problems: https://github.com/golang/go/issues/22085.
	err := unregisterHandlers(
		[]string{
			pprofPprof,
			pprofCmdline,
			pprofProfile,
			pprofSymbol,
			pprofTrace,
		},
	)
	if err != nil {
		log.Errorf("failed to unregister pprof handlers from http.DefaultServeMux, err: %+v", err)
	}
	return r
}

// Register implements server.Service.
func (s *Server) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	// The admin service does not need to do anything in this registration function.
	return nil
}

// RegisterHealthCheck registers a new service and returns two functions, one for unregistering the service and one for
// updating the status of the service.
func (s *Server) RegisterHealthCheck(
	serviceName string,
) (unregister func(), update func(healthcheck.Status), err error) {
	update, err = s.healthCheck.Register(serviceName)
	return func() {
		s.healthCheck.Unregister(serviceName)
	}, update, err
}

// Serve starts the admin HTTP server.
func (s *Server) Serve() error {
	cfg := s.config
	if cfg.skipServe {
		return nil
	}
	if cfg.enableTLS {
		return errors.New("admin service does not support tls")
	}

	const network = "tcp"
	ln, err := s.listen(network, cfg.addr)
	if err != nil {
		return err
	}

	s.server = &http.Server{
		Addr:         ln.Addr().String(),
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
		Handler:      s.router,
	}
	if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close shuts down server.
func (s *Server) Close(ch chan struct{}) error {
	pid := os.Getpid()
	s.closeOnce.Do(s.close)
	log.Infof("process:%d, admin server, closed", pid)
	if ch != nil {
		ch <- struct{}{}
	}
	return s.closeErr
}

// WatchStatus HealthCheck proxy, registers health status watcher for service.
func (s *Server) WatchStatus(serviceName string, onStatusChanged func(healthcheck.Status)) {
	s.healthCheck.Watch(serviceName, onStatusChanged)
}

var pattern2Handler map[string]http.HandlerFunc

// HandleFunc registers the handler function for the given pattern.
// Each time NewServer is called, all handlers registered through HandleFunc will be in effect.
func HandleFunc(patten string, handler http.HandlerFunc) {
	pattern2Handler[patten] = handler
}

// HandleFunc registers the handler function for the given pattern.
func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	_ = s.router.add(pattern, handler)
}

func (s *Server) listen(network, addr string) (net.Listener, error) {
	ln, err := s.obtainListener(network, addr)
	if err != nil {
		return nil, fmt.Errorf("get admin listener error: %w", err)
	}
	if ln == nil {
		ln, err = reuseport.Listen(network, addr)
		if err != nil {
			return nil, fmt.Errorf("admin reuseport listen error: %w", err)
		}
	}
	if err := transport.SaveListener(ln); err != nil {
		return nil, fmt.Errorf("save admin listener error: %w", err)
	}
	return ln, nil
}

func (s *Server) obtainListener(network, addr string) (net.Listener, error) {
	ok, _ := strconv.ParseBool(os.Getenv(transport.EnvGraceRestart)) // Ignore error caused by messy values.
	if !ok {
		return nil, nil
	}
	pln, err := transport.GetPassedListener(network, addr)
	if err != nil {
		return nil, err
	}
	ln, ok := pln.(net.Listener)
	if !ok {
		return nil, fmt.Errorf("the passed listener %T is not of type net.Listener", pln)
	}
	return ln, nil
}

func (s *Server) close() {
	if s.server == nil {
		return
	}
	s.closeErr = s.server.Close()
}

// ErrorOutput normalizes the error output.
func ErrorOutput(w http.ResponseWriter, error string, code int) {
	ret := newDefaultRes()
	ret[retErrCode] = code
	ret[retMessage] = error
	_ = json.NewEncoder(w).Encode(ret)
}

// handleCmds gives a list of all currently available administrative commands.
func (s *Server) handleCmds(w http.ResponseWriter, r *http.Request) {
	setCommonHeaders(w)

	list := s.router.list()
	cmds := make([]string, 0, len(list))
	for _, item := range list {
		cmds = append(cmds, item.pattern)
	}
	ret := newDefaultRes()
	ret["cmds"] = cmds
	_ = json.NewEncoder(w).Encode(ret)
}

// newDefaultRes returns admin Default output format.
func newDefaultRes() map[string]interface{} {
	return map[string]interface{}{
		retErrCode: 0,
		retMessage: "",
	}
}

// handleVersion gives the current version number.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	setCommonHeaders(w)

	ret := newDefaultRes()
	ret["version"] = s.config.version
	_ = json.NewEncoder(w).Encode(ret)
}

// getLevel returns the level of logger's output stream.
func getLevel(logger log.Logger, output string) string {
	return log.LevelStrings[logger.GetLevel(output)]
}

// handleLogLevel returns the output level of the current logger.
func (s *Server) handleLogLevel(w http.ResponseWriter, r *http.Request) {
	setCommonHeaders(w)

	if err := r.ParseForm(); err != nil {
		ErrorOutput(w, err.Error(), errCodeServer)
		return
	}

	name := r.Form.Get("logger")
	if name == "" {
		name = "default"
	}
	output := r.Form.Get("output")
	if output == "" {
		output = "0" // If no output is given in the request parameters, the first output is used.
	}

	logger := log.Get(name)
	if logger == nil {
		ErrorOutput(w, fmt.Sprintf("logger %s not found", name), errCodeServer)
		return
	}

	ret := newDefaultRes()
	if r.Method == http.MethodGet {
		ret["level"] = getLevel(logger, output)
		_ = json.NewEncoder(w).Encode(ret)
	} else if r.Method == http.MethodPut {
		ret["prelevel"] = getLevel(logger, output)
		level := r.PostForm.Get("value")
		logger.SetLevel(output, log.LevelNames[level])
		ret["level"] = getLevel(logger, output)
		_ = json.NewEncoder(w).Encode(ret)
	}
}

// handleConfig outputs the content of the current configuration file.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	buf, err := os.ReadFile(s.config.configPath)
	if err != nil {
		ErrorOutput(w, err.Error(), errCodeServer)
		return
	}

	unmarshaler := config.GetUnmarshaler("yaml")
	if unmarshaler == nil {
		ErrorOutput(w, "cannot find yaml unmarshaler", errCodeServer)
		return
	}

	conf := make(map[string]interface{})
	if err = unmarshaler.Unmarshal(buf, &conf); err != nil {
		ErrorOutput(w, err.Error(), errCodeServer)
		return
	}
	ret := newDefaultRes()
	ret["content"] = conf
	_ = json.NewEncoder(w).Encode(ret)
}

// handleHealthCheck handles health check requests.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	check := s.healthCheck.CheckServer
	if service := r.URL.Path; service != "" {
		check = func() healthcheck.Status {
			return s.healthCheck.CheckService(service)
		}
	}
	switch check() {
	case healthcheck.Serving:
		w.WriteHeader(http.StatusOK)
	case healthcheck.NotServing:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleRPCZSpansList returns #xxx span from r by url "http://ip:port/cmds/rpcz/spans?num=xxx".
func (s *Server) handleRPCZSpansList(w http.ResponseWriter, r *http.Request) {
	num, err := parseNumParameter(r.URL)
	if err != nil {
		newResponse("", err).print(w)
		return
	}
	var content string
	for i, span := range rpcz.GlobalRPCZ.BatchQuery(num) {
		content += fmt.Sprintf("%d:\n", i+1)
		content += span.PrintSketch("  ")
	}
	newResponse(content, nil).print(w)
}

// handleRPCZSpanGet returns span with id from r by url "http://ip:port/cmds/rpcz/span/{id}".
func (s *Server) handleRPCZSpanGet(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParameter(r.URL)
	if err != nil {
		newResponse("", err).print(w)
		return
	}

	span, ok := rpcz.GlobalRPCZ.Query(rpcz.SpanID(id))
	if !ok {
		newResponse("", errs.New(errCodeServer, fmt.Sprintf("cannot find span-id: %d", id))).print(w)
		return
	}
	newResponse(span.PrintDetail(""), nil).print(w)
}

func parseIDParameter(url *url.URL) (id int64, err error) {
	id, err = strconv.ParseInt(strings.TrimPrefix(url.Path, patternRPCZSpanGet), 10, 64)
	if err != nil {
		return id, fmt.Errorf("undefined command, please follow http://ip:port/cmds/rpcz/span/{id}), %w", err)
	}
	if id < 0 {
		return id, fmt.Errorf("span_id: %d can not be negative", id)
	}
	return id, err
}

func parseNumParameter(url *url.URL) (int, error) {
	queryNum := url.Query().Get("num")
	if queryNum == "" {
		const defaultNum = 10
		return defaultNum, nil
	}

	num, err := strconv.Atoi(queryNum)
	if err != nil {
		return num, fmt.Errorf("http://ip:port/cmds/rpcz?num=xxx, xxx must be a integer, %w", err)
	}
	if num < 0 {
		return num, errors.New("http://ip:port/cmds/rpcz?num=xxx, xxx must be a non-negative integer")
	}
	return num, nil
}

type response struct {
	content string
	err     error
}

func newResponse(content string, err error) response {
	return response{
		content: content,
		err:     err,
	}
}
func (r response) print(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.err != nil {
		e := struct {
			ErrCode    trpcpb.TrpcRetCode `json:"err-code"`
			ErrMessage string             `json:"err-message"`
		}{
			ErrCode:    errs.Code(r.err),
			ErrMessage: errs.Msg(r.err),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(e); err != nil {
			log.Trace("json.Encode failed when write to http.ResponseWriter")
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(r.content)); err != nil {
		log.Trace("http.ResponseWriter write error")
	}
}

func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}
