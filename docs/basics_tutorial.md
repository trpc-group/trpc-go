English | [中文](./basics_tutorial.zh_CN.md)

## Basics Tutorial

In [Quick Start](./quick_start.md), you have successfully run tRPC-Go helloworld. However, we ignore many details. In this chapter, you will understand the tRPC-Go service development process in more detail. We will introduce in turn:
- How to define tRPC service by protobuf?
- How to configure `trpc_go.yaml`?
- What extension capabilities does tRPC-Go have?
- Various features supported by tRPC-Go.

Our service definition depends on Protocol Buffer v3. You can refer to [Official Documents of Golang Protobuf](https://protobuf.dev/getting-started/gotutorial/).

### Define Service

To define a new service, we need to declare it in protobuf first. The following example defines a service named `MyService`:
```protobuf
service MyService {
  // ...
}
```

A service may have many methods which are defined inside service body. In following example, we define a method `Hello` for service `Greeter`. The `Hello` use `HelloReq` as its parameter and returns `HelloRsp`.
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

Note that `Method` has a `{}` at the end, which can also have content. We will see later.

### Write Client and Server Code

What protobuf gives is a language-independent service definition, and we need to use [trpc command line tool](https://github.com/trpc-group/trpc-cmdline) to translate it into a corresponding language stub code. You can see the various options it supports with `$ trpc create -h`. You can refer to the quick start [helloworld](/examples/helloworld/pb/Makefile) project to quickly create your own stub code.

The stub code is mainly divided into two parts: client and server.  
Below is part of the generated client code. In [Quick Start](./quick_start.md), we use `NewGreeterClientProxy` to create a client instance and call its `Hello` method:
```go
type GreeterClientProxy interface {
    Hello(ctx context.Context, req *HelloReq, opts ...client.Option) (rsp *HelloRsp, err error)
}

var NewGreeterClientProxy = func(opts ...client.Option) GreeterClientProxy {
    return &GreeterClientProxyImpl{client: client.DefaultClient, opts: opts}
}
```

The following is part of the generated server code, `GreeterService` defines the interface you need to implement. `RegisterGreeterService` will register your implementation to the framework. In [Quick Start](./quick_start.md), we first create a tRPC-Go instance through `s := trpc.NewServer()`, and then register the `Greeter` structure that implements the business logic to `s`.
```go
type GreeterService interface {
    Hello(ctx context.Context, req *HelloReq) (*HelloRsp, error)
}

func RegisterGreeterService(s server.Service, svr GreeterService) { /* ... */ }
```

### Configuration

Maybe you have noticed a little difference between client and server. On the client side, we specified the address of the server through `client.WithTarget`, but on the server side, we did not find the corresponding address in the code. In fact, it is configured in `./server/trpc_go.yaml`.  
This is the yaml configuration capability supported by tRPC-Go. Almost all tRPC-Go framework capabilities can be customized through file configuration. When you execute tRPC-Go, the framework will look for the `trpc_go.yaml` file in the current directory and load the relevant configuration. This allows you to change the behavior of your application without recompiling the service.  
Below are some necessary configurations required for this tutorial, please refer to [Framework Configuration](/docs/user_guide/framework_conf.md) for complete configurations.
```yaml
server:
  service:  # you can config multiple services
    - name: helloworld
      ip: 127.0.0.1
      port: 8000
      protocol: trpc
```

### Filter and Plugin

tRPC-Go has rich scalability, you can inject various new capabilities into the RPC process through filters, and the plugin factory allows you to easily integrate new functions.

#### Filter

[Filter](/filter) likes an onion. An RPC goes through each layer of the onion in turn. You can customize this onion model through Filters.

The client filter is defined as follows:
```go
type ClientFilter func(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error
type ClientHandleFunc func(ctx context.Context, req, rsp interface{}) error
```
When implement your own filters：
```go
func MyFilter(ctx context.Context, req, rsp interface{}, next ClientHandleFunc) error {
    // pre-RPC processes
    err := next(ctx, req, rsp)
    // post-RPC processes
    return err
}
```
Codes before and after `next` will be executed before and after the actual RPC, that is, pre-RPC processes and post-RPC processes. You can implement many filters, which are used when calling [`client.WithFilters`](/client/options.go). Framework Will automatically concat these filter into a chain.

The signature of server filter is slightly different from client:
```go
type ServerFilter func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error)
type ServerHandleFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)
```
`rsp` is in the return value, not the argument. Server filter should be injected into the framework by [`server.WithFilters`](/server/options.go). The framework will automatically concat these filters into a chain.

In addition to adding filters directly through code mentioned above, you can also load filters through configuration files.
```yaml
client:
  filter:  # these are client global filters
    - client_filter_name_1
    - client_filter_name_2
  service:
    - name: xxx
      filter:  # these are special filters for service xxx, they will be appended to global filters.
        - client_filter_name_3
server:
  filter:  # these are server global filters
    - server_filter_name_1
    - server_filter_name_2
  service:
    - name: yyy
      filter:  # these are special filters for service yyy, they will be appended to global filters.
        - server_filter_name_3
```
These filters need to be registered in the framework in advance via [`filter.Register`](/filter/filter.go). They are automatically loaded by the framework when `trpc.NewServer` is executed.  
Note that when the code and the configuration file exist at the same time, the interceptor specified by the code will be executed first, and then the interceptor specified by the configuration file.

You can see an example of filter usage [here](/examples/features/filter).

#### Plugin

[Plugin](/plugin) is an automatic module loading mechanism designed by tRPC-Go based on yaml configuration file. Its interface is defined as follows:
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
`Type` returns the type of the plugin, and `Setup` will pass in the plugin name and a `Decoder` for parsing the content of yaml. The content of plugin is configured in `trpc_go.yaml`:
```yaml
plugins:
  __type:
    __name:
      # plugin contents
```
where `__type` should be replaced by the value returned by `Factory.Type()` and `__name` should be replaced by the first parameter of `plugin.Register`.

When implementing a `plugin`, you should create a `func init()` function to register your plugin via `Register`. In this way, when others use your plugin, they only need to import your package anonymously in the code. When `trpc.NewServer()` is called, the plugin will call the `Factory.Setup` function for initialization.

Plugins often cooperate with filters, such as calling `filter.Register` in the `Factory.Setup` function to register a filter. The framework guarantees that plugin initialization completes before filters are loaded. This way, you can configure the behavior of the filter by modifying plugin of `trpc_go.yaml`.

### Other Supported Protocols

[Quick start](./quick_start.md) introduces a common one-request-one-response RPC. tRPC-Go also supports streaming RPC, HTTP, and more.

#### streaming RPC

Streaming RPC supports more flexible interactions between clients and servers. It can be divided into three modes: client-side streaming, server-side streaming, and bidirectional streaming.
Client streaming allows the client to send multiple packets in sequence, and the server returns a packet after receiving all of them. It is a many-to-one relationship.  
Server-side streaming allows the server to generate multiple responses for a client request. It is a one-to-many relationship.  
Bidirectional streaming allows the client and the server to send requests to each other in parallel, in order, just like two people in a conversation. It is a many-to-many relationship.

The code in this section is based on [`example/stream`](/examples/features/stream).

Unlike normal RPCs, declaring streaming RPCs in protobuf requires the use of the `stream` keyword.
```protobuf
service TestStream {
  rpc ClientStream (stream HelloReq) returns (HelloRsp);
  rpc ServerStream (HelloReq) returns (stream HelloRsp);
  rpc BidirectionalStream (stream HelloReq) returns (stream HelloRsp);
}
```
When `stream` only appears before the method request, the method is client-side streaming; when `stream` only appears before the method response, the method is server-side streaming; when `steam` appears both before the method request and the method response, the method is bidirectional.

The stub code generated by streaming is very different from normal RPC. Take client streaming as an example:
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
From the stub code above, you can probably guess how the business code is written. The client sends requests multiple times by `TestStream_ClientStreamClient.Send`, and finally closes the stream by `TestStream_ClientStreamClient.CloseAndRecv` and waits for the response. The server receives the client's streaming requests by `TestStream_ClientStreamServer.Recv`, if it returns `io.EOF`, it means that the client has called `CloseAndRecv` and is waiting for the return packet, and it will finally send response by `TestStream_ClientStreamServer.SendAndClose` and confirm to close the stream. Note that the client streaming termination is initialized by the client. `CloseAndRecv` indicates that the client closes first, and then waits for a response. `SendAndClose` indicates that the server sends a response first, and then confirm close.

Server-side streaming is the opposite of client-side streaming. As soon as the `TestStreamService.ServerStream` function exits, it means that the server has finished sending responses. The client obtains the stream responses by calling `TestStream_ServerStreamClient.Recv` multiple times. When the `io.EOF` error is received, it indicates that the server has completed the stream responses.

Bidirectional streaming is a combination of the previous two. Their sending and reading can be interleaved, just like two people talking, and more complex interaction logic can be realized.

The configuration of streaming RPC is no different from normal RPC.

#### Standard HTTP Service

tRPC-Go supports registering HTTP services from the Go standard library into the framework. Suppose that you need to listen an HTTP service on port 8080, first, add the following service configuration to `trpc_go.yaml`:
```yaml
server:
  service:
    - name: std_http
      ip: 127.0.0.1
      port: 8080
      protocol: http
```
Then add following code:
```go
import thttp trpc.group/trpc-go/trpc-go/http

func main() {
    s := trpc.NewServer()
    thttp.RegisterNoProtocolServiceMux(s.Service("std_http"), your_http_handler) 
    log.Info(s.Serve())
}
```

Note that unlike ordinary RPC, the `protocol` field in yaml needs to be changed to `http`. The first parameter of the `thttp.RegisterNoProtocolServiceMux` method needs to specify the service name in yaml, namely `s.Service("std_http")`.

#### Translate RPC to HTTP Quickly

In tRPC-Go, changing `server.service[i].protocol` in `trpc_go.yaml` from `trpc` to `http` can convert the service of ordinary tRPC to HTTP. When calling, the HTTP url corresponds to the [method name](/examples/helloworld/pb/helloworld.trpc.go).

For example, if you change the RPC in [Quick Start](./quick_start.md) to HTTP, you need to use the following curl command to call it:
```bash
$ curl -XPOST -H"Content-Type: application/json" -d'{"msg": "world"}' 127.0.0.1:8000/trpc.helloworld.Greeter/Hello
{"msg":"Hello world!"}
```

If you need to change the HTTP url, you can add the following extension when declaring the service in protobuf:
```protobuf
import "trpc/proto/trpc_options.proto";

service Greeter {
  rpc Hello (HelloRequest) returns (HelloReply) {
    option (trpc.alias) = "/alias/hello";
  }
}
```
The curl command is as follows:
```bash
$ curl -XPOST -H"Content-Type: application/json" -d'{"msg": "world"}' 127.0.0.1:8000/alias/hello
{"msg":"Hello world!"}
```

#### RESTful HTTP

Although it is convenient to quickly convert RPC to HTTP, it is impossible to customize the mapping relationship between HTTP url/parameter/body and RPC Request. RESTful provides a more flexible RPC to HTTP conversion.

You can refer to [RESTful example](/examples/features/restful) to get start quickly. For more details, please refer to the [documentation](/restful) of RESTful.

### Further Reading

tRPC-Go also supports other rich features, you can check their [documentations](/docs/user_guide) for more details.

