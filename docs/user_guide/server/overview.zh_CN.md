[English](./overview.md) | 中文

tRPC-Go 服务端开发向导

# 前言

本文梳理了开发一个服务端程序需要考虑的问题，如：

- 服务需要使用什么协议？
- 如何定义服务？
- 如何选择插件？
- 如何测试？

# 服务选型

## 内置协议服务

tRPC 框架内置支持 **tRPC 服务**，**tRPC 流式服务**，**泛 HTTP RPC 服务** 和 **泛 HTTP 标准服务**。“泛 HTTP”特指服务的底层协议采用“http”， “https”， “http2”和“http3”这四种协议。

- **泛 HTTP 标准服务**和**泛 HTTP RPC 服务** 有什么区别？泛 HTTP RPC 服务是 tRPC 框架自行设计的一套基于泛 HTTP 协议的 rpc 模型，其协议细节都已在框架内部封装，对用户来完全透明。泛 HTTP RPC 服务通过 PB IDL 协议来定义服务，由脚手架自动生成 rpc 桩代码。泛 HTTP 标准服务在使用上跟 golang http 标准库一模一样，由用户自行定义 handle 请求函数，自行注册 http 路由，自行填充 http head 等。标准 http 服务不需要 IDL 协议文件。
- **泛 HTTP RPC 服务**和 **tRPC 服务**有什么区别？泛 HTTP RPC 服务和 tRPC 服务唯一的区别在于协议上的不同，泛 HTTP RPC 服务使用泛 http 协议，tRPC 服务使用 tRPC 私有协议。区别仅仅在框架内部可见，在业务开发使用上几乎没有区别。
- **tRPC 服务**与 **tRPC 流式服务**有什么区别？tRPC 服务单次 RPC 调用需要客户端发起请求，等服务端处理完毕再返回给客户端。而 tRPC 流式服务在建立流连接之后，可支持客户端不断发送数据，服务端不断接收数据，持续进行响应。两种服务在协议格式、IDL 语法上都有所不同。

## 定时任务服务

