# 1. 前言

tRPC-Go 框架配置是由框架定义的、供框架初始化使用的配置文件。正如 [tRPC 架构概述](https://iwiki.woa.com/pages/viewpage.action?pageId=490794790) 所讲的，tRPC 框架核心采用了插件化架构，将所有核心功能组件化，通过基于接口编程思想，将所有组件功能串联起来，而每个组件都是通过配置和插件 SDK 关联。tRPC 框架默认提供 `trpc_go.yaml` 框架配置文件，将所有基础组件的配置统一收拢到框架配置文件中，并在服务启动时传给组件。这样各自组件不用独立管理各自的配置。

通过本文的介绍，希望帮助用户了解以下内容：

- 框架配置的组成部分
- 如何获取配置参数的含义、取值范围和默认值
- 如何生成和管理配置文件
- 如何使用框架配置，是否可以动态配置

# 2. 使用方式

首先 tRPC-Go 框架 **不支持框架配置的动态更新**，用户在修改完框架配置后，需要 **重新启动服务** 才会生效。

如何设置框架配置大致分为两类：

- 以使用配置文件为主
- 以使用代码构建 `Config` 数据为主

这两种方式都允许使用 `Option` 参数来对配置进行局部修改。

## 2.1 使用配置文件

**系统推荐方式**：使用框架配置文件，在 `NewServer()` 启动时，会先解析框架配置文件，自动初始化所有配置好的插件，并启动服务。建议其他初始化逻辑都放在 `trpc.NewServer()` 之后，以确保框架功能已经初始化完成。tRPC-Go 的默认框架配置文件名称是 `trpc_go.yaml`，默认路径为当前程序启动的工作路径，也可以通过 `-conf` 命令行参数指定配置路径。

```go
// 使用框架配置文件方式初始化 tRPC 服务程序
func NewServer(opt ...server.Option) *server.Server
```

## 2.2 使用代码构建 `Config` 数据

此方式不需要框架配置文件，但用户需要自行组装启动参数 `Config`。`Config` 的数据结构请参考 [这里](http://godoc.woa.com/git.woa.com/trpc-go/trpc-go#Config)。使用这种方式的缺点是配置更改灵活性差，任何配置的修改都需要更改代码，且不能实现配置和程序代码的解耦。
具体例子可以参考 [examples/features/noconfig](../../examples/features/noconfig/README.md)。

```go
// 用户构建 cfg 框架配置数据，初始化 tRPC 服务程序
func NewServerWithConfig(cfg *Config, opt ...server.Option) *server.Server
```

## 2.3 使用 `Option` 修改配置

这两种方式都提供了 `Option` 参数来更改局部参数，`Option` 提供的参数请参考 [这里](http://godoc.woa.com/git.woa.com/trpc-go/trpc-go/server#Option "这里")。**`Option` 配置的优先级要高于框架配置文件配置和 `Config` 配置数据**。使用 `Option` 修改框架配置示例如下：

```go
import（
    trpc "git.code.oa.com/trpc-go/trpc-go"
    server "git.code.oa.com/trpc-go/trpc-go/server"
）

func main() {
    s := trpc.NewServer(server.WithEnvName("test"), server.WithAddress("127.0.0.1:8001"))
    // ...
}
```

> PS：在本文后面章节，我们只会讨论框架配置文件模式。使用代码构建 `Config` 数据和使用 `Option` 修改配置中参数的含义可以参考第 3 节关于配置的介绍。

# 3 配置设计

## 3.1 总体结构

框架配置文件设计主要分为四大部分：

| 分组      | 描述                                                                                             |
|---------|------------------------------------------------------------------------------------------------|
| global  | 全局配置定义了环境相关等通用配置                                                                               |
| server  | 服务端配置定义了程序作为服务端的通用配置，包括 应用名，程序名，配置路径，拦截器列表，Naming Service 列表等                                  |
| client  | 客户端配置定义了程序作为客户端时的通用配置，包括拦截器列表，要访问的 Naming Service 列表配置等。推荐客户端配置优先使用配置中心，然后才是框架配置文件中的 client 配置 |
| plugins | 插件配置收集了所有使用插件的配置，由于 plugins 使用 map 是无序管理，在启动时框架会随机逐个把插件配置传给 sdk，启动插件。插件配置格式由插件自行决定             |

### 3.2 配置详情

```yaml
# 以下配置中，如未特殊说明：String 类型默认为 ""；Integer 类型默认为 0；Boolean 类型默认为 false；[String] 类型默认为 []。

# 全局配置
global:
  # 必填，通常使用 Production 或 Development
  namespace: String
  # 选填，环境名称，具体请参考 [多环境](https://iwiki.woa.com/pages/viewpage.action?pageId=99485673) 文档
  env_name: String
  # 选填，容器名
  container_name: String
  # 选填，当未配置服务端 IP 时，使用该字段作为默认 IP
  local_ip: String(ipv4 or ipv6)
  # 选填，是否开启 set 功能，用于服务发现，默认为 N（注意，它的类型是 String，不是 Boolean）
  enable_set: String(Y, N)
  # 选填，set 分组的名字
  full_set_name: String([set 名].[set 地区].[set 组名])
  # 选填，网络收包缓冲区大小（单位 B)。<=0 表示禁用，不填使用默认值 4096
  read_buffer_size: Integer
  # 选填，定期更新 GOMAXPROCS 的时间 默认不开启
  # 123 平台支持 VPA 垂直动态扩缩容，框架可以采用 UpdateDataGOMAXPROCSInterval 周期性的更新
  # 适用于版本 >= v0.16.0
  update_gomaxprocs_interval: time.Duration
  # 选填，是否开启对 GOMAXPROCS 参数向上取整，默认为 false 
  # 采用 UpdateDataGOMAXPROCSInterval 默认只支持采用向下取整的方式来计算 GOMAXPROCS 
  # 向下取整的方式在 CPU 核数为非整数情况下可能不能充分利用 CPU 资源
  # 适用于版本 >= v0.18.5
  round_up_cpu_quota: Bool
  # 选填，最大帧长，单位为 Byte，默认为 10485760（表示 10MB）
  # 如果要调节，注意上下游要同时修改，而不要只改一端
  # 适用于版本 >= v0.15.0
  max_frame_size: Integer
  # 选填，是否关闭优雅重启功能，默认开启优雅重启功能，对 Windows 不生效
  # 适用于版本 >= v0.20.0
  disable_graceful_restart: Bool
# 服务端配置
server:
  # 必填，服务所属的应用名
  app: String
  # 必填，服务所属的服务名
  server: String
  # 选填，可执行文件的路径
  bin_path: String
  # 选填，关闭服务器时的最短等待时间（以毫秒为单位），以便完成服务注销，框架版本 v0.18.3 之后默认为 1000
  close_wait_time: Integer
  # 选填，关闭服务器时的最长等待时间（以毫秒为单位），以便完成所有请求的处理，框架版本 v0.18.3 之后默认为 2000
  max_close_wait_time: Integer 
  # 选填，数据文件的路径
  data_path: String
  # 选填，配置文件的路径
  conf_path: String
  # 选填，网络类型，当 service 未配置 network 时，以该字段为准，默认为 tcp
  network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
  # 选填，协议类型，当 service 未配置 protocol 时，以该字段为准，默认为 trpc
  protocol: String(trpc, grpc, http, etc.)
  # 选填，所有 service 共享的拦截器配置
  filter: [String]
  # 选填，所有 service 的默认超时时间，单位毫秒
  timeout: Integer
  # 选填，服务整体的默认过载保护配置，会设置到各个 service 上（如果 service 自己没有配置的话）
  # 用于在 filter 前、decode 后进行拦截，使用 trpc-overload-control 组件时，此处填 "default"
  # 适用于版本 >= v0.19.0
  overload_ctrl: String 
  # 必填，service 列表
  service:
    - # 选填，是否禁止继承上游的超时时间，用于关闭全链路超时机制，默认为 false
      disable_request_timeout: Boolean
      # 选填，方法级别的配置，要求框架版本 >= v0.15.0
      method:
        method_name:
          timeout: Integer  # 方法级别的超时时间，单位毫秒
      # 选填，service 监听的 IP 地址，如果为空，则会尝试获取网卡 IP，如果仍为空，则使用 global.local_ip
      ip: String(ipv4 or ipv6)
      # 必填，服务名，用于服务发现
      name: String
      # 选填，该 service 绑定的网卡，只有 ip 为空时，才会生效
      nic: String
      # 选填，该 service 绑定的端口，address 为空时，port 必填
      port: Integer
      # 选填，service 监听的地址，为空时使用 ip:port，非空时，忽略 ip:port
      address: String
      # 选填，网络类型，为空时，使用 server.network
      network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
      # 选填，协议类型，为空时，使用 server.protocol
      protocol: String(trpc, grpc, http, etc.)
      # 选填，可以填 tnet 来启用 tnet server transport，要求框架版本 >= v0.11.0
      transport: String(tnet, gonet)
      # 选填，service 处理请求的超时时间 单位 毫秒
      timeout: Integer
      # 选填，service 读取请求的超时时间 单位 毫秒
      # read_timeout 指定了从客户端连接读取请求的最大持续时间
      # 表示该服务中读取请求的最大持续时间
      #
      # 如果未设置，读取超时将默认为与空闲超时 (idletime) 相同的值
      #
      # 区分“超时”(timeout) 和“读取超时”(read_timeout)：
      #  - 超时：处理请求的处理程序允许的最大持续时间
      #  - 读取超时：从客户端连接读取请求允许的最大持续时间
      #
      # 至于“读取超时”(read_timeout) 和“空闲时间”(idletime) 之间的区别：
      # 在当前实现下，如果达到了读取超时但未达到空闲时间，服务端将尝试再次从连接读取请求
      # 这意味着读取过程会被读取超时定期中断，只有在达到空闲超时时才会关闭连接
      #
      # 默认情况下，read_timeout 设置为与 idletime 的默认值相同，即 60000 (60 秒)
      # 如果 read_timeout 设置得过小，可能会偶发读包不完整问题导致客户端连接被服务端关闭
      # 适用于版本 >= v0.18.0
      read_timeout: Integer
      # 选填，长连接空闲时间，单位 毫秒
      # 默认为 60000 (即 60 秒)
      idletime: Integer
      # 选填，使用哪个注册中心 polaris
      registry: String
      # 选填，拦截器列表，优先级低于 server.filter
      filter: [String]
      # 选填，服务端需要提供的 TLS 私钥，当 tls_key 和 tls_cert 都非空时，才会开启 tls 服务
      tls_key: String
      # 选填，服务端需要提供的 TLS 公钥，当 tls_key 和 tls_cert 都非空时，才会开启 tls 服务
      tls_cert: String
      # 选填，如果开启反向认证，需要提供 client 的 CA 证书
      ca_cert: String
      # 选填，服务端是否开启异步处理，默认为 true
      server_async: Boolean
      # 选填，服务在异步处理模式下，最大协程数限制，不设置或者 <=0，使用默认值：1<<31 - 1。异步模式生效，同步模式不生效
      max_routines: Integer
      # 选填，启用服务器批量发包 (writev 系统调用）, 默认为 false
      writev: Boolean
      # 选填，服务的过载保护配置，用于在 filter 前、decode 后进行拦截，使用 trpc-overload-control 组件时，此处填 "default"
      # 适用于版本 >= v0.8.1
      overload_ctrl: String 
  # 选填，服务常用的管理功能
  admin:
    # 选填，admin 绑定的 IP，默认为 localhost
    ip: String
    # 选填，网卡名，ip 字段为空时，会尝试从网卡获取 IP
    nic: String
    # 选填，admin 绑定的端口，如果为 0，即默认值，admin 功能不会开启
    port: Integer
    # 选填，读超时时间，单位为 ms，默认为 3000ms
    read_timeout: Integer
    # 选填，写超时时间，单位为 ms，默认为 3000ms
    write_timeout: Integer
    # 选填，是否开启 TLS，目前不支持，设置为 true 会直接报错
    enable_tls: Boolean

# 客户端配置
client:
  # 选填，被调命名空间，为空时，使用 global.namespace
  namespace: String
  # 选填，主调命名空间，为空时，使用 global.namespace，要求框架版本 >= v0.19.0
  # 增加主调的命名空间、环境名和 set 名是为了配置规则路由，概念可参考这里：https://iwiki.woa.com/pages/viewpage.action?pageId=102467866
  # 简单来说，对于一条请求，会先根据主调规则 (caller_namespace, caller_env_name, caller_set_name) 过滤节点，
  # 然后根据负载均衡策略和被调规则 (namespace, env_name, set_name) 等进一步筛选节点
  caller_namespace: String
  # 选填，网络类型，当 service 未配置 network 时，以该字段为准
  network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
  # 选填，协议类型，当 service 未配置 protocol 时，以该字段为准
  protocol: String(trpc, grpc, http, etc.)
  # 选填，可以填 tnet 来启用 tnet server transport，要求框架版本 >= v0.11.0
  transport: String(tnet, gonet)
  # 选填，所有 service 共享的拦截器配置
  filter: [String]
  # 选填，客户端超时时间，当 service 未配置 timeout，以该字段为准 单位 毫秒
  timeout: Integer
  # 选填，服务发现策略，当 service 未配置 discovery 时，以该字段为准
  discovery: String
  # 选填，负载均衡策略，当 service 未配置 loadbalance 时，以该字段为准
  loadbalance: String
  # 选填，熔断策略，当 service 未配置 circuitbreaker 时，以该字段为准
  circuitbreaker: String
  # 选填，客户端调用访问范围（客户端全局配置），可选项：
  #  "local": 标识 `scope` 为 `local` 的客户端将只能访问统一进程下的服务，无法按通常 RPC 方式访问远程服务（从而开启本地调用）
  #  "remote": 标识 `scope` 为 `remote` 的客户端将只能按通常 RPC 的方式访问远程服务，无法访问寻找统一进程内的服务做快捷访问以跳过序列化及网络开销，这一项是默认值（以保证和之前版本的兼容性）
  #  "all": 标识 `scope` 为 `all` 的客户端会先尝试按照 `local` 的方式进行访问，出现任何错误时会再尝试按照 `remote` 的方式进行访问
  # 要求框架版本 >= v0.20.0
  scope: String
  # 必填，被调服务列表
  service:
    - # 被调服务名
      # 如果使用 pb，callee 必须与 pb 中定义的服务名保持一致
      # callee 和 name 至少填写一个，为空时，使用 name 字段
      # 例如 trpc.test.helloworld.Greeter1
      callee: String
      # 被调服务名，常用于服务发现
      # 注意区分 [naming service 和 proto service](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)
      # name 和 callee 至少填写一个，为空时，使用 callee 字段
      # 例如 trpc.test.helloworld.Greeter1
      name: String
      # 选填，被调环境名，用于服务路由
      # 例如 test
      env_name: String 
      # 选填，被调 set 名，用于服务路由
      # 例如 set
      set_name: String
      # 选填，主调环境名，可用版本: >= v0.19.0
      caller_env_name: String
      # 选填，主调 set 名，可用版本: >= v0.19.0
      caller_set_name: String
      # 选填，是否禁用服务路由，默认为 false，服务路由概念可参考这里：https://iwiki.woa.com/pages/viewpage.action?pageId=99485673
      disable_servicerouter: Boolean
      # 选填，指定主调元数据，默认为空，可用版本: >= v0.19.0
      caller_metadata: Map (map[string]string)
      # 选填，指定被调元数据，默认为空
      callee_metadata: Map (map[string]string)
      # 选填，指定被调命名空间，为空时，使用 client.namespace
      # 例如 Production / Development
      namespace: String
      # 选填，指定主调命名空间，为空时，使用 client.caller_namespace，可用版本: >= v0.19.0
      caller_namespace: String
      # 选填，目标服务，非空时，selector 将以 target 中的信息为准
      target: String(type:endpoint[,endpoint...])
      # 选填，被调服务密码
      password: String
      # 选填，服务发现策略
      discovery: String
      # 选填，负载均衡策略
      loadbalance: String
      # 选填，熔断策略
      circuitbreaker: String
      # 选填，网络类型，为空时，使用 client.network
      network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
      # 选填，超时时间，为空时，使用 client.timeout 单位 毫秒
      timeout: Integer
      # 选填，方法级别的配置，要求框架版本 >= v0.15.0
      method:
        method_name:
          timeout: Integer  # 方法级别的超时时间，为空时，使用 client.service.timeout 单位 毫秒
      # 选填，协议类型，为空时，使用 client.protocol
      protocol: String(trpc, grpc, http, etc.)
      # 选填，序列化协议，默认为 -1，即不设置
      serialization: Integer(0=pb, 1=JCE, 2=json, 3=flat_buffer, 4=bytes_flow)
      # 选填，压缩协议，默认为 0，即不压缩
      compression: Integer(0=no_compression, 1=gzip, 2=snappy, 3=zlib)
      # 选填，client 私钥，必须与 tls_cert 配合使用
      tls_key: String
      # 选填，client 公钥，必须与 tls_key 配合使用
      tls_cert: String
      # 选填，服务端 CA 证书路径，为 none 时，跳过对服务端的认证
      ca_cert: String
      # 选填，校验 TLS 时的服务名
      tls_server_name: String
      # 选填，拦截器列表，优先级低于 client.filter
      filter: [String]
      # 选填，客户端调用访问范围，可选项：
      #  "local": 标识 `scope` 为 `local` 的客户端将只能访问统一进程下的服务，无法按通常 RPC 方式访问远程服务（从而开启本地调用）
      #  "remote": 标识 `scope` 为 `remote` 的客户端将只能按通常 RPC 的方式访问远程服务，无法访问寻找统一进程内的服务做快捷访问以跳过序列化及网络开销，这一项是默认值（以保证和之前版本的兼容性）
      #  "all": 标识 `scope` 为 `all` 的客户端会先尝试按照 `local` 的方式进行访问，出现任何错误时会再尝试按照 `remote` 的方式进行访问
      # 要求框架版本 >= v0.20.0
      scope: String
      # 以下 conn_type 相关配置适用于版本 >= v0.15.0
      # 以下是 client 连接类型为 connpool 的配置
      # connpool 配置仅支持 trpc 协议以及使用了框架连接池的协议，不支持 HTTP 等协议
      # HTTP 协议相关的连接池配置见后面的 conn_type: httppool 部分
      # 注意，conn_type 只能配置一个
      conn_type: connpool  # 连接类型为连接池，以下选项均为 connpool 配置
      connpool:
        # 优先级：选项 dial_timeout ≈ context timeout > yaml dial_timeout
        # 当选项 dial_timeout 和 context timeout 都存在时，实际拨号超时时间 = min(选项拨号超时时间，context 超时时间)
        dial_timeout: 200ms  # 连接池：拨号超时时间，默认 200ms
        force_close: false  # 连接池：是否强制关闭连接，默认 false
        idle_timeout: 50s  # 连接池：空闲超时时间，默认 50s
        max_active: 0  # 连接池：最大活动连接数，默认 0（表示无限制）
        max_conn_lifetime: 0s  # 连接池：连接最大生命周期，默认 0s（表示无限制）
        max_idle: 65536  # 连接池：最大空闲连接数，默认 65536
        min_idle: 0  # 连接池：最小空闲连接数，默认 0
        pool_idle_timeout: 100s  # 连接池：关闭整个池的空闲超时时间，默认 100s
        push_idle_conn_to_tail: false  # 连接池：将连接回收到空闲列表的头部/尾部，默认 false（头部）
        wait: false  # 连接池：当连接总数达到 max_active 时，是否等待直到超时或立即返回错误，默认 false

      # 以下是 client 连接类型为 multiplexed 的配置
      # multiplexed 配置仅支持 trpc 协议以及使用了框架多路服务池的协议，不支持 HTTP 等协议
      # 注意，conn_type 只能配置一个
      conn_type: multiplexed  # 连接类型为多路复用，以下选项均为 multiplexed 配置
      multiplexed:
        multiplexed_dial_timeout: 1s  # 多路复用：拨号超时时间，默认 1s
        conns_per_host: 2  # 多路复用：每个主机的具体（实际）连接数，默认 2
        max_vir_conns_per_conn: 0  # 多路复用：每个具体（实际）连接的最大虚拟连接数，默认 0（表示无限制）
        max_idle_conns_per_host: 0  # 多路复用：每个主机的最大空闲具体（实际）连接数，与 max_vir_conns_per_conn 一起使用，默认 0（禁用）
        queue_size: 1024  # 多路复用：每个具体（实际）连接的发送队列大小，默认 1024
        drop_full: false  # 多路复用：当队列满时是否丢弃发送包，默认 false
        max_reconnect_count: 10  # 多路复用：最大重连次数，0 表示禁用重连，默认 10，适用于版本 >= v0.18.5
        initial_backoff: 5ms  # 多路复用：第一次重连尝试的初始退避时间，默认 5ms，适用于版本 >= v0.18.5
        reconnect_count_reset_interval: 600s # 多路复用：重连次数重置间隔，适用于版本 >= v0.19.0

      # 以下是 client 连接类型为短连接的配置
      # 注意，conn_type 只能配置一个
      # 短连接配置支持 trpc 协议，也支持使用了框架的 tcp transport 的协议，也支持 HTTP 协议
      conn_type: short  # 连接类型为短连接
     
      # 以下详细介绍 tnet-multiplexed 的配置（tnet-connpool 的配置与 connpool 相同）
      # tnet-multiplexed 配置仅支持 trpc 协议以及使用了框架多路服务池的协议，不支持 HTTP 等协议
      transport: tnet
      conn_type: multiplexed  # 连接类型为多路复用，以下选项均为 multiplex 配置
      multiplexed:
        multiplexed_dial_timeout: 1s  # 多路复用：拨号超时时间，默认 1s
        max_vir_conns_per_conn: 0  # 多路复用：每个具体（实际）连接的最大虚拟连接数，默认 0（表示无限制）
        enable_metrics: true  # tnet-multiplex：是否启用指标，与 'transport: tnet' 一起使用，默认 false

      # 以下 conn_type 相关配置适用于版本 >= v0.19.0
      # 以下是 client 连接类型为 httppool 的配置，用于 HTTP 连接池配置
      # 注意，conn_type 只能配置一个
      conn_type: httppool  # 连接类型为 HTTP 连接池，以下选项均为 httppool 配置
      httppool:
        max_idle_conns: 100  # HTTP 连接池：最大空闲连接数，默认为 0（表示无限制）。
        max_idle_conns_per_host: 10  # HTTP 连接池：每个主机的最大空闲连接数，默认为 2。
        max_conns_per_host: 20  # HTTP 连接池：最大连接数，默认为 0（表示无限制）。
        idle_conn_timeout: 1s  # HTTP 连接池：空闲超时时间，默认为 0（表示无限制）。

# 插件配置，请在 [插件生态](https://iwiki.woa.com/pages/viewpage.action?pageId=447434212) 中查询插件文档链接
# 如果你想自定义插件，请参考 [插件开发](https://iwiki.woa.com/pages/viewpage.action?pageId=500033089 "插件开发")
plugins:
  # 插件类型
  ${type}:
    # 插件名
    ${name}:
      # 插件详细配置，具体请参考各个插件的说明
      Object
```

注意：服务端超时时间的配置可以为 server 级别、service 级别、method 级别：

```yaml
server:
  timeout: 100  # server 级别的超时配置，以毫秒为单位
  service:
    - name: trpc.test.helloworld.Greeter
      timeout: 200  # service 级别的超时配置，以毫秒为单位
      method:  # method 级别的配置，可用版本：>= v0.15.0
        method_name:  # 此处 method_name 需要改为具体的方法名
          timeout: 300  # method 级别的超时配置，以毫秒为单位
        method_name2:  # 此处 method_name2 需要改为具体的方法名
          timeout: 300  # method 级别的超时配置，以毫秒为单位
```

这三个级别的优先级顺序为 server 级别 < service 级别 < method 级别，比如以上配置实际 `method_name` 对应接口的服务端超时时间为 300 毫秒。

客户端超时时间的配置可以为 client 级别、service 级别、method 级别：

```yaml
client:
  timeout: 100  # client 级别的超时配置，以毫秒为单位
  service:
    - name: trpc.test.helloworld.Greeter
      timeout: 200  # service 级别的超时配置，以毫秒为单位
      method:  # method 级别的配置，可用版本：>= v0.15.0
        method_name:  # 此处 method_name 需要改为具体的方法名
          timeout: 300  # method 级别的超时配置，以毫秒为单位
        method_name2:  # 此处 method_name2 需要改为具体的方法名
          timeout: 300  # method 级别的超时配置，以毫秒为单位
```

这三个级别的优先级顺序为 client 级别 < service 级别 < method 级别，比如以上配置实际 `method_name` 对应接口的客户端超时时间为 300 毫秒。

实际上的超时时间还和全链路超时时间（由上游透传下来的超时时间，对应 `disable_request_timeout: false`）有关，会取所有能够拿到的超时时间的最小值。

此外要注意框架内没有对服务端超时做显式的控制，他被放到 ctx 的超时当中去，执行用户的 handle 的时候，用户的 handle 内部直接或间接（间接：比如使用框架的 client 发起下游调用，框架的 client 里会查 `ctx.Done()`）地 select 到这个 `ctx.Done()` 并将超时错误返回给框架时，框架才知道超时了。

具体怎么做显式 select？比如：

```go
func serverHandle(ctx context.Context, req ..) error {
    c := make(chan struct{})
    go func() {
        // do work
        c <- struct{}{}
    }()
    select {
    case <-ctx.Done():
        return errors.New("deadline exceeded")
    case <-c:
        // work done
    }
    // ...
}
```

### 3.3 client 后端配置的管理

client 后端配置除了可以放置在 trpc_go.yaml 文件里面，也可以放在 rainbow 远程配置中心，利用 [trpc-config-rainbow 插件](https://git.woa.com/trpc-go/trpc-config-rainbow) 动态获取 client 后端配置，更多的配置细节请参考 trpc-config-rainbow 插件的配置说明中的 [`enable_client_provider` 字段](https://git.woa.com/trpc-go/trpc-config-rainbow#%E4%BD%BF%E7%94%A8-enable_client_provider-%E6%B3%A8%E6%84%8F)。

```yaml
plugins:
  config:
    rainbow:
      providers:
        - name: rainbow1 # provider 名字，代码使用如：`config.WithProvider("rainbow1")`
          type: kv # 七彩石数据格式，目前只支持 kv 类型
          # 在使用七彩石来加载 client 配置时需要添加以下两行
          enable_client_provider: true # 是否开启七彩石动态修改 trpc 主调配置信息，默认为不开启，如果设置为 true 则为开启；开启后，默认动态配置主调信息全量替换框架配置，如果想要增量添加主调配置信息，可设置 client_provider_mode 为 merge
          client_provider_mode: replace # （版本要求 v0.2.11+) 七彩石配置主调服务的修改模式，默认为 replace：全量取代框架配置的主调信息，merge：增量添加主调信息，如果原来框架配置中已有，会覆盖；可以在插件初始化前调用 RegisterClientProvider 注册自定义的其他 mode
```

从 rainbow 中获取 client 后端配置可以提高服务的安全性，例如使用 trpc-database 里面的部分组件时，可能需要在 client 配置下的 target 字段设置密码，如果直接配置在 trpc_go.yaml 文件里面则可能导致密码泄露。

# 4. 创建配置

在第 2 节我们介绍了程序的启动是通过读取框架配置文件来初始化框架的。那么如何生成框架配置文件呢？本节会介绍以下三种常见方式。

## 4.1 通过工具创建配置

框架配置文件可以通过 trpc 脚手架工具在生成服务端桩代码时，自动生成相应的 `trpc_go.yaml` 文件。配置文件中会自动添加 PB 文件中定义的服务。trpc 脚手架工具命令为：

```shell
# 通过 PB 文件生成桩代码和框架配置文件"trpc_go.yaml"
trpc create --protofile=helloworld.proto
```

需要强调的是，通过工具生成的配置仅为模板配置，用户需要按照自身需求来修改配置内容。

## 4.2 通过运营平台创建

对于大型复杂系统来说，最好的实践方式是通过服务运营平台来统一管理框架配置文件，由平台统一生成框架配置文件，并下发到程序要运行的机器。

下面我们以 PCG 123 平台为例，介绍通常运营平台是怎么样管理框架配置的。123 平台负责服务的编排，知道服务的基本信息，同时 123 平台整合了服务运行所需要的所有服务治理能力，能自动生成框架配置模板。对于配置中和具体环境相关的配置，123 平台使用了`占位符`（比如 ${app} ${server} 等）来自动填充框架配置。在 123 发布服务时，框架配置会自动生成，并在服务启动时，自动将占位符替换为具体数值。

123 平台提供的默认配置见 [这里](https://git.woa.com/wod_csc_paas/123_process_script/blob/master/trpc_go/trpc_go.yaml) 。

## 4.3 环境变量替换配置

tRPC-Go 也提供了通过 golang template 模板的方式生成框架配置：支持通过读取环境变量来自动替换框架配置占位符。环境变量方式可以与 4.1 或 4.2 章节组合使用。通过工具或者运营平台创建配置文件模板，然后用环境变量替换配置文件中的环境变量占位符。

对于环境变量方式的使用，首先要在配置文件中对可变参数使用 `${var}` 来表示，如：

```yaml
server:
  app: ${app}
  server: ${server}
  service: 
    - name: trpc.test.helloworld.Greeter
      ip: ${ip}
      port: ${port}
```

框架启动时会先读取出配置文件 `trpc_go.yaml` 的文本内容，当识别到占位符时，框架自动到环境变量读取相对应的值，有则替换对应值，没有则替换成空值。

如上面的配置内容所示，环境变量需要预先设置好以下数据：

```shell
export app=test
export server=helloworld
export ip=1.1.1.1
export port=8888
```

由于框架配置会解析 `$` 符号，所以用户配置时，除了占位符以外，不要包含 `$` 字符。比如 Redis 和 MySQL 等的密码不要包含 `$` 字符。

# 5. 示例

请参考 123 平台提供了一套完整配置（默认配置见 [这里](https://git.woa.com/wod_csc_paas/123_process_script/blob/master/trpc_go/trpc_go.yaml)）。在这份配置中使用了占位符，如果你使用的是 123 平台发布服务，在服务启动时，系统会自动将占位符替换为具体数值，用户只需要修改 service name 字段的最后一段。如果你没有使用 123 平台，请自行替换配置中的占位符。

# 6. FAQ

## 6.1 框架配置相关问题

### Q1 - 如何通过代码读取框架配置数据？

请使用 `trpc.GlobalConfig().Server.Xxx` 来读取。

> PS：有些同学喜欢在代码里面获取正式环境还是测试环境，然后做不同的逻辑，建议最好还是不要这样，代码里面不要有跟环境相关的概念，而应该是使用**功能特性开关**概念，使用配置中心来切换逻辑开关。

### Q2 - Redis/MySQL 等后端配置如何使用？

后端配置可以使用 rainbow 配置中心，也可以使用发布平台的框架配置，禁止把 Redis 等密码放在 `trpc_go.yaml` 文件并提交到 git 上。

### Q3 - 报错 yaml: line xx: did not find expected key？

yaml 文件的格式配置有问题。确保每层都是两个空格缩进，上下对齐，不要有多余空格，标点符号不要用中文全角符号。

### Q4 - 报错 yaml: line 8: found character that cannot start any token？

解析配置文件失败，yaml 配置文件必须是两个空格缩进，查看是否有特殊不可见字符。

### Q5 - 如何通过命令行指定配置文件地址？

可以在启动时使用 `-conf` 命令行参数来指定：

```shell
./server -conf ../conf/trpc_go.yaml
```

也可以通过代码来指定配置文件地址：

```go
trpc.ServerConfigPath = "../conf/trpc_go.yaml"
```

如果同时通过命令行和代码指定的话，那么将以代码为准，命令行无效。

### Q6 - 客户端如何设置网络读包缓冲区大小？

对于客户端除了可以通过主动导入 `trpc_go.yaml` 配置文件外，还可以使用 [`GetReaderSize`](https://git.woa.com/trpc-go/trpc-go/blob/v0.11.0/codec/framer_builder.go#L28) 和 [`SetReaderSize`](https://git.woa.com/trpc-go/trpc-go/blob/v0.11.0/codec/framer_builder.go#L33) 两个 API 来读取或设置。

### Q7 - client 配置中的 `callee` 和 `name` 的区别是什么？

#### 解释

`callee` 是指被调方的 pb 协议文件的 service name，格式是 `pbpackage.service`。比如 pb 为：

```protobuf
package trpc.a.b

service Greeter {
    rpc SayHello(request) returns reply
}
```

那么 `callee` 即为 `trpc.a.b.Greeter`。而 `name` 是指被调方注册在名字服务（如北极星）上面的服务名，也就是被调服务的 `trpc_go.yaml` 里面的 `server.service.name` 的配置值。

> **注意**：上面这句话只在这个 client 使用 `WithServiceName` 的寻址方式下才成立。对于 `WithTarget` 的寻址方式来说，name（callee 存在则是 callee）则仅用于配置的查找，不再用于服务发现，服务发现则是通过 `target: polaris://xxxx` 中指定的 selector 来进行。对于 `WithServiceName` 以及 `WithTarget` 的详细介绍可以参考 [`client.WithServiceName` 寻址与 `client.WithTarget` 寻址的区别](https://git.woa.com/trpc-go/trpc-naming-polaris#clientwithservicename-寻址与-clientwithtarget-寻址的区别以及-enable_servicerouter-的语义) 以及 [tRPC 服务路由](https://iwiki.woa.com/p/4008319150)。

正常情况，tRPC 会默认把 pb 协议文件的 service name 注册到北极星，所以一般情况下，`callee` 和 `name` 是相同的，只需配置其中任何一个即可。但是有些场景下，如存储服务，同一份 pb 会部署多个实例，这个时候的名字服务的 service name 和 pb service name 就不一样了，此时配置文件就必须同时配置 `callee` 和 `name`：

```yaml
client:
  service:
    - callee: pbpackage.service  // 必须同时配置 callee 和 name，callee 是 pb 的 service name，用于匹配 client proxy 和配置
      name: polaris-service-name // 北极星名字服务的 service name，用于寻址
      protocol: trpc
```

通过 pb 生成的 client 桩代码，默认会把 pb service name 填入到 client 中，所以 client 寻找配置时只会**以 `callee` 为 key（也就是 pb 的 service name）来匹配**。而通过类似 `redis.NewClientProxy("trpc.a.b.c")` 等（包括 database 下面所有插件以及 http）生成的 client，默认 service name 就是用户自己输入的字符串，所以 client 寻址配置时**以 `NewClientProxy` 的输入参数为 key（即以上的 `"trpc.a.b.c"`）来匹配**。

> 1. 不是说配置中的 `name` 与代码中的 `pb.NewXxxClientProxy(name)` 传的 `name` 保持一致，并且北极星上也有 `name` 的注册就是没问题的！一定要去桩代码 `xxx.trpc.go` 里面看 service descriptor 中的 service name（往下面看具体在哪里）是否和这些值一致，如果不一致，那么就要在配置中显式写出来 `callee` 和 `name`，把 `callee` 填成桩代码中的 service descriptor 字段才行（这里可以参考下边的实际业务示例中的说明）！
> 2. 在 trpc-go 框架版本 v0.10.0 之后，支持了同时以 callee 及 name 为 key 来寻找配置，比如以下两个客户端配置共享了相同的 callee:
>
> ```yaml
> client:
>   service:
>     - callee: pbpackage.service   # 必须同时配置 callee 和 name，callee 是 pb 的 service name，用于匹配 client proxy 和配置
>       name: polaris-service-name1 # 北极星名字服务的 service name，用于寻址
>       protocol: trpc
>     - callee: pbpackage.service   # 必须同时配置 callee 和 name，callee 是 pb 的 service name，用于匹配 client proxy 和配置
>       name: polaris-service-name2 # 北极星名字服务的 service name，用于寻址
>       protocol: trpc
> ```
>
> 用户在代码中可以使用 `client.WithServiceName` 来同时用 `callee` 以及 `name` 作为 key 进行配置的寻找：
>
> ```go
> // proxy1 使用第一项配置
> proxy1 := pb.NewClientProxy(client.WithServiceName("polaris-service-name1"))
> // proxy2 使用第二项配置
> proxy2 := pb.NewClientProxy(client.WithServiceName("polaris-service-name2"))
> ```
>
> 在低于 v0.10.0 的版本中，上述写法都只会找到第二项配置 (存在 `callee` 相同的配置时，后面的会覆盖前面的)。

#### 实际业务示例

比如现在有个用户反馈他们的客户端配置疑似没有生效，他们的调用目标在北极星上注册的名字为 `trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline`，他们在 [代码](https://git.woa.com/yyb-cloud-game/cloud-game/blob/2fa4177be0519783a20a501a58b65e8e20593e71/cloud_game_midgame/proxy/cloud_game.go#L56) 中初始化 client proxy 的代码为：

```go
pipeline.NewPipelineClientProxy(client.WithServiceName("trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline"))
```

其中 `client.WithServiceName` 指定的 `name` 和北极星注册的是完全一样的，然后他们的客户端配置如下：

```yaml
client:
  service:
    - name: "trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline"
      namespace: Production
      target: "polaris://trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline"  
      network: tcp
      protocol: trpc
      timeout: 5000
      disable_servicerouter: true
```

可以看到，配置里面的 `name` 以及 `target` 的对象都和北极星注册的完全一致，但是用户反馈 `disable_servicerouter: true` 配置项疑似没有生效，也就是说，这段客户端配置是看上去没有生效的。我们需要去找调用的桩代码所在 `xxx.trpc.go` 中的 service descriptor 里的 service name，[具体代码](https://git.woa.com/trpcprotocol/yybgame/blob/cloud_game_midgame_pipeline_pipeline/v1.1.97/cloud_game_midgame_pipeline_pipeline/pipeline.trpc.go#L667) 如下：

```go
var PipelineServer_ServiceDesc = server.ServiceDesc {
    ServiceName: "trpc.yybgame.cloud_game_midgame_pipeline.Pipeline",
    HandlerType: ((*PipelineService)(nil)),
    // ...
}
```

我们发现桩代码中的 proto service name 为 `trpc.yybgame.cloud_game_midgame_pipeline.Pipeline`，这个 proto name 和北极星 `name` 是不一致的，所以此时需要显式在客户端配置中写上 `callee` 以使其能够找到客户端配置，即：

```yaml
client:
  service:
    - name: "trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline"
      callee: "trpc.yybgame.cloud_game_midgame_pipeline.Pipeline"  # 添加这一行，从而使这个配置能够被框架找到
      namespace: Production
      target: "polaris://trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline"  
      network: tcp
      protocol: trpc
      timeout: 5000
      disable_servicerouter: true
```

业务方又问，为什么之前配置没找到，但是服务调用是能通的呢？—— 这是因为代码中刚好有一个 client option 指定了 `WithServiceName => client.WithServiceName("trpc.yybgame.cloud_game_midgame_pipeline.midgamepipeline")`，这个 option 会启用框架的 `WithServiceName` 的寻址模式。又因为 trpc-naming-polaris 插件会自动替换框架的各种寻址模块，因此实际会走北极星服务发现，按照指定的 `name` 做寻址。假如没有这个 option 的话，就会按照 proto name 做北极星服务发现，而 proto name 没有在北极星上注册的话，就会直接报错。

> PS：即使没有 `client.WithServiceName` 这个 option，在配置不生效，并且没有 `client.WithTarget` 的情况下，trpc 的 client proxy 默认走的也是 `WithServiceName` 的寻址模式，使用的 service name 是 `pb.NewXxxClientProxy("some-name")` 时传入的 `"some-name"`。

关于 `WithServiceName` 和 `WithTarget` 寻址的具体区别可以阅读：[tRPC-Go 服务路由](https://iwiki.woa.com/p/4008319150)，以及 [trpc-naming-polaris 的 README](https://git.woa.com/trpc-go/trpc-naming-polaris#trpc-go-北极星名字服务插件)。

### Q8 - 框架配置不生效如何排查？

框架配置不生效有很多原因，注意看有没有全部满足以下条件：

- 框架配置是通过 `trpc.NewServer` 加载的，所以必须在 `NewServer` 之后才能使用配置。
- 如果是 client 配置相关，先理解以上 `Q7 - callee 和 name 的区别`，client 配置只会以 callee 为 key 匹配，不会以 name 为 key（如果同一个 server 里面调用了多个相同 pb 的不同被调服务，则配置文件只能匹配一个，其他的只能通过代码 `Option` 设置）。
- 仔仔细细查看是否有字符拼写错误情况。
- 如果是怀疑超时配置不生效，那大概率是对超时原理还不了解，先仔细看一遍 [超时控制](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688) 文档，注意：
  - trpc-go 的超时是通过 context 控制的，务必提前仔细理解一下 context 的原理。
  - 发起 RPC 请求时，必须从请求入口的 ctx 一直透传下去。
  - 自己启动 goroutine 发起异步请求时，不可使用请求入口的 ctx。
  - filter 内部不能有阻塞操作，不然会导致超时失效进而导致请求卡死。
  - 注意确认 client 配置的 name 是否正确，有可能是配置没对齐。
- 框架新版本 v0.8.2 以上增加了更加严格的校验，只能配置必须字段，没有使用的字段不允许配置，必须删除。
- 如果是配置中心的 client 配置，配置有错时，首次启动会 panic，运行过程中更新配置则不会生效。
- `trpc_go.yaml` 框架配置的 client 块和配置中心的 client 配置，两者只能选一个，不能同时配置。
- 还有问题就把框架和配置中心插件全部升级到最新版。

### Q9 - 如何配置 log 的 trace 级别？

trpc-go 的 log 底层使用的是 zap 开源库，由于 zap 不支持 trace，所以需要另外设置环境变量才能开启 trace 级别日志（level 也需要配置成`trace`）：

```shell
export TRPC_LOG_TRACE=1
```

### Q10 - 如何自定义 plugin 配置？

请参考 [tRPC-Go 插件开发向导](https://iwiki.woa.com/p/500033089) 中的介绍。

### Q11 - 配置文件如何管理？

1. tRPC 配置分为框架配置和业务自定义配置。插件配置是属于框架配置，放在 `trpc_go.yaml` 中。对于 PCG 来说，`trpc_go.yaml` 文件由 123 平台自动生成并可以在平台上管理。
2. 业务配置支持自定义 yaml 配置，由业务自己管理。同时，tRPC 支持 rainbow 远程配置，业务可以在 rainbow 平台上进行动态配置。
3. client 后端配置可以配置在 `trpc_go.yaml` 框架配置里面，也可配置在 [rainbow 远程配置中心](https://git.woa.com/trpc-go/trpc-config-rainbow)，rainbow 默认提供 `client.yaml` 格式的配置，自动更新注册到 client。

### Q12 - 如何动态更新日志级别？

请见 [这里](https://iwiki.woa.com/p/99485663#设置框架日志级别)。

### Q13 - 如何确定运行时使用的配置文件路径？

可以在 `trpc.NewServer()` 后读取 `trpc.ServerConfigPath`:
 ```go
func main() {
    s := trpc.NewServer()
    // 确保在 trpc.NewServer() 之后获取最新的 trpc.ServerConfigPath。
    log.Debugf("server config path: %s", trpc.ServerConfigPath)
    pb.RegisterGreeterService(s, new(greeterImpl))
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

## 6.2 rainbow 配置相关问题

### Q1 - 七彩石没有删除类型事件吗？

是的，由于七彩石 sdk 的实现问题，暂时不支持通知删除事件。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
