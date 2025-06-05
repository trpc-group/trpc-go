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
	"fmt"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/filter"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	inet "trpc.group/trpc-go/trpc-go/internal/net"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/scope"
	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/httppool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

// BackendConfig defines the configuration needed to call the backend service.
// It's empty by default and can be replaced.
type BackendConfig struct {
	// Callee is the name of the backend service.
	// The config file uses it as the key to set the parameters.
	// Usually, it is the proto name of the callee service defined in proto stub file,
	// and it is the same as ServiceName below.
	Callee        string `yaml:"callee,omitempty"`          // Name of the backend service.
	ServiceName   string `yaml:"name,omitempty"`            // Backend service name.
	Tag           string `yaml:"tag,omitempty"`             // Tag is extended identifier for config.
	EnvName       string `yaml:"env_name,omitempty"`        // Env name of the callee.
	SetName       string `yaml:"set_name,omitempty"`        // "Set" name of the callee.
	CallerEnvName string `yaml:"caller_env_name,omitempty"` // Env name of the caller.
	CallerSetName string `yaml:"caller_set_name,omitempty"` // "Set" name of the caller.

	// DisableServiceRouter, despite its inherent inappropriate and vague nomenclature,
	// is an option for naming service that denotes the de-facto meaning of disabling
	// out-rule routing for the source service.
	DisableServiceRouter bool              `yaml:"disable_servicerouter,omitempty"`
	Namespace            string            `yaml:"namespace,omitempty"`        // Namespace of the callee: Production/Development.
	CallerNamespace      string            `yaml:"caller_namespace,omitempty"` // Namespace of the caller: Production/Development.
	CallerMetadata       map[string]string `yaml:"caller_metadata,omitempty"`  // Set caller metadata.
	CalleeMetadata       map[string]string `yaml:"callee_metadata,omitempty"`  // Set callee metadata.

	OverloadCtrl overloadctrl.Impl `yaml:"overload_ctrl,omitempty"` // Overload control.

	Target   string `yaml:"target,omitempty"`   // Polaris by default, generally no need to configure this.
	Password string `yaml:"password,omitempty"` // Password for authentication.

	// Naming service four swordsmen.
	// Discovery.List => ServiceRouter.Filter => Loadbalancer.Select => Circuitbreaker.Report
	Discovery      string `yaml:"discovery,omitempty"`      // Discovery for the backend service.
	ServiceRouter  string `yaml:"servicerouter,omitempty"`  // Service router for the backend service.
	Loadbalance    string `yaml:"loadbalance,omitempty"`    // Load balancing algorithm.
	Circuitbreaker string `yaml:"circuitbreaker,omitempty"` // Circuit breaker configuration.

	Network   string `yaml:"network,omitempty"`   // Transport protocol type: tcp or udp.
	Timeout   int    `yaml:"timeout,omitempty"`   // Client timeout in milliseconds.
	Protocol  string `yaml:"protocol,omitempty"`  // Business protocol type: trpc, http, http_no_protocol, etc.
	Transport string `yaml:"transport,omitempty"` // Transport type.

	Method map[string]*MethodConfig `yaml:"method,omitempty"`

	// Serialization type. Use a pointer to check if it has been set (0 means pb).
	Serialization *int `yaml:"serialization,omitempty"`
	Compression   int  `yaml:"compression,omitempty"` // Compression type.

	TLSKey  string `yaml:"tls_key,omitempty"`  // Client TLS key.
	TLSCert string `yaml:"tls_cert,omitempty"` // Client TLS certificate.
	// CA certificate used to validate the server cert when calling a TLS service (e.g., an HTTPS server).
	CACert string `yaml:"ca_cert,omitempty"`
	// Server name used to validate the server (default: hostname) when calling an HTTPS server.
	TLSServerName string `yaml:"tls_server_name,omitempty"`

	Filter       []string `yaml:"filter,omitempty"`        // Filters for the backend service.
	StreamFilter []string `yaml:"stream_filter,omitempty"` // Stream filters for the backend service.

	// Report any error to the selector if this value is true.
	ReportAnyErrToSelector bool `yaml:"report_any_err_to_selector,omitempty"`

	// ConnType decides connection type to use: "connpool" for connection pool, "multiplexed" for multiplexed pool.
	ConnType *ConnType `yaml:"conn_type,omitempty"`
	// Connpool specifies the detailed configuration for connection pool.
	Connpool ConnpoolConfig `yaml:"connpool,omitempty"`
	// Multiplexed specifies the detailed configuration for multiplexed pool.
	Multiplexed MultiplexedConfig `yaml:"multiplexed,omitempty"`
	// HTTPPool specifies the detailed configuration for http pool.
	HTTPPool HTTPPoolConfig `yaml:"httppool,omitempty"`

	// Scope specifies the current scope of the backend service.
	Scope   scope.Scope `yaml:"scope,omitempty"`
	LocalIP string
}

