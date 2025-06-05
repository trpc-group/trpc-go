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
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	ierror "trpc.group/trpc-go/trpc-go/internal/error"
	ikeeporder "trpc.group/trpc-go/trpc-go/internal/keeporder"
	iserver "trpc.group/trpc-go/trpc-go/internal/local/server"
	ireflect "trpc.group/trpc-go/trpc-go/internal/reflect"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
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
type FilterFunc = func(reqBody interface{}) (filter.ServerChain, error)

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
	activeCount    int64                    // active requests count for graceful close if set MaxCloseWaitTime
	ctx            context.Context          // context of this service
	cancelCause    icontext.CancelCauseFunc // function that cancels this service
	opts           *Options                 // options of this service
	name           string                   // from stub code xxx.trpc.go, should equal to service.callee field in yaml
	handlers       map[string]Handler       // rpcname => handler
	streamHandlers map[string]StreamHandler
	streamInfo     map[string]*StreamServerInfo
	stopListening  chan<- struct{}
}

// desensitizers desensitize sensitive information of address and replace it with *.
var desensitizers = []struct {
	r       *regexp.Regexp
	replace string
}{
	{
		// Kafka address pattern like ip:port?topics={topics}&user=${user}&password=${password}.
		// replace password=${password} with password=*.
		r:       regexp.MustCompile(`pass(wd|word)=([^&]+)`),
		replace: `pass$1=*`,
	},
	{
		// RabbitMQ address pattern like user:password@ip:port.
		// replace user:password@ip:port with user:*@ip:port.
		r:       regexp.MustCompile(`^(\S+):\S+@`),
		replace: `$1:*@`,
	},
}

// New creates a service.
// It will use transport.DefaultServerTransport unless Option WithTransport()
// is called to replace its transport.ServerTransport plugin.
var New = func(opts ...Option) Service {
	const (
		invalidCompressType      = -1
		invalidSerializationType = -1
	)
	s := &service{
		opts: &Options{
			protocol:                 "unknown-protocol",
			ServiceName:              "empty-name",
			CurrentSerializationType: invalidSerializationType,
			CurrentCompressType:      invalidCompressType,
			Transport:                transport.DefaultServerTransport,
			OverloadCtrl:             overloadctrl.NoopOC{},
			methods:                  make(map[string]*methodOptions),
		},
		handlers:       make(map[string]Handler),
		streamHandlers: make(map[string]StreamHandler),
		streamInfo:     make(map[string]*StreamServerInfo),
	}
	// Pass the service (which implements Handler interface) as default handler of transport plugin.
	s.opts.ServeOptions = append(s.opts.ServeOptions, transport.WithHandler(s))

	for _, o := range opts {
		o(s.opts)
	}

	stopListening := make(chan struct{})
	s.stopListening = stopListening
	s.opts.ServeOptions = append(s.opts.ServeOptions,
		transport.WithStopListening(stopListening))
	if s.opts.MaxCloseWaitTime == 0 {
		// By default, set MaxCloseWaitTime to a value greater than CloseWaitTime,
		// providing a specific interval between closing listeners and canceling connections
		// to allow sufficient time for the processing of existing connections to complete.
		s.opts.MaxCloseWaitTime = 2 * s.opts.CloseWaitTime
	}
	if s.opts.MaxCloseWaitTime > s.opts.CloseWaitTime || s.opts.MaxCloseWaitTime > MaxCloseWaitTime {
		s.opts.ServeOptions = append(s.opts.ServeOptions, transport.WithServiceActiveCnt(&s.activeCount))
	}
	s.ctx, s.cancelCause = icontext.WithCancelCause(context.Background())
	return s
}

// Serve implements Service, starting serving.
func (s *service) Serve() error {
	pid := os.Getpid()

	// Make sure ListenAndServe succeeds before Naming Service Registry.
	if err := s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
		log.Errorf("process: %d service: %s ListenAndServe fail: %v, with protocol %s",
			pid, s.opts.ServiceName, err, s.opts.protocol)
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
			log.Errorf("process: %d, service: %s register fail: %v", pid, s.opts.ServiceName, err)
			return err
		}
	}

	log.Infof("process: %d, %s service: %s launch success, %s: %s, serving ...",
		pid, s.opts.protocol, s.opts.ServiceName, s.opts.network, desensitize(s.opts.Address))

	report.ServiceStart.Incr()
	<-s.ctx.Done()
	return nil
}

