//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
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
	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
	"trpc.group/trpc-go/trpc-go/transport"
)

// BackendConfig defines the configuration needed to call the backend service.
// It's empty by default and can be replaced.
type BackendConfig struct {
	// Callee is the name of the backend service.
	// The config file uses it as the key to set the parameters.
	// Usually, it is the proto name of the callee service defined in proto stub file,
	// and it is the same as ServiceName below.
	Callee      string `yaml:"callee"`   // Name of the backend service.
	ServiceName string `yaml:"name"`     // Backend service name.
	EnvName     string `yaml:"env_name"` // Env name of the callee.
	SetName     string `yaml:"set_name"` // "Set" name of the callee.

	// DisableServiceRouter, despite its inherent inappropriate and vague nomenclature,
	// is an option for naming service that denotes the de-facto meaning of disabling
	// out-rule routing for the source service.
	DisableServiceRouter bool              `yaml:"disable_servicerouter"`
	Namespace            string            `yaml:"namespace"`       // Namespace of the callee: Production/Development.
	CalleeMetadata       map[string]string `yaml:"callee_metadata"` // Set callee metadata.

	Target   string `yaml:"target"`   // Polaris by default, generally no need to configure this.
	Password string `yaml:"password"` // Password for authentication.

	// Naming service four swordsmen.
	// Discovery.List => ServiceRouter.Filter => Loadbalancer.Select => Circuitbreaker.Report
	Discovery      string `yaml:"discovery"`      // Discovery for the backend service.
	ServiceRouter  string `yaml:"servicerouter"`  // Service router for the backend service.
	Loadbalance    string `yaml:"loadbalance"`    // Load balancing algorithm.
	Circuitbreaker string `yaml:"circuitbreaker"` // Circuit breaker configuration.

	Network   string `yaml:"network"`   // Transport protocol type: tcp or udp.
	Timeout   int    `yaml:"timeout"`   // Client timeout in milliseconds.
	Protocol  string `yaml:"protocol"`  // Business protocol type: trpc, http, http_no_protocol, etc.
	Transport string `yaml:"transport"` // Transport type.

	// Serialization type. Use a pointer to check if it has been set (0 means pb).
	Serialization *int `yaml:"serialization"`
	Compression   int  `yaml:"compression"` // Compression type.

	TLSKey  string `yaml:"tls_key"`  // Client TLS key.
	TLSCert string `yaml:"tls_cert"` // Client TLS certificate.
	// CA certificate used to validate the server cert when calling a TLS service (e.g., an HTTPS server).
	CACert string `yaml:"ca_cert"`
	// Server name used to validate the server (default: hostname) when calling an HTTPS server.
	TLSServerName string `yaml:"tls_server_name"`

	Filter       []string `yaml:"filter"`        // Filters for the backend service.
	StreamFilter []string `yaml:"stream_filter"` // Stream filters for the backend service.

	// Report any error to the selector if this value is true.
	ReportAnyErrToSelector bool `yaml:"report_any_err_to_selector"`
}

// genOptions generates options for each RPC from BackendConfig.
func (cfg *BackendConfig) genOptions() (*Options, error) {
	opts := NewOptions()
	if err := cfg.setNamingOptions(opts); err != nil {
		return nil, err
	}

	if cfg.Timeout > 0 {
		opts.Timeout = time.Duration(cfg.Timeout) * time.Millisecond
	}
	if cfg.Serialization != nil {
		opts.SerializationType = *cfg.Serialization
	}
	if icodec.IsValidCompressType(cfg.Compression) && cfg.Compression != codec.CompressTypeNoop {
		opts.CompressType = cfg.Compression
	}

	// Reset the transport to check if the user has specified any transport.
	opts.Transport = nil
	WithTransport(transport.GetClientTransport(cfg.Transport))(opts)
	WithStreamTransport(transport.GetClientStreamTransport(cfg.Transport))(opts)
	WithProtocol(cfg.Protocol)(opts)
	WithNetwork(cfg.Network)(opts)
	opts.Transport = attemptSwitchingTransport(opts)
	WithPassword(cfg.Password)(opts)
	WithTLS(cfg.TLSCert, cfg.TLSKey, cfg.CACert, cfg.TLSServerName)(opts)
	if cfg.Protocol != "" && opts.Codec == nil {
		return nil, fmt.Errorf("codec %s not exists", cfg.Protocol)
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
	if cfg.DisableServiceRouter {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDisableServiceRouter())
		opts.DisableServiceRouter = true
	}
	if cfg.ReportAnyErrToSelector {
		opts.shouldErrReportToSelector = func(err error) bool { return true }
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
		Network:  "tcp",
		Protocol: "trpc",
	}
	defaultBackendOptions *Options

	mutex   sync.RWMutex
	configs = make(map[string]*configsWithFallback) // Key: callee.
	options = make(map[string]*optionsWithFallback) // Key: callee.
)

