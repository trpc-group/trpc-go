//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package admin implements some common management functions.
package admin

import (
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

	jsoniter "github.com/json-iterator/go"
	reuseport "github.com/kavu/go_reuseport"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/healthcheck"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	// The pprof functionality supported by the admin package relies on the imported net/http/pprof package.
	// However, the imported net/http/pprof package implicitly registers HTTP handlers for
	// "/debug/pprof/", "/debug/pprof/cmdline", "/debug/pprof/profile", "/debug/pprof/symbol", "/debug/pprof/trace"
	// in http.DefaultServeMux in its init function. This implicit behavior is too subtle and may contribute to people
	// inadvertently leaving such endpoints open, and may cause security problems：https://github.com/golang/go/issues/22085
	// if people use http.DefaultServeMux. So we decide to reset default serve mux to remove pprof registration.
	// This requires making sure that people are not using http.DefaultServeMux before we reset it.
	// In most cases, this works, which is guaranteed by the execution order of the init function.
	// If you need to enable pprof on http.DefaultServeMux you need to
	// register it explicitly after importing the admin package:
	//
	// http.DefaultServeMux.HandleFunc("/debug/pprof/", pprof.Index)
	// http.DefaultServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	// http.DefaultServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	// http.DefaultServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	// http.DefaultServeMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	//
	// Simply importing the net/http/pprof package anonymously will not work.
	http.DefaultServeMux = http.NewServeMux()
}

// ServiceName is the service name of admin service.
const ServiceName = "admin"

var (
	pattenCmds         = "/cmds"
	pattenVersion      = "/version"
	pattenLoglevel     = "/cmds/loglevel"
	pattenConfig       = "/cmds/config"
	patternHealthCheck = "/is_healthy/"

	patternRPCZSpansList = "/cmds/rpcz/spans"
	patternRPCZSpanGet   = "/cmds/rpcz/spans/"

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// return param.
var (
	ReturnErrCodeParam = "errorcode"
	ReturnMessageParam = "message"
	ErrCodeServer      = 1
)

// Server admin manage server，implements server.Service.
type Server struct {
	config      *adminConfig
	server      *http.Server
	closeOnce   sync.Once
	closeErr    error
	router      Router
	healthCheck *healthcheck.HealthCheck
}

// NewTrpcAdminServer creates a new AdminServer.
func NewTrpcAdminServer(opts ...Option) *Server {
	cfg := loadDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	s := &Server{
		config:      cfg,
		healthCheck: healthcheck.New(healthcheck.WithStatusWatchers(healthcheck.GetWatchers())),
	}
	if !cfg.skipServe {
		s.initRouter()
	}
	return s
}

// inner router.
var defaultRouter = NewRouter()

// init at least once defaultRouter.
var once sync.Once

// initialization.
func (s *Server) initRouter() {
	once.Do(
		func() {
			defaultRouter.Config(pattenCmds, s.handleCmds).Desc("Admin Command List")
			defaultRouter.Config(pattenVersion, s.handleVersion).Desc("Framework version")
			defaultRouter.Config(pattenLoglevel, s.handleLogLevel).Desc("View/Set the log level of the framework")
			defaultRouter.Config(pattenConfig, s.handleConfig).Desc("View framework configuration files")
			defaultRouter.Config(patternHealthCheck,
				http.StripPrefix(patternHealthCheck,
					http.HandlerFunc(s.handleHealthCheck),
				).ServeHTTP,
			).Desc("Health check")

			defaultRouter.Config(patternRPCZSpansList, s.handleRPCZSpansList)
			defaultRouter.Config(patternRPCZSpanGet, s.handleRPCZSpanGet)

			defaultRouter.Config("/debug/pprof/", pprof.Index)
			defaultRouter.Config("/debug/pprof/cmdline", pprof.Cmdline)
			defaultRouter.Config("/debug/pprof/profile", pprof.Profile)
			defaultRouter.Config("/debug/pprof/symbol", pprof.Symbol)
			defaultRouter.Config("/debug/pprof/trace", pprof.Trace)
			s.router = defaultRouter
		},
	)
}

// Register implements server.Service.
func (s *Server) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	// return nil， server.Server.Register, All business implementation interfaces will be registered in all services
	// (TrpcAdminServer.Register will also be called).
	return nil
}