// ConnpoolConfig defines the configuration for connection pool.
type ConnpoolConfig struct {
	// DialTimeout decides dial timeout, default 200ms.
	DialTimeout *time.Duration `yaml:"dial_timeout,omitempty"`
	// ForceClose decides whether force close the connection, default false.
	ForceClose *bool `yaml:"force_close,omitempty"`
	// IdleTimeout decides idle timeout, default 50s.
	IdleTimeout *time.Duration `yaml:"idle_timeout,omitempty"`
	// MaxActive decides max active connections, default 0 (means no limit).
	MaxActive *int `yaml:"max_active,omitempty"`
	// MaxConnLifetime decides max lifetime for connection, default 0s (means no limit).
	MaxConnLifetime *time.Duration `yaml:"max_conn_lifetime,omitempty"`
	// MaxIdle decides max idle connections, default 65536.
	MaxIdle *int `yaml:"max_idle,omitempty"`
	// MinIdle decides min idle connections, default 0.
	MinIdle *int `yaml:"min_idle,omitempty"`
	// PoolIdleTimeout decides idle timeout to close the entire pool, default 100s.
	PoolIdleTimeout *time.Duration `yaml:"pool_idle_timeout,omitempty"`
	// PushIdleConnToTail decides recycle the connection to head/tail of the idle list, default false (head).
	PushIdleConnToTail *bool `yaml:"push_idle_conn_to_tail,omitempty"`
	// Wait decides whether wait util timeout or return err immediately when
	// the number of total connections reach max_active, default false.
	Wait *bool `yaml:"wait,omitempty"`
}

// MultiplexedConfig defines the configuration for multiplexed pool.
type MultiplexedConfig struct {
	// MultiplexedDialTimeout decides dial timeout, default 1s.
	MultiplexedDialTimeout *time.Duration `yaml:"multiplexed_dial_timeout,omitempty"`
	// ConnsPerHost decides the number of concrete(real) connections for each host, default 2.
	ConnsPerHost *int `yaml:"conns_per_host,omitempty"`
	// MaxVirConnsPerConn decides the max number of virtual connections for
	// each concrete(real) connection, default 0 (means no limit).
	MaxVirConnsPerConn *int `yaml:"max_vir_conns_per_conn,omitempty"`
	// MaxIdleConnsPerHost decides the max number of idle concrete(real) connections for each host,
	// used together with max_vir_conns_per_conn, default 0 (disabled).
	MaxIdleConnsPerHost *int `yaml:"max_idle_conns_per_host,omitempty"`
	// QueueSize decides the size of send queue for each concrete(real) connection, default 1024.
	QueueSize *int `yaml:"queue_size,omitempty"`
	// DropFull decides whether to drop the send package when queue is full, default false.
	DropFull *bool `yaml:"drop_full,omitempty"`
	// EnableMetrics decides whether to enable metrics, used in tnet-multiplexed only, default false.
	EnableMetrics *bool `yaml:"enable_metrics,omitempty"`
	// MaxReconnectCount decides the maximum number of reconnection attempts, 0 means reconnect is disable, default 10.
	MaxReconnectCount *int `yaml:"max_reconnect_count,omitempty"`
	// InitialBackoff decides the initial backoff time during the first reconnection attempt, default 5ms.
	InitialBackoff *time.Duration `yaml:"initial_backoff,omitempty"`
	// ReconnectCountResetInterval decides the time to reset the reconnect counts,
	// default is 2*[sum(dialTimeout) + sum(backoff)].
	ReconnectCountResetInterval *time.Duration `yaml:"reconnect_count_reset_interval,omitempty"`
}

