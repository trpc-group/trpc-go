[TOC]





# 前言

tRPC 框架使用 PB 定义服务，但是服务提供基于 HTTP 协议的 REST 风格 API 仍然是一个广泛的需求。RPC 和 REST 的统一是一件不容易的事情，tRPC-Go 框架本身的 HTTP RPC 协议，就是希望可以做到定义同一套 PB 文件，提供的服务既可以通过 RPC 方式调用（即通过桩代码提供的客户端 NewXXXClientProxy 调用），也可以通过原生 HTTP 请求调用，但这样的 HTTP 调用是不满足 RESTful 规范的，譬如说：无法自定义路由，不支持通配符，报错时 response body 为空（错误信息只能塞到 response header 里）等。所以我们额外支持了 RESTful 协议，而且不再尝试强行统一 RPC 和 REST，如果服务指定为 RESTful 协议，则其不支持用桩代码调用，仅支持 http 客户端调用，但是获得的好处是可以在同一套 PB 文件中通过 protobuf annotation 提供满足 RESTful 规范的 API，而且可以使用 tRPC 框架的各种 插件/filter 能力。

# 原理

## 转码器

和 tRPC-Go 框架其他协议插件不同的是，RESTful 协议插件在 Transport 层就基于 tRPC HttpRule 实现了一个 tRPC 和 HTTP/JSON 的转码器，这样就不再需要走 Codec 编解码的流程，转码完成得到 PB 后直接到 trpc 工具为其专门生成的 REST Stub 中进行处理：

![restful-overall-design](/.resources/user_guide/server/restful/restful-overall-design_zh_CN.png)

## 转码器核心：HttpRule

同一套 PB 定义的服务，既要支持 RPC 调用，也要支持 REST 调用，需要一套规则来指明 RPC 和 REST 之间的映射，更确切的是：PB 和 HTTP/JSON 之间的转码。在业界，Google 定义了一套这样的规则，即 `HttpRule`，tRPC 的实现也参考了这个规则。tRPC 的 HttpRule 需要你在 PB 文件中以 Options 的方式指定：`option (trpc.api.http)`，这就是所谓的同一套 PB 定义的服务既支持 RPC 调用也支持 REST 调用。

下面，我们来看一个例子，如何给一个 Greeter 服务中的 SayHello 方法绑定 HttpRule：

```protobuf
// Greeter service
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option (trpc.api.http) = {
      post: "/v1/foobar/{name}"
      body: "*"
      additional_bindings: {
        post: "/v1/foo/{name=/x/y/**}"
        body: "single_nested"
        response_body: "message"
      }
    };
  }
}
// Hello Request
message HelloRequest {
  string name = 1;
  Nested single_nested = 2;
  oneof oneof_value {
    google.protobuf.Empty oneof_empty = 3;
    string oneof_string = 4;
  }
}
// Nested
message Nested {
  string name = 1;
}
// Hello Response
message HelloReply {
  string message = 1;
}
```

通过上述例子，可见 HttpRule 有以下几个字段：

> - selector 字段，表明要注册的 RESTful 路由，格式为 [ HTTP 动词小写 ] : [ URL Path ]。
> - body 字段，表明 HTTP 请求 Body 中携带的是 PB 请求 Message 的哪个字段。
> - response_body 字段，表明 HTTP 响应 Body 中携带的是 PB 响应 Message 的哪个字段。
> - additional_bindings 字段，表示额外的 HttpRule，即一个 RPC 方法可以绑定多个 HttpRule。

**结合 HttpRule 的具体规则看一下上述例子中 HTTP 请求/响应 怎么映射到 HelloRequest 和 HelloReply 中：**

