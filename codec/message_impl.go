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

package codec

import (
	"context"
	"net"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/errs"
)

// msg is the context of rpc.
type msg struct {
	context             context.Context
	frameHead           interface{}
	requestTimeout      time.Duration
	serializationType   int
	compressType        int
	streamID            uint32
	dyeing              bool
	dyeingKey           string
	serverRPCName       string
	clientRPCName       string
	serverMetaData      MetaData
	clientMetaData      MetaData
	callerServiceName   string
	calleeServiceName   string
	calleeContainerName string
	serverRspErr        error
	clientRspErr        error
	serverReqHead       interface{}
	serverRspHead       interface{}
	clientReqHead       interface{}
	clientRspHead       interface{}
	localAddr           net.Addr
	remoteAddr          net.Addr
	logger              interface{}
	callerApp           string
	callerServer        string
	callerService       string
	callerMethod        string
	calleeApp           string
	calleeServer        string
	calleeService       string
	calleeMethod        string
	namespace           string
	setName             string
	envName             string
	envTransfer         string
	requestID           uint32
	calleeSetName       string
	streamFrame         interface{}
	commonMeta          CommonMeta
	callType            RequestType
}

// resetDefault reset all fields of msg to default value.
func (m *msg) resetDefault() {
	m.context = nil
	m.frameHead = nil
	m.requestTimeout = 0
	m.serializationType = 0
	m.compressType = 0
	m.dyeing = false
	m.dyeingKey = ""
	m.serverRPCName = ""
	m.clientRPCName = ""
	m.serverMetaData = nil
	m.clientMetaData = nil
	m.callerServiceName = ""
	m.calleeServiceName = ""
	m.calleeContainerName = ""
	m.serverRspErr = nil
	m.clientRspErr = nil
	m.serverReqHead = nil
	m.serverRspHead = nil
	m.clientReqHead = nil
	m.clientRspHead = nil
	m.localAddr = nil
	m.remoteAddr = nil
	m.logger = nil
	m.callerApp = ""
	m.callerServer = ""
	m.callerService = ""
	m.callerMethod = ""
	m.calleeApp = ""
	m.calleeServer = ""
	m.calleeService = ""
	m.calleeMethod = ""
	m.namespace = ""
	m.setName = ""
	m.envName = ""
	m.envTransfer = ""
	m.requestID = 0
	m.streamFrame = nil
	m.streamID = 0
	m.calleeSetName = ""
	m.commonMeta = nil
	m.callType = 0
}

// Context restores old context when create new msg.
func (m *msg) Context() context.Context {
	return m.context
}

// WithNamespace set server's namespace.
func (m *msg) WithNamespace(namespace string) {
	m.namespace = namespace
}

// Namespace returns namespace.
func (m *msg) Namespace() string {
	return m.namespace
}

// WithEnvName sets environment.
func (m *msg) WithEnvName(envName string) {
	m.envName = envName
}

// WithSetName sets set name.
func (m *msg) WithSetName(setName string) {
	m.setName = setName
}

// SetName returns set name.
func (m *msg) SetName() string {
	return m.setName
}

// WithCalleeSetName sets the callee set name.
func (m *msg) WithCalleeSetName(s string) {
	m.calleeSetName = s
}

// CalleeSetName returns the callee set name.
func (m *msg) CalleeSetName() string {
	return m.calleeSetName
}

// EnvName returns environment.
func (m *msg) EnvName() string {
	return m.envName
}

// WithEnvTransfer sets environment transfer value.
func (m *msg) WithEnvTransfer(envTransfer string) {
	m.envTransfer = envTransfer
}

// EnvTransfer returns environment transfer value.
func (m *msg) EnvTransfer() string {
	return m.envTransfer
}

// WithRemoteAddr sets remote address.
func (m *msg) WithRemoteAddr(addr net.Addr) {
	m.remoteAddr = addr
}