// HTTPPoolConfig defines the configuration for http pool.
type HTTPPoolConfig struct {
	// MaxIdleConns controls the maximum number of idle connections across all hosts, default 0, which means no limit.
	MaxIdleConns *int `yaml:"max_idle_conns,omitempty"`
	// MaxIdleConnsPerHost controls the maximum idle connections to keep per-host, default 2.
	MaxIdleConnsPerHost *int `yaml:"max_idle_conns_per_host,omitempty"`
	// MaxConnsPerHost optionally limits the total number of connections per host, default 0, which means no limit.
	MaxConnsPerHost *int `yaml:"max_conns_per_host,omitempty"`
	// IdleConnTimeout is the maximum amount of time an idle connection will remain idle before closing,
	// default 0, which means no limit.
	IdleConnTimeout *time.Duration `yaml:"idle_conn_timeout,omitempty"`
}

// MethodConfig is the method level configurations.
type MethodConfig struct {
	Timeout *int `yaml:"timeout,omitempty"` // ms
}

// UnmarshalYAML sets default values for BackendConfig on yaml unmarshal.
func (cfg *BackendConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Introduce a tmp type which does not implement UnmarshalYAML to prevent infinite loop.
	type tmp BackendConfig
	if err := unmarshal((*tmp)(cfg)); err != nil {
		return err
	}

	// Repair Callee & ServiceName, referring to repairClientConfig.
	name := cfg.ServiceName
	if name == "" {
		name = cfg.Callee
	}

	return cfg.OverloadCtrl.Build(overloadctrl.GetClient, &overloadctrl.ServiceMethodInfo{
		ServiceName: name,
		MethodName:  overloadctrl.AnyMethod,
	})
}