type configsWithFallback struct {
	fallback     *BackendConfig
	serviceNames map[string]*BackendConfig // Key: service name.
}

type optionsWithFallback struct {
	fallback     *Options
	serviceNames map[string]*Options // Key: service name.
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
	if defaultBackendOptions != nil {
		mutex.Unlock()
		return defaultBackendOptions
	}
	opts, err := defaultBackendConf.genOptions()
	if err != nil {
		defaultBackendOptions = NewOptions()
	} else {
		defaultBackendOptions = opts
	}
	mutex.Unlock()
	return defaultBackendOptions
}

// DefaultClientConfig returns the default client config.
//
// Note: if multiple client configs with same callee and different service name
// exist in trpc_go.yaml, this function will only return the last config for
// the same callee key.
func DefaultClientConfig() map[string]*BackendConfig {
	mutex.RLock()
	c := make(map[string]*BackendConfig)
	for k, v := range configs {
		c[k] = v.fallback
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
	RegisterConfig(tmp)
	return nil
}

// Config returns BackendConfig by callee service name.
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
	return conf.fallback
}

func getOptionsByCalleeAndUserOptions(callee string, opt ...Option) *Options {
	// Each RPC call uses new options to ensure thread safety.
	inputOpts := &Options{}
	for _, o := range opt {
		o(inputOpts)
	}
	if inputOpts.ServiceName != "" {
		// If user passes in a service name option, use callee and service name
		// as a combined key to retrieve client config.
		return getOptionsByCalleeAndServiceName(callee, inputOpts.ServiceName)
	}
	// Otherwise use callee only.
	return getOptions(callee)
}

// getOptions returns Options by callee service name.
func getOptions(callee string) *Options {
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
	return opts.fallback
}

func getOptionsByCalleeAndServiceName(callee, serviceName string) *Options {
	mutex.RLock()
	serviceOptions, ok := options[callee]
	if !ok {
		mutex.RUnlock()
		return getOptions(callee) // Fallback to use callee as the single key.
	}
	opts, ok := serviceOptions.serviceNames[serviceName]
	if !ok {
		mutex.RUnlock()
		return getOptions(callee) // Fallback to use callee as the single key.
	}
	mutex.RUnlock()
	return opts
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
			fallback:     o,
			serviceNames: make(map[string]*Options),
		}
		opts[key].serviceNames[cfg.ServiceName] = o
		confs[key] = &configsWithFallback{
			fallback:     cfg,
			serviceNames: make(map[string]*BackendConfig),
		}
		confs[key].serviceNames[cfg.ServiceName] = cfg
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
		// Reset the callee and service name to enable wildcard matching.
		conf.Callee = ""
		conf.ServiceName = ""
	}
	opts, err := conf.genOptions()
	if err != nil {
		return err
	}
	mutex.Lock()
	if opt, ok := options[callee]; !ok || opt == nil {
		options[callee] = &optionsWithFallback{
			serviceNames: make(map[string]*Options),
		}
		configs[callee] = &configsWithFallback{
			serviceNames: make(map[string]*BackendConfig),
		}
	}
	options[callee].fallback = opts
	configs[callee].fallback = conf
	options[callee].serviceNames[conf.ServiceName] = opts
	configs[callee].serviceNames[conf.ServiceName] = conf
	mutex.Unlock()
	return nil
}