// WithLocalAddr set local address.
func (m *msg) WithLocalAddr(addr net.Addr) {
	m.localAddr = addr
}

// RemoteAddr returns remote address.
func (m *msg) RemoteAddr() net.Addr {
	return m.remoteAddr
}

// LocalAddr returns local address.
func (m *msg) LocalAddr() net.Addr {
	return m.localAddr
}

// RequestTimeout returns request timeout set by
// upstream business protocol.
func (m *msg) RequestTimeout() time.Duration {
	return m.requestTimeout
}

// WithRequestTimeout sets request timeout.
func (m *msg) WithRequestTimeout(t time.Duration) {
	m.requestTimeout = t
}

// FrameHead returns frame head.
func (m *msg) FrameHead() interface{} {
	return m.frameHead
}

// WithFrameHead sets frame head.
func (m *msg) WithFrameHead(f interface{}) {
	m.frameHead = f
}

// SerializationType returns the value of body serialization, which is
// defined in serialization.go.
func (m *msg) SerializationType() int {
	return m.serializationType
}

// WithSerializationType sets body serialization type of body.
func (m *msg) WithSerializationType(t int) {
	m.serializationType = t
}

// CompressType returns compress type value, which is defined in compress.go.
func (m *msg) CompressType() int {
	return m.compressType
}

// WithCompressType sets compress type.
func (m *msg) WithCompressType(t int) {
	m.compressType = t
}

// ServerRPCName returns server rpc name.
func (m *msg) ServerRPCName() string {
	return m.serverRPCName
}

// WithServerRPCName sets server rpc name.
func (m *msg) WithServerRPCName(s string) {
	if m.serverRPCName == s {
		return
	}
	m.serverRPCName = s
	m.updateMethodNameUsingRPCName(s)
}

// ClientRPCName returns client rpc name.
func (m *msg) ClientRPCName() string {
	return m.clientRPCName
}

// WithClientRPCName sets client rpc name, which will be called
// by client stub.
func (m *msg) WithClientRPCName(s string) {
	if m.clientRPCName == s {
		return
	}
	m.clientRPCName = s
	m.updateMethodNameUsingRPCName(s)
}

func (m *msg) updateMethodNameUsingRPCName(s string) {
	if m.CalleeMethod() == "" {
		m.WithCalleeMethod(s)
	}
}

// ServerMetaData returns server meta data, which is passed to server.
func (m *msg) ServerMetaData() MetaData {
	return m.serverMetaData
}

// WithServerMetaData sets server meta data.
func (m *msg) WithServerMetaData(d MetaData) {
	if d == nil {
		d = MetaData{}
	}
	m.serverMetaData = d
}

// ClientMetaData returns client meta data, which will pass to downstream.
func (m *msg) ClientMetaData() MetaData {
	return m.clientMetaData
}

// WithClientMetaData set client meta data.
func (m *msg) WithClientMetaData(d MetaData) {
	if d == nil {
		d = MetaData{}
	}
	m.clientMetaData = d
}

// CalleeServiceName returns callee service name.
func (m *msg) CalleeServiceName() string {
	return m.calleeServiceName
}

// WithCalleeServiceName sets callee service name.
func (m *msg) WithCalleeServiceName(s string) {
	if m.calleeServiceName == s {
		return
	}
	m.calleeServiceName = s
	if s == "*" {
		return
	}
	app, server, service := getAppServerService(s)
	m.WithCalleeApp(app)
	m.WithCalleeServer(server)
	m.WithCalleeService(service)
}

// CalleeContainerName returns callee container name.
func (m *msg) CalleeContainerName() string {
	return m.calleeContainerName
}

// WithCalleeContainerName sets callee container name.
func (m *msg) WithCalleeContainerName(s string) {
	m.calleeContainerName = s
}

// WithStreamFrame sets stream frame.
func (m *msg) WithStreamFrame(i interface{}) {
	m.streamFrame = i
}