// genOptions generates options for each RPC from BackendConfig.
func (cfg *BackendConfig) genOptions() (*Options, error) {
	opts := NewOptions()
	opts.localAddr = inet.ResolveAddress(cfg.Network, cfg.LocalIP+":0")
	if err := cfg.setNamingOptions(opts); err != nil {
		return nil, err
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

	WithTransport(transport.GetClientTransport(cfg.Transport))(opts)
	WithStreamTransport(transport.GetClientStreamTransport(cfg.Transport))(opts)
	WithProtocol(cfg.Protocol)(opts)
	WithNetwork(cfg.Network)(opts)
	WithPassword(cfg.Password)(opts)
	WithTLS(cfg.TLSCert, cfg.TLSKey, cfg.CACert, cfg.TLSServerName)(opts)
	if cfg.Protocol != "" && opts.Codec == nil {
		return nil, fmt.Errorf("codec %s not exists", cfg.Protocol)
	}
	WithScope(cfg.Scope)(opts)
	if err := cfg.setClientPool(opts); err != nil {
		return nil, fmt.Errorf("set client pool: %w", err)
	}
	for method, methodConfig := range cfg.Method {
		var methodTimeout *time.Duration
		if methodConfig.Timeout != nil {
			timeout := time.Millisecond * time.Duration(*methodConfig.Timeout)
			methodTimeout = &timeout
		}
		opts.methods[method] = &methodOptions{timeout: methodTimeout}
	}
	for _, name := range cfg.Filter {
		f := filter.GetClient(name)
		if f == nil {
			if name == DefaultSelectorFilterName {
				// selector filter is configured
				// need to set selector filter pos
				opts.selectorFilterPosFixed = true
				opts.Filters = append(opts.Filters, selectorFilter)
				opts.FilterNames = append(opts.FilterNames, name)
				continue
			}
			return nil, fmt.Errorf("client config: filter %s no registered, do not configure", name)
		}
		opts.Filters = append(opts.Filters, f)
		opts.FilterNames = append(opts.FilterNames, name)
	}
	for _, name := range cfg.StreamFilter {
		f := GetStreamFilter(name)
		if f == nil {
			return nil, fmt.Errorf("client config: stream filter %s no registered, do not configure", name)
		}
		opts.StreamFilters = append(opts.StreamFilters, f)
	}
	opts.rebuildSliceCapacity()
	return opts, nil
}

// ConnType defines the connection type for backend.
type ConnType string

const (
	// ConnTypeConnPool represents the connection type that uses a connection pool mode.
	ConnTypeConnPool ConnType = "connpool"
	// ConnTypeMultiplexedPool represents the connection type that uses multiplexing mode.
	ConnTypeMultiplexedPool ConnType = "multiplexed"
	// ConnTypeShort represents the connection type that uses short-lived connections.
	ConnTypeShort ConnType = "short"
	// ConnTypeHTTPPool represents the connection type that uses a http pool mode.
	ConnTypeHTTPPool ConnType = "httppool"
)

func (cfg *BackendConfig) multiplexedEnabled() bool {
	return cfg.ConnType != nil && *cfg.ConnType == ConnTypeMultiplexedPool
}

// setClientPool configures the client pool options based on the transport and connection type
// specified in the BackendConfig.
func (cfg *BackendConfig) setClientPool(opts *Options) error {
	if cfg.ConnType == nil {
		return nil
	}
	var (
		transportName string = cfg.Transport
		roundTripOpt  transport.RoundTripOption
		err           error
	)
	// Determine the transport to use; default to the configured transport unless overridden
	// by a specific protocol, like http protocol.
	if transportName == "" && cfg.Protocol != "" && transport.GetClientTransport(cfg.Protocol) != nil {
		transportName = cfg.Protocol
	}

	switch transportName {
	case protocol.TNET:
		roundTripOpt, err = cfg.tnetClientPoolOption()
	case protocol.HTTP:
		roundTripOpt, err = cfg.httpClientPoolOption()
	default:
		roundTripOpt, err = cfg.gonetClientPoolOption()
	}
	if err != nil {
		return err
	}
	opts.CallOptions = append(opts.CallOptions, roundTripOpt)
	opts.EnableMultiplexed = cfg.multiplexedEnabled()
	return nil
}

func (cfg *BackendConfig) gonetClientPoolOption() (transport.RoundTripOption, error) {
	switch *cfg.ConnType {
	case ConnTypeShort:
		return transport.WithDisableConnectionPool(), nil
	case ConnTypeConnPool:
		return cfg.clientConnPoolOption(), nil
	case ConnTypeMultiplexedPool:
		return cfg.clientMultiplexedPoolOption(), nil
	case ConnTypeHTTPPool:
		// Default transport doesn't use http pool, but custom transport maybe use it.
		return cfg.httpClientHTTPPoolOption(), nil
	default:
		return nil,
			fmt.Errorf("invalid connection type %v; supported connection types are [%v, %v, %v, %v]",
				*cfg.ConnType, ConnTypeShort, ConnTypeConnPool, ConnTypeMultiplexedPool, ConnTypeHTTPPool)
	}
}

func (cfg *BackendConfig) clientConnPoolOption() transport.RoundTripOption {
	return transport.WithDialPool(connpool.NewConnectionPool(cfg.connpoolOptions()...))
}

func (cfg *BackendConfig) connpoolOptions() []connpool.Option {
	var opts []connpool.Option
	if cfg.Connpool.DialTimeout != nil {
		opts = append(opts, connpool.WithDialTimeout(*cfg.Connpool.DialTimeout))
	}
	if cfg.Connpool.ForceClose != nil {
		opts = append(opts, connpool.WithForceClose(*cfg.Connpool.ForceClose))
	}
	if cfg.Connpool.IdleTimeout != nil {
		opts = append(opts, connpool.WithIdleTimeout(*cfg.Connpool.IdleTimeout))
	}
	if cfg.Connpool.MaxActive != nil {
		opts = append(opts, connpool.WithMaxActive(*cfg.Connpool.MaxActive))
	}
	if cfg.Connpool.MaxConnLifetime != nil {
		opts = append(opts, connpool.WithMaxConnLifetime(*cfg.Connpool.MaxConnLifetime))
	}
	if cfg.Connpool.MaxIdle != nil {
		opts = append(opts, connpool.WithMaxIdle(*cfg.Connpool.MaxIdle))
	}
	if cfg.Connpool.MinIdle != nil {
		opts = append(opts, connpool.WithMinIdle(*cfg.Connpool.MinIdle))
	}
	if cfg.Connpool.PoolIdleTimeout != nil {
		opts = append(opts, connpool.WithPoolIdleTimeout(*cfg.Connpool.PoolIdleTimeout))
	}
	if cfg.Connpool.PushIdleConnToTail != nil {
		opts = append(opts, connpool.WithPushIdleConnToTail(*cfg.Connpool.PushIdleConnToTail))
	}
	if cfg.Connpool.Wait != nil {
		opts = append(opts, connpool.WithWait(*cfg.Connpool.Wait))
	}
	return opts
}

func (cfg *BackendConfig) clientMultiplexedPoolOption() transport.RoundTripOption {
	var opts []multiplexed.PoolOption
	if cfg.Multiplexed.MultiplexedDialTimeout != nil {
		opts = append(opts, multiplexed.WithDialTimeout(*cfg.Multiplexed.MultiplexedDialTimeout))
	}
	if cfg.Multiplexed.ConnsPerHost != nil {
		opts = append(opts, multiplexed.WithConnectNumber(*cfg.Multiplexed.ConnsPerHost))
	}
	if cfg.Multiplexed.MaxVirConnsPerConn != nil {
		opts = append(opts, multiplexed.WithMaxVirConnsPerConn(*cfg.Multiplexed.MaxVirConnsPerConn))
	}
	if cfg.Multiplexed.MaxIdleConnsPerHost != nil {
		opts = append(opts, multiplexed.WithMaxIdleConnsPerHost(*cfg.Multiplexed.MaxIdleConnsPerHost))
	}
	if cfg.Multiplexed.QueueSize != nil {
		opts = append(opts, multiplexed.WithQueueSize(*cfg.Multiplexed.QueueSize))
	}
	if cfg.Multiplexed.DropFull != nil {
		opts = append(opts, multiplexed.WithDropFull(*cfg.Multiplexed.DropFull))
	}
	if cfg.Multiplexed.MaxReconnectCount != nil {
		opts = append(opts, multiplexed.WithMaxReconnectCount(*cfg.Multiplexed.MaxReconnectCount))
	}
	if cfg.Multiplexed.InitialBackoff != nil {
		opts = append(opts, multiplexed.WithInitialBackoff(*cfg.Multiplexed.InitialBackoff))
	}
	if cfg.Multiplexed.ReconnectCountResetInterval != nil {
		opts = append(opts, multiplexed.WithReconnectCountResetInterval(*cfg.Multiplexed.ReconnectCountResetInterval))
	}
	return transport.WithMultiplexedPool(multiplexed.New(opts...))
}

func (cfg *BackendConfig) httpClientPoolOption() (transport.RoundTripOption, error) {
	switch *cfg.ConnType {
	case ConnTypeShort:
		return transport.WithDisableConnectionPool(), nil
	case ConnTypeHTTPPool:
		return cfg.httpClientHTTPPoolOption(), nil
	default:
		return nil,
			fmt.Errorf("transport %v doesn't support connection type %v; supported connection types are [%v, %v]",
				protocol.HTTP, *cfg.ConnType, ConnTypeShort, ConnTypeHTTPPool)
	}
}

func (cfg *BackendConfig) httpClientHTTPPoolOption() transport.RoundTripOption {
	poolOpts := httppool.Options{}
	if cfg.HTTPPool.MaxIdleConns != nil {
		poolOpts.MaxIdleConns = *cfg.HTTPPool.MaxIdleConns
	}
	if cfg.HTTPPool.MaxIdleConnsPerHost != nil {
		poolOpts.MaxIdleConnsPerHost = *cfg.HTTPPool.MaxIdleConnsPerHost
	}
	if cfg.HTTPPool.MaxConnsPerHost != nil {
		poolOpts.MaxConnsPerHost = *cfg.HTTPPool.MaxConnsPerHost
	}
	if cfg.HTTPPool.IdleConnTimeout != nil {
		poolOpts.IdleConnTimeout = *cfg.HTTPPool.IdleConnTimeout
	}
	httpOpts := transport.HTTPRoundTripOptions{Pool: poolOpts}
	return transport.WithHTTPRoundTripOptions(httpOpts)
}

// setNamingOptions sets naming related options.
func (cfg *BackendConfig) setNamingOptions(opts *Options) error {
	if cfg.ServiceName != "" {
		opts.ServiceName = cfg.ServiceName
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
	if cfg.CallerNamespace != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithSourceNamespace(cfg.CallerNamespace))
	}
	if cfg.CallerEnvName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithSourceEnvName(cfg.CallerEnvName))
	}
	if cfg.CallerSetName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithSourceSetName(cfg.CallerSetName))
	}
	if cfg.DisableServiceRouter {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDisableServiceRouter())
		opts.DisableServiceRouter = true
	}
	if cfg.ReportAnyErrToSelector {
		opts.shouldErrReportToSelector = func(err error) bool { return true }
	}
	for key, val := range cfg.CallerMetadata {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithSourceMetadata(key, val))
	}
	for key, val := range cfg.CalleeMetadata {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationMetadata(key, val))
	}
	if cfg.Target != "" {
		opts.Target = cfg.Target
		return opts.parseTarget()
	}
	if cfg.Discovery != "" {
		d := discovery.Get(cfg.Discovery)
		if d == nil {
			return fmt.Errorf("client config: discovery %s no registered", cfg.Discovery)
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDiscovery(d))
	}
	if cfg.ServiceRouter != "" {
		r := servicerouter.Get(cfg.ServiceRouter)
		if r == nil {
			return fmt.Errorf("client config: servicerouter %s no registered", cfg.ServiceRouter)
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithServiceRouter(r))
	}
	if cfg.Loadbalance != "" {
		balancer := loadbalance.Get(cfg.Loadbalance)
		if balancer == nil {
			return fmt.Errorf("client config: balancer %s no registered", cfg.Loadbalance)
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithLoadBalancer(balancer))
	}
	if cfg.Circuitbreaker != "" {
		cb := circuitbreaker.Get(cfg.Circuitbreaker)
		if cb == nil {
			return fmt.Errorf("client config: circuitbreaker %s no registered", cfg.Circuitbreaker)
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithCircuitBreaker(cb))
	}
	return nil
}

