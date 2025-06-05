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
package main

// //
// //
// // Tencent is pleased to support the open source community by making tRPC available.
// //
// // Copyright (C) 2023 THL A29 Limited, a Tencent company.
// // All rights reserved.
// //
// // If you have downloaded a copy of the tRPC source code from Tencent,
// // please note that tRPC source code is licensed under the  Apache 2.0 License,
// // A copy of the Apache 2.0 License is included in this file.
// //
// //

// package main

// import (
// 	"time"

// 	"trpc.group/trpc-go/trpc-go"
// 	"trpc.group/trpc-go/trpc-go/admin"
// 	"trpc.group/trpc-go/trpc-go/examples/features/common"
// 	"trpc.group/trpc-go/trpc-go/filter"
// 	"trpc.group/trpc-go/trpc-go/log"
// 	"trpc.group/trpc-go/trpc-go/naming/registry"
// 	"trpc.group/trpc-go/trpc-go/plugin"
// 	"trpc.group/trpc-go/trpc-go/server"
// 	pb "trpc.group/trpc-go/trpc-go/testdata"
// 	polaris "git.code.oa.com/trpc-go/trpc-naming-polaris"
// 	poregistry "git.code.oa.com/trpc-go/trpc-naming-polaris/registry"
// 	"git.woa.com/galileo/eco/go/sdk/base/configs/ocp"
// 	"git.woa.com/galileo/eco/go/sdk/base/model"
// 	_ "git.woa.com/galileo/trpc-go-galileo"
// )

// var (
// 	serviceName = "trpc.test.helloworld.Greeter"
// 	namespace   = "Development"
// )

// func main() {
// 	setGlobalVariables()

// 	closePlugins := setupPlugins()

// 	stop := trpc.PeriodicallyUpdateGOMAXPROCS(0)

// 	s := newServer()

// 	s.RegisterOnShutdown(func() {
// 		if err := closePlugins(); err != nil {
// 			log.Errorf("failed to close plugins, err: %s", err)
// 		}
// 	})

// 	s.RegisterOnShutdown(stop)

// 	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

// 	if err := s.Serve(); err != nil {
// 		log.Fatalf("failed to serve: %v", err)
// 	}
// }

// func newServer() *server.Server {
// 	s := &server.Server{}
// 	s.AddService(
// 		admin.ServiceName,
// 		admin.NewTrpcAdminServer(
// 			admin.WithAddr(":9000"),
// 		))
// 	opts := []server.Option{
// 		server.WithServiceName(serviceName),
// 		server.WithAddress("127.0.0.1:8000"),
// 		server.WithNetwork("tcp"),
// 		server.WithProtocol("trpc"),
// 		server.WithTimeout(time.Second),
// 		server.WithRegistry(registry.Get(serviceName)),
// 		server.WithFilter(filter.GetServer("debuglog")),
// 	}
// 	if f := filter.GetServer("debuglog"); f != nil {
// 		opts = append(opts, server.WithFilter(f))
// 	}
// 	s.AddService(serviceName, server.New(opts...))
// 	return s
// }

// func setGlobalVariables() {
// 	trpc.SetGlobalConfig(
// 		&trpc.Config{
// 			Global: trpc.GlobalCfg{
// 				Namespace: namespace,
// 				EnvName:   "test",
// 			},
// 		})

// 	trpc.DefaultMaxFrameSize = 10 * 1024 * 1024

// 	plugin.SetupTimeout = 3 * time.Second
// }

// func setupPlugins() (close func() error) {
// 	configs := plugin.NewPluginConfigs()
// 	setupLogs(configs)
// 	setupPolaris(configs)
// 	setupGalileo(configs)
// 	closeFunc, err := plugin.SetupPlugins(configs)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return closeFunc
// }

// func setupLogs(configs plugin.PluginConfigs) {
// 	configs.Add("log", "default", &log.Config{
// 		log.OutputConfig{
// 			Writer: "console",
// 			Level:  "debug",
// 		},
// 		log.OutputConfig{
// 			Writer:    "file",
// 			Level:     "debug",
// 			Formatter: "json",
// 			WriteConfig: log.WriteConfig{
// 				Filename: "./trpc.log",
// 				MaxSize:  10,
// 				Compress: false,
// 			},
// 		},
// 	})
// }

// // For the complete configuration, refer to https://git.woa.com/trpc-go/trpc-naming-polaris
// func setupPolaris(configs plugin.PluginConfigs) {
// 	configs.Add("registry", "polaris", &poregistry.FactoryConfig{
// 		Services: []poregistry.Service{
// 			{
// 				ServiceName: serviceName,
// 				Namespace:   namespace,
// 				Token:       "token", // created from https://polaris.woa.com/
// 			},
// 		},
// 		Protocol: "grpc",
// 		// EnableRegister: true,
// 	})
// 	configs.Add("selector", "polaris", &polaris.Config{
// 		Timeout:  int(time.Second / time.Millisecond),
// 		Protocol: "grpc",
// 	})
// }

// // For the complete configuration, refer to https://iwiki.woa.com/p/4009274553
// func setupGalileo(configs plugin.PluginConfigs) {
// 	configs.Add("telemetry", "galileo", &ocp.GalileoConfig{
// 		Verbose: "error",
// 		Config: model.GetConfigResponse{
// 			MetricsConfig: model.MetricsConfig{Enable: true},
// 			TracesConfig: model.TracesConfig{
// 				Enable: true,
// 				Processor: model.TracesProcessor{
// 					Sampler: model.SamplerConfig{
// 						Fraction:        0.0001,
// 						ErrorFraction:   1,
// 						EnableMinSample: true,
// 						EnableDyeing:    true,
// 					},
// 				},
// 			},
// 			LogsConfig: model.LogsConfig{
// 				Enable: true,
// 				Processor: model.LogsProcessor{
// 					OnlyTraceLog:   false,
// 					MustLogTraced:  false,
// 					TraceLogMode:   0,
// 					Level:          "debug",
// 					EnableRecovery: true,
// 				},
// 			},
// 			ProfilesConfig: model.ProfilesConfig{
// 				Enable: true,
// 				Processor: model.ProfilesProcessor{
// 					ProfileTypes: []string{"cpu", "heap"},
// 				},
// 			},
// 			Version: 1,
// 		},
// 		Resource: model.Resource{
// 			Platform: "PCG-123",
// 		},
// 	})
// }