// PreDecode pre-decodes the given request, which is typically used in keep-order feature.
func (s *service) PreDecode(ctx context.Context, reqBuf []byte) (reqBodyBuf []byte, err error) {
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span, ender, ctx = rpcz.NewSpanContext(ctx, "PreDecode")
		defer func() {
			span.SetAttribute(rpcz.TRPCAttributeError, err)
			ender.End()
		}()
	}

	// If the server codec is empty, simply returns error.
	if s.opts.Codec == nil {
		log.ErrorContextf(ctx, "server codec empty")
		report.ServerCodecEmpty.Incr()
		return nil, errors.New("server codec empty")
	}

	msg := codec.Message(ctx)

	if rpczenable.Enabled {
		_, ender = span.NewChild("DecodeProtocolHead")
	}
	reqBodyBuf, err = s.decode(ctx, msg, reqBuf)
	if rpczenable.Enabled {
		ender.End()
	}
	return
}

// PreUnmarshal does the pre-unmarshaling for the raw request, which is typically used in keep-order feature.
func (s *service) PreUnmarshal(ctx context.Context, reqBuf []byte) (reqBody interface{}, err error) {
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span, ender, ctx = rpcz.NewSpanContext(ctx, "PreUnmarshal")
		defer func() {
			span.SetAttribute(rpcz.TRPCAttributeError, err)
			ender.End()
		}()
	}
	reqBodyBuf, err := s.PreDecode(ctx, reqBuf)
	if err != nil {
		return nil, err
	}
	msg := codec.Message(ctx)
	handler, ok := s.handlers[msg.ServerRPCName()]
	if !ok {
		handler, ok = s.handlers["*"] // Defaults to wildcard.
		if !ok {
			report.ServiceHandleRPCNameInvalid.Incr()
			return nil, errs.NewFrameError(errs.RetServerNoFunc,
				fmt.Sprintf("service handle: rpc name %s invalid, current service: %s. "+
					"This error occurs if the current service (which the client wants to access) isn't registered "+
					"on the server or the RPC name isn't registered with the current service, "+
					"possibly due to an outdated pb file.",
					msg.ServerRPCName(), msg.CalleeServiceName()))

		}
	}
	info, ok := ikeeporder.PreUnmarshalInfoFromContext(ctx)
	if !ok {
		return nil, errors.New("failed to get keeporder pre-unmarshal info")
	}
	newFilterFunc := s.filterFunc(ctx, msg, reqBodyBuf, nil)
	if _, err := handler(ctx, newFilterFunc); err != nil {
		return nil, fmt.Errorf("do handler during pre-unmarshal error: %w", err)
	}
	reqBody = info.ReqBody
	return
}