var (
	// DefaultSelectorFilterName is the default name of selector filter.
	// It can be modified if conflict exists.
	DefaultSelectorFilterName = "selector"

	defaultBackendConf = &BackendConfig{
		Network:  protocol.TCP,
		Protocol: protocol.TRPC,
	}
	defaultBackendOptions *Options

	mutex   sync.RWMutex
	configs = make(map[string]*configsWithFallback) // Key: callee.
	options = make(map[string]*optionsWithFallback) // Key: callee.
)

type configsWithFallback struct {
	fallback4ServiceName *BackendConfig
	serviceNames         map[string]map[string]*BackendConfig // Key: service name => tag
}

type optionsWithFallback struct {
	fallback4ServiceName *Options
	serviceNames         map[string]map[string]*Options // Key: service name => tag
}

// getDefaultOptions returns default options.
func getDefaultOptions() *Options {
	mutex.RLock()
	opts := defaultBackendOptions
	mutex.RUnlock()
	if opts != nil {
		return opts
	}

	mutex.Lock()
	defer mutex.Unlock()
	if defaultBackendOptions != nil {
		return defaultBackendOptions
	}
	opts, err := defaultBackendConf.genOptions()
	if err != nil {
		defaultBackendOptions = NewOptions()
	} else {
		defaultBackendOptions = opts
	}
	return defaultBackendOptions
}

