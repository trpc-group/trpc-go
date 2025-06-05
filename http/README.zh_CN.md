- [tRPC-Go HTTP 协议](#trpc-go-http-协议)
  - [泛 HTTP 标准服务](#泛-http-标准服务)
    - [服务端](#服务端)
      - [配置编写](#配置编写)
      - [代码编写](#代码编写)
        - [单一 URL 注册](#单一-url-注册)
        - [MUX 注册](#mux-注册)
    - [客户端](#客户端)
      - [配置编写](#配置编写-1)
      - [代码编写](#代码编写-1)
  - [泛 HTTP RPC 服务](#泛-http-rpc-服务)
    - [服务端](#服务端-1)
      - [配置编写](#配置编写-2)
      - [代码编写](#代码编写-2)
      - [自定义 URL path](#自定义-url-path)
      - [自定义错误码处理函数](#自定义错误码处理函数)
    - [客户端](#客户端-1)
      - [配置编写](#配置编写-3)
      - [代码编写](#代码编写-3)
  - [HTTP 连接池配置](#http-连接池配置)
    - [配置编写](#配置编写-4)
    - [代码编写](#代码编写-4)
  - [FAQ](#faq)
    - [客户端及服务端开启 HTTPS](#客户端及服务端开启-https)
      - [双向认证](#双向认证)
        - [仅配置填写](#仅配置填写)
        - [仅代码填写](#仅代码填写)
      - [不认证客户端证书](#不认证客户端证书)
        - [仅配置填写](#仅配置填写-1)
        - [仅代码填写](#仅代码填写-1)
    - [客户端使用 io.Reader 进行流式发送文件](#客户端使用-ioreader-进行流式发送文件)
    - [客户端使用 io.Reader 进行流式读取回包](#客户端使用-ioreader-进行流式读取回包)
    - [收发 SSE](#收发-sse)
    - [收发 SSE (基于 github.com/r3labs/sse )](#收发-sse-基于-githubcomr3labssse-)
    - [客户端做转发](#客户端做转发)
    - [客户端服务端收发 HTTP chunked](#客户端服务端收发-http-chunked)
    - [客户端发送任意 Content-Type 的数据](#客户端发送任意-content-type-的数据)
    - [客户端提交 Form 数据](#客户端提交-form-数据)
      - [提交 Content-Type 为 `application/x-www-form-urlencoded` 的 Form 数据](#提交-content-type-为-applicationx-www-form-urlencoded-的-form-数据)
      - [提交 Content-Type 为 `multipart/form-data` 的 Form 数据](#提交-content-type-为-multipartform-data-的-form-数据)
    - [服务端接收文件上传（使用 `multipart/form-data`）](#服务端接收文件上传使用-multipartform-data)
    - [使用泛 HTTP 标准服务及客户端时，监控上报 req,rsp 为空](#使用泛-http-标准服务及客户端时监控上报-reqrsp-为空)
    - [收到的响应内容为空的原因](#收到的响应内容为空的原因)
    - [限制只接收 POST 方法的请求](#限制只接收-post-方法的请求)
    - [为 http\_no\_protocol 服务的每个 handler 提供各自的 timeout](#为-http_no_protocol-服务的每个-handler-提供各自的-timeout)
    - [对框架构造的 http.Request 做自定义修改（如修改 Content-Length）](#对框架构造的-httprequest-做自定义修改如修改-content-length)
    - [同时支持泛 HTTP 标准服务以及 RESTful 服务](#同时支持泛-http-标准服务以及-restful-服务)
    - [设置 GetSerialization 反序列化 query parameters 的行为](#设置-getserialization-反序列化-query-parameters-的行为)
    - [关于 value detached transport 导致的资源泄露问题](#关于-value-detached-transport-导致的资源泄露问题)

# tRPC-Go HTTP 协议

tRPC-Go 框架支持搭建与 HTTP 相关的三种服务：

1. 泛 HTTP 标准服务 (无需桩代码及 IDL 文件)
2. 泛 HTTP RPC 服务 (共享 RPC 协议使用的桩代码以及 IDL 文件)
3. 泛 HTTP RESTful 服务 (基于 IDL 及桩代码提供 RESTful API)

其中 RESTful 相关文档见 [restful](https://git.woa.com/trpc-go/trpc-go/tree/master/restful)

## 泛 HTTP 标准服务

tRPC-Go 框架提供了泛 HTTP 标准服务能力，主要是在标准库 HTTP 的能力上添加了服务注册、服务发现、拦截器等能力，使 HTTP 协议能够无缝接入 tRPC 生态

相较于 tRPC 协议而言，泛 HTTP 标准服务服务不依赖桩代码，因此服务侧对应的 protocol 名为 `http_no_protocol`

### 服务端

#### 配置编写

在 `trpc_go.yaml` 配置文件中配置 service，协议为 `http_no_protocol`，http2 则为 `http2_no_protocol`:

```yaml
server:
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp  # service 的路由名称
      network: tcp  # 网络监听类型，tcp 或 udp
      protocol: http_no_protocol  # 应用层协议 http_no_protocol
      timeout: 1000  # 请求最长处理时间，单位毫秒
      ip: 127.0.0.1
      port: 8080  # 服务监听端口
```

注意确保配置文件的正常加载

#### 代码编写

##### 单一 URL 注册

```go
import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/log"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    trpc "git.code.oa.com/trpc-go/trpc-go"
)

func main() {
    s := trpc.NewServer()
    thttp.HandleFunc("/xxx", handle) 
    // 注册 NoProtocolService 时传的参数必须和配置中的 service name 一致：s.Service("trpc.app.server.stdhttp")
    thttp.RegisterNoProtocolService(s.Service("trpc.app.server.stdhttp")) 
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // handle 的编写方法完全同标准库 HTTP 的使用方式一致
    // 比如可以在 r 中读取 Header 等
    // 可以在 r.Body 对 client 进行流式读包
    clientReq, err := io.ReadAll(r.Body)
    if err != nil { /*..*/ }
    // 最后使用 w 来进行回包
    w.Header().Set("Content-type", "application/text")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("response body"))
    return nil
}
```

##### MUX 注册

```go
import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/log"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "github.com/gorilla/mux"
)

func main() {
    s := trpc.NewServer()
    // 路由注册
    router := mux.NewRouter()
    router.HandleFunc("/{dir0}/{dir1}/{day}/{hour}/{vid:[a-z0-9A-Z]+}_{index:[0-9]+}.jpg", handle).
        Methods(http.MethodGet)
    // 注册 RegisterNoProtocolServiceMux 时传的参数必须和配置中的 service name 一致：s.Service("trpc.app.server.stdhttp")
    thttp.RegisterNoProtocolServiceMux(s.Service("trpc.app.server.stdhttp"), router)
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // 取 url 中的参数
    vars := mux.Vars(r)
    vid := vars["vid"]
    index := vars["index"]
    log.Infof("vid: %s, index: %s", vid, index)
    return nil
}
```

### 客户端

这里指的是调用一个标准 HTTP 服务，下游这个标准 HTTP 服务并不一定是基于 tRPC-Go 框架构建的

最简洁的方式实际上是直接使用标准库提供的 HTTP Client, 但是就无法使用服务发现以及各种插件拦截器提供的能力 (比如监控上报)

#### 配置编写

```yaml
client:  # 客户端调用的后端配置
  timeout: 1000  # 针对所有后端的请求最长处理时间
  namespace: Development  # 针对所有后端的环境
  filter:  # 针对所有后端调用函数前后的拦截器列表
    - simpledebuglog  # 这是 debug log 拦截器，可以再添加其他拦截器，比如监控等
  service:  # 针对单个后端的配置
    - name: trpc.app.server.stdhttp  # 下游 http 服务的 service name 
    ## 可以使用 target 来选用其他的 selector, 只有 service name 的情况下默认会使用北极星做服务发现 (在使用了北极星插件的情况下)
    #   target: polaris://trpc.app.server.stdhttp  # 或者 ip://127.0.0.1:8080 来指定 ip:port 进行调用
    #   ca_cert: "none" # CA 证书，不认证客户端证书时此处必须填写，并且要填 "none"
```

其中配置部分要注意假如访问的是 HTTPS 的话，需要加上 `ca_cert: "none"`（或指定齐全的证书文件），详情可参考 [客户端及服务端开启 HTTPS](#客户端及服务端开启-https)

#### 代码编写

```go
package main

import (
    "context"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
)

// Data 请求报文数据
type Data struct {
    Msg string
}

func main() {
    // 省略掉 tRPC-Go 框架配置加载部分，假如以下逻辑在某个 RPC handle 中，配置一般已经正常加载
    // 创建 ClientProxy, 设置协议为 HTTP 协议，序列化为 JSON
    httpCli := http.NewClientProxy("trpc.app.server.stdhttp",
        client.WithSerializationType(codec.SerializationTypeJSON))
    reqHeader := &http.ClientReqHeader{
        // 注：当使用了自定义的 ClientReqHeader 时，
        // 需要明确指定所需的 HTTP 方法 
        Method: http.MethodPost,
    }
    // 为 HTTP Head 添加 request 字段
    reqHeader.AddHeader("request", "test")
    rspHead := &http.ClientRspHeader{}
    req := &Data{Msg: "Hello, I am stdhttp client!"}
    rsp := &Data{}
    // 发送 HTTP POST 请求
    if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
    ); err != nil {
        log.Warn("get http response err")
        return
    }
    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

## 泛 HTTP RPC 服务

相较于**泛 HTTP 标准服务**, 泛 HTTP RPC 服务的最大区别是复用了 IDL 协议文件及其生成的桩代码，同时无缝融入了 tRPC 生态 (服务注册、服务路由、服务发现、各种插件拦截器等)

注意：

在这种服务形式下，HTTP 协议与 tRPC 协议保持一致：当服务端返回失败时，body 为空，错误码错误信息放在 HTTP header 里

### 服务端

#### 配置编写

首先需要生成桩代码：

```shell
trpc create -p helloworld.proto --protocol http -o out
```

假如本身已经是一个 tRPC 服务已经存在桩代码，只是想在同样的接口上支持 HTTP 协议，那么无需再次生成桩代码，而是在配置中添加 `http` 协议项即可

```yaml
server: # 服务端配置
  service:
    ## 同一套接口可以通过两份配置同时提供 trpc 协议以及 http 协议服务
    - name: trpc.test.helloworld.Greeter  # service 的路由名称
      ip: 127.0.0.0  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000  # 服务监听端口 可使用占位符 ${port}
      protocol: trpc  # 应用层协议 trpc http
    ## 以下为主要示例，注意应用层协议为 http
    - name: trpc.test.helloworld.GreeterHTTP  # service 的路由名称
      ip: 127.0.0.0  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8001  # 服务监听端口 可使用占位符 ${port}
      protocol: http  # 应用层协议 trpc http
```

#### 代码编写

```go
// Reference:
//   https://git.woa.com/cooperyan/trpc-go-in-a-nutshell
import (
    "context"
    "fmt"

    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    pb "git.woa.com/xxxx/helloworld/pb"
)

func main() {
    s := trpc.NewServer()
    hello := Hello{}
    pb.RegisterHelloTrpcGoService(s.Service("trpc.test.helloworld.Greeter"), &hello)
    // 和一般的 tRPC 服务注册是一致的
    pb.RegisterHelloTrpcGoService(s.Service("trpc.test.helloworld.GreeterHTTP"), &hello)
    log.Println(s.Serve())
}

type Hello struct {}

// RPC 服务接口的实现无需感知 HTTP 协议，只需按照通常的逻辑处理请求并返回响应即可
func (h *Hello) Hello(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
    fmt.Println("--- got HelloReq", req)
    time.Sleep(time.Second)
    return &pb.HelloRsp{Msg: "Welcome " + req.Name}, nil
}
```

#### 自定义 URL path

默认为 `/package.service/method`，可通过 alias 参数自定义任意 URL

- 协议定义：

```protobuf
syntax = "proto3";
package trpc.app.server;
option go_package="git.code.oa.com/trpcprotocol/app/server";

import "trpc.proto";

message Request {
    bytes req = 1;
}

message Reply {
    bytes rsp = 1;
}

service Greeter {
    rpc SayHello(Request) returns (Reply) {
        option (trpc.alias) = "/cgi-bin/module/say_hello";
    };
}
```

#### 自定义错误码处理函数

默认错误码处理函数，会将错误码填充到 HTTP header 的 `trpc-ret/trpc-func-ret` 字段中，也可以通过自己定义 ErrorHandler 进行替换。

```golang
import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/errs"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
        // 一般自行定义 retcode retmsg 字段，组成 json 并写到 response body 里
        w.Write([]byte(fmt.Sprintf(`{"retcode": %d, "retmsg": "%s"}`, e.Code, e.Msg)))
        // 每个业务团队可以定义到自己的 git 里，业务代码 import 进来即可
    }
}
```

### 客户端

#### 配置编写

和一般的 RPC Client 书写方式相同，只需把配置 `protocol` 改为 `http`:

```yaml
client:
  namespace: Development  # 针对所有后端的环境
  filter:  # 针对所有后端调用函数前后的拦截器列表
  service:  # 针对单个后端的配置
    - name: trpc.test.helloworld.GreeterHTTP  # 后端服务的 service name
      network: tcp  # 后端服务的网络类型 tcp udp
      protocol: http  # 应用层协议 trpc http
      ## 可以使用 target 来选用其他的 selector, 只有 service name 的情况下默认会使用北极星做服务发现 (在使用了北极星插件的情况下)
      # target: ip://127.0.0.1:8000  # 请求服务地址
      timeout: 1000  # 请求最长处理时间
```

#### 代码编写

```go
import (
    "context"
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
    pb "git.code.oa.com/trpcprotocol/test/rpchttp"
)

func main() {
    // 省略掉 tRPC-Go 框架配置加载部分，假如以下逻辑在某个 RPC handle 中，配置一般已经正常加载
    // 创建 ClientProxy, 设置协议为 HTTP 协议，序列化为 JSON
    proxy := pb.NewHelloClientProxy()
    reqHeader := &thttp.ClientReqHeader{}
    // 必须留空或设置为 "POST"
    reqHeader.Method = http.MethodPost
    // 为 HTTP Head 添加 request 字段
    reqHeader.AddHeader("request", "test")
    // 设置 Cookie
    cookie := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
    reqHeader.AddHeader("Cookie", cookie.String())
    req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    rspHead := &thttp.ClientRspHeader{}
    // 发送 HTTP RPC 请求
    rsp, err := proxy.SayHello(context.Background(), req,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
        // 此处可以使用代码强制覆盖 trpc_go.yaml 配置中的 target 字段来设置其他 selector, 一般没必要，这里只是展示有这个功能
        // client.WithTarget("ip://127.0.0.1:8000"),
    )
    if err != nil {
        log.Warn("get http response err")
        return
    }
    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

## HTTP 连接池配置

`HTTP Transport` 允许通过配置文件或者代码来设定连接池参数。

### 配置编写

通过配置文件设置连接池参数。

```yaml
client:
  service:
    - name: trpc.test.helloworld.GreeterHTTP
      protocol: http
      conn_type: httppool  # connection type is httppool, the following options are all for httppool.
      httppool:
        max_idle_conns: 100  # httppool: max number of idle connections, default 0 (means no limit).
        max_idle_conns_per_host: 10  # httppool: max number of idle connections per-host, default 2.
        max_conns_per_host: 20  # httppool: max number of connections, default 0 (means no limit).
        idle_conn_timeout: 1s  # httppool: idle timeout, default 0s (means no limit).
```

### 代码编写

通过 `client.WithHTTPRoundTripOptions` 设置 `transport.HTTPRoundTripOptions`，以配置 HTTP 连接池的相关参数。

```go
httpOpts := transport.HTTPRoundTripOptions{
    Pool: httppool.Options{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        MaxConnsPerHost:     20,
        IdleConnTimeout:     time.Second,
    },
}
proxy := pb.NewGreeterClientProxy(
    client.WithTarget("ip://127.0.0.1:8000"),
    client.WithProtocol("http"),
    client.WithHTTPRoundTripOptions(httpOpts),
)
```

## FAQ

### 客户端及服务端开启 HTTPS

分为双向认证以及单向认证，在使用框架时大部分是使用单向认证，构造一个 trpc-go HTTPS 的客户端去访问一个已存在的 HTTPS 服务

#### 双向认证

##### 仅配置填写

只需在 `trpc_go.yaml` 中添加相应的配置项 (证书以及私钥):

```yaml
server:  # 服务端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol # 泛 HTTP RPC 服务则填 http (v0.16.0 起此处可以填 https_no_protocol 或 https)
      tls_cert: "../testdata/server.crt" # 添加证书路径
      tls_key: "../testdata/server.key" # 添加私钥路径
      ca_cert: "../testdata/ca.pem" # CA 证书，需要双向认证时可填写
client:  # 客户端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http # v0.16.0 起此处可以填 https
      # 1. 证书/私钥/CA
      tls_cert: "../testdata/server.crt" # 添加证书路径
      tls_key: "../testdata/server.key" # 添加私钥路径
      ca_cert: "../testdata/ca.pem" # CA 证书，需要双向认证时可填写
      # 2. 将原本 https://some-example.com 的域名写到 dns://some-example.com 中作为 target
      #    直接访问 ip:port 时可以直接写 target: ip://x.x.x.x:xx
      target: dns://some-example.com  # 对应 curl "https://some-example.com"
```

代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作 (不需要指定 scheme 为 `https`, 不需要手动添加 `WithTLS` option, 也不需要在 `WithTarget` 等其他地方想办法塞一个有关 HTTPS 的标识进去)

##### 仅代码填写

服务端使用 `server.WithTLS` 依次指定服务端证书、私钥、CA 证书即可：

```go
server.WithTLS(
    "../testdata/server.crt",
    "../testdata/server.key",
    "../testdata/ca.pem",
),
```

客户端使用 `client.WithTLS` 依次指定客户端端证书、私钥、CA 证书即可：

```go
// 1. 证书/私钥/CA
client.WithTLS(
    "../testdata/client.crt",
    "../testdata/client.key",
    "../testdata/ca.pem",
    "localhost", // 填写 server name
),
// 2. 将原本 https://some-example.com 的域名写到 dns://some-example.com 中作为 target
client.WithTarget("dns://some-example.com")
//    直接访问 ip:port 时可以直接写 target: ip://x.x.x.x:xx
client.WithTarget("ip://x.x.x.x:xx")
```

除了这些 option 以外，代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作

示例如下：

```go
func TestHTTPSUseClientVerify(t *testing.T) {
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork("tcp"),
        server.WithProtocol("http_no_protocol"), // v0.16.0 起此处可以填 https_no_protocol
        server.WithListener(ln),
        server.WithTLS(
            "../testdata/server.crt",
            "../testdata/server.key",
            "../testdata/ca.pem",
        ),
    )
    thttp.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) error {
        w.Write([]byte(t.Name()))
        return nil
    })
    thttp.RegisterNoProtocolService(service)
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    req := &codec.Body{}
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithTLS(
                "../testdata/client.crt",
                "../testdata/client.key",
                "../testdata/ca.pem",
                "localhost",
            ),
        ))
    require.Equal(t, []byte(t.Name()), rsp.Data)
}
```

#### 不认证客户端证书

##### 仅配置填写

只需在 `trpc_go.yaml` 中添加相应的配置项 (证书以及私钥):

```yaml
server:  # 服务端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol # 泛 HTTP RPC 服务则填 http (v0.16.0 起此处可以填 https_no_protocol 或 https)
      tls_cert: "../testdata/server.crt" # 添加证书路径
      tls_key: "../testdata/server.key" # 添加私钥路径
      # ca_cert: "" # CA 证书，不认证客户端证书时此处不填或留空
client:  # 客户端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http # 从 v0.16.0 起，此处可以直接填写 https，并且不需要再指定 ca_cert 为 "none" 来开启 HTTPS
      # 1. 证书/私钥/CA
      # tls_cert: "" # 证书路径，不认证客户端证书时此处不填或留空
      # tls_key: "" # 私钥路径，不认证客户端证书时此处不填或留空
      ca_cert: "none" # CA 证书，不认证客户端证书时此处必须填写，并且要填 "none"
      # 2. 将原本 https://some-example.com 的域名写到 dns://some-example.com 中作为 target
      #    直接访问 ip:port 时可以直接写 target: ip://x.x.x.x:xx
      target: dns://some-example.com  # 对应 curl "https://some-example.com"
```

可以双向认证部分，主要的区别在于服务端的 `ca_cert` 需要留空，客户端的 `ca_cert` 需要填 `none`

代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作 (不需要指定 scheme 为 `https`, 不需要手动添加 `WithTLS` option, 也不需要在 `WithTarget` 等其他地方想办法塞一个有关 HTTPS 的标识进去)

**注**：从 v0.16.0 开始，用户可以直接在 protocol 字段上填写 `https` 以开启 HTTPS，不需要再指定 `ca_cert` 或其他的选项 参考 <https://mk.woa.com/note/7509>

##### 仅代码填写

服务端使用 `server.WithTLS` 依次指定服务端证书、私钥、CA 证书即可：

```go
server.WithTLS(
    "../testdata/server.crt",
    "../testdata/server.key",
    "", // CA 证书，不认证客户端证书时此处留空
),
```

客户端使用 `client.WithTLS` 依次指定客户端端证书、私钥、CA 证书即可：

```go
// 1. 证书/私钥/CA
client.WithTLS(
    "", // 证书路径，留空
    "", // 私钥路径，留空
    "none", // CA 证书，不认证客户端证书时此处必须填 "none"
    "", // server name, 留空
),
// 2. 将原本 https://some-example.com 的域名写到 dns://some-example.com 中作为 target
client.WithTarget("dns://some-example.com")
//    直接访问 ip:port 时可以直接写 target: ip://x.x.x.x:xx
client.WithTarget("ip://x.x.x.x:xx")
```

除了这些 option 以外，代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作

示例如下：

```go
func TestHTTPSSkipClientVerify(t *testing.T) {
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork("tcp"),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
        server.WithTLS(
            "../testdata/server.crt",
            "../testdata/server.key",
            "",
        ),
    )
    thttp.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) error {
        w.Write([]byte(t.Name()))
        return nil
    })
    thttp.RegisterNoProtocolService(service)
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    req := &codec.Body{}
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithTLS(
                "", "", "none", "",
            ),
        ))
    require.Equal(t, []byte(t.Name()), rsp.Data)
}
```

### 客户端使用 io.Reader 进行流式发送文件

需要 trpc-go 版本 >= v0.13.0

关键点在于将一个 `io.Reader` 填到 `thttp.ClientReqHeader.ReqBody` 字段上 (`body` 是一个 `io.Reader`):

```go
reqHeader := &thttp.ClientReqHeader{
    Method:  http.MethodPost,
    Header:  header,
    ReqBody: body, // Stream send.
}
```

然后在调用时指定 `client.WithReqHead(reqHeader)`:

```go
c.Post(context.Background(), "/", req, rsp,
    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
    client.WithSerializationType(codec.SerializationTypeNoop),
    client.WithCurrentCompressType(codec.CompressTypeNoop),
    client.WithReqHead(reqHeader),
)
```

示例如下：

```go
func TestHTTPStreamFileUpload(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    go http.Serve(ln, &fileHandler{})
    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    // Open and read file.
    fileDir, err := os.Getwd()
    require.Nil(t, err)
    fileName := "README.md"
    filePath := path.Join(fileDir, fileName)
    file, err := os.Open(filePath)
    require.Nil(t, err)
    defer file.Close()
    // Construct multipart form file.
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, err := writer.CreateFormFile("field_name", filepath.Base(file.Name()))
    require.Nil(t, err)
    io.Copy(part, file)
    require.Nil(t, writer.Close())
    // Add multipart form data header.
    header := http.Header{}
    header.Add("Content-Type", writer.FormDataContentType())
    reqHeader := &thttp.ClientReqHeader{
        Method:  http.MethodPost,
        Header:  header,
        ReqBody: body, // Stream send.
    }
    req := &codec.Body{}
    rsp := &codec.Body{}
    // Upload file.
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithReqHead(reqHeader),
        ))
    require.Equal(t, []byte(fileName), rsp.Data)
}

type fileHandler struct{}

func (*fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    _, h, err := r.FormFile("field_name")
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
    // Write back file name.
    w.Write([]byte(h.Filename))
    return
}
```

### 客户端使用 io.Reader 进行流式读取回包

需要 trpc-go 版本 >= v0.15.0

关键在于添加 `thttp.ClientRspHeader` 并指定 `thttp.ClientRspHeader.ManualReadBody` 字段为 `true`:

```go
rspHead := &thttp.ClientRspHeader{
    ManualReadBody: true,
}
```

然后调用时加上 `client.WithRspHead(rspHead)`:

```go
c.Post(context.Background(), "/", req, rsp,
    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
    client.WithSerializationType(codec.SerializationTypeNoop),
    client.WithCurrentCompressType(codec.CompressTypeNoop),
    client.WithRspHead(rspHead),
)
```

最后可以在 `rspHead.Response.Body` 上进行流式读包：

```go
body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
defer body.Close()            // Do remember to close the body.
bs, err := io.ReadAll(body)
```

示例如下：

```go
func TestHTTPStreamRead(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    go http.Serve(ln, &fileServer{})

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )

    // Enable manual body reading in order to
    // disable the framework's automatic body reading capability,
    // so that users can manually do their own client-side streaming reads.
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: true,
    }
    req := &codec.Body{}
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithRspHead(rspHead),
        ))
    require.Nil(t, rsp.Data)
    body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
    defer body.Close()            // Do remember to close the body.
    bs, err := io.ReadAll(body)
    require.Nil(t, err)
    require.NotNil(t, bs)
}

type fileServer struct{}

func (*fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./README.md")
    return
}
```

### 收发 SSE

Server-Sent Events (SSE) 是一种在服务器和客户端之间建立单向通信的技术，服务器可以通过这种方式向客户端推送实时更新。实现 SSE 主要有两个关键点：

- **服务端及客户端对于 Content-Type 以及相关 header 的设置**
  - 设置 `Content-Type` 为 `text/event-stream`，并确保响应是流式的。

- **服务器及客户端遵循 [SSE 格式](https://html.spec.whatwg.org/multipage/server-sent-events.html#server-sent-events) 通信**
  - 服务端
    - 需要按照 SSE 格式发送事件，并需要及时 `flush` 到客户端。
    - 在版本 >= v0.19.0 时，`thttp` 提供了一个 `WriteSSE` 函数，用于将 `sse.Event` 结构体按照 SSE 格式快速写进 `io.Writer` 中。用户无需再关心 SSE 数据格式。
    - 在版本 < v0.19.0 时，需要**手动拼接响应体**，然后再写入 `http.ResponseWriter` 中。
  - 客户端
    - 在版本 >= v0.17.0 时，**`thttp.ClientRspHeader` 提供了一个名为 `SSEHandler` 的字段，用于注册接收 SSE 数据的回调实现**。
    - 在版本 < v0.17.0 时，需要**手动进行原始的解析操作，使用 `io.Reader` 进行流式读取回包**（见上一节）。

以下是一个完整的 SSE 测试示例，包括服务端和客户端的实现。如果需要更详细的例子，可以参考 [SSE normal example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/normal)。

```go
func TestHTTPSendAndReceiveSSE(t *testing.T) {
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )
    pattern := "/" + t.Name()
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set(thttp.Connection, "keep-alive")
        w.Header().Set("Access-Control-Allow-Origin", "*")
        bs, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        msg := string(bs)
        for i := 0; i < 3; i++ {
            e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
            if err := thttp.WriteSSE(w, e); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            flusher.Flush()
            time.Sleep(500 * time.Millisecond)
        }
        return
    }))
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    t.Run("automatically", func(t *testing.T) {
        reqHeader := &thttp.ClientReqHeader{
            Method: http.MethodPost,
        }
        var data []byte
        rspHead := &thttp.ClientRspHeader{
            ManualReadBody: false,
            SSEHandler: sseHandler(func(e *sse.Event) error {
                t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
                if string(e.Event) == "message" {
                    data = append(data, e.Data...)
                }
                return nil
            }),
        }
        req := &codec.Body{Data: []byte("hello")}
        rsp := &codec.Body{}
        require.Nil(t,
            c.Post(context.Background(), pattern, req, rsp,
                client.WithCurrentSerializationType(codec.SerializationTypeNoop),
                client.WithSerializationType(codec.SerializationTypeNoop),
                client.WithCurrentCompressType(codec.CompressTypeNoop),
                client.WithReqHead(reqHeader),
                client.WithRspHead(rspHead),
                client.WithTimeout(time.Minute),
            ))
        require.Equal(t, "hello0hello1hello2", string(data))
    })

    t.Run("manually", func(t *testing.T) {
        reqHeader := &thttp.ClientReqHeader{
            Method: http.MethodPost,
        }
        rspHead := &thttp.ClientRspHeader{
            ManualReadBody: true,
        }
        req := &codec.Body{Data: []byte("hello")}
        rsp := &codec.Body{}
        require.Nil(t,
            c.Post(context.Background(), pattern, req, rsp,
                client.WithCurrentSerializationType(codec.SerializationTypeNoop),
                client.WithSerializationType(codec.SerializationTypeNoop),
                client.WithCurrentCompressType(codec.CompressTypeNoop),
                client.WithReqHead(reqHeader),
                client.WithRspHead(rspHead),
                client.WithTimeout(time.Minute),
            ))

        body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
        defer body.Close()            // Do remember to close the body.
        // Note that the following code disobeys the SSE protocol, which is simply splitting the lines with '\n'
        // and discarding the "data:" prefix. Since the manual process is too troublesome, we do not recommend this.
        buf := make([]byte, 1024)
        var data strings.Builder
        for {
            n, err := body.Read(buf)
            if err == io.EOF {
                break
            }
            require.Nil(t, err)
            lines := bytes.Split(buf[:n], []byte("\n"))
            for _, line := range lines {
                if !bytes.HasPrefix(line, []byte("data:")) {
                    continue
                }
                fromIndex := len("data:")
                if line[fromIndex] == ' ' {
                    fromIndex++ // Ignore the optional space after the data: prefix.
                }
                data.Write(line[fromIndex:])
            }
        }

        require.Equal(t, "hello0hello1hello2", data.String())
    })
}
```

对于可能返回 SSE 或非 SSE 的接口，客户端提供了以下字段：

- 在版本 >= v0.19.0 时，**`thttp.ClientRspHeader` 提供了 `SSECondition` 和 `ResponseHandler` 两个字段，用于根据服务器的响应采取不同的回调策略**。
  - `SSECondition`: 如果 **`SSECondition` 返回 `true`，且用户实现了 `SSEHandler`**，则回调 `SSEHandler`。用户可以自行实现该接口，可以判断响应头是否包含 `Content-Type: text/event-stream`，但是请注意**并不是所有服务实现都严格遵守此规则**；
  如果将该字段置空，框架将使用默认的实现（返回 `true`）。
  - `ResponseHandler`: 如果 **`SSECondition` 返回 `false`，或用户没有实现 `SSEHandler`**，则回调 `ResponseHandler`。如果用户没有实现该接口，框架的兜底策略为自动读取回包。

- 在版本 < v0.19.0 时，需要**手动进行原始的解析操作，根据响应区分是否为 SSE 消息，然后使用 `io.Reader` 采取不同的策略进行流式读取回包**（见上一节）。

请注意，**`SSEHandler` 和 `ResponseHandler` 均需在设置 `ManualReadBody` 为 `false` 时才会生效**。

以下是一个完整的 SSE 测试示例，包括服务端和客户端的实现。如果需要更详细的例子，可以参考 [SSE multiple example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/multiple)。

```go
func TestHTTPSendAndReceiveSSEAndNormalResponse(t *testing.T) {
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )
    pattern := "/" + t.Name()
    isSSE := true // Whether to send an SSE event, the first time is true.
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Switch between SSE and normal response.
        defer func() { isSSE = !isSSE }()
        if isSSE {
            sseHandlerFunc(w, r)
            return
        }
        normalHandlerFunc(w, r)
    }))

    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )

    reqHeader := &thttp.ClientReqHeader{
        Method: http.MethodPost,
    }

    var data []byte
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: false,
        SSECondition: func(r *http.Response) bool {
            return r.Header.Get("Content-Type") == "text/event-stream"
        },
        ResponseHandler: rspHandler(func(r *http.Response) error {
            bs, err := io.ReadAll(r.Body)
            if err != nil {
                return err
            }
            t.Logf("Receive http response: %s", string(bs))
            data = append(data, bs...)
            return nil
        }),
        SSEHandler: sseHandler(func(e *sse.Event) error {
            t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
            if string(e.Event) == "message" {
                data = append(data, e.Data...)
            }
            return nil
        }),
    }

    req := &codec.Body{Data: []byte("hello")}
    rsp := &codec.Body{}
    // The first time we send a request, the response is an SSE event, and the second is a normal response.
    // It is to say, the handler will switch between SSE and normal response, but the response data are the same.
    for i := 0; i < 4; i++ {
        t.Run(fmt.Sprintf("request "+strconv.Itoa(i)), func(t *testing.T) {
            data = []byte{} // Clear the data.
            require.Nil(t,
                c.Post(context.Background(), pattern, req, rsp,
                    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
                    client.WithSerializationType(codec.SerializationTypeNoop),
                    client.WithCurrentCompressType(codec.CompressTypeNoop),
                    client.WithReqHead(reqHeader),
                    client.WithRspHead(rspHead),
                    client.WithTimeout(time.Minute),
                ))
            require.Equal(t, "hello0hello1hello2", string(data))
        })
    }
}

// sseHandler is a handler that handles sse events.
// It sends responses with the header of "Content-Type: text/event-stream".
func sseHandlerFunc(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set(thttp.Connection, "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    bs, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    msg := string(bs)
    // Send sse message.
    for i := 0; i < 3; i++ {
        e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
        if err := thttp.WriteSSE(w, e); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        flusher.Flush()
        time.Sleep(500 * time.Millisecond)
    }
}

// normalHandler is a handler that handles normal responses.
// It sends responses with the header of "Content-Type: text/plain".
func normalHandlerFunc(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set(thttp.Connection, "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    bs, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    msg := string(bs)
    var data []byte
    for i := 0; i < 3; i++ {
        data = append(data, []byte(msg+strconv.Itoa(i))...)
    }
    _, _ = w.Write(data)
}

type sseHandler func(*sse.Event) error

// Handle handles sse event, if the returned error is non-nil,
// the framework will abort the reading of the HTTP connection.
func (h sseHandler) Handle(e *sse.Event) error {
    return h(e)
}

type rspHandler func(*http.Response) error

// Handle handles common HTTP response.
func (h rspHandler) Handle(r *http.Response) error {
    return h(r)
}
```

### 收发 SSE (基于 github.com/r3labs/sse )

对于更复杂的 SSE 处理，可以考虑使用第三方库 [r3labs/sse](https://github.com/r3labs/sse)。

> 请注意，[r3labs/sse](https://github.com/r3labs/sse) 使用的是 `sse.Client` 而不是标准库的 `http.Client`，而且仅支持 `http.MethodGet` 请求，并且可定制化的内容较少。
> 如果你需要更多的定制化功能，可以将 [r3labs/sse](https://github.com/r3labs/sse) 中的客户端实现逻辑提取出来，与上一节中提到的 **收发 SSE** 的客户端写法结合使用。
> 然而，这种方式对于客户端做转发可能有一定的影响，因此目前**暂不推荐**使用这种方式处理 SSE。

以下是一个基于 r3labs/sse 完整的 SSE 测试示例，包括服务端和客户端的实现。如果需要更详细的例子，可以参考 [SSE r3labs example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/r3labs)
以及 [r3labs/sse/http_test.go](https://github.com/r3labs/sse/blob/v2.10.0/http_test.go)。

```go
func TestHTTPSendAndReceiveSSEWithR3Lab(t *testing.T) {
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()

    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )

    pattern := "/" + t.Name()

    svr := sse.New()
    mux := http.NewServeMux()
    mux.Handle(pattern, svr)
    thttp.RegisterNoProtocolServiceMux(service, mux)
    svr.CreateStream("test")

    for i := 0; i < 3; i++ {
        event := &sse.Event{
            ID:    []byte(fmt.Sprintf("%d", i)),
            Event: []byte("message"),
            Data:  []byte(fmt.Sprintf("This is message %d", i)),
        }
        svr.Publish("test", event)
    }

    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := sse.NewClient(fmt.Sprintf("http://%s%s", ln.Addr().String(), pattern))

    events := make(chan *sse.Event)
    go func() {
        err = c.Subscribe("test", func(msg *sse.Event) {
            if len(msg.Data) > 0 {
                events <- msg
            }
        })
    }()

    // Wait for the subscription to succeed.
    time.Sleep(200 * time.Millisecond)
    require.Nil(t, err)

    for i := 0; i < 3; i++ {
        msg, err := wait(events, 500*time.Millisecond)
        require.Nil(t, err)
        require.Equal(t, []byte(fmt.Sprintf("This is message %d", i)), msg)
    }
}

// wait waits for the sse event and read data into msg. If timeout, return error.
func wait(ch chan *sse.Event, duration time.Duration) ([]byte, error) {
    var err error
    var msg []byte

    select {
    case event := <-ch:
        msg = event.Data
    case <-time.After(duration):
        err = errors.New("timeout")
    }
    return msg, err
}
```

### 客户端做转发

场景：客户端请求服务端，将服务端的回包转发给其他服务。

在一些情况下服务端回包的具体形式未知，所以当前客户端无法提前构造出一个响应结构体来做反序列化。

此时可以使用 `client.WithCurrentSerializationType(codec.SerializationTypeNoop)` 来指定序列化反序列化方式为空操作，从而直接操作原始数据。

示例如下：

```go
func TestHTTPProxy(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, err := io.ReadAll(r.Body)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        w.Header().Add("Content-Type", "application/json")
        w.Write(bs)
        return
    }))
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    type request struct {
        Message string `json:"message"`
    }
    data := "hello"
    bs, err := json.Marshal(&request{Message: data})
    require.Nil(t, err)
    req := &codec.Body{Data: bs}
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeJSON),
        ))
    require.Equal(t, bs, rsp.Data)
}
```

同时这个示例可以结合流式读取回包，如：

```go
    // Enable manual body reading in order to
    // disable the framework's automatic body reading capability,
    // so that users can manually do their own client-side streaming reads.
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: true,
    }
    req = &codec.Body{Data: bs}
    rsp = &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithRspHead(rspHead),
        ))
    require.Nil(t, rsp.Data)
    body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
    defer body.Close()            // Do remember to close the body.
    result, err := io.ReadAll(body)
    require.Nil(t, err)
    require.Equal(t, bs, result)
```

### 客户端服务端收发 HTTP chunked

1. 客户端发送 HTTP chunked:
   1. 添加 `chunked` Transfer-Encoding header
   2. 然后使用 io.Reader 进行发包
2. 客户端接收 HTTP chunked: Go 标准库 HTTP 自动支持了对 chunked 的处理，上层用户对其是无感知的，只需在 resp.Body 上面循环读直至 `io.EOF` (或者用 `io.ReadAll`)
3. 服务端读取 HTTP chunked: 和客户端读取类似
4. 服务端发送 HTTP chunked: 将 `http.ResponseWriter` 断言为 `http.Flusher`, 然后在每发送一部分数据后调用 `flusher.Flush()`, 这样就会自动触发 `chunked` encoding 从而发送出一个 chunk

示例如下：

```go
func TestHTTPSendReceiveChunk(t *testing.T) {
    // HTTP chunked example:
    //   1. Client sends chunks: Add "chunked" transfer encoding header, and use io.Reader as body.
    //   2. Client reads chunks: The Go/net/http automatically handles the chunked reading.
    //                           Users can simply read resp.Body in a loop until io.EOF.
    //   3. Server reads chunks: Similar to client reads chunks.
    //   4. Server sends chunks: Assert http.ResponseWriter as http.Flusher, call flusher.Flush() after
    //         writing a part of data, it will automatically trigger "chunked" encoding to send a chunk.

    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    go http.Serve(ln, &chunkedServer{})

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )

    // Open and read file.
    fileDir, err := os.Getwd()
    require.Nil(t, err)
    fileName := "README.md"
    filePath := path.Join(fileDir, fileName)
    file, err := os.Open(filePath)
    require.Nil(t, err)
    defer file.Close()

    // 1. Client sends chunks.

    // Add request headers.
    header := http.Header{}
    header.Add("Content-Type", "text/plain")
    // Add chunked transfer encoding header.
    header.Add("Transfer-Encoding", "chunked")
    reqHead := &thttp.ClientReqHeader{
        Method:  http.MethodPost,
        Header:  header,
        ReqBody: file, // Stream send (for chunks).
    }

    // Enable manual body reading in order to
    // disable the framework's automatic body reading capability,
    // so that users can manually do their own client-side streaming reads.
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: true,
    }
    req := &codec.Body{}
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithReqHead(reqHead),
            client.WithRspHead(rspHead),
        ))
    require.Nil(t, rsp.Data)

    // 2. Client reads chunks.

    // Do stream reads directly from rspHead.Response.Body.
    body := rspHead.Response.Body
    defer body.Close() // Do remember to close the body.
    buf := make([]byte, 4096)
    var idx int
    for {
        n, err := body.Read(buf)
        if err == io.EOF {
            t.Logf("reached io.EOF\n")
            break
        }
        t.Logf("read chunk %d of length %d: %q\n", idx, n, buf[:n])
        idx++
    }
}

type chunkedServer struct{}

func (*chunkedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 3. Server reads chunks.

    // io.ReadAll will read until io.EOF.
    // Go/net/http will automatically handle chunked body reads.
    bs, err := io.ReadAll(r.Body)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(fmt.Sprintf("io.ReadAll err: %+v", err)))
        return
    }

    // 4. Server sends chunks.

    // Send HTTP chunks using http.Flusher.
    // Reference: https://stackoverflow.com/questions/26769626/send-a-chunked-http-response-from-a-go-server.
    // The "Transfer-Encoding" header will be handled by the writer implicitly, so no need to set it.
    flusher, ok := w.(http.Flusher)
    if !ok {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("expected http.ResponseWriter to be an http.Flusher"))
        return
    }
    chunks := 10
    chunkSize := (len(bs) + chunks - 1) / chunks
    for i := 0; i < chunks; i++ {
        start := i * chunkSize
        end := (i + 1) * chunkSize
        if end > len(bs) {
            end = len(bs)
        }
        w.Write(bs[start:end])
        flusher.Flush() // Trigger "chunked" encoding and send a chunk.
        time.Sleep(500 * time.Millisecond)
    }
    return
}
```

### 客户端发送任意 Content-Type 的数据

两步：

- 请求和响应使用 `*codec.Body` 类型，将期望发送的请求体（以你期望的序列化方式处理后）放入 `(*code.Body).Data` 中
- 通过 `ClientReqHeader` 指定你需要的 `Content-Type` 并传入两个选项 (1. 传入 reqHead, 2. 指定 noop serialization)：

```go
reqHead := &thttp.ClientReqHeader{} 
reqHead.AddHeader("Content-Type", "application/soap+xml; charset=utf-8")
c.Post(.., 
    client.WithReqHead(reqHead),
    client.WithCurrentSerializationType(codec.SerializationTypeNoop))
```

```go
func TestHTTPArbitraryContentType(t *testing.T) {
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://127.0.0.1:80"),
    )
    req := &codec.Body{
        Data: []byte(`<?xml version="1.0" encoding="utf-8"?>` +
        `<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
        `xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
        `xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">` +
        `<soap12:Body>` +
        `<GetActivityInfo xmlns="http://tempuri.org/">` +
        `<ActivityID>id</ActivityID>` +
        `</GetActivityInfo>` +
        `</soap12:Body>` +
        `</soap12:Envelope>`),
    }
    reqHead := &thttp.ClientReqHeader{}
    reqHead.AddHeader("Content-Type", "application/soap+xml; charset=utf-8")
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithReqHead(reqHead),
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
        ))
    require.NotNil(t, rsp.Data)
    t.Logf("receive: %q\n", rsp.Data)
}
```

### 客户端提交 Form 数据

#### 提交 Content-Type 为 `application/x-www-form-urlencoded` 的 Form 数据

指定 `client.WithSerializationType(codec.SerializationTypeForm)` 并传入类型为 `url.Values` 的请求

读取回包时可以通过添加 `thttp.ClientRspHeader` 并指定 `thttp.ClientRspHeader.ManualReadBody` 字段为 `true` 以通过 `io.Reader` 进行流式读取回包（需要 trpc-go 版本 >= v0.15.0）

或者预先定义响应结构体以避免使用到高版本的 `ManualReadBody` 特性

```golang
func TestHTTPSendFormData(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    type response struct {
        Message string `json:"message"`
    }
    s := http.Server{
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            bs, err := io.ReadAll(r.Body)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                return
            }
            t.Logf("server read: %q\n", bs)
            rsp := &response{Message: string(bs)}
            bs, err = json.Marshal(rsp)
            if err != nil {
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
            w.Header().Add("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            w.Write(bs)
        }),
    }
    go s.Serve(ln)

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    req := make(url.Values)
    req.Add("key", "value")

    // Option 1: Use manual read to read response (requires trpc-go >= v0.15.0) 
    // (If you are using an older version of trpc-go, please refer to Option 2 below.)
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: true, // Requires trpc-go >= v0.15.0.
    }
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithSerializationType(codec.SerializationTypeForm),
            client.WithRspHead(rspHead),
        ))
    require.Nil(t, rsp.Data)
    body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
    defer body.Close()            // Do remember to close the body.
    bs, err := io.ReadAll(body)
    require.Nil(t, err)
    require.NotNil(t, bs)

    // Option 2: Predefine the response struct to avoid manual read.
    rsp1 := &response{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp1,
            client.WithSerializationType(codec.SerializationTypeForm),
        ))
    require.NotNil(t, rsp1.Message)
    t.Logf("receive: %s\n", rsp1.Message)
}
```

注意：通过以上形式发送的数据都会被 url encode (如 [Percent-encoding](https://en.wikipedia.org/wiki/Percent-encoding))，如果不希望如此，可以使用 `codec.SerializationTypeNoop`，此时要注意请求和响应都要为 `*codec.Body`

```go
func TestHTTPSendFormData2(t *testing.T) {
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://127.0.0.1:43221"),
    )
    req := &codec.Body{
        Data: []byte(`data='{"cycle":10}'`),
    }
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithSerializationType(codec.SerializationTypeForm),
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
        ))
    require.NotNil(t, rsp.Data)
    t.Logf("receive: %q\n", rsp.Data)
}
```

#### 提交 Content-Type 为 `multipart/form-data` 的 Form 数据

请按照以下步骤操作：

1. 先使用[mime/multipart](https://pkg.go.dev/mime/multipart)将请求参数进行编码
2. 将编码后的结果包装成 `io.Reader`，
3. 参考上面 FAQ"客户端使用 io.Reader 进行流式发送文件"的例子

### 服务端接收文件上传（使用 `multipart/form-data`）

涉及 `multipart/form-data` 类型的数据时，一律推荐使用一个单独的泛 HTTP 标准服务（而非泛 HTTP RPC 或 RESTful 服务）来进行处理，示例如下：

```go
package main

import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    s := trpc.NewServer()
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

对于 RESTful 服务的自定义路由问题，可额外参考 [为 RESTful 服务添加额外的自定义路由](../restful/README.zh_CN.md#%E4%B8%BA-restful-%E6%9C%8D%E5%8A%A1%E6%B7%BB%E5%8A%A0%E9%A2%9D%E5%A4%96%E7%9A%84%E8%87%AA%E5%AE%9A%E4%B9%89%E8%B7%AF%E7%94%B1)

### 使用泛 HTTP 标准服务及客户端时，监控上报 req,rsp 为空

首先确认下业务服务是否可以直接使用泛 HTTP RPC 服务或 RESTful，在这两种情况下，req,rsp 是可以正常在监控插件拦截器中拿到的。

泛 HTTP 标准服务的话，req,rsp 是 nil 是设计如此，因为 HTTP 协议无法和 RPC 框架完美地一一对应起来。
回包为 chunk 或者 multipart form data 等形式无法类比于 RPC 来提供一个具体的 rsp 结构体。
假如用户的需求更偏向于是把 HTTP 当成 RPC 来用，也就是说 req rsp 都是明确具体的有字段定义的结构体，在这种情况下，可以考虑使用带有 proto 文件的 HTTP RPC 服务 或者是 RESTful 服务。

如果必须要做的话，可以自定义一对服务端或客户端拦截器将监控插件拦截器夹在中间

- "http_req_collector": 在监控插件拦截器之前，为其提供需要上报的 req，并恢复被 "http_rsp_collector" 修改了的 rsp
- "http_rsp_collector": 在监控插件拦截器之后，为其提供需要上报的 rsp，并恢复被 "http_req_collector" 修改了的 req

```go
import (
    "bytes"
    "context"
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/filter"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func ExampleRegister() {
    name1 := "http_req_collector"
    name2 := "http_rsp_collector"
    // Example trpc_go.yaml:
    //
    // server:
    //   service:
    //     - name: trpc.server.service.StdHTTPMethod
    //       filter:
    //         - http_req_collector
    //         - metric_filter_name
    //         - http_rsp_collector
    // client:
    //   service:
    //     - name: trpc.server.service.StdHTTPMethod
    //       filter:
    //         - http_req_collector
    //         - metric_filter_name
    //         - http_rsp_collector
    filter.Register(name1, func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
        h := thttp.Head(ctx)
        if h != nil {
            w := &customResponseWriter{ResponseWriter: h.Response}
            h.Response = w
            _, err := next(ctx, &customRequest{req, h.Request}) // Pass the request you want to report.
            return w.originalRsp, err                           // Preserve the original rsp.
        }
        return next(ctx, req)
    }, func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
        msg := codec.Message(ctx)
        reqHeader, ok := msg.ClientReqHead().(*thttp.ClientReqHeader)
        if ok {
            // For thttp.Get, you can pass msg.ClientRPCName() to report the url parameters.
            return next(ctx, &customRequest{req, reqHeader}, rsp) // Pass the request you want to report.
        }
        return nil
    })
    filter.Register(name2, func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
        if cr, ok := req.(*customRequest); ok {
            h := thttp.Head(ctx)
            if h != nil {
                if w, ok := h.Response.(*customResponseWriter); ok {
                    rsp, err := next(ctx, cr.originalReq) // Preserve the original req.
                    w.originalRsp = rsp
                    return w.response.Bytes(), err // Return the response you want to report.
                }
            }
        }
        return next(ctx, req)
    }, func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
        if cr, ok := req.(*customRequest); ok {
            return next(ctx, cr.originalReq, rsp) // Preserve the original req.
        }
        return next(ctx, req, rsp)
    })
}

type customRequest struct {
    originalReq interface{}
    request     interface{}
}

type customResponseWriter struct {
    originalRsp interface{}
    http.ResponseWriter
    code     int
    response bytes.Buffer
}

func (w *customResponseWriter) WriteHeader(statusCode int) {
    w.code = statusCode
    w.ResponseWriter.WriteHeader(statusCode)
}

func (w *customResponseWriter) Write(bs []byte) (int, error) {
    w.response.Write(bs)
    return w.ResponseWriter.Write(bs)
}
```

### 收到的响应内容为空的原因

1. 错误地使用了 `client.WithCurrentSerializationType`，这个选项通常用于透明转发，其本质作用是强制请求和响应均使用这个选项指定的序列化方式，在正常情况下，框架对于回包的反序列话操作是通过读取回包中的 `Content-Type` header 来确定的，假如 `WithCurrentSerializationType` 指定的序列化类型和回包本身的类型不符，那么就有可能得到空的回包
2. 服务端的回包中使用了不恰当的 `Content-Type`，比如回包内容的实质序列化方式是 `application/json`，但是 `Content-Type` 却写了 `application/protobuf`，对于这种情况最好的做法是让服务端改正其错误的做法；对于一些不准确的 `Content-Type`，比如使用 `text/html` 作为 header，实质内容为 `application/json` 的，用户可以在服务初始化时调用 `thttp.SetContentType("text/html", codec.SerializationTypeJSON)` 来对这个 `Content-Type` 进行手动注册
3. 服务端的回包内容和指定的响应结构体无法对应上，比如代码中指定的响应体为 `type rsp struct { Message string }`，但是实际的回包是 `{'data':{'message':'hello'}}`，那么需要用户自己构造一个正确的响应结构体以确保正常的序列化，或者使用 [manual read body 一节中提到的操作](#客户端使用-ioreader-进行流式读取回包) 进行手动读包然后反序列化

### 限制只接收 POST 方法的请求

在 HTTP RPC 服务中，GET/POST 请求都是可以接受的，假如只希望用户通过 POST 方法进行请求，可以设置 `thttp.ServerCodec` 的 `POSTOnly` 字段（要求版本 >= v0.16.0）

```go
// 更改所有 protocol: http 的服务只接收 POST 请求
thttp.DefaultServerCodec.POSTOnly = true
```

此时当使用 GET 方法发送请求时，发送方会收到 "400 Bad Request" 的错误码，并在 "trpc-error-msg" header 中看到如下错误信息："service codec Decode: server codec only allows POST method request, the current method is GET"

### 为 http_no_protocol 服务的每个 handler 提供各自的 timeout

关键点在于使用 `http.TimeoutHandler` 将自己定义的 `http.Handler` 给封装起来

示例如下：

```go
func TestHTTPTimeoutHandler(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    s := server.New(
        server.WithServiceName("trpc.app.server.Service_http"),
        server.WithListener(ln),
        server.WithProtocol("http_no_protocol"))
    defer s.Close(nil)
    const timeout = 50 * time.Millisecond
    thttp.Handle("/", http.TimeoutHandler(&fileServer{sleep: 2 * timeout}, timeout, "timeout"))
    thttp.RegisterNoProtocolService(s)
    go s.Serve()

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )

    req := &codec.Body{}
    rsp := &codec.Body{}
    err = c.Post(context.Background(), "/", req, rsp,
        client.WithCurrentSerializationType(codec.SerializationTypeNoop),
        client.WithSerializationType(codec.SerializationTypeNoop),
        client.WithCurrentCompressType(codec.CompressTypeNoop),
    )
    require.NotNil(t, err)
    require.Contains(t, fmt.Sprint(err), "timeout", "expect err is timeout err, got: %s", err)
}

type fileServer struct {
    sleep time.Duration
}

func (s *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    time.Sleep(s.sleep)
    http.ServeFile(w, r, "./README.md")
    return
}
```

### 对框架构造的 http.Request 做自定义修改（如修改 Content-Length）

通过 `client.WithReqHead(&thttp.ClientReqHeader{Request: xx})` 可以指定直接指定框架要发送的 `http.Request`，但是这种方法无法使框架的服务发现构造的 `Address` 生效（比如通过北极星寻址会不生效）

框架在 `thttp.ClientReqHeader` 中提供了 `DecorateRequest` 字段用来对框架构造的 `http.Request` 进行自定义的修改

> trpc-go 版本要求：>= v0.16.0

比如一个场景是使用自定义的 `io.Reader` 发送请求，并手动设置 `http.Request` 中的 Content-Length:

```go
data := []byte("hello")
reader := bytes.NewBuffer(data)
reqHeader := &thttp.ClientReqHeader{
    ReqBody: io.LimitReader(reader, int64(len(data))),
    DecorateRequest: func(r *http.Request) *http.Request {
        r.ContentLength = int64(len(data))
        return r
    },
}
req := &codec.Body{}
rsp := &codec.Body{}
c.Post(context.Background(), "/", req, rsp,
    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
    client.WithReqHead(reqHeader),
)
```

在框架构造 `http.Request` 时，由于 `thttp.ClientReqHeader.ReqBody` 的长度无法被识别，最终标准库会采用 chunked encoding 的形式进行请求的发送，通过指定 `thttp.ClientReqHeader.DecorateRequest` 以显式设置 Content-Length 可以避免这种情况发生（即：不使用 chunked encoding）

完整的测试用例可以参考 `transport_test.go` 中的 `TestDecorateRequest`

原始问题可以参考：[码客问题：trpc-go 的 http client 怎么在设置 content-length 的同时使用北极星插件呢？](http://mk.woa.com/q/292458)

### 同时支持泛 HTTP 标准服务以及 RESTful 服务

用户期望在使用泛 HTTP 标准服务处理文件的同时，能够使用到基于桩代码的 RESTful 服务，推荐阅读 [为-restful-服务添加额外的自定义路由](../restful/README.zh_CN.md#为-restful-服务添加额外的自定义路由) 一节分为两个服务来支持。

### 设置 GetSerialization 反序列化 query parameters 的行为

在 trpc-go v0.16.0 之前，`GetSerialization` 反序列化 query parameters 的行为默认是**大小写不敏感**的。
在 trpc-go v0.16.0 - v0.18.1，`GetSerialization` 反序列化 query parameters 的行为默认是**大小写敏感**的。
如今，即 trpc-go > v0.18.1，`GetSerialization` 反序列化 query parameters 的行为默认是**大小写不敏感**的。
若用户期望 `GetSerialization` 以**大小写敏感**的方式反序列化 query parameters，可进行如下操作：

```go
// Remember to invoke codec.RegisterSerializer to register the new Serializer.
codec.RegisterSerializer(codec.SerializationTypeGet,
    // Set the GetSerialization's caseSensitive = false.
    http.NewGetSerializationWithCaseSensitive("json", true))
```

请注意，假如设置 `GetSerialization` 为大小写不敏感的话，存在是无法 unmarshal 到 nested structure 上的缺陷，推荐阅读 <https://git.woa.com/trpc-go/trpc-go/issues/865>

### 关于 value detached transport 导致的资源泄露问题

由于标准库 `net/http` 在 go1.22 之前会持有传入的 `ctx`，从而间接持有 `ClientReqHeader` 中的 `ReqBody`，造成内存泄漏，框架设计了 value detached transport，将 `ctx` 上的 value detach 之后再传给下层的 transport，同时为了保留 `ctx` 上的超时及取消能力，新创建了 goroutine 来监听 `ctx.Done()`，而假如传入的 `ctx` 仅有 cancel，没有 timeout，并且 `ctx` 又永远不调用 cancel 时，这个新建的 goroutine 以及原 `ctx` 上的资源都会一并泄露掉，尽管 !2403 尝试减少 goroutine 的泄露，但是资源的泄露无法避免，如果用户存在这种场景，推荐使用 go1.22 以上的版本进行编译，并加上以下代码以去除 value detached transport:

```go
import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    thttp.NewRoundTripper = func(r http.RoundTripper) http.RoundTripper {
        return r
    }
}
```
