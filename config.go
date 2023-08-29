// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package trpc

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/rpcz"

	"gopkg.in/yaml.v3"
)

// ServerConfigPath is the file path of trpc server config file.
// By default, it's ./trpc_go.yaml. It can be set by the flag -conf.
var ServerConfigPath = defaultConfigPath

const (
	defaultConfigPath  = "./trpc_go.yaml"
	defaultIdleTimeout = 60000 // in ms
)

// serverConfigPath returns the file path of trpc server config file.
// With the highest priority: modifying value of ServerConfigPath.
// With second-highest priority: setting by flag --conf or -conf.
// With third-highest priority: using ./trpc_go.yaml as default path.
func serverConfigPath() string {
	if ServerConfigPath == defaultConfigPath && !flag.Parsed() {
		flag.StringVar(&ServerConfigPath, "conf", defaultConfigPath, "server config path")
		flag.Parse()
	}
	return ServerConfigPath
}

// Config is the configuration for trpc, which can be divided into 4 parts:
// 1. Global config.
// 2. Server config.
// 3. Client config.
// 4. Plugins config.
type Config struct {
	Global struct {
		Namespace     string `yaml:"namespace"`      // Namespace for the configuration.
		EnvName       string `yaml:"env_name"`       // Environment name.
		ContainerName string `yaml:"container_name"` // Container name.
		LocalIP       string `yaml:"local_ip"`       // Local IP address.
		EnableSet     string `yaml:"enable_set"`     // Y/N. Whether to enable Set. Default is N.
		// Full set name with the format: [set name].[set region].[set group name].
		FullSetName string `yaml:"full_set_name"`
		// Size of the read buffer in bytes. <=0 means read buffer disabled. Default value will be used if not set.
		ReadBufferSize *int `yaml:"read_buffer_size,omitempty"`
	}
	Server struct {
		App      string `yaml:"app"`       // Application name.
		Server   string `yaml:"server"`    // Server name.
		BinPath  string `yaml:"bin_path"`  // Binary file path.
		DataPath string `yaml:"data_path"` // Data file path.
		ConfPath string `yaml:"conf_path"` // Configuration file path.
		Admin    struct {
			IP           string      `yaml:"ip"`            // NIC IP to bind, e.g., 127.0.0.1.
			Nic          string      `yaml:"nic"`           // NIC to bind.
			Port         uint16      `yaml:"port"`          // Port to bind, e.g., 80. Default is 9028.
			ReadTimeout  int         `yaml:"read_timeout"`  // Read timeout in milliseconds for admin HTTP server.
			WriteTimeout int         `yaml:"write_timeout"` // Write timeout in milliseconds for admin HTTP server.
			EnableTLS    bool        `yaml:"enable_tls"`    // Whether to enable TLS.
			RPCZ         *RPCZConfig `yaml:"rpcz"`          // RPCZ configuration.
		}
		Transport    string           `yaml:"transport"`     // Transport type.
		Network      string           `yaml:"network"`       // Network type for all services. Default is tcp.
		Protocol     string           `yaml:"protocol"`      // Protocol type for all services. Default is trpc.
		Filter       []string         `yaml:"filter"`        // Filters for all services.
		StreamFilter []string         `yaml:"stream_filter"` // Stream filters for all services.
		Service      []*ServiceConfig `yaml:"service"`       // Configuration for each individual service.
		// Minimum waiting time in milliseconds when closing the server to wait for deregister finish.
		CloseWaitTime int `yaml:"close_wait_time"`
		// Maximum waiting time in milliseconds when closing the server to wait for requests to finish.
		MaxCloseWaitTime int `yaml:"max_close_wait_time"`
		Timeout          int `yaml:"timeout"` // Timeout in milliseconds.
	}
	Client  ClientConfig  `yaml:"client"`  // Client configuration.
	Plugins plugin.Config `yaml:"plugins"` // Plugins configuration.
}

// RPCZConfig is the config for rpcz.GlobalRPCZ, and is a field of Config.Admin.
type RPCZConfig struct {
	Fraction   float64           `yaml:"fraction"`
	Capacity   uint32            `yaml:"capacity"`
	RecordWhen *RecordWhenConfig `yaml:"record_when"`
}

