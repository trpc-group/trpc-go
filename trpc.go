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

// Package trpc is the Go implementation of tRPC, designed for high performance,
// pluggability, and ease of testing.
package trpc

import (
	"errors"
	"fmt"
	"math"
	"time"

	"go.uber.org/automaxprocs/maxprocs"

	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/reflection"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/transport"
)

// NewServer parses the YAML config file to quickly start a server with multiple services.
// The default config file is ./trpc_go.yaml, which can be overridden by the -conf flag.
// This method should be called only once.
func NewServer(opt ...server.Option) *server.Server {
	// Load and parse the config file.
	cfg, err := LoadConfig(serverConfigPath())
	if err != nil {
		panic("load config fail: " + err.Error())
	}

	// Set the global config for other plugins to access.
	SetGlobalConfig(cfg)

	// Use the config to set global variables.
	SetGlobalVariables(cfg)

	// Set default MaxCloseWaitTime and CloseWaitTime.
	if cfg.Server.CloseWaitTime == 0 {
		cfg.Server.CloseWaitTime = 1000
	}
	if cfg.Server.MaxCloseWaitTime == 0 {
		cfg.Server.MaxCloseWaitTime = 2000
	}

	// Setup plugins.
	closePlugins, err := SetupPlugins(cfg.Plugins)
	if err != nil {
		panic("setup plugin fail: " + err.Error())
	}

	// Setup clients.
	if err := SetupClients(&cfg.Client); err != nil {
		panic("failed to setup client: " + err.Error())
	}

	// Mark plugin setup as complete.
	// (To keep backward compatible with Setup.)
	plugin.SetupFinished()

	// Set default GOMAXPROCS for Docker.
	var interval time.Duration
	if i := cfg.Global.UpdateGOMAXPROCSInterval; i != nil {
		interval = *i
	}

	// Periodically update GOMAXPROCS.
	stop := PeriodicallyUpdateGOMAXPROCS(interval)

	// Initialize the server with the provided configuration.
	s := NewServerWithConfig(cfg, opt...)

	// Register shutdown functions.
	s.RegisterOnShutdown(func() {
		if err := closePlugins(); err != nil {
			log.Errorf("Failed to close plugins, err: %s", err)
		}
	})
	s.RegisterOnShutdown(func() {
		stop()
	})
	return s
}

// NewServerWithConfig initializes a server with a given Config.
// If a YAML config file is not used, custom Config parsing is needed to pass the Config into this function.
// Plugin setup is left to be done if this method is called.
func NewServerWithConfig(cfg *Config, opt ...server.Option) *server.Server {
	// Repair the config.
	if err := RepairConfig(cfg); err != nil {
		panic("repair config fail: " + err.Error())
	}

	// Set the global Config.
	SetGlobalConfig(cfg)

	// Initialize the server with maximum close wait time.
	s := server.NewServer(server.WithDisableGracefulRestart(cfg.Global.DisableGracefulRestart),
		server.WithMaxCloseWaitTime(getMillisecond(cfg.Server.MaxCloseWaitTime)))

	// Setup the admin service.
	setupAdmin(s, cfg)

	// Initialize each service one by one.
	for _, c := range cfg.Server.Service {
		s.AddService(c.Name, newServiceWithConfig(cfg, c, opt...))
	}

	// Register the reflection service if specified.
	if rs := cfg.Server.ReflectionService; rs != "" {
		service := s.Service(rs)
		if service == nil {
			panic("getting nil reflection service by service name: " + rs)
		}
		reflection.Register(service, s)
	}
	return s
}

// SetGlobalVariables sets the global variables using the given config.
func SetGlobalVariables(cfg *Config) {
	// Set the maximum frame size for both the client and server.
	if cfg.Global.MaxFrameSize != nil {
		DefaultMaxFrameSize = *cfg.Global.MaxFrameSize
	}
	// Set the plugin setup timeout.
	if cfg.Global.PluginSetupTimeout != nil {
		plugin.SetupTimeout = *cfg.Global.PluginSetupTimeout
	}
}