// DefaultClientConfig returns the default client config.
//
// Note: if multiple client configs with same callee and different service name
// exist in trpc_go.yaml, this function will only return the last config for
// the same callee key.
func DefaultClientConfig() map[string]*BackendConfig {
	mutex.RLock()
	c := make(map[string]*BackendConfig, len(configs))
	for k, v := range configs {
		c[k] = v.fallback4ServiceName
	}
	mutex.RUnlock()
	return c
}

// LoadClientConfig loads client config by path.
func LoadClientConfig(path string, opts ...config.LoadOption) error {
	conf, err := config.DefaultConfigLoader.Load(path, opts...)
	if err != nil {
		return err
	}
	tmp := make(map[string]*BackendConfig)
	if err := conf.Unmarshal(tmp); err != nil {
		return err
	}
	return RegisterConfig(tmp)
}

// Config returns BackendConfig by callee service name.
// Deprecated: use GetConfig instead.
func Config(callee string) *BackendConfig {
	mutex.RLock()
	if len(configs) == 0 {
		mutex.RUnlock()
		return defaultBackendConf
	}
	conf, ok := configs[callee]
	if !ok {
		conf, ok = configs["*"]
		if !ok {
			mutex.RUnlock()
			return defaultBackendConf
		}
	}
	mutex.RUnlock()
	return conf.fallback4ServiceName
}

