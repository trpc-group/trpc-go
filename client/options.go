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

package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	"trpc.group/trpc-go/trpc-go/internal/scope"
	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

// Options are client side options.
type Options struct {
	ServiceName       string        // Backend service name.
	Tag               string        // Tag of the Backend config.
	CallerServiceName string        // Service name of caller itself.
	CalleeMethod      string        // Callee method name, usually used for metrics.
	Timeout           time.Duration // Timeout.

	// Target is address of backend service: name://endpoint,
	// also compatible with old addressing like cl5://sid cmlb://appid ip://ip:port
	Target   string
	endpoint string // The same as service name if target is not set.

	OverloadCtrl overloadctrl.OverloadController // Client side overload control.

	Network           string
	CallType          codec.RequestType           // Type of request, referring to transport.RequestType.
	CallOptions       []transport.RoundTripOption // Options for client transport to call server.
	Transport         transport.ClientTransport
	EnableMultiplexed bool
	StreamTransport   transport.ClientStreamTransport

	SelectOptions             []selector.Option
	Selector                  selector.Selector
	DisableServiceRouter      bool
	shouldErrReportToSelector func(error) bool

	CurrentSerializationType int
	CurrentCompressType      int
	SerializationType        int
	CompressType             int

	Codec                 codec.Codec
	MetaData              codec.MetaData
	ClientStreamQueueSize int // Size of client stream's queue.

	Filters                filter.ClientChain // Filter chain.
	FilterNames            []string           // The name of filters.
	DisableFilter          bool               // Whether to disable filter.
	selectorFilterPosFixed bool               // Whether selector filter pos is fixed, if not, put it to the end.

	methods map[string]*methodOptions

	ReqHead interface{} // Allow custom req head.
	RspHead interface{} // Allow custom rsp head.
	Node    *onceNode   // For getting node info.

	MaxWindowSize uint32            // Max size of stream receiver's window.
	SControl      SendControl       // Sender's flow control.
	RControl      RecvControl       // Receiver's flow control.
	StreamFilters StreamFilterChain // Stream filter chain.

	// Scope is the scope of the current client, the allowed values are:
	// "local": the client can only call the local server.
	// "remote": the client can only call the remote server.
	// "all": the client can call the local and remote server (first try local, then remote).
	Scope     scope.Scope
	localAddr net.Addr

	fixTimeout func(error) error

	attachment *attachment.Attachment

	// protocol is the current protocol used by client.
	protocol string
}

// methodOptions defines the method level options.
type methodOptions struct {
	timeout *time.Duration
}

type onceNode struct {
	*registry.Node
	once sync.Once
}

func (n *onceNode) set(node *registry.Node, address string, cost time.Duration) {
	if n == nil {
		return
	}
	n.once.Do(func() {
		*n.Node = *node
		n.Node.Address = address
		n.Node.CostTime = cost
	})
}

// Option sets client options.
type Option func(*Options)

// WithNamespace returns an Option that sets namespace of backend service: Production/Development.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithNamespace(namespace))
	}
}

// WithClientStreamQueueSize returns an Option that sets the size of client stream's buffer queue,
// that is, max number of received messages to put into the channel.
func WithClientStreamQueueSize(size int) Option {
	return func(o *Options) {
		o.ClientStreamQueueSize = size
	}
}

// WithServiceName returns an Option that sets service name of backend service.
func WithServiceName(s string) Option {
	return func(o *Options) {
		o.ServiceName = s
		o.endpoint = s
	}
}

// WithTag returns an Option that sets tag of backend service.
func WithTag(tag string) Option {
	return func(o *Options) {
		o.Tag = tag
	}
}

// WithCallerServiceName returns an Option that sets service name of the caller service itself.
func WithCallerServiceName(s string) Option {
	return func(o *Options) {
		o.CallerServiceName = s
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceServiceName(s))
	}
}

