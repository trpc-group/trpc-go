
# 前言

目前，tRPC-Go 框架启动服务依赖 trpc_go.yaml 配置文件，如果没有配置文件，服务启动会失败。在一些场景下，用户并不方便指定配置文件，但也无法做到使用纯代码的形式启动 tRPC-Go 服务，导致使用不便。本示例旨在指导用户，抛开框架配置文件，使用纯代码的形式，启动你的 tRPC-Go 服务。

# 介绍

tRPC-Go 默认使用方式是使用 trpc_go.yaml 配置文件，然后服务启动时只需要很简单的代码

```go
s := trpc.NewServer()

pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

if err := s.Serve(); err != nil {
    log.Fatalf("failed to serve: %v", err)
}
```

其中 `trpc.NewServer` 函数实现了所有的初始化工作，函数内部会读取 trpc_go.yaml 配置文件，解析得到配置信息，根据配置信息设置全局变量，初始化插件，初始化 service 信息等等。如果现在希望不使用配置文件启动服务，就不能直接调用 `trpc.NewServer` 初始化服务，而需要根据自己的需求自己调用框架 API 初始化服务。

# 全局变量

在框架配置中，包含全局配置，例如

```yaml
global:
  namespace: Development
  env_name: test
  plugin_setup_timeout: 3s
  max_frame_size: 10485760
```

可以通过设置全局变量和全局配置来使用代码实现

```golang
trpc.SetGlobalConfig(
    &trpc.Config{
        Global: trpc.GlobalCfg{
            Namespace: "Development",
            EnvName:   "test",
        },
    })

trpc.DefaultMaxFrameSize = 10 * 1024 * 1024

plugin.SetupTimeout = 3 * time.Second
```

# 插件

在框架配置中，包含插件配置，包括日志插件，名字服务插件等等。

## 日志插件

这里以本地文件日志插件作为例子，假设用户希望日志以 Debug 级别输出到命令行；同时以 Debug 级别输出到 trpc.log 文件，文件的日志格式使用 json，文件最大为 10MB，不做压缩。本来需要添加如下配置到 trpc_go.yaml。

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
      - writer: file
        level: debug
        formatter: json
        writer_config:
          filename: ./trpc.log
          max_size: 10
          compress: false
```

现在可以通过代码的形式实现上述配置：

```golang
configs := plugin.NewPluginConfigs()
configs.Add("log", "default", &log.Config{
    log.OutputConfig{
        Writer: "console",
        Level:  "debug",
    },
    log.OutputConfig{
        Writer:    "file",
        Level:     "debug",
        Formatter: "json",
        WriteConfig: log.WriteConfig{
            Filename: "./trpc.log",
            MaxSize:  10,
            Compress: false,
        },
    },
})

if _, err := plugin.SetupPlugins(configs); err != nil {
    panic(err)
}
```

## 路由插件 - 北极星

这里以北极星插件作为例子，假设用户希望配置北极星服务注册和路由寻址功能，实现 `trpc.test.helloworld.Greeter` 服务的注册，并且使用北极星进行服务发现，本来需要添加如下配置到 trpc_go.yaml。

```yaml
plugins:
  registry:
    polaris:  # 北极星名字注册服务的配置
      # register_self: true  # 是否进行服务自注册，默认为 false, 交由 123 平台注册 (非 123 平台的话一般这里要改为 true)
      heartbeat_interval: 3000  # 名字注册服务心跳上报间隔
      protocol: grpc  # 名字服务远程交互协议类型
      service:  # 需要进行注册的各服务信息
        - name: trpc.test.helloworld.Greeter  # 服务名 1, 一般和 trpc_go.yaml 中 server config 处的各个 service 一一对应
          namespace: Development  # 该服务需要注册的命名空间，分正式 Production 和非正式 Development 两种类型
          token: xxx  # 前往 https://polaris.woa.com/ 进行申请或查看

  selector:  # 针对 trpc 框架服务发现的配置
    polaris:  # 北极星服务发现的配置
      timeout: 1000  # 单位 ms，默认 1000ms，北极星获取实例接口的超时时间
      protocol: grpc  # 名字服务远程交互协议类型
```

现在可以通过代码的形式实现上述配置：

```golang
configs := plugin.NewPluginConfigs()
configs.Add("registry", "polaris", &poregistry.FactoryConfig{
    Services: []poregistry.Service{
        {
            ServiceName: serviceName,
            Namespace:   namespace,
            Token:       "token", // created from https://polaris.woa.com/
        },
    },
    Protocol: "grpc",
    // EnableRegister: true,
})