// RegisterHealthCheck registers a new service and return two functions, one for unregistering the service and one for
// updating the status of the service.
func (s *Server) RegisterHealthCheck(
	serviceName string,
) (unregister func(), update func(healthcheck.Status), err error) {
	update, err = s.healthCheck.Register(serviceName)
	return func() {
		s.healthCheck.Unregister(serviceName)
	}, update, err
}

// Serve start up http Server.
func (s *Server) Serve() error {
	cfg := s.config
	if cfg.skipServe {
		return nil
	}
	if cfg.enableTLS {
		return errors.New("not support yet")
	}

	ln, err := s.listen(protocol.TCP, cfg.getAddr())
	if err != nil {
		return err
	}

	log.Infof("admin service launch success, %s: %s, serving ...", ln.Addr().Network(), ln.Addr().String())

	s.server = &http.Server{
		Addr:         ln.Addr().String(),
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
		Handler:      s.router,
	}
	// Restricted access to the internal/poll.ErrNetClosing type necessitates comparing a string literal.
	const closeError = "use of closed network connection"
	if err := s.server.Serve(ln); err != nil &&
		err != http.ErrServerClosed && !strings.Contains(err.Error(), closeError) {
		return err
	}
	return nil
}

// Close shut down server.
func (s *Server) Close(ch chan struct{}) error {
	pid := os.Getpid()
	s.closeOnce.Do(s.close)
	log.Infof("process: %d, admin server, closed", pid)
	if ch != nil {
		ch <- struct{}{}
	}
	return s.closeErr
}

// WatchStatus HealthCheck proxy, registers health status watcher for service.
func (s *Server) WatchStatus(serviceName string, onStatusChanged func(healthcheck.Status)) {
	s.healthCheck.Watch(serviceName, onStatusChanged)
}

// HandleFunc registers custom service interface.
func HandleFunc(patten string, handler func(w http.ResponseWriter, r *http.Request)) *RouterHandler {
	return defaultRouter.Config(patten, handler)
}

func (s *Server) getListener(network, addr string) (net.Listener, error) {
	value := os.Getenv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(value) // ignore error with messy values for compatibility
	if !ok {
		return nil, nil
	}
	pln, err := transport.GetPassedListener(network, addr)
	if err != nil {
		return nil, err
	}
	ln, ok := pln.(net.Listener)
	if !ok {
		return nil, fmt.Errorf("invalid net.Listener")
	}
	return ln, nil
}

func (s *Server) listen(network, addr string) (net.Listener, error) {
	ln, err := s.getListener(network, addr)
	if err != nil {
		return nil, fmt.Errorf("get admin listener error: %w", err)
	}
	if ln == nil {
		ln, err = reuseport.Listen(network, addr)
		if err != nil {
			return nil, fmt.Errorf("admin reuseport listen error: %w", err)
		}
	}
	err = transport.SaveListener(ln)
	if err != nil {
		return nil, fmt.Errorf("save admin listener error: %w", err)
	}
	return ln, nil
}

func (s *Server) close() {
	if s.server == nil {
		return
	}
	s.closeErr = s.server.Close()
}

// ErrorOutput Unified error output.
func ErrorOutput(w http.ResponseWriter, error string, code int) {
	var ret = newDefaultRes()
	ret[ReturnErrCodeParam] = code
	ret[ReturnMessageParam] = error

	_ = json.NewEncoder(w).Encode(ret)
}

// handleCmds Admin Command List.
func (s *Server) handleCmds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	list := s.router.List()
	cmds := make([]string, 0, len(list))
	for _, item := range list {
		cmds = append(cmds, item.GetPatten())
	}
	var ret = newDefaultRes()
	ret["cmds"] = cmds

	_ = json.NewEncoder(w).Encode(ret)
}

