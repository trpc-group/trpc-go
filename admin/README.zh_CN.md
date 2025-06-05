- [管理命令概述](#管理命令概述)
- [管理命令列表](#管理命令列表)
  - [查看所有管理命令](#查看所有管理命令)
  - [查看框架版本信息](#查看框架版本信息)
  - [查看框架日志级别](#查看框架日志级别)
  - [设置框架日志级别](#设置框架日志级别)
  - [查看框架配置文件](#查看框架配置文件)
- [自定义管理命令](#自定义管理命令)
  - [定义函数](#定义函数)
  - [注册路由](#注册路由)
  - [触发命令](#触发命令)
- [pprof 性能分析](#pprof-性能分析)
  - [使用配置有 go 环境并且与服务连通的机器](#使用配置有-go-环境并且与服务连通的机器)
  - [官方火焰图代理服务](#官方火焰图代理服务)
  - [将 pprof 文件下载到本地，本地 go 工具进行分析](#将-pprof-文件下载到本地本地-go-工具进行分析)
  - [内存管理命令 debug/pprof/heap](#内存管理命令-debugpprofheap)
  - [PCG 123 发布平台查看火焰图](#pcg-123-发布平台查看火焰图)
  - [请求成本度量](#请求成本度量)
    - [普通 RPC 服务请求成本度量](#普通-rpc-服务请求成本度量)
    - [流式 RPC 服务请求成本度量](#流式-rpc-服务请求成本度量)
    - [ProfilerTagger 性能调优示例](#profilertagger-性能调优示例)

# 管理命令概述

管理命令（admin）是服务内部的管理后台，它是框架在普通服务端口之外额外提供的 http 服务，通过这个 http 接口可以给服务发送指令，如查看日志等级，动态设置日志等级等，具体命令可以看下面的命令列表。
admin 一般用于查询服务内部状态信息，用户也可以自己定义任意的命令。
admin 内部使用标准 restful 协议对外提供 http 服务。

框架默认不会开启 admin 能力，需要配置才会启动（生成配置时，可以默认配置好 admin，这样就能默认打开 admin 了）：

```yaml
server:
    app: app       # 业务的应用名，注意要改成你自己的业务应用名
    server: server # 进程服务名，注意要改成你自己的服务进程名
    admin:
        ip: 127.0.0.1 # admin 的 ip，配置网卡 nic 也可以
        port: 11014   # admin 的 port，必须同时配置这里的 ip port 才会启动 admin
        read_timeout: 3000 # ms. 请求被接受到请求信息被完全读取的超时时间设置，防止慢客户端
        write_timeout: 60000 # ms. 处理的超时时间，同时控制了获取 pprof/{profile,trace} 的最长时间，默认为 60s
```

# 管理命令列表

框架已经内置以下命令，注意：命令中的 ip:port 是上述 admin 配置的地址，不是 service 配置的地址。

## 查看所有管理命令

```bash
curl "http://ip:port/cmds"
```

返回结果

```json
{
    "cmds":[
        "/cmds",
        "/version",
        "/cmds/loglevel",
        "/cmds/config"
    ],
    "errorcode":0,
    "message":""
}
```

## 查看框架版本信息

```bash
curl "http://ip:port/version"
```

返回结果

```json
{
    "errorcode": 0,
    "message": "",
    "version": "v0.1.0-dev"
}
```

## 查看框架日志级别

```bash
curl -XGET "http://ip:port/cmds/loglevel?logger=xxx&output=0"
```

说明：logger 是为了支持多日志，不填即为框架的 default 日志，output 同一个 logger 下的不同输出，数组下标，不填即为 0, 第一个 output。

返回结果

```json
{
    "errorcode":0,
    "loglevel":"info",
    "message":""
}
```

**注：** 通过这个方法无法判断 `trace` 级别是否真正开启，因为 `trace` 级别的开启还依赖了环境变量或代码设置。

## 设置框架日志级别

(value 为日志级别，值为：trace debug info warn error fatal)

```bash
curl "http://ip:port/cmds/loglevel?logger=xxx" -XPUT -d value="debug"
```

说明：logger 是为了支持多日志，不填即为框架的 default 日志，output 同一个 logger 下的不同输出，数组下标，不填即为 0, 第一个 output。
注意：这里是设置的服务内部的内存数据，不会更新到配置文件中，重启即失效。

返回结果

```json
{
    "errorcode":0,
    "level":"debug",
    "message":"",
    "prelevel":"info"
}
```

**注：** `trace` 级别的开启除了要设置这里为 `trace` 或 `debug` 以外，还要设置环境变量 `export TRPC_LOG_TRACE=1` 或者添加代码 `log.EnableTrace()`。

## 查看框架配置文件

```bash
curl "http://ip:port/cmds/config"
```

返回结果

content 为 json 化的配置文件内容

```json
{
    "content":{ 
        
    },
    "errorcode":0,
    "message":""
}
```

# 自定义管理命令

## 定义函数

首先自己定义一个 http 接口形式的处理函数，你可以在任何文件位置自己定义：

```golang
// load 触发加载本地文件更新内存特定值
func load(w http.ResponseWriter, r *http.Request) {
    reader, err := ioutil.ReadFile("xxx.txt")
    if err != nil {
        w.Write([]byte(`{"errorcode":1000, "message":"read file fail"}`))  // 错误码，错误信息自己定义
        return
    }

    // 业务逻辑

    // 返回成功错误码
    w.Write([]byte(`{"errorcode":0, "message":"ok"}`))
}
```

## 注册路由

init 函数或者自己的内部函数注册 admin：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/admin"
)

func init() {
    admin.HandleFunc("/cmds/load", load)  // 路径自己定义，一般在/cmds 下面，注意不要重复，不然会相互覆盖
}
```

## 触发命令

触发执行自定义命令

```bash
curl "http://ip:port/cmds/load"
```

# pprof 性能分析

> v0.5.2~0.v0.18.3: trpc-go 会自动帮你去掉 golang http 包 DefaultServeMux 上注册的 pprof 路由，规避掉 golang net/http/pprof 包的安全问题（这是 go 本身的问题）。
**所以，使用 trpc-go 框架搭建的服务直接可以用 pprof 命令，但用```http.ListenAndServe("xxx", xxx)```方式起的服务会无法用 pprof 命令。**
如果一定要起原生 http 服务，并且要用 pprof 分析内存，可以通过```mux := http.NewServeMux()```的方式，不要用 http.DefaultServeMux。

> v0.19.0: admin 包支持的 pprof 功能依赖于导入的 net/http/pprof 包。然而，导入的 net/http/pprof 包在其 init 函数中隐式注册了 HTTP 处理程序，用于 "/debug/pprof/"、"/debug/pprof/cmdline"、"/debug/pprof/profile"、"/debug/pprof/symbol"、"/debug/pprof/trace"，并将它们注册在 `http.DefaultServeMux` 中。这种隐式行为过于微妙，如果使用 `http.DefaultServeMux`，可能会导致你无意中开放这些端口，从而导致安全问题：<https://github.com/golang/go/issues/22085>，因此，我们决定在 admin 的 init 函数中重置默认的 `http.DefaultServeMux` 以删除 pprof 注册。这需要确保在我们重置之前，你没有使用 `http.DefaultServeMux`。在大多数情况下，这是可行的，这由 init 函数的执行顺序保证。如果您需要在 `http.DefaultServeMux` 上启用 pprof，则需要在导入 admin 包后显式注册它，仅匿名导入 net/http/pprof 包是不起作用的。更多详情请参见：<https://git.woa.com/trpc-go/trpc-go/issues/912 和 https://github.com/golang/go/issues/42834>。

```go
http.DefaultServeMux.HandleFunc("/debug/pprof/", pprof.Index) 
http.DefaultServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline) 
http.DefaultServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
http.DefaultServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
http.DefaultServeMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
```

pprof 是 go 语言自带的性能分析工具，默认跟 admin 服务同一个端口号，只要开启了 admin 服务，即可以使用服务的 pprof。

idc 机器`配置好 admin 配置`以后，有以下三种方式使用 pprof：

## 使用配置有 go 环境并且与服务连通的机器

```bash
go tool pprof http://{$ip}:${port}/debug/pprof/profile?seconds=20
```

## 官方火焰图代理服务

tRPC-Go 官方已经搭建好火焰图代理服务，只需要在办公网浏览器输入以下地址即可查看自己服务的火焰图，其中 ip:port 参数为你的服务的 admin 地址。

```text
https://trpcgo.debug.woa.com/debug/proxy/profile?ip=${ip}&port=${port}
https://trpcgo.debug.woa.com/debug/proxy/heap?ip=${ip}&port=${port}
```

另外，还搭建了 golang 官方的 go tool pprof web 服务，（owner: terrydang）有 ui 界面：

```text
https://qqops.woa.com/pprof/
```

注意：如果接入了其他火焰图代理平台（如伽利略）后，使用此火焰图代理服务会出现 `500 Internal Server Error 错误`，请直接使用已接入的平台查看火焰图。

## 将 pprof 文件下载到本地，本地 go 工具进行分析

```bash
curl "http://${ip}:{$port}/debug/pprof/profile?seconds=20" > profile.out
go tool pprof profile.out

curl "http://${ip}:{$port}/debug/pprof/trace?seconds=20" > trace.out
go tool trace trace.out
```

**注：** 在默认配置下，获取 profile/trace 的时间最长为 60s，可以通过配置 admin 的 `write_timeout` 来调大时间。

## 内存管理命令 debug/pprof/heap

在安装了 go 命令的机器进行内存分析：

```bash
go tool pprof -inuse_space http://xxx:11029/debug/pprof/heap
go tool pprof -alloc_space http://xxx:11029/debug/pprof/heap
```

## PCG 123 发布平台查看火焰图

你可以在[插件市场](https://123.woa.com/v2/formal#/plugins-platform/index?_tab_=pluginsMarket)搜索[查看火焰图](https://123.woa.com/v2/formal#/plugins-platform/detail?pluginID=10025)插件。

## 请求成本度量

在构建和优化 RPC 服务时，了解服务的运行时开销是非常重要的。这可以帮助我们找出性能瓶颈，优化代码，提高服务的整体性能。为了实现这一目标，按照 RPC 服务类型划分，框架提供了 `WithProfilerTagger` 和 `WithStreamProfilerTagger` 来度量 RPC 服务的请求成本。

`ProfilerTagger` 提供了一种在 Goroutine 级别进行性能分析的方法。通过给 Goroutine 添加标签，我们可以在查看 pprof 图时，根据不同的标签过滤出更精细的信息。例如，当你发现服务的响应时间比预期的长，或者服务的 CPU 使用率过高时，就可以使用 `ProfilerTagger` 给 Goroutine 添加标签，更细粒度地了解到每个 RPC 请求的运行时开销，从而更好地优化服务性能。

### 普通 RPC 服务请求成本度量

使用 `server.WithProfilerTagger` 选项为普通 RPC 服务指定 ProfilerTagger。

下方示例代码为普通 RPC 服务指定 ProfilerTagger。

```go
type tagger struct{}

func (t *tagger) Tag(ctx context.Context, req interface{}) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    profileLabel.Store("serviceName", "trpc.test.helloworld.Greeter")
    if helloRsp, ok := req.(*pb.HelloRequest); ok {
        profileLabel.Store("msg", helloRsp.GetMsg())
    }
    return profileLabel, nil
}

s := trpc.NewServer(server.WithProfilerTagger(&tagger{}))
```

我们为 tagger 实现了 ProfilerTagger 接口，每次 RPC 调用时，服务端处理函数的 Goroutine 将携带两对标签，标签键值对如下表所示。

| 标签键        | 标签值                         |
| :------------ | ------------------------------ |
| `serviceName` | `trpc.test.helloworld.Greeter` |
| `msg`         | 服务端发送的消息               |

### 流式 RPC 服务请求成本度量

使用 `server.WithStreamProfilerTagger` 选项为流式 RPC 服务指定 StreamProfilerTagger。

下方示例代码为流式 RPC 服务指定 StreamProfilerTagger。

```go
type tagger struct {
}

// 对一次流式 RPC 服务调用打标签。
func (t *tagger) Tag(ctx context.Context, info *server.StreamServerInfo) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    profileLabel.Store("serviceName", "trpc.test.helloworld.Greeter")
    return profileLabel, nil
}

// 对一次 RecvMsg 打标签。
func (t *tagger) TagRecvMsg(ctx context.Context) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    profileLabel.Store("RecvMsg", "RecvMsgValue")
    return profileLabel, nil
}

// 对一次 SendMsg 打标签。
func (t *tagger) TagSendMsg(ctx context.Context, m interface{}) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    if rsp, ok := m.(*pb.HelloReply); ok {
        profileLabel.Store("SendMsg", rsp.GetMsg())
    }
    return profileLabel, nil
}

s := trpc.NewServer(server.WithStreamProfilerTagger(&tagger{}))
```

我们为 tagger 实现了 StreamProfilerTagger 接口，每次 RPC 调用时，服务端处理函数的 Goroutine 将携带三对标签，标签键值对如下表所示。

| 标签键        | 标签值                         |
| :------------ | ------------------------------ |
| `serviceName` | `trpc.test.helloworld.Greeter` |
| `RecvMsg`     | `RecvMsgValue`                 |
| `SendMsg`     | 服务端发送的消息               |

### ProfilerTagger 性能调优示例

假设有 RPC 服务方法 `Say`，实现逻辑如下。

```go
func (g *Greeter) Say(ctx context.Context, req *pb.SayRequest) (*pb.SayReply, error) {
    if req.GetMsg() == "hello" {
        // Redundant operation
        for i := 0; i < 1_000_000_000; i++ {
        }
    }
    // Normal operation
    for i := 0; i < 1_000_000_000; i++ {
    }
    return &pb.SayReply{}, nil
}
```

使用 `server.WithProfilerTagger` 选项为服务指定 `ProfilerTagger`。

```go
type tagger struct{}

func (t *tagger) Tag(ctx context.Context, req interface{}) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    if helloRsp, ok := req.(*pb.HelloRequest); ok {
        profileLabel.Store("msg", helloRsp.GetMsg())
    }
    return profileLabel, nil
}

s := trpc.NewServer(server.WithProfilerTagger(&tagger{}))
```

我们为 tagger 实现了 `ProfilerTagger` 接口，每次 RPC 调用时，服务端处理函数的 Goroutine 将携带标签，标签键为 `"msg"`，值为客户端请求消息。

服务端启动后，客户端一直调用服务端的 `Say` 方法，请求消息在 `"hello"` 和 `"hi"` 之间随机。

查看 pprof 信息，可以观察到 msg 有两种值 `"hello"` 和 `"hi"`，其中 `"hello"` 耗时较长。

![pprof before optimize](../.resources/admin/pprof-before-optimize.png)

分析服务端的 `Say` 方法，可以发现实现逻辑有冗余，优化后代码如下。

```go
func (g *Greeter) Say(ctx context.Context, req *pb.SayRequest) (*pb.SayReply, error) {
    // Normal operation
    for i := 0; i < 1_000_000_000; i++ {
    }
    return &pb.SayReply{}, nil
}
```

再次查看 pprof 信息，可以观察到 `"hello"` 和 `"hi"` 的耗时基本相同，性能得到优化。

![pprof after optimize](../.resources/admin/pprof-after-optimize.png)