// WithCallerNamespace returns an Option that sets namespace of the caller service itself.
func WithCallerNamespace(s string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceNamespace(s))
	}
}

// WithDisableFilter returns an Option that sets whether to disable filter.
// It's used when a plugin setup and need a client to send request
// but filters' initialization has not been done.
func WithDisableFilter() Option {
	return func(o *Options) {
		o.DisableFilter = true
	}
}

// WithDisableServiceRouter returns an Option that disables service router.
func WithDisableServiceRouter() Option {
	return func(o *Options) {
		o.DisableServiceRouter = true
		o.SelectOptions = append(o.SelectOptions, selector.WithDisableServiceRouter())
	}
}

// WithEnvKey returns an Option that sets env key.
func WithEnvKey(key string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithEnvKey(key))
	}
}

// WithCallerEnvName returns an Option that sets env name of the caller service itself.
func WithCallerEnvName(envName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceEnvName(envName))
	}
}

// WithCallerSetName returns an Option that sets "Set" of the caller service itself.
func WithCallerSetName(setName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceSetName(setName))
	}
}

// WithCalleeSetName returns an Option that sets "Set" of the callee service.
func WithCalleeSetName(setName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDestinationSetName(setName))
	}
}

// WithCalleeEnvName returns an Option that sets env name of the callee service.
func WithCalleeEnvName(envName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDestinationEnvName(envName))
	}
}

// WithCalleeMethod returns an Option that sets callee method name.
func WithCalleeMethod(method string) Option {
	return func(o *Options) {
		o.CalleeMethod = method
	}
}

// WithCallerMetadata returns an Option that sets metadata of caller, only used for polaris routing addressing.
// CallerMetadata is also called SourceMetadata in Polaris router.
// It should not be used for env/set as specific methods are provided for env/set.
// If you need to transparently transmit business data to the downstream, please use WithMetaData.
func WithCallerMetadata(key string, val string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceMetadata(key, val))
	}
}

// WithCalleeMetadata returns an Option that sets metadata of callee, only used for polaris routing addressing.
// CalleeMetadata is also called DestinationMetadata in Polaris router.
// It should not be used for env/set as specific methods are provided for env/set.
// If you need to transparently transmit business data to the downstream, please use WithMetaData.
func WithCalleeMetadata(key string, val string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDestinationMetadata(key, val))
	}
}

// WithBalancerName returns an Option that sets load balancer by name.
func WithBalancerName(balancerName string) Option {
	balancer := loadbalance.Get(balancerName)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions,
			selector.WithLoadBalancer(balancer),
			selector.WithLoadBalanceType(balancerName),
		)
	}
}

// WithDiscoveryName returns an Option that sets service discovery by name.
func WithDiscoveryName(name string) Option {
	d := discovery.Get(name)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDiscovery(d))
	}
}

// WithServiceRouterName returns an Option that sets service router by name.
func WithServiceRouterName(name string) Option {
	r := servicerouter.Get(name)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithServiceRouter(r))
	}
}

// WithCircuitBreakerName returns an Option that sets circuit breaker by name.
func WithCircuitBreakerName(name string) Option {
	cb := circuitbreaker.Get(name)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithCircuitBreaker(cb))
	}
}

// WithKey returns an Option that sets the hash key of stateful routing.
func WithKey(key string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithKey(key))
	}
}

// WithReplicas returns an Option that sets node replicas of stateful routing.
func WithReplicas(r int) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithReplicas(r))
	}
}

// WithTarget returns an Option that sets target address with scheme name://endpoint,
// like cl5://sid ons://zkname ip://ip:port.
func WithTarget(t string) Option {
	return func(o *Options) {
		o.Target = t
		o.endpoint = "" // should parse endpoint again after calling WithTarget
	}
}