// newDefaultRes admin Default output format.
func newDefaultRes() map[string]interface{} {
	return map[string]interface{}{
		ReturnErrCodeParam: 0,
		ReturnMessageParam: "",
	}
}

// handleVersion handle version number,
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	ret := map[string]interface{}{
		ReturnErrCodeParam: 0,
		ReturnMessageParam: "",
		"version":          s.config.version,
	}
	_ = json.NewEncoder(w).Encode(ret)
}

// getLevel returns the level of logger's output stream.
func getLevel(logger log.Logger, output string) string {
	level := logger.GetLevel(output)
	return log.LevelStrings[level]
}

// handleLogLevel returns logger's level.
func (s *Server) handleLogLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut}, ", "))
		w.WriteHeader(http.StatusMethodNotAllowed)
		ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := r.ParseForm(); err != nil {
		ErrorOutput(w, err.Error(), ErrCodeServer)
		return
	}

	name := r.Form.Get("logger")
	if name == "" {
		name = "default"
	}
	output := r.Form.Get("output")
	if output == "" {
		output = "0" // don't have output, the first output，ordinary users can only configure one.
	}

	logger := log.Get(name)
	if logger == nil {
		ErrorOutput(w, "logger not found", ErrCodeServer)
		return
	}

	var ret = newDefaultRes()
	if r.Method == http.MethodGet {
		ret["level"] = getLevel(logger, output)
		_ = json.NewEncoder(w).Encode(ret)
	} else if r.Method == http.MethodPut {
		level := r.PostForm.Get("value")

		ret["prelevel"] = getLevel(logger, output)
		logger.SetLevel(output, log.LevelNames[level])
		ret["level"] = getLevel(logger, output)

		_ = json.NewEncoder(w).Encode(ret)
	}
}

// handleConfig configuration file content query.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	buf, err := os.ReadFile(s.config.configPath)
	if err != nil {
		ErrorOutput(w, err.Error(), ErrCodeServer)
		return
	}

	unmarshaler := config.GetUnmarshaler("yaml")
	if unmarshaler == nil {
		ErrorOutput(w, "cannot find yaml unmarshaler", ErrCodeServer)
		return
	}

	conf := map[interface{}]interface{}{}
	if err = unmarshaler.Unmarshal(buf, &conf); err != nil {
		ErrorOutput(w, err.Error(), ErrCodeServer)
		return
	}

	var ret = newDefaultRes()
	ret["content"] = conf

	_ = json.NewEncoder(w).Encode(ret)
}

// handleHealthCheck handles health check requests.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

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
			ErrCode    int    `json:"err-code"`
			ErrMessage string `json:"err-message"`
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

// handleRPCZSpansList return #xxx span from r by url "http://ip:port/cmds/rpcz/spans?num=xxx".
func (s *Server) handleRPCZSpansList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

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

// handleRPCZSpanGet return span with id from r by url "http://ip:port/cmds/rpcz/span/{id}".
func (s *Server) handleRPCZSpanGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := parseIDParameter(r.URL)
	if err != nil {
		newResponse("", err).print(w)
		return
	}

	span, ok := rpcz.GlobalRPCZ.Query(rpcz.SpanID(id))
	if !ok {
		newResponse("", errs.New(ErrCodeServer, fmt.Sprintf("cannot find span-id: %d", id))).print(w)
		return
	}
	newResponse(span.PrintDetail(""), nil).print(w)
}

func parseIDParameter(url *url.URL) (id int64, err error) {
	id, err = strconv.ParseInt(strings.TrimPrefix(url.Path, patternRPCZSpanGet), 10, 64)
	if err != nil {
		return id, fmt.Errorf("undefined command, please follow http://ip:port/cmds/rpcz/spans/{id}), %w", err)
	}
	if id < 0 {
		return id, fmt.Errorf("span_id: %d can not be negative", id)
	}
	return id, err
}
