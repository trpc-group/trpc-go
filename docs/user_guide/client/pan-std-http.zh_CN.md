最新的文档内容可以同步参考代码仓库中的 README：
<https://git.woa.com/trpc-go/trpc-go/blob/master/http/README.zh_CN.md>

（其中包含了 HTTPS 等配置以及各种常见场景的示例）

# 1 前言

tRPC-Go 框架对**“泛 HTTP 标准服务”**调用提供了一套统一的调用接口。它一方面简化了服务的调用，同时也整合了服务治理的能力，包括服务寻址、调用链跟踪、监控上报等，为开发人员提供了类似于 RPC 调用的统一开发风格和功能体验。本文会着重介绍如何开发“泛 HTTP 标准服务”客户端，包括接口的使用、协议的配置、以及开发中的一些典型用法。泛 HTTP 协议”特指使用 http 语义的 http，https，http2 和 http3 协议。

在真正开始之前，用户需要掌握以下知识：

- 关于什么是泛 HTTP 标准服务，它和泛 HTTP RPC 服务的区别，请参考 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)
- 关于客户端开发中涉及的基本概念和开发流程，请参考 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)

tRPC-Go 从 v19.0.0 后支持 fasthttp 调用泛 HTTP 标准服务，[使用 fasthttp 调用泛 HTTP 标准服务](#5-使用-fasthttp-调用泛-http-标准服务)。

在设计上，tfasthttp 在行为和用法上尽可能地与 thttp 保持一致，但由于各种原因（主要是 `net/http` 与 `fasthttp` 带来的不一致），其用法可能兼容性较差。

本文主要从如何使用出发，指导用户快速上手 fasthttp，关于细节，请用户查看 [tfasthttp 使用指南](https://doc.weixin.qq.com/doc/w3_Ac0AYwanAIUfx1rVLYYTm2A4u2oHj?scode=AJEAIQdfAAowr0OpC7Ac0AYwanAIU&version=4.1.28.6010&platform=win)。

# 2 接口

tRPC-Go 框架对于泛 HTTP 标准服务的调用和 tRPC 服务的调用一样，都采用了“ClientProxy”来封装服务调用过程。不同点在于：泛 HTTP 标准服务并不需要通过 IDL 文件来生成业务接口，服务的调用统一抽象成“Get”，“Post”，“Put”，“Delete”四个接口。业务层接口数据的定义是由业务代码自行实现。本节主要从客户端创建、服务接口调用、HTTP 报文头三个方面来介绍客户端 API。在第 4 节，我们会通过示例来展示如何使用这些 API。

## 2.1 客户端创建

由于框架对于泛 HTTP 标准服务的调用是采用“ClientProxy”来封装的，对于每个 HTTP 服务后端，用户需要先创建一个 ClientProxy，接口定义为：

```go
// NewClientProxy 新建一个 http 后端请求代理 必传参数 http 服务名
// name 后端 http 服务的服务名，主要用于配置 key，监控上报，name 格式遵循对应名字系统的定义规范
var NewClientProxy = func(name string, opts ...client.Option) Client
```

其中“name”为服务的 Naming Service，可以通过名字服务来寻址。用户可以通过 opts 来设置 client 的配置，具体 API 函数请参考 [客户端开发向导](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)。这里列出在 HTTP 协议中经常使用的协议，序列化，压缩等 API 的定义：

```go
// WithProtocol 指定服务协议名字
func WithProtocol(s string) Option
// WithTLS 指定 tls 配置，支持单向认证，双向认证
// 框架版本 >= v0.19.0 时，支持在 certFile, keyFile, caFile 参数多个文件路径
// 两个文件路径之间用英文冒号 `:` 分隔，中间不要带空格，如：WithTLS("a.crt:b.crt", "a.key:b.key", "caA.pem:caB.pem")
func WithTLS(certFile, keyFile, caFile string) Option

// 设置序列化类型：需要使用 tRPC 协议对应的数值，框架会自动转变成 "Content-Type"
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

tRPC-Go 框架支持用户自定义序列化类型和压缩方式，在添加序列化类型和压缩方式时，客户端和服务端都必须添加。具体操作请参考 [搭建泛 HTTP RPC 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=490796254) 第 **7.1** 和 **7.2** 章节

## 2.2 HTTP 报文头处理

框架提供了以下接口来设置 HTTP 请求和响应报文头：

```go
// 以下接口定义在 git.code.oa.com/trpc-go/trpc-go/client 包中
// WithReqHead 设置后端请求包头
func WithReqHead(h interface{}) Option
// WithRspHead 设置后端响应包头
func WithRspHead(h interface{}) Option

// 以下接口定义在 git.code.oa.com/trpc-go/trpc-go/http 包中
// ClientReqHeader 封装 http client 请求的上下文
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

## 2.3 服务接口调用

在创建好 "ClientProxy" 之后，用户就可以使用 "Get"，"Post"，"Put"，"Delete" 接口来调用标准 HTTP 服务了。接口定义为：

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

上面的函数中的 opts 可以为每一次的服务调用单独设置 client 配置。**注意：如果 opts 中使用了“WithReqHead()”, 业务则需要为“ClientReqHeader”中 Method 设置正确的值。** 原因在于如果业务自行设置 Head 头，则此 Head 会替换掉框架设置的 Head 值。

# 3 配置

对于客户端配置，框架提供了两种设置方式：**框架配置文件方式** 和**Option 配置**（第 2.1 节已介绍）。系统推荐使用框架配置文件方式，这样可以和代码解耦，便于管理和修改。对于客户端通用配置，这里不做赘述，具体请参考 [客户端开发向导](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)。本节重点介绍 协议、序列化、压缩方式在框架配置文件中的定义。

## 3.1 协议

协议在这里特指底层协议使用 "http", "https", "http2", "http3" 中的其中一种，客户端协议的设置取决于服务端的设置。协议配置在框架配置文件中的位置为：

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

泛 HTTP 标准服务的客户端开发和服务端不同，客户端框架实现了 HTTP Body 的序列化/反序列化，用户只需要设置序列化类型即可，服务端根据客户端携带的 "Content-Type" 来进行反序列化。序列化配置在框架配置文件中的位置为：

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

泛 HTTP 标准服务的客户端开发和服务端不同，客户端框架实现了 HTTP Body 的压缩/解压缩，用户只需要设置压缩方式即可，服务端根据客户端携带的 "Content-Encoding" 来进行解压缩。压缩方式配置在框架配置文件中的位置为：

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

本节会展示一个完整的例子，客户端调用“搭建泛 HTTP 标准服务”中第 4.1 节提供的服务，客户端端采用“json”序列化格式，http 协议，并在 HTTP 请求中携带“request”，打印 HTTP 响应报文和响应头里“reply”字段和响应数据。

## 4.1 代码

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
    // 如果需要使用框架配置文件来设置 client 端的配置
    trpc.NewServer()

    // 创建 ClientProxy, 设置协议为 HTTP 协议，序列化为 Json
    httpCli := http.NewClientProxy("trpc.test.stdhttp.hello",
        client.WithProtocol("http"),
        client.WithSerializationType(codec.SerializationTypeJSON))

    reqHeader := &http.ClientReqHeader{}
    // 必须设置正确的 Method
    reqHeader.Method = "POST"
    // 为 HTTP Head 添加 request 字段
    reqHeader.AddHeader("request", "test")

    req := &Data{Msg: "Hello, I am stdhttp client!"}
    rsp := &Data{}
    rspHead := &http.ClientRspHeader{}

    // 发送 HTTP POST 请求
    // req 中需要进行序列化发送给下游的属性需要【大写】
    err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead))
    if err != nil {
        log.Warn("get http response err")
        return
    }

    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