// StreamFrame returns stream frame.
func (m *msg) StreamFrame() interface{} {
	return m.streamFrame
}

// CallerServiceName returns caller service name.
func (m *msg) CallerServiceName() string {
	return m.callerServiceName
}

// WithCallerServiceName sets caller service name.
func (m *msg) WithCallerServiceName(s string) {
	if m.callerServiceName == s {
		return
	}
	m.callerServiceName = s
	if s == "*" {
		return
	}
	app, server, service := getAppServerService(s)
	m.WithCallerApp(app)
	m.WithCallerServer(server)
	m.WithCallerService(service)
}

// ServerRspErr returns server response error, which is created
// by handler.
func (m *msg) ServerRspErr() *errs.Error {
	if m.serverRspErr == nil {
		return nil
	}
	e, ok := m.serverRspErr.(*errs.Error)
	if !ok {
		return &errs.Error{
			Type: errs.ErrorTypeBusiness,
			Code: errs.RetUnknown,
			Msg:  m.serverRspErr.Error(),
		}
	}
	return e
}

// WithServerRspErr sets server response error.
func (m *msg) WithServerRspErr(e error) {
	m.serverRspErr = e
}

// WithStreamID sets stream id.
func (m *msg) WithStreamID(streamID uint32) {
	m.streamID = streamID
}

// StreamID returns stream id.
func (m *msg) StreamID() uint32 {
	return m.streamID
}

// ClientRspErr returns client response error, which created when client call downstream.
func (m *msg) ClientRspErr() error {
	return m.clientRspErr
}

// WithClientRspErr sets client response err, this method will called
// when client parse response package.
func (m *msg) WithClientRspErr(e error) {
	m.clientRspErr = e
}

// ServerReqHead returns the package head of request
func (m *msg) ServerReqHead() interface{} {
	return m.serverReqHead
}

// WithServerReqHead sets the package head of request
func (m *msg) WithServerReqHead(h interface{}) {
	m.serverReqHead = h
}

// ServerRspHead returns the package head of response
func (m *msg) ServerRspHead() interface{} {
	return m.serverRspHead
}

// WithServerRspHead sets the package head returns to upstream
func (m *msg) WithServerRspHead(h interface{}) {
	m.serverRspHead = h
}

// ClientReqHead returns the request package head of client,
// this is set only when cross protocol call.
func (m *msg) ClientReqHead() interface{} {
	return m.clientReqHead
}

// WithClientReqHead sets the request package head of client.
func (m *msg) WithClientReqHead(h interface{}) {
	m.clientReqHead = h
}

// ClientRspHead returns the request package head of client.
func (m *msg) ClientRspHead() interface{} {
	return m.clientRspHead
}

// WithClientRspHead sets the response package head of client.
func (m *msg) WithClientRspHead(h interface{}) {
	m.clientRspHead = h
}

// Dyeing return the dyeing mark.
func (m *msg) Dyeing() bool {
	return m.dyeing
}

// WithDyeing sets the dyeing mark.
func (m *msg) WithDyeing(dyeing bool) {
	m.dyeing = dyeing
}

// DyeingKey returns the dyeing key.
func (m *msg) DyeingKey() string {
	return m.dyeingKey
}

// WithDyeingKey sets the dyeing key.
func (m *msg) WithDyeingKey(key string) {
	m.dyeingKey = key
}

// CallerApp returns caller app.
func (m *msg) CallerApp() string {
	return m.callerApp
}

// WithCallerApp sets caller app.
func (m *msg) WithCallerApp(app string) {
	m.callerApp = app
}

// CallerServer returns caller server.
func (m *msg) CallerServer() string {
	return m.callerServer
}

// WithCallerServer sets caller server.
func (m *msg) WithCallerServer(s string) {
	m.callerServer = s
}

// CallerService returns caller service.
func (m *msg) CallerService() string {
	return m.callerService
}

// WithCallerService sets caller service.
func (m *msg) WithCallerService(s string) {
	m.callerService = s
}