// WithNetwork returns an Option that sets dial network: tcp/udp, tcp by default.
func WithNetwork(s string) Option {
	return func(o *Options) {
		if s == "" {
			return
		}
		o.Network = s
		o.CallOptions = append(o.CallOptions, transport.WithDialNetwork(s))
	}
}

// WithPassword returns an Option that sets dial password.
func WithPassword(s string) Option {
	return func(o *Options) {
		if s == "" {
			return
		}
		o.CallOptions = append(o.CallOptions, transport.WithDialPassword(s))
	}
}

// WithPool returns an Option that sets dial pool.
func WithPool(pool connpool.Pool) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDialPool(pool))
	}
}

// WithMultiplexedPool returns an Option that sets multiplexed pool.
// Calling this method enables multiplexing.
func WithMultiplexedPool(p multiplexed.Pool) Option {
	return func(o *Options) {
		o.EnableMultiplexed = true
		o.CallOptions = append(o.CallOptions, transport.WithMultiplexedPool(p))
	}
}

// WithTimeout returns an Option that sets timeout.
func WithTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// WithScope returns an Option that sets the client's Scope.
// "local": the client can only call the local server.
// "remote": the client can only call the remote server.
// "all": the client can call the local and remote server (first try local, then remote).
//
// The default value is "remote".
func WithScope(scope scope.Scope) Option {
	return func(o *Options) {
		o.Scope = scope
	}
}

// WithCurrentSerializationType returns an Option that sets serialization type of caller itself.
// WithSerializationType should be used to set serialization type of backend service.
//
// When WithSerializationType and WithCurrentSerializationType are used together, their roles differ:
//   - WithSerializationType specifies the intended serialization type for the final payload,
//     although the framework may not necessarily perform the serialization.
//   - WithCurrentSerializationType determines the actual serialization operation that the framework needs to carry out.
//
// The most common practice is to use WithCurrentSerializationType(codec.SerializationTypeNoop),
// and then use WithSerializationType to specify an actual serialization method.
// In this way, you can directly provide the serialized []byte, and also specify that the serialization method
// filled in the protocol header is a certain serialization method, thereby skipping the step of the framework
// executing the serialization.
func WithCurrentSerializationType(t int) Option {
	return func(o *Options) {
		o.CurrentSerializationType = t
	}
}

// WithSerializationType returns an Option that sets serialization type of backend service.
// Generally, only WithSerializationType will be used as WithCurrentSerializationType is used
// for reverse proxy.
func WithSerializationType(t int) Option {
	return func(o *Options) {
		o.SerializationType = t
	}
}

// WithCurrentCompressType returns an Option that sets compression type of caller itself.
// WithCompressType should be used to set compression type of backend service.
//
// When WithCompressType and WithCurrentCompressType are used together, their roles differ:
//   - WithCompressType specifies the intended compression type for the final payload,
//     although the framework may not necessarily perform the compression.
//   - WithCurrentCompressType determines the actual compression operation that the framework needs to carry out.
//
// The most common practice is to use WithCurrentCompressType(codec.CompressTypeNoop),
// and then use WithCompressType to specify an actual compression method.
// In this way, you can directly provide the compressed []byte, and also specify that the compression method
// filled in the protocol header is a certain compression method, thereby skipping the step of the framework
// executing the compression.
func WithCurrentCompressType(t int) Option {
	return func(o *Options) {
		o.CurrentCompressType = t
	}
}

// WithCompressType returns an Option that sets compression type of backend service.
// Generally, only WithCompressType will be used as WithCurrentCompressType is used
// for reverse proxy.
func WithCompressType(t int) Option {
	return func(o *Options) {
		o.CompressType = t
	}
}

// WithTransport returns an Option that sets client transport plugin.
func WithTransport(t transport.ClientTransport) Option {
	return func(o *Options) {
		if t != nil {
			o.Transport = t
		}
	}
}