func (c *RPCZConfig) generate() *rpcz.Config {
	if c.Capacity == 0 {
		const defaultCapacity uint32 = 10000
		c.Capacity = defaultCapacity
	}

	return &rpcz.Config{
		Fraction:     c.Fraction,
		Capacity:     c.Capacity,
		ShouldRecord: c.RecordWhen.shouldRecord(),
	}
}

// RecordWhenConfig stores the RecordWhenConfig field of Config.
type RecordWhenConfig struct {
	ErrorCodes       []int          `yaml:"error_codes,omitempty"`
	MinDuration      *time.Duration `yaml:"min_duration"`
	SamplingFraction float64        `yaml:"sampling_fraction"`
}

func (c *RecordWhenConfig) shouldRecord() rpcz.ShouldRecord {
	if c == nil {
		return rpcz.AlwaysRecord
	}

	return func(s rpcz.Span) bool {
		if c.checkErrorCodes(s) || c.checkoutMinDuration(s) {
			return c.checkoutSamplingFraction()
		}
		return false
	}
}

func (c *RecordWhenConfig) checkErrorCodes(s rpcz.Span) bool {
	err, ok := s.Attribute(rpcz.TRPCAttributeError)
	if !ok {
		return false
	}

	e, ok := err.(error)
	if !ok || e == nil {
		return false
	}

	code := errs.Code(e)
	for _, c := range c.ErrorCodes {
		if c == int(code) {
			return true
		}
	}
	return false
}

func (c *RecordWhenConfig) checkoutMinDuration(s rpcz.Span) bool {
	if c.MinDuration == nil {
		return false
	}
	et := s.EndTime()
	return et.IsZero() || et.Sub(s.StartTime()) > *c.MinDuration
}

func (c *RecordWhenConfig) checkoutSamplingFraction() bool {
	return c.SamplingFraction > rand.Float64()
}

// ServiceConfig is a configuration for a single service. A server process might have multiple services.
type ServiceConfig struct {
	// Disable request timeout inherited from upstream service.
	DisableRequestTimeout bool   `yaml:"disable_request_timeout"`
	IP                    string `yaml:"ip"` // IP address to listen to.
	// Service name in the format: trpc.app.server.service. Used for naming the service.
	Name string `yaml:"name"`
	Nic  string `yaml:"nic"`  // Network Interface Card (NIC) to listen to. No need to configure.
	Port uint16 `yaml:"port"` // Port to listen to.
	// Address to listen to. If set, ipport will be ignored. Otherwise, ipport will be used.
	Address  string `yaml:"address"`
	Network  string `yaml:"network"`  // Network type like tcp/udp.
	Protocol string `yaml:"protocol"` // Protocol type like trpc.
	// Longest time in milliseconds for a handler to handle a request.
	Timeout int `yaml:"timeout"`
	// Maximum idle time in milliseconds for a server connection. Default is 1 minute.
	Idletime          int      `yaml:"idletime"`
	DisableKeepAlives bool     `yaml:"disable_keep_alives"`    // Disables keep-alives.
	Registry          string   `yaml:"registry"`               // Registry to use, e.g., polaris.
	Filter            []string `yaml:"filter"`                 // Filters for the service.
	StreamFilter      []string `yaml:"stream_filter"`          // Stream filters for the service.
	TLSKey            string   `yaml:"tls_key"`                // Server TLS key.
	TLSCert           string   `yaml:"tls_cert"`               // Server TLS certificate.
	CACert            string   `yaml:"ca_cert"`                // CA certificate to validate client certificate.
	ServerAsync       *bool    `yaml:"server_async,omitempty"` // Whether to enable server asynchronous mode.
	// Maximum number of goroutines for server asynchronous mode.
	MaxRoutines int    `yaml:"max_routines"`
	Writev      *bool  `yaml:"writev,omitempty"` // Whether to enable writev.
	Transport   string `yaml:"transport"`        // Transport type.
}