> 映射时 RPC 请求 Proto Message 里的 **"叶子字段"** （所谓叶子字段，即不能再继续嵌套遍历的字段，上述例子中 HelloRequest.Name 是叶子字段，HelloRequest.SingleNested 不是叶子字段，HelloRequest.SingleNested.Name 才是）分三种情况映射：
>
> - 叶子字段被 HttpRule 的 URL Path 引用：HttpRule 的 URL Path 引用了 RPC 请求 Message 中的一个或多个字段，则 RPC 请求 Message 的这些字段就通过 HTTP 请求 URL Path 传递。但这些字段必须是原生基础类型的非数组字段，不支持消息类型的字段，也不支持数组字段。在上述例子中，HttpRule selector 字段被定义为 post: "/v1/foobar/{name}"，则 HTTP 请求：POST /v1/foobar/xyz 会把 HelloRequest.Name 字段值映射为 "xyz" 。
> - 叶子字段被 HttpRule 的 Body 引用：HttpRule 的 Body 里指明了映射的字段，则 RPC 请求 Message 的这个字段就通过 HTTP 请求 Body 传递。上述例子中，如果 HttpRule body 字段定义为 body: "name"，则 HTTP 请求 Body: "xyz" 把 HelloRequest.Name 字段值映射为 "xyz"
> - 其他叶子字段：其他叶子字段都会自动成为 URL 查询参数，而且如果是 repeated 字段，则支持同一个 URL 查询参数多次查询。上述例子中，additional_bindings 里面 selector 如果指定了 post: "/v1/foo/{name=/x/y/**}"，body 如果不指定 body: ""，则 HelloRequest 里面除了 HelloRequest.Name 字段外的字段都通过 URL 查询参数传递，譬如说，HTTP 请求 POST /v1/foo/x/y/z/xyz?single_nested.name=abc 会把 HelloRequest.Name 字段值映射为 "/x/y/z/xyz"，HelloRequest.SingleNested.Name 字段值映射为 "abc"。
>
> **补充：**
>
> - 如果 HttpRule 的 Body 里未指明字段，用 "*" 来定义，则没有被 URL Path 绑定的每个请求 Message 字段都通过 HTTP 请求的 Body 传递。即 URL 查询参数会失效。
> - 如果 HttpRule 的 Body 为空，则没有被 URL Path 绑定的每个请求 Message 字段都会自动成为 URL 查询参数。即 Body 失效。
> - 如果 HttpRule 的 response_body 为空，则整个 PB 响应 Message 会序列化到 HTTP 响应 Body 里，上述例子中，response_body: ""，则 HTTP Response Body 是整个 HelloReply 的序列化
> - HttpRule body 和 response_body 字段若要引用 PB Message 的字段，可以是叶子字段，也可以不是，但必须是 PB Message 里面的第一层的字段，譬如对于 HelloRequest，可以定义 HttpRule body: "name"，也可以定义 body: "single_nested"，但不能定义 body: "single_nested.name"

下面我们再看几个例子，能更好地理解 HttpRule 到底要怎么使用：

**一、将 URL Path 里面匹配 messages/\* 的内容作为 name 字段值：**

```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc.api.http) = {
        get: "/v1/{name=messages/*}"
    };
  }
}
message GetMessageRequest {
  string name = 1; // Mapped to URL path.
}
message Message {
  string text = 1; // The resource content.
}
```
上述 HttpRule 可得以下映射：

| HTTP                    | tRPC                                |
| ----------------------- | ----------------------------------- |
| GET /v1/messages/123456 | GetMessage(name: "messages/123456") |

**二、较为复杂的嵌套 message 构造，URL Path 里的 123456 作为 message_id，sub.subfield 的值作为嵌套 message 里的 subfield：**

```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc.api.http) = {
        get:"/v1/messages/{message_id}"
    };
  }
}
message GetMessageRequest {
  message SubMessage {
    string subfield = 1;
  }
  string message_id = 1; // Mapped to URL path.
  int64 revision = 2;    // Mapped to URL query parameter `revision`.
  SubMessage sub = 3;    // Mapped to URL query parameter `sub.subfield`.
}
```

上述 HttpRule 可得以下映射：

| HTTP                                                | tRPC                                                                          |
| --------------------------------------------------- | ----------------------------------------------------------------------------- |
| GET /v1/messages/123456?revision=2&sub.subfield=foo | GetMessage(message_id: "123456" revision: 2 sub: SubMessage(subfield: "foo")) |

**三、将 HTTP Body 的整体作为 Message 类型解析，即将 "Hi!" 作为 message.text 的值：**

```protobuf
service Messaging {
  rpc UpdateMessage(UpdateMessageRequest) returns (Message) {
    option (trpc.api.http) = {
      post: "/v1/messages/{message_id}"
      body: "message"
    };
  }
}
message UpdateMessageRequest {
  string message_id = 1; // mapped to the URL
  Message message = 2;   // mapped to the body
}
```


上述 HttpRule 可得以下映射：

| HTTP                                       | tRPC                                                        |
| ------------------------------------------ | ----------------------------------------------------------- |
| POST /v1/messages/123456 { "text": "Hi!" } | UpdateMessage(message_id: "123456" message { text: "Hi!" }) |

**四、将 HTTP Body 里的字段解析为 Message 的 text 字段：**

```protobuf
service Messaging {
  rpc UpdateMessage(Message) returns (Message) {
    option (trpc.api.http) = {
      post: "/v1/messages/{message_id}"
      body: "*"
    };
  }
}
message Message {
  string message_id = 1;
  string text = 2;
}
```