// WithProtocol returns an Option that sets protocol of backend service like trpc.
func WithProtocol(s string) Option {
	return func(o *Options) {
		if s == "" {
			return
		}
		o.protocol = s
		o.Codec = codec.GetClient(s)
		if b := transport.GetFramerBuilder(s); b != nil {
			o.CallOptions = append(o.CallOptions,
				transport.WithClientFramerBuilder(b),
				transport.WithProtocol(s),
			)
		}
		if t := transport.GetClientTransport(s); t != nil {
			o.Transport = t
		}
	}
}

// WithConnectionMode returns an Option that sets whether connection mode is connected.
// If connection mode is connected, udp will isolate packets from non-same path.
func WithConnectionMode(connMode transport.ConnectionMode) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithConnectionMode(connMode))
	}
}

// WithSendOnly returns an Option that sets CallType SendOnly.
// Generally it's used for udp async sending.
func WithSendOnly() Option {
	return func(o *Options) {
		o.CallType = codec.SendOnly
		o.CallOptions = append(o.CallOptions, transport.WithReqType(codec.SendOnly))
	}
}

// WithFilter returns an Option that appends client filter to client filter chain.
// ClientFilter processing could be before encoding or after decoding.
// Selector filter is built-in filter and is at the end of the client filter chain by default.
// It is also supported to set pos of selector filter through the yaml config file.
func WithFilter(f filter.ClientFilter) Option {
	return func(o *Options) {
		o.Filters = append(o.Filters, f)
		o.FilterNames = append(o.FilterNames, "client.WithFilter")
	}
}

// WithNamedFilter returns an Option that adds named filter
func WithNamedFilter(name string, f filter.ClientFilter) Option {
	return func(o *Options) {
		o.FilterNames = append(o.FilterNames, name)
		o.Filters = append(o.Filters, f)
	}
}

// WithFilters returns an Option that appends multiple client filters to the client filter chain.
func WithFilters(fs []filter.ClientFilter) Option {
	return func(o *Options) {
		for _, f := range fs {
			WithFilter(f)(o)
		}
	}
}

// WithStreamFilters returns an Option that appends multiple client stream filters to
// the client stream filter chain.
// StreamFilter processing could be before or after stream's establishing, before or after sending data,
// before or after receiving data.
func WithStreamFilters(sfs ...StreamFilter) Option {
	return func(o *Options) {
		o.StreamFilters = append(o.StreamFilters, sfs...)
	}
}

// WithStreamFilter returns an Option that appends a client stream filter to
// the client stream filter chain.
func WithStreamFilter(sf StreamFilter) Option {
	return func(o *Options) {
		o.StreamFilters = append(o.StreamFilters, sf)
	}
}

// WithReqHead returns an Option that sets req head.
// It's default to clone server req head from source request.
func WithReqHead(h interface{}) Option {
	return func(o *Options) {
		o.ReqHead = h
	}
}

// WithRspHead returns an Option that sets rsp head.
// Usually used for gateway service.
func WithRspHead(h interface{}) Option {
	return func(o *Options) {
		o.RspHead = h
	}
}

// WithAttachment returns an Option that sets attachment.
func WithAttachment(attachment *Attachment) Option {
	return func(o *Options) {
		o.attachment = &attachment.attachment
	}
}

// WithMetaData returns an Option that sets transparent transmitted metadata.
func WithMetaData(key string, val []byte) Option {
	return func(o *Options) {
		if o.MetaData == nil {
			o.MetaData = codec.MetaData{}
		}
		o.MetaData[key] = val
	}
}

// WithSelectorNode returns an Option that records the selected node.
// It's usually used for debugging. The Node should be set for each RPC call, not just once when calling NewClientProxy.
// After the RPC is completed, the framework will automatically populate the downstream IP port and current elapsed
// time into the node. The node is not thread-safe and cannot be reused by multiple goroutines.
func WithSelectorNode(n *registry.Node) Option {
	return func(o *Options) {
		o.Node = &onceNode{Node: n}
	}
}