// ClientConfig is the configuration for the client to request backends.
type ClientConfig struct {
	Network        string                  `yaml:"network"`        // Network for all backends. Default is tcp.
	Protocol       string                  `yaml:"protocol"`       // Protocol for all backends. Default is trpc.
	Filter         []string                `yaml:"filter"`         // Filters for all backends.
	StreamFilter   []string                `yaml:"stream_filter"`  // Stream filters for all backends.
	Namespace      string                  `yaml:"namespace"`      // Namespace for all backends.
	Transport      string                  `yaml:"transport"`      // Transport type.
	Timeout        int                     `yaml:"timeout"`        // Timeout in milliseconds.
	Discovery      string                  `yaml:"discovery"`      // Discovery mechanism.
	ServiceRouter  string                  `yaml:"servicerouter"`  // Service router.
	Loadbalance    string                  `yaml:"loadbalance"`    // Load balancing algorithm.
	Circuitbreaker string                  `yaml:"circuitbreaker"` // Circuit breaker configuration.
	Service        []*client.BackendConfig `yaml:"service"`        // Configuration for each individual backend.
}

// trpc server config, set after the framework setup and the yaml config file is parsed.
var globalConfig atomic.Value

func init() {
	globalConfig.Store(defaultConfig())
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.Global.EnableSet = "N"
	cfg.Server.Network = "tcp"
	cfg.Server.Protocol = "trpc"
	cfg.Client.Network = "tcp"
	cfg.Client.Protocol = "trpc"
	return cfg
}

// GlobalConfig returns the global Config.
func GlobalConfig() *Config {
	return globalConfig.Load().(*Config)
}

// SetGlobalConfig set the global Config.
func SetGlobalConfig(cfg *Config) {
	globalConfig.Store(cfg)
}

// LoadGlobalConfig loads a Config from the config file path and sets it as the global Config.
func LoadGlobalConfig(configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	SetGlobalConfig(cfg)
	return nil
}