// GetConfig returns BackendConfig by callee and service name.
// If service name is empty or not found in callee configurations,
// it returns the callee config.
// If callee and service name are both not found, it returns the default config(registered with name "*").
// If no config is found, it returns an error.
func GetConfig(callee, serviceName string) (*BackendConfig, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	conf, ok := configs[callee]
	if !ok {
		conf, ok = configs["*"]
		if !ok {
			return nil, fmt.Errorf(
				"client config: callee %s service name %s not found",
				callee, serviceName,
			)
		}
		return conf.fallback4ServiceName, nil
	}

	if serviceName == "" {
		serviceName = callee
	}

	cs, ok := conf.serviceNames[serviceName]
	if !ok {
		return conf.fallback4ServiceName, nil
	}

	if c, ok := cs[""]; ok {
		return c, nil
	}

	return conf.fallback4ServiceName, nil
}

// getConfigWithTag returns BackendConfig by callee, service name and tag.
// If no exact config is found, it returns an error.
func getConfigWithTag(callee, serviceName, tag string) (*BackendConfig, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	conf, ok := configs[callee]
	if !ok {
		return nil, fmt.Errorf(
			"client config: callee %s service name %s tag %s not found",
			callee, serviceName, tag,
		)
	}

	cs, ok := conf.serviceNames[serviceName]
	if !ok {
		return nil, fmt.Errorf(
			"client config: callee %s service name %s tag %s not found",
			callee, serviceName, tag,
		)
	}

	if c, ok := cs[tag]; ok {
		return c, nil
	}

	return nil, fmt.Errorf(
		"client config: callee %s service name %s tag %s not found",
		callee, serviceName, tag,
	)
}

func getOptionsByCalleeAndUserOptions(callee string, opt ...Option) (*Options, error) {
	// Each RPC call uses new options to ensure thread safety.
	inputOpts := &Options{}
	for _, o := range opt {
		o(inputOpts)
	}

	// If user passes in a tag option, use callee, service name and tag as a combined key to retrieve client config.
	// When using the 'tag' option, it is mandatory to include the 'ServiceName' option.
	// This is because the 'tag' option performs precise matching logic and is typically used only when the
	// 'callee' and 'ServiceName' cannot distinguish the config.
	if inputOpts.Tag != "" {
		return getOptionsByCalleeAndServiceNameAndTag(callee, inputOpts.ServiceName, inputOpts.Tag)
	}

	// If user passes in a service name option, use callee and service name
	// as a combined key to retrieve client config.
	if inputOpts.ServiceName != "" {
		return getOptionsByCalleeAndServiceName(callee, inputOpts.ServiceName), nil
	}

	// Otherwise use callee only.
	return getOptionsByCallee(callee), nil
}

// getOptionsByCallee returns Options by callee service name.
func getOptionsByCallee(callee string) *Options {
	mutex.RLock()
	if len(options) == 0 {
		mutex.RUnlock()
		return getDefaultOptions()
	}
	opts, ok := options[callee]
	if !ok {
		opts, ok = options["*"]
		if !ok {
			mutex.RUnlock()
			return getDefaultOptions()
		}
	}
	mutex.RUnlock()
	return opts.fallback4ServiceName
}

// getOptionsByCalleeAndServiceName returns Options by callee and service name.
func getOptionsByCalleeAndServiceName(callee, serviceName string) *Options {
	mutex.RLock()

	serviceOptions, ok := options[callee]
	if !ok {
		mutex.RUnlock()
		return getOptionsByCallee(callee) // Fallback to use callee as the single key.
	}

	opts, ok := serviceOptions.serviceNames[serviceName]
	if !ok {
		mutex.RUnlock()
		return getOptionsByCallee(callee) // Fallback to use callee as the single key.
	}

	// Tag = "" means using the default tag.
	opt, ok := opts[""]
	if !ok {
		mutex.RUnlock()
		return getOptionsByCallee(callee)
	}

	mutex.RUnlock()
	return opt
}

