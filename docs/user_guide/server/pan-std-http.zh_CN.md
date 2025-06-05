# 1 前言

**“泛 HTTP 标准服务”** 为业务开发者提供了既可以用 **`net/http` 标准库方式来开发 HTTP 服务，同时也能复用框架的服务治理能力，如自动上报监控，模调，调用链等关键信息**。“泛 HTTP 标准服务”特指使用 http 语义的 http，https，http2 和 http3 协议。通过本文的介绍，旨在为用户提供如何搭建“泛 HTTP 标准服务”，并对一些常见的使用场景做介绍。

在真正开始之前，首先需要掌握以下知识：

- 关于如何使用 `net/http` 开发 http 服务，请参考 [这里](https://golang.org/pkg/net/http/ "这里") 了解 `net/http` 库的用法。
- 关于“泛 HTTP 标准服务”与“泛 HTTP RPC 服务”的区别，请参考 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=490796254 "这里")。
- 关于 Proto Service 与 Naming Service 的关系，请参考 [tRPC 术语介绍](https://iwiki.woa.com/pages/viewpage.action?pageId=490794774 "tRPC 术语介绍")。

tRPC-Go 从 v0.19.0 后支持 fasthttp 搭建泛 HTTP 标准服务，[使用 fasthttp 搭建泛 HTTP 标准服务](#5-基于-fasthttp-搭建泛-http-标准服务)。

在设计上，tfasthttp 在行为和用法上尽可能地与 thttp 保持一致，但由于各种原因（主要是 `net/http` 与 `fasthttp` 带来的不一致），其用法可能兼容性较差。

本文主要从如何使用出发，指导用户快速上手 fasthttp，关于细节，请用户查看 [tfasthttp 使用指南](https://doc.weixin.qq.com/doc/w3_Ac0AYwanAIUfx1rVLYYTm2A4u2oHj?scode=AJEAIQdfAAowr0OpC7Ac0AYwanAIU&version=4.1.28.6010&platform=win)。

# 2 接口介绍

对泛 HTTP 标准服务，tRPC-Go 框架在报文处理上只负责 HTTP 原始报文的接收和发送。HTTP 报文的序列化/反序列化，压缩/解压缩以及接口定义均需要业务按 `net/http` 提供的 API 自行实现。框架提供了 URL 注册模式 和 Mux 注册模式。

## 2.1 URL 注册模式

URL 注册模式是用户直接注册接口的 URL 和处理函数的方式。框架提供的接口包括：

```go
// URL 注册函数：pattern 为 http 请求的 URL，handler 为路由处理函数
func HandleFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request) error)

// 注册 HTTP 标准服务
func RegisterNoProtocolService(s server.Service)
```

泛 HTTP 标准服务在内部实现上，同样采用 Proto Service 向 Naming Service 注册的方式来实现服务的组合。Proto Service 不需要用户定义，由框架默认创建。`HandleFunc()` 函数用于把路由函数以 **pattern** 做为 rpc name 注册到 Proto Service。`RegisterNoProtocolService()` 用于实现把默认的 Proto Service 注册到 Naming Service。

## 2.2 Mux 注册模式

Mux 注册模式是用户只需要注册 HTTP 标准的 ServeMux Handler 就可以了，用于业务使用第三方的插件路由。框架提供的接口包括：

```go
func RegisterNoProtocolServiceMux(s server.Service, mux http.Handler)
```

同样框架会默认创建 Proto Service，`RegisterNoProtocolServiceMux()` 用于实现把默认的 Proto Service 注册到 Naming Service。

# 3 服务定义

对于泛 HTTP 标准服务，我们可以在 trpc_go.yaml 框架配置文件中通过 `protocol` 字段来指定具体协议类型。

## 3.1 作为 HTTP 服务

我们可以通过设置 `protocol` 为 `http_no_protocol`，即可启动一个无协议的 HTTP 服务。

```yaml
...
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: http_no_protocol                   # 应用层协议 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
```

## 3.2 作为 HTTPS 服务

我们可以通过设置 `protocol` 为 `http_no_protocol`，并同时设置私钥 `tls_key` 和证书 `tls_cert`，即可启动一个 https 服务。https 协议分为 **单向认证** 和 **双向认证** 两种。

**框架版本 >= v0.19.0 时**，支持在 `tls_key`, `tls_cert` 和 `ca_cert` 字段配置多个文件路径，两个文件路径之间用 **英文冒号`:`** 分隔，中间不要带空格。

**单向认证**：只有一方验证另一方是否合法，通常是客户端验证服务端，因此服务端配置只需要设置 `tls_key`、`tls_cert` 即可开启单向认证。一般面向公众的 HTTPS 网站都是单向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: http_no_protocol                   # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

**双向认证**：服务端与客户端需要互相验证，在单向认证的基础上，增加 `ca_cert` 配置来验证客户端的合法性。一般银行等金融网站使用双向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: http_no_protocol                   # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
      ca_cert: ca.cert                             # ca 证书，用于校验 client 证书，以更严格识别客户端的身份，限制客户端的访问
      # ca_cert: ./caA.cert:./caB.cert             # 多个 ca 证书，框架版本 >= v0.19.0
```

## 3.3 作为 HTTP/2 服务

因为 http2 协议需要在 https 协议的基础上使用，所以我们需要通过设置 `protocol` 为 `http2_no_protocol`，并设置 TLS 配置即可启动一个 http2 服务。http2 同样支持 **单向认证** 和 **双向认证** 两种方式，具体参考 https 的配置。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: http2_no_protocol                  # 应用层协议 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

## 3.4 作为 HTTP/3 服务

因为 http3 协议需要在 https 协议的基础上使用，所以我们需要通过设置 `network` 为 **`udp`**，`protocol` 为 `http3`，并设置 TLS 配置即可启动一个 http3 服务。http3 同样支持 **单向认证** 和 **双向认证** 两种方式，具体参考 https 的配置。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: udp                                 # 网络监听类型 tcp udp
      protocol: http3                              # 应用层协议 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

# 4 代码示例

本节我们会通过示例介绍几种常见的场景： **普通服务** 、 **使用 Mux 的服务** 、 **协议代理** 、 **SSE 服务** 和 **前端服务** 等。

## 4.1 普通服务

本示例实现了一个 "Hello World" 的简单 HTTP 服务，在示例中我们展示了 HTTP Head 的读写，Cookie 的设置以及如何设置 HTTP 状态码。

可以通过以下命令进行验证：

``` shell
curl -X POST -d '{msg:"hello"}' -H "Content-Type:application/json" -H "request:test" "http://127.0.0.1:8000/v1/hello" -v
```

```go
package main

import (
    "encoding/json"
    "io/ioutil"
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/log"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

// Data 请求报文数据
type Data struct {
    Msg string
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // 获取请求报文头里的 "request" 字段
    reqHead := r.Header.Get("request")

    // 获取请求报文中的数据
    msg, _ := ioutil.ReadAll(r.Body)
    log.Infof("data is %s, request head is %s\n", msg, reqHead)

    // 为响应报文设置 Cookie
    cookie := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
    http.SetCookie(w, cookie)
    // 注意：使用 ResponseWriter 回包时，Set/WriteHeader/Write 这三个方法必须严格按照以下顺序调用
    w.Header().Set("Content-type", "application/json")
    // 为响应报文头添加“reply”字段
    w.Header().Add("reply", "tested")

    // 为响应报文设置 HTTP 状态码
    // w.WriteHeader(403)

    // 为响应报文设置 Body
    rsp, _ := json.Marshal(&Data{Msg: "Hello, World!"})
    w.Write(rsp)

    return nil
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.HandleFunc("/v1/hello", handle)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置文件 trpc_go.yaml 的配置为：

```yaml
global:                               # 全局配置
  namespace: Development              # 环境类型，分正式 production 和非正式 development 两种类型
  env_name: test                      # 环境名称，非正式环境下多环境的名称

server:                               # 服务端配置
  app: test                           # 业务的应用名
  server: stdhttp                     # 进程服务名
  service:                            # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp   # service 的路由名称
      ip: 127.0.0.1                   # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                      # 服务监听端口 可使用占位符 ${port}
      network: tcp                    # 网络监听类型 tcp udp
      protocol: http_no_protocol      # 应用层协议 trpc http
      timeout: 1000                   # 请求最长处理时间 单位 毫秒
```

## 4.2 使用 Mux 的服务

本节展示如何使用 gorilla/mux 和 trpc-go 框架配合来实现 http 标准服务，而 fasthttp 没有提供 mux 功能。

```go
package main

import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "github.com/gorilla/mux"
)

func main() {
    s := trpc.NewServer()

    // 路由注册
    router := mux.NewRouter()
    router.HandleFunc("/{dir0}/{dir1}/{day}/{hour}/{vid:[a-z0-9A-Z]+}_{index:[0-9]+}.jpg", URLHandle).
        Methods("GET")

    // 服务注册
    thttp.RegisterNoProtocolServiceMux(s, router)

    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}

// URLHandle 处理 url 请求
func URLHandle(w http.ResponseWriter, r *http.Request) {
    //取 url 中的参数
    vars := mux.Vars(r)
    vid := vars["vid"]
    index := vars["index"]

    log.Infof("vid: %s, index: %s", vid, index)
}
```

框架配置同 4.1 章节。

## 4.3 协议代理

本节展示的示例是：服务作为一个 Proxy，接收标准 HTTP 服务请求，然后转化成 tRPC 协议格式向后端的 tRPC 服务发送请求。
可以通过以下命令进行验证：

``` shell
curl -X POST -d "hello" -H "Content-Type:application/text" "http://127.0.0.1:8000/v1/hello" -v
```

```go
package main

import (
    "context"
    "io/ioutil"
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/client"
    pb "git.code.oa.com/trpcprotocol/test/helloworld"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func handle(w http.ResponseWriter, r *http.Request) error {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "can't read body", http.StatusBadRequest)
        return nil
    }

    proxy := pb.NewGreeterClientProxy()
    req := &pb.HelloRequest{Msg: string(body[:])}

    // 向 tRPC 服务请求
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTarget("ip://127.0.0.1:8001"))
    if err != nil {
        http.Error(w, "call fails!", http.StatusBadRequest)
        return nil
    }

    // 回响应给 HTTP 客户端
    w.Header().Set("Content-type", "application/text")
    w.Write([]byte(rsp.Msg))

    return nil
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.HandleFunc("/v1/hello", handle)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置文件 trpc_go.yaml 配置为：

```yaml
global:                             # 全局配置
  namespace: Development            # 环境类型，分正式 production 和非正式 development 两种类型
  env_name: test                    # 环境名称，非正式环境下多环境的名称

server:                             # 服务端配置
  app: test                         # 业务的应用名
  server: hello                     # 进程服务名
  service:                          # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp # service 的路由名称
      ip: 127.0.0.1                 # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                    # 服务监听端口 可使用占位符 ${port}
      network: tcp                  # 网络监听类型  tcp udp
      protocol: http_no_protocol    # 应用层协议 trpc http
      timeout: 1000                 # 请求最长处理时间 单位 毫秒
```

## 4.4  文件下载服务

本示例实现了一个文件下载的简单 HTTP 服务，在示例中我们展示了指定文件的读取及文件的返回。

可以通过以下命令进行验证：

```shell
curl -X POST -d "filename=hello.txt" "http://127.0.0.1:8000/test/hello" -v
```

```go
package main

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func downloadHandler(w http.ResponseWriter, r *http.Request) error {
    r.ParseForm()
    fileName := r.Form["filename"]
    fileNames := url.QueryEscape(fileName[0])
    w.Header().Add("Content-Type", "application/octet-stream;Charset=utf-8")
    w.Header().Add("Content-Disposition", "attachment; filename=\""+fileNames+"\"")
    w.Header().Add("Content-Transfer-Encoding", "binary")

    // 文件存放地址
    path := "/files/"
    file, err := os.Open(path + fileName[0])
    if err != nil {
        fmt.Println("文件不存在")
        return err
    }
    defer file.Close()

    // 为响应报文设置 Body
    content, err := ioutil.ReadAll(file)
    if err != nil {
        fmt.Println("读取文件内容失败")
        return err
    }
    w.Write(content)
    return nil
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.HandleFunc("/test/hello", downloadHandler)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置同 4.1.1 章节。

## 4.5 配置服务端参数（读/写超时、Header 最大大小等）

通过修改使用的 transport 变量来实现，比如框架默认进行的注册为：

```go
// http
// Server transport (protocol file service).
transport.RegisterServerTransport(protocol.HTTP, DefaultServerTransport)
transport.RegisterServerTransport(protocol.HTTPS, DefaultHTTPSServerTransport)
transport.RegisterServerTransport(protocol.HTTP2, DefaultHTTP2ServerTransport)
// Server transport (no protocol file service).
transport.RegisterServerTransport(protocol.HTTPNoProtocol, DefaultServerTransport)
transport.RegisterServerTransport(protocol.HTTPSNoProtocol, DefaultHTTPSServerTransport)
transport.RegisterServerTransport(protocol.HTTP2NoProtocol, DefaultHTTP2ServerTransport)
```

用户可以通过修改 `DefaultServerTransport` 中的 `Server` 字段以提供额外的 HTTP 服务配置，比如：

```go
import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    s, ok := thttp.DefaultServerTransport.(*thttp.ServerTransport)
    if !ok { panic("...") }
    // 目前支持以下参数做配置：
    s.Server = &http.Server{
        ReadTimeout:       time.Second * 10,
        ReadHeaderTimeout: time.Second * 10,
        WriteTimeout:      time.Second * 10,
        MaxHeaderBytes:    1024,
        IdleTimeout:       time.Second * 10,
        ConnState: func(c net.Conn, cs stdhttp.ConnState) {
            // ...
        },
        ErrorLog: nil,
        ConnContext: func(ctx context.Context, c net.Conn) context.Context {
            // ...
            return ctx
        },
    }
    // ...
}
```

用户也可以通过重新注册自定义的 transport 来达到类似的效果：

```go
st := thttp.NewServerTransport(transport.WithReusePort(true))
s, _ := thttp.DefaultServerTransport.(*thttp.ServerTransport)
s.Server = &http.Server{ /* ... */ } // 自定义参数
transport.RegisterServerTransport("http", st)
```

## 4.6 SSE 服务

- 在版本 >= v0.19.0 (未发布时为 master 分支) 时，`thttp` 提供了一个 `WriteSSE` 的函数，用于将 `sse.Event` 结构体按照 SSE 格式快速写进 `io.Writer` 中。用户无需再关心 SSE 数据格式。
- 在版本 < v0.19.0 时，需要**手动拼接响应体**，然后再写入 `http.ResponseWriter` 中。

本示例实现了一个服务端的简单 HTTP SSE 服务，在示例中我们展示了使用 `WriteSSE` 函数封装消息，请确保 trpc-go 版本 >= v0.19.0。
你也可以参考 [SSE example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse) 获取更完整的用法示例

可以通过以下命令进行验证：

```shell
curl -X POST --data-raw "hello" "http://127.0.0.1:8000/v1/hello" -v
```

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"

    "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"

    "github.com/r3labs/sse/v2"
)

func handle(w http.ResponseWriter, r *http.Request) error {
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
        return fmt.Errorf("http: Read request body: %v", err)
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
    return nil
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.HandleFunc("/v1/hello", handle)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.app.server.ServiceSSE"))
    s.Serve()
}
```

框架配置同 4.1.1 章节。

## 4.7 前端服务

本节展示如何使用 gorilla/mux、html/template、embed和 trpc-go 框架配合来实现携带动态数据的前端服务。

服务启动后，可在本地浏览器里输入以下链接验证：

```http request
http://127.0.0.1:8000/class/23/student/jack
```

```go
package main

import (
    "embed"
    "html/template"
    "net/http"
    
    "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "github.com/gorilla/mux"
)

func main() {
    s := trpc.NewServer()
    // 路由注册
    router := mux.NewRouter()
    router.HandleFunc("/class/{class}/student/{name}", getStudent)
    // 服务注册
    thttp.RegisterNoProtocolServiceMux(s, router)
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}

//go:embed *
var tplFS embed.FS
var globalTemplate = template.Must(template.New("").ParseFS(tplFS, "*.tpl"))

func getStudent(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    globalTemplate.ExecuteTemplate(w, "student.tpl", map[string]interface{}{
        "class": vars["class"],
        "name":  vars["name"],
    })
}

```

模板文件命名为 student.tpl

```html
<!DOCTYPE html>
<html>
       <body>
               学生班级：{{.class}}
               学生名字：{{.name}}
       </body>
</html>

```

框架配置同 4.1 章节。

# 5 基于 fasthttp 搭建泛 HTTP 标准服务

## 5.1 接口介绍

对泛 HTTP 标准服务，tRPC-Go 框架在报文处理上只负责 HTTP 原始报文的接收和发送。HTTP 报文的序列化/反序列化，压缩/解压缩以及接口定义均需要业务按 `fasthttp` 提供的 API 自行实现。框架为 tfasthttp 提供了 URL 注册模式。

URL 注册模式是用户直接注册接口的 URL 和处理函数的方式。框架提供的接口包括：

```go
// URL 注册函数：pattern 为 http 请求的 URL，handler 为路由处理函数
func FastHTTPHandleFunc(pattern string, handler func(requestCtx *fasthttp.RequestCtx))

// 注册泛 HTTP 标准服务
func RegisterNoProtocolService(s server.Service)
```

泛 HTTP 标准服务在内部实现上，同样采用 Proto Service 向 Naming Service 注册的方式来实现服务的组合。Proto Service 不需要用户定义，由框架默认创建。`HandleFunc()` 函数用于把路由函数以 **pattern** 做为 rpc name 注册到 Proto Service。`RegisterNoProtocolService()` 用于实现把默认的 Proto Service 注册到 Naming Service。

如果想要使用 mux，其实也可以将 mux 对应的 handler 直接注册进来，以使用 `github.com/qiangxue/fasthttp-routing` 作为例子：

主要注意使用 `thttp.FastHTTPHandleFunc("*", router.HandleRequest)` 和 `thttp.RegisterNoProtocolService(s.Service("trpc.app.server.fasthttp"))` 给 proto 服务和名字服务做映射。

```go
import (
    "fmt"

    "git.code.oa.com/trpc-go/trpc-go"
    routing "github.com/qiangxue/fasthttp-routing"
    "github.com/valyala/fasthttp"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    // Init server.
    s := trpc.NewServer()

    router := routing.New()
    router.Get("/v1/hello", func(ctx *routing.Context) error {
        ctx.Response.Header.SetContentType("application/text")
        ctx.Response.Header.Set("reply", "response head")
        ctx.SetStatusCode(fasthttp.StatusOK)
        ctx.WriteString("/v1/hello, " + string(ctx.Request.Header.Peek("hello")))
        return nil
    })

    router.Get("/v2/hello", func(ctx *routing.Context) error {
        ctx.Response.Header.SetContentType("application/text")
        ctx.Response.Header.Set("reply", "response head")
        ctx.SetStatusCode(fasthttp.StatusOK)
        ctx.WriteString("/v2/hello, " + string(ctx.Request.Header.Peek("hello")))
        return nil
    })

    router.Post("/v1/hello", func(ctx *routing.Context) error {
        ctx.Response.Header.SetContentType("application/text")
        ctx.Response.Header.Set("reply", "response head")
        ctx.SetStatusCode(fasthttp.StatusOK)
        ctx.WriteString("/v1/hello, " + string(ctx.Request.Header.Peek("hello")))
        ctx.WriteString("[POST]")
        return nil
    })

    router.Post("/v2/hello", func(ctx *routing.Context) error {
        ctx.Response.Header.SetContentType("application/text")
        ctx.Response.Header.Set("reply", "response head")
        ctx.SetStatusCode(fasthttp.StatusOK)
        ctx.WriteString("/v2/hello, " + string(ctx.Request.Header.Peek("hello")))
        ctx.WriteString("[POST]")
        return nil
    })

    thttp.FastHTTPHandleFunc("*", router.HandleRequest)
    thttp.FastHTTPHandleFunc("/123", func(ctx *fasthttp.RequestCtx) {
        ctx.WriteString("no routing")
    })
    thttp.RegisterNoProtocolService(s.Service("trpc.app.server.fasthttp"))

    // Start serving and listening.
    if err := s.Serve(); err != nil {
        fmt.Println(err)
    }
}
```

## 5.2 服务定义

对于泛 HTTP 标准服务，我们可以在 trpc_go.yaml 框架配置文件中通过 `protocol` 字段来指定具体协议类型。

注意，fasthttp_no_protocol 搭建在 [fasthttp](https://pkg.go.dev/github.com/valyala/fasthttp) 而非 [net/http](https://pkg.go.dev/net/http)，因此许多类型与 api 都发生了改变，请有需要的读者阅读 FastHTTP 迁移手册。

同时，FastHTTP 服务并不使用协议来区分是否提供 https，而是通过 TLS 配置来确定。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: fasthttp_no_protocol               # 应用层协议 trpc http 簇
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
```

**框架版本 >= v0.19.0 时**，支持在 `tls_key`, `tls_cert` 和 `ca_cert` 字段配置多个文件路径，两个文件路径之间用 **英文冒号`:`** 分隔，中间不要带空格。

**单向验证**：往往是客户端验证服务器，服务器不验证客户端。服务端只需要设置 `tls_key`、`tls_cert` 即可开启单向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: fasthttp_no_protocol               # 应用层协议 trpc http 簇
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

**双向认证**：服务端与客户端需要互相验证，在单向认证的基础上，增加 `ca_cert` 配置来验证客户端的合法性。一般银行等金融网站使用双向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: fasthttp_no_protocol               # 应用层协议 trpc http 簇
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
      ca_cert: ca.cert                             # ca 证书，用于校验 client 证书，以更严格识别客户端的身份，限制客户端的访问
      # ca_cert: ./caA.cert:./caB.cert             # 多个 ca 证书，框架版本 >= v0.19.0
```

## 5.3 代码示例

本节我们会通过提供基于 fasthttp 但功能与第四节相同的代码，帮助用户进行迁移和使用 fasthttp。

### 5.3.1 普通服务

本示例实现了一个 "Hello World" 的简单 HTTP 服务，在示例中我们展示了 HTTP 头部的读写，Cookie 的设置以及如何设置 HTTP 状态码。

可以通过以下命令进行验证：

``` shell
curl -X POST -d '{msg:"hello"}' -H "Content-Type:application/json" -H "request:test" "http://127.0.0.1:8000/v1/hello" -v
```

```go
package main

import (
    "encoding/json"

    "git.code.oa.com/trpc-go/trpc-go/log"
    "github.com/valyala/fasthttp"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

// Data 请求报文数据
type Data struct {
    Msg string
}

func handle(requestCtx *fasthttp.RequestCtx) {
    // 获取请求报文头里的 "request" 字段
    reqHead := string(requestCtx.Request.Header.Peek("request"))
    
    // 获取请求报文中的数据
    msg := requestCtx.Response.Body()
    log.Infof("data is %s, request head is %s\n", msg, reqHead)
    
    // 为响应报文设置 Cookie
    cookie := fasthttp.AcquireCookie()
    defer fasthttp.ReleaseCookie(cookie)
    cookie.SetKey("sample")
    cookie.SetValue("sample")
    cookie.SetHTTPOnly(false)
    requestCtx.Response.Header.SetCookie(cookie)
    
    // 无需在意顺序
    requestCtx.SetContentType("application/json")
    // 为响应报文头添加 reply 字段
    requestCtx.Response.Header.Add("reply", "tested")
    
    // 为响应报文设置 HTTP 状态码
    // requestCtx.SetStatusCode(403)
    
    // 为响应报文设置 Body
    rsp, _ := json.Marshal(&Data{Msg: "Hello, World!"})
    requestCtx.Write(rsp)
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.FastHTTPHandleFunc("/v1/hello", handle)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置文件 trpc_go.yaml 的配置为：

```yaml
global:                               # 全局配置
  namespace: Development              # 环境类型，分正式 production 和非正式 development 两种类型
  env_name: test                      # 环境名称，非正式环境下多环境的名称

server:                               # 服务端配置
  app: test                           # 业务的应用名
  server: stdhttp                     # 进程服务名
  service:                            # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp   # service 的路由名称
      ip: 127.0.0.1                   # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                      # 服务监听端口 可使用占位符 ${port}
      network: tcp                    # 网络监听类型 tcp udp
      protocol: fasthttp_no_protocol  # 应用层协议 trpc http
      timeout: 1000                   # 请求最长处理时间 单位 毫秒
```

### 5.3.2 协议代理

本节展示的示例是：服务作为一个 Proxy，接收标准 HTTP 服务请求，然后转化成 tRPC 协议格式向后端的 tRPC 服务发送请求。
可以通过以下命令进行验证：

``` shell
curl -X POST -d "hello" -H "Content-Type:application/text" "http://127.0.0.1:8000/v1/hello" -v
```

```go
package main

import (
    "context"
    
    "git.code.oa.com/trpc-go/trpc-go/client"
    pb "git.code.oa.com/trpcprotocol/test/helloworld"
    "github.com/valyala/fasthttp"
    
    trpc "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func handle(requestCtx *fasthttp.RequestCtx) {
    body := requestCtx.Response.Body()
    
    proxy := pb.NewGreeterClientProxy()
    req := &pb.HelloRequest{Msg: string(body[:])}
    
    // 向 tRPC 服务请求
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTarget("ip://127.0.0.1:8001"))
    
    if err != nil {
        requestCtx.SetContentType("text/plain; charset=utf-8")
        requestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
        requestCtx.SetStatusCode(fasthttp.StatusBadRequest)
        requestCtx.WriteString("call fails!")
        return
    }
    
    // 回响应给 HTTP 客户端
    requestCtx.SetContentType("application/text")
    requestCtx.WriteString(rsp.Msg)
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.FastHTTPHandleFunc("/v1/hello", handle)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置文件 trpc_go.yaml 配置为：

```yaml
global:                              # 全局配置
  namespace: Development             # 环境类型，分正式 production 和非正式 development 两种类型
  env_name: test                     # 环境名称，非正式环境下多环境的名称

server:                              # 服务端配置
  app: test                          # 业务的应用名
  server: hello                      # 进程服务名
  service:                           # 业务服务提供的 service，可以有多个
    - name: trpc.test.hello.stdhttp  # service 的路由名称
      ip: 127.0.0.1                  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                     # 服务监听端口 可使用占位符 ${port}
      network: tcp                   # 网络监听类型  tcp udp
      protocol: fasthttp_no_protocol # 应用层协议 trpc http
      timeout: 1000                  # 请求最长处理时间 单位 毫秒
```

### 5.3.3 文件下载服务

本示例实现了一个文件下载的简单 HTTP 服务，在示例中我们展示了指定文件的读取及文件的返回。

可以通过以下命令进行验证：

```shell
curl -X POST -d "filename=hello.txt" "http://127.0.0.1:8000/test/hello" -v
```

```go
func downloadHandler(requestCtx *fasthttp.RequestCtx) {
     fileName := requestCtx.PostArgs().PeekMulti("filename")
    fileNames := url.QueryEscape(string(fileName[0]))
    requestCtx.Response.Header.Add("Content-Type", "application/octet-stream;Charset=utf-8")
    requestCtx.Response.Header.Add("Content-Disposition", "attachment; filename=\""+fileNames+"\"")
    requestCtx.Response.Header.Add("Content-Transfer-Encoding", "binary")

    // 文件存放地址
    path := "/files/" + string(fileName[0])
    file, err := os.Open(path)
    if err != nil {
        fmt.Println("文件不存在", err)
        return
    }
    defer file.Close()

    // 为响应报文设置 Body
    content, err := io.ReadAll(file)
    if err != nil {
        fmt.Println("读取文件内容失败")
        return
    }
    requestCtx.Write(content)
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.FastHTTPHandleFunc("/test/hello", downloadHandler)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置同 5.3.1 章节。

### 5.3.4 配置服务端参数

通过修改使用的 transport 变量来实现，比如框架默认进行的注册为：

```go
// fasthttp
// Server transport (protocol file service).
transport.RegisterServerTransport(protocol.FastHTTP, DefaultFastHTTPServerTransport)
// Server transport (no protocol file service).
transport.RegisterServerTransport(protocol.FastHTTPNoProtocol, DefaultFastHTTPServerTransport)
// Client transport.
transport.RegisterClientTransport(protocol.FastHTTP, DefaultFastHTTPClientTransport)
```

用户可以通过修改 `DefaultFastHTTPServerTransport` 中的 `Server` 字段以提供额外的 HTTP 服务配置，比如：

```go
package main

import (
    "time"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/transport"
    "github.com/valyala/fasthttp"
)

func main() {
    st := thttp.DefaultFastHTTPServerTransport
    // 目前支持以下参数做配置：
    st.Server = &fasthttp.Server{
        ReadTimeout:  time.Second * 10,
        WriteTimeout: time.Second * 10,
        IdleTimeout:  time.Second * 10,
        // ...
    }
    // ...
}
```

用户也可以通过重新注册自定义的 transport 来达到类似的效果：

```go
st := thttp.NewFastHTTPServerTransport(transport.WithReusePort(true))
st.Server = &fasthttp.Server{ /* ... */ } // 自定义参数
transport.RegisterServerTransport("fasthttp", st)
```

### 5.3.5 SSE 服务

- 在版本 >= v0.19.0 (未发布时为 master 分支) 时，`thttp` 提供了一个 `WriteSSE` 的函数，用于将 `sse.Event` 结构体按照 SSE 格式快速写进 `io.Writer` 中。用户无需再关心 SSE 数据格式。
- 在版本 < v0.19.0 时，需要**手动拼接响应体**，然后再写入 `http.ResponseWriter` 中。

本示例实现了一个服务端的简单 HTTP SSE 服务，在示例中我们展示了使用 `WriteSSE` 函数封装消息，请确保 trpc-go 版本 >= v0.19.0。
你也可以参考 [SSE example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse) 获取更完整的用法示例

可以通过以下命令进行验证：

```shell
curl -X POST --data-raw "hello" "http://127.0.0.1:8000/v1/hello" -v
```

```go
package main

import (
    "bufio"
    "strconv"
    "time"

    "git.code.oa.com/trpc-go/trpc-go"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"

    "github.com/r3labs/sse/v2"
    "github.com/valyala/fasthttp"
)

func handle(requestCtx *fasthttp.RequestCtx) {
    requestCtx.Response.Header.SetContentType("text/event-stream")
    requestCtx.Response.Header.Set("Cache-Control", "no-cache")
    // fasthttp 默认设置长连接
    requestCtx.Response.Header.Set(thttp.Connection, "keep-alive")
    requestCtx.Response.Header.Set("Access-Control-Allow-Origin", "*")
        msg := string(requestCtx.Request.Body())
    requestCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
        for i := 0; i < 3; i++ {
            e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
            if err := thttp.WriteSSE(w, e); err != nil {
                requestCtx.SetContentType("text/plain; charset=utf-8")
                requestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
                requestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
                requestCtx.WriteString(err.Error())
                return
            }
            w.Flush()                          // 将写入的数据 flush 到客户端，使其可以立即读入到 SSE 事件，而不是等缓冲结束后再一次性发送
            time.Sleep(500 * time.Millisecond) // 模拟服务器延迟，在业务中不必要
        }
    })
}

func main() {
    s := trpc.NewServer()
    // 路由注册
    thttp.FastHTTPHandleFunc("/v1/hello", handle)
    // 服务注册
    thttp.RegisterNoProtocolService(s.Service("trpc.test.hello.stdhttp"))
    s.Serve()
}
```

框架配置同 5.3.1 章节。

# 6 FAQ

## 6.1 HTTP Server 相关问题

### Q1 - RESTful Server 是否支持 tke 健康检查接口？

目前不支持，不允许 http option 只有一个 `'/'`。

### Q2 - 泛 HTTP 服务如何在 Filter 获取 Request？

因为泛 HTTP 服务的实现和普通 RPC 服务有区别，http request 不在参数 req 中，需要调用 `http.Head` 从 ctx 里获取。

```go
import "git.code.oa.com/trpc-go/trpc-go/http"

func serverFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
    request := http.Head(ctx).Request
    log.Info("header:", request.Header)
    log.Info("method:", request.Method)

    rsp, err := next(ctx, req)
    return rsp, err
}
```

对于 fasthttp 而言，则使用 `http.RequestCtx(ctx).Request` 获取。

## 6.2 HTTP Client 相关问题

### Q1 - 如何使用域名调用 HTTP？

target 使用 dns selector，如

```go
WithTarget("dns://www.qq.com:80")
```

### Q2 - 如何避免复用 header？

需要使用自定义 header 时，禁止在 `http.NewClientProxy` 时设置，需要在每次调用时指定，避免复用 header。

### Q3 - 如何自己序列化二进制方式请求 HTTP，不用框架自动序列化？

```go
proxy := http.NewClientProxy("xxxx")
var (
    reqBody = &codec.Body{Data:[]byte("your request bytes data")} // 自己先序列化好请求的二进制数据
    rspBody = &codec.Body{}
)
err := proxy.Post(ctx, "url", reqBody, rspBody, client.WithCurrentSerializationType(codec.SerializationTypeNoop)) // 通过二进制方式请求 http，回包数据会自动填充到 rspBody.Data 里面
```

### Q4 - 如何调用 https 服务？

在 client 配置上 TLS 证书即可，配置项 `protocol` 仍然是 `http`。

```yaml
client:
  service:
    - name: trpc.xx.xx.xx  # 后端 http 的服务名，自己随便定义，跟代码 http.NewClientProxy("trpc.xx.xx.xx") 匹配即可，最好是点号分隔的四段字符串
      protocol: http
      tls_key: ./license.key
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
      ca_cert: ./ca.cert
      # ca_cert: ./caA.cert:./caB.cert             # 多个 ca 证书，框架版本 >= v0.19.0
```

详细见 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=482598119) 的 3.1 小节。

对于 fasthttp 而言，如果使用 InsecuritySkip 需要显式配置 ca_cert: none

### Q5 - 在进行跨语言调用时透传数据出错？

tRPC-Go v0.6.2 版本之前，服务端收到 HTTP 请求时，处理透传数据只使用 base64 解码，如果解码失败直接报错。tRPC-Cpp 发送的透传数据是不经过 base64 编码的，导致 tRPC-Cpp 调用 tRPC-Go 的时候如果透传数据，调用就会失败：

```go
func setTransInfo(trpcReq *trpc.RequestProtocol, msg codec.Msg, v string) error {
    m := make(map[string]string)
    if err := codec.Unmarshal(codec.SerializationTypeJSON, []byte(v), &m); err != nil {
        return err
    }
    trpcReq.TransInfo = make(map[string][]byte)
    // 由于 http header 只能传明文字符串，但是 trpc transinfo 是二进制流，所以需要经过 base64 保护一下
    for k, v := range m {
        decoded, err := base64.StdEncoding.DecodeString(v)
        if err != nil {
            return err
        }
        trpcReq.TransInfo[k] = decoded

        if k == TrpcEnv {
            msg.WithEnvTransfer(string(decoded))
        }
        if k == TrpcDyeingKey {
            msg.WithDyeingKey(string(decoded))
        }
    }
    msg.WithServerMetaData(trpcReq.GetTransInfo())
    return nil
}
```

解决方法：升级到 tRPC-Go v0.6.2 以上版本。

## 6.3 其他使用问题

### Q1 - 公司内部有没有 http 网关？

tRPC 没有专门做 http 或者 trpc 网关，公司内部已经有一个 IAS 网关可以使用，详见 [IAS](http://ias.woa.com/)。

### Q2 - http 服务定义的 pb 里面的 int64 字段在转成 json 时变成 string？

这个是谷歌 jsonpb 定义的标准做法，为了避免 int64 在前端溢出，因为 js 只有 50 多 bit 来存储数字，如果数字确实是 int64 类型，那就应该返回 string 给前端，由前端适配处理，如果数字不会超过 uint32 类型的最大值，那定义成 uint32 就好了。

trpc-go 的 json 序列化方式默认使用的是 pbjson，所以有上面这个特性。如果需要用自己的序列化方式，可以自己注册一个 json serialization type 到框架中：

```go
import (
    "git.woa.com/trpc-go/trpc-go/codec"
)

codec.RegisterSerializer(codec.SerializationTypeJSON, &codec.JSONSerialization{})
```

### Q3 - http 服务定义的 pb 里面的 json_name 字段转成 json 时不起作用？

服务启动的时候，修改掉 [serialization_jsonpb.go](https://git.woa.com/trpc-go/trpc-go/blob/master/codec/serialization_jsonpb.go) 文件中包名 `"git.woa.com/trpc-go/trpc-go/codec"` 对应的全局变量 `Marshaler.OrigName = false` 就可以了。

### Q4 - http 返回 err 时 body 为空，返回码放到 header 里面？

http 协议和 trpc 协议保持统一，返回失败时 body 为空，返回码都放到包头，也就是 http 协议的 header，或者 trpc 协议的 pb 包头。

用户也可以自己通过 ErrHandler 自定义错误码和错误信息字段，详细看这里的 [自定义错误码处理函数](https://git.woa.com/trpc-go/trpc-go/tree/master/http)。

### Q5 - curl 发送 http 请求时，返回失败：Connection reset by peer？

因为对端服务协议不是 http，确保同一个端口是否被其他服务占用。

### Q6 - trpc-go http 对 pb 默认值字段的序列化处理是怎么做的？

重点关注默认值字段，通常根据 pb 生成桩代码时会为每个 message field 生成 `omitempty` 这样的 tag，这个 tag 控制着字段为默认值的时候是否被进行序列化。

1. 首先要明确的是，框架不会给未赋值字段设置值。
2. 默认值是否参与序列化，encoding/json 会参考这里的 `omitempty`，trpc-go 用的是 protobuf/jsonpb，也会参考这里，但是为了大家使用方便，`jsonpb.Marshaler` 开启了 EmitDefaults 选项，即便是默认值也会传递。
3. 说到默认值，要考虑下 pb 中指定的是 syntax2 还是 syntax3，syntax2 是指针。

- 如果 jsonpb.Marshaler.EmitDefaults=true，序列化后字段值为 null，
- 如果 jsonpb.Marshaler.EmitDefaults=false，不对该字段进行序列化

### Q7 - http 已定义 `clienttrace.GotConn(connInfo)` 后，方法体内获取连接地址 panic？

代码如下：

```go
trace := &httptrace.ClientTrace {
    GotConn: func(connInfo httptrace.GotConnInfo) {
        msg.WithRemoteAddr(connInfo.Conn.RemoteAddr())
    }
}
```

通过 panic 记录发现 `msg.WithRemoteAddr(connInfo.Conn.RemoteAddr())` 这行会 panic，原因是因为 `connInfo.Conn` 为 nil。按照接口定义，`GotConn` 是在连接创建成功之后才会调用的，那这里的 `connInfo.Conn` 不应该为 nil，但是调试器跟踪发现该字段为 nil，如下：

这个错误的原因是因为 Go1.13 中引入了一个 [bug](https://github.com/golang/go/issues/34282)，请升级 Go 语言版本来解决。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