定时任务服务采用了普通服务模型，提供定时任务能力。比如程序需要定时加载 cache, 定时校验流水等场景。一个定时任务服务只支持一个定时任务，如果有多个定时任务，那么就需要创建多个定时任务服务。定时器任务服务的功能描述，请参考 [tRPC-Go 搭建定时器服务](https://github.com/trpc-ecosystem/go-database/tree/main/timer)。

定时任务服务并不是 RPC 服务，它不对客户端提供服务调用。定时任务服务和 RPC 服务互不影响，一个应用程序可同时存在多个 RPC 服务和多个定时任务服务。

## 消费者服务

消费者服务的使用场景是：程序作为消费者来消费消息队列中的消息。和定时任务服务一样，采用了普通服务模型，复用框架的服务治理能力，如自动上报监控，模调，调用链等功能。服务并不对客户端提供服务调用。

目前 tRPC-Go 支持的消息队列包括：[kafka](https://github.com/trpc-ecosystem/go-database/tree/main/kafka) 等。各种消息队列的开发和配置略有区别，请参考各自文档。

# 定义 Naming Service

选择服务的协议之后，我们就需要定义 **Naming Service** 了，也就是确定提供服务的地址是什么，在名字系统用来寻址的路由标识是什么。

Naming Service 负责网络通信和协议解析。一个 naming service 在寻址上最终用来代表一个 `[ip，port，protocol]` 组合。Naming Service 是通过框架配置文件中的“server”部分的“service”配置来定义。

我们通常使用一个字符串来表示一个 Naming Service。其命名格式取决于服务所在的服务管理平台是如何定义服务模型的，本文以常用做法 `trpc.{app}.{server}.{service}` 四段式为例。

```yaml
server:  # 服务端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter1  # service 的路由名称，这里是一个数组，注意：name 前面的减号
      ip: 127.0.0.1  # 服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8000  # 服务监听端口
      network: tcp  # 网络监听类型  tcp udp
      protocol: trpc  # 应用层协议 trpc http
      timeout: 1000  # 请求最长处理时间 单位 毫秒
```

在这个示例中，服务的路由标识是“trpc.test.helloworld.Greeter1”，协议类型为“trpc”，地址为“127.0.0.1:8000”。程序在启动时会自动读取这个配置，并生成 Naming Service。如果服务端选择了“服务注册”插件，则应用程序会自动注册 Naming Service 的“name”和“ipport”等信息到名字服务，这样客户端就可以使用这个名字进行寻址了。

# 定义 Proto Service

Proto Service 是一组接口的逻辑组合，它需要定义 package，proto service，rpc name 以及接口请求和响应的数据类型。同时还需要把 Proto Service 和 Naming Service 进行组合，完成服务的组装。对于服务的组装，虽然“IDL 协议类型”和“非 IDL 协议类型”提供给开发者的注册接口略有区别，但框架内部对两者实现是一致的。

## IDL 协议类型

IDL 语言可以通过一种中立的方式来描述接口，并使用工具把 IDL 文件转换成指定语言的桩代码，使程序员专注于业务逻辑开发。tRPC 服务，tRPC 流式服务和泛 HTTP RPC 服务都是 IDL 协议类型服务。对于 IDL 协议类型的服务，Proto Service 的定义通常分为以下三步：

**以下示例均以 tRPC 服务为例**

第一步：采用了 IDL 语言描述 RPC 接口规范，生成 IDL 文件。以 tRPC 服务为例，其 IDL 文件的定义如下：

```protobuf
syntax = "proto3";

package trpc.test.helloworld;
option go_package="github.com/some-repo/examples/helloworld";

service Greeter {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
    string msg = 1;
}

message HelloReply {
    string msg = 1;
}
```

第二步：通过 [trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline) 工具可以生成对应服务端和客户端的桩代码

```shell
trpc create -p helloworld.proto
```

第三步：就把 Proto Service 注册到 Naming Service 上，完成服务的组装。

```go
type greeterServerImpl struct{}

// 接口处理函数
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{ Msg: "Hello, I am tRPC-Go server." }, nil
}

func main() {
    // 通过读取框架配置中的 server.service 配置项，创建 Naming Service
    s := trpc.NewServer()
    // 注册 Proto Service 的实现实例到 Naming Service 中
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    // ...
}
```

对于程序只有一个 Proto Service 和 Naming Service 时，可以直接使用 `trpc.NewServer()` 生成的 server 来和 Proto Service 映射。

## 非 IDL 协议类型

对于非 IDL 协议类型，同样需要有 Proto Service 的定义和注册过程。通常框架和插件对各协议有不同程度的封装，开发时需要遵循各自协议的使用文档。以泛 HTTP 标准服务为例，其代码如下：

```go
// 接口处理函数
func handle(w http.ResponseWriter, r *http.Request) error {
    // 构建 Http Head
    w.WriteHeader(403)
    // 构建 Http Body
    w.Write([]byte("response body"))

    return nil
}

func main() {
    // 通过读取框架配置中的 server.service 配置项，创建 Naming Service
    s := trpc.NewServer()

    thttp.HandleFunc("/xxx/xxx", handle)
    // 注册 Proto Service 的实现实例到 Naming Service 中
    thttp.RegisterNoProtocolService(s)
    s.Serve()
}
```

## 多服务注册

对于程序不是单服务模式时（只有一个 naming service 和一个 proto service），用户需要明确指定 naming service 和 proto service 的映射关系。

对于多服务的注册，我们以 tRPC 服务为例定义两个 Proto Service：`trpc.test.helloworld.Greeter` 和 `trpc.test.helloworld.Hello`：

```protobuf
syntax = "proto3";
package trpc.test.helloworld;
option go_package="github.com/some-repo/examples/helloworld";
service Greeter {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

service Hello {
    rpc SayHi (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
    string msg = 1;
}

message HelloReply {
    string msg = 1;
}
```

与之对应也需要定义两个 Naming Service：`trpc.test.helloworld.Greeter` 和 `trpc.test.helloworld.Hello`：

``` yaml
server:  # 服务端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter  # service 的路由名称，这里是一个数组，注意：name 前面的减号
      ip: 127.0.0.1  # 服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8000  # 服务监听端口
      network: tcp  # 网络监听类型  tcp udp
      protocol: trpc  # 应用层协议 trpc http
      timeout: 1000  # 请求最长处理时间 单位 毫秒
    - name: trpc.test.helloworld.Hello  # service 的路由名称，这里是一个数组，注意：name 前面的减号
      ip: 127.0.0.1  # 服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8001  # 服务监听端口
      network: tcp  # 网络监听类型  tcp udp
      protocol: trpc  # 应用层协议 trpc http
      timeout: 1000  # 请求最长处理时间 单位 毫秒
```

把 Proto Service 注册到 Naming Service，多服务场景需要指定 Naming Service 的名称。

```go
func main() {
    // 通过读取框架配置中的 server.service 配置项，创建 Naming Service
    s := trpc.NewServer()
    // 注册 Greeter 服务
    pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &greeterServerImpl{})
    // 注册 Hello 服务
    pb.RegisterHelloService(s.Service("trpc.test.helloworld.Hello"), &helloServerImpl{})
    ...
}
```

## 接口管理

对于框架内置的 tRPC 服务，tRPC 流式服务和泛 HTTP RPC 服务，建议遵守一定的研发规范。

这三类服务均采用 PB 文件来定义接口。为了方便上下游能更透明的获知接口信息，我们建议使用 **pb 文件与服务分离，与语言无关，通过独立的中心仓库进行统一版本管理** 的思路，通过一个公共平台来管理 PB 文件。

# 服务开发

常见服务类型的搭建，请参考如下链接：

- [搭建 tRPC 服务](todo)
- [搭建 tRPC 流式服务](todo)
- [搭建泛 HTTP RPC 服务](todo)
- [搭建泛 HTTP 标准服务](todo)
- [搭建 gRPC 服务](todo)
- [搭建定时器服务](todo)
- [搭建消费者服务](todo)

一些第三方协议插件见：[trpc-ecosystem/go-codec](https://github.com/trpc-ecosystem/go-codec)。

## 常用 API

对于 log，metrics 和 config，框架提供了标准调用接口，服务开发只有使用这些标准接口才能和服务治理系统对接。比如日志，如果不使用标准日志接口，而直接使用“fmt.Printf()”，日志信息是无法上报到远程日志中心的。

tRPC-Go 服务端配置支持“**通过框架配置文件**”和“**函数调用传参**”两种方式来配置服务。“函数调用传参”的优先级要大于“通过框架配置文件”的设置。建议优先使用框架配置文件来配置服务端，其好处是配置和代码解耦，方便管理。

## 错误码

tRPC-Go 推荐在写服务端业务逻辑时，使用 tRPC-Go 封装的 `errors.New()` 来返回业务错误码，这样框架能自动上报业务错误码到监控系统。如果业务自定义 error 的话，就只能靠业务主动调用 Metrics SDK 来上报错误码。关于错误码的 API 使用，请参考 [这里](/errs)。

# 框架配置

对于服务端，必须要配置框架配置中“global”，“server”两部分的配置，配置参数的具体含义，取值范围等信息请参考 [框架配置](/docs/user_guide/framework_conf.zh_CN.md) 文档。“plugins”部分的配置取决于所选的插件，具体参考下面的插件选择章节。

# 插件选择

tRPC 框架的核心在于把框架功能插件化，框架核心并不包括具体的实现。对于插件的使用，我们需要同时“**在 main 文件中 import 插件**”和“**在框架配置文件中配置插件**”的方式来引入插件，这里需要强调的是 **插件的选择必须要在开发阶段确定**。如何使用插件请参考 [北极星名字服务](https://github.com/trpc-ecosystem/go-naming-polarismesh) 中的示例。

tRPC 插件生态提供了丰富的插件，程序如何选择合适的插件呢？这里我们提供了一些思路供大家参考。我们可以把插件可以大致分成三类：独立插件，服务治理插件 和 存储接口插件。

- 独立插件：比如协议，压缩，序列化，本地内存缓存等插件，其插件的运行不依赖外部系统组件。这类插件的思路比较简单，主要是依据业务功能的需要，和插件的成熟度来做选择。
- 服务治理插件：绝大部分服务治理插件，比如远程日志，名字服务，配置中心等，它们都需要和外部系统对接，对于微服务治理体系有很大的依赖。对这类插件的选择，我们需要明确服务最终运行在什么运营平台上，平台提供了哪些治理组件，服务有哪些能力一定要和平台对接，哪些则不需要。
- 存储接口插件：存储插件主要封装了业界和公司内部成熟数据库，消息队列等组件的接口调用。关于这部分插件，我们首先需要考虑业务的技术选型，什么样的数据库更适合业务的需求。然后基于技术选型来看 tRPC 是否支持，如果不支持，我们可以选择使用数据库原生 SDK，或者建议大家贡献插件到 tRPC 社区。

## 内置插件

框架为服务内置了一些必要的插件，这样可以确保用户在不设置任何插件的情况下，框架仍然能够使用默认插件提供正常的 RPC 调用能力。用户可以自行替换默认插件。

下面表格列出了作为服务端时框架提供的默认插件，以及插件的默认行为。

| 插件类型  | 插件名称   | 默认插件 | 插件行为  |
| ---------- | --------- | --------  | ------------------------------------- |
| log      | Console  | 是      | 默认 debug 级别以上日志打 console，级别可通过配置或者 API 可设置   |
| metric   | Noop     | 是      | 不上报 metric 信息     |
| config   | File     | 是      | 支持用户使用接口从指定本地文件获取配置项   |
| registry | Noop     | 是      | 不做服务的注册和注销   |

# 拦截器

tRPC-Go 提供了拦截器（filter）机制，拦截器在 RPC 请求和响应的上下文设置埋点，允许业务在埋点处插入自定义处理逻辑。像调用链跟踪和认证鉴权等功能通常是采用拦截器来实现的。常用拦截器请在 [trpc-ecosystem/go-filter](https://github.com/trpc-ecosystem/go-filter) 中查找。

业务可以自定义拦截器。拦截器通常和插件组合来实现功能的，插件提供配置，而拦截器用于在 RPC 调用上下文插入处理逻辑。关于拦截器的原理，触发时机，执行顺序和自定义拦截器的示例代码，请参考 [trpc-go/filter](/filter)。

# 测试相关

tRPC-Go 从设计之初就考虑了框架的易测性，在通过 pb 生成桩代码时，默认会生成 mock 代码。

# 高级功能

## 超时控制

tRPC-Go 为 RPC 调用提供了 3 种超时机制控制：链路超时，消息超时和调用超时。关于这 3 种超时机制的原理介绍和相关配置，请参考 [tRPC-Go 超时控制](/docs/user_guide/timeout_control.zh_CN.md)。

此功能需要协议的支持（协议需要携带 timeout 元数据到下游），tRPC 协议，泛 HTTP RPC 协议均支持超时控制功能。其

## 链路透传

tRPC-Go 框架提供在客户端与服务端之间透传字段，并在整个调用链路透传下去的机制。关于链路透传的机制和使用，请参考 [tRPC-Go 链路透传](/docs/user_guide/metadata_transmission.zh_CN.md)。

此功能需要协议支持元数据下发功能，tRPC 协议，泛 HTTP RPC 协议均支持链路透传功能。

## 反向代理

tRPC-Go 为类似做反向代理的程序提供了完成透传二进制 body 数据，不进行序列化、反序列化处理的机制，以提升转发效率。关于反向代理的原理和示例程序，请参考 [tRPC-Go 反向代理](/docs/user_guide/reverse_proxy.zh_CN.md)。

## 自定义压缩方式

tRPC-Go 自定义 RPC 消息体的压缩、解压缩方式，业务可以自己定义并注册压缩、解压缩算法。具体示例请参考 [这里](/codec/compress_gzip.go)。

## 自定义序列化方式

tRPC-Go 自定义 RPC 消息体的序列化、反序列化方式，业务可以自己定义并注册序列化、反序列化算法。具体示例请参考 [这里](/codec/serialization_json.go)。

## 设置服务最大协程数

tRPC-Go 支持服务级别的同/异步包处理模式，对于异步模式采用协程池来提升协程使用效率和性能。用户可以通过框架配置和 Option 配置两种方式来设置服务的最大协程数，具体请参考 [tPRC-Go 框架配置](/docs/user_guide/framework_conf.zh_CN.md) 章节的 service 配置。