// WithCallerMethod sets caller method.
func (m *msg) WithCallerMethod(s string) {
	m.callerMethod = s
}

// CallerMethod returns caller method.
func (m *msg) CallerMethod() string {
	return m.callerMethod
}

// CalleeApp returns caller app.
func (m *msg) CalleeApp() string {
	return m.calleeApp
}

// WithCalleeApp sets callee app.
func (m *msg) WithCalleeApp(app string) {
	m.calleeApp = app
}

// CalleeServer returns callee server.
func (m *msg) CalleeServer() string {
	return m.calleeServer
}

// WithCalleeServer sets callee server.
func (m *msg) WithCalleeServer(s string) {
	m.calleeServer = s
}

// CalleeService returns callee service.
func (m *msg) CalleeService() string {
	return m.calleeService
}

// WithCalleeService sets callee service.
func (m *msg) WithCalleeService(s string) {
	m.calleeService = s
}

// WithCalleeMethod sets callee method.
func (m *msg) WithCalleeMethod(s string) {
	m.calleeMethod = s
}

// CalleeMethod returns callee method.
func (m *msg) CalleeMethod() string {
	return m.calleeMethod
}

// WithLogger sets logger into context message. Generally, the logger is
// created from WithFields() method.
func (m *msg) WithLogger(l interface{}) {
	m.logger = l
}

// Logger returns logger from context message.
func (m *msg) Logger() interface{} {
	return m.logger
}

// WithRequestID sets request id.
func (m *msg) WithRequestID(id uint32) {
	m.requestID = id
}

// RequestID returns request id.
func (m *msg) RequestID() uint32 {
	return m.requestID
}

// WithCommonMeta sets common meta data.
func (m *msg) WithCommonMeta(c CommonMeta) {
	m.commonMeta = c
}

// CommonMeta returns common meta data.
func (m *msg) CommonMeta() CommonMeta {
	return m.commonMeta
}

// WithCallType sets type of call.
func (m *msg) WithCallType(t RequestType) {
	m.callType = t
}

// CallType returns type of call.
func (m *msg) CallType() RequestType {
	return m.callType
}

// WithNewMessage create a new empty message, and put it into ctx,
func WithNewMessage(ctx context.Context) (context.Context, Msg) {

	m := msgPool.Get().(*msg)
	ctx = context.WithValue(ctx, ContextKeyMessage, m)
	m.context = ctx
	return ctx, m
}

// PutBackMessage return struct Message to sync pool,
// and reset all the members of Message to default
func PutBackMessage(sourceMsg Msg) {
	m, ok := sourceMsg.(*msg)
	if !ok {
		return
	}
	m.resetDefault()
	msgPool.Put(m)
}

// WithCloneContextAndMessage creates a new context, then copy the message of current context
// into new context, this method will return the new context and message for stream mod.
func WithCloneContextAndMessage(ctx context.Context) (context.Context, Msg) {
	newMsg := msgPool.Get().(*msg)
	newCtx := context.Background()
	val := ctx.Value(ContextKeyMessage)
	m, ok := val.(*msg)
	if !ok {
		newCtx = context.WithValue(newCtx, ContextKeyMessage, newMsg)
		newMsg.context = newCtx
		return newCtx, newMsg
	}
	newCtx = context.WithValue(newCtx, ContextKeyMessage, newMsg)
	newMsg.context = newCtx
	copyCommonMessage(m, newMsg)
	copyServerToServerMessage(m, newMsg)
	return newCtx, newMsg
}

