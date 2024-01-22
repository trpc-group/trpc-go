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

package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

// MaxCloseWaitTime is the max waiting time for closing services.
const MaxCloseWaitTime = 10 * time.Second

// Service is the interface that provides services.
type Service interface {
	// Register registers a proto service.
	Register(serviceDesc interface{}, serviceImpl interface{}) error
	// Serve starts serving.
	Serve() error
	// Close stops serving.
	Close(chan struct{}) error
}

// FilterFunc reads reqBody, parses it, and returns a filter.Chain for server stub.
type FilterFunc func(reqBody interface{}) (filter.ServerChain, error)

// Method provides the information of an RPC Method.
type Method struct {
	Name     string
	Func     func(svr interface{}, ctx context.Context, f FilterFunc) (rspBody interface{}, err error)
	Bindings []*restful.Binding
}

// ServiceDesc describes a proto service.
type ServiceDesc struct {
	ServiceName  string
	HandlerType  interface{}
	Methods      []Method
	Streams      []StreamDesc
	StreamHandle StreamHandle
}

// StreamDesc describes a server stream.
type StreamDesc struct {
	// StreamName is the name of stream.
	StreamName string
	// Handler is a stream handler.
	Handler StreamHandlerWapper
	// ServerStreams indicates whether it's server streaming.
	ServerStreams bool
	// ClientStreams indicates whether it's client streaming.
	ClientStreams bool
}

// Handler is the default handler.
type Handler func(ctx context.Context, f FilterFunc) (rspBody interface{}, err error)

// StreamHandlerWapper is server stream handler wrapper.
// The input param srv should be an implementation of server stream proto service.
// The input param stream is used by srv.
type StreamHandlerWapper func(srv interface{}, stream Stream) error

// StreamHandler is server stream handler.
type StreamHandler func(stream Stream) error

// Stream is the interface that defines server stream api.
type Stream interface {
	// Context is context of server stream.
	Context() context.Context
	// SendMsg sends streaming data.
	SendMsg(m interface{}) error
	// RecvMsg receives streaming data.
	RecvMsg(m interface{}) error
}

// service is an implementation of Service
type service struct {
	ctx            context.Context    // context of this service
	cancel         context.CancelFunc // function that cancels this service
	opts           *Options           // options of this service
	handlers       map[string]Handler // rpcname => handler
	streamHandlers map[string]StreamHandler
	streamInfo     map[string]*StreamServerInfo
	activeCount    int64 // active requests count for graceful close if set MaxCloseWaitTime
}