## 4.2 配置

对于客户端的配置，我们更推荐使用框架配置文件来实现，这样可以实现代码和配置的分离。客户端配置示例如下：

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
      protocol: http                  # 应用层协议 trpc http
      target: ip://127.0.0.1:800      # 请求服务地址 可用任意 selector 如 dns://xx, polaris://xx
      timeout: 1000                   # 请求最长处理时间
```

## 4.3 客户端通过流式（io.Reader）上传文件

需要 trpc-go 版本 >= v0.13.0

关键点在于将一个 `io.Reader` 填到 `thttp.ClientReqHeader.ReqBody` 字段上 (`body` 是一个 `io.Reader`):

```go
reqHeader := &thttp.ClientReqHeader{
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

## 4.4 客户端通过流式（io.Reader）下载文件

需要 trpc-go 版本 >= v0.13.0

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

## 4.5 客户端服务端收发 HTTP chunked 数据

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

## 4.6 客户端提交 Form 数据

只需指定 `client.WithSerializationType(codec.SerializationTypeForm)` 并传入类型为 `url.Values` 的请求即可，示例如下：

```go
c := thttp.NewClientProxy(
    "trpc.app.server.Service_http",
    client.WithTarget("ip://localhost:8080"),
)
req := make(url.Values)
req.Add("key", "value")
rsp := &codec.Body{}
c.Post(context.Background(), "/", req, rsp,
    client.WithSerializationType(codec.SerializationTypeForm),
)
```

由于原始 form/v4 库在序列化 map 类型数据时，存在多加 '[]' 的情况，所以用户可以根据情况修改 FormSerialization 的 MapType 字段，该字段默认
为 false 即走 form/v4 库的逻辑即对于 map 类型序列化后结果为 [key]=value，而 MapType 为 true 序列化后结果为 key=value。
详细信息见 <https://git.woa.com/trpc-go/trpc-go/issues/986>。

```go
s := codec.GetSerializer(codec.SerializationTypeForm)
serialization, _ := s.(*http.FormSerialization)
serialization.MapType = true
```

## 4.7 客户端接收 SSE 数据

> SSE（Server-Sent Events）是一种基于 HTTP 的应用层协议，用于实时推送服务器端事件到客户端。它允许服务器通过单向连接持续向客户端发送更新，而无需客户端轮询服务器。
> SSE 是一种轻量级的、易于实现的实时通信方式，协议规范可以阅读 [Server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html)。
> 简单来说，使用两个换行符来分隔不同的消息，而每个消息内部使用一个换行符来分隔内容。
> 每个 SSE 消息由以下部分组成：
>
> - **事件类型**（可选）：使用 `event:` 前缀指定事件类型。
> - **数据**：使用 `data:` 前缀指定消息数据，可以包含多行。
> - **ID**（可选）：使用 `id:` 前缀指定消息的唯一标识符。
> - **重试时间**（可选）：使用 `retry:` 前缀指定客户端在连接断开后重新连接的时间间隔（以毫秒为单位）。
> - **注释**：以冒号 (:) 开头，后面可以跟随任意文本内容。客户端会忽略这些注释内容，不会触发任何事件处理程序。
>
> 每个字段之间使用换行符分隔，不同消息之间使用两个换行符分隔。
>
> 以下是一个简单的 SSE 消息示例：
>
> ```raw
> event: message
> data: Hello, world!
> id: 1
> 
> event: update
> data: {"status": "updated"}
> id: 2
> ```

在版本 >= v0.17.0 时，`thttp.ClientRspHeader` 提供了一个名为 `SSEHandler` 的字段，用于注册接收 SSE 数据的回调实现。

在版本 < v0.17.0 时，需要手动进行原始的解析操作。如果需要了解更多细节，可以参考 [收发 SSE](https://git.woa.com/trpc-go/trpc-go/blob/master/http/README.zh_CN.md#%E6%94%B6%E5%8F%91-sse) 和 [收发 SSE (基于 github.com/r3labs/sse )](https://git.woa.com/trpc-go/trpc-go/blob/master/http/README.zh_CN.md#%E6%94%B6%E5%8F%91-sse-%E5%9F%BA%E4%BA%8E-githubcomr3labssse)。

下面展示了在版本 >= v0.17.0 中使用 `thttp.ClientRspHeader.SSEHandler` 的示例，
你也可以参考 [SSE normal example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/normal) 获取更完整的代码。

```go
import (
    // ...
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/server"
    "github.com/r3labs/sse/v2"
    "github.com/stretchr/testify/require"
)

func TestHTTPSendAndReceiveSSE(t *testing.T) {
    // 1. 启动 SSE 协议服务端（简单实现）
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
        // 以下代码在实现 SSE(server-sent events) 时十分必要，可以参考：
        // https://html.spec.whatwg.org/multipage/server-sent-events.html#server-sent-events
        
        // 开始
        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
            return fmt.Errorf("http: ResponseWriter from %T does not implement http.Flusher", w)
        }
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set(thttp.Connection, "keep-alive")
        // 结束
  
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
                return fmt.Errorf("thttp WriteSSE: %v", err)
            }
            flusher.Flush() // 将写入的数据 flush 到客户端，使其可以立即读入到 SSE 事件，而不是等缓冲结束后再一次性发送
            time.Sleep(500 * time.Millisecond) // 模拟服务器延迟，在业务中不必要
        }
        return
    }))
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)
    
    // 2. 使用 thttp 客户端来连接 SSE 服务端
    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    // 规范推荐使用 GET，但是某些服务端可能会要求用 POST
    // 此处 thttp 选用 GET/POST 均可
    reqHeader := &thttp.ClientReqHeader{
        Method: http.MethodPost,
    }
    var data []byte
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: false, // ManualReadBody 默认保留为 false
        // 设置 SSEHandler 来注册接收 SSE 数据后的回调
        // 可以查看 sse.Event 中的具体字段信息来确定用法
        SSEHandler: sseHandler(func(e *sse.Event) error {
            if string(e.Event) == "message" {
                data = append(data, e.Data...)
            }
            return nil
        }),
    }
    req := &codec.Body{Data: []byte("hello")}
    rsp := &codec.Body{}
    // 发起调用，注意：此处调用会持续到 handler 内部返回错误或者对端发送 io.EOF 才结束
    // 从而使得客户端的监控上报为一次完整的 SSE 接收（接收完这一轮的所有 message）的耗时
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
}

type sseHandler func(*sse.Event) error

// Handle 处理 SSE 事件。如果返回的 error 不为空，框架将会终止 HTTP 连接的读取。
func (h sseHandler) Handle(e *sse.Event) error {
    return h(e)
}
```

对于可能返回 SSE 或非 SSE 的接口，客户端提供了以下字段：

- 在版本 >= v0.19.0 时，**`thttp.ClientRspHeader` 提供了 `SSECondition` 和 `ResponseHandler` 两个字段，用于根据服务器的响应采取不同的回调策略**。
  - `SSECondition`: 如果 **`SSECondition` 返回 `true`，且用户实现了 `SSEHandler`**，则回调 `SSEHandler`。用户可以自行实现该接口，可以判断响应头是否包含 `Content-Type: text/event-stream`，但是请注意**并不是所有服务实现都严格遵守此规则**；
  如果将该字段置空，框架将使用默认的实现（返回 `true`）。
  - `ResponseHandler`: 如果 **`SSECondition` 返回 `false`，或用户没有实现 `SSEHandler`**，则回调 `ResponseHandler`。如果用户没有实现该接口，框架的兜底策略为自动读取回包。

- 在版本 < v0.19.0 时，需要**手动进行原始的解析操作，根据响应区分是否为 SSE 消息，然后使用 `io.Reader` 采取不同的策略进行流式读取回包**（见上一节）。

请注意，**`SSEHandler` 和 `ResponseHandler` 均需在设置 `ManualReadBody` 为 `false` 时才会生效**。

下面展示了在版本 >= v0.19.0 中使用 `thttp.ClientRspHeader` 的 `SSECondition`, `SSEHandler` 和 `ResponseHandler` 的示例，
你也可以参考 [SSE multiple example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/multiple) 获取更完整的代码。

如果客户端需要结合 SSE 做转发，可以参考 [这里](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/multiple/proxy) 。

```go
import (
    // ...
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/server"
    "github.com/r3labs/sse/v2"
    "github.com/stretchr/testify/require"
)

func TestHTTPSendAndReceiveSSEAndNormalResponse(t *testing.T) {
     // 1. 启动 SSE 协议服务端（简单实现）
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
    isSSE := true // 是否发送 SSE，初始为 true
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 切换 SSE 的开关，每次请求都会切换一次
        defer func() { isSSE = !isSSE }()
        if isSSE {
            sseHandleFunc(w, r)
            return
        }
        normalHandleFunc(w, r)
    }))

    // 2. 使用 thttp 客户端来连接 SSE 服务端
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    ) 
    // 规范推荐使用 GET，但是某些服务端可能会要求用 POST
    // 此处 thttp 选用 GET/POST 均可
    reqHeader := &thttp.ClientReqHeader{
        Method: http.MethodPost,
    }

    var data []byte
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: false, // ManualReadBody 默认保留为 false
        // 可以自行实现 SSECondition 的逻辑，如果置空则框架采用默认的 SSECondition (return true)
  SSECondition: func(r *http.Response) bool { // 这里采用自定义实现，判断响应头的 header
            return r.Header.Get("Content-Type") == "text/event-stream"
        },
        // 设置 ResponseHandler 来注册处理非 SSE 数据或普通的 HTTP 响应
        ResponseHandler: rspHandler(func(r *http.Response) error {
            bs, err := io.ReadAll(r.Body)
            if err != nil {
                return err
            }
            t.Logf("Receive http response: %s", string(bs))
            data = append(data, bs...)
            return nil
        }),
        // 设置 SSEHandler 来注册接收 SSE 数据后的回调
        // 可以查看 sse.Event 中的具体字段信息来确定用法
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
    // 第偶数次（从 0 开始）响应是 SSE 消息，第奇数次是普通 HTTP 响应，但是这里两种响应的结果在经过不同 Handler 处理之后应该相同
    for i := 0; i < 4; i++ {
        t.Run(fmt.Sprintf("request "+strconv.Itoa(i)), func(t *testing.T) {
            data = []byte{} // 先清空数据
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

// 发送 SSE 响应
func sseHandleFunc(w http.ResponseWriter, r *http.Request) {
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
}

// 发送非 SSE 响应
func normalHandleFunc(w http.ResponseWriter, r *http.Request) {
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

func (h sseHandler) Handle(e *sse.Event) error {
    return h(e)
}

type rspHandler func(*http.Response) error

func (h rspHandler) Handle(r *http.Response) error {
    return h(r)
}
```

这里提供一个表格来帮助大家理解 `thttp.ClientRspHeader` 字段的生效逻辑。简单来说，所有字段均需在 `ManualReadBody` 为 `false` 时才生效；
如果 `SSECondition` 未实现返回 `true`，且实现了 `SSEHandler`，则执行 `SSEHandler`；否则判断 `ResponseHandler` 的实现情况，有则调用 `ResponseHandler`，无则执行框架默认逻辑读取回包。

 | `ManualReadBody` | `SSECondition` | `SSEHandler` | `ResponseHandler` | 效果                           |
|------------------|----------------|--------------|-------------------|------------------------------|
 | true             | -              | -            | -                 | 所有 Handler 都不执行，用户手动执行响应读取逻辑 |
 | false            | 未实现 / 返回 true  | 实现           | -                 | 执行 `SSEHandler`              |
 | false            | -              | nil          | 实现                | 执行 `ResponseHandler`         |
 | false            | -              | nil          | nil               | 执行框架默认逻辑，读取回包                |
 | false            | 返回 false       | -            | 实现                | 执行 `ResponseHandler`         |
 | false            | 返回 false       | -            | nil               | 执行框架默认逻辑，读取回包                |

## 4.8 客户端自定义 Decode 时错误处理逻辑

在 tRPC-Go v0.19.0 后，用户可以修改 ClientCodec 中的 ErrHandler 字段来自定义 Decode 时如何处理错误。

默认实现与 tRPC-Go v0.19.0 之前版本一致。注意，ClientCodec 的 ErrHandler 有兜底策略，如果设置为 nil，则会走默认处理。

若想不做处理，请自行实现 NoopDecodeErrorHandler。

```go
// ClientCodec decodes http client request.
type ClientCodec struct {
    // ErrHandler is error code handle function, which is filled into header by default.
    // Business can set this with http.DefaultClientCodec.ErrHandler = func(rsp, msg, body) ([]byte, error) {}.
    ErrHandler DecodeErrorHandler
}

// DecodeErrorHandler is used to handle error in ClientCodec.Decode()
type DecodeErrorHandler func(rsp *http.Response, msg codec.Msg, body []byte) ([]byte, error)

var defaultDecodeErrHandler = func(rsp *http.Response, msg codec.Msg, body []byte) ([]byte, error) {
    if val := fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcFrameworkErrorCode); val != "" {
        i, _ := strconv.Atoi(val)
        if i != 0 {
            msg.WithClientRspErr(
                errs.NewCalleeFrameError(i, fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcErrorMessage)))
            return nil, nil
        }
    }
    if val := fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcUserFuncErrorCode); val != "" {
        i, _ := strconv.Atoi(val)
        if i != 0 {
            msg.WithClientRspErr(
                errs.New(i, fastop.CanonicalHeaderGet(rsp.Header, canonicalTrpcErrorMessage)))
            return nil, nil
        }
    }
    if rsp.StatusCode >= http.StatusMultipleChoices {
        msg.WithClientRspErr(errs.New(
            rsp.StatusCode,
            fmt.Sprintf("http client codec StatusCode: %s, body: %q", http.StatusText(rsp.StatusCode), body)))
        return nil, nil
    }
    return body, nil
}
```

# 5 使用 fasthttp 调用泛 HTTP 标准服务

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
// 框架版本 >= v0.19.0 时，支持在 certFile, keyFile, caFile 参数配置多个文件路径
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

tfasthttp 和 thttp 客户端的配置差异主要体现在 `protocol`，即从 `protocol: http` -> `protocol: fasthttp` 以下是一个简单的配置例子：

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

本部分将提供与 http 对应的 fasthttp 代码供用户迁移使用。

### 5.3.1 示例

本节会展示一个完整的例子，客户端调用搭建泛 HTTP 标准服务中第 4.1 节提供的服务，客户端采用 json 序列化格式，http 协议，并在 HTTP 请求中携带 request，打印 HTTP 响应报文和响应头里 reply 字段和响应数据。

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
    // 如果需要使用框架配置文件来设置 client 端的配置
    // 其实不推荐纯客户端使用这种方式（side effect）
    trpc.NewServer()

    // 创建 FastHTTPClientProxy, 设置协议为 HTTP 协议，序列化为 Json
    fcp := http.NewFastHTTPClientProxy("trpc.test.stdhttp.hello",
        client.WithSerializationType(codec.SerializationTypeJSON))

    reqHeader := &http.FastHTTPClientReqHeader{}
    // 必须设置正确的 Method
    reqHeader.Method = "POST"
    // 为 FastHTTP Head 添加 request 字段
    reqHeader.DecorateRequest = func(r *fasthttp.Request) *fasthttp.Request {
        r.Header.Add("request", "test")
        return r
    }

    req := &Data{Msg: "Hello, I am stdhttp client!"}
    rsp := &Data{}
    rspHead := &http.FastHTTPClientRspHeader{}

    // 发送 FastHTTP POST 请求
    // req 中需要进行序列化发送给下游的属性需要【大写】
    err := fcp.Post(context.Background(), "/v1/hello", req, rsp,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead))
    if err != nil {
        log.Warn("get http response err")
        return
    }

    // 获取 FastHTTP 响应报文头中的 reply 字段
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

### 5.3.2 客户端通过流式上传文件

关键点在于将为 fasthttp 请求设置 io.Reader 作为 Body，即调用 `r.SetBodyStream(body, -1)`

与 thttp 不同的是，用户需要使用 DecorateRequest 完成该操作

```go
reqHeader := &thttp.FastHTTPClientReqHeader{
    Method: fasthttp.MethodPost,
    // set by DecorateRequest
    DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
        r.Header.SetContentType(writer.FormDataContentType())
        r.SetBodyStream(body, -1)
        return r
    },
}
```

然后在调用时指定 `client.WithReqHead(reqHeader)`

```go
fcp.Post(context.Background(), "/", req, rsp,
    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
    client.WithSerializationType(codec.SerializationTypeNoop),
    client.WithCurrentCompressType(codec.CompressTypeNoop),
    client.WithReqHead(reqHeader),
)
```

完整示例如下

```go
func TestFastHTTPStreamFileUpload(t *testing.T) {
    // Start server.
    ln := mustListen(t)
    defer ln.Close()
    go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
        h, err := ctx.FormFile("field_name")
        if err != nil {
            ctx.SetStatusCode(fasthttp.StatusBadRequest)
        }
        ctx.SetStatusCode(fasthttp.StatusOK)
        ctx.Write([]byte(h.Filename))
    })
    // Start client.
    fcp := thttp.NewFastHTTPClientProxy(
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
    reqHeader := &thttp.FastHTTPClientReqHeader{
        Method: fasthttp.MethodPost,
        // set by DecorateRequest
        DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
            r.Header.SetContentType(writer.FormDataContentType())
            r.SetBodyStream(body, -1)
            return r
        },
    }
    req := &codec.Body{}
    rsp := &codec.Body{}
    // Upload file.
    require.Nil(t,
        fcp.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithReqHead(reqHeader),
        ))
    require.Equal(t, []byte(fileName), rsp.Data)
}
```

### 5.3.3 客户端通过流式下载文件

与 thttp 客户端大同小异，注意类型的变化。完整示例如下

```go
func TestFastHTTPStreamRead(t *testing.T) {
    ln := mustListen(t)
    defer ln.Close()
    go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
        fasthttp.ServeFile(ctx, "./README.md")
    })
    fcp := thttp.NewFastHTTPClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    rspHead := &thttp.FastHTTPClientRspHeader{ManualReadBody: true}
    req := &codec.Body{}
    rsp := &codec.Body{}
    require.Nil(t,
        fcp.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithRspHead(rspHead),
        ),
    )
    require.Nil(t, rsp.Data)
    require.NotNil(t, rspHead.Response.Body())
}
```

### 5.3.4 客户端服务器收发 chunked 数据

注意：SetBodyStreamWriter 和 SetBodyStream 的相关辨析，以及
> Access to RequestCtx and/or its members is forbidden from sw.

```go
// SetBodyStream sets request body stream and, optionally body size.
// If bodySize is >= 0, then the bodyStream must provide exactly bodySize bytes before returning io.EOF.
// If bodySize < 0, then bodyStream is read until io.EOF.
// bodyStream.Close() is called after finishing reading all body data if it implements io.Closer.
// Note that GET and HEAD requests cannot have body.
func (req *Request) SetBodyStream(bodyStream io.Reader, bodySize int)


// SetBodyStreamWriter registers the given sw for populating request body.
// This function may be used in the following cases:
// if request body is too big (more than 10MB).
// if request body is streamed from slow external sources.
// if request body must be streamed to the server in chunks (aka `http client push` or `chunked transfer-encoding`).
// Note that GET and HEAD requests cannot have body.
func (req *Request) SetBodyStreamWriter(sw StreamWriter)

// SetBodyStream sets response body stream and, optionally body size.
// If bodySize is >= 0, then the bodyStream must provide exactly bodySize bytes before returning io.EOF.
// If bodySize < 0, then bodyStream is read until io.EOF.
// bodyStream.Close() is called after finishing reading all body data if it implements io.Closer.
func (resp *Response) SetBodyStream(bodyStream io.Reader, bodySize int)

// SetBodyStreamWriter registers the given sw for populating response body.

// This function may be used in the following cases:

// if response body is too big (more than 10MB).
// if response body is streamed from slow external sources.
// if response body must be streamed to the client in chunks (aka `http server push` or `chunked transfer-encoding`).
func (resp *Response) SetBodyStreamWriter(sw StreamWriter)
```

主要逻辑就是通过 DecorateRequest 为请求调用 SetBodyStreamWriter，示例如下

```go
func TestFastHTTPSendReceiveChunk(t *testing.T) {
    // Start server.
    ln := mustListen(t)
    defer ln.Close()
    go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
        b := make([]byte, len(ctx.Request.Body()))
        copy(b, ctx.Request.Body())
        ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
            // 3. Server reads chunks.
            // io.ReadAll will read until io.EOF.
            // fasthttp will automatically handle chunked body reads.
            w.Write(b)
            // 4. Server sends chunks.
            for i := 0; i < 10; i++ {
                fmt.Fprintf(w, "this is a rsp number %d\n", i)
                time.Sleep(100 * time.Millisecond)
            }
            // Do not forget flushing streamed data.
            if err := w.Flush(); err != nil {
                return
            }
        })
    })
    // Start client.
    fcp := thttp.NewFastHTTPClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    // 1. Client sends chunks.
    reqHead := &thttp.FastHTTPClientReqHeader{
        Method: fasthttp.MethodPost,
        DecorateRequest: func(r *fasthttp.Request) *fasthttp.Request {
            r.Header.SetContentType("text/plain")
            r.SetBodyStreamWriter(func(w *bufio.Writer) {
                for i := 0; i < 10; i++ {
                    fmt.Fprintf(w, "this is a req number %d\n", i)
                    time.Sleep(100 * time.Millisecond)
                }
                // Do not forget flushing streamed data.
                if err := w.Flush(); err != nil {
                    return
                }
            })
            return r
        },
    }
    // Enable manual body reading in order to
    // disable the framework's automatic body reading capability,
    // so that users can manually do their own client-side streaming reads.
    rspHead := &thttp.FastHTTPClientRspHeader{
        ManualReadBody: true,
    }
    req := &codec.Body{}
    rsp := &codec.Body{}
    require.Nil(t,
        fcp.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithCurrentCompressType(codec.CompressTypeNoop),
            client.WithReqHead(reqHead),
            client.WithRspHead(rspHead),
        ),
    )
    require.Nil(t, rsp.Data)
    // 2. Client reads chunks.
    t.Log(string(rspHead.Response.Body()))
    require.Equal(t, "chunked", string(reqHead.Request.Header.Peek("Transfer-Encoding")))
    require.Equal(t, "chunked", string(rspHead.Response.Header.Peek("Transfer-Encoding")))
}
```

### 5.3.5 客户端自定义 Decode 时错误处理逻辑

在 tRPC-Go v0.19.0 后，用户可以修改 FastHTTPClientCodec 中的 ErrHandler 字段来自定义 Decode 时如何处理错误。

默认实现与 tRPC-Go v0.19.0 之前版本一致。注意，FastHTTPClientCodec 的 ErrHandler 有兜底策略，如果设置为 nil，则会走默认处理。

若想不做处理，请自行实现 NoopFastHTTPDecodeErrorHandler。

```go
// FastHTTPClientCodec is the fasthttp client side codec.
type FastHTTPClientCodec struct {
    // ErrHandler is error code handle function, which is filled into header by default. Business can
    // set this with thttp.DefaultFastHTTPClientCodec.ErrHandler = func(rsp, msg, body) ([]byte, error) {}.
    ErrHandler FastHTTPDecodeErrorHandler
}

// FastHTTPDecodeErrorHandler is used to handle error in FastHTTPClientCodec.Decode()
type FastHTTPDecodeErrorHandler func(rsp *fasthttp.Response, msg codec.Msg, body []byte) ([]byte, error)

var defaultFastHTTPDecodeErrHandler = func(rsp *fasthttp.Response, msg codec.Msg, body []byte) ([]byte, error) {
    if fec := string(rsp.Header.Peek(canonicalTrpcFrameworkErrorCode)); fec != "" {
        frameworkErrcode, err := strconv.Atoi(fec)
        if err != nil {
            return nil, err
        }
        if frameworkErrcode != 0 {
            msg.WithClientRspErr(
                errs.NewCalleeFrameError(
                    frameworkErrcode,
                    string(rsp.Header.Peek(canonicalTrpcErrorMessage)),
                ),
            )
            return nil, nil
        }
    }
    if uec := string(rsp.Header.Peek(canonicalTrpcUserFuncErrorCode)); uec != "" {
        userFuncErrcode, err := strconv.Atoi(uec)
        if err != nil {
            return nil, err
        }
        if userFuncErrcode != 0 {
            msg.WithClientRspErr(
                errs.New(
                    userFuncErrcode,
                    string(rsp.Header.Peek(canonicalTrpcErrorMessage)),
                ),
            )
            return nil, nil
        }
    }
    // If rsp.StatusCode() >= 300, tfasthttp will invoke msg.WithClientRspErr.
    // Align with thttp.
    if rsp.StatusCode() >= fasthttp.StatusMultipleChoices {
        msg.WithClientRspErr(
            errs.New(rsp.StatusCode(), fmt.Sprintf("fasthttp client codec StatusCode: %s, body: %q",
                fasthttp.StatusMessage(rsp.StatusCode()), rsp.Body()),
            ),
        )
        return nil, nil
    }
    return body, nil
}
```

# 6 FAQ

**Q1: tfasthttp 客户端只能调用 fasthttp 服务器吗？**

不是，tfasthttp 只是将发送请求的方式从 `net/http` 变成了 `fasthttp`，可以调用一切接受 http 请求的服务器。

其余请参考搭建泛 HTTP 标准服务的 [FAQ](https://iwiki.woa.com/p/490796278#5-faq) 部分。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