// PeriodicallyUpdateGOMAXPROCS regularly updates the runtime.GOMAXPROCS, primarily used in
// container scenarios. This function allows for timely updates of the runtime.GOMAXPROCS value when
// vertical scaling of containers occurs. If the interval is less than or equal to 0, the
// runtime.GOMAXPROCS is set only once.
// It is recommended to enable this by default for users on container platforms to prevent ineffective scaling.
func PeriodicallyUpdateGOMAXPROCS(interval time.Duration) (stop func()) {
	clf := GlobalConfig()

	// Configure the maximum number of CPU processes based on the system's CPU quota.
	configureMaxProcs(clf.Global.RoundUpCPUQuota)

	if interval <= 0 {
		return func() {}
	}

	done := make(chan struct{})

	go func() {
		tick := time.NewTicker(interval)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				configureMaxProcs(clf.Global.RoundUpCPUQuota)
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

// configureMaxProcs configures the maximum number of CPU processes based on the system's CPU quota.
// If enableRoundUp is "true", it rounds up the CPU quota to the nearest whole number.
func configureMaxProcs(enableRoundUp bool) {
	if enableRoundUp {
		_, _ = maxprocs.Set(maxprocs.Logger(log.Debugf),
			maxprocs.RoundQuotaFunc(func(v float64) int { return int(math.Ceil(v)) }))
	} else {
		_, _ = maxprocs.Set(maxprocs.Logger(log.Debugf))
	}
}

// GetAdminService retrieves the admin service from a server.Server instance.
func GetAdminService(s *server.Server) (*admin.Server, error) {
	adminServer, ok := s.Service(admin.ServiceName).(*admin.Server)
	if !ok {
		return nil, errors.New("admin server may not be enabled")
	}
	return adminServer, nil
}

// setupAdmin configures and starts the admin service based on the provided configuration.
func setupAdmin(s *server.Server, cfg *Config) {
	// Configure the admin service, then start it if configured.
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

	s.AddService(admin.ServiceName, admin.NewTrpcAdminServer(opts...))
}

// newServiceWithConfig initializes a new service with the specified configuration and server options.
func newServiceWithConfig(cfg *Config, serviceCfg *ServiceConfig, opt ...server.Option) server.Service {
	// Deduplicate global filters and configure them.
	filterNames := Deduplicate(cfg.Server.Filter, serviceCfg.Filter)
	filters := make([]filter.ServerFilter, 0, len(filterNames))
	for _, name := range filterNames {
		f := filter.GetServer(name)
		if f == nil {
			panic(fmt.Sprintf("filter %s no registered, do not configure", name))
		}
		filters = append(filters, f)
	}

	// Deduplicate and configure stream filters.
	streamFilterName := Deduplicate(cfg.Server.StreamFilter, serviceCfg.StreamFilter)
	streamFilter := make([]server.StreamFilter, 0, len(streamFilterName))
	for _, name := range streamFilterName {
		f := server.GetStreamFilter(name)
		if f == nil {
			panic(fmt.Sprintf("stream filter %s no registered, do not configure", name))
		}
		streamFilter = append(streamFilter, f)
	}

	// Retrieve the registry by service name.
	reg := registry.Get(serviceCfg.Name)
	if serviceCfg.Registry != "" && reg == nil {
		log.Warnf("Service: %s registry not exist.", serviceCfg.Name)
	}

	// Configure server options.
	opts := []server.Option{
		server.WithNamespace(cfg.Global.Namespace),
		server.WithEnvName(cfg.Global.EnvName),
		server.WithContainer(cfg.Global.ContainerName),
		server.WithServiceName(serviceCfg.Name),
		server.WithTransport(transport.GetServerTransport(serviceCfg.Transport)),
		server.WithProtocol(serviceCfg.Protocol),
		server.WithNetwork(serviceCfg.Network),
		server.WithAddress(serviceCfg.Address),
		server.WithStreamFilters(streamFilter...),
		server.WithRegistry(reg),
		server.WithTimeout(getMillisecond(serviceCfg.Timeout)),
		server.WithReadTimeout(getMillisecond(serviceCfg.ReadTimeout)),
		server.WithDisableRequestTimeout(serviceCfg.DisableRequestTimeout),
		server.WithDisableKeepAlives(serviceCfg.DisableKeepAlives),
		server.WithCloseWaitTime(getMillisecond(cfg.Server.CloseWaitTime)),
		server.WithMaxCloseWaitTime(getMillisecond(cfg.Server.MaxCloseWaitTime)),
		server.WithIdleTimeout(getMillisecond(serviceCfg.Idletime)),
		server.WithTLS(serviceCfg.TLSCert, serviceCfg.TLSKey, serviceCfg.CACert),
		server.WithServerAsync(*serviceCfg.ServerAsync),
		server.WithMaxRoutines(serviceCfg.MaxRoutines),
		server.WithWritev(*serviceCfg.Writev),
		server.WithOverloadCtrl(&serviceCfg.OverloadCtrl),
	}

	// Apply serialization and compression types if specified.
	if serviceCfg.CurrentSerializationType != nil {
		opts = append(opts, server.WithCurrentSerializationType(*serviceCfg.CurrentSerializationType))
	}

	if serviceCfg.CurrentCompressType != nil {
		opts = append(opts, server.WithCurrentCompressType(*serviceCfg.CurrentCompressType))
	}

	for i := range filters {
		opts = append(opts, server.WithNamedFilter(filterNames[i], filters[i]))
	}

	// Configure method-specific timeouts.
	for method, mcfg := range serviceCfg.Method {
		if mcfg.Timeout != nil {
			opts = append(opts, server.WithMethodTimeout(method,
				time.Millisecond*time.Duration(*mcfg.Timeout)))
		}
	}

	// Configure the set name if enabled.
	if cfg.Global.EnableSet == "Y" {
		opts = append(opts, server.WithSetName(cfg.Global.FullSetName))
	}

	// Append additional server options.
	opts = append(opts, opt...)

	// Create and return the new server service.
	return server.New(opts...)
}