// WithTLS returns an Option that sets client tls files.
// If caFile="none", no server cert validation.
// If caFile="root", local ca cert will be used to validate server.
// certFile is only used for mTLS or should be empty.
// serverName is used to validate the name of server. hostname by default for https.
func WithTLS(certFile, keyFile, caFile, serverName string) Option {
	return func(o *Options) {
		if caFile == "" {
			return
		}
		o.CallOptions = append(o.CallOptions, transport.WithDialTLS(certFile, keyFile, caFile, serverName))
	}
}

// WithDisableConnectionPool returns an Option that disables connection pool.
func WithDisableConnectionPool() Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDisableConnectionPool())
	}
}

// WithMultiplexed returns an Option that enables multiplexed.
// WithMultiplexedPool should be used for custom Multiplexed.
func WithMultiplexed(enable bool) Option {
	return func(o *Options) {
		o.EnableMultiplexed = enable
	}
}

// WithLocalAddr returns an Option that sets local addr. Randomly picking for multiple NICs.
//
// for non-persistent conn, ip & port can be specified:
// client.WithLocalAddr("127.0.0.1:8080")
// for conn pool or multiplexed, only ip can be specified:
// client.WithLocalAddr("127.0.0.1:")
func WithLocalAddr(addr string) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithLocalAddr(addr))
	}
}

// WithDialTimeout returns an Option that sets timeout.
func WithDialTimeout(dur time.Duration) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDialTimeout(dur))
	}
}

// WithStreamTransport returns an Option that sets client stream transport.
func WithStreamTransport(st transport.ClientStreamTransport) Option {
	return func(o *Options) {
		if st != nil {
			o.StreamTransport = st
		}
	}
}

// WithOverloadCtrl returns an Option that sets client overload control strategy.
func WithOverloadCtrl(oc overloadctrl.OverloadController) Option {
	return func(o *Options) {
		o.OverloadCtrl = oc
	}
}

// WithMaxWindowSize returns an Option that sets max size of receive window.
// Client as the receiver will notify the sender of the window.
func WithMaxWindowSize(s uint32) Option {
	return func(o *Options) {
		o.MaxWindowSize = s
	}
}

// WithSendControl returns an Option that sets send control.
func WithSendControl(sc SendControl) Option {
	return func(o *Options) {
		o.SControl = sc
	}
}

// WithRecvControl returns an Option that sets recv control.
func WithRecvControl(rc RecvControl) Option {
	return func(o *Options) {
		o.RControl = rc
	}
}

// WithShouldErrReportToSelector returns an Option that sets should err report to selector
func WithShouldErrReportToSelector(f func(error) bool) Option {
	return func(o *Options) {
		o.shouldErrReportToSelector = f
	}
}

// WithHTTPRoundTripOptions returns an Option that sets http round trip options.
func WithHTTPRoundTripOptions(h transport.HTTPRoundTripOptions) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithHTTPRoundTripOptions(h))
	}
}

type optionsKey struct{}

func contextWithOptions(ctx context.Context, opts *Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, opts)
}

// OptionsFromContext returns options from context.
func OptionsFromContext(ctx context.Context) *Options {
	opts, ok := ctx.Value(optionsKey{}).(*Options)
	if ok {
		return opts
	}
	return NewOptions()
}

type optionsImmutability struct{}

// WithOptionsImmutable marks options of outermost layer immutable.
// Cloning options is needed for modifying options in lower layers.
//
// It should only be used by filters that call the next filter concurrently.
func WithOptionsImmutable(ctx context.Context) context.Context {
	return context.WithValue(ctx, optionsImmutability{}, optionsImmutability{})
}

// IsOptionsImmutable checks the ctx if options are immutable.
func IsOptionsImmutable(ctx context.Context) bool {
	_, ok := ctx.Value(optionsImmutability{}).(optionsImmutability)
	return ok
}

// ---------------------------- Options util api ---------------------//

