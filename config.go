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

package trpc

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/expandenv"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/random"
	"trpc.group/trpc-go/trpc-go/internal/scope"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
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
	Global  GlobalCfg     `yaml:"global,omitempty"`  // Global configuration.
	Server  ServerConfig  `yaml:"server,omitempty"`  // Server configuration.
	Client  ClientConfig  `yaml:"client,omitempty"`  // Client configuration.
	Plugins plugin.Config `yaml:"plugins,omitempty"` // Plugins configuration.
}

// GlobalCfg is the global configuration.
type GlobalCfg struct {
	Namespace     string `yaml:"namespace,omitempty"`      // Namespace for the configuration.
	EnvName       string `yaml:"env_name,omitempty"`       // Environment name.
	ContainerName string `yaml:"container_name,omitempty"` // Container name.
	LocalIP       string `yaml:"local_ip,omitempty"`       // Local IP address.
	EnableSet     string `yaml:"enable_set,omitempty"`     // Y/N. Whether to enable Set. Default is N.
	// Full set name with the format: [set name].[set region].[set group name].
	FullSetName string `yaml:"full_set_name,omitempty"`
	// Size of the read buffer in bytes. <=0 means read buffer disabled. Default value will be used if not set.
	ReadBufferSize *int `yaml:"read_buffer_size,omitempty"`
	// MaxFrameSize is the maximum frame size (in bytes) set for both the client and server.
	// The default value is 10485760 (10MB).
	MaxFrameSize *int `yaml:"max_frame_size,omitempty"`
	// PluginSetupTimeout is the setup timeout for each plugin, default 3 seconds.
	PluginSetupTimeout *time.Duration `yaml:"plugin_setup_timeout,omitempty"`
	// UpdateDataGOMAXPROCSInterval periodically update GOMAXPROCS.
	// For more details, see https://git.woa.com/trpc-go/trpc-go/issues/891.
	UpdateGOMAXPROCSInterval *time.Duration `yaml:"update_gomaxprocs_interval,omitempty"`
	// RoundUpCPUQuota provides the option to enable rounding up the CPU quota. Default is false.
	// 'go.uber.org/automaxprocs/maxprocs' library introduces round up option
	// to improve CPU utilization on non-integer number of cores.
	// For more details, see https://github.com/uber-go/automaxprocs/issues/78.
	RoundUpCPUQuota bool `yaml:"round_up_cpu_quota,omitempty"`
	// DisableGracefulRestart determines whether to disable graceful restart.
	// For more details, see https://git.woa.com/trpc-go/trpc-go/issues/1015.
	DisableGracefulRestart bool `yaml:"disable_graceful_restart,omitempty"`
}

// ServerConfig is the configuration for trpc server.
type ServerConfig struct {
	App       string      `yaml:"app,omitempty"`       // Application name.
	Server    string      `yaml:"server,omitempty"`    // Server name.
	BinPath   string      `yaml:"bin_path,omitempty"`  // Binary file path.
	DataPath  string      `yaml:"data_path,omitempty"` // Data file path.
	ConfPath  string      `yaml:"conf_path,omitempty"` // Configuration file path.
	Admin     AdminConfig `yaml:"admin,omitempty"`     // Admin configuration.
	Transport string      `yaml:"transport,omitempty"` // Transport type.
	Network   string      `yaml:"network,omitempty"`   // Network type for all services. Default is tcp.
	Protocol  string      `yaml:"protocol,omitempty"`  // Protocol type for all services. Default is trpc.

	// CurrentSerializationType specifies the current serialization type.
	// It's often used for transparent proxy without serialization.
	// If current serialization type is not set, serialization type will be determined by
	// serialization field of request protocol.
	CurrentSerializationType *int `yaml:"current_serialization_type,omitempty"`
	// CurrentCompressType specifies the current compress type.
	CurrentCompressType *int `yaml:"current_compress_type,omitempty"`

	Filter            []string         `yaml:"filter,omitempty"`             // Filters for all services.
	StreamFilter      []string         `yaml:"stream_filter,omitempty"`      // Stream filters for all services.
	Service           []*ServiceConfig `yaml:"service,omitempty"`            // Configuration for each individual service.
	ReflectionService string           `yaml:"reflection_service,omitempty"` // Specify a Service as a reflection service.
	// Minimum waiting time in milliseconds when closing the server to wait for deregister finish.
	CloseWaitTime int `yaml:"close_wait_time,omitempty"`
	// Maximum waiting time in milliseconds when closing the server to wait for requests to finish.
	MaxCloseWaitTime int `yaml:"max_close_wait_time,omitempty"`
	Timeout          int `yaml:"timeout,omitempty"` // Timeout in milliseconds.

	// Overload control is the server global configuration for trpc-overload-control.
	OverloadCtrl overloadctrl.Impl `yaml:"overload_ctrl,omitempty"`
}