// getOptionsByCalleeAndServiceNameAndTag returns Options by callee, service name and tag.
// If no exact option is found, it returns an error.
func getOptionsByCalleeAndServiceNameAndTag(callee, serviceName, tag string) (*Options, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	serviceOptions, ok := options[callee]
	if !ok {
		// No Fallback for tag.
		return nil, fmt.Errorf("unable to find exact matched options: "+
			"callee %s, serviceName %s, and tag %s in options, "+
			"please check for configuration errors in callee",
			callee, serviceName, tag)
	}

	opts, ok := serviceOptions.serviceNames[serviceName]
	if !ok {
		// No Fallback for tag.
		return nil, fmt.Errorf("unable to find exact matched options: "+
			"callee %s, serviceName %s, and tag %s in options, "+
			"please check for configuration errors in serviceName",
			callee, serviceName, tag)
	}

	if opt, ok := opts[tag]; ok {
		return opt, nil
	}

	return nil, fmt.Errorf("unable to find exact matched options: "+
		"callee %s, serviceName %s, and tag %s in options, "+
		"please check for configuration errors in tag",
		callee, serviceName, tag)
}

// RegisterConfig is called to replace the global backend config,
// allowing updating backend config regularly.
func RegisterConfig(conf map[string]*BackendConfig) error {
	opts := make(map[string]*optionsWithFallback)
	confs := make(map[string]*configsWithFallback)
	for key, cfg := range conf {
		o, err := cfg.genOptions()
		if err != nil {
			return err
		}
		opts[key] = &optionsWithFallback{
			fallback4ServiceName: o,
			serviceNames:         make(map[string]map[string]*Options),
		}
		opts[key].serviceNames[cfg.ServiceName] = make(map[string]*Options)
		opts[key].serviceNames[cfg.ServiceName][cfg.Tag] = o
		if cfg.Tag != "" {
			opts[key].serviceNames[cfg.ServiceName][""] = o
		}

		confs[key] = &configsWithFallback{
			fallback4ServiceName: cfg,
			serviceNames:         make(map[string]map[string]*BackendConfig),
		}
		confs[key].serviceNames[cfg.ServiceName] = make(map[string]*BackendConfig)
		confs[key].serviceNames[cfg.ServiceName][cfg.Tag] = cfg
		if cfg.Tag != "" {
			confs[key].serviceNames[cfg.ServiceName][""] = cfg
		}
	}
	mutex.Lock()
	options = opts
	configs = confs
	mutex.Unlock()
	return nil
}

// RegisterClientConfig is called to replace backend config of single callee service by name.
func RegisterClientConfig(callee string, conf *BackendConfig) error {
	if callee == "*" {
		// Reset the callee, service name and tag to enable wildcard matching.
		conf.Callee = ""
		conf.ServiceName = ""
		conf.Tag = ""
	}
	opts, err := conf.genOptions()
	if err != nil {
		return err
	}
	mutex.Lock()
	if _, ok := options[callee]; !ok {
		options[callee] = &optionsWithFallback{
			serviceNames: make(map[string]map[string]*Options),
		}
		configs[callee] = &configsWithFallback{
			serviceNames: make(map[string]map[string]*BackendConfig),
		}
	}
	options[callee].fallback4ServiceName = opts
	configs[callee].fallback4ServiceName = conf

	if _, ok := options[callee].serviceNames[conf.ServiceName]; !ok {
		options[callee].serviceNames[conf.ServiceName] = make(map[string]*Options)
		configs[callee].serviceNames[conf.ServiceName] = make(map[string]*BackendConfig)
	}

	options[callee].serviceNames[conf.ServiceName][conf.Tag] = opts
	configs[callee].serviceNames[conf.ServiceName][conf.Tag] = conf
	if conf.Tag != "" {
		options[callee].serviceNames[conf.ServiceName][""] = opts
		configs[callee].serviceNames[conf.ServiceName][""] = conf
	}

	mutex.Unlock()
	return nil
}