// NewOptions creates a new Options with fields set to default value.
func NewOptions() *Options {
	const (
		invalidSerializationType = -1
		invalidCompressType      = -1
	)
	return &Options{
		Transport:         transport.DefaultClientTransport,
		StreamTransport:   transport.DefaultClientStreamTransport,
		Selector:          selector.DefaultSelector,
		OverloadCtrl:      overloadctrl.NoopOC{},
		SerializationType: invalidSerializationType, // the initial value is -1
		// CurrentSerializationType is the serialization type of caller itself.
		// SerializationType is the serialization type of backend service.
		// For proxy, CurrentSerializationType should be noop but SerializationType should not.
		CurrentSerializationType: invalidSerializationType,
		CurrentCompressType:      invalidCompressType,

		methods: make(map[string]*methodOptions),

		fixTimeout:                func(err error) error { return err },
		shouldErrReportToSelector: func(err error) bool { return false },
	}
}

// clone clones new options to ensure thread safety.
// Note that this is a shallow copy.
func (opts *Options) clone() *Options {
	if opts == nil {
		return NewOptions()
	}
	o := *opts
	return &o
}

// rebuildSliceCapacity rebuilds slice capacity.
// Since new options will be cloned for each RPC,
// to prevent that appending slice may affect the original data of the slice,
// cap of slice should be set to equal len of slice so that a new slice will be
// created for each slice appending.
func (opts *Options) rebuildSliceCapacity() {
	if len(opts.CallOptions) != cap(opts.CallOptions) {
		o := make([]transport.RoundTripOption, len(opts.CallOptions))
		copy(o, opts.CallOptions)
		opts.CallOptions = o
	}
	if len(opts.SelectOptions) != cap(opts.SelectOptions) {
		o := make([]selector.Option, len(opts.SelectOptions))
		copy(o, opts.SelectOptions)
		opts.SelectOptions = o
	}
	if len(opts.Filters) != cap(opts.Filters) {
		o := make(filter.ClientChain, len(opts.Filters), len(opts.Filters))
		copy(o, opts.Filters)
		opts.Filters = o
	}
	if len(opts.FilterNames) != cap(opts.FilterNames) {
		o := make([]string, len(opts.FilterNames), len(opts.FilterNames))
		copy(o, opts.FilterNames)
		opts.FilterNames = o
	}
}

func (opts *Options) parseTarget() error {
	if opts.Target == "" {
		return nil
	}

	// Target should be like: selector://endpoint.
	substr := "://"
	index := strings.Index(opts.Target, substr)
	if index == -1 {
		return fmt.Errorf("client: target %s scheme invalid, format must be selector://endpoint", opts.Target)
	}
	opts.Selector = selector.Get(opts.Target[:index])
	if opts.Selector == nil {
		return fmt.Errorf("client: selector %s not exist", opts.Target[:index])
	}
	opts.endpoint = opts.Target[index+len(substr):]
	if opts.endpoint == "" {
		return fmt.Errorf("client: target %s endpoint empty, format must be selector://endpoint", opts.Target)
	}

	return nil
}

// LoadNodeConfig loads node config from config center.
func (opts *Options) LoadNodeConfig(node *registry.Node) {
	opts.CallOptions = append(opts.CallOptions, transport.WithDialAddress(node.Address))
	// Naming service has higher priority.
	// Use network from local config file only if it's not set by the naming service.
	if node.Network != "" {
		opts.Network = node.Network
		opts.CallOptions = append(opts.CallOptions, transport.WithDialNetwork(node.Network))
	}
	if node.Protocol != "" {
		WithProtocol(node.Protocol)(opts)
	}
}

// ------------------------------ the following code is deprecated ------------------------------ //
// Deprecated