// UnmarshalYAML implements yaml.Unmarshaler.
// It mainly deals with overload control configuration.
func (cfg *ServerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type tmp ServerConfig
	if err := unmarshal((*tmp)(cfg)); err != nil {
		return err
	}
	return cfg.OverloadCtrl.Build(overloadctrl.GetServer, &overloadctrl.ServiceMethodInfo{
		MethodName: overloadctrl.AnyMethod,
	})
}

// AdminConfig is the configuration for admin.
type AdminConfig struct {
	IP           string      `yaml:"ip,omitempty"`            // NIC IP to bind, e.g., 127.0.0.1.
	Nic          string      `yaml:"nic,omitempty"`           // NIC to bind.
	Port         uint16      `yaml:"port,omitempty"`          // Port to bind, e.g., 80. Default is 9028.
	ReadTimeout  int         `yaml:"read_timeout,omitempty"`  // Read timeout in milliseconds for admin HTTP server.
	WriteTimeout int         `yaml:"write_timeout,omitempty"` // Write timeout in milliseconds for admin HTTP server.
	EnableTLS    bool        `yaml:"enable_tls,omitempty"`    // Whether to enable TLS.
	RPCZ         *RPCZConfig `yaml:"rpcz,omitempty"`          // RPCZ configuration.
}

// RPCZConfig is the config for rpcz.GlobalRPCZ, and is a field of Config.Admin.
type RPCZConfig struct {
	Fraction   float64           `yaml:"fraction,omitempty"`
	Capacity   uint32            `yaml:"capacity,omitempty"`
	RecordWhen *RecordWhenConfig `yaml:"record_when,omitempty"`
}

func (c *RPCZConfig) generate() *rpcz.Config {
	if c.Capacity == 0 {
		const defaultCapacity uint32 = 10000
		c.Capacity = defaultCapacity
	}

	config := &rpcz.Config{
		Fraction: c.Fraction,
		Capacity: c.Capacity,
	}
	if c.RecordWhen != nil {
		config.ShouldRecord = c.RecordWhen.shouldRecord()
	}
	return config
}

type node interface {
	yaml.Unmarshaler
	shouldRecorder
}

type nodeKind string

const (
	kindAND              nodeKind = "AND"
	kindOR               nodeKind = "OR"
	kindNOT              nodeKind = "NOT"
	kindMinDuration      nodeKind = "__min_duration"
	kindMinRequestSize   nodeKind = "__min_request_size"
	kindMinResponseSize  nodeKind = "__min_response_size"
	kindRPCName          nodeKind = "__rpc_name"
	kindErrorCodes       nodeKind = "__error_code"
	kindErrorMessages    nodeKind = "__error_message"
	kindSamplingFraction nodeKind = "__sampling_fraction"
	kindHasAttributes    nodeKind = "__has_attribute"
)

var kindToNode = map[nodeKind]func() node{
	kindAND:              func() node { return &andNode{} },
	kindOR:               func() node { return &orNode{} },
	kindNOT:              func() node { return &notNode{} },
	kindMinDuration:      func() node { return &minMinDurationNode{} },
	kindMinRequestSize:   func() node { return &minRequestSizeNode{} },
	kindMinResponseSize:  func() node { return &minResponseSizeNode{} },
	kindRPCName:          func() node { return &rpcNameNode{} },
	kindErrorCodes:       func() node { return &errorCodeNode{} },
	kindErrorMessages:    func() node { return &errorMessageNode{} },
	kindSamplingFraction: func() node { return &samplingFractionNode{} },
	kindHasAttributes:    func() node { return &hasAttributeNode{} },
}

func formatKindToNode(kindToNode map[nodeKind]func() node) string {
	ks := make([]string, 0, len(kindToNode))
	for k := range kindToNode {
		ks = append(ks, string(k))
	}
	return "[\"" + strings.Join(ks, "\", \"") + "\"]"
}

