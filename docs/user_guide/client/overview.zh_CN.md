[English](overview.md) | 中文

tRPC-Go 客户端开发向导

# 前言

tRPC-Go 框架和插件为服务提供了接口调用，用户可以像调用本地函数一样来调用下游服务，而不用关心底层实现细节。本文首先通过对服务调用的整个处理流程的梳理，来介绍框架为服务调用能提供哪些能力，用户可以采用哪些手段来控制服务调用各个环节的行为。接下来，本文会从服务调用，配置，寻址，拦截器，协议选择等关键环节来阐述如何开发和配置一个客户端调用。文章会就服务调用的典型场景为用户提供开发指导，尤其是程序既作为服务端又作为客户端的场景。

# 框架能力

本节首先会介绍框架支持的服务调用类型，然后通过对服务调用的整个处理流程的梳理，来了解框架为服务调用提供了哪些能力，哪些关键环节的行为是可以定制。从而为后面客户端的开发提供知识基础。

## 调用类型

tRPC-Go 框架提供了多种类型的服务调用，我们把服务调用按协议大致分为：内置协议调用，第三方协议调用，存储协议调用，消息队列生产者调用 这 4 类。这些协议的调用接口各不相同，比如 tRPC 协议提供的是 PB 文件定义好的服务接口，而 mysql 则提供的接口是“query()，exec()，transaction()”。用户在开发客户端时，需要查询各自协议文档来获取接口信息，可以参考以下开发指导文档：

**内置协议**：

tRPC-Go 提供了以下内置协议的服务调用：

- [调用 tRPC 服务](/docs/quick_start.zh_CN.md)

**第三方协议**：

