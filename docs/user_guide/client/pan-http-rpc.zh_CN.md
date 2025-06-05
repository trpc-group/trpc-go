# 1 前言

正如“搭建泛 HTTP RPC 服务”文档中所描述的，对于绝大部分应用场景，开发人员是不需要关注 RPC 服务内部协议细节，由框架内部封装。这一原则对于客户端的（使用 tRPC-Go 框架）开发同样适用，所以泛 HTTP RPC 服务客户端的开发和 tRPC 服务的调用完全一致，具体请参考 [tRPC-Go 快速上手](https://iwiki.woa.com/pages/viewpage.action?pageId=118272478)。

但是对于少数业需要感知底层协议的场景，比如对于泛 HTTP 协议的“Cookie”的处理，框架在原有 API 的基础上扩充了接口用于 HTTP Head 的操作。文本在 [tRPC-Go 快速上手](https://iwiki.woa.com/pages/viewpage.action?pageId=118272478)的基础上，重点对泛 HTTP RPC 服务调用需要特别关注的部分做介绍。

在真正开始之前，用户首先需要掌握以下知识：

- 关于客户端开发中涉及的基本概念和开发流程，请参考 [客户端开发向导](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)
- 关于什么是“泛 HTTP RPC 服务”，请参考 [搭建泛 HTTP RPC 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=490796254)

tRPC-Go 从 v19.0.0 后支持 fasthttp 调用泛 HTTP 标准服务，[使用 fasthttp 调用泛 HTTP RPC 服务](#5-使用-fasthttp-调用泛-http-rpc-服务)。

在设计上，tfasthttp 在行为和用法上尽可能地与 thttp 保持一致，但由于各种原因（主要是 `net/http` 与 `fasthttp` 带来的不一致），其用法可能兼容性较差。

本文主要从如何使用出发，指导用户快速上手 fasthttp，关于细节，请用户查看 [tfasthttp 使用指南](https://doc.weixin.qq.com/doc/w3_Ac0AYwanAIUfx1rVLYYTm2A4u2oHj?scode=AJEAIQdfAAowr0OpC7Ac0AYwanAIU&version=4.1.28.6010&platform=win)。

# 2 接口

对于 RPC 类型的服务调用，框架使用了“ClientProxy”来进行服务接口调用的，框架为“client”提供了一系列的函数来设置 RPC 调用的配置。具体 API 函数请参考[客户端开发向导](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)。本节主要对 HTTP 报文头操作和其它一些常用 API 做介绍。

## 2.1 HTTP 报文头处理

对于 HTTP 请求和响应报文头的处理接口包括：

以下接口为 client 设置 HTTP 请求和响应头，定义在“git.code.oa.com/trpc-go/trpc-go/client”包中：

```go
// WithReqHead 设置后端请求包头
func WithReqHead(h interface{}) Option
// WithRspHead 设置后端响应包头
func WithRspHead(h interface{}) Option
```

以下接口为 client 添加 Head 字段，定义在“git.code.oa.com/trpc-go/trpc-go/http”包中

```go
// ClientReqHeader 封装 http client 请求的上下文
// 禁止在初始化时指定 header，需要在每次调用时设置
type ClientReqHeader struct {
    Schema  string // http https
    Method  string
    Host    string
    Request *stdhttp.Request
    Header  stdhttp.Header
}
// AddHeader 添加 http header
func (h *ClientReqHeader) AddHeader(key string, value string)
// ClientRspHeader 封装 http client 请求响应的上下文
type ClientRspHeader struct {
    Response *stdhttp.Response
}
```

## 2.2 常用 API 介绍

tRPC-Go 框架提供了 Option 配置函数，用于协议，序列化类型和压缩方式的设置。这些函数通常用于一些客户端工具程序，通过避免使用配置文件来达到使用的便利性。常用 Client 配置函数包括：

```go
// WithProtocol 指定服务协议名字
func WithProtocol(s string) Option 
// WithNetwork 指定 server 监听网络 tcp or udp 默认 tcp
func WithNetwork(s string) Option
// WithTLS 指定 tls 配置，支持单向认证，双向认证
// 框架版本 >= v0.19.0 时，支持在 certFile, keyFile 和 caFile 参数配置多个文件路径
// 两个文件路径之间用英文冒号 `:` 分隔，中间不要带空格，如：WithTLS("a.crt:b.crt", "a.key:b.key", "caA.pem:caB.pem")
func WithTLS(certFile, keyFile, caFile string) Option
// 设置序列化类型：需要使用 tRPC 协议对应的数值，框架会自动转变成“Content-Type”
func WithSerializationType(t int) Option
// 设置压缩方式
func WithCompressType(t int) Option
// 内置序列化类型
const (
    SerializationTypePB         = 0 // protobuf
    SerializationTypeJCE        = 1 // jce
    SerializationTypeJSON       = 2 // json
    SerializationTypeFlatBuffer = 3 // flat buffer
    SerializationTypeNoop       = 4 // bytes 二进制数据空序列化方式
    SerializationTypeUnsupported = 128 // 不支持
    SerializationTypeForm        = 129 // http form data 表单 kv 结构
    SerializationTypeGet         = 130 // http server 处理 get 请求
)
// 内置压缩方式
const (
    CompressTypeNoop   = 0
    CompressTypeGzip   = 1
    CompressTypeSnappy = 2
    CompressTypeZlib   = 3
)
```

tRPC-Go 框架支持用户自定义序列化类型和压缩方式，在添加序列化类型和压缩方式时，客户端和服务端都必须添加。具体操作请参考 [搭建泛 HTTP RPC 服务](https://iwiki.woa.com/p/490796254) 第 **7.1** 和 **7.2** 章节

# 3 配置

对于客户端配置，框架提供了两种设置方式：**框架配置文件方式** 和**Option 配置**（第 2.2 节已介绍）。系统推荐使用框架配置文件方式，这样可以和代码解耦，便于管理和修改。对于客户端通用配置，这里不做赘述，具体请参考 [客户端开发向导](https://iwiki.woa.com/p/284289117)。本节重点介绍 协议，序列化，压缩方式 在框架配置文件中的定义。

## 3.1 协议

协议在这里特指底层协议使用"http","https","http2","http3"中的其中一种，客户端协议的设置，取决于服务端的设置。协议配置在框架配置文件中的位置为：

```yaml
global:
  ...
server:
  ...
client:
  service:
    - name: trpc.test.stdhttp.hello
      ...
      # 对于泛 HTTP 服务，除 http3 需要填 udp 之外，其它都需要填 tcp
      network: tcp
      # 对于泛 HTTP 服务，http，https 类型需要填 http，http2 类型需要填 http2，http3 类型需要填 http3
      protocol: http2
      # 对于泛 HTTP 服务，HTTP 协议不需要填，其它协议必填
      # 框架版本 >= v0.19.0 时，支持在 tls_key 字段配置多个文件路径，两个文件路径之间用英文冒号分隔，中间不要带空格
      tls_key: ./license.key # ./licenseA.key:./licenseB.key
      # 对于泛 HTTP 服务，HTTP 协议不需要填，其它协议必填
      # 框架版本 >= v0.19.0 时，支持在 tls_cert 字段配置多个文件路径，两个文件路径之间用英文冒号分隔，中间不要带空格
      tls_cert: ./license.crt # ./licenseA.crt:./licenseB.crt
      # 对于泛 HTTP 服务，HTTP 协议不需要填，其它协议如果开启反向认证，需要提供 client 的 CA 证书
      # 框架版本 >= v0.19.0 时，支持在 ca_cert 字段配置多个文件路径，两个文件路径之间用英文冒号分隔，中间不要带空格
      ca_cert: ./ca.cert # ./caA.cert:./caB.cert
```

## 3.2 序列化

客户端可以指定接口数据的序列化方式，服务端根据客户端携带的“Content-Type”来进行反序列化的。序列化配置在框架配置文件中的位置为：

```yaml
global:
  ...
server:
  ...
client:
  service:
    - name: trpc.test.stdhttp.hello
      ...
      # 选填，序列化协议，默认为 -1，即不设置
      serialization: Integer(0=pb, 1=JCE, 2=json, 3=flat_buffer, 4=bytes_flow)
```

## 3.3 压缩方式

客户端可以指定接口数据的压缩方式，服务端根据客户端携带的 "Content-Encoding" 来进行解压缩的。压缩方式配置在框架配置文件中的位置为：

```yaml
global:
  ...
server:
  ...
client:
  service:
    - name: trpc.test.stdhttp.hello
      ...
      # 选填，压缩协议，默认为 0，即不压缩
      compression: Integer(0=no_compression, 1=gzip, 2=snappy, 3=zlib)
```

# 4 示例

本节会展示一个完整的例子，客户端调用“搭建泛 HTTP RPC 服务”中第 6.5 节提供的服务，客户端端采用 http 协议，并在 HTTP 请求中携带“request”报文头，并打印 RPC 响应 Msg 和响应头里的“reply”字段。

## 4.1 准备

在开发之前，客户端需要获取服务 PB IDL 文件生成的桩代码。通常情况下服务端在发布服务的同时，会提供桩代码的 git 代码路径，客户端直接引用就行。如何服务端的桩代码没有上传到 git 仓库，可以把桩代码拷贝到客户端工程下，并在 go.mod 里面 replace 本地路径进行引用。

```shell
# 创建工程
go mod init client

# 拷贝桩代码, 并修改 go.mod
echo "replace git.code.oa.com/trpcprotocol/test/rpchttp => ./stub/git.code.oa.com/trpcprotocol/test/rpchttp" >> go.mod

# 添加main.go
vim main.go
```

## 4.2 开发

客户端的代码如下：

```go
package main

import (
    "context"
    stdhttp "net/http"

    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
    pb "git.code.oa.com/trpcprotocol/test/rpchttp"
)

func main() {
    // 如果需要使用框架配置文件来设置 client 端票配置
    trpc.NewServer()

    // 创建 ClientProxy, 设置协议为 HTTP 协议，序列化为 Json
    proxy := pb.NewHelloClientProxy()

    reqHeader := &http.ClientReqHeader{}
    // 必须留空或设置为 "POST"
    reqHeader.Method = "POST"
    // 为 HTTP Head 添加 request 字段
    reqHeader.AddHeader("request", "test")
    // 设置 Cookie
    cookie := &stdhttp.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
    reqHeader.AddHeader("Cookie", cookie.String())

    req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    rspHead := &http.ClientRspHeader{}

    // 发送 HTTP RPC 请求  
    rsp, err := proxy.SayHello(context.Background(), req,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
        client.WithTarget("ip://127.0.0.1:8000"))

    if err != nil {
        log.Warn("get http response err")
        return
    }

    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

**注意：** HTTP RPC 服务端默认可以同时支持 GET/POST 请求，假如服务端使用了 `POSTOnly` 能力限制了只能接受 POST 请求，那么客户端需要指明 POST 请求：

```go
proxy := pb.NewHelloClientProxy()
reqHeader := &http.ClientReqHeader{}
reqHeader.Method = "POST"  // 指明为 POST 请求
req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
rspHead := &http.ClientRspHeader{}
rsp, err := proxy.SayHello(context.Background(), req, client.WithReqHead(reqHeader), client.WithTarget("ip://127.0.0.1:8000"))
```

## 4.3 配置

对于客户端的配置，我们更推荐使用框架配置文件“trpc_go.yaml”来实现，这样可以实现代码和配置的分离。客户端配置示例如下：

```yaml
global:                                 #全局配置
  namespace: Development                #环境类型，分正式 production 和非正式 development 两种类型
  env_name: test                        #环境名称，非正式环境下多环境的名称

client:                                 #客户端调用的后端配置
  timeout: 1000                         #针对所有后端的请求最长处理时间
  namespace: Development                #针对所有后端的环境
  filter:                               #针对所有后端调用函数前后的拦截器列表
  service:                              #针对单个后端的配置
    - name: trpc.test.rpchttp.Hello     #后端服务的 service name
      namespace: Development            #后端服务的环境
      network: tcp                      #后端服务的网络类型 tcp udp 配置优先
      protocol: http                    #应用层协议 trpc http
      target: ip://127.0.0.1:8000       #请求服务地址
      timeout: 1000                     #请求最长处理时间
```

## 4.4 运行

编译客户端：

```shell
go build
./client
```

结果如下：

```log
2020-12-21 20:47:14.045 DEBUG maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=8: CPU quota undefined
2020-12-21 20:47:14.047 INFO client/main.go:43 data is msg:"Hello, World!", request head is tested
```

# 5 使用 fasthttp 调用泛 HTTP RPC 服务

## 5.1 接口

### 5.1.1 客户端创建

tfasthttp 提供了 FastHTTPClientProxy 和 FastHTTPClient 两种客户端封装，一般而言，前者会走服务发现，而后者需要用户自行构建好请求路径。请注意，与 thttp 不同的是，tfasthttp 中 NewFastHTTPClientProxy 返回的是结构体指针而非接口。

```go
// NewFastHTTPClientProxy 新建一个 fasthttp 后端请求代理 必传参数 fasthttp 服务名
// name 后端 fasthttp 服务的服务名，主要用于配置 key，监控上报，name 格式遵循对应名字系统的定义规范
var NewFastHTTPClientProxy = func(name string, opts ...client.Option) *FastHTTPCli

func NewFastHTTPClient(name string, opts ...client.Option) *fasthttp.Client
```

以下给出常见的 client.Option 配置

```go
// WithProtocol 指定服务协议名字
// NewFastHTTPClientProxy 和 NewFastHTTPClient 内部已经调用
func WithProtocol(s string) Option

// WithTLS 指定 tls 配置，支持单向认证，双向认证
// 不验证：设置 caFile == ""
// 单向验证：设置 caFile == "xxx"
// 双向验证：在客户端设置单项验证的基础上，需要服务器配置。
// 框架版本 >= v0.19.0 时，支持在 certFile, keyFile 和 caFile 参数配置多个文件路径
// 两个文件路径之间用英文冒号 `:` 分隔，中间不要带空格，如：WithTLS("a.crt:b.crt", "a.key:b.key", "caA.pem:caB.pem")
func WithTLS(certFile, keyFile, caFile string) Option

// 设置序列化类型：需要使用 tRPC 协议对应的数值，框架会自动转变成 "Content-Type"
func WithSerializationType(t int) Option

// 设置压缩方式
func WithCompressType(t int) Option
```

### 5.1.2 fasthttp 报文头处理

框架提供了以下接口来设置 FastHTTP 请求和响应头，注意，类型与 thttp 完全不同。同时，为了防止过分暴露，tfasthttp 删除了 FastHTTPClientReqHeader 中的 Header 字段，因此需要用户通过 DecorateRequest 进行实现。

```go
// 以下接口定义在 git.code.oa.com/trpc-go/trpc-go/client 包中
// WithReqHead 设置后端请求包头
func WithReqHead(h interface{}) Option
// WithRspHead 设置后端响应包头
func WithRspHead(h interface{}) Option

// 以下接口定义在 git.code.oa.com/trpc-go/trpc-go/http 包中
// ClientReqHeader 封装 fasthttp client 请求的上下文
type FastHTTPClientReqHeader struct {
    Request *fasthttp.Request
    Scheme  string
    Method  string
    Host string
    DecorateRequest func(*fasthttp.Request) *fasthttp.Request
}

// FastHTTPClientRspHeader 封装 fasthttp client 请求响应的上下文
type FastHTTPClientRspHeader struct {
    Response *fasthttp.Response
    ManualReadBody bool
    ResponseHandler FastHTTPRspHandler
    SSECondition func(*fasthttp.Response) bool
    SSEHandler SSEHandler
}
```

以下是添加头部的例子，值得一提的是 DecorateRequest 的调用时机是 Do() 前最后一步。

```go
// Create a FastHTTPClientReqHeader with the POST method.
reqHeader := &thttp.FastHTTPClientReqHeader{
    Method: fasthttp.MethodPost,
    // Add a custom header "Hello": "fcp-post".
    // Notice: "hello" -> "Hello". But we can get "fcp-post" by string(req.Header.Peek("hello")).
    DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
        r.Header.Add("hello", "fcp-post")
        return r
    },
}

// 进行再一步扩展
old := reqHeader.DecorateRequest
if old != nil {
    reqHeader.DecorateRequest = func(r *fasthttp.Request) *fasthttp.Request {
        r = old(r)
        ...
        return r
    }
}
```

### 5.1.3 服务接口调用

在创建好 `FastHTTPClientProxy` 之后，用户就可以使用 "Get"，"Post"，"Put"，"Delete" 接口来调用标准 [Fast]HTTP 服务了。

`FastHTTPClientProxy` 实现了 Client 接口。

```go
type Client interface {
    // HTTP Get 请求，path 为 url 域名后字符串：/cgi-bin/getxxx?k1=v1&k2=v2，响应包默认采用 json 序列化
    Get(ctx context.Context, path string, rspBody interface{}, opts ...client.Option) error
    // HTTP Post 请求，path 为 url 域名后字符串：/cgi-bin/addxxx，请求和响应包默认采用 json 序列化
    Post(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
    // HTTP Put 请求，path 为 url 域名后字符串：/cgi-bin/updatexxx，请求和响应包默认采用 json 序列化
    Put(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
    // HTTP Delete 请求，path 为 url 域名后字符串：/cgi-bin/deletexxx，请求和响应包默认采用 json 序列化
    Delete(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
}
```

上面的函数中的 opts 可以为每一次的服务调用单独设置 client 配置。注意：如果 opts 中使用了 `WithReqHead()`, 业务则需要为 `FastHTTPClientReqHeader` 中 Method 设置正确的值。
原因在于如果业务自行设置 Head 头，则此 Head 会替换掉框架设置的 Head 值。

而 `FastHTTPClient` 则使用的是 `fasthttp` 包下的接口。尽管有些繁琐，但还是推荐大家使用 Do() 来达到对请求和响应的精准控制，以更好利用 `fasthttp`。

```go
fc := thttp.NewFastHTTPClient("fasthttp-client")
fasthttpReq := fasthttp.AcquireRequest()
fasthttpRsp := fasthttp.AcquireResponse()
defer fasthttp.ReleaseRequest(fasthttpReq)
defer fasthttp.ReleaseResponse(fasthttpRsp)

fasthttpReq.SetRequestURI(s.unaryCallCustomURL())
fasthttpReq.Header.SetContentType("application/pb")
fasthttpReq.SetBody(bs)

err = fc.Do(fasthttpReq, fasthttpRsp)
```

## 5.2 配置

tfasthttp 和 thttp 客户端的配置差异主要体现在 `protocol`。以下是一个简单的配置例子：

```yaml
global:
  ...
server:
  ...
client:
  service:
    - name: trpc.test.stdhttp.hello
      ...
      # 对于泛 HTTP 服务，除 http3 需要填 udp 之外，其它都需要填 tcp
      network: tcp
      # 对于泛 HTTP 服务，http，https 类型需要填 fasthttp
      protocol: fasthttp
      # 对于泛 HTTP 服务，HTTP 协议不需要填，其它协议必填
      # 框架版本 >= v0.19.0 时，支持在 tls_key 字段配置多个文件路径，两个文件路径之间用英文冒号分隔，中间不要带空格
      tls_key: ./license.key # ./licenseA.key:./licenseB.key
      # 对于泛 HTTP 服务，HTTP 协议不需要填，其它协议必填
      # 框架版本 >= v0.19.0 时，支持在 tls_cert 字段配置多个文件路径，两个文件路径之间用英文冒号分隔，中间不要带空格
      tls_cert: ./license.crt # ./licenseA.crt:./licenseB.crt
      # 对于泛 HTTP 服务，HTTP 协议不需要填，其它协议如果开启反向认证，需要提供 client 的 CA 证书
      # 框架版本 >= v0.19.0 时，支持在 ca_cert 字段配置多个文件路径，两个文件路径之间用英文冒号分隔，中间不要带空格
      ca_cert: ./ca.cert # ./caA.cert:./caB.cert
```

## 5.3 代码

本节会展示一个完整的例子，客户端调用 "搭建泛 HTTP RPC 服务" 中第 6.5 节提供的服务，客户端采用 fasthttp 协议，并在请求中携带 "request" 报文头，并打印 RPC 响应 Msg 和响应头里的 `reply` 字段。

### 5.3.1 准备

在开发之前，客户端需要获取服务 PB IDL 文件生成的桩代码。通常情况下服务端在发布服务的同时，会提供桩代码的 git 代码路径，客户端直接引用就行。如何服务端的桩代码没有上传到 git 仓库，可以把桩代码拷贝到客户端工程下，并在 go.mod 里面 replace 本地路径进行引用。

```shell
# 创建工程
go mod init client

# 拷贝桩代码, 并修改 go.mod
echo "replace git.code.oa.com/trpcprotocol/test/rpchttp => ./stub/git.code.oa.com/trpcprotocol/test/rpchttp" >> go.mod

# 添加main.go
vim main.go
```

### 5.3.2 开发

客户端的代码如下

```go
package main

import (
    "context"
    
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
    pb "git.code.oa.com/trpcprotocol/test/rpchttp"
    "github.com/valyala/fasthttp"
)

func main() {
    // 如果需要使用框架配置文件来设置 client 端票配置
    trpc.NewServer()
    // 创建 ClientProxy, 设置协议为 HTTP 协议，序列化为 Json
    proxy := pb.NewHelloClientProxy()
    reqHeader := &http.FastHTTPClientReqHeader{}
    // 必须留空或设置为 "POST"
    reqHeader.Method = "POST"
    // 为 HTTP Head 添加 request 字段
    reqHeader.DecorateRequest = func(r *fasthttp.Request) *fasthttp.Request {
        r.Header.Add("request", "test")
        return r
    }
    // 设置 Cookie
    cookie := fasthttp.AcquireCookie()
    defer fasthttp.ReleaseCookie(cookie)
    cookie.SetKey("sample")
    cookie.SetValue("sample")
    cookie.SetHTTPOnly(false)
    old := reqHeader.DecorateRequest
    reqHeader.DecorateRequest = func(r *fasthttp.Request) *fasthttp.Request {
        r = old(r)
        r.Header.Add("Cookie", cookie.String())
        return r
    }
    req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    rspHead := &http.FastHTTPClientRspHeader{}
    // 发送 HTTP RPC 请求
    rsp, err := proxy.SayHello(context.Background(), req,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
        client.WithTarget("ip://127.0.0.1:8000"))
    if err != nil {
        log.Warn("get http response err")
        return
    }
    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Peek("reply")
    log.Infof("data is %s, request head is %q\n", rsp, replyHead)
}

```

配置文件如下

```yaml
global:                               # 全局配置
  namespace: Development              # 环境类型，分正式 production 和非正式 development 两种类型
  env_name: test                      # 环境名称，非正式环境下多环境的名称

client:                               # 客户端调用的后端配置
  timeout: 1000                       # 针对所有后端的请求最长处理时间
  namespace: Development              # 针对所有后端的环境
  filter:                             # 针对所有后端调用函数前后的拦截器列表
  service:                            # 针对单个后端的配置
    - name: trpc.test.stdhttp.hello   # 后端服务的 service name
      namespace: Development          # 后端服务的环境
      network: tcp                    # 后端服务的网络类型 tcp udp 配置优先
      protocol: fasthttp              # 应用层协议 trpc http
      target: ip://127.0.0.1:8000     # 请求服务地址 可用任意 selector 如 dns://xx, polaris://xx
      timeout: 1000                   # 请求最长处理时间
```

### 5.3.3 运行

```shell
go run main.go
2024-08-16 21:32:02.540 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=32: CPU quota undefined
2024-08-16 21:32:02.541 INFO    fasthttprpc-client/main.go:59   data is msg:"Hello, World!", request head is "tested"
```

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
