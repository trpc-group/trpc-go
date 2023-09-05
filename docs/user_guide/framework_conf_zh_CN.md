# tRPC-Go 框架配置

## 前言

tRPC-Go 框架配置是由框架定义的，供框架初始化使用的配置文件。tRPC 框架核心采用了插件化架构，将所有核心功能组件化，通过基于接口编程思想，将所有组件功能串联起来，而每个组件都是通过配置和插件 SDK 关联。tRPC 框架默认提供 `trpc_go.yaml` 框架配置文件，将所有基础组件的配置统一收拢到框架配置文件中，并在服务启动时传给组件。这样各自组件不用独立管理各自的配置。

通过本文的介绍，希望帮助用户了解以下内容：
- 框架配置的组成部分
- 如何获取配置参数的含义，取值范围，默认值
- 如何生成和管理配置文件
- 如何使用框架配置，是否可以动态配置

## 使用方式

首先 tRPC-Go 框架不支持框架配置的动态更新，用户在修改完框架配置后，需要**重新启动服务**才会生效。如何设置框架配置有以下三种方式。

### 使用配置文件

**推荐**：使用框架配置文件，`trpc.NewServer()` 在启动时，会先解析框架配置文件，自动初始化所有配置好的插件，并启动服务。建议其他初始化逻辑都放在 `trpc.NewServer()` 之后，以确保框架功能已经初始化完成。tRPC-Go 的默认框架配置文件名称是`trpc_go.yaml`，默认路径为当前程序启动的工作路径，可通过 `-conf` 命令行参数指定配置路径。

```go
// 使用框架配置文件方式，初始化 tRPC 服务程序
func NewServer(opt ...server.Option) *server.Server
```

### 构建配置数据

**不推荐**：此方式不需要框架配置文件，但用户需要自行组装启动参数 `*trpc.Config`。使用这种方式的缺点是配置更改灵活性差，任何配置的修改都需要更改代码，不能实现配置和程序代码的解耦。

```go
// 用户构建 cfg 框架配置数据，初始化 tRPC 服务程序
func NewServerWithConfig(cfg *Config, opt ...server.Option) *server.Server
```

### Option 修改配置

这两种方式都提供了 `Option` 参数来更改局部参数。`Option` 配置的优先级高于框架配置文件配置和 `Config` 配置数据。使用 `Option` 修改框架配置示例如下：

``` go
import (
    "git.code.woa.com/trpc-go/trpc-go"
    server "git.code.woa.com/trpc-go/trpc-go/server"
)
func main() {
    s := trpc.NewServer(server.WithEnvName("test"), server.WithAddress("127.0.0.1:8001"))
    //...
}
```

> 在本文后面章节，我们只会讨论框架配置文件模式。


## 配置设计

### 总体结构

框架配置文件设计主要分为四大部分：

| 分组 | 描述 |
| ------ | ------ |
| global | 全局配置定义了环境相关等通用配置 |
| server | 服务端配置定义了程序作为服务端的通用配置，包括 应用名，程序名，配置路径，拦截器列表，Naming Service 列表等 |
| client | 客户端配置定义了程序作为客户端时的通用配置，包括拦截器列表，要访问的 Naming Service 列表配置等。推荐客户端配置优先使用配置中心，然后才是框架配置文件中的 client 配置 |
| plugins | 插件配置收集了所有使用插件的配置，由于 plugins 使用 map 是无序管理，在启动时框架会随机逐个把插件配置传给 sdk，启动插件。插件配置格式由插件自行决定 |

### 配置详情

``` yaml
# 以下配置中，如未特殊说明：String 类型默认为 ""；Integer 类型默认为 0；Boolean 类型默认为 false；[String] 类型默认为 []。
# 全局配置
global:
  # 必填，通常使用 Production 或 Development
  namespace: String
  # 选填，环境名称
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
# 服务端配置
server:
  # 必填，服务所属的应用名
  app: String
  # 必填，服务所属的服务名
  server: String
  # 选填，可执行文件的路径
  bin_path: String
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
  # 必填，service 列表
  service:
    - # 选填，是否禁止继承上游的超时时间，用于关闭全链路超时机制，默认为 false
      disable_request_timeout: Boolean
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
      # 选填，service 处理请求的超时时间 单位 毫秒
      timeout: Integer
      # 选填，长连接空闲时间，单位 毫秒
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
  # 选填，为空时，使用 global.namespace
  namespace: String
  # 选填，网络类型，当 service 未配置 network 时，以该字段为准
  network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
  # 选填，协议类型，当 service 未配置 protocol 时，以该字段为准
  protocol: String(trpc, grpc, http, etc.)
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
  # 必填，被调服务列表
  service:
    - # 被调服务名
      # 如果使用 pb，callee 必须与 pb 中定义的服务名保持一致
      # callee 和 name 至少填写一个，为空时，使用 name 字段
      callee: String
      # 被调服务名，常用于服务发现
      # name 和 callee 至少填写一个，为空时，使用 callee 字段
      name: String
      # 选填，环境名，用于服务路由
      env_name: String
      # 选填，set 名，用于服务路由
      set_name: String
      # 选填，是否禁用服务路由，默认为 false
      disable_servicerouter: Boolean
      # 选填，为空时，使用 client.namespace
      namespace: String
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
# 插件配置
plugins:
  # 插件类型
  ${type}:
    # 插件名
    ${name}:
      # 插件详细配置，具体请参考各个插件的说明
      Object
```

## 创建配置

我们已经介绍了程序的启动是通过读取框架配置文件来初始化框架的。那么如何生成框架配置文件呢？本节会介绍以下三种常见方式。

### 通过工具创建配置

框架配置文件可以通过 [trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline) 工具生成。配置文件中会自动添加 PB 文件中定义的服务。

```shell
# 通过 PB 文件生成桩代码和框架配置文件 "trpc_go.yaml"
trpc create -p helloworld.proto
```

需要强调的是，通过工具生成的配置仅为模板配置，用户需要按照自身需求来修改配置内容。

### 通过运营平台创建

对于大型复杂系统来说，最好的实践方式是通过服务运营平台来统一管理框架配置文件，由平台统一生成框架配置文件，并下发到程序要运行的机器。

### 环境变量替换配置

tRPC-Go 也提供了 golang template 模板的方式生成框架配置：支持通过读取环境变量来自动替换框架配置占位符。通过工具或者运营平台创建配置文件模板，然后用环境变量替换配置文件中的环境变量占位符。

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

框架启动时会先读取出配置文件 `trpc_go.yaml`  的文本内容，当识别到占位符时，框架自动到环境变量读取相对应的值，有则替换对应值，没有则替换成空值。

如上面的配置内容所示，环境变量需要预先设置好以下数据：

```shell
export app=test
export server=helloworld
export ip=1.1.1.1
export port=8888
```

由于框架配置会解析 `$` 符号，所以用户配置时，除了占位符以外，不要包含 `$` 字符，比如 redis/mysql 等密码不要包含 `$`.


## 示例

[https://github.com/trpc-group/trpc-go/blob/main/testdata/trpc_go.yaml](https://github.com/trpc-group/trpc-go/blob/main/testdata/trpc_go.yaml)
