[English](README.md) | 中文

## 前言

tRPC 框架使用 PB 定义服务，但是服务提供基于 HTTP 协议的 REST 风格 API 仍然是一个广泛的需求。RPC 和 REST 的统一是一件不容易的事情，tRPC-Go 框架本身的 HTTP RPC 协议，就是希望可以做到定义同一套 PB 文件，提供的服务既可以通过 RPC 方式调用（即通过桩代码提供的客户端 NewXXXClientProxy 调用），也可以通过原生 HTTP 请求调用，但这样的 HTTP 调用是不满足 RESTful 规范的，譬如说：无法自定义路由，不支持通配符，报错时 response body 为空（错误信息只能塞到 response header 里）等。所以我们额外支持了 RESTful 协议，而且不再尝试强行统一 RPC 和 REST，如果服务指定为 RESTful 协议，则其不支持用桩代码调用，仅支持 http 客户端调用，但是获得的好处是可以在同一套 PB 文件中通过 protobuf annotation 提供满足 RESTful 规范的 API，而且可以使用 tRPC 框架各种插件的能力。

## 原理

### 转码器

和 tRPC-Go 框架其他协议插件不同的是，RESTful 协议插件在 Transport 层就基于 tRPC HttpRule 实现了一个 tRPC 和 HTTP/JSON 的转码器，这样就不再需要走 Codec 编解码的流程，转码完成得到 PB 后直接到 trpc 工具为其专门生成的 REST Stub 中进行处理：

```ascii
                                                                +----------------------+
                                                                |     tRPC Server      |
                                                                |                      |
                                                                |       +------+       |
                          +---------------------------------------------> Stub |       |
+---------------------+   |   +------------------------------+  |       +------+       |
| HTTPRule Annotation +-+ |   |   RESTful Server Transport   |  |                      |
+---------------------+ | |   |        +-----------+         |  |   +-------------+    |
                        +-+------------> REST Stub |         |  |   | Code Plugin |    |
+---------------------+ |     |        +---+---^---+         |  |   +-------------+    |
|       PB File       +-+     | +----------v---+-----------+ |  | +------------------+ |
+---------------------+       | |                          +------>                  | |
                              | | tRPC/HTTPJSON Transcoder | |  | | Transport Plugin | |
                              | |                          <------+                  | |
                              | +--------------------------+ |  | +------------------+ |
                              +------------------------------+  +----------------------+
```

### 转码器核心：HttpRule

同一套 PB 定义的服务，既要支持 RPC 调用，也要支持 REST 调用，需要一套规则来指明 RPC 和 REST 之间的映射，更确切的是：PB 和 HTTP/JSON 之间的转码。在业界，Google 定义了一套这样的规则，即 `HttpRule`，tRPC 的实现也参考了这个规则。tRPC 的 HttpRule 需要你在 PB 文件中以 Options 的方式指定：`option (trpc.api.http)`，这就是所谓的同一套 PB 定义的服务既支持 RPC 调用也支持 REST 调用。

下面，我们来看一个例子，如何给一个 Greeter 服务中的 SayHello 方法绑定 HttpRule：

```protobuf
// Greeter service
import "trpc/api/annotations.proto";

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

 | HTTP | tRPC |
 | ----- | ----- |
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

 | HTTP | tRPC |
 | ----- | ----- |
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

 | HTTP | tRPC |
 | ----- | ----- |
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

 | HTTP | tRPC |
 | ----- | ----- |
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

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456 | GetMessage(message_id: "123456") |
 | GET /v1/users/me/messages/123456 | GetMessage(user_id: "me" message_id: "123456") |

## 实现

见 [trpc-go/restful 包](https://git.woa.com/trpc-go/trpc-go)

## 示例

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
    req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/v1/foobar", bytes.Newbuffer([]byte(`{"name": "xyz"}`)))
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

type FastHTTPErrorHandler func(context.Context, *fasthttp.RequestCtx, error)
```

用户可以通过 `WithOptions` 的方式定义错误处理：

```go
var xxxErrorHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
    if err == errors.New("say hello failed") {
        w.WriteHeader(500)
    }
    ...
}
// FastHTTP 使用 WithFastHTTPErrorHandler
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

**十一、自定义 [FastHTTP]RespSerializerGetter**

此前，RESTful 通过以下步骤进行序列化方式的协商：

1. 读取 request 的 Accept header 来设置 response serializer
2. 若无，则读取 request 的 Content-Type header 来设置 response serializer
3. 若无，则设置为默认的 JSONPBSerializer

RESTful 现在提供 [FastHTTP]RespSerializerGetter 给服务端用户指定回包的序列化方式。如：

[码客需求](https://mk.woa.com/q/294398)：用户可以通过自定义 SerializerGetter 来实现服务端侧指定回包的序列化方式，而无需通过协商。

RESTful 服务的 [FastHTTP]SerializerGetter 定义如下：

```go
type RespSerializerGetter func(ctx context.Context, r *http.Request) Serializer