// New creates a service.
// It will use transport.DefaultServerTransport unless Option WithTransport()
// is called to replace its transport.ServerTransport plugin.
var New = func(opts ...Option) Service {
	o := defaultOptions()
	s := &service{
		opts:           o,
		handlers:       make(map[string]Handler),
		streamHandlers: make(map[string]StreamHandler),
		streamInfo:     make(map[string]*StreamServerInfo),
	}
	for _, o := range opts {
		o(s.opts)
	}
	o.Transport = attemptSwitchingTransport(o)
	if !s.opts.handlerSet {
		// if handler is not set, pass the service (which implements Handler interface)
		// as handler of transport plugin.
		s.opts.ServeOptions = append(s.opts.ServeOptions, transport.WithHandler(s))
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

// Serve implements Service, starting serving.
func (s *service) Serve() error {
	pid := os.Getpid()

	// make sure ListenAndServe succeeds before Naming Service Registry.
	if err := s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
		log.Errorf("process:%d service:%s ListenAndServe fail:%v", pid, s.opts.ServiceName, err)
		return err
	}

	if s.opts.Registry != nil {
		opts := []registry.Option{
			registry.WithAddress(s.opts.Address),
		}
		if isGraceful, isParental := checkProcessStatus(); isGraceful && !isParental {
			// If current process is the child process forked for graceful restart,
			// service should notify the registry plugin of graceful restart event.
			// The registry plugin might handle registry results according to this event.
			// For example, repeat registry cause error but the plugin would consider it's ok
			// according to this event.
			opts = append(opts, registry.WithEvent(registry.GracefulRestart))
		}
		if err := s.opts.Registry.Register(s.opts.ServiceName, opts...); err != nil {
			// if registry fails, service needs to be closed and error should be returned.
			log.Errorf("process:%d, service:%s register fail:%v", pid, s.opts.ServiceName, err)
			return err
		}
	}

	log.Infof("process:%d, %s service:%s launch success, %s:%s, serving ...",
		pid, s.opts.protocol, s.opts.ServiceName, s.opts.network, s.opts.Address)

	report.ServiceStart.Incr()
	<-s.ctx.Done()
	return nil
}

// Handle implements transport.Handler.
// service itself is passed to its transport plugin as a transport handler.
// This is like a callback function that would be called by service's transport plugin.
func (s *service) Handle(ctx context.Context, reqBuf []byte) (rspBuf []byte, err error) {
	if s.opts.MaxCloseWaitTime > s.opts.CloseWaitTime || s.opts.MaxCloseWaitTime > MaxCloseWaitTime {
		atomic.AddInt64(&s.activeCount, 1)
		defer atomic.AddInt64(&s.activeCount, -1)
	}

	// if server codec is empty, simply returns error.
	if s.opts.Codec == nil {
		log.ErrorContextf(ctx, "server codec empty")
		report.ServerCodecEmpty.Incr()
		return nil, errors.New("server codec empty")
	}

	msg := codec.Message(ctx)
	span := rpcz.SpanFromContext(ctx)
	span.SetAttribute(rpcz.TRPCAttributeFilterNames, s.opts.FilterNames)

	_, end := span.NewChild("DecodeProtocolHead")
	reqBodyBuf, err := s.decode(ctx, msg, reqBuf)
	end.End()

	if err != nil {
		return s.encode(ctx, msg, nil, err)
	}
	// ServerRspErr is already set,
	// since RequestID is acquired, just respond to client.
	if err := msg.ServerRspErr(); err != nil {
		return s.encode(ctx, msg, nil, err)
	}

	rspbody, err := s.handle(ctx, msg, reqBodyBuf)
	if err != nil {
		// no response
		if err == errs.ErrServerNoResponse {
			return nil, err
		}
		// failed to handle, should respond to client with error code,
		// ignore rspBody.
		report.ServiceHandleFail.Incr()
		return s.encode(ctx, msg, nil, err)
	}
	return s.handleResponse(ctx, msg, rspbody)
}

// HandleClose is called when conn is closed.
// Currently, only used for server stream.
func (s *service) HandleClose(ctx context.Context) error {
	if codec.Message(ctx).ServerRspErr() != nil && s.opts.StreamHandle != nil {
		_, err := s.opts.StreamHandle.StreamHandleFunc(ctx, nil, nil, nil)
		return err
	}
	return nil
}

func (s *service) encode(ctx context.Context, msg codec.Msg, rspBodyBuf []byte, e error) (rspBuf []byte, err error) {
	if e != nil {
		log.DebugContextf(
			ctx,
			"service: %s handle err (if caused by health checking, this error can be ignored): %+v",
			s.opts.ServiceName, e)
		msg.WithServerRspErr(e)
	}

	rspBuf, err = s.opts.Codec.Encode(msg, rspBodyBuf)
	if err != nil {
		report.ServiceCodecEncodeFail.Incr()
		log.ErrorContextf(ctx, "service:%s encode fail:%v", s.opts.ServiceName, err)
		return nil, err
	}
	return rspBuf, nil
}

// handleStream handles server stream.
func (s *service) handleStream(ctx context.Context, msg codec.Msg, reqBuf []byte, sh StreamHandler,
	opts *Options) (resbody interface{}, err error) {
	if s.opts.StreamHandle != nil {
		si := s.streamInfo[msg.ServerRPCName()]
		return s.opts.StreamHandle.StreamHandleFunc(ctx, sh, si, reqBuf)
	}
	return nil, errs.NewFrameError(errs.RetServerNoService, "Stream method no Handle")
}

func (s *service) decode(ctx context.Context, msg codec.Msg, reqBuf []byte) ([]byte, error) {
	s.setOpt(msg)
	reqBodyBuf, err := s.opts.Codec.Decode(msg, reqBuf)
	if err != nil {
		report.ServiceCodecDecodeFail.Incr()
		return nil, errs.NewFrameError(errs.RetServerDecodeFail, "service codec Decode: "+err.Error())
	}

	// call setOpt again to avoid some msg infos (namespace, env name, etc.)
	// being modified by request decoding.
	s.setOpt(msg)
	return reqBodyBuf, nil
}

func (s *service) setOpt(msg codec.Msg) {
	msg.WithNamespace(s.opts.Namespace)           // service namespace
	msg.WithEnvName(s.opts.EnvName)               // service environment
	msg.WithSetName(s.opts.SetName)               // service "Set"
	msg.WithCalleeServiceName(s.opts.ServiceName) // from perspective of the service, callee refers to itself
}

func (s *service) handle(ctx context.Context, msg codec.Msg, reqBodyBuf []byte) (interface{}, error) {
	// whether is server streaming RPC
	streamHandler, ok := s.streamHandlers[msg.ServerRPCName()]
	if ok {
		return s.handleStream(ctx, msg, reqBodyBuf, streamHandler, s.opts)
	}
	handler, ok := s.handlers[msg.ServerRPCName()]
	if !ok {
		handler, ok = s.handlers["*"] // wildcard
		if !ok {
			report.ServiceHandleRPCNameInvalid.Incr()
			return nil, errs.NewFrameError(errs.RetServerNoFunc,
				fmt.Sprintf("service handle: rpc name %s invalid, current service:%s",
					msg.ServerRPCName(), msg.CalleeServiceName()))
		}
	}

	var fixTimeout filter.ServerFilter
	if s.opts.Timeout > 0 {
		fixTimeout = mayConvert2NormalTimeout
	}
	timeout := s.opts.Timeout
	if msg.RequestTimeout() > 0 && !s.opts.DisableRequestTimeout { // 可以配置禁用
		if msg.RequestTimeout() < timeout || timeout == 0 { // 取最小值
			fixTimeout = mayConvert2FullLinkTimeout
			timeout = msg.RequestTimeout()
		}
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	newFilterFunc := s.filterFunc(ctx, msg, reqBodyBuf, fixTimeout)
	rspBody, err := handler(ctx, newFilterFunc)
	if err != nil {
		if e, ok := err.(*errs.Error); ok &&
			e.Type == errs.ErrorTypeFramework &&
			e.Code == errs.RetServerFullLinkTimeout {
			err = errs.ErrServerNoResponse
		}
		return nil, err
	}
	if msg.CallType() == codec.SendOnly {
		return nil, errs.ErrServerNoResponse
	}
	return rspBody, nil
}

// handleResponse handles response.
// serialization type is set to msg.SerializationType() by default,
// if serialization type Option is called, serialization type is set by the Option.
// compress type's setting is similar to it.
func (s *service) handleResponse(ctx context.Context, msg codec.Msg, rspBody interface{}) ([]byte, error) {
	// marshal response body

	serializationType := msg.SerializationType()
	if icodec.IsValidSerializationType(s.opts.CurrentSerializationType) {
		serializationType = s.opts.CurrentSerializationType
	}
	span := rpcz.SpanFromContext(ctx)

	_, end := span.NewChild("Marshal")
	rspBodyBuf, err := codec.Marshal(serializationType, rspBody)
	end.End()

	if err != nil {
		report.ServiceCodecMarshalFail.Incr()
		err = errs.NewFrameError(errs.RetServerEncodeFail, "service codec Marshal: "+err.Error())
		// rspBodyBuf will be nil if marshalling fails, respond only error code to client.
		return s.encode(ctx, msg, rspBodyBuf, err)
	}

	// compress response body
	compressType := msg.CompressType()
	if icodec.IsValidCompressType(s.opts.CurrentCompressType) {
		compressType = s.opts.CurrentCompressType
	}

	_, end = span.NewChild("Compress")
	rspBodyBuf, err = codec.Compress(compressType, rspBodyBuf)
	end.End()

	if err != nil {
		report.ServiceCodecCompressFail.Incr()
		err = errs.NewFrameError(errs.RetServerEncodeFail, "service codec Compress: "+err.Error())
		// rspBodyBuf will be nil if compression fails, respond only error code to client.
		return s.encode(ctx, msg, rspBodyBuf, err)
	}

	_, end = span.NewChild("EncodeProtocolHead")
	rspBuf, err := s.encode(ctx, msg, rspBodyBuf, nil)
	end.End()

	return rspBuf, err
}

// filterFunc returns a FilterFunc, which would be passed to server stub to access pre/post filter handling.
func (s *service) filterFunc(
	ctx context.Context,
	msg codec.Msg,
	reqBodyBuf []byte,
	fixTimeout filter.ServerFilter,
) FilterFunc {
	// Decompression, serialization of request body are put into a closure.
	// Both serialization type & compress type can be set.
	// serialization type is set to msg.SerializationType() by default,
	// if serialization type Option is called, serialization type is set by the Option.
	// compress type's setting is similar to it.
	return func(reqBody interface{}) (filter.ServerChain, error) {
		// decompress request body
		compressType := msg.CompressType()
		if icodec.IsValidCompressType(s.opts.CurrentCompressType) {
			compressType = s.opts.CurrentCompressType
		}
		span := rpcz.SpanFromContext(ctx)
		_, end := span.NewChild("Decompress")
		reqBodyBuf, err := codec.Decompress(compressType, reqBodyBuf)
		end.End()
		if err != nil {
			report.ServiceCodecDecompressFail.Incr()
			return nil, errs.NewFrameError(errs.RetServerDecodeFail, "service codec Decompress: "+err.Error())
		}

		// unmarshal request body
		serializationType := msg.SerializationType()
		if icodec.IsValidSerializationType(s.opts.CurrentSerializationType) {
			serializationType = s.opts.CurrentSerializationType
		}
		_, end = span.NewChild("Unmarshal")
		err = codec.Unmarshal(serializationType, reqBodyBuf, reqBody)
		end.End()
		if err != nil {
			report.ServiceCodecUnmarshalFail.Incr()
			return nil, errs.NewFrameError(errs.RetServerDecodeFail, "service codec Unmarshal: "+err.Error())
		}

		if fixTimeout != nil {
			// this heap allocation cannot be avoided unless we change the generated xxx.trpc.go.
			filters := make(filter.ServerChain, len(s.opts.Filters), len(s.opts.Filters)+1)
			copy(filters, s.opts.Filters)
			return append(filters, fixTimeout), nil
		}
		return s.opts.Filters, nil
	}
}

// Register implements Service interface, registering a proto service impl for the service.
func (s *service) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	desc, ok := serviceDesc.(*ServiceDesc)
	if !ok {
		return errors.New("serviceDesc is not *ServiceDesc")
	}
	if desc.StreamHandle != nil {
		s.opts.StreamHandle = desc.StreamHandle
		if s.opts.StreamTransport != nil {
			s.opts.Transport = s.opts.StreamTransport
		}
		// IdleTimeout is not used by server stream, set it to 0.
		s.opts.ServeOptions = append(s.opts.ServeOptions, transport.WithServerIdleTimeout(0))
		err := s.opts.StreamHandle.Init(s.opts)
		if err != nil {
			return err
		}
	}

	if serviceImpl != nil {
		ht := reflect.TypeOf(desc.HandlerType).Elem()
		hi := reflect.TypeOf(serviceImpl)
		if !hi.Implements(ht) {
			return fmt.Errorf("%s not implements interface %s", hi.String(), ht.String())
		}
	}

	var bindings []*restful.Binding
	for _, method := range desc.Methods {
		n := method.Name
		if _, ok := s.handlers[n]; ok {
			return fmt.Errorf("duplicate method name: %s", n)
		}
		h := method.Func
		s.handlers[n] = func(ctx context.Context, f FilterFunc) (rsp interface{}, err error) {
			return h(serviceImpl, ctx, f)
		}
		bindings = append(bindings, method.Bindings...)
	}

	for _, stream := range desc.Streams {
		n := stream.StreamName
		if _, ok := s.streamHandlers[n]; ok {
			return fmt.Errorf("duplicate stream name: %s", n)
		}
		h := stream.Handler
		s.streamInfo[stream.StreamName] = &StreamServerInfo{
			FullMethod:     stream.StreamName,
			IsClientStream: stream.ClientStreams,
			IsServerStream: stream.ServerStreams,
		}
		s.streamHandlers[stream.StreamName] = func(stream Stream) error {
			return h(serviceImpl, stream)
		}
	}
	return s.createOrUpdateRouter(bindings, serviceImpl)
}

func (s *service) createOrUpdateRouter(bindings []*restful.Binding, serviceImpl interface{}) error {
	// If pb option (trpc.api.http) is set，creates a RESTful Router.
	if len(bindings) == 0 {
		return nil
	}
	handler := restful.GetRouter(s.opts.ServiceName)
	if handler != nil {
		if router, ok := handler.(*restful.Router); ok { // A router has already been registered.
			for _, binding := range bindings { // Add binding with a specified service implementation.
				if err := router.AddImplBinding(binding, serviceImpl); err != nil {
					return fmt.Errorf("add impl binding during service registration: %w", err)
				}
			}
			return nil
		}
	}
	// This is the first time of registering the service router, create a new one.
	router := restful.NewRouter(append(s.opts.RESTOptions,
		restful.WithNamespace(s.opts.Namespace),
		restful.WithEnvironment(s.opts.EnvName),
		restful.WithContainer(s.opts.container),
		restful.WithSet(s.opts.SetName),
		restful.WithServiceName(s.opts.ServiceName),
		restful.WithTimeout(s.opts.Timeout),
		restful.WithFilterFunc(func() filter.ServerChain { return s.opts.Filters }))...)
	for _, binding := range bindings {
		if err := router.AddImplBinding(binding, serviceImpl); err != nil {
			return err
		}
	}
	restful.RegisterRouter(s.opts.ServiceName, router)
	return nil
}

// Close closes the service，registry.Deregister will be called.
func (s *service) Close(ch chan struct{}) error {
	pid := os.Getpid()
	if ch == nil {
		ch = make(chan struct{}, 1)
	}
	log.Infof("process:%d, %s service:%s, closing ...", pid, s.opts.protocol, s.opts.ServiceName)

	if s.opts.Registry != nil {
		// When it comes to graceful restart, the parent process will not call registry Deregister(),
		// while the child process would call registry Deregister().
		if isGraceful, isParental := checkProcessStatus(); !(isGraceful && isParental) {
			if err := s.opts.Registry.Deregister(s.opts.ServiceName); err != nil {
				log.Errorf("process:%d, deregister service:%s fail:%v", pid, s.opts.ServiceName, err)
			}
		}
	}
	if remains := s.waitBeforeClose(); remains > 0 {
		log.Infof("process %d service %s remains %d requests before close",
			os.Getpid(), s.opts.ServiceName, remains)
	}

	// this will cancel all children ctx.
	s.cancel()

	timeout := time.Millisecond * 300
	if s.opts.Timeout > timeout { // use the larger one
		timeout = s.opts.Timeout
	}
	if remains := s.waitInactive(timeout); remains > 0 {
		log.Infof("process %d service %s remains %d requests after close",
			os.Getpid(), s.opts.ServiceName, remains)
	}
	log.Infof("process:%d, %s service:%s, closed", pid, s.opts.protocol, s.opts.ServiceName)
	ch <- struct{}{}
	return nil
}

func (s *service) waitBeforeClose() int64 {
	closeWaitTime := s.opts.CloseWaitTime
	if closeWaitTime > MaxCloseWaitTime {
		closeWaitTime = MaxCloseWaitTime
	}
	if closeWaitTime > 0 {
		// After registry.Deregister() is called, sleep a while to let Naming Service (like Polaris) finish
		// updating instance ip list.
		// Otherwise, client request would still arrive while the service had already been closed (Typically, it occurs
		// when k8s updates pods).
		log.Infof("process %d service %s remain %d requests wait %v time when closing service",
			os.Getpid(), s.opts.ServiceName, atomic.LoadInt64(&s.activeCount), closeWaitTime)
		time.Sleep(closeWaitTime)
	}
	return s.waitInactive(s.opts.MaxCloseWaitTime - closeWaitTime)
}

func (s *service) waitInactive(maxWaitTime time.Duration) int64 {
	const sleepTime = 100 * time.Millisecond
	for start := time.Now(); time.Since(start) < maxWaitTime; time.Sleep(sleepTime) {
		if atomic.LoadInt64(&s.activeCount) <= 0 {
			return 0
		}
	}
	return atomic.LoadInt64(&s.activeCount)
}

func checkProcessStatus() (isGracefulRestart, isParentalProcess bool) {
	v := os.Getenv(transport.EnvGraceRestartPPID)
	if v == "" {
		return false, true
	}

	ppid, err := strconv.Atoi(v)
	if err != nil {
		return false, false
	}
	return true, ppid == os.Getpid()
}

func defaultOptions() *Options {
	const (
		invalidSerializationType = -1
		invalidCompressType      = -1
	)
	return &Options{
		protocol:                 "unknown-protocol",
		ServiceName:              "empty-name",
		CurrentSerializationType: invalidSerializationType,
		CurrentCompressType:      invalidCompressType,
	}
}