configs.Add("selector", "polaris", &polaris.Config{
    Timeout:  int(time.Second / time.Millisecond),
    Protocol: "grpc",
})

if _, err := plugin.SetupPlugins(configs); err != nil {
    panic(err)
}
```

配置文件中的每个字段都和代码中的结构体字段一一对应，这里只展示了北极星插件基础的功能，完整的北极星插件配置见：https://git.woa.com/trpc-go/trpc-naming-polaris。

## Telemetry 插件 - 伽利略

这里以伽利略插件作为例子，假设用户希望配置伽利略的链路追踪，远程日志和 自动上报 Profile 功能，实现调用链路的数据上报和监控，本来需要添加如下配置到 trpc_go.yaml。

```yaml
plugins:
  telemetry:
    galileo:
      verbose: error   # 伽利略自身的诊断日志级别，取值范围：debug, info, error, none，日志输出在 ./galileo/galileo.log 中。
      config: #配置
        metrics_config: # 指标配置
          enable: true    # 是否启用指标
        traces_config: # 追踪配置
          enable: true    # 是否启用追踪，默认 true。如果设置为 false，会中断 trace，让上游的调用链不完整。v0.3.7 以上生效。
          processor: # 追踪数据处理相关配置
            sampler: # 采样器配置
              fraction: 0.0001   # 采样比例，默认 0。（v0.11.0)
              error_fraction: 1
              enable_min_sample: true  # 启用每分钟每接口最少 1 个请求采样，默认 true (v0.11.0)。
              enable_dyeing: true # 开启染色采样，默认 true。
            disable_trace_body: false          # 若为 true，则关闭 trace 中对 req 和 rsp 的 body 上报，可以大幅提高上报性能。默认 true。
            enable_deferred_sample: false     # 开启延迟采样（请求处理完采样），默认 false。0.3.0 以上生效。
            deferred_sample_error: false      # 开启延迟采样出错采样（请求处理完出现错误采样），默认 false。0.3.0 以上生效。
            deferred_sample_slow_duration_ms: 1000    # 慢操作阈值（请求耗时超过该值采样），单位 ms，默认 1000。0.3.0 以上生效。
            disable_parent_sampling: false            # 忽略上游的采样结果，默认 false。v0.3.7 以上生效。
        logs_config: # 日志配置
          enable: true    # 是否启用日志
          processor: # 日志数据处理相关配置
            only_trace_log: false  # 是否只上报命中 trace 的 log，默认关闭
            must_log_traced: false # 是否命中 traced 不管任何级别日志都上报，默认关闭。v0.3.22 以上生效，详细参考「2.2.3.2 命中采样突破日志级别」
            trace_log_mode: 0   # debug 访问日志 (access_log) 打印模式，0 不打印，1：单行打印，3：多行打印，2：不打印，默认 0
            level: debug        # 上报到远程的日志级别，默认 error
            enable_recovery: true # 是否捕获 panic，默认 true
        profiles_config:    # profile 配置
          enable: true # 是否启用 profile
          processor: # profile 数据处理相关配置
            profile_types: ["cpu", "heap"] # 采集 profile 的类型，支持 cpu、heap、mutex、block、goroutine，默认开启 cpu 和 heap。
        version: 1        # 版本号，默认 0，此版本号用于控制远程配置和本地配置的优先级，版本号高的优先，一般设置成 1 即可。
      resource: # resource 资源信息，在 SDK 运行期间不会改变。resource 中的字段一般不需要配置，默认会填充。
        platform: PCG-123   # 服务部署的平台，如 PCG-123, STKE, 默认 PCG-123
```

现在可以通过代码的形式实现上述配置：

```go
configs := plugin.NewPluginConfigs()
configs.Add("telemetry", "galileo", &ocp.GalileoConfig{
    Verbose: "error",
    Config: model.GetConfigResponse{
        MetricsConfig: model.MetricsConfig{Enable: true},
        TracesConfig: model.TracesConfig{
            Enable: true,
            Processor: model.TracesProcessor{
                Sampler: model.SamplerConfig{
                    Fraction:        0.0001,
                    ErrorFraction:   1,
                    EnableMinSample: true,
                    EnableDyeing:    true,
                },
            },
        },
        LogsConfig: model.LogsConfig{
            Enable: true,
            Processor: model.LogsProcessor{
                OnlyTraceLog:   false,
                MustLogTraced:  false,
                TraceLogMode:   0,
                Level:          "debug",
                EnableRecovery: true,
            },
        },
        ProfilesConfig: model.ProfilesConfig{
            Enable: true,
            Processor: model.ProfilesProcessor{
                ProfileTypes: []string{"cpu", "heap"},
            },
        },
        Version: 1,
    },
    Resource: model.Resource{
        Platform: "PCG-123",
    },
})
if _, err := plugin.SetupPlugins(configs); err != nil {
    panic(err)
}
```

配置文件中的每个字段都和代码中的结构体字段一一对应，这里只展示了伽利略插件基础的功能，完整的伽利略插件配置见：https://iwiki.woa.com/p/4009274553。

# 添加 server.service

## 普通 service

在框架配置中，包含 service 信息，例如

```yaml
server:
  service:
    - name: trpc.test.helloworld.Greeter
      ip: 127.0.0.1
      port: 8080
      network: tcp
      protocol: trpc
      timeout: 1000
      filter:
        - debuglog