type FastHTTPRespSerializerGetter func(ctx context.Context, requestCtx *fasthttp.RequestCtx) Serializer
```

默认的 [FastHTTP]RespSerializerGetter 实现如下：

```go
var DefaultRespSerializerGetter = func(_ context.Context, r *http.Request) Serializer {
    s, ok := responseSerializer(r.Header[headerAccept])
    if !ok {
        s = requestSerializer(r.Header[headerContentType])
    }
    return s
}

var DefaultFastHTTPRespSerializerGetter = func(_ context.Context, requestCtx *fasthttp.RequestCtx) Serializer {
    s, ok := responseSerializer([]string{string(requestCtx.Request.Header.Peek(headerAccept))})
    if !ok {
        s = requestSerializer([]string{string(requestCtx.Request.Header.Peek(headerContentType))})
    }
    return s
}
```

用户可以通过 WithOptions 的方式设置 [FastHTTP]RespSerializerGetter:

```go
// 方式 1: 直接指定 [推荐]
s := server.New(
    // ...
    // FastHTTP 使用 WithFastHTTPRespSerializerGetter
    server.WithRESTOptions(restful.WithRespSerializerGetter(
        func(ctx context.Context, r *http.Request) restful.Serializer {
            return &restful.ProtoSerializer{}
        },)
    ),
)

// 方式 2: 使用 msg.SerializationType 进行协商 [不推荐]
// 需要用户对具体处理链路非常熟悉
s := server.New(
    // ...
    server.WithFilter(func(
        ctx context.Context, req interface{}, next filter.ServerHandleFunc,
    ) (rsp interface{}, err error) {
        msg := trpc.Message(ctx)
        msg.WithSerializationType(codec.SerializationTypePB)
        return
    }),
    server.WithRESTOptions(
        // FastHTTP 使用 WithFastHTTPRespSerializerGetter
        restful.WithRespSerializerGetter(
            func(ctx context.Context, r *http.Request) restful.Serializer {
                // Users need to maintain the mapping between
                // msg.SerializationType() and the corresponding serializer.Name().
                // GetSerializer returns the serializer using serializer.Name().
                var serializationTypeContentType = map[int]string{
                    codec.SerializationTypePB: "application/octet-stream",
                }

                // Get serializer
                // Note: If users specify the response serializer using msg.SerializationType(),
                // the following behavior will occur:
                // Since the value of codec.SerializationTypePB is 0,
                // when the user does not set the SerializationType,
                // the &ProtoSerializer{} will be chosen as the default serializer.
                msg := trpc.Message(ctx)
                st := msg.SerializationType()
                s := restful.GetSerializer(serializationTypeContentType[st])

                // Note: When a serializer is not obtained,
                // it is recommended to use DefaultRespSerializerGetter as a fallback.
                // In most cases, the failure to obtain a serializer 
                // is due to the user not having registered the serializer.
                if s == nil {
                    s = restful.DefaultRespSerializerGetter(ctx, r)
                    log.Warnf("the serializer %s not found, get the serializer %s by default",
                        serializationTypeContentType[st], s.Name())
                }
                return s
            },
        ),
    ),
)
```

**十二、支持忽略冗余参数的配置**

在使用 tRPC-Go 构建 RESTful 服务时，我们可能会遇到需要处理请求中包含未知或额外参数的情况。这些未知或额外参数是指那些在服务的 proto 文件中未定义的字段。例如，考虑以下服务定义：

```proto
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc.api.http) = {
        get:"/v1/messages/{message_id}"
    };
  }
}