// Handle implements transport.Handler.
// service itself is passed to its transport plugin as a transport handler.
// This is like a callback function that would be called by service's transport plugin.
func (s *service) Handle(ctx context.Context, reqBuf []byte) (rspBuf []byte, err error) {
	var span rpcz.Span
	if rpczenable.Enabled {
		var ender rpcz.Ender
		span, ender, ctx = rpcz.NewSpanContext(ctx, "Handler")
		span.SetAttribute(rpcz.TRPCAttributeFilterNames, s.opts.FilterNames)
		defer func() {
			span.SetAttribute(rpcz.TRPCAttributeError, err)
			ender.End()
		}()
	}
	// If the server codec is empty, simply returns error.
	if s.opts.Codec == nil {
		log.ErrorContextf(ctx, "server codec empty")
		report.ServerCodecEmpty.Incr()
		return nil, errors.New("server codec empty")
	}
	msg := codec.Message(ctx)
	var reqBodyBuf []byte
	if info, ok := ikeeporder.PreDecodeInfoFromContext(ctx); ok && info != nil {
		// Use the predecoded request body buffer to skip decoding.
		reqBodyBuf = info.ReqBodyBuf
		// Release the pre-decoded request body.
		info.ReqBodyBuf = nil
	} else {
		var ender rpcz.Ender
		if rpczenable.Enabled {
			_, ender = span.NewChild("DecodeProtocolHead")
		}
		reqBodyBuf, err = s.decode(ctx, msg, reqBuf)
		if rpczenable.Enabled {
			ender.End()
		}
	}
	if err != nil {
		return s.encode(ctx, msg, nil, err)
	}
	// ServerRspErr is already set,
	// since RequestID is acquired, just respond to client.
	if err := msg.ServerRspErr(); err != nil {
		return s.encode(ctx, msg, nil, err)
	}

	var token overloadctrl.Token = overloadctrl.NoopToken{}
	if !overloadctrl.IsNoop(s.opts.OverloadCtrl) {
		// Only construct addr string when overload is not noop.
		var addr string
		if msg.RemoteAddr() != nil {
			addr = msg.RemoteAddr().String()
		}
		token, err = s.opts.OverloadCtrl.Acquire(ctx, addr)
		if err != nil {
			report.TCPServerTransportRequestLimitedByOverloadCtrl.Incr()
			return s.encode(ctx, msg, nil,
				errs.NewFrameError(errs.RetServerOverload, err.Error()))
		}
	}

	rspBody, err := s.handle(ctx, msg, reqBodyBuf)
	if err != nil {
		// no response
		if err == errs.ErrServerNoResponse {
			token.OnResponse(ctx, nil)
			return nil, err
		}
		defer token.OnResponse(ctx, err)
		// failed to handle, should respond to client with error code,
		// ignore rspBody.
		report.ServiceHandleFail.Incr()
		return s.encode(ctx, msg, nil, err)
	}
	defer func() {
		token.OnResponse(ctx, err)
		if s.opts.OnResponseObsoleted != nil {
			s.opts.OnResponseObsoleted(ctx, rspBody)
		}
	}()
	return s.handleResponse(ctx, msg, rspBody)
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
		log.TraceContextf(ctx, "service: %s handle err: %+v", s.opts.ServiceName, e)
		msg.WithServerRspErr(e)
	}

	rspBuf, err = s.opts.Codec.Encode(msg, rspBodyBuf)
	if err != nil {
		report.ServiceCodecEncodeFail.Incr()
		log.ErrorContextf(ctx, "service: %s encode fail: %v", s.opts.ServiceName, err)
		return nil, err
	}
	return rspBuf, nil
}

// handleStream handles server stream.
func (s *service) handleStream(
	ctx context.Context, msg codec.Msg, reqBuf []byte, sh StreamHandler, _ *Options,
) (rspBody interface{}, err error) {
	if s.opts.StreamHandle != nil {
		// Only the init frame requires a streamInfo, and only the init frame can locate
		// the streamInfo. For other frame types, the msg.ServerRPCName() is empty.
		si := s.streamInfo[msg.ServerRPCName()]
		return s.opts.StreamHandle.StreamHandleFunc(ctx, sh, si, reqBuf)
	}
	return nil, errs.NewFrameError(errs.RetServerNoService, "Stream method no Handle")
}