上述 HttpRule 可得以下映射：

| HTTP                                      | tRPC                                            |
| ----------------------------------------- | ----------------------------------------------- |
| POST/v1/messages/123456 { "text": "Hi!" } | UpdateMessage(message_id: "123456" text: "Hi!") |

**五、使用 additional_bindings 表示追加绑定的 API：**

```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc.api.http) = {
      get: "/v1/messages/{message_id}"
      additional_bindings {
        get: "/v1/users/{user_id}/messages/{message_id}"
      }
    };
  }
}
message GetMessageRequest {
  string message_id = 1;
  string user_id = 2;
}
```

上述 HttpRule 可得以下映射：

| HTTP                             | tRPC                                           |
| -------------------------------- | ---------------------------------------------- |
| GET /v1/messages/123456          | GetMessage(message_id: "123456")               |
| GET /v1/users/me/messages/123456 | GetMessage(user_id: "me" message_id: "123456") |

# 实现

见 [trpc-go/restful 包](https://git.woa.com/trpc-go/trpc-go)

# 示例

理解了 HttpRule 后，我们来看一下具体要如何开启 tRPC-Go 的 RESTful 服务。

**一、PB 定义**

先更新 `trpc-go-cmdline` 工具到最新版本，要使用 **trpc.api.http** 注解，需要 import 一个 proto 文件：

```protobuf
import "trpc/api/annotations.proto";
```

我们还是定义一个 Greeter 服务 的 PB:

```protobuf
...
import "trpc/api/annotations.proto";
// Greeter service
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option (trpc.api.http) = {
      post: "/v1/foobar"
      body: "*"
      additional_bindings: {
        post: "/v1/foo/{name}"
      }
    };
  }
}
// Hello Request
message HelloRequest {
  string name = 1;
  ...
}  
...
```

**二、生成桩代码**

直接用 `trpc create` 命令生成桩代码。

**三、配置**

和其他协议配置一样，`trpc_go.yaml` 里面 service 的 protocol 配置成 `restful` 即可

```yaml
server: 
  ...
  service:                                         
    - name: trpc.test.helloworld.Greeter      
      ip: 127.0.0.1                            
      # nic: eth0
      port: 8080                
      network: tcp                             
      protocol: restful              
      timeout: 1000
```

更普遍的场景是，我们会配置一个 tRPC 协议的 service，再加一个 RESTful 协议的 service，这样就能做到一套 PB 文件同时支持提供 RPC 服务和 RESTful 服务：

```yaml
server: 
  ...
  service:                                         
    - name: trpc.test.helloworld.Greeter1      
      ip: 127.0.0.1                            
      # nic: eth0
      port: 12345                
      network: tcp                             
      protocol: trpc              
      timeout: 1000
    - name: trpc.test.helloworld.Greeter2      
      ip: 127.0.0.1                            
      # nic: eth0
      port: 54321                
      network: tcp                             
      protocol: restful              
      timeout: 1000
```

**注意：tRPC 每个 service 必须配置不同的端口。**

**四、启动服务**

启动服务和其他协议方式一致：

```go
package main
import (
    ...
    pb "git.code.oa.com/trpc-go/trpc-go/examples/restful/helloworld"
)
func main() {
    s := trpc.NewServer()
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    // 启动
      if err := s.Serve(); err != nil {
          ...
      }
}
```
**五、调用**

搭建的是 RESTful 服务，所以请用任意的 REST 客户端调用，不支持用 NewXXXClientProxy 的 RPC 方式调用：

```go
package main
import "net/http"
func main() {
    ...
    // native HTTP invocation
    req, err := http.NewRequest("POST", "http://127.0.0.1:8080/v1/foobar", bytes.Newbuffer([]byte(`{"name": "xyz"}`)))
    if err != nil {
        ...
    }
    cli := http.Client{}
    resp, err := cli.Do(req)
    if err != nil {
        ...
    }
    ...
}
```

当然如果上面第三点【配置】中，如果配置了 tRPC 协议的 service，我们还是可以通过 NewXXXClientProxy 的 RPC 方式去调用 tRPC 协议的 service，注意区分端口。

**六、自定义 HTTP 头到 RPC Context 映射**

HttpRule 解决的是 tRPC Message Body 和 HTTP/JSON 之间的转码，那么 HTTP 请求如何传递 RPC 调用的上下文呢？这就需要定义 HTTP 头到 RPC Context 映射。

RESTful 服务的 HeaderMatcher 定义如下：

```go
type HeaderMatcher func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    serviceName, methodName string,
) (context.Context, error)
```

默认的 HeaderMatcher 处理如下：

```go
var defaultHeaderMatcher = func(
    ctx context.Context,
    w http.ResponseWriter,
    req *http.Request,
    serviceName, methodName string,
) (context.Context, error) {
    // It is recommended to customize and pass the codec.Msg in the ctx, and specify the target service and method name.
    ctx, msg := codec.WithNewMessage(ctx)
    msg.WithCalleeServiceName(service)
    msg.WithServerRPCName(method)
    msg.WithSerializationType(codec.SerializationTypePB)
    return ctx, nil
}
```

用户可以通过 `WithOptions` 的方式设置 HeaderMatcher：

```go
service := server.New(server.WithRESTOptions(restful.WithHeaderMatcher(xxx)))
```

**七、自定义回包处理 [设置请求处理成功的返回码]**

HttpRule 的 response_body 字段指定了 RPC 响应，譬如上面例子中的 HelloReply 要整个或者将其某个字段序列化到 HTTP Response Body 里面。但是用户可能想额外做一些自定义的操作，譬如：设置成功时候的响应码。

RESTful 服务的自定义回包处理函数定义如下：

```go
type CustomResponseHandler func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    resp proto.Message,
    body []byte,
) error
```

trpc-go/restful 包提供了一个让用户设置请求处理成功时候的响应码的函数：

```go
func SetStatusCodeOnSucceed(ctx context.Context, code int) {}
```

默认的自定义回包处理函数如下：

```go
var defaultResponseHandler = func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    resp proto.Message,
    body []byte,
) error {
    // compress
    var writer io.Writer = w
    _, compressor := compressorForRequest(r)
    if compressor != nil {
        writeCloser, err := compressor.Compress(w)
        if err != nil {
            return fmt.Errorf("failed to compress resp body: %w", err)
        }
        defer writeCloser.Close()
        w.Header().Set(headerContentEncoding, compressor.ContentEncoding())
        writer = writeCloser
    }
    // Set StatusCode
    statusCode := GetStatusCodeOnSucceed(ctx)
    w.WriteHeader(statusCode)
    // Set body
    if statusCode != http.StatusNoContent && statusCode != http.StatusNotModified {
        writer.Write(body)
    }
    return nil
}
```

如果使用默认自定义回包处理函数，则支持用户在自己的 RPC 处理函数中设置返回码（不设置则成功返回 200）：

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) (err error) {   
    ...
    restful.SetStatusCodeOnSucceed(ctx, 200) // Set the return code for success.
    return nil
}
```

用户可以通过 `WithOptions` 的方式定义回包处理：

```go
var xxxResponseHandler = func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    resp proto.Message,
    body []byte,
) error {
    reply, ok := resp.(*pb.HelloReply)
    if !ok {
        return errors.New("xxx")
    }
    ...
    w.Header().Set("x", "y")
    expiration := time.Now()
    expiration := expiration.AddDate(1, 0, 0)
    cookie := http.Cookie{Name: "abc", Value: "def", Expires: expiration}
    http.SetCookie(w, &cookie)
    w.Write(body)
    return nil
}
...
service := server.New(server.WithRESTOptions(restful.WithResponseHandler(xxxResponseHandler)))
```

**八、自定义错误处理 [错误码]**

RESTful 错误处理函数定义如下：

```go
type ErrorHandler func(context.Context, http.ResponseWriter, *http.Request, error)
```

用户可以通过 `WithOptions` 的方式定义错误处理：

```
var xxxErrorHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
    if err == errors.New("say hello failed") {
        w.WriteHeader(500)
    }
    ...
}
service := server.New(server.WithRESTOptions(restful.WithErrorHandler(xxxErrorHandler)))
```

**建议使用 trpc-go/restful 包默认的错误处理函数，或者参考实现用户自己的错误处理函数。**

关于**错误码：**

如果 RPC 处理过程中返回了 trpc-go/errs 包定义的错误类型，trpc-go/restful 默认的错误处理函数会将 tRPC 的错误码都映射为 HTTP 错误码。如果用户想自己决定返回的某个错误用什么错误码，请使用 trpc-go/restful 包定义的 `WithStatusCode` :

```go
type WithStatusCode struct {
    StatusCode int
    Err        error
}
```

将自己的 error 包起来并返回，如：

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *hpb.HelloRequest, rsp *hpb.HelloReply) (err error) {
    if req.Name != "xyz" {
        return &restful.WithStatusCode{
            StatusCode: 400,
            Err:        errors.New("test error"),
        }
    }
    return nil
}
```