message GetMessageRequest {
  string message_id = 1;
  int64 revision = 2; // Mapped to URL query parameter `revision`.
}
```

在这个例子中，对于请求 `GET /v1/messages/123456?revision=2`，`revision` 是一个已知参数，因为它在 `GetMessageRequest` 消息中定义了。然而，对于请求 `GET /v1/messages/123456?foo=anything`，`foo` 是一个未知参数，因为它没有在 `GetMessageRequest` 消息中定义。

默认情况下，tRPC-Go 会对这些未知参数进行严格检查，并在发现未知参数时返回错误。为了提高服务的灵活性，tRPC-Go 提供了一种配置选项，允许服务在遇到未知参数时选择忽略这些参数，而不是报错。

要配置 tRPC-Go 服务以忽略请求中的未知参数，您可以在创建服务时使用 `WithDiscardUnknownParams()` 方法。此方法接受一个布尔值参数：

> - `true`：开启忽略未知参数。当服务接收到包含未知参数的请求时，这些参数将被忽略，服务不会因此返回错误。
> - `false`：关闭忽略未知参数，默认值。服务将对请求中的所有参数进行严格检查，任何未知参数都会导致错误响应。

示例代码：

```go
s := server.New(
    // ...
    server.WithRESTOptions(
        // 设置为 true 时，服务将忽略请求中的未知参数，而不会因此报错
        restful.WithDiscardUnknownParams(true),
    ),
)
```

## 性能

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

| 模式 | QPS when P99 < 10ms |
| ---- | ---- |
| 基于 net/http | 16w |
| 基于 fasthttp | 25w |

### 启用 fasthttp

在创建服务之前调用 `transport.RegisterServerTransport(thttp.NewRESTServerTransport(true))` 将 "restful" 默认传输层覆盖为基于 fasthttp。

```golang
package main

import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/transport"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)
func main() {
    transport.RegisterServerTransport("restful", thttp.NewRESTServerTransport(true))
    s := trpc.NewServer()
    // ...
}
```

### 注意事项

- 获取 HTTP 请求头：开启 fasthttp 的情况之后，`fasthttp.RequestCtx` 不会被传入到 handle 函数，因此调用 `thttp.Head(ctx)` 无法获取 HTTP 的请求头。
需要在创建服务的时候使用 `server.WithRESTOptions` 来设置 `FastHTTPHeaderMatcher`  或者 `FastHTTPRespHandler` 才行。
如果在*进入 handle 函数之前*需要获取 HTTP 的请求头用于 API 版本控制、身份验证、缓存控制，路由、负载均衡则必须使用 `restful.WithFastHTTPHeaderMatcher`。
如果在*handle 函数之中*需要获取 HTTP 的请求头，可以考虑使用 `restful.WithFastHTTPHeaderMatcher` 将 `fasthttp.RequestCtx` 中的 Header 信息塞到桩代码的`context` 中。
如果在*handle 函数处理之后*需要获取 HTTP 的请求头用于编写回包处理逻辑，则可以使用 `restful.WithFastHTTPRespHandler`。
下面的代码向你展示了使用 `restful.WithFastHTTPHeaderMatcher` 在*进入 handle 函数之前*做身份验证的例子。

```golang
package main

import (
    "github.com/valyala/fasthttp"

    "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/restful"
    "git.code.oa.com/trpc-go/trpc-go/server"
    "git.code.oa.com/trpc-go/trpc-go/transport"
)

func main() {
    transport.RegisterServerTransport("restful", thttp.NewRESTServerTransport(true))
    s := trpc.NewServer(server.WithRESTOptions(restful.WithFastHTTPHeaderMatcher(func(
        ctx context.Context,
        requestCtx *fasthttp.RequestCtx,
        serviceName, methodName string,
    ) (context.Context, error) {
        // auth is a bool function used to verify id.
        if id := string(requestCtx.Request.Header.Peek("id-key")); !auth(id) {
            return ctx, fmt.Errorf("id %s does not pass authentication", id)
        }
        return ctx, nil 
    })))
}
```

## FAQ

### 为 RESTful 服务添加额外的自定义路由

RESTful 服务在 proto 文件中通过 http rule 指定了 PB 和 HTTP/JSON 之间的映射关系，这种映射存在局限性，比如无法处理 multipart/formdata 这种二进制格式的数据类型，框架建议为 RESTful 服务无法处理的路由创建额外的服务（对应新的端口）以进行支持。

示例如下：

配置：

```yaml
server: 
  ...
  service:                                         
    - name: trpc.test.helloworld.stdhttp
      ip: 127.0.0.1                            
      port: 12345                
      network: tcp                             
      protocol: http_no_protocol
      timeout: 1000
    - name: trpc.test.helloworld.Greeter
      ip: 127.0.0.1                            
      port: 54321                
      network: tcp                             
      protocol: restful              
      timeout: 1000
```

代码：

```go
package main