tRPC-Go 提供了丰富的协议插件，供客户端实现和第三方协议服务进行对接。同时框架也支持用户自定义协议插件。关于协议插件的开发，请参考 [这里](/docs/developer_guide/develop_plugins/protocol.zh_CN.md)，常见的第三方协议可以参考 [trpc-ecosystem/go-codec](https://github.com/trpc-ecosystem/go-codec)

**存储协议**：

tRPC-Go 对常见数据库的访问做了封装，通过以服务访问的方式来进行数据库操作，具体可以参考 [tRPC-Go 调用存储服务](/docs/developer_guide/develop_plugins/storage.zh_CN.md)。

**消息队列**：

tRPC-Go 提供了对常见消息队列的生产者操作做了封装，通过以服务访问的方式来生产消息。

- [kafka](https://github.com/trpc-ecosystem/go-database/tree/main/kafka)

虽然各个协议的调用接口各不相同，但是框架采用了统一服务调用流程，让所有的服务调用都能复用相同的服务治理能力，包括拦截器，服务寻址，监控上报等能力。

## 调用流程

接下来让我们来看看一次完整的服务调用流程是怎么样的。下面这张图展示了客户端从发生服务调用请求到收到服务响应的全过程，图中第一行从左往右代表服务请求的流程。第二行从右往左的方向，代表客户端处理服务响应报文的流程。

![call_flow](/.resources/user_guide/client/overview/call_flow_zh_CN.png)

框架为每个服务都提供了一个服务调用代理 (又称为 "ClientProxy"), 它封装了服务调用的接口函数（“桩函数”），包括接口的入参，出参和错误返回码。从用户使用上来讲，桩函数的调用和本地函数的调用是一样的。

正如 tRPC 框架概述所描述的，框架采用了基于接口编程的思想，框架只提供了标准接口，由插件来实现具体功能。从流程图可以看到，服务调用的核心流程包括拦截器的执行，服务寻址，协议处理和网络连接这四部分，而每个部分都是通过插件来实现的，用户需要选择和配置插件，来完成整个调用流程的串联。

用户可以通过框架配置文件和 Option 函数选项两种方式来选择和配置插件，同时框架也支持用户自行开发插件来实现服务调用行为的定制。拦截器是最为典型的使用场景，例如自定义拦截器实现服务调用的认证和授权，调用质量的监控和上报等。

服务寻址是服务调用流程中非常重要的一个环节，寻址插件（selector）在服务大规模使用场景，提供服务实例的策略路由选择，负载均衡和熔断处理能力，是客户端开发中需要特别关注的部分。

## 治理能力

tRPC-Go 除了为各种协议提供了接口调用外，还为服务的调用提供了丰富的服务治理能力，实现与服务治理组件的对接，开发人员只需要关注业务自身逻辑即可。框架通过插件可以实现以下服务治理能力：

- 服务寻址
- [调用超时控制](/docs/user_guide/timeout_control.zh_CN.md)
- [拦截器机制](/docs/developer_guide/develop_plugins/interceptor_zh-CN.md)，实现包括，调用链跟踪，监控上报，[重试对冲](https://github.com/trpc-ecosystem/go-filter/tree/main/slime)....
- [远程日志](/log/README.zh_CN.md)
- [配置中心](/config/README.zh_CN.md)
- ......

# 客户端开发

本节主要以代码开发的角度，阐述业务如何初始化客户端，如何调用服务接口，以及如何通过参数配置来控制服务调用的行为。

## 开发模式

客户端开发主要分成以下两种模式：

- 模式一：程序既作为服务端也作为客户端。tRPC-Go 服务调用下游的客户端请求为最常见的场景
- 模式二：非服务的纯客户端小工具请求，常见于开发运维小工具的场景

### 服务内调用 client

对于模式一，在创建启动服务的时候会读取框架配置文件，所有配置插件的初始化都会在 trpc.NewServer() 里自动完成。代码示例为：

```go
import (
    "trpc.group/trpc-go/trpc-go/errs"
    // 被调服务的协议生成文件 pb.go 的 git 地址，协议接口管理看这里：todo
    pb "github.com/trpcprotocol/app/server"
)

// SayHello 是 server 请求入口函数，一般的客户端调用都是在一个服务内部再调用下游服务。
// SayHello 携带了 ctx 信息，在该函数内部继续调用下游服务时需要一路透传 ctx。
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    // 创建一个客户端调用代理，该操作很轻量不会创建连接，可以每次请求创建，也可以全局初始化一个 proxy，建议放在 service impl struct 里面，方便 mock 测试，详细 demo 见框架源码 examples/helloworld
    proxy := pb.NewGreeterClientProxy()
    // 正常情况 都不要自己在代码里面指定任何 option 参数，全部使用配置，更灵活，指定 option 的话，以 option 为最高优先级
    reply, err := proxy.SayHi(ctx, req)
    if err != nil {
        log.ErrorContextf(ctx, "say hi fail:%v", err)
        return nil, errs.New(10000, "xxxxx") 
    }
    return &pb.HelloReply{Xxx: reply.Xxx} nil
}

func main(){
    // 创建一个服务对象，底层会自动读取服务配置及初始化插件，必须放在 main 函数首行，业务初始化逻辑必须放在 NewServer 后面
    s := trpc.NewServer()
    // 注册当前实现到服务对象中
    pb.RegisterService(s, &greeterServerImpl{})
    // 启动服务，并阻塞在这里
    if err := s.Serve(); err != nil {
        panic(err)
    }
}
```

### 纯客户端小工具

对于模式二，客户端小工具没有配置文件，需要自己设置 option 发起后端调用，而且也没有 ctx，必须使用 trpc.BackgroundContext()，因为没有配置文件初始化插件，所以一些寻址方式需要自己手动注册，如北极星。代码样例如下：

```go
import (
    "trpc.group/trpc-go/trpc-go/client"
    pb "github.com/trpcprotocol/app/server"
    pselector "trpc.group/trpc-go/trpc-naming-polarismesh/selector" // 需要自己引入需要的名字服务插件代码
    trpc "trpc.group/trpc-go/trpc-go"
)

// 一般小工具都是从 main 函数写起
func main {
    // 由于没有配置文件帮忙初始化插件，所以需要自己手动初始化北极星
    pselector.RegisterDefault()
    // 创建一个客户端调用代理
    proxy := pb.NewGreeterClientProxy() 
    // 必须自己通过 trpc.BackgroundContext() 创建 ctx，通过代码传入 option 参数
    rsp, err := proxy.SayHi(trpc.BackgroundContext(), req, client.WithTarget("ip://ip:port")) 
    if err != nil {
        log.Errorf("say hi fail:%v", err)
        return
    }
    return
}
```

其实大部分的小工具都可以使用定时器模式执行，所以尽量使用 timer 来实现，以模式一的方式来执行工具，所有的服务端功能就能自动配备齐全。

## 接口调用

在客户端，框架为每个服务都定义了一个“ClientProxy”，“ClientProxy”会提供服务调用的桩函数，用户只需要像调用普通函数一样调用桩函数即可。proxy 是一个很轻量级的结构，内部不会创建链接等资源。proxy 的调用是并发安全的，用户可以为每个服务全局初始化一个 proxy，也可以为每次服务调用都生产一个 proxy。

对应不同的协议，它们所提供的服务接口都是不一样的，用户在具体的开发过程中需要参考各自协议的客户端开发文档来做（参考 服务类型 章节）。虽然接口的定义各部相同，但是它们都有共性的部分：“ClientProxy”和“Option 函数选项”。我们基于协议类型和桩代码的生产方式，把它们分成两类：“IDL 类型服务调用”和“非 IDL 类型服务调用”

**1. IDL 类型服务调用**

对于 IDL 类型的服务（比如 tRPC 服务和泛 HTTP RPC 服务），通常使用工具来生成客户端桩函数，生成的代码包括“ClientProxy 创建函数”和“接口调用函数”，函数定义大致如下：

```go
// ClientProxy 的初始化函数
var NewHelloClientProxy = func(opts ...client.Option) HelloClientProxy{...}

// 服务接口定义
type HelloClientProxy interface {
    SayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error)
}
```

桩代码为用户提供了 ClientProxy 的创建函数，服务接口函数，以及对应的参数定义。用户使用这两套函数就可以调用下游服务了。接口的调用是采用同步调用的方式完成的。option 参数可以配置服务调用的行为，具体在后面章节介绍。一次完整的服务调用示例如下：

```go
import (
    "context"

    "trpc.group/trpc-go/trpc-go/client"
    "trpc.group/trpc-go/trpc-go/log"
    pb "github.com/trpcprotocol/test/helloworld"
)

func main() {
    // 创建 ClientProxy
    proxy := pb.NewGreeterClientProxy()
    // 填充请求参数
    req :=  &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    // 调用服务请求接口
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTarget("ip://127.0.0.1:8000"))
    if err != nil {
        return
    }
    // 获取请求响应数据
    log.Debugf("response: %v", rsp)
}
```

**2. 非 IDL 类型服务调用**

对于非 IDL 类型的服务，同样是使用“ClientProxy”来封装服务调用接口的。ClientProxy 创建函数和接口调用函数通常是由协议插件来提供的，不同插件对函数的封装略有不同，开发时需要遵循各自协议的使用文档。以泛 HTTP 标准服务为例，接口定义如下：

```go
// NewClientProxy 新建一个 ClientProrxy, 必传参数 http 服务名
var NewClientProxy = func(name string, opts ...client.Option) Client
// 泛 HTTP 标准服务，提供 get，put，delete，post 四个通用接口
type Client interface {
    Get(ctx context.Context, path string, rspbody interface{}, opts ...client.Option) error
    Post(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
    Put(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
    Delete(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
}
```

一次完整的服务调用示例如下：

```go
import (
    "trpc.group/trpc-go/trpc-go/client"
    "trpc.group/trpc-go/trpc-go/codec"
    "trpc.group/trpc-go/trpc-go/http"
)

func main() {
    // 创建 ClientProxy
    httpCli := http.NewClientProxy("trpc.http.inews_importable",
        client.WithSerializationType(codec.SerializationTypeForm))
    // 填充请求参数
    req := url.Values{} // 设置表单参数
    req.Add("certify", "1")
    req.Add("clientIP", ip)
    // 调用服务请求接口
    rsp: = &A{}
    if err := httpCli.Post(ctx, "/i/getUserUid", req, rsp); err != nil {
        return
    }
    // 获取请求响应数据
    log.Debugf("response: %v", rsp)
}
```

## Option

tRPC-Go 框架提供两级 Option 函数选项来设置 Client 参数，它们分别为“ClientProxy 级配置”和“接口调用级配置”。Option 实现使用的是函数选项设计模式 (Functional Options Pattern)。Option 配置通常用于作为纯客户端的工具上。

```go
// ClientProxy 级 Option 设置，配置对每次使用 clientProxy 调用服务时都生效
clientProxy := NewXxxClientProxy(option1, option2...)
// 接口调用级 Option 设置，配置仅对此次服务调用生效
rsp, err := proxy.SayHello(ctx, req, option1, option2...)
```

对于程序既是服务端也是客户端的场景，系统推荐使用框架配置文件的方式来配置 Client，这样可以实现配置与程序的解耦，方便配置管理。对于 Option 和配置文件组合使用的场景，配置设置的优先级为：`接口调用级 Option` > `ClientProxy 级 Option` > `框架配置文件`。

框架提供了丰富的 Option 参数，本文重点介绍在开发中经常使用的一些配置。

**1. 我们可以通过以下参数来设置服务的协议，序列化类型，压缩方式和服务地址**

```go
proxy.SayHello(ctx, req,
    client.WithProtocol("http"),
    client.WithSerializationType(codec.SerializationTypeJSON),
    client.WithCompressType(codec.CompressTypeGzip),
    client.WithTLS(certFile, keyFile, caFile, serverName),
    client.WithTarget("ip://127.0.0.1:8000"))
```

**2. 我们可以通过以下参数获取下游被调服务的调用地址**

```go
node := &registry.Node{}
proxy.SayHello(ctx, req, client.WithSelectorNode(node))
```

**3. 我们可以通过以下参数设置透传信息**

```go
proxy.SayHello(ctx, req, client.WithMetaData(version.Header, []byte(version.Version)))
```

**4. 我们可以通过以下参数获取下游服务返回的透传信息**

```go
trpcRspHead := &trpc.ResponseProtocol{} // 不同的协议对应不同的 head 结构体
proxy.SayHello(ctx, req, client.WithRspHead(trpcRspHead))
// trpcRspHead.TransInfo 即为下游服务返回的透传信息
```

**5. 我们可以通过以下参数设置服务调用为单向调用**

```go
proxy.SayHello(ctx, req, client.WithSendOnly())
```

## 常用 API

tRPC-Go 采用 GoDoc 来管理 tRPC-Go 框架 API 文档的。通过查阅 [tRPC-Go API 文档](https://pkg.go.dev/github.com/trpc.group/trpc-go) 可以获取 API 的接口规范，参数含义和使用示例。

对于 log，metrics 和 config，框架提供了标准调用接口，服务开发只有使用这些标准接口才能和服务治理系统对接。比如日志，如果不使用标准日志接口，而直接使用“fmt.Printf()”，日志信息是无法上报到远程日志中心的。

## 错误码

tRPC-Go 对错误码的数据类型和含义都做了规划，对于常见错误码的问题定位也都做了解释。具体请参考 [tRPC-Go 错误码手册](/docs/user_guide/error_codes.zh_CN.md)。

# 客户端配置

客户端配置可以通过框架配置文件中的“client”部分来配置，配置分为“全局服务配置”和“指定服务配置”。具体配置的含义，取值范围和默认值请参考 [tRPC-Go 框架配置](/docs/user_guide/framework_conf.zh_CN.md)。

以下是 client 配置的一个典型示例：

```yaml
client:                                    # 客户端调用的后端配置
  timeout: 1000                            # 针对所有后端的请求最长处理时间，单位 ms
  namespace: Development                   # 针对所有后端的环境，正式环境 Production，测试环境 Development
  filter:                                  # 针对所有后端的拦截器配置数组
    - debuglog                             # 强烈建议使用这个debuglog打印日志，非常方便排查问题，具体可以参考：https://github.com/trpc-ecosystem/go-filter/tree/main/debuglog
  service:                                 # 针对单个后端的配置，默认都有默认值，可以完全不用配置
    - callee: trpc.test.helloworld.Greeter # 后端服务协议文件的 service name, 如果 callee 和下面的 name 一样，那只需要配置一个即可
      name: trpc.test.helloworld.Greeter1  # 后端服务名字路由的 service name，有注册到北极星名字服务的话，下面 target 不用配置
      target: ip://127.0.0.1:8000          # 后端服务地址，ip://ip:port polaris://servicename
      network: tcp                         # 后端服务的网络类型 tcp udp, 默认 tcp
      protocol: trpc                       # 应用层协议 trpc http...，默认 trpc
      timeout: 800                         # 当前这个请求最长处理时间，默认 0 不超时
      serialization: 0                     # 序列化方式 0-pb 1-jce 2-json 3-flatbuffer，默认不要配置
      compression: 1                       # 压缩方式 1-gzip 2-snappy 3-zlib，默认不要配置
      filter:                              # 针对单个后端的拦截器配置数组
        - debuglog                         # 只有当前这个后端使用 debuglog
```

需要重点关注的配置项为：

**1. 关于“callee”和“name”的区别：**

“callee”表示下游服务的 Proto Service，格式为：“{package}.{proto service}”。“name”表示下游服务的 Naming Service，用于服务寻址。

按照 tRPC-Go 研发规范 建议的，通常情况“callee”和“name”是一样的，用户可以只配置“name”。对于一个 Proto Service 映射到多个 Naming Service 的场景，用户需要同时设置“callee”和“name”。


**2. 关于"target"的设置：**

tRPC-Go 提供了两套寻址配置：“基于 Naming Service 寻址”和“基于 Target 寻址”。“target”配置可以不配，框架默认使用 name 寻址。当配置了“target”时，框架会基于 Target 寻址。“target”的格式为：`选择器://服务标识`，例如：`ip://127.0.0.1:1000`.

**3. 关于协议的配置**

服务协议相关配置主要包括“network”，“protocol”，“serialization”， “compression”这几个字段。“network”和“protocol”需要以服务端配置为准。

**4. 关于 TLS 的配置**

对于 tRPC 协议，https, http2 和 http3 协议都支持 tls 配置，典型 tls 配置示例如下：

```yaml
client:
  service:                               # 下游服务的 service
    - name: trpc.test.helloworld.Greeter # service 的路由名称
      network: tcp                       # 网络监听类型  tcp udp
      protocol: trpc                     # 应用层协议 trpc http
      timeout: 1000                      # 请求最长处理时间 单位 毫秒
      tls_key: client.pem                # client 秘钥文件地址路径，秘钥文件不要直接提交到 git 上，应该在程序启动时，从配置中心拉取到本地存到该指定路径上
      tls_cert: client.cert              # client 证书文件地址路径
      ca_cert: ca.cert                   # ca 证书文件地址路径，用于校验 server 证书，调用 tls 服务，如 https server
      tls_server_name: xxx               # client 校验 server 服务名，调用 https 时，默认为 hostname
```

对于纯客户端工具，需要通过 option 指定：

```go
proxy.SayHello(ctx, req, client.WithTLS(certFile, keyFile, caFile, serverName))
```

**5. 关于拦截器配置**

框架支持为两级拦截器配置：全局配置和单一服务配置，执行的优先级为：全局设置 > 单一服务配置。如果两者有重复的拦截器，则只执行优先级最高的那个。具体示例如下：

```yaml
client:                                   # 客户端调用的后端配置
  timeout: 1000                           # 针对所有后端的请求最长处理时间，单位 ms
  namespace: Development                  # 针对所有后端的环境，正式环境 Production，测试环境 Development
  filter:                                 # 针对所有后端的拦截器配置数组
    - debuglog                            # debuglog 打印日志
  service:                                # 针对单个后端的配置，默认都有默认值，可以完全不用配置
    - name: trpc.test.helloworld.Greeter1 # 后端服务名字路由的 service name，有注册到北极星名字服务的话，下面 target 不用配置
      network: tcp                        # 后端服务的网络类型 tcp udp, 默认 tcp
      protocol: trpc                      # 应用层协议 trpc http tars oidb ...，默认 trpc
      timeout: 800                        # 当前这个请求最长处理时间，默认 0 不超时
      filter:                             # 针对单个后端的拦截器配置数组
        - debuglog
```

# 服务寻址

服务寻址是服务调用中非常重要的环节，框架通过插件的方式来实现服务发现，策略路由，负载均衡和熔断器，框架不包括任何具体实现，用户可根据需要引入相应的插件。服务寻址的很多功能都是和名字服务提供的功能密切相关的，用户需要结合名字服务文档和对应插件文档来获取功能详情。本节后续的描述均已北极星插件为例。

## 命名空间与环境

框架通过命名空间（namespace）和环境（env_name）两个概念来实现服务调用的隔离。namespace 通常用于区分生产环境和非生产环境，两个 namespace 的服务是完全隔离的。env_name 只用于非生产环境，通过 env_name 为用户提供个人测试环境。

系统建议通过框架配置文件来设置客户端的 namespace 和 env_name, 在服务调用时默认使用客户端的 namespace 和 env_name。

```yaml
global:
  # 必填，通常使用 Production 或 Development
  namespace: String
  # 选填，环境名称
  env_name: String
```

框架也支持在服务调用时，指定服务的 namespace 和 env_name，我们把它称为指定环境服务调用。指定环境服务调用需要关闭服务路由功能（系统默认是打开的）。可以通过 Option 函数来设置：

```go
opts := []client.Option{
    // 命名空间，不填写默认使用本服务所在环境的 namespace
    client.WithNamespace("Development"),
    // 服务名
    client.WithServiceName("trpc.test.helloworld.Greeter"),
    // 设置被调服务环境
    client.WithCalleeEnvName("62a30eec"),
    // 关闭服务路由
    client.WithDisableServiceRouter()
}
```

也可以通过框架配置文件来设置：

```yaml
client:                                   # 客户端调用的后端配置
  namespace: Development                  # 针对所有后端的环境
  service:                                # 针对单个后端的配置
    - name: trpc.test.helloworld.Greeter1 # 后端服务名字路由的 service name
      disable_servicerouter: true         # 单个 client 是否禁用服务路由
      env_name: eef23fdab                 # 设置下游服务多环境的环境名，需要 disable_servicerouter 为 true 才生效
      namespace: Development              # 对端服务环境
```

## 寻址方式

框架提供了两套寻址配置：“基于 Naming Service 寻址”和“基于 Target 寻址”。可以通过 Option 函数选项来设置，系统默认和推荐使用“基于 Naming Service 寻址”，基于 Naming Service 寻址的 Option 函数定义和示例为：

### 基于 Namine Service 寻址

```go
// 基于 Naming Service 寻址接口定义
func WithServiceName(s string) Option

// 示例代码
func main() {
    opts := []client.Option{
        client.WithServiceName("trpc.app.server.service"),
    }
    rsp, err := clientProxy.SayHello(ctx, req, opts...)
}
```

### 基于 Target 寻址

使用基于 Target 寻址的 Option 函数定义和示例为：

```go
// 基于 Target 寻址接口定义，target 格式：选择器://服务标识
func WithTarget(t string) Option

// 示例代码
func main() {
    opts := []client.Option{
        client.WithNamespace("Development"),
        client.WithTarget("ip://127.0.0.1:8000"),
    }
    rsp, err := clientProxy.SayHello(ctx, req, opts...)
}
```

“ip”和“dns”在工具类型的客户端中使用比较常见的选择器，target 的格式为：`ip://ip1:port1,ip2:port2`，支持 ip 列表。IP 选择器会在 IP 列表中随机选择一个 IP 用于服务调用。IP 和 DNS 选择器不依赖外部名字服务。

#### `ip://<ip>:<port>`

指定直连 ip 寻址，如 ip://127.1.1.1:8080，也可以设置多个ip，格式为 ip://ip1:port1,ip2:port2

#### `dns://<domain>:<port>`

指定域名寻址，常用于 http 请求，如 dns://www.qq.com:80

## 插件设计

服务寻址包括服务发现、负载均衡、服务路由、熔断器等部分，服务发现的流程可以简化为：

![server_discovery](/.resources/user_guide/client/overview/server_discovery_zh_CN.png)

框架通过“selector”来组合这四个模块，并提供了两种插件方式来实现服务寻址：

- 整体接口：名字服务作为整体注册到框架，作为一个 selector 插件。整体接口的优势在于注册到框架比较简单，框架不关心名字服务流程中各个模块的具体实现，插件可以整体控制名字服务寻址的整个流程，方便做性能优化和逻辑控制。
- 分模块接口：使用框架默认提供的 selector，服务发现、负载均衡、服务路由、熔断器等分别注册到框架，框架组合这些模块。分模块优势在于更加的灵活，用户可以根据自己的需要对不同模块进行选择然后自由组合，但同时会增加插件的实现复杂度。

框架支持用户开发新的名字服务插件。名字服务插件的开发请参考 [tRPC-Go 开发名字服务插件](/docs/developer_guide/develop_plugins/naming.zh_CN.md)。

# 插件选择

对于插件的使用，我们需要同时“在 main 文件中 import 插件”和“在框架配置文件中配置插件”的方式来引入插件。如何使用插件请参考 [北极星名字服务](https://trpc.group/trpc-go/trpc-naming-polarismesh) 中的示例。

tRPC 插件生态提供了丰富的插件，程序如何选择合适的插件呢？这里我们提供了一些思路供大家参考。我们可以把插件可以大致分成三类：独立插件，服务治理插件 和 存储接口插件。

- 独立插件：比如协议，压缩，序列化，本地内存缓存等插件，其插件的运行不依赖外部系统组件。这类插件的思路比较简单，主要是依据业务功能的需要，和插件的成熟度来做选择
- 服务治理插件：绝大部分服务治理插件，比如远程日志，名字服务，配置中心等，它们都需要和外部系统对接，对于微服务治理体系有很大的依赖。对这类插件的选择，我们需要明确服务最终运行在什么运营平台上，平台提供了哪些治理组件，服务有哪些能力一定要和平台对接，哪些则不需要。
- 存储接口插件：存储插件主要封装了业界和公司内部成熟数据库，消息队列等组件的接口调用。关于这部分插件，我们首先需要考虑业务的技术选型，什么样的数据库更适合业务的需求。然后基于技术选型来看 tRPC 是否支持，如果不支持，我们可以选择使用数据库原生 SDK，或者建议大家贡献插件到 tRPC 社区

# 拦截器

tRPC-Go 提供了拦截器（filter）机制，拦截器在服务请求和响应的上下文设置埋点，允许业务在埋点处插入自定义处理逻辑。tRPC-Go [插件生态](https://github.com/trpc-ecosystem) 提供了丰富的拦截器，其中 调用链，监控插件也都是通过拦截器来实现的。

关于拦截器的原理，触发时机，执行顺序和自定义拦截器的示例代码，请参考 [tRPC-Go 开发拦截器插件](/filter)。

# 调用场景

对于程序作为纯客户端的场景，服务调用的方式比较简单，通常采用同步调用方式直接等待调用返回，或者创建一个 goroutine 并在 goroutine 中同步调用等待返回结果，这里不做赘述。

对于程序既做服务端又做客户端的场景（服务在收到上游请求时，需要调用下游服务）会相对复杂点，本文按照同步处理，异步处理，多并发处理三种方式来给用户开发提供思路。

## 同步处理

同步处理的典型场景：一个服务当它收到上游的服务请求时，需要调用下游服务并等待下游服务调用完成后再给上游回包。

对于同步处理，程序对下游服务调用可以使用请求服务的 ctx，支持包括 ctx 日志，全链路超时控制等功能。代码示例为：

```go
func (s *serverImpl) Call(ctx context.Context, req *pb.Req) (*pb.Rsp, error) {
    ....

    // 同步处理后续服务调用，可以使用服务请求里的 ctx
    proxy := redis.NewClientProxy("trpc.redis.test.service") // proxy 不要每次创建，这里只是演示
    val1, err := redis.String(proxy.Do(ctx, "GET", "key1")) 
    ....
}
```

## 异步处理

异步处理的典型场景：一个服务当它收到上游的服务请求，需要提前给上游回包，然后再慢慢处理下游的服务调用。

对于异步处理，程序可以启一个 goroutine 执行后续服务调用，但是后续服务调用不能使用原服务请求的 ctx，因为原 ctx 完成回包后会自动取消。后续服务调用可以使用 trpc.BackgroundContext() 创建一个新的 ctx，也可以直接使用 trpc 提供的 trpc.Go 工具函数：

```go
func (s *serverImpl) Call(ctx context.Context, req *pb.Req) (*pb.Rsp, error) {
    ....

    trpc.Go(ctx, time.Minute, func(ctx context.Context) {  // 这里可以直接传入请求入口的 ctx，trpc.Go 里面会先 clone context 再 go and recover，内部会包含日志，监控，recover，超时控制
        proxy := redis.NewClientProxy("trpc.redis.test.service")  // proxy 不要每次创建，这里只是演示
        val1, err := redis.String(proxy.Do(ctx, "GET", "key1")) 
    })

    // 不用等待下游响应，直接回包。ctx 在完成回包后会自动 cancel
    ....
}
```

## 多并发处理

多并发调用的典型场景：一个上线服务，当它收到上游的服务请求时，需要同时调用多个下游服务，并等待所有下游服务的响应。

这种场景，业务可以自己启动多个 goroutine 来发起请求，但是这样比较麻烦，需要自己 waitgroup，recover，如果没有 recover，自己启动的 goroutine 很容易导致服务 crash，框架封装了一个简单的多并发函数 GoAndWait() 供用户使用。

```go
// GoAndWait 封装更安全的多并发调用，启动 goroutine 并等待所有处理流程完成，自动 recover
// 返回值 error: 返回的是多并发协程里面第一个返回的不为 nil 的 error
func GoAndWait(handlers ...func() error) error
```

示例：假设服务收到 Call() 请求后，服务需要向两个后端服务 redis 获取 key1，key2 的值，只有完成下游服务调用后，才会返回响应给上游。

```go
func (s *serverImpl) Call(ctx context.Context, req *pb.Req) (*pb.Rsp, error) {
    var value [2]string
    proxy := redis.NewClientProxy("trpc.redis.test.service")
    if err := trpc.GoAndWait(
        func() error {
            // 假设第一个下游服务调用是从 redis 获取 key1 的值，由于 GoAndWait 会等待所有 goroutine 都完成才会退出，ctx 不会取消，所以这里可以使用请求入口的 ctx，若要拷贝新的 ctx，可以在 GoAndWait 前面使用`newCtx := trpc.CloneContext(ctx)`
            val1, err := redis.String(proxy.Do(ctx, "GET", "key1"))
            if err != nil {
                // key1 不是关键数据，失败了也无所谓，可以兜底一个假数据并返回成功
                value[0] = "fake1"
                return nil
            }
            log.DebugContextf(ctx, "get key1, val1:%s", val1)
            value[0] = val1
            return nil
        },
        func() error {
            // 假设第二个下游服务调用是从 redis 获取 key2 的值
            val2, err := redis.String(proxy.Do(ctx, "GET", "key2"))
            if err != nil {
                // key2 是关键数据，获取不到需要提前终止逻辑，所以这里返回失败
                return errs.New(10000, "get key2 fail: "+err.Error())
            }
            log.DebugContextf(ctx, "get key2, val2:%s", val2)
            
            value[1] = val2
            return nil
        },
    );     err != nil { // 多并发请求有失败，返回错误码给上游服务
        return nil, err
    }
    // ...
}
```

# 高级功能

## 超时控制

tRPC-Go 框架为服务的调用提供了调用超时机制。关于调用超时机制的介绍和相关配置，请参考 [tRPC-Go 超时控制](/docs/user_guide/timeout_control.zh_CN.md) 。

## 链路透传

tRPC-Go 框架提供在客户端与服务端之间透传字段，并在整个调用链路透传下去的机制。关于链路透传的机制和使用，请参考 [tRPC-Go 链路透传](/docs/user_guide/metadata_transmission.zh_CN.md)。此功能需要协议支持元数据下发功能，tRPC 协议，泛 HTTP RPC 协议，taf 协议均支持链路透传功能。其它协议请联系各自协议负责人。

## 自定义压缩

tRPC-Go 框架支持业务自己定义压缩、解压缩方式。具体请参考 [这里](/codec/compress_gzip.go)。

## 自定义序列化

tRPC-Go 框架业务自己定义序列化、反序列化类型。具体示例请参考 [这里](/codec/serialization_json.go)。