如果错误类型不是 trpc-go/errs 的 Error 类型，也没用 trpc-go/restful 包定义的 `WithStatusCode` 包起来，则默认错误码返回 500。

**九、Body 序列化与压缩**

和普通 REST 请求一样，通过 HTTP 头指定，支持比较主流的几个。

> **序列化支持的 Content-Type (或 Accept)：application/json，application/x-www-form-urlencoded，application/octet-stream。默认为 application/json。**

序列化接口定义如下：

```go
type Serializer interface {
    // Marshal serializes the tRPC message or one of its fields into the HTTP body. 
    Marshal(v interface{}) ([]byte, error)
    // Unmarshal deserializes the HTTP body into the tRPC message or one of its fields. 
    Unmarshal(data []byte, v interface{}) error
    // Name Serializer Name
    Name() string
    // ContentType  is set when returning the HTTP response.
    ContentType() string
}
```

**用户可自己实现并通过 `restful.RegisterSerializer()` 函数注册。**

> **压缩支持 Content-Encoding (或 Accept-Encoding): gzip。默认不压缩。**

压缩接口定义如下：

```go
type Compressor interface {
    // Compress 
    Compress(w io.Writer) (io.WriteCloser, error)
    // Decompress 
    Decompress(r io.Reader) (io.Reader, error)
    // Name represents the name of the compressor.
    Name() string
    // ContentEncoding represents the Content-Encoding that is set when returning the HTTP response.
    ContentEncoding() string
}
```