import (
    "net/http"

    pb "git.woa.com/some/path/to/your/stub/helloworld"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    s := trpc.NewServer()
    // 注册 RESTful 服务（基于 fasthttp 的可以将下行进行相应替换）
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    
    // 注册泛 HTTP 标准服务
    thttp.RegisterNoProtocolServiceMux(
        s.Service("trpc.test.hello.stdhttp"),
        http.HandlerFunc(handle),
    )

    // 启动
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) {
    // 对 RequestURI 进行自定义解析以及判断处理
    uri := r.RequestURI
    if match(uri) { /*..*/ }

    r.ParseMultipartForm(0) // 解析 multipart/formdata
    // 通过访问 r.MultipartForm 来获取收到的文件等
}
```

### 在同一端口上为 RESTful 服务添加额外的自定义路由

**注：** 推荐将 RESTful 无法处理的路由单独分为另外一个服务，使用额外的端口（见上一小节），而非使用本小节的做法。

我们可以通过框架提供的 `restful.Get/RegisterRouter` 来取出已经注册的 restful router，在上面进行一层额外的封装以添加额外的自定义路由。

以下分为基于 stdhttp 和 fasthttp 两部分进行使用上的介绍。

#### 基于 stdhttp

示例如下（完整示例见 `router_test.go` 中的 `TestRegisterRouterAddAdditionalPatternUsingServerMux`）：

```go
s := server.New(
    server.WithListener(l),
    server.WithServiceName(serviceName),
    server.WithNetwork("tcp"),
    server.WithProtocol("restful"),
)
pb.RegisterGreeterService(s, &greeter{})

// 1. Get the old stdhttp router.
r := restful.GetRouter(serviceName)
// 2. Create a new stdhttp router.
mux := http.NewServeMux()
// 3. Pass the old stdhttp router as the "/*" for the new fasthttp router.
mux.Handle("/", r)
// 4. Register an additional pattern to the new stdhttp router.
additionalPattern := "/path"
dataForAdditionalPattern := []byte("data")
mux.Handle(additionalPattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // You may use `r.ParseMultipartForm(1024)` to parse 'multipart/formdata' here.
    w.Write(dataForAdditionalPattern)
}))
// 5. Register the new stdhttp router to replace the original one.
restful.RegisterRouter(serviceName, mux)

s.Serve()
```

要点：在 `pb.RegisterXxx` 之后，在 `s.Serve` 之前添加如下代码：

```go
r := restful.GetRouter(serviceName)
mux := http.NewServeMux()
mux.Handle("/", r)
additionalPattern := "/path"
dataForAdditionalPattern := []byte("data")
mux.Handle(additionalPattern, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
    w.Write(dataForAdditionalPattern)
}))
restful.RegisterRouter(serviceName, mux)
```

其中 `http.NewServeMux` 可以替换为任意形式的 mux，比如 [gorilla/mux](https://github.com/gorilla/mux), [gin](https://github.com/gin-gonic/gin) 等。

#### 基于 fasthttp

**要求 trpc-go 框架版本 >= v0.17.0**

示例与基于 stdhttp 的基本类似，使用的是 `restful.Get/RegisterFasthttpRouter`
（完整示例见 `router_test.go` 中的 `TestRegisterFasthttpRouterAddAdditionalPatternUsingServerMux`）：

```go
import frouter "github.com/fasthttp/router"
s := server.New(
    server.WithListener(l),
    server.WithServiceName(serviceName),
    server.WithNetwork("tcp"),
    server.WithProtocol(restfulProtocolBasedOnFasthttp))
pb.RegisterGreeterService(s, &greeter{})

// 1. Get the old fasthttp router.
r := restful.GetFasthttpRouter(serviceName)
// 2. Create a new fasthttp router.
fr := frouter.New()
// 3. Pass the old fasthttp router as the "/*" for the new fasthttp router.
fr.Handle(frouter.MethodWild, "/{filepath:*}", r)
// 4. Register an additional pattern to the new fasthttp router.
additionalPattern := "/path"
dataForAdditionalPattern := []byte("data")
fr.Handle(http.MethodGet, additionalPattern, func(ctx *fasthttp.RequestCtx) {
    // You may use `ctx.MultipartForm()` to access 'multipart/formdata' here.
    ctx.Response.BodyWriter().Write(dataForAdditionalPattern)
})
// 5. Register the new fasthttp router to replace the original one.
restful.RegisterFasthttpRouter(serviceName, fr.Handler)

s.Serve()
```

其中 `frouter.New()` 需要使用到 [github.com/fasthttp/router](https://github.com/fasthttp/router)。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