// LoadConfig loads a Config from the config file path.
func LoadConfig(configPath string) (*Config, error) {
	cfg, err := parseConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}
	if err := RepairConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseConfigFromFile(configPath string) (*Config, error) {
	buf, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	// expand environment variables
	buf = []byte(expandEnv(string(buf)))

	cfg := defaultConfig()
	if err := yaml.Unmarshal(buf, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Setup registers client config and setups plugins according to the Config.
func Setup(cfg *Config) error {
	if _, err := SetupPlugins(cfg.Plugins); err != nil {
		return err
	}
	if err := SetupClients(&cfg.Client); err != nil {
		return err
	}
	return nil
}

// SetupPlugins sets up all plugins and returns a function to close them.
func SetupPlugins(cfg plugin.Config) (func() error, error) {
	if cfg == nil {
		return func() error { return nil }, nil
	}
	return cfg.SetupClosables()
}

// SetupClients sets up all backends and the wildcard.
func SetupClients(cfg *ClientConfig) error {
	for _, backendCfg := range cfg.Service {
		if err := client.RegisterClientConfig(backendCfg.Callee, backendCfg); err != nil {
			return err
		}
	}
	// * represents general config for all backends
	if _, ok := client.DefaultClientConfig()["*"]; !ok { // should not be covered if already registered by some plugins
		if err := client.RegisterClientConfig("*", &client.BackendConfig{
			Network:        cfg.Network,
			Protocol:       cfg.Protocol,
			Namespace:      cfg.Namespace,
			Transport:      cfg.Transport,
			Timeout:        cfg.Timeout,
			Filter:         cfg.Filter,
			StreamFilter:   cfg.StreamFilter,
			Discovery:      cfg.Discovery,
			ServiceRouter:  cfg.ServiceRouter,
			Loadbalance:    cfg.Loadbalance,
			Circuitbreaker: cfg.Circuitbreaker,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RepairConfig repairs the Config by filling in some fields with default values.
func RepairConfig(cfg *Config) error {
	// nic -> ip
	if err := repairServiceIPWithNic(cfg); err != nil {
		return err
	}
	// set default read buffer size
	if cfg.Global.ReadBufferSize == nil {
		readerSize := codec.DefaultReaderSize
		cfg.Global.ReadBufferSize = &readerSize
	}
	codec.SetReaderSize(*cfg.Global.ReadBufferSize)

	// protocol network ip empty
	for _, serviceCfg := range cfg.Server.Service {
		setDefault(&serviceCfg.Protocol, cfg.Server.Protocol)
		setDefault(&serviceCfg.Network, cfg.Server.Network)
		setDefault(&serviceCfg.IP, cfg.Global.LocalIP)
		setDefault(&serviceCfg.Transport, cfg.Server.Transport)
		setDefault(&serviceCfg.Address, net.JoinHostPort(serviceCfg.IP, strconv.Itoa(int(serviceCfg.Port))))

		// server async mode by default
		if serviceCfg.ServerAsync == nil {
			enableServerAsync := true
			serviceCfg.ServerAsync = &enableServerAsync
		}
		// writev disabled by default
		if serviceCfg.Writev == nil {
			enableWritev := false
			serviceCfg.Writev = &enableWritev
		}
		if serviceCfg.Timeout == 0 {
			serviceCfg.Timeout = cfg.Server.Timeout
		}
		if serviceCfg.Idletime == 0 {
			serviceCfg.Idletime = defaultIdleTimeout
			if serviceCfg.Timeout > defaultIdleTimeout {
				serviceCfg.Idletime = serviceCfg.Timeout
			}
		}
	}

	setDefault(&cfg.Client.Namespace, cfg.Global.Namespace)
	for _, backendCfg := range cfg.Client.Service {
		repairClientConfig(backendCfg, &cfg.Client)
	}
	return nil
}

// repairServiceIPWithNic repairs the Config when service ip is empty according to the nic.
func repairServiceIPWithNic(cfg *Config) error {
	for index, item := range cfg.Server.Service {
		if item.IP == "" {
			ip := getIP(item.Nic)
			if ip == "" && item.Nic != "" {
				return fmt.Errorf("can't find service IP by the NIC: %s", item.Nic)
			}
			cfg.Server.Service[index].IP = ip
		}
		setDefault(&cfg.Global.LocalIP, item.IP)
	}

	if cfg.Server.Admin.IP == "" {
		ip := getIP(cfg.Server.Admin.Nic)
		if ip == "" && cfg.Server.Admin.Nic != "" {
			return fmt.Errorf("can't find admin IP by the NIC: %s", cfg.Server.Admin.Nic)
		}
		cfg.Server.Admin.IP = ip
	}
	return nil
}

func repairClientConfig(backendCfg *client.BackendConfig, clientCfg *ClientConfig) {
	// service name in proto file will be used as key for backend config by default
	// generally, service name in proto file is the same as the backend service name.
	// therefore, no need to config backend service name
	setDefault(&backendCfg.Callee, backendCfg.ServiceName)
	setDefault(&backendCfg.ServiceName, backendCfg.Callee)
	setDefault(&backendCfg.Namespace, clientCfg.Namespace)
	setDefault(&backendCfg.Network, clientCfg.Network)
	setDefault(&backendCfg.Protocol, clientCfg.Protocol)
	setDefault(&backendCfg.Transport, clientCfg.Transport)
	if backendCfg.Target == "" {
		setDefault(&backendCfg.Discovery, clientCfg.Discovery)
		setDefault(&backendCfg.ServiceRouter, clientCfg.ServiceRouter)
		setDefault(&backendCfg.Loadbalance, clientCfg.Loadbalance)
		setDefault(&backendCfg.Circuitbreaker, clientCfg.Circuitbreaker)
	}
	if backendCfg.Timeout == 0 {
		backendCfg.Timeout = clientCfg.Timeout
	}
	// Global filter is at front and is deduplicated.
	backendCfg.Filter = deduplicate(clientCfg.Filter, backendCfg.Filter)
	backendCfg.StreamFilter = deduplicate(clientCfg.StreamFilter, backendCfg.StreamFilter)
}

// getMillisecond returns time.Duration by the input value in milliseconds.
func getMillisecond(sec int) time.Duration {
	return time.Millisecond * time.Duration(sec)
}

// setDefault points dst to def if dst is not nil and points to empty string.
func setDefault(dst *string, def string) {
	if dst != nil && *dst == "" {
		*dst = def
	}
}