func generate(k nodeKind) (node, error) {
	if fn, ok := kindToNode[k]; ok {
		return fn(), nil
	}
	return nil, fmt.Errorf("unknown node: %s, valid node must be one of %v", k, formatKindToNode(kindToNode))
}

type shouldRecorder interface {
	shouldRecord() rpcz.ShouldRecord
}

type recorder struct {
	rpcz.ShouldRecord
}

func (n *recorder) shouldRecord() rpcz.ShouldRecord {
	return n.ShouldRecord
}

// RecordWhenConfig stores the RecordWhenConfig field of Config.
type RecordWhenConfig struct {
	andNode
}

// UnmarshalYAML customizes RecordWhenConfig's behavior when being unmarshalled from a YAML document.
func (c *RecordWhenConfig) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&c.andNode); err != nil {
		return fmt.Errorf("decoding RecordWhenConfig's andNode: %w", err)
	}
	return nil
}

type nodeList struct {
	shouldRecords []rpcz.ShouldRecord
}

func (nl *nodeList) UnmarshalYAML(node *yaml.Node) error {
	var nodes []map[nodeKind]yaml.Node
	if err := node.Decode(&nodes); err != nil {
		return fmt.Errorf("decoding []map[nodeKind]yaml.Node: %w", err)
	}
	nl.shouldRecords = make([]rpcz.ShouldRecord, 0, len(nodes))
	for _, n := range nodes {
		if size := len(n); size != 1 {
			return fmt.Errorf("%v node has %d element currently, "+
				"but the valid number of elements can only be 1", n, size)
		}
		for nodeKind, value := range n {
			if valueEmpty(value) {
				return fmt.Errorf("decoding %s node: value is empty", nodeKind)
			}
			node, err := generate(nodeKind)
			if err != nil {
				return fmt.Errorf("generating %s node: %w", nodeKind, err)
			}
			if err := value.Decode(node); err != nil {
				return fmt.Errorf("decoding %s node: %w", nodeKind, err)
			}
			nl.shouldRecords = append(nl.shouldRecords, node.shouldRecord())
		}
	}
	return nil
}

func valueEmpty(node yaml.Node) bool {
	return len(node.Content) == 0 && len(node.Value) == 0
}

type andNode struct {
	recorder
}

func (n *andNode) UnmarshalYAML(node *yaml.Node) error {
	nl := &nodeList{}
	if err := node.Decode(nl); err != nil {
		return fmt.Errorf("decoding andNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		if len(nl.shouldRecords) == 0 {
			return false
		}
		for _, r := range nl.shouldRecords {
			if !r(s) {
				return false
			}
		}
		return true
	}
	return nil
}

type orNode struct {
	recorder
}

func (n *orNode) UnmarshalYAML(node *yaml.Node) error {
	nl := &nodeList{}
	if err := node.Decode(nl); err != nil {
		return fmt.Errorf("decoding orNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		for _, r := range nl.shouldRecords {
			if r(s) {
				return true
			}
		}
		return false
	}
	return nil
}

type notNode struct {
	recorder
}

func (n *notNode) UnmarshalYAML(node *yaml.Node) error {
	var not map[nodeKind]yaml.Node
	if err := node.Decode(&not); err != nil {
		return fmt.Errorf("decoding notNode: %w", err)
	}
	const numInvalidChildren = 1
	if n := len(not); n != numInvalidChildren {
		return fmt.Errorf("NOT node has %d child node currently, "+
			"but the valid number of child node can only be %d", n, numInvalidChildren)
	}
	for nodeKind, value := range not {
		node, err := generate(nodeKind)
		if err != nil {
			return fmt.Errorf("generating %s node: %w", nodeKind, err)
		}
		if err := value.Decode(node); err != nil {
			return fmt.Errorf("decoding %s node: %w", nodeKind, err)
		}
		n.ShouldRecord = func(s rpcz.Span) bool {
			return !node.shouldRecord()(s)
		}
	}
	return nil
}

type hasAttributeNode struct {
	recorder
}

func (n *hasAttributeNode) UnmarshalYAML(node *yaml.Node) error {
	var attribute string
	if err := node.Decode(&attribute); err != nil {
		return fmt.Errorf("decoding hasAttributeNode: %w", err)
	}

	key, value, err := parse(attribute)
	if err != nil {
		return fmt.Errorf("parsing attribute %s : %w", attribute, err)
	}

	n.ShouldRecord = func(s rpcz.Span) bool {
		v, ok := s.Attribute(key)
		return ok && strings.Contains(fmt.Sprintf("%s", v), value)
	}
	return nil
}