**用户可自己实现并通过 `restful.RegisterCompressor()` 函数注册。**

**十、跨域请求**

RESTful 也支持 [trpc-filter/cors](https://git.woa.com/trpc-go/trpc-filter/tree/master/cors) 跨域插件。使用时，需要在先 pb 中通过 [`custom`](https://git.woa.com/trpc/trpc-protocol/blob/v0.2.1/trpc/api/http.proto#L37) 添加 HTTP OPTIONS 方法，比如：

```go
service HelloTrpcGo {
  rpc Hello(HelloReq) returns (HelloRsp) {
    option (trpc.api.http) = {
      post: "/hello"
      body: "*"
      additional_bindings: {
        get: "/hello/{name}"
      }
      additional_bindings: {
        custom: { // use custom verb
          kind: "OPTIONS"
          path: "/hello"
        }
      }
    };
  }
}
```

然后，通过 [trpc](https://git.woa.com/trpc-go/trpc-go-cmdline)(>= v0.7.5) 命令行工具重新生成桩代码。
最后，在 service 拦载器中配上 CORS 插件。

如果不想修改 pb。RESTful 也提供了代码自定义跨域的方式。
RESTful 协议插件会为每个 Service 生成一个对应的 http.Handler，我们可以在启动监听前取出来，替换成我们自己的 http.Handler：

```go
func allowCORS(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if origin := r.Header.Get("Origin"); origin != "" {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
                preflightHandler(w, r)
                return
            }
        }
        h.ServeHTTP(w, r)
    })
}
func main() {
    // set custom header matcher
    s := trpc.NewServer()
    //  register service implementation
    pb.RegisterPingService(s, &pingServiceImpl{})
    // retrieve restful.Router
    router := restful.GetRouter(pb.PingServer_ServiceDesc.ServiceName)
    // wrap it up and re-register it again
    restful.RegisterRouter(pb.PingServer_ServiceDesc.ServiceName, allowCORS(router))
    // start
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

# 性能

为了提升性能，RESTful 协议插件额外支持基于 [fasthttp](https://github.com/valyala/fasthttp) 来处理 HTTP 包，RESTful 协议插件性能和注册的 URL 路径复杂度有关，和通过哪种方式传递 PB Message 字段也有关，这里仅给出最简单的 echo 测试场景下两种模式的对比：

测试 PB:

```go
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option (trpc.api.http) = {
      get: "/v1/foobar/{name}"
    };
  }
}
message HelloRequest {
  string name = 1;
}
message HelloReply {
  string message = 1;
}
```

Greeter 实现：

```go
type greeterServiceImpl struct{}
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    rsp.Message = req.Name
    return nil
}
```

测试机器：绑定 8 核

| 模式          | QPS when P99 < 10ms |
| ------------- | ------------------- |
| 基于 net/http | 16w                 |
| 基于 fasthttp | 25w                 |

- fasthttp 开启方式：代码里加一行（加在 `trpc.NewServer()` 前）：

```go
package main
import (
    "git.code.oa.com/trpc-go/trpc-go/transport"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)
func main() {
    transport.RegisterServerTransport("restful", thttp.NewRESTServerTransport(true))
    s := trpc.NewServer()
    ...
}
```

# FAQ

todo