// LoadClientConfig loads client config by key which is
// the callee service name from the proto file by default.
// Deprecated
func (opts *Options) LoadClientConfig(key string) error {
	cfg := Config(key)
	if err := opts.SetNamingOptions(cfg); err != nil {
		return err
	}

	opts.OverloadCtrl = &cfg.OverloadCtrl
	if cfg.Timeout > 0 {
		opts.Timeout = time.Duration(cfg.Timeout) * time.Millisecond
	}
	if cfg.Serialization != nil {
		opts.SerializationType = *cfg.Serialization
	}

	if icodec.IsValidCompressType(cfg.Compression) && cfg.Compression != codec.CompressTypeNoop {
		opts.CompressType = cfg.Compression
	}
	if cfg.Protocol != "" {
		o := WithProtocol(cfg.Protocol)
		o(opts)
	}
	if cfg.Network != "" {
		opts.Network = cfg.Network
		opts.CallOptions = append(opts.CallOptions, transport.WithDialNetwork(cfg.Network))
	}
	if cfg.Password != "" {
		opts.CallOptions = append(opts.CallOptions, transport.WithDialPassword(cfg.Password))
	}
	if cfg.CACert != "" {
		opts.CallOptions = append(opts.CallOptions,
			transport.WithDialTLS(cfg.TLSCert, cfg.TLSKey, cfg.CACert, cfg.TLSServerName))
	}
	if cfg.Scope != "" {
		opts.Scope = cfg.Scope
	}
	return nil
}

// SetNamingOptions sets naming related options.
// Deprecated
func (opts *Options) SetNamingOptions(cfg *BackendConfig) error {
	if cfg.ServiceName != "" {
		opts.ServiceName = cfg.ServiceName
		opts.endpoint = cfg.ServiceName
	}
	if cfg.Namespace != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithNamespace(cfg.Namespace))
	}
	if cfg.EnvName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationEnvName(cfg.EnvName))
	}
	if cfg.SetName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationSetName(cfg.SetName))
	}
	if cfg.DisableServiceRouter {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDisableServiceRouter())
		opts.DisableServiceRouter = true
	}
	if cfg.ReportAnyErrToSelector {
		opts.shouldErrReportToSelector = func(err error) bool { return true }
	}
	if cfg.Target != "" {
		opts.Target = cfg.Target
		return nil
	}
	if cfg.Discovery != "" {
		d := discovery.Get(cfg.Discovery)
		if d == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: discovery %s no registered", cfg.Discovery))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDiscovery(d))
	}
	if cfg.ServiceRouter != "" {
		r := servicerouter.Get(cfg.ServiceRouter)
		if r == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: servicerouter %s no registered", cfg.ServiceRouter))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithServiceRouter(r))
	}
	if cfg.Loadbalance != "" {
		balancer := loadbalance.Get(cfg.Loadbalance)
		if balancer == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: balancer %s no registered", cfg.Loadbalance))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithLoadBalancer(balancer))
	}
	if cfg.Circuitbreaker != "" {
		cb := circuitbreaker.Get(cfg.Circuitbreaker)
		if cb == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: circuitbreaker %s no registered", cfg.Circuitbreaker))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithCircuitBreaker(cb))
	}
	return nil
}

// LoadClientFilterConfig loads client filter config by key.
// Deprecated
func (opts *Options) LoadClientFilterConfig(key string) error {
	if opts.DisableFilter {
		opts.Filters = filter.EmptyChain
		return nil
	}
	cfg := Config(key)
	for _, filterName := range cfg.Filter {
		f := filter.GetClient(filterName)
		if f == nil {
			if filterName == DefaultSelectorFilterName {
				opts.selectorFilterPosFixed = true
				opts.Filters = append(opts.Filters, selectorFilter)
				opts.FilterNames = append(opts.FilterNames, DefaultSelectorFilterName)
				continue
			}
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: filter %s no registered", filterName))
		}
		opts.Filters = append(opts.Filters, f)
		opts.FilterNames = append(opts.FilterNames, filterName)
	}
	return nil
}