// copyCommonMessage copy common data of message.
func copyCommonMessage(m *msg, newMsg *msg) {
	// Do not copy compress type here, as it will cause subsequence RPC calls to inherit the upstream
	// compress type which is not the expected behavior. Compress type should not be propagated along
	// the entire RPC invocation chain.
	newMsg.frameHead = m.frameHead
	newMsg.requestTimeout = m.requestTimeout
	newMsg.serializationType = m.serializationType
	newMsg.serverRPCName = m.serverRPCName
	newMsg.clientRPCName = m.clientRPCName
	newMsg.serverReqHead = m.serverReqHead
	newMsg.serverRspHead = m.serverRspHead
	newMsg.dyeing = m.dyeing
	newMsg.dyeingKey = m.dyeingKey
	newMsg.serverMetaData = m.serverMetaData.Clone()
	newMsg.logger = m.logger
	newMsg.namespace = m.namespace
	newMsg.envName = m.envName
	newMsg.setName = m.setName
	newMsg.envTransfer = m.envTransfer
	newMsg.commonMeta = m.commonMeta.Clone()
}

// copyClientMessage copy the message transferred from server to client.
func copyServerToClientMessage(m *msg, newMsg *msg) {
	newMsg.clientMetaData = m.serverMetaData.Clone()
	// clone this message for downstream client, so caller is equal to callee.
	newMsg.callerServiceName = m.calleeServiceName
	newMsg.callerApp = m.calleeApp
	newMsg.callerServer = m.calleeServer
	newMsg.callerService = m.calleeService
	newMsg.callerMethod = m.calleeMethod
}

func copyServerToServerMessage(m *msg, newMsg *msg) {
	newMsg.callerServiceName = m.callerServiceName
	newMsg.callerApp = m.callerApp
	newMsg.callerServer = m.callerServer
	newMsg.callerService = m.callerService
	newMsg.callerMethod = m.callerMethod

	newMsg.calleeServiceName = m.calleeServiceName
	newMsg.calleeService = m.calleeService
	newMsg.calleeApp = m.calleeApp
	newMsg.calleeServer = m.calleeServer
	newMsg.calleeMethod = m.calleeMethod
}

// WithCloneMessage copy a new message and put into context, each rpc call should
// create a new message, this method will be called by client stub.
func WithCloneMessage(ctx context.Context) (context.Context, Msg) {
	newMsg := msgPool.Get().(*msg)
	val := ctx.Value(ContextKeyMessage)
	m, ok := val.(*msg)
	if !ok {
		ctx = context.WithValue(ctx, ContextKeyMessage, newMsg)
		newMsg.context = ctx
		return ctx, newMsg
	}
	ctx = context.WithValue(ctx, ContextKeyMessage, newMsg)
	newMsg.context = ctx
	copyCommonMessage(m, newMsg)
	copyServerToClientMessage(m, newMsg)
	return ctx, newMsg
}

// Message returns the message of context.
func Message(ctx context.Context) Msg {
	val := ctx.Value(ContextKeyMessage)
	m, ok := val.(*msg)
	if !ok {
		return &msg{context: ctx}
	}
	return m
}

// EnsureMessage returns context and message, if there is a message in context,
// returns the original one, if not, returns a new one.
func EnsureMessage(ctx context.Context) (context.Context, Msg) {
	val := ctx.Value(ContextKeyMessage)
	if m, ok := val.(*msg); ok {
		return ctx, m
	}
	return WithNewMessage(ctx)
}

// getAppServerService returns app, server and service parsed from service name.
// service name example: trpc.app.server.service
func getAppServerService(s string) (app, server, service string) {
	if strings.Count(s, ".") >= ServiceSectionLength-1 {
		i := strings.Index(s, ".") + 1
		j := strings.Index(s[i:], ".") + i + 1
		k := strings.Index(s[j:], ".") + j + 1
		app = s[i : j-1]
		server = s[j : k-1]
		service = s[k:]
		return
	}
	// app
	i := strings.Index(s, ".")
	if i == -1 {
		app = s
		return
	}
	app = s[:i]
	// server
	i++
	j := strings.Index(s[i:], ".")
	if j == -1 {
		server = s[i:]
		return
	}
	j += i + 1
	server = s[i : j-1]
	// service
	service = s[j:]
	return
}