func (s *service) decode(_ context.Context, msg codec.Msg, reqBuf []byte) ([]byte, error) {
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
	// Whether is server streaming RPC.
	if fh, ok := msg.FrameHead().(icodec.FrameHead); ok && fh.IsStream() {
		// Only the init frame requires a stream handler, and only the init frame can locate
		// the streamHandler. For other frame types, the msg.ServerRPCName() is empty.
		streamHandler := s.streamHandlers[msg.ServerRPCName()]
		return s.handleStream(ctx, msg, reqBodyBuf, streamHandler, s.opts)
	}
	handler, ok := s.handlers[msg.ServerRPCName()]
	if !ok {
		handler, ok = s.handlers["*"] // wildcard
		if !ok {
			report.ServiceHandleRPCNameInvalid.Incr()
			return nil, errs.NewFrameError(errs.RetServerNoFunc,
				fmt.Sprintf("service handle: rpc name %s invalid, current service: %s. "+
					"This error occurs if the current service (which the client wants to access) isn't registered "+
					"on the server or the RPC name isn't registered with the current service, "+
					"possibly due to an outdated pb file.",
					msg.ServerRPCName(), msg.CalleeServiceName()))

		}
	}

	var timeout = s.opts.Timeout
	if mo, ok := s.opts.methods[msg.CalleeMethod()]; ok && mo.timeout != nil {
		timeout = *mo.timeout
	}

	var fixTimeout filter.ServerFilter
	if timeout > 0 {
		fixTimeout = mayConvert2NormalTimeout
	}
	if msg.RequestTimeout() > 0 && !s.opts.DisableRequestTimeout {
		if msg.RequestTimeout() < timeout || timeout == 0 {
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
// Compress type's setting is similar to it.
func (s *service) handleResponse(ctx context.Context, msg codec.Msg, rspBody interface{}) ([]byte, error) {
	// Marshal response body.
	serializationType := msg.SerializationType()
	if icodec.IsValidSerializationType(s.opts.CurrentSerializationType) {
		serializationType = s.opts.CurrentSerializationType
	}
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	if rpczenable.Enabled {
		span = rpcz.SpanFromContext(ctx)
		_, ender = span.NewChild("Marshal")
	}
	rspBodyBuf, err := codec.Marshal(serializationType, rspBody)
	if rpczenable.Enabled {
		ender.End()
	}

	if err != nil {
		report.ServiceCodecMarshalFail.Incr()
		// rspBodyBuf will be nil if marshalling fails, respond only error code to client.
		return s.encode(ctx, msg, rspBodyBuf, errs.NewFrameError(
			errs.RetServerEncodeFail, "service codec Marshal: "+err.Error()))
	}

	// compress response body
	compressType := msg.CompressType()
	if icodec.IsValidCompressType(s.opts.CurrentCompressType) {
		compressType = s.opts.CurrentCompressType
	}

	if rpczenable.Enabled {
		_, ender = span.NewChild("Compress")
	}
	rspBodyBuf, err = codec.Compress(compressType, rspBodyBuf)
	if rpczenable.Enabled {
		ender.End()
	}

	if err != nil {
		report.ServiceCodecCompressFail.Incr()
		// rspBodyBuf will be nil if compression fails, respond only error code to client.
		return s.encode(ctx, msg, rspBodyBuf, errs.NewFrameError(
			errs.RetServerEncodeFail, "service codec Compress: "+err.Error()))
	}

	if rpczenable.Enabled {
		_, ender = span.NewChild("EncodeProtocolHead")
	}
	rspBuf, err := s.encode(ctx, msg, rspBodyBuf, nil)
	if rpczenable.Enabled {
		ender.End()
	}

	return rspBuf, err
}

// filterFunc returns a FilterFunc, which would be passed to server stub to access pre/post filter handling.
func (s *service) filterFunc(
	ctx context.Context,
	msg codec.Msg,
	reqBodyBuf []byte,
	fixTimeout filter.ServerFilter,
) FilterFunc {
	info, hasPreUnmarshal := ikeeporder.PreUnmarshalInfoFromContext(ctx)
	// Decompression, serialization of request body are put into a closure.
	// Both serialization type & compress type can be set.
	// serialization type is set to msg.SerializationType() by default,
	// if serialization type Option is called, serialization type is set by the Option.
	// compress type's setting is similar to it.
	return func(reqBody interface{}) (filter.ServerChain, error) {
		if hasPreUnmarshal && info != nil && info.Stored {
			if err := ireflect.Assign(reqBody, info.ReqBody); err != nil {
				return nil, fmt.Errorf("assigning pre-unmarshal value to stub error: %w", err)
			}
			// Release the pre-unmarshal value.
			info.ReqBody = nil
		} else {
			if err := s.decompressAndUnmarshal(ctx, msg, reqBodyBuf, reqBody); err != nil {
				return nil, err
			}
			// Check pre-unmarshal.
			if hasPreUnmarshal && info != nil && !info.Stored {
				info.ReqBody = reqBody
				info.Stored = true
				// Under the pre-unmarshal scenario, only noop server filter is needed.
				return filter.ServerChain{
					func(context.Context, interface{}, filter.ServerHandleFunc) (interface{}, error) {
						return nil, nil
					}}, nil
			}
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

func (s *service) decompressAndUnmarshal(
	ctx context.Context,
	msg codec.Msg,
	reqBodyBuf []byte,
	reqBody interface{},
) error {
	var (
		span  rpcz.Span
		ender rpcz.Ender
	)
	// Decompress the request body.
	if rpczenable.Enabled {
		span = rpcz.SpanFromContext(ctx)
		_, ender = span.NewChild("Decompress")
	}
	compressType := msg.CompressType()
	if icodec.IsValidCompressType(s.opts.CurrentCompressType) {
		compressType = s.opts.CurrentCompressType
	}
	reqBodyBuf, err := codec.Decompress(compressType, reqBodyBuf)
	if rpczenable.Enabled {
		ender.End()
	}
	if err != nil {
		report.ServiceCodecDecompressFail.Incr()
		return errs.NewFrameError(errs.RetServerDecodeFail, "service codec Decompress: "+err.Error())
	}

	// Unmarshal the request body.
	if rpczenable.Enabled {
		_, ender = span.NewChild("Unmarshal")
		defer ender.End()
	}
	serializationType := msg.SerializationType()
	if icodec.IsValidSerializationType(s.opts.CurrentSerializationType) {
		serializationType = s.opts.CurrentSerializationType
	}
	if err := codec.Unmarshal(serializationType, reqBodyBuf, reqBody); err != nil {
		report.ServiceCodecUnmarshalFail.Incr()
		return errs.NewFrameError(errs.RetServerDecodeFail, "service codec Unmarshal: "+err.Error())
	}
	return nil
}

// Register implements Service interface, registering a proto service impl for the service.
func (s *service) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	desc, ok := serviceDesc.(*ServiceDesc)
	if !ok {
		return errors.New("serviceDesc is not *ServiceDesc")
	}
	s.name = desc.ServiceName
	if desc.StreamHandle != nil {
		s.opts.StreamHandle = desc.StreamHandle
		if s.opts.StreamTransport != nil {
			s.opts.Transport = s.opts.StreamTransport
		}
		// IdleTimeout is not used by server stream, set it to 0.
		s.opts.ServeOptions = append(s.opts.ServeOptions, transport.WithServerIdleTimeout(0))
		if err := s.opts.StreamHandle.Init(s.opts); err != nil {
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
		handler := func(ctx context.Context, f FilterFunc) (rsp interface{}, err error) {
			return h(serviceImpl, ctx, f)
		}
		s.handlers[n] = handler
		// Here we must use the s.opt.ServerName as the argument, not s.name,
		// since the former is the one that comes from the configuration.
		iserver.Register(s.opts.ServiceName, n, handler, iserver.Options{
			Protocol: s.opts.protocol,
			Filters:  s.opts.Filters,
			ServerCodecGetter: func() codec.Codec {
				return s.opts.Codec
			},
		})
		bindings = append(bindings, method.Bindings...)
	}

	for _, stream := range desc.Streams {
		streamName := stream.StreamName
		if _, ok := s.streamHandlers[streamName]; ok {
			return fmt.Errorf("duplicate stream name: %s", streamName)
		}
		s.streamInfo[streamName] = &StreamServerInfo{
			FullMethod:     streamName,
			IsClientStream: stream.ClientStreams,
			IsServerStream: stream.ServerStreams,
		}
		h := stream.Handler
		s.streamHandlers[streamName] = func(s Stream) error {
			return h(serviceImpl, s)
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

	opts := append(s.opts.RESTOptions,
		restful.WithNamespace(s.opts.Namespace),
		restful.WithEnvironment(s.opts.EnvName),
		restful.WithContainer(s.opts.container),
		restful.WithSet(s.opts.SetName),
		restful.WithServiceName(s.opts.ServiceName),
		restful.WithServiceImpl(serviceImpl),
		restful.WithTimeout(s.opts.Timeout),
		restful.WithDisableRequestTimeout(s.opts.DisableRequestTimeout),
		restful.WithFilterFunc(func() filter.ServerChain { return s.opts.Filters }),
	)
	for method, mo := range s.opts.methods {
		if mo.timeout != nil {
			opts = append(opts, restful.WithMethodTimeout(method, *mo.timeout))
		}
	}

	// This is the first time of registering the service router, create a new one.
	router := restful.NewRouter(opts...)
	for _, binding := range bindings {
		if err := router.AddBinding(binding); err != nil {
			return err
		}
	}
	restful.RegisterRouter(s.opts.ServiceName, router)
	restful.RegisterFasthttpRouter(s.opts.ServiceName, router.HandleRequestCtx)
	return nil
}

var _ causeCloser = (*service)(nil)

// CloseCause closes the service, registry.Deregister will be called.
func (s *service) CloseCause(err error) error {
	return s.closeCause(err)
}

// Close closes the service, registry.Deregister will be called.
func (s *service) Close(ch chan struct{}) error {
	err := s.closeCause(nil)
	if ch != nil {
		ch <- struct{}{}
	}
	return err
}

func (s *service) closeCause(err error) error {
	pid := os.Getpid()
	log.Infof("process: %d, %s service: %s, closing...", pid, s.opts.protocol, s.opts.ServiceName)

	if s.opts.Registry != nil {
		// When it comes to graceful restart, the parent process will not call registry Deregister(),
		// while the child process would call registry Deregister().
		if isGraceful, isParental := checkProcessStatus(); !(isGraceful && isParental) &&
			!errors.Is(err, ierror.GracefulRestart) {
			if err := s.opts.Registry.Deregister(s.opts.ServiceName); err != nil {
				log.Errorf("process: %d, deregister service: %s fail: %v", pid, s.opts.ServiceName, err)
			}
		}
	}

	remaining := s.waitBeforeClose()

	close(s.stopListening)
	s.cancelCause(err)

	maxWaitTime := time.Millisecond * 300
	if s.opts.Timeout*2 > maxWaitTime { // use the larger one
		maxWaitTime = s.opts.Timeout * 2
	}
	if remaining > maxWaitTime {
		maxWaitTime = remaining
	}
	if remains := s.waitInactive(maxWaitTime); remains > 0 {
		log.Infof("process %d service %s still remains %d requests/listeners/conns after force closing",
			os.Getpid(), s.opts.ServiceName, remains)
	}

	log.Infof("process: %d, %s service: %s, closed", pid, s.opts.protocol, s.opts.ServiceName)
	return nil
}

func (s *service) waitBeforeClose() (remaining time.Duration) {
	closeWaitTime := s.opts.CloseWaitTime
	if closeWaitTime > MaxCloseWaitTime {
		closeWaitTime = MaxCloseWaitTime
	}
	if closeWaitTime > 0 {
		// After registry.Deregister() is called, sleep a while to let Naming Service (like Polaris) finish
		// updating instance ip list.
		// Otherwise, client request would still arrive while the service had already been closed (Typically, it occurs
		// when k8s updates pods).
		log.Infof("process %d service %s remain %d requests/listeners/conns, wait %v before closing service",
			os.Getpid(), s.opts.ServiceName, atomic.LoadInt64(&s.activeCount), closeWaitTime)
		time.Sleep(closeWaitTime)
	}
	return s.opts.MaxCloseWaitTime - closeWaitTime
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

// desensitize desensitizes sensitive information of address using desensitizers.
func desensitize(s string) string {
	for _, desensitizer := range desensitizers {
		s = desensitizer.r.ReplaceAllString(s, desensitizer.replace)
	}
	return s
}
