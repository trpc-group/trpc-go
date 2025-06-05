# 1. 前言

通过前面的 [快速上手](https://iwiki.woa.com/pages/viewpage.action?pageId=118272478 "快速上手")，我们已经知道如何开发一个简单的 tRPC 服务。下面我们会带着大家重新梳理一下，开发一个服务端程序需要考虑哪些问题，做哪些事情，以及在开发中会涉及到哪些开发的话题。

本文按照 **协议选择、服务定义、业务开发、插件/拦截器选择、测试手段** 这条开发路线来展开，尝试和大家一起思考以下问题：

- 服务需要使用什么协议？
- 如何定义服务？
- 如何选择插件？
- 如何测试？

# 2. 服务选型

## 2.1 内置协议服务

tRPC 框架内置支持 **tRPC 服务**，**tRPC 流式服务**，**泛 HTTP RPC 服务** 和 **泛 HTTP 标准服务**。“泛 HTTP”特指服务的底层协议采用“http”、“https”、“http2”和“http3”这四种协议。

- **泛 HTTP 标准服务和泛 HTTP RPC 服务有什么区别？** 泛 HTTP RPC 服务是 tRPC 框架自行设计的一套基于泛 HTTP 协议的 rpc 模型，其协议细节都已在框架内部封装，对用户来完全透明。泛 HTTP RPC 服务通过 PB IDL 协议来定义服务，由脚手架自动生成 rpc 桩代码。泛 HTTP 标准服务在使用上跟 golang http 标准库一模一样，由用户自行定义 handle 请求函数，自行注册 http 路由，自行填充 http head 等。标准 http 服务不需要 IDL 协议文件。

- **泛 HTTP RPC 服务和 tRPC 服务有什么区别？**  泛 HTTP RPC 服务和 tRPC 服务唯一的区别在于协议上的不同，泛 HTTP RPC 服务使用泛 http 协议，tRPC 服务使用 trpc 私有协议。区别仅仅在框架内部可见，在业务开发使用上几乎没有区别。

- **tRPC 服务与 tRPC 流式服务有什么区别？** tRPC 服务单次 RPC 调用需要客户端发起请求，等服务端处理完毕再返回给客户端。而 tRPC 流式服务在建立流连接之后，可支持客户端不断发送数据，服务端不断接收数据，持续进行响应。两种服务在协议格式、IDL 语法上都有所不同。tRPC 流式服务的使用场景请参考 [tRPC 协议](https://iwiki.woa.com/pages/viewpage.action?pageId=145446228 "tRPC 协议")。

## 2.2 第三方协议服务

有时候为了和存量系统对接，服务需要提供老系统的协议类型。tRPC 插件生态提供了大量存量系统的协议插件。请在 [插件生态](https://iwiki.woa.com/pages/viewpage.action?pageId=447434212) 章节查找。

框架支持自行实现第三方协议，对于第三方协议的开发请参考 [协议开发](https://iwiki.woa.com/pages/viewpage.action?pageId=99485626) 章节。

## 2.3 定时任务服务

定时任务服务采用了普通服务模型，提供定时任务能力。比如程序需要定时加载 cache, 定时校验流水等场景。一个定时任务服务只支持一个定时任务，如果有多个定时任务，那么就需要创建多个定时任务服务。定时器任务服务的功能描述，请参考 [tRPC-Go 搭建定时器服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289170)。

定时任务服务并不是 RPC 服务，它不对客户端提供服务调用。定时任务服务和 RPC 服务互不影响，一个应用程序可同时存在多个 RPC 服务和多个定时任务服务。

## 2.4 消费者服务

消费者服务的使用场景是：程序作为消费者来消费消息队列中的消息。和定时任务服务一样，采用了普通服务模型，复用框架的服务治理能力，如自动上报监控，模调，调用链等功能。服务并不对客户端提供服务调用。

目前 tRPC-Go 支持的消息队列包括：[kafka](https://git.woa.com/tRPC-Go/trpc-database/tree/master/kafka "kafka"), [rabbitmq](https://git.woa.com/tRPC-Go/trpc-database/tree/master/rabbitmq "rabbitmq"), [rocketmq](https://git.woa.com/tRPC-Go/trpc-database/tree/master/rocketmq "rocketmq"), [tdmq](https://git.woa.com/tRPC-Go/trpc-database/tree/master/tdmq "tdmq"), [tube](https://git.woa.com/tRPC-Go/trpc-database/tree/master/tube "tube"), [redis](https://git.woa.com/pcg-csd/trpc-ext/redis/tree/master/trpc/mq "redis"), [hippo](https://git.woa.com/tRPC-Go/trpc-database/tree/master/hippo "hippo") 等。各种消息队列的开发和配置略有区别，请参考各自文档。

# 3. 定义 Naming Service

选择服务的协议之后，我们就需要定义 **Naming Service** 了，也就是确定提供服务的地址是什么，在名字系统中用来寻址的路由标识是什么。Naming Service 的定义请参考 [tRPC 术语介绍](https://iwiki.woa.com/pages/viewpage.action?pageId=490794774 "术语介绍")。

Naming Service 负责网络通信和协议解析。一个 Naming Service 在寻址上最终用来代表一个 `[ip，port，protocol]` 组合。Naming Service 是通过框架配置文件中的 `server` 部分的 `service` 配置来定义。

我们通常使用一个字符串来表示一个 Naming Service。其命名格式取决于服务所在的服务管理平台是如何定义服务模型的，本文所有示例均使用了 PCG 123 平台定义的 `trpc.{app}.{server}.{service}` 的四段式来命名。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter1          # service 的路由名称，这里是一个数组，注意：name 前面的减号
      ip: 127.0.0.1                                # 服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口
      network: tcp                                 # 网络监听类型  tcp udp unix
      protocol: trpc                               # 应用层协议 trpc http
      transport: tnet                              # 要求框架版本 >= 0.11.0，为 tcp trpc 启用 tnet，其他协议可以自行验证
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
```

> 注：`network` 字段填写 udp/unix 时可以自动以 udp 或 unix domain socket 的形式作为网络类型。

在这个示例中，服务的路由标识是 `trpc.test.helloworld.Greeter1`，协议类型为 `trpc`，地址为 `127.0.0.1:8000`。程序在启动时会自动读取这个配置，并生成 Naming Service。如果服务端选择了 **服务注册** 插件，则应用程序会自动注册 Naming Service 的 `name` 和 `ip:port` 等信息到名字服务，这样客户端就可以使用这个名字进行寻址了。

# 4. 定义 Proto Service

Proto Service 是一组接口的逻辑组合，它需要定义 package，proto service，rpc name 以及接口请求和响应的数据类型。同时还需要把 Proto Service 和 Naming Service 进行组合，完成服务的组装。关于 Proto Service 与 Naming Service 之间的关系请参考 [tRPC 术语介绍](https://iwiki.woa.com/pages/viewpage.action?pageId=490794774 "术语介绍")。对于服务的组装，虽然 **IDL 协议类型** 和 **非 IDL 协议类型** 提供给开发者的注册接口略有区别，但框架内部对两者实现是一致的。

## 4.1 IDL 协议类型

IDL 语言可以通过一种中立的方式来描述接口，并使用工具把 IDL 文件转换成指定语言的桩代码，使程序员专注于业务逻辑开发。tRPC 服务、tRPC 流式服务和泛 HTTP RPC 服务都是 IDL 协议类型服务。对于 IDL 协议类型的服务，Proto Service 的定义通常分为以下三步：

> 注：以下示例均以 tRPC 服务为例。

第一步：采用 IDL 语言描述 RPC 接口规范，生成 IDL 文件。以 tRPC 服务为例，其 IDL 文件的定义如下：

```protobuf
syntax = "proto3";

package trpc.test.helloworld;
option go_package="git.code.oa.com/trpcprotocol/test/helloworld";

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

第二步：通过开发工具可以生成对应服务端和客户端的桩代码。

```shell
trpc create --protofile=helloworld.proto
```

第三步：把 Proto Service 注册到 Naming Service 上，完成服务的组装。

```go
type greeterServerImpl struct{}

// 接口处理函数
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    rsp.Msg = "Hello, I am tRPC-Go server."
    return nil
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

## 4.2 非 IDL 协议类型

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
    thttp.RegisterDefaultService(s)
    s.Serve()
}
```

## 4.3 多服务注册

对于程序不是单服务模式时（只有一个 Naming Service 和一个 Proto Service），用户需要明确指定 Naming Service 和 Proto Service 的映射关系。关于两者映射关系的介绍请参考 [tRPC 术语介绍](https://iwiki.woa.com/pages/viewpage.action?pageId=490794774 "tRPC 术语介绍") 章节。

对于多服务的注册，我们以 tRPC 服务为例定义两个 Proto Service：`trpc.test.helloworld.Greeter` 和 `trpc.test.helloworld.Hello`：

```protobuf
syntax = "proto3";
package trpc.test.helloworld;
option go_package="git.code.oa.com/trpcprotocol/test/helloworld";

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

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter           # service 的路由名称，这里是一个数组，注意：name 前面的减号
      ip: 127.0.0.1                                # 服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口
      network: tcp                                 # 网络监听类型  tcp udp unix
      protocol: trpc                               # 应用层协议 trpc http
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
    - name: trpc.test.helloworld.Hello             # service 的路由名称，这里是一个数组，注意：name 前面的减号
      ip: 127.0.0.1                                # 服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8001                                   # 服务监听端口
      network: tcp                                 # 网络监听类型  tcp udp unix
      protocol: trpc                               # 应用层协议 trpc http
      transport: tnet                              # 要求框架版本 >= 0.11.0，为 tcp trpc 启用 tnet，其他协议可以自行验证
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
```

把 Proto Service 注册到 Naming Service，在多服务场景下需要指定 Naming Service 的名称。

```go
func main() {
    // 通过读取框架配置中的 server.service 配置项，创建 Naming Service
    s := trpc.NewServer()
    // 注册 Greeter 服务
    pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &greeterServerImpl{})
    // 注册 Hello 服务
    pb.RegisterHelloService(s.Service("trpc.test.helloworld.Hello"), &helloServerImpl{})
    // ...
}
```

## 4.4 接口管理

对于框架内置的 tRPC 服务、tRPC 流式服务和泛 HTTP RPC 服务，建议严格遵守 [tRPC-Go 研发规范](https://iwiki.woa.com/pages/viewpage.action?pageId=99485634 "tRPC-Go 研发规范") 来规范服务工程和接口定义。

这三类服务均采用 PB 文件来定义接口。为了方便上下游能更透明的获知接口信息，我们建议使用 **pb 文件与服务分离，与语言无关，通过独立的中心仓库进行统一版本管理** 的思路，通过 **rick 协议管理平台** 来管理 PB 文件，具体请参考 [tRPC-Go 接口管理](https://iwiki.woa.com/pages/viewpage.action?pageId=99485686 " tRPC-Go 接口管理")。

# 5. 服务开发

常见服务类型的搭建，请参考如下链接：

- [搭建 tRPC 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=118272478 "tRPC-Go 快速上手")
- [搭建 tRPC 流式服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289215 "搭建 tRPC 流式服务")
- [搭建泛 HTTP RPC 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=490796254 "搭建泛 HTTP RPC 服务")
- [搭建泛 HTTP 标准服务](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278 "搭建泛 HTTP 标准服务")
- [搭建 gRPC 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289174 "搭建 gRPC 服务")
- [搭建 tars 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=410399255 "搭建 tars 服务")
- [搭建定时器服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289170 "搭建定时器服务")
- [搭建消费者服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289140 "搭建消费者服务")

对于第三方协议服务的开发，请先在 [插件生态](https://iwiki.woa.com/pages/viewpage.action?pageId=447434212 "插件生态") 章节查找协议。对于已支持的插件，可以通过插件文档获取插件的功能、使用接口、示例、配置和限制等信息。

如果在插件生态中没有合适的协议，用户需要自行开发第三方协议，请参考 [协议开发](https://iwiki.woa.com/pages/viewpage.action?pageId=99485626 "协议开发") 章节。同时也欢迎大家贡献第三协议插件到 tRPC 插件生态社区。请参考 [如何加入 tRPC](https://iwiki.woa.com/pages/viewpage.action?pageId=194213720 "如何加入 tRPC") 来贡献代码。

## 5.1 常用 API

tRPC-Go 采用 GoDoc 来管理 tRPC-Go 框架 API 文档。通过查阅 [tRPC-Go API 文档](https://iwiki.woa.com/pages/viewpage.action?pageId=261303106 "tRPC-Go API 文档") 可以获取 API 的接口规范、参数含义和使用示例。

对于 log，metrics 和 config，框架提供了标准调用接口，服务开发只有使用这些标准接口才能和服务治理系统对接。比如日志，如果不使用标准日志接口，而直接使用 `fmt.Printf()`，日志信息是无法上报到远程日志中心的。

- 日志的使用请参考 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=465532424 "这里")
- Metrics API 在 [这里](https://pkg.woa.com/git.code.oa.com/trpc-go/trpc-go/metrics "这里")
- 业务配置使用请参考 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=443605268 "这里")

tRPC-Go 服务端配置支持 **通过框架配置文件** 和 **函数调用传参** 两种方式来配置服务。采用函数调用传参时，参数设置可以参考 [这里](https://pkg.woa.com/git.code.oa.com/trpc-go/trpc-go/server#Option "这里")，**函数调用传参的优先级要大于通过框架配置文件的设置**。建议优先使用框架配置文件来配置服务端，其好处是配置和代码解耦，方便管理。

## 5.2 错误码

tRPC-Go 推荐在写服务端业务逻辑时，使用 tRPC-Go 封装的 `errs.New()` 来返回业务错误码，这样框架能自动上报业务错误码到监控系统。如果业务自定义 error 的话，就只能靠业务主动调用 Metrics SDK 来上报错误码。关于错误码的 API 使用，请参考 [这里](http://godoc.woa.com/git.code.oa.com/tRPC-Go/tRPC-Go/errs "这里")。

tRPC-Go 对错误码的数据类型和含义都做了规划，对于常见错误码的问题定位也都做了解释。具体请参考 [tRPC-Go 错误码手册](https://iwiki.woa.com/pages/viewpage.action?pageId=276029299 "tRPC-Go 错误码手册")。

# 6. 框架配置

对于服务端，必须要配置框架配置中的 `global` 和 `server` 两部分，配置参数的具体含义和取值范围等信息请参考 [框架配置](https://iwiki.woa.com/pages/viewpage.action?pageId=99485621 "框架配置") 文档。而 `plugins` 部分的配置则取决于所选的插件，具体请参考下面的 `7. 插件选择` 章节。

# 7. 插件选择

正如 [tRPC 架构](https://iwiki.woa.com/pages/viewpage.action?pageId=490794790 "架构") 中所描述的，tRPC 框架核心把框架功能插件化，框架核心并不包括具体的实现。对于插件的使用，我们需要以同时 **在 main 文件中 import 插件** 和 **在框架配置文件中配置插件** 的方式来引入插件，这里需要强调的是：**插件的选择必须要在开发阶段确定**。如何使用插件请参考 [北极星名字服务](https://git.woa.com/tRPC-Go/trpc-naming-polaris "北极星名字服务") 中的示例。

tRPC 插件生态提供了丰富的插件，程序如何选择合适的插件呢？这里我们提供了一些思路供大家参考。我们可以把插件可以大致分成三类：独立插件、服务治理插件和存储接口插件。

- 独立插件：比如协议，压缩，序列化，本地内存缓存等插件，其插件的运行不依赖外部系统组件。这类插件的思路比较简单，主要是依据业务功能的需要，和插件的成熟度来做选择。

- 服务治理插件：绝大部分服务治理插件，比如远程日志，名字服务，配置中心等，它们都需要和外部系统对接，对于微服务治理体系有很大的依赖。对这类插件的选择，我们需要明确服务最终运行在什么运营平台上，平台提供了哪些治理组件，服务有哪些能力一定要和平台对接，哪些则不需要。[tRPC-Go 落地实践](https://iwiki.woa.com/pages/viewpage.action?pageId=134416698 "tRPC-Go 落地实践 ") 列举的公司内部各 BG 和 tRPC 对接的实践方案，可供参考。

- 存储接口插件：存储插件主要封装了业界和公司内部成熟数据库，消息队列等组件的接口调用。关于这部分插件，我们首先需要考虑业务的技术选型，什么样的数据库更适合业务的需求。然后基于技术选型来看 tRPC 是否支持，如果不支持，我们可以选择使用数据库原生 SDK，或者建议大家贡献插件到 tRPC 社区。

关于插件详细信息，包括插件的功能、使用、示例、配置和限制等信息，请在 [插件生态](https://iwiki.woa.com/pages/viewpage.action?pageId=447434212 "插件生态") 中获取。

## 7.1 内置插件

框架为服务内置了一些必要的插件，这样可以确保用户在不设置任何插件的情况下，框架仍然能够使用默认插件提供正常的 RPC 调用能力。用户可以自行替换默认插件。

下面的表格列出了作为服务端时框架提供的默认插件，以及插件的默认行为。

|插件类型 | 插件名称 | 默认插件 | 插件行为|
|---|---|---|---|
|log|Console|是 | 默认 debug 级别以上日志打 console，级别可通过配置或者 API 可设置|
|metric|Noop|是 | 不上报 metric 信息|
|config|File|是 | 支持用户使用接口从指定本地文件获取配置项|
|registry|Noop|是 | 不做服务的注册和注销|

# 8. 拦截器

tRPC-Go 提供了拦截器（filter）机制，拦截器在 RPC 请求和响应的上下文设置埋点，允许业务在埋点处插入自定义处理逻辑。像调用链跟踪和认证鉴权等功能通常就是采用拦截器来实现的。常用拦截器请在 [插件生态](https://iwiki.woa.com/pages/viewpage.action?pageId=447434212 "插件生态") 中查找。

业务可以自定义拦截器。拦截器通常和插件组合来实现功能。插件提供配置，而拦截器用于在 RPC 调用上下文插入处理逻辑。关于拦截器的原理、触发时机、执行顺序和自定义拦截器的示例代码，请参考 [tRPC-Go 开发拦截器插件](https://iwiki.woa.com/pages/viewpage.action?pageId=274914183 "tRPC-Go 开发拦截器插件")。

# 9. 测试相关

tRPC-Go 从设计之初就考虑到了框架的易测性，在通过 pb 生成桩代码时，默认会生成 mock 代码。所有的数据库插件也都默认集成了 mock 能力。对于如何对服务做单元测试，[tRPC-Go 单元测试](https://iwiki.woa.com/pages/viewpage.action?pageId=119530324 "tRPC-Go 单元测试") 章节给大家提供了测试的方法和思路。

对于服务的接口测试，tRPC 则提供了 trpc-cli 测试工具，辅助开发人员进行接口调试，与 DevOps 流水线结合进行接口自动化测试。同时公司内部也有一些优秀的图形化接口测试工具可供参考。具体请参考 [tRPC-Go 接口测试](https://iwiki.woa.com/pages/viewpage.action?pageId=346696646 "tRPC-Go 接口测试")。

# 10. 高级功能

## 10.1 超时控制

tRPC-Go 为 RPC 调用提供了 3 种超时机制控制：链路超时、消息超时和调用超时。关于这 3 种超时机制的原理介绍和相关配置，请参考 [tRPC-Go 超时控制](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688 "tRPC-Go 超时控制")。

此功能需要协议的支持（协议需要携带 timeout 元数据到下游）。tRPC 协议、泛 HTTP RPC 协议和 taf 协议均支持超时控制功能。其它协议请联系各自协议负责人。

## 10.2 链路透传

tRPC-Go 框架提供在客户端与服务端之间透传字段，并在整个调用链路透传下去的机制。关于链路透传的机制和使用，请参考 [tRPC-Go 链路透传](https://iwiki.woa.com/pages/viewpage.action?pageId=284269846 "tRPC-Go 链路透传")。

此功能需要协议支持元数据下发功能，tRPC 协议、泛 HTTP RPC 协议和 taf 协议均支持链路透传功能。其它协议请联系各自协议负责人。

## 10.3 反向代理

tRPC-Go 为类似做反向代理的程序提供了完成透传二进制 body 数据，不进行序列化、反序列化处理的机制，以提升转发效率。关于反向代理的原理和示例程序，请参考 [tRPC-Go 反向代理](https://iwiki.woa.com/pages/viewpage.action?pageId=253291617 " tRPC-Go 反向代理")。

## 10.4 自定义压缩方式

tRPC-Go 自定义 RPC 消息体的压缩、解压缩方式，业务可以自己定义并注册压缩、解压缩算法。具体示例请参考 [这里](https://git.woa.com/tRPC-Go/tRPC-Go/blob/master/codec/compress_gzip.go "这里")。

## 10.5 自定义序列化方式

tRPC-Go 自定义 RPC 消息体的序列化、反序列化方式，业务可以自己定义并注册序列化、反序列化算法。具体示例请参考 [这里](https://git.woa.com/tRPC-Go/tRPC-Go/blob/master/codec/serialization_json.go "这里")。

## 10.6 设置服务最大协程数

tRPC-Go 支持服务级别的同/异步包处理模式，对于异步模式采用协程池来提升协程使用效率和性能。用户可以通过框架配置和 `Option` 配置两种方式来设置服务的最大协程数，具体请参考 [tRPC-Go 框架配置](https://iwiki.woa.com/pages/viewpage.action?pageId=99485621) 章节的 service 配置。

## 10.7 性能提升

tRPC-Go 支持高性能网络库 [tnet](https://iwiki.woa.com/p/1387022417)，版本 v0.15.1 可以直接使用 tnet 的 tag v0.15.1-tnet-enabled 来获得性能的提升。

## 10.8 认证鉴权

tRPC-Go 支持使用 Token 的 Knocknock 鉴权方式和 mTLS 鉴权方式来传输 trpc 协议。关于 Knocknock 的机制和使用，请参考 [tRPC-Go 认证鉴权](https://iwiki.woa.com/p/99485623 "tRPC-Go 认证鉴权")；关于 mTLS 的具体示例请参考 [这里](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/mtls "这里")。

## 10.9 保序通信

版本要求：>= v0.19.0（未发布时为 master 分支）

tRPC-Go 支持 服务端保序通信（客户端保序同样支持，一般仅用服务端保序通信即可，客户端保序通信见客户端开发向导），用户可以指定用于提取保序通信 key 的方法以实现在服务端不同 key 之间并行执行，相同 key 内部的请求串行执行，设计文档以及背景见：

* [保序通信 v2-服务端保序](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMcHVLxkAbQJadC2C1On?scode=AJEAIQdfAAoL2FHWInAGkAxgZOAFM&isEnterEdit=1)
* [保序通信 v2-客户端保序](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMI8isHzi9QGW7bCf4YO?scode=AJEAIQdfAAobsV1S0QAGkAxgZOAFM&isEnterEdit=1)
* [支持在解码请求的 header/body 后再对请求进行分发](https://git.woa.com/trpc-go/trpc-go/issues/839)

示例见：[examples/features/keeporder](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/keeporder)

用户使用时仅需要关注以下两个 server.Option：

```go
// WithKeepOrderPreDecodeExtractor returns a ListenServeOption which enables the keep order feature
// by providing pre-decoding extractor.
//
// By providing the pre-decoding extractor, a keep-order key will be extracted from the decoding result
// or the raw binary request body.
// Requests sharing the same keep-order key are processed serially within the same group.
// Requests from different groups, identified by different keys, are processed in parallel.
//
// The default value is nil (do not keep order).
func WithKeepOrderPreDecodeExtractor(preDecodeExtractor keeporder.PreDecodeExtractor) Option {
    return func(o *Options) {
        o.ServeOptions = append(o.ServeOptions, transport.WithKeepOrderPreDecodeExtractor(preDecodeExtractor))
    }
}

// WithKeepOrderPreUnmarshalExtractor returns a ListenServeOption which enables the keep order feature
// by providing pre-unmarshalling extractor.
//
// By providing the pre-unmarshalling extractor, a keep-order key will be extracted from the unmarshalled request.
// Requests sharing the same keep-order key are processed serially within the same group.
// Requests from different groups, identified by different keys, are processed in parallel.
//
// The default value is nil (do not keep order).
func WithKeepOrderPreUnmarshalExtractor(preUnmarshalExtractor keeporder.PreUnmarshalExtractor) Option {
    return func(o *Options) {
        o.ServeOptions = append(o.ServeOptions, transport.WithKeepOrderPreUnmarshalExtractor(preUnmarshalExtractor))
    }
}
```

分别用于：1. 从元数据中提取保序 key，2. 从请求结构体中提取保序 key

`keeporder.PreDecodeExtractor` 和 `keeporder.PreUnmarshalExtractor` 的定义如下：

```go
// PreDecodeExtractor defines a function type that extracts a key which is used to maintain the order of requests
// from the decoded results and the raw request body.
//
// It returns a keep-order key and a boolean.
//
// If the boolean is false, the keep-order feature is disabled for the request.
//
// When enabled, requests sharing the same keep-order key are processed serially within the same group.
// Requests from different groups, identified by different keys, are processed in parallel.
type PreDecodeExtractor func(ctx context.Context, reqBody []byte) (string, bool)

// PreUnmarshalExtractor defines a function type that extracts a key which is used to maintain the order of requests
// from the unmarshalled request body.
//
// It returns a keep-order key and a boolean.
//
// If the boolean is false, the keep-order feature is disabled for the request.
//
// When enabled, requests sharing the same keep-order key are processed serially within the same group.
// Requests from different groups, identified by different keys, are processed in parallel.
type PreUnmarshalExtractor func(ctx context.Context, reqBody interface{}) (string, bool)
```

* `PreDecodeExtractor` 接收 `reqBody []byte`（解码之后的二进制形式的请求体），返回用于保序的 key 以及是否进行保序，用户在实现时可以从 `ctx` 中获取解码时得到的信息。
* `PreUnmarshalExtractor` 接收 `reqBody interface{}` （反序列化之后的请求结构体），返回用于保序的 key 以及是否进行保序，用户在实现时可以对 `reqBody` 做类型断言，以从请求结构体中获取用于保序通信的 key。

**注意：** `WithKeepOrderPreUnmarshalExtractor` 从请求结构体中提取保序 key 的实现会用到反射，性能会有损失，推荐使用 `WithKeepOrderPreDecodeExtractor`。

### 从元数据中提取保序 key

可以在某个公共 package 中（比如 package meta）定义一个保序标识 `KeepOrderKey`，通过客户端设置，然后在服务端的 `server.WithKeepOrderPreDecodeExtractor` 中进行提取以作为保序通信的 key。

#### 服务端写法

关键在于提供 `server.WithKeepOrderPreDecodeExtractor` 选项：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/examples/features/keeporder/meta"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main() {
    s := trpc.NewServer(server.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
            // Implement keep-order logic for pre-decoding.
            msg := codec.Message(ctx)
            m := msg.ServerMetaData()
            if m == nil {
                log.Errorf("meta data is nil for %q\n", reqBody)
                return "", false
            }
            key, ok := m[meta.KeepOrderKey]
            if !ok {
                log.Errorf("meta key %q does not exist for %q\n", meta.KeepOrderKey, reqBody)
                return "", false
            }
            return string(key), true
        }))
    // ...
}
```

#### 客户端写法

关键在于通过 `client.WithMetaData(meta.KeepOrderKey, []byte(key))`，这样服务端收到相同的 key 的请求时会串行执行，不同 key 的请求之间则是并行执行。

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/examples/features/keeporder/meta"
    "git.code.oa.com/trpc-go/trpc-go/examples/features/keeporder/proto"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main() {
    key := "some-key"
    proxy := proto.NewPlayerClientProxy(
        client.WithMetaData(
            meta.KeepOrderKey, []byte(key),
        ))
    ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), time.Second)
    defer cancel()
    req := &proto.UpdateReq{}
    rsp, err := proxy.Update(ctx, req)
    // ...
}
```


### 从请求结构体中提取保序 key

此时不再需要元数据相关的操作，用于保序的 key 直接存在于请求结构体的字段当中，比如我们当前的请求结构体为：

```go
type UpdateReq struct {
    ID string // ...
    // ...
}
```

我们以其中的 `ID` 字段作为保序 key。

#### 服务端写法

关键在于提供 `server.WithKeepOrderPreUnmarshalExtractor` 选项，对请求做类型断言然后返回 `ID` 字段作为保序 key：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/examples/features/keeporder/proto"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main() {
    s := trpc.NewServer(server.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
            // Implement keep-order logic for pre-unmarshaling.
            request, ok := req.(*proto.UpdateReq)
            if !ok {
                log.Errorf("invalid request type %T, want *proto.HelloReq", req)
                return "", false
            }
            return request.ID, true
        }))
    // ...
}
```

#### 客户端写法

无需再指定元数据，只需要在请求结构体的 `ID` 字段上填上期望用于的保序 key 即可：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/examples/features/keeporder/meta"
    "git.code.oa.com/trpc-go/trpc-go/examples/features/keeporder/proto"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main() {
    key := "some-key"
    proxy := proto.NewPlayerClientProxy()
    ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), time.Second)
    defer cancel()
    req := &proto.UpdateReq{ID: key}
    rsp, err := proxy.Update(ctx, req)
    // ...
}
```


# 11. 命令行参数
## 11.1 默认的命令行参数

可以在启动时使用 `-conf` 或 `--conf` 命令行参数来指定配置文件的地址：

```shell
./server -conf ../conf/trpc_go.yaml
```

除此之外，也可以通过代码来指定配置文件地址：

```go
trpc.ServerConfigPath = "../conf/trpc_go.yaml"
```

优先级关系如下：
优先级最高：修改 `ServerConfigPath` 的值。
次高优先级：通过命令行标志 `--conf` 或 `-conf` 设置。
第三优先级：使用 `./trpc_go.yaml` 作为默认路径。

## 11.2 用户自定义命令行参数
用户自定义命令行参数时，需要注意一些问题。
1. 用户自定义的命令行参数，需要放在 `trpc.NewServer()` 之前或者放在 `init()` 之中，这样不需要用户再手动执行 `flag.parse()` 解析命令行参数，而是在 `trpc.NewServer()` 的逻辑中自动解析。例如：

```go
var (
    customFlag bool
)

func init() {
    // 定义自定义的 flag 参数
    flag.BoolVar(&customFlag, "customFlag", false, "Enable some mode")
}

func main() {
    s := trpc.NewServer() // 这里会自动解析用户自定义的命令行参数
    ...
}
```

2. 如果用户通过代码来指定配置文件地址，那么用户需要手动地调用 `flag.parse()` 解析命令行参数，因为代码指定配置文件地址的优先级更高，框架不会再执行命令行的解析。

```go
var (
    customFlag bool
)

func init() {
    // 定义自定义的 flag 参数
    flag.BoolVar(&customFlag, "customFlag", false, "Enable some mode")
}

func main() {
    // 使用代码的方式指定配置文件地址
    trpc.ServerConfigPath = "../conf/trpc_go.yaml" 
    flag.Parse() // 需要用户手动解析自己定义的命令行参数
    // 或者
    // flag.Parse()
    // trpc.ServerConfigPath
    // 在两种方式中，命令行参数中的配置文件地址都不会覆盖代码指定的配置文件地址
    s := trpc.NewServer() // 这里不会再执行 flag.Parse()
    ...
}
```

具体原因可以参考 `trpc.NewServer()` 过程中的一段代码逻辑：
```go
func serverConfigPath() string {
    // 配置文件地址的修改或者 flag.Parse() 执行过都会让 flag.Parse() 不再执行
    if ServerConfigPath == defaultConfigPath && !flag.Parsed() {
        flag.StringVar(&ServerConfigPath, "conf", defaultConfigPath, "server config path")
            flag.Parse()
    }
    return ServerConfigPath
}
```

# 12. FAQ

## 12.1 服务使用相关问题

### Q1 - 同一个服务如何同时对外暴露 trpc 协议和 http 协议？

一个 server 可以有多个 service，每个 service 一个端口一个协议，同一个服务对外暴露两种协议，配置两个 service 即可。pb 只需要定义一个 service，register 的时候会自动注册到配置的所有 service 上（如果不了解 service 的映射关系，请看 [这里](https://git.woa.com/trpc-go/trpc-go/tree/master/server#service-mapping)）。

```yaml
server:                                            #服务端配置
  app: test                                        #业务的应用名
  server: Greeter                                  #进程服务名
  service:                                         #业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter1          #service 的路由名称
      ip: 127.0.0.1                                #服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8000                                   #服务监听端口
      network: tcp                                 #网络监听类型  tcp udp
      protocol: trpc                               #应用层协议 trpc http
      timeout: 1000                                #请求最长处理时间 单位 毫秒
    - name: trpc.test.helloworld.Greeter2          #service 的路由名称
      ip: 127.0.0.1                                #服务监听 ip 地址，ip 和 nic 二选一，优先 ip
      port: 8080                                   #服务监听端口
      network: tcp                                 #网络监听类型  tcp udp
      protocol: http                               #应用层协议 trpc http
      timeout: 1000                                #请求最长处理时间 单位 毫秒
```

### Q2 - 如何获取上游调用方的 IP 和 port？

```go
msg := trpc.Message(ctx)
addr := msg.RemoteAddr() // 返回标准库 net.Addr 结构体，可以通过 addr.String() 获取 ip:port 地址字符串
```

### Q3 - 如何提前回包再慢慢处理其他逻辑？

直接启动 goroutine 即可，return 后框架会自动回包。

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    // implement business logic here ...
    // ...

    trpc.Go(ctx, time.Minute, func(ctx context.Context) {
        // 慢慢处理较慢逻辑
        // 注意：请求入口函数 SayHello return 后会马上 cancel ctx，所以这里的异步逻辑不可以使用请求入口的 ctx，详细见客户端基础功能文档
    })

    return nil
}
```

### Q4 - 如何修改接收数据的最大大小限制？

修改全局变量 `trpc.DefaultMaxFrameSize`，例如 `trpc.DefaultMaxFrameSize = 11111111111`。

### Q5 - 多个 service 能否监听同一个 ip:port？

不可以，不同的 service、不同的协议就是通过不同的 port 来定位的，如果配置成相同 ip:port 则会出现混乱问题。

### Q6 - 多个不同 server 是否可以共用一个 pb 文件？

可以。
很多场景，如一些数据服务，需要使用同样的 pb 部署不同的实例，此时即可多个不同服务共用一个 pb 文件。
首先，pb 文件 package 服务名格式（trpc.app.server）只是一个建议，一个默认值（名字服务默认值是 pb 的 package.service），但是具体的服务名还是需要用户自己在框架配置上面填写。
由于 rick 平台限制了 pb 的 package 格式必须是 trpc.app.server，所以自己定义的 pb 的 package 如果不是这个格式的话，那么就不能使用 rick 平台，可以自己通过 trpc 工具本地生成好桩代码，自己 push 到自己的 git 仓库。
部署多个服务时，所有服务使用相同的 pb 文件（import 上面说的 git 地址），只需要在框架配置 server.service.name 里面自己配置独立的服务名即可：

```yaml
server:
  service:
    name: trpc.app.server.yourservicename  # 可以自己随意配置，会把该服务注册到北极星
    protocol: trpc
```

在上游调用方，调用服务时，需要自己手动设置被调服务名，可以通过代码设置 `client.WithServiceName("trpc.app.server.yourservicename")`，也可以配置 client：

```yaml
client:
  service:
    callee: pbpackagename.pbservice  # 这里是 pb 文件的被调配置 pb 包名.pbservice 名，用于框架通过 client 桩代码寻找该配置，如果同个 server 内部调用了多个使用了相同 pb 的下游 server，则只能使用代码 option
    name: trpc.app.server.yourservicename  # 上面 server 端配置的服务名，用于北极星寻址
```

### Q7 - pb 中 `package/service/method` 以及 trpc_go.yaml 中的 `service.name` 与服务注册发现和请求路由的关系？

service 的方法分发是通过这个格式 `/package/service/method` 来分发的。

1. 一个 pb 协议文件可能定义多个 service，每个 service 的 method 有可能一样，所以直接一个 method 肯定是不够的。
2. 多个 pb 协议文件也可以注册到同一个端口服务，每个协议文件可能除了 package 不一样，service 和 method 都有可能一样。
3. 同一份代码可能部署到不同业务实例上（特别是存储），路由用的 service name 肯定要不一样，所以路由 service name 和 pb service name 也可以不一样。

### Q8 - 如何关闭 reuseport？

有些老机器如 tlinux1.2 不支持 reuseport，可通过以下代码关闭

```go
func main() {
    // main 函数启动入口调用
    transport.DefaultServerTransport = transport.NewServerStreamTransport(transport.WithReusePort(false))

    s := trpc.NewServer()
    // xxx
}
```

### Q9 - 消费者的 `service.name` 该怎么写？

- 如果只有消费者一个 service，则名字可以任意起（不用关心映射问题，tRPC-Go 框架默认会把实现注册到 server 里面的所有 service）。名字这里就是一个符号，在上报等地方会用到。
- 如果有多个 service，则需要在注册时指定与配置文件相同的名字（以 kafka 消息队列为例）

  ``` yaml
  server:
    service:
      - name: trpc.databaseDemo.kafka.consumer1                             
        address: 9.134.192.186:9092?topics=test_topic&group=group_consumer1 
        protocol: kafka                                                     
        timeout: 1000                                                       
      - name: trpc.databaseDemo.kafka.consumer2                             
        address: 9.134.192.186:9092?topics=test_topic&group=group_consumer2 
        protocol: kafka                                                     
        timeout: 1000     
  ```

  ``` go
      s := trpc.NewServer()    
      kafka.RegisterConsumerService(s.Service("trpc.databaseDemo.kafka.consumer1"), &Consumer{})
      kafka.RegisterConsumerService(s.Service("trpc.databaseDemo.kafka.consumer2"), &Consumer{})
  ```

### Q10 - 消费者 service 配置里面的 timeout 设置了不管用？

每个消息队列实现超时时间的方式不同，不同的消息队列需要到各自组件的 README 里面找对应超时时间的设置方法。

### Q11 - 123 平台发布的消费者服务的状态为什么是 unhealthy？

在 123 平台发布服务时 app server 必须使用占位符，不允许自己乱填。

### Q12 - 如何实现一个 http 转 trpc 代理？

trpc-go 支持反向代理，见 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=253291617)。
http 做代理直接使用原生标准库提供 http 服务就可以了，不用这么复杂，然后通过客户端透传模式转发给下游即可。服务端透传模式主要用于自定义协议。

### Q13 - tRPC-Go 和 tRPC-Cpp 互调时，如果使用 snappy 压缩会出错？

snappy 压缩分两种模式：stream 和 block，两者互不兼容。
trpc-go 的 snappy 压缩使用的是 stream 模式的，trpc-cpp 的 snappy 压缩使用的是 block 模式的。
解决方法：在代码中替换 snappy 压缩模式成 block 模式。

```go
import "git.code.oa.com/trpc-go/trpc-go/codec"

func main() {
    codec.RegisterCompressor(codec.CompressTypeSnappy, codec.NewSnappyBlockCompressor())
}
```

### Q14 - 服务端如何指定不进行序列化？

存在业务场景需要直接传输二进制数据，不对数据进行序列化。tRPC-Go 中提供了 `codec.Body` 来传输二进制数据，请求包和响应包都应该使用 `codec.Body`，否则会出现序列化失败。
客户端代码见 [客户端如何指定不进行序列化](https://iwiki.woa.com/p/284289117#q8-客户端如何指定不进行序列化？)。
单次 RPC 服务端代码：

```go
import (
    "context"
    "fmt"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

type GreeterService interface {
    SayHello(ctx context.Context, req *codec.Body) (*codec.Body, error)
}

var GreeterServer_ServiceDesc = server.ServiceDesc{
    ServiceName: "trpc.test.helloworld.Greeter",
    HandlerType: ((*GreeterService)(nil)),
    Methods: []server.Method{
        {
            Name: "/trpc.test.helloworld.Greeter/SayHello",
            Func: GreeterService_SayHello_Handler,
        },
    },
}

func GreeterService_SayHello_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
    req := &codec.Body{}
    filters, err := f(req)
    if err != nil {
        return nil, err
    }
    handleFunc := func(ctx context.Context, reqbody interface{}) (interface{}, error) {
        return svr.(GreeterService).SayHello(ctx, reqbody.(*codec.Body))
    }

    var rsp interface{}
    rsp, err = filters.Filter(ctx, req, handleFunc)
    if err != nil {
        return nil, err
    }
    return rsp, nil
}

type greeterImpl struct{}

func (s *greeterImpl) SayHello(ctx context.Context, req *codec.Body) (*codec.Body, error) {
    fmt.Println(string(req.Data))
    return &codec.Body{Data: []byte("world")}, nil
}

func main() {
    s := trpc.NewServer()
    if err := s.Register(&GreeterServer_ServiceDesc, &greeterImpl{}); err != nil {
        panic(err)
    }
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

流式 RPC 服务端代码：

```go
func (s *greeterServiceImpl) ClientStreamSayHello(stream pb.Greeter_ClientStreamSayHelloServer) error {
    var rspbuf string
    for {
        m := new(codec.Body)
        err := stream.RecvMsg(m)
        if err == io.EOF {
            if err := stream.SendMsg(&codec.Body{Data: []byte(rspbuf)}); err != nil {
                return err
            }
            return nil
        }
        if err != nil {
            return err
        }
        rspbuf = rspbuf + string(m.Data) + ", "
        fmt.Println(string(m.Data))
    }
}
```

## 12.2 流式服务相关问题

### Q1 - trpc 流式和 grpc 流式的区别？

trpc 流式是基于 tcp 协议的 rpc 协议，而 grpc 是基于 http2 的通用 7 层协议。

- 从实现复杂度上说，http2.0 肯定比 trpc 流式复杂，作为一个标准协议，需要考虑和遵循的细节肯定比流式要复杂得多。
- 从功能的角度上来说，二者都可以进行流式传输，服务端和客户端可以进行交互式响应。但 http2.0 只是标准协议，没有 trpc 流式类似于 rpc，流控，异常处理等能力，这些需要在流式协议进行支持。grpc 的流式就是基于 http2.0 协议的，在上面增加了 rpc，流控等功能。
- 在性能方面，目前还没有这方面的对比数据，但可以从协议的角度对比，trpc 流式是直接基于 tcp 的，而 grpc 流式是基于 http2.0，协议上就多了一层，性能上可能会优于 grpc。

### Q2 - 使用 trpc create 生成的桩代码不包含 stream 逻辑？

请升级到 trpc 工具的最新版本，具体见 [这里](https://iwiki.woa.com/p/99485252#251-trpc)。

### Q3 - trpc create 生成的桩代码报错 `unknown embedded interface git.code.oa.com/trpc-go/trpc-go/client.ClientStream`？

原因：`ClientStream` 是 v0.8.4 及其之后从 stream 包中调整到 client 包的。trpc create 生成了 v0.8.4 之后的桩代码。
解决方案：升级 go.mod 中的 trpc-go 版本到 v0.8.4 及其之后的某个 v0.8.x 版本。

### Q4 - 服务端没有 `CloseSend` 接口？

服务端没有实现 `CloseSend` 接口，怎么告知服务端停止发送？ —— 设计如此，服务端如果想停止接收，直接 return 即可，代表流的结束。

### Q5 - 报错 StreamTransport is not implemented？

同个 service 下面不能同时支持 http 和 stream，请分成两个 service 注册。多服务注册可以采用下面方法：

```go
func main() {
    // 通过读取框架配置中的 server.service 配置项，创建 Naming Service
    s := trpc.NewServer()
    // 注册 Greeter 服务
    pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &greeterServerImpl{})
    // 注册 Hello 服务
    pb.RegisterHelloService(s.Service("trpc.test.helloworld.Hello"), &helloServerImpl{})
    // ...
}
```

在某些情况下，如果分开注册到不同的 service 中也出现 StreamTransport is not implemented 错误的话，可能是 [部分第三方库修改了 trpc-go 框架的上述执行逻辑，例如修改 server.New 函数导致错误](https://mk.woa.com/q/283093)。

### Q6 - 报错 msg: client streaming protocol violation: get nil, want EOF？

解决方法：请更新 trpc 工具到 v0.6.8 版本以上后重新生成桩代码。
trpc-go 框架 v0.9.4 版本流式 `Recv()` 接口会判断流的类型，旧版的桩代码没有正确设置流的类型，需要更新 trpc 工具，重新生成桩代码后即可解决。

## 12.3 tars 服务相关问题

### Q1 - tars 服务如何调用 trpc 服务？

请参考 [TarsgoCallTrpcExample](https://git.woa.com/tarsgo/tars-examples/tree/master/TarsgoCallTrpcExample)。

### Q2 - trpc-tars 服务是否支持通过 http 协议访问？

支持自由切换 http/tars 协议，只需要在配置文件修改 protocol 字段即可，重启服务即可。

```yaml
server:                                            # 服务端配置
  app: test                                        # 业务的应用名
  server: Greeter                                  # 进程服务名
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter           # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      protocol: http                               # 应用层协议 trpc http
```

访问命令：

```shell
curl -v -d '{"req":{"msg": "hello"}}' -H "Content-Type: application/json" -X POST "http://127.0.0.1:8000/hello"
```

### Q3 - trpc-tars 服务是否类似 trpc 协议服务支持一个 service 绑定多个 interface？

为了和老的 tars 服务兼容，目前是不支持的。

### Q4 - tars 服务调用 trpc-tars 服务并发量高的时候出现大量超时？

trpc-go 框架对于同一个连接，默认是串行处理的，而 tars client 调用对端对于同一个节点则是长连接多路复用，当并发量大一点时 trpc-go 串行处理不过来，就会出现大量超时的情况。
新版本的 trpc-go(v0.3.2 以上) 已经支持异步处理请求了，可以修改框架配置 trpc_go.yaml，将异步处理的开关打开。

```yaml
server:                                            # 服务端配置
  app: test                                        # 业务的应用名
  server: Greeter                                  # 进程服务名
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter           # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      protocol: trpc                               # 应用层协议 trpc http
      server_async: true                           # 开启异步处理
```

### Q5 - 有没有现有 tars 服务迁移到 trpc-go 的案例？

可以参考 [这篇文章](http://km.woa.com/group/46995/articles/show/440134)，原服务是 tafcpp，迁移为 trpc-go。

### Q6 - trpc 服务调用 tars 服务，报如下错误：client codec empty？

检查 main.go 中是否有引入 tars 插件：

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-codec/tars"
)
```

### Q7 - trpc 服务调用 tars 服务，是否支持按 set 调用？

已经支持，具体请参考 [tRPC-Go Set 路由](https://iwiki.woa.com/pages/viewpage.action?pageId=118669392)。

### Q8 - trpc 调用 tars 服务报错，code:121, msg:client codec Mashal:not jce.Message？

原因：jce 仓库升级了 woa 域名，trpc 最新版本引用 woa 的 jce 的仓库，而老的 trpc4tars 工具生成的桩代码引用的是 `git.code.oa` 的 jce 仓库。

解决方案：升级 trpc4tars 工具，并重新生成桩代码。

```bash
go get git.code.oa.com/trpc-go/trpc-codec/tars && go install git.code.oa.com/trpc-go/trpc-codec/tars/tools/trpc4tars
```

### Q9 - 调用 tars 服务返回 -3 错误码？

-3 错误码表示服务端没有实现该函数，一般是因为你实际调用的服务和你用的 jce 协议文件不匹配，建议仔细检查一下是否调错服务。

更加详细的 tars 框架错误码见下：

```go
const int JCESERVERSUCCESS        = 0;      // 服务器端处理成功
const int JCESERVERDECODEERR     = -1;      // 服务器端解码异常
const int JCESERVERENCODEERR     = -2;      // 服务器端编码异常
const int JCESERVERNOFUNCERR     = -3;      // 服务器端没有该函数
const int JCESERVERNOSERVANTERR  = -4;      // 服务器端没有该 Servant 对象
const int JCESERVERRESETGRID     = -5;      // 服务器端灰度状态不一致
const int JCESERVERQUEUETIMEOUT  = -6;      // 服务器队列超过限制
const int JCEASYNCCALLTIMEOUT    = -7;      // 异步调用超时
const int JCEINVOKETIMEOUT       = -7;      // 调用超时
const int JCEPROXYCONNECTERR     = -8;      // proxy 链接异常
const int JCESERVEROVERLOAD      = -9;      // 服务器端超负载，超过队列长度
const int JCEADAPTERNULL         = -10;     // 客户端选路为空，服务不存在或者所有服务 down 掉了
const int JCEINVOKEBYINVALIDESET = -11;     // 客户端按 set 规则调用非法
const int JCECLIENTDECODEERR     = -12;     // 客户端解码异常
const int JCESERVERUNKNOWNERR    = -99;     // 服务器端位置异常
```

## 12.4 服务运行相关问题

### Q1 - 服务部署好以后，如何自测？

开发阶段自己给自己的服务发包测试。

切记：**公司的办公网，开发网，idc 网是不互通的，要确保客户端工具和服务端在同一个环境上**。
比如 server 部署到 idc 上以后，可以把 trpc-cli 工具 rz 到 idc 测试机上，再发起自测，在 devcloud 是不通的，除非自己申请开通网络策略。
除了 trpc-cli 工具，现在也有很多测试平台，比如 rick，123 接口测试插件等，可以使用这些平台发起自测。
更加详细的接口测试文档可查看[这里](https://iwiki.woa.com/pages/viewpage.action?pageId=346696646)。

### Q2 - 我的服务 CPU 负载很高，QPS 也小，性能很低怎么办？

1. 首先把框架和所有依赖都升级到最新版，框架一直都在性能优化中，新版本框架肯定比老版本性能高。
2. 确认下 trpc 工具的 protoc-gen-go 版本和 gomod 文件里面的 protobuf 版本，都必须 1.4 版本以上。
3. 不要打印太多日志，尽量只打印关键字段，不要把整个请求包体几千个字符都打印出来。
4. 不要频繁创建碎小结构体指针，不要缓存大量结构体指针，缓存可以考虑使用 [bigcache](https://git.woa.com/trpc-go/trpc-database/tree/master/bigcache)。
5. 利用 [管理命令](https://iwiki.woa.com/pages/viewpage.action?pageId=99485663) 的火焰图代理分析自己的服务性能瓶颈。
6. 如果性能主要耗在 gc 上，可以自己调一下 gc 参数。
7. 如果性能主要耗在创建连接上，可以设置使用 I/O 复用减少连接数，[`client.WithMultiplexed(true)`](https://git.woa.com/trpc-go/trpc-go/blob/master/client/options.go#L539) ，注意：这里的前提是被调 server 支持 I/O 复用，如被调 server 不支持则 client 会出现大量超时错误。

### Q3 - 服务运行一段时间 panic 重启是什么原因？

- golang 的 map 不是线程安全的，map 并发读写会出现 panic 导致 crash 而且是无法捕获的 panic，所以服务重启 99% 的概率都是因为写了 map 并发读写的代码。仔细排查下自己的代码，特别是用到 map 的地方，不能有并发读写。
- 另外 ctx 里面也包含了 map，所以自己启动的 goroutine 一定不要使用请求入口的 ctx，具体原因看 [客户端开发这里](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)。
- 启动异步任务时，使用 `trpc.Go(ctx, timeout, handler)`，尽量不要自己启动 goroutine。
- 因为 ctx 在 rpc 函数 return 之后会 cancel 销毁，所以对 ctx 的任何操作都不能有并发，包括 clone ctx，必须在 go 之前就 clone 好。
- 确定排除不是 map 问题的话，那就大概率是 OOM 了，可以自己查看内存增长曲线监控或者直接看系统日志（/var/log/messages）。
- 如有提示 nil pointer，out of index 等这些都是你的业务代码有 bug，出现了空指针，数组越界问题，需要好好定位一下。
- 如果是 panic 在 server_transport_udp.go framerBuilder 空指针上，说明协议不存在，请见本小节的 Q8。
- 如果是 panic 在 `trpc.SetMetaData` 上，说明代码使用方式不对，`trpc.SetMetaData` 函数注释明确说明是 **非并发安全** 的，不允许在多个协程里面调用，主要用于返回 metadata 给上游调用方，或者设置 metadata 给所有下游被调方，而不是传 metadata 给单个下游被调方。所以 `trpc.SetMetadata` 只能在 server rpc 入口协程里面调用，不能在调用 client 的协程里面调用，你要透传给下游应该用 `client.WithMetadata` 选项。
- 使用 HTTP client 产生 panic，出现 `http.(*valueDetachedCtx).Err` 这样的错误信息，需要升级 trpc-go 版本 >= `v0.8.1`。具体见 [码客](https://mk.woa.com/q/281087)。

### Q4 - codec empty 是什么原因？

trpc-go 的协议插件都是可插拔的，用户使用第三方协议的时候必须 import 对应的协议包，如：

```go
import (
    _ "trpc.tech/trpc-go/trpc-codec/tars/v2"
)
```

### Q5 - 插件初始化失败：setup plugin xxx timeout？

标准输出可以看到初始化的详细 log，新版的框架会收集标准库 log 的输出。异常情况下 007 配置 `debuglogOpen: true` 可以看到 [初始化的详细步骤与每次上报的详情](https://mk.woa.com/note/1067)，注意线上不要开启。

setup plugin metrics-m007 timeout。007 SDK 拉取远程配置依赖北极星，一般是 polaris-discover.oa.com 解析有问题，北极星初始化超时。

1. 升级插件到最新版本，支持北极星默认埋点推荐。删除 polarisAddrs、polarisProto 配置项。
2. 若还有问题，可能是北极星 SDK 拉取 IP 超时，可以拉北极星 helper 和 evannzhang（007 SDK 测）一起看下。

trpc-metrics-m007:pcgmonitor.Setup error:init error。一般是机器问题，无法连接 attaagent(未启动或机器 fd 数过多无法连接), attaapi 错误码意义见 [这里](https://git.woa.com/atta/attaapi_go/blob/master/attaapi_go.go)。

1. 非 123 环境的话业务安装启动 atta agent，见 [这里](http://km.oa.com/articles/show/447456?kmref=search&from_page=1&no=1)。
2. 123 环境的话，要 atta 测来看了，只能提供机器信息拉群解决了 相关人：DataPlatform_helper & 运维。可以切换容器快速解决下。

不支持 devcloud 环境，不建议折腾，需要启动服务，临时删除 007 的相关配置即可。想解决阅读 [pcgmonitor](https://git.woa.com/pcgmonitor/trpc_report_api_go/blob/master/pcgmonitor.go) 的 startup 函数，了解相关的依赖，自行解决网络策略问题，主要 3 点依赖：

- attaagent
- 北极星
- 007 远程服务，路由 64939329:131073

插件启动较慢，还有一个原因是 cpu 核数太小，比如只有 1 核，这种情况也是大概率失败的，需要把核数调大。

### Q6 - read framer head magic not match?

- 确保上下游协议一致，必须同时为 trpc 或者同时为 http 协议。
- 用 [debuglog](https://git.woa.com/trpc-go/trpc-filter/tree/master/debuglog) 插件来定位问题。
- 查看请求的 ipport 是否真的是你想请求的下游服务。
- 确定是否在同一个 ipport 启动了多个 reuseport 服务。

### Q7 - plugin xxx no registered?

插件未注册，检查是否 import 相应插件，详细看下对应插件的 README 文档里面的使用方式。

### Q8 - FramerBuilder empty?

协议插件不存在

1. 确保配置 protocol 是否正确，注意确认协议到底存不存在，配置文件有没有拼写错误。
2. 确保协议插件是否 import。
3. 确保 yaml 配置格式是否正确，大概率是空格没对齐之类的。

### Q9 - panic: filter xxx no registered, do not configure?

拦截器没有注册，不要配置。配置文件删除相应拦截器即可。

### Q10 - client: selector xxx not exist?

trpc 框架的 selector 是插件化的，按需使用，前提是需要自己 import 对应的 selector 注册到框架中才能用。

如报 selector cl5 not exist 错误，则需要 import

```go
import (
    _ "trpc.tech/trpc-go/trpc-selector-cl5/v2"
)
```

### Q11 - trpc framer: read frame header total len too large?

回包过大，无法接收。默认最大包不可超过 10M，请确认下是否有 bug
确实需要传输超大包，可以自己修改 `trpc.DefaultMaxFrameSize` 的值：`trpc.DefaultMaxFrameSize = 111111111`。

比如：

```go
import "git.code.oa.com/trpc-go/trpc-go"

func main() {
    trpc.DefaultMaxFrameSize = 111111111
}
```

在 >= v0.15.0 版本中可以通过配置修改：

```yaml
# 全局配置
global:
  # 选填，最大帧长，单位为 Byte，默认为 10485760（表示 10MB）
  # 如果要调节，注意上下游要同时修改，而不要只改一端
  # 适用于版本 >= v0.15.0
  max_frame_size: Integer
```

### Q12 - 框架 v0.8.0 版本出现 not jce.Message 错误？

由于 v0.8.0 将 jce 的 go.mod 升级到 woa，导致 jce 的前后版本不兼容，用户使用到 jce 的地方也需要升级到 woa（如果使用的是 tars codec 插件，直接升级到 v1.3.0 即可），如果不升级，可以自己实现一个 jce 序列化方式注册到框架也行（以下代码可以写到业务基础库里面，用户 import 即可）。

```go
import (
    "git.code.oa.com/jce/jce"  // 注意这里的地址是 git.code.oa.com 不是 git.woa.com，而且 go.mod 里面的 jce 版本是 v1.0.2，可以执行如下命令：go get git.code.oa.com/jce/jce@v1.0.2
    "git.code.oa.com/trpc-go/trpc-go/codec"
)

func init() {
    codec.RegisterSerializer(codec.SerializationTypeJCE, &JCESerialization{})
}

// JCESerialization 序列化 jce 包体
type JCESerialization struct{}

// Unmarshal 反序列化 jce
func (j *JCESerialization) Unmarshal(in []byte, body interface{}) error {
    return jce.Unmarshal(in, body.(jce.Message))
}

// Marshal 序列化 jce
func (j *JCESerialization) Marshal(body interface{}) ([]byte, error) {
    return jce.Marshal(body.(jce.Message))
}
```

## 12.5 代码编译问题

### Q1 - panic: not implemented

这个是由于 golang/protobuf 升级最新版与 gogo/protobuf 不兼容导致的问题，有以下两个解决办法：

1. 不要使用 gogo/protobuf，所有地方全部改成 github.com/golang/protobuf。
2. 不要使用 golang/protobuf 1.4.0 版本，降级到 github.com/golang/protobuf v1.3.4 版本。

### Q2 - undefined: codec.PutBackMessage

更新框架和 trpc 工具到最新版。

### Q3 - cannot range over client.DefaultClientConfig (type func() map[string]*client.BackendConfig)

低版本的 tconf 和 rainbow 插件会出现这个错误，直接把框架和插件都升级到最新版即可。

### Q4 - proto 1.5.0 新版本因为 proto 文件名重复 panic

```text
panic: proto: file "material_server.proto" is already registered
See https://developers.google.com/protocol-buffers/docs/reference/go/faq#namespace-conflict
```

导入很多 pb 导致重名 proto panic 的原因：google.golang.org/protobuf 这个库在今年 3.18 发布的 1.26 版本会 panic 重名的 proto 文件。目前 trpc 和 rick 都没有升级到这个版本。golang/protobuf v1.5 使用了这个，可能有些库会升级。

如果大家因为 import 很多其他人的 pb 导致有重名 proto 启动时出现 panic，可以尝试以下解决方式：

1. 可以在自己项目 go.mod 里把 github.com/golang/protobuf replace 到 1.4.3 并把 google.golang.org/protobuf 这个 replace 到 1.25.0：

    ```go
    replace (
        github.com/golang/protobuf => github.com/golang/protobuf v1.4.3
        google.golang.org/protobuf => google.golang.org/protobuf v1.25.0
    )
    ```

2. 如果可以自定义 build 的话，关掉 panic：

    ```shell
    go build -ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn"
    ```

3. beanjia 同学提供了一种方式即在 123 上面加环境变量：

    ```text
    GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn
    ```

比较建议使用第一种方法，与环境和配置无关。trpc 工具和 rick 也正在解决这个问题，给传入的 proto 加上路径。

### Q5 - v0.7.0 版本 tRPC-Go 出现 metrics/prometheus/trpc_admin.go: 40: 14: undefined: admin.Run

新版本 trpc-go(v0.7.0) 废弃了 `admin.Run`。天机阁插件升级到 v0.4.3 即可。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
