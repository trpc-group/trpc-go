// Package trpc is the Go implementation of tRPC, which is designed to be high-performance,
// everything-pluggable and easy for testing.
package trpc

import (
	"errors"
	"fmt"

	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"

	"go.uber.org/automaxprocs/maxprocs"
)

// NewServer parses the yaml config file to quickly start the server with multiple services.
// The config file is ./trpc_go.yaml by default and can be set by the flag -conf.
// This method should be called only once.
func NewServer(opt ...server.Option) *server.Server {
	// load and parse config file
	cfg, err := LoadConfig(serverConfigPath())
	if err != nil {
		panic("load config fail: " + err.Error())
	}

	// set to global config for other plugins' accessing to the config
	SetGlobalConfig(cfg)

	closePlugins, err := SetupPlugins(cfg.Plugins)
	if err != nil {
		panic("setup plugin fail: " + err.Error())
	}
	if err := SetupClients(&cfg.Client); err != nil {
		panic("failed to setup client: " + err.Error())
	}

	// set default GOMAXPROCS for docker
	maxprocs.Set(maxprocs.Logger(log.Debugf))
	s := NewServerWithConfig(cfg, opt...)
	s.RegisterOnShutdown(func() {
		if err := closePlugins(); err != nil {
			log.Errorf("failed to close plugins, err: %s", err)
		}
	})
	return s
}

// NewServerWithConfig initializes a server with a Config.
// If yaml config file not used, custom Config parsing is needed to pass the Config into this function.
// Plugins' setup is left to do if this method is called.
func NewServerWithConfig(cfg *Config, opt ...server.Option) *server.Server {
	// repair config
	if err := RepairConfig(cfg); err != nil {
		panic("repair config fail: " + err.Error())
	}

	// set to global Config
	SetGlobalConfig(cfg)

	s := &server.Server{
		MaxCloseWaitTime: getMillisecond(cfg.Server.MaxCloseWaitTime),
	}

	// setup admin service
	setupAdmin(s, cfg)

	// init service one by one
	for _, c := range cfg.Server.Service {
		s.AddService(c.Name, newServiceWithConfig(cfg, c, opt...))
	}
	return s
}

// GetAdminService gets admin service from server.Server.
func GetAdminService(s *server.Server) (*admin.Server, error) {
	adminServer, ok := s.Service(admin.ServiceName).(*admin.Server)
	if !ok {
		return nil, errors.New("admin server may not be enabled")
	}
	return adminServer, nil
}

func setupAdmin(s *server.Server, cfg *Config) {
	// admin configured, then admin service will be started
	opts := []admin.Option{
		admin.WithSkipServe(cfg.Server.Admin.Port == 0),
		admin.WithVersion(Version()),
		admin.WithTLS(cfg.Server.Admin.EnableTLS),
		admin.WithConfigPath(ServerConfigPath),
		admin.WithReadTimeout(getMillisecond(cfg.Server.Admin.ReadTimeout)),
		admin.WithWriteTimeout(getMillisecond(cfg.Server.Admin.WriteTimeout)),
	}
	if cfg.Server.Admin.Port > 0 {
		opts = append(opts, admin.WithAddr(fmt.Sprintf("%s:%d", cfg.Server.Admin.IP, cfg.Server.Admin.Port)))
	}
	if cfg.Server.Admin.RPCZ != nil {
		rpcz.GlobalRPCZ = rpcz.NewRPCZ(cfg.Server.Admin.RPCZ.generate())
	}
	s.AddService(admin.ServiceName, admin.NewServer(opts...))
}

func newServiceWithConfig(cfg *Config, serviceCfg *ServiceConfig, opt ...server.Option) server.Service {
	var (
		filters     filter.ServerChain
		filterNames []string
	)
	// Global filter is at front and is deduplicated.
	for _, name := range deduplicate(cfg.Server.Filter, serviceCfg.Filter) {
		f := filter.GetServer(name)
		if f == nil {
			panic(fmt.Sprintf("filter %s no registered, do not configure", name))
		}
		filters = append(filters, f)
		filterNames = append(filterNames, name)
	}
	filterNames = append(filterNames, "fixTimeout")

	var streamFilter []server.StreamFilter
	for _, name := range deduplicate(cfg.Server.StreamFilter, serviceCfg.StreamFilter) {
		f := server.GetStreamFilter(name)
		if f == nil {
			panic(fmt.Sprintf("stream filter %s no registered, do not configure", name))
		}
		streamFilter = append(streamFilter, f)
	}

	// get registry by service
	reg := registry.Get(serviceCfg.Name)
	if serviceCfg.Registry != "" && reg == nil {
		log.Warnf("service:%s registry not exist", serviceCfg.Name)
	}

	opts := []server.Option{
		server.WithNamespace(cfg.Global.Namespace),
		server.WithEnvName(cfg.Global.EnvName),
		server.WithContainer(cfg.Global.ContainerName),
		server.WithServiceName(serviceCfg.Name),
		server.WithProtocol(serviceCfg.Protocol),
		server.WithTransport(transport.GetServerTransport(serviceCfg.Transport)),
		server.WithNetwork(serviceCfg.Network),
		server.WithAddress(serviceCfg.Address),
		server.WithStreamFilters(streamFilter...),
		server.WithRegistry(reg),
		server.WithTimeout(getMillisecond(serviceCfg.Timeout)),
		server.WithDisableRequestTimeout(serviceCfg.DisableRequestTimeout),
		server.WithDisableKeepAlives(serviceCfg.DisableKeepAlives),
		server.WithCloseWaitTime(getMillisecond(cfg.Server.CloseWaitTime)),
		server.WithMaxCloseWaitTime(getMillisecond(cfg.Server.MaxCloseWaitTime)),
		server.WithIdleTimeout(getMillisecond(serviceCfg.Idletime)),
		server.WithTLS(serviceCfg.TLSCert, serviceCfg.TLSKey, serviceCfg.CACert),
		server.WithServerAsync(*serviceCfg.ServerAsync),
		server.WithMaxRoutines(serviceCfg.MaxRoutines),
		server.WithWritev(*serviceCfg.Writev),
	}
	for i := range filters {
		opts = append(opts, server.WithNamedFilter(filterNames[i], filters[i]))
	}

	if cfg.Global.EnableSet == "Y" {
		opts = append(opts, server.WithSetName(cfg.Global.FullSetName))
	}
	opts = append(opts, opt...)
	return server.New(opts...)
}
