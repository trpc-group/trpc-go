[English](./basics_tutorial.md) | 中文

## 基础教程

在[快速开始](./quick_start.zh_CN.md)中，你已经成功地运行了 tRPC-Go helloworld。但是，我们忽略了很多细节。这章中，你将更加细致地了解 tRPC-Go 服务开发流程。我们将依次介绍：
- 如何通过 protobuf 定义 tRPC 服务？
- `trpc_go.yaml` 要如何配置？
- tRPC-Go 具有哪些扩展能力？
- tRPC-Go 支持的各种能力。

我们服务依赖 Protocol Buffer v3，你可以参考它 Go 语言的[官方文档](https://protobuf.dev/getting-started/gotutorial/)。

### 定义服务

为了定义一个新服务，我们首先需要在 protobuf 中声明它。下面的示例声明了一个名为 `MyService` 的服务。
```protobuf
service MyService {
  // ...
}
```

一个服务有各种各样的方法，它们需要声明在 service 内部。比如，下面的示例中，我们为 `Greeter` 声明了一个方法 `Hello`，它以 `HelloReq` 作为参数，返回 `HelloRsp`。
```protobuf
service Greeter {
  rpc Hello(HelloReq) returns (HelloRsp) {}
  // ...
}

message HelloReq {
  // ...
}

message HelloRsp {
  // ...
}
```

注意，`Method` 最后有一个 `{}`，其内部也是可以有内容的。我们将在后面看到。

### 编写客户端和服务端代码

protobuf 给出的是一个语言无关的服务定义，我们还要用 [trpc 命令行工具](https://github.com/trpc-group/trpc-cmdline)将它翻译成对应语言的桩代码。你可以通过 `$ tprc create -h` 查看它支持的各种选项。你可以参考快速开始的 [helloworld](/examples/helloworld/pb/Makefile) 项目来快速创建你自己的桩代码。

桩代码主要分为 client 和 server 两部分。  
下面是生成的部分 client 代码。在[快速开始](./quick_start.zh_CN.md)中，我们通过 `NewGreeterClientProxy` 来创建一个 client 实例，并调用了它的 `Hello` 方法：
```go
type GreeterClientProxy interface {
    Hello(ctx context.Context, req *HelloReq, opts ...client.Option) (rsp *HelloRsp, err error)
}

var NewGreeterClientProxy = func(opts ...client.Option) GreeterClientProxy {
    return &GreeterClientProxyImpl{client: client.DefaultClient, opts: opts}
}
```

下面是生成的部分 server 代码，`GreeterService` 约定了你需要实现的接口。`RegisterGreeterService` 会将你的实现注册到框架中。在[快速开始](./quick_start.zh_CN.md)中，我们先通过 `s := trpc.NewServer()` 创建了一个 tRPC-Go 实例，然后把实现了业务逻辑的 `Greeter` 结构体注册到了 `s` 中。
```go
type GreeterService interface {
    Hello(ctx context.Context, req *HelloReq) (*HelloRsp, error)
}

func RegisterGreeterService(s server.Service, svr GreeterService) { /* ... */ }
```

### 框架配置

也许你已经注意到了客户端和服务端的些许不同。在客户端，我们通过 `client.WithTarget` 指定了服务端的地址，但是在服务端，我们并没有在代码中找到对应的地址，实际上它配置在 `./server/trpc_go.yaml` 中。  
这是 tRPC-Go 的支持的 yaml 配置能力。几乎所有 tRPC-Go 框架能力都可以通过文件配置进行定制化。当你执行 tRPC-Go 时，框架会在当前目录下寻找 `trpc_go.yaml` 文件，并加载相关配置。这使得你可以在不重新编译服务的前提下，就更改程序的行为。  
下面是一些本教程所需的必要配置，完整配置请参考[框架配置](/docs/user_guide/framework_conf.zh_CN.md)。
```yaml
server:  # 服务端配置
  service:  # 可以配置多个 service
    - name: helloworld  # 服务名
      ip: 127.0.0.1  # 服务监听的 IP
      port: 8000  # 服务监听的端口
      protocol: trpc  # 服务使用的协议
```

### 拦截器和插件

tRPC-Go 有着丰富的可扩展性，你可通过拦截器在请求执行流程中注入各种新能力，插件工厂则允许你方便地集成新功能。

#### 拦截器

[拦截器](/filter)像一颗洋葱，当 tRPC-Go 处理请求时，会依次经过洋葱的每一层。你可以通过拦截器定制这颗洋葱模型。

客户端拦截器定义如下：
```go
type ClientFilter func(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error
type ClientHandleFunc func(ctx context.Context, req, rsp interface{}) error
```
当实现你自己的拦截器时：
```go
func MyFilter(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
	// 前置流程
	err := next(ctx, req, rsp)
	// 后置流程
	return err
}
```
`next` 前后的代码代会在实际请求之前和之后分别执行，即前置流程和后置流程。你可以实现很多拦截器，通过 [`client.WithFilters`](/client/options.go) 在调用时使用，框架会自动将这些拦截器串成一个链。

服务端拦截器的签名与客户端略有不同：
```go
type ServerFilter func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error)
type ServerHandleFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)
```
`rsp` 在返回值中，而不是参数中。在使用，需要通过 [`server.WithFilters`](/server/options.go) 注入到框架中。框架会自动将这些拦截器串成一个链。

除了上面提到的通过代码直接添加拦截器，你也可以通过配置文件加载拦截器。
```yaml
client:
  filter:  # 指定客户端全局拦截器
    - client_filter_name_1
    - client_filter_name_2
  service:
    - name: xxx
      filter:  # 指定 xxx 客户端特有的拦截器，它们会追加在全局拦截器之后
        - client_filter_name_3
server:
  filter:  # 指定服务端全局拦截器
    - server_filter_name_1
    - server_filter_name_2
  service:
    - name: yyy
      filter:  # 指定 yyy 服务端特有的拦截器，它们会追加在全局拦截器之后
        - server_filter_name_3
```
这些拦截器需要提前通过 [`filter.Register`](/filter/filter.go) 注册在框架中。在执行 `trpc.NewServer` 时，框架会自动加载它们。  
注意，当代码和配置文件同时存在时，代码指定的拦截器会先执行，然后再执行配置文件指定的拦截器。

你可以在[这里](/examples/features/filter)看到拦截器的使用示例。

#### 插件

[插件](/plugin)是 tRPC-Go 基于 yaml 配置文件设计的一套自动化模块加载机制。它的接口定义如下：
```go
package plugin

type Factory interface {
	Type() string
	Setup(name string, dec Decoder) error
}

type Decoder interface {
	Decode(cfg interface{}) error
}
```
`Type` 返回插件的类型，`Setup` 时会传入插件名和一个 `Decoder`，用于解析 yaml 的内容。yaml 来自 `trpc_go.yaml` 配置：
```yaml
plugins:
  __type:
    __name:
      # plugin contents
```
其中 `__type` 应替换为 `Factory.Type()` 返回的值，`__name` 应替换为 `plugin.Register` 的第一个参数。

在实现 `plugin` 时，你应该创建一个 `func init()` 函数，通过 `Register` 注册你的插件。这样别人用你的插件时，只需要在代码中匿名 import 你的包即可。当调用 `trpc.NewServer()` 时，插件就会调用 `Factory.Setup` 函数进行初始化。

插件经常和拦截器配合，比如在 `Factory.Setup` 函数中调用 `filter.Register` 注册拦截器。框架保证插件初始化在拦截器加载之前完成。这样，你就可以通过修改 `trpc_go.yaml` 来配置拦截器的行为。

### 多协议支持

[快速开始](./quick_start.zh_CN.md)中介绍的是普通的一应一答式 RPC。tRPC-Go 还支持流式 RPC、HTTP 等。

#### 流式 RPC

流式 RPC 支持客户端与服务端之间进行更加灵活的交互。它可以分为三种模式：客户端流式、服务端流式和双向流式。  
客户端流式允许客户端依次发送多个包，服务端全收到后，再返回一个包。它是多对一关系。  
服务端流式允许服务端为一个客户端请求生成多次回包。它是一对多关系。  
双向流式允许客户端服务端可以并行地给对方发送请求，并且是保序的，就像两个交谈中的人一样。它是多对多关系。

本小节代码基于 [`example/stream`](/examples/features/stream)。

与普通 RPC 不同，在 protobuf 中声明流式 RPC 需要使用 `stream` 关键字。
```protobuf
service TestStream {
  rpc ClientStream (stream HelloReq) returns (HelloRsp);
  rpc ServerStream (HelloReq) returns (stream HelloRsp);
  rpc BidirectionalStream (stream HelloReq) returns (stream HelloRsp);
}
```
当 `stream` 只出现在方法请求前时，该方法为客户端流式；当 `stream` 只出现在方法返回值前时，该方法为服务端流式；当 `steam` 同时出现在方法的请求和返回值前时，该方法为双向流式。

流式生成的桩代码与普通 RPC 有很大区别，以客户端流式为例：
```go
type TestStreamService interface {
    ClientStream(TestStream_ClientStreamServer) error
	// ...
}

type TestStream_ClientStreamServer interface {
    SendAndClose(*HelloRsp) error
    Recv() (*HelloReq, error)
    server.Stream
}

type TestStreamClientProxy interface {
    ClientStream(ctx context.Context, opts ...client.Option) (TestStream_ClientStreamClient, error)
}

type TestStream_ClientStreamClient interface {
    Send(*HelloReq) error
    CloseAndRecv() (*HelloRsp, error)
    client.ClientStream
}
```
从上面的桩代码，大概可以猜到业务代码的编写方式。客户端通过 `TestStream_ClientStreamClient.Send` 多次发送请求，最后通过 `TestStream_ClientStreamClient.CloseAndRecv` 关闭流并等待回包。服务端则通过 `TestStream_ClientStreamServer.Recv` 接收客户端的流式请求，如果它返回了 `io.EOF`，说明客户端调用了 `CloseAndRecv` 并正在等待回包，它最后再通过 `TestStream_ClientStreamServer.SendAndClose` 发送回包并确认关闭流。注意，客户端流式是由客户端主动结束的，`CloseAndRecv` 表明客户端先关闭，再等待回包，`SendAndClose` 表明服务端先发送回包，再关闭。

服务端流式与客户端流式相反。只要 `TestStreamService.ServerStream` 函数退出，就表示服务端已经完成了流的发送。客户端通过多次调用 `TestStream_ServerStreamClient.Recv` 来获取流式回包，当收到 `io.EOF` 错误时，表明服务端已完成流式回包。

双向流式是前两者的结合。它们的发送和读取可以交叉起来，就像两个人交谈一样，可以实现更加复杂的交互逻辑。

流式 RPC 的配置和普通 RPC 没有任何不同。

#### 标准 HTTP 服务

tRPC-Go 支持将 Go 标准库的 HTTP 服务注册到框架中。假如你需要在 8080 端口监听你的 HTTP 服务，首先，在 `trpc_go.yaml` 中加入如下 service 配置：
```yaml
server:
  service:
    - name: std_http
      ip: 127.0.0.1
      port: 8080
      protocol: http
```
再加入下面的代码即可：
```go
import thttp trpc.group/trpc-go/trpc-go/http

func main() {
	s := trpc.NewServer()
	thttp.RegisterNoProtocolServiceMux(s.Service("std_http"), your_http_handler) 
	log.Info(s.Serve())
}
```

注意，与普通 RPC 不同，yaml 中的 `protocol` 字段需要改成 `http`。`thttp.RegisterNoProtocolServiceMux` 方法的第一个参数需要指定 yaml 中的 service 名，即 `s.Service("std_http")`。

#### 将 RPC 快速转为 HTTP 服务

在 tRPC-Go 中，将 `trpc_go.yaml` 中的 `server.service[i].protocol` 从 `trpc` 改为 `http`，就可以将普通 tRPC 协议的服务转换为 HTTP 协议。在调用时，HTTP url 对应生成的桩代码中的[方法名](/examples/helloworld/pb/helloworld.trpc.go)。
比如，如果将[快速开始](./quick_start.zh_CN.md)里的 RPC 改为 HTTP，在调用时，你需要用下面的 curl 命令：
```bash
$ curl -XPOST -H"Content-Type: application/json" -d'{"msg": "world"}' 127.0.0.1:8000/trpc.helloworld.Greeter/Hello
{"msg":"Hello world!"}
```

如果你需要更改 HTTP url，可以在 protobuf 中声明服务时，加上下面的扩展：
```protobuf
import "trpc/proto/trpc_options.proto";

service Greeter {
  rpc Hello (HelloRequest) returns (HelloReply) {
    option (trpc.alias) = "/alias/hello";
  }
}
```
curl 命令如下：
```bash
$ curl -XPOST -H"Content-Type: application/json" -d'{"msg": "world"}' 127.0.0.1:8000/alias/hello
{"msg":"Hello world!"}
```

#### RESTful HTTP

将 RPC 快速转为 HTTP 虽然很方便，但是却无法定制 HTTP url/parameter/body 和 RPC Request 的映射关系。RESTful 提供了更加灵活的 RPC 到 HTTP 的转换。

你可以参考 [RESTful example](/examples/features/restful) 快速了解 RESTful。更多细节请参考 RESTful 的[文档](/restful)。

### 进一步阅读

tRPC-Go 还支持其他丰富的功能特性，你可以查看它们的[文档](/main/docs/user_guide)了解更多细节。