var errInvalidAttribute = errors.New("invalid attribute form [ valid attribute form: (key, value), " +
	"only one space character after comma character, and key can't contain comma(',') character ]")

func parse(attribute string) (key string, value string, err error) {
	if len(attribute) == 0 || attribute[0] != '(' {
		return "", "", errInvalidAttribute
	}
	attribute = attribute[1:]

	if n := len(attribute); n == 0 || attribute[n-1] != ')' {
		return "", "", errInvalidAttribute
	}
	attribute = attribute[:len(attribute)-1]

	const delimiter = ", "
	i := strings.Index(attribute, delimiter)
	if i == -1 {
		return "", "", errInvalidAttribute
	}
	return attribute[:i], attribute[i+len(delimiter):], nil
}

type minRequestSizeNode struct {
	recorder
}

func (n *minRequestSizeNode) UnmarshalYAML(node *yaml.Node) error {
	var minRequestSize int
	if err := node.Decode(&minRequestSize); err != nil {
		return fmt.Errorf("decoding minRequestSizeNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		size, ok := s.Attribute(rpcz.TRPCAttributeRequestSize)
		if !ok {
			return false
		}
		if size, ok := size.(int); !ok || size < minRequestSize {
			return false
		}
		return true
	}
	return nil
}

type minResponseSizeNode struct {
	recorder
}

func (n *minResponseSizeNode) UnmarshalYAML(node *yaml.Node) error {
	var minResponseSize int
	if err := node.Decode(&minResponseSize); err != nil {
		return fmt.Errorf("decoding minResponseSizeNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		responseSize, ok := s.Attribute(rpcz.TRPCAttributeResponseSize)
		if !ok {
			return false
		}
		if size, ok := responseSize.(int); !ok || size < minResponseSize {
			return false
		}
		return true
	}
	return nil
}

type minMinDurationNode struct {
	recorder
}

func (n *minMinDurationNode) UnmarshalYAML(node *yaml.Node) error {
	var dur time.Duration
	if err := node.Decode(&dur); err != nil {
		return fmt.Errorf("decoding minMinDurationNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		if dur == 0 {
			return true
		}
		et := s.EndTime()
		return et.IsZero() || et.Sub(s.StartTime()) >= dur
	}
	return nil
}

type rpcNameNode struct {
	recorder
}

func (n *rpcNameNode) UnmarshalYAML(node *yaml.Node) error {
	var rpcName string
	if err := node.Decode(&rpcName); err != nil {
		return fmt.Errorf("decoding rpcNameNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		name, ok := s.Attribute(rpcz.TRPCAttributeRPCName)
		if !ok {
			return false
		}
		if name, ok := name.(string); !ok || !strings.Contains(name, rpcName) {
			return false
		}
		return true
	}
	return nil
}

type samplingFractionNode struct {
	recorder
}

var safeRand = random.New()

func (n *samplingFractionNode) UnmarshalYAML(node *yaml.Node) error {
	var f float64
	if err := node.Decode(&f); err != nil {
		return fmt.Errorf("decoding samplingFractionNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		return f > safeRand.Float64()
	}
	return nil
}

type errorCodeNode struct {
	recorder
}

func (n *errorCodeNode) UnmarshalYAML(node *yaml.Node) error {
	var code int
	if err := node.Decode(&code); err != nil {
		return fmt.Errorf("decoding errorCodeNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		err, ok := extractError(s)
		if !ok {
			return false
		}
		c := errs.Code(err)
		return c == code
	}
	return nil
}

type errorMessageNode struct {
	recorder
}

func (n *errorMessageNode) UnmarshalYAML(node *yaml.Node) error {
	var message string
	if err := node.Decode(&message); err != nil {
		return fmt.Errorf("decoding errorMessageNode: %w", err)
	}
	n.ShouldRecord = func(s rpcz.Span) bool {
		err, ok := extractError(s)
		if !ok {
			return false
		}
		return strings.Contains(message, errs.Msg(err))
	}
	return nil
}

func extractError(span rpcz.Span) (error, bool) {
	err, ok := span.Attribute(rpcz.TRPCAttributeError)
	if !ok {
		return nil, false
	}

	e, ok := err.(error)
	return e, ok
}

// ServiceConfig is a configuration for a single service. A server process might have multiple services.
type ServiceConfig struct {
	// Disable request timeout inherited from upstream service.
	DisableRequestTimeout bool   `yaml:"disable_request_timeout,omitempty"`
	IP                    string `yaml:"ip,omitempty"` // IP address to listen to.
	// Service name in the format: trpc.app.server.service. Used for naming the service.
	Name string `yaml:"name,omitempty"`
	Nic  string `yaml:"nic,omitempty"`  // Network Interface Card (NIC) to listen to. No need to configure.
	Port uint16 `yaml:"port,omitempty"` // Port to listen to.
	// Address to listen to. If set, ipport will be ignored. Otherwise, ipport will be used.
	Address  string `yaml:"address,omitempty"`
	Network  string `yaml:"network,omitempty"`  // Network type like tcp/udp.
	Protocol string `yaml:"protocol,omitempty"` // Protocol type like trpc.

	// CurrentSerializationType specifies the current serialization type.
	// It's often used for transparent proxy without serialization.
	// If current serialization type is not set, serialization type will be determined by
	// serialization field of request protocol.
	CurrentSerializationType *int `yaml:"current_serialization_type,omitempty"`
	// CurrentCompressType specifies the current compress type.
	CurrentCompressType *int `yaml:"current_compress_type,omitempty"`

	// Longest time in milliseconds for a handler to handle a request.
	Timeout int `yaml:"timeout,omitempty"`

	// ReadTimeout specifies the maximum duration in milliseconds for reading a request
	// from a client connection in this service.
	//
	// If not set, the read timeout will default to the same value as the idle timeout.
	//
	// It is important to distinguish between "timeout" and "read_timeout":
	//  - timeout: the maximum duration allowed for a handler to process a request.
	//  - read_timeout: the maximum duration allowed for reading a request from a client connection.
	//
	// As for the difference between "read_timeout" and "idletime":
	// Under the current implementation, if read_timeout is reached but idletime is not,
	// the server will attempt to read requests from the connection again. This means the reading process
	// is interrupted by the read timeout at regular intervals, and the connection is only closed if the
	// idle timeout is reached.
	//
	// By default, read_timeout is set to the idletime's default value, which is 60 seconds.
	// This extended duration can cause the graceful restart process to seem sluggish.
	// However, setting read_timeout to a smaller value might lead to the server closing the client
	// connection prematurely, potentially resulting in the client receiving errors
	// such as error code 141.
	ReadTimeout int `yaml:"read_timeout,omitempty"`

	Method map[string]*ServiceMethodConfig `yaml:"method,omitempty"`

	// Maximum idle time in milliseconds for a server connection. Default is 60000 (1 minute).
	Idletime          int      `yaml:"idletime,omitempty"`
	DisableKeepAlives bool     `yaml:"disable_keep_alives,omitempty"` // Disables keep-alives.
	Registry          string   `yaml:"registry,omitempty"`            // Registry to use, e.g., polaris.
	Filter            []string `yaml:"filter,omitempty"`              // Filters for the service.
	StreamFilter      []string `yaml:"stream_filter,omitempty"`       // Stream filters for the service.
	TLSKey            string   `yaml:"tls_key,omitempty"`             // Server TLS key.
	TLSCert           string   `yaml:"tls_cert,omitempty"`            // Server TLS certificate.
	CACert            string   `yaml:"ca_cert,omitempty"`             // CA certificate to validate client certificate.
	ServerAsync       *bool    `yaml:"server_async,omitempty"`        // Whether to enable server asynchronous mode.
	// MaxRoutines is the maximum number of goroutines for server asynchronous mode.
	// Requests exceeding MaxRoutines will be queued. Prolonged overages may lead to OOM!
	// MaxRoutines is not the solution to alleviate server overloading.
	MaxRoutines int    `yaml:"max_routines,omitempty"`
	Writev      *bool  `yaml:"writev,omitempty"`    // Whether to enable writev.
	Transport   string `yaml:"transport,omitempty"` // Transport type.

	OverloadCtrl overloadctrl.Impl `yaml:"overload_ctrl,omitempty"` // Overload control.
	// For compatibility with the old version. Only the first element will be used if not empty.
	OverloadCtrls []string `yaml:"overload_ctrls,omitempty"`
}

// ServiceMethodConfig is the configuration for method.
type ServiceMethodConfig struct {
	Timeout *int `yaml:"timeout,omitempty"` // ms
}

// ClientConfig is the configuration for the client to request backends.
type ClientConfig struct {
	Network         string                  `yaml:"network,omitempty"`          // Network for all backends. Default is tcp.
	Protocol        string                  `yaml:"protocol,omitempty"`         // Protocol for all backends. Default is trpc.
	Filter          []string                `yaml:"filter,omitempty"`           // Filters for all backends.
	StreamFilter    []string                `yaml:"stream_filter,omitempty"`    // Stream filters for all backends.
	Namespace       string                  `yaml:"namespace,omitempty"`        // Callee Namespace for all backends.
	CallerNamespace string                  `yaml:"caller_namespace,omitempty"` // Caller Namespace of current service.
	Transport       string                  `yaml:"transport,omitempty"`        // Transport type.
	Timeout         int                     `yaml:"timeout,omitempty"`          // Timeout in milliseconds.
	Discovery       string                  `yaml:"discovery,omitempty"`        // Discovery mechanism.
	ServiceRouter   string                  `yaml:"servicerouter,omitempty"`    // Service router.
	Loadbalance     string                  `yaml:"loadbalance,omitempty"`      // Load balancing algorithm.
	Circuitbreaker  string                  `yaml:"circuitbreaker,omitempty"`   // Circuit breaker configuration.
	Scope           scope.Scope             `yaml:"scope,omitempty"`            // Scope is the current scope of the global client.
	Service         []*client.BackendConfig `yaml:"service,omitempty"`          // Configuration for each individual backend.
}

// trpc server config, set after the framework setup and the yaml config file is parsed.
var globalConfig atomic.Value

func init() {
	globalConfig.Store(defaultConfig())
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.Global.EnableSet = "N"
	cfg.Server.Network = protocol.TCP
	cfg.Server.Protocol = protocol.TRPC
	cfg.Client.Network = protocol.TCP
	cfg.Client.Protocol = protocol.TRPC
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

	cfg := defaultConfig()
	if err := yaml.Unmarshal(expandenv.ExpandEnv(buf), cfg); err != nil {
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
	// notify that plugins' setup is done
	// since client config is not yet registered when plugins are set up, it is better not to use client within plugin
	// setup. Options are preferred than client config if client must be used. plugin.WaitForDone should be called
	// async to use client after plugins' setup done, by the plugin which needs to use client to send requests and
	// relies on client config during its setup.
	plugin.SetupFinished()
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
			Scope:          cfg.Scope,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RepairConfig repairs the Config by filling in some fields with default values.
func RepairConfig(cfg *Config) error {
	// set default read buffer size
	if cfg.Global.ReadBufferSize == nil {
		readerSize := codec.DefaultReaderSize
		cfg.Global.ReadBufferSize = &readerSize
	}
	codec.SetReaderSize(*cfg.Global.ReadBufferSize)

	// nic -> ip
	if err := repairServiceIPWithNic(cfg); err != nil {
		return err
	}

	// Set empty ip to "0.0.0.0" to prevent malformed key matching
	// for passed listeners during hot restart.
	const defaultIP = "0.0.0.0"
	setDefault(&cfg.Global.LocalIP, defaultIP)
	setDefault(&cfg.Server.Admin.IP, cfg.Global.LocalIP)
	// protocol network ip empty
	for _, serviceCfg := range cfg.Server.Service {
		setDefault(&serviceCfg.Protocol, cfg.Server.Protocol)
		setDefault(&serviceCfg.Network, cfg.Server.Network)
		setDefault(&serviceCfg.IP, cfg.Global.LocalIP)
		setDefault(&serviceCfg.Transport, cfg.Server.Transport)
		setDefault(&serviceCfg.Address, net.JoinHostPort(serviceCfg.IP, strconv.Itoa(int(serviceCfg.Port))))

		if cfg.Server.CurrentSerializationType != nil && serviceCfg.CurrentSerializationType == nil {
			serviceCfg.CurrentSerializationType = cfg.Server.CurrentSerializationType
		}
		if cfg.Server.CurrentCompressType != nil && serviceCfg.CurrentCompressType == nil {
			serviceCfg.CurrentCompressType = cfg.Server.CurrentCompressType
		}

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
		// If service overload control is not provided, use the server overload control by default.
		if serviceCfg.OverloadCtrl.Builder == "" {
			serviceCfg.OverloadCtrl = cfg.Server.OverloadCtrl
		}
		if serviceCfg.Idletime == 0 {
			serviceCfg.Idletime = defaultIdleTimeout
			if serviceCfg.Timeout > defaultIdleTimeout {
				serviceCfg.Idletime = serviceCfg.Timeout
			}
		}
		if serviceCfg.ReadTimeout == 0 {
			serviceCfg.ReadTimeout = serviceCfg.Idletime
		}
	}

	// If client callee namespace is not provided, use the global namespace by default.
	setDefault(&cfg.Client.Namespace, cfg.Global.Namespace)
	// If client caller namespace is not provided, use the global namespace by default, too.
	setDefault(&cfg.Client.CallerNamespace, cfg.Global.Namespace)
	for _, backendCfg := range cfg.Client.Service {
		repairClientConfig(backendCfg, &cfg.Client, cfg.Global.LocalIP)
	}
	return nil
}

// repairServiceIPWithNic repairs the Config when service ip is empty according to the nic.
func repairServiceIPWithNic(cfg *Config) error {
	for index, item := range cfg.Server.Service {
		if item.IP == "" {
			ip := GetIP(item.Nic)
			if ip == "" && item.Nic != "" {
				return fmt.Errorf("can't find service IP by the NIC: %s", item.Nic)
			}
			cfg.Server.Service[index].IP = ip
		}
		setDefault(&cfg.Global.LocalIP, item.IP)
	}

	if cfg.Server.Admin.IP == "" {
		ip := GetIP(cfg.Server.Admin.Nic)
		if ip == "" && cfg.Server.Admin.Nic != "" {
			return fmt.Errorf("can't find admin IP by the NIC: %s", cfg.Server.Admin.Nic)
		}
		cfg.Server.Admin.IP = ip
	}
	return nil
}

func repairClientConfig(backendCfg *client.BackendConfig, clientCfg *ClientConfig, localIP string) {
	// service name in proto file will be used as key for backend config by default
	// generally, service name in proto file is the same as the backend service name.
	// therefore, no need to config backend service name
	setDefault(&backendCfg.Callee, backendCfg.ServiceName)
	setDefault(&backendCfg.ServiceName, backendCfg.Callee)
	setDefault(&backendCfg.Namespace, clientCfg.Namespace)
	setDefault(&backendCfg.CallerNamespace, clientCfg.CallerNamespace)
	setDefault(&backendCfg.Network, clientCfg.Network)
	setDefault(&backendCfg.Protocol, clientCfg.Protocol)
	setDefault(&backendCfg.Transport, clientCfg.Transport)
	setDefault(&backendCfg.Scope, clientCfg.Scope)
	setDefault(&backendCfg.LocalIP, localIP)
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
	backendCfg.Filter = Deduplicate(clientCfg.Filter, backendCfg.Filter)
	backendCfg.StreamFilter = Deduplicate(clientCfg.StreamFilter, backendCfg.StreamFilter)
}

// getMillisecond returns time.Duration by the input value in milliseconds.
func getMillisecond(ms int) time.Duration {
	return time.Millisecond * time.Duration(ms)
}

// setDefault points dst to def if dst is not nil and points to empty string.
func setDefault(dst *string, def string) {
	if dst != nil && *dst == "" {
		*dst = def
	}
}

// UnmarshalYAML implements yaml.Unmarshaler.
// Used for compatibility of the field overload_ctrls.
func (cfg *ServiceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type tmp ServiceConfig
	if err := unmarshal((*tmp)(cfg)); err != nil {
		return err
	}

	// ensure compatibility
	if len(cfg.OverloadCtrls) > 1 {
		return errors.New("multiple overload controllers are not supported any more")
	}
	if len(cfg.OverloadCtrls) == 1 && cfg.OverloadCtrl.Builder != "" {
		return errors.New("both overload_ctrl and overload_ctrls are set")
	}
	if len(cfg.OverloadCtrls) == 1 {
		cfg.OverloadCtrl.Builder = cfg.OverloadCtrls[0]
	}

	return cfg.OverloadCtrl.Build(overloadctrl.GetServer, &overloadctrl.ServiceMethodInfo{
		ServiceName: cfg.Name,
		MethodName:  overloadctrl.AnyMethod,
	})
}