```

可以通过 server 包相关 API 实现

```golang
s := &server.Server{}
serviceName := "trpc.test.helloworld.Greeter"
opts := []server.Option{
    server.WithServiceName(serviceName),
    server.WithAddress("127.0.0.1:8000"),
    server.WithNetwork("tcp"),
    server.WithProtocol("trpc"),
    server.WithTimeout(time.Second),
    server.WithRegistry(registry.Get(serviceName)),
    server.WithFilter(filter.GetServer("debuglog")),
}
if f := filter.GetServer("debuglog"); f != nil {
    opts = append(opts, server.WithFilter(f))
}
s.AddService(serviceName, server.New(opts...))
```

## Admin service

admin 是服务提供管理的 service，例如如下配置的 admin 

```yaml
server:                                            # server configuration.
  admin:
    ip: 127.0.0.1                                  # ip.
    port: 9028                                     # default: 9028.
```

admin 也属于 service，可以用 server 包相关的 API 实现

```golang
s := &server.Server{}
s.AddService(
    admin.ServiceName,
    admin.NewTrpcAdminServer(
        admin.WithAddr("127.0.0.1:9028"),
    ))
```

# 添加 client.service
在框架配置中，包含 service 信息，例如

```yaml
global:
  namespace: Development
  env_name: test
client:                                     
  service:                                  
    - callee: trpc.test.helloworld.Greeter  
      name: trpc.test.helloworld.Greeter1   
      target: ip://127.0.0.1:8521           
      network: tcp                          
      protocol: trpc                        
      timeout: 800                         
      serialization: 0                     
```

可以通过 client 包相关 API 实现

```go
func setupClients() error {
    backendCfg := &client.BackendConfig{
    Callee:      "trpc.test.helloworld.Greeter",
    ServiceName: "trpc.test.helloworld.Greeter1",
    Target:      "ip://127.0.0.1:8123",
    Network:     "tcp",
    Protocol:    "trpc",
    Timeout:     800,
    }
    if err := client.RegisterClientConfig(backendCfg.Callee, backendCfg); err != nil {
        return err
    }
    return nil
}
```

# 其它

`trpc.NewServer` 除了实现上述的初始化操作外，还有一些额外的初始化行为，这里重点说明下

## 定期更新 runtime.GOMAXPROCS

runtime.GOMAXPROCS 变量表示进程允许使用的最大 CPU 数量，默认值是 runtime.NumCPU，在物理机和虚拟机上，这个默认配置没有问题。但是在容器化环境里，runtime.NumCPU 的值通常是宿主机的 CPU 数量而不是容器配额，这就导致容器化环境里进程的 runtime.GOMAXPROCS 变量会设置得比实际配额大，容器中的进程就可能会触发 throttle，请求处理时延上升。为了解决这个问题，tRPC-Go 框架会使用开源库 [automaxprocs](go.uber.org/automaxprocs/maxprocs) 根据容器实际配额设置 runtime.GOMAXPROCS 变量。考虑到容器可能出现垂直扩缩容，所以需要定时的更新 runtime.GOMAXPROCS 值。

建议在服务启动的时候开启定时更新 runtime.GOMAXPROCS 变量的功能：

```golang
trpc.PeriodicallyUpdateGOMAXPROCS(10 * time.Second)
```

## 服务停止回调

在服务停止后，需要执行一些回调函数，例如执行插件的关闭回调函数，关闭定期更新 runtime.GOMAXPROCS 协程。

必须要向 server.Server 注册服务停止的回调函数：

```golang
s := &server.Server{}

closePlugins, _ := plugin.SetupPlugins(configs)

stop := trpc.PeriodicallyUpdateGOMAXPROCS(10 * time.Second)

s.RegisterOnShutdown(func() {
    if err := closePlugins(); err != nil {
        log.Errorf("failed to close plugins, err: %s", err)
    }
})

s.RegisterOnShutdown(stop)
```
