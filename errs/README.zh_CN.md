[English](./README.md) | 中文

# 1. 前言

tRPC 的多语言统一的错误由错误码 `code` 和错误描述 `msg` 组成，这与 go 语言常规的 error 只有一个字符串不是很匹配，所以 tRPC-Go 这边通过 [errs](https://git.woa.com/trpc-go/trpc-go/tree/master/errs) 包装了一层，方便用户使用。用户在接口失败时，返回错误码应该使用 `errs.New(code, msg)` 来返回，而不是直接返回标准库的 `errors.New(msg)`。如：

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    if failed { // 业务逻辑失败
        return nil, errs.New(your-int-code, "your business error message") // 失败 自己定义错误码，错误信息返回给上游
    }
    return &pb.HelloReply{xxx}, nil // 成功返回 nil
}
```

# 2. 错误码定义

tRPC-Go 的错误码分为框架错误码 `framework`、下游框架错误码 `callee framework` 和业务错误码 `business`。

## 2.1 框架错误码

当前自身服务的框架自动返回的错误码，如调用下游服务超时，解包失败等，tRPC 使用的所有框架错误码都定义在 [trpc.proto](https://git.woa.com/trpc/trpc-protocol/blob/master/trpc/proto/trpc.proto) 中。
`0~100` 为服务端的错误，即当前服务在 `收到请求包之后，进入处理函数之前` 的失败，框架会自动返回给上游，业务是无感知的。在上游服务的视角来看就是下游框架错误码（见 2.2 小节）。
`101~200` 为客户端的错误，即当前服务调用下游返回的失败。
`201~300` 为流式错误。

一般日志表现如下：

```go
type:framework, code:101, msg:xxx timeout
```

## 2.2 下游框架错误码

当前服务调用下游时，`下游服务（被调服务）的框架` 返回的错误码，这对于下游服务的业务开发来说可能是无感知的，但很明确就是下游服务返回的错误，跟当前自身服务没有关系，当前服务是正常的，不过一般也是由于自己参数错误引起下游失败。
出现这个错误，请联系下游服务负责人。

一般日志表现如下：

```go
type:callee framework, code:12, msg:rpcname:xxx invalid
```

## 2.3 业务错误码

当前服务调用下游时，下游服务的 `业务逻辑` 通过 `errs.New` 返回的错误码。注意：该错误类型是下游服务的业务逻辑返回的错误码，是开发自己任意定义的，具体含义需要找对应开发，跟框架无关。
tRPC-Go 推荐：业务错误时，使用 `errs.New` 来返回业务错误码，而不是在 body 里面自己定义错误码，这样框架就会自动上报业务错误的监控了，自己定义的话，那只能自己调用监控 sdk 自己上报。
建议用户自定义的错误码范围大于 10000，与框架错误码明显区分开。
出现这个错误，请联系下游服务负责人。

一般日志表现如下：

```go
type:business, code:10000, msg:xxx fail
```

# 3. 错误码含义

**注意：以下错误码说的是框架错误码和下游框架错误码。业务错误码是业务自己任意定义的，具体含义需要问具体开发。错误码只是大致错误类型，具体错误原因一定要仔细看错误详细信息。**

| 错误码 | 错误信息 |
| :----: | :----   |
| 0 | 成功 |
| 1 | 服务端解码错误，一般是上下游服务 pb 字段没有对齐或者没有同步更新，解包失败，上下游服务全部更新到 pb 最新版，保持 pb 同步即可解决 |
| 2 | 服务端编码错误，序列化响应包失败，一般是 pb 字段设置问题，如把不可见字符的二进制数据设置到 string 字段里面了，具体看 error 信息 |
| 11 | 服务端没有调用相应的 service 实现，tRPC-Go 没有该错误码，其他语言 tRPC 服务有 |
| 12 | 服务端没有调用相应的接口实现，调用函数填错，具体请看下边的 FAQ |
| 21 | 服务端业务逻辑处理时间过长超时，超过了链路超时时间或者消息超时时间，请联系下游被调服务负责人 |
| 22 | 请求在服务端过载，一般是下游服务端使用了限流插件，超过容量阈值了，请联系下游被调服务负责人 |
| 23 | 请求被服务端限流 |
| 24 | 服务端全链路超时，即上游调用方给的超时时间过短，还来不及进入本服务的业务逻辑 |
| 31 | 服务端系统错误，一般是 panic 引起的错误，大概率是被调服务空指针，数组越界等 bug，请联系下游被调服务负责人 |
| 41 | 鉴权不通过，比如 cors 跨域检查不通过，ptlogin 登陆态校验不通过，knocknock 没有权限，请联系下游被调服务负责人 |
| 51 | 请求参数自动校验不通过 |
| 101 | 请求在客户端调用超时，原因较多，具体请看下边的 FAQ |
| 102 | 客户端全链路超时，即当前发起 rpc 的超时时间过短，有可能是上游给的超时时间不够，也有可能是前面的 rpc 调用已经耗尽了大部分时间 |
| 111 | 客户端连接错误，一般是下游没有监听该 ipport，如下游启动失败 |
| 121 | 客户端编码错误，序列化请求包失败，类似上面的 2 |
| 122 | 客户端解码错误，一般是 pb 没有对齐，类似上面的 1 |
| 123 | 请求被客户端限流 |
| 124 | 客户端过载错误 |
| 131 | 客户端选 ip 路由错误，一般是服务名填错，或者该服务名下没有可用实例 |
| 141 | 客户端网络错误，原因较多，具体请看下边的 FAQ |
| 151 | 响应参数自动校验不通过 |
| 161 | 上游调用方提前取消请求 |
| 171 | 客户端读取帧数据错误 |
| 201 | 客户端流式队列满 |
| 351 | 客户端流式结束 |
| 999 | 未明确的错误，一般是下游直接用 `errors.New(msg)` 返回了不带数字的错误了，没有用框架自带的 `errs.New(code, msg)` |
| 其他 | 以上列的是框架定义的框架错误码，不在该列表中的错误码说明都是业务错误码，是业务开发自己定义的错误码，需要找被调服务负责人 |

# 4. 实现

错误码具体实现结构如下：

```go
type Error struct { 
    Type int    // 错误码类型 1 框架错误码 2 业务错误码 3 下游框架错误码
    Code int32  // 错误码
    Msg  string // 错误信息描述
    Desc string // 错误额外描述，主要用于监控前缀，如 trpc 框架错误为 trpc 前缀，http 协议错误为 http 前缀，用户可以通过实现拦截器捕获该 err 并更改该字段实现上报任意前缀的监控
}
```

错误处理流程：

- 当用户通过 `errs.New` 明确返回业务错误或者框架失败时，此时会将该 err 通过不同的 type 分别填到 trpc 协议里面的框架错误 `ret` 或者业务错误 `func_ret` 字段里面。
- 框架打包返回时，会判断是否有错误，有错误则会抛弃 rsp body，所以 `如果返回失败时，不要再试图通过 rsp 返回数据`。
- 上游调用方调用时，如果调用失败直接构造 `框架错误 err` 返回给用户，如果成功则解析出 trpc 协议里面的框架错误或者业务错误，构造 `下游框架错误或者业务错误 err` 返回给用户。

# 5. FAQ

## 5.1 rpc 请求返回错误

### 12 rpc name:xxx invalid

- 首先要了解：rpcname 是 proto 协议文件里面的方法名，格式是 `/package.service/method`，跟配置无关，不是配置文件里面的 servicename。
- 检查被调服务协议文件生成代码的方法名 `/package.service/method` 与主调 client 设置的 `方法名` 是否一致。
- 检查 pb 是否引用出错，被调方服务 ip 地址是否填错，是否调用到其他人的服务了。
- 因为 trpc-go 默认支持 reuseport，所以本地开发时要确认同一个 ipport 是否启动了多个不同的服务，如果启动多个不同服务，则会出现时而正常，时而失败。
- `NewServer` 后面确保注册了正确的 pb 实现，如：`pb.RegisterService(s, &GreeterServerImpl{})`。
- 检查被调方 pb 生成工具生成的 pb.go 文件中 `serviceDesc` 的描述 `serviceName` 和 `Func` 是否正确。
- 检查是否间接使用了 "git.woa.com/polaris/polaris-go" v0.4.1 版本，该版本存在异常，需升级到 v0.4.2 以上版本。

### 31 runtime error: index out of range (or nil pointer)

下游服务数组越界或者空指针导致 server panic 了，是下游服务的问题，不是你的问题。

### 161 context canceled

context 取消，有两种情况：

- client 连接断开导致的 context 提前取消，一般是由于时间不够用，上游 client 超时主动断开连接，当前服务检测到这个事件，把当前正在执行的网络请求 cancel 了，避免做无用功，这种情况是属于正常的。该错误常见于 http server，外网 web 异常较多，用户手动刷新页面马上退出，客户端 crash，或者客户端 webview bug 都会触发这个错误。
- rpc 函数退出导致的 context 提前取消，service 入口的 rpc 函数在 return 返回后，框架会自动 cancel context，所以不可以在异步协程中继续使用请求入口传入的 ctx，因为此时的 ctx 已经销毁了，异步调用不要使用请求入口的 ctx，可以使用框架提供的异步启动 api：[`trpc.Go(ctx, timeout, handler)`](https://git.woa.com/trpc-go/trpc-go/blob/master/trpc_util.go#L152)。

### 141 EOF

"End of file" 错误，对端关闭链接，可能是对端服务 panic，也可能是对端服务异常关闭，需要让被调服务查看相关原因。

- 1 如果被调方不是 trpc 服务，则大概率是由于连接空闲时间引起的，trpc-go client 的连接空闲时间默认是 50s，当被调方服务空闲时间小于 50s，server 端主动关闭连接了，trpc-go client 这边会拿出一个已经关闭的连接进行复用导致出错，解决办法有以下三种（选择其中一种即可）：
  - 1.1 被调方服务把连接空闲时间调大，大于 50s（trpc go server 默认空闲时间是 1min，大于 50s，不会出问题，除非用户自己胡乱设置了 server idletime）。
  - 1.2 trpc-go 主调这边把连接空闲时间调小，小于被调方的空闲时间：`connpool.WithIdleTimeout(time.Second)`。

    ```go
    // example main 函数初始化时调用
    connpool.DefaultConnectionPool = connpool.NewConnectionPool(connpool.WithIdleTimeout(time.Second))
    ```

  - 1.3 如果 server 支持 client 连接多路复用（即一个连接里面多发多收，trpc-go server v0.5.0 以上默认支持多路复用，其实就是服务端异步 server_async，一般要求是复用了 trpc 协议的 server transport 逻辑的），则可在 client 调用方这边开启连接多路复用 option：`client.WithMultiplexed(true)`。
- 2 如果是被调方发布重启导致的，说明发布流程有问题，发布服务时，正确流程应该是先从名字服务上剔除待发布的 ipport，并且等待一段时间（不同名字服务，缓存时间不一样，北极星约 30s 即可）后，开始删除老容器，重建新容器，新容器启动成功后再把新 ipport 加入到名字服务中。
- 3 如果被调服务处理时间太长超过了 server 的 idletime（默认 1min），则 server 会主动关闭连接，此时 client 就会拿到 EOF 的连接，解决方案可以按上面的 1.1 调大 server 的 idletime，大于处理时间，或者按上面的 1.3 开启 client 连接多路复用，前提是 server 支持连接多路复用。
- 4 服务端框架版本在 < v0.9.5 时，对于超过 10MB 的 trpc 协议包会直接关掉客户端的连接，没有返回包长过长这一错误给客户端，导致客户端只能看到一个 141 EOF 的错误，在 >= v0.9.5 之后的服务端对该出错信息有所 [优化](https://git.woa.com/trpc-go/trpc-go/-/merge_requests/1467)，对于这一错误，客户端和服务端都需要手动在代码里设置 `trpc.DefaultMaxFrameSize = xxx` 进行调大（服务端和客户端都要设置）。

### 141 tcp client transport ReadFrame: trpc framer: read framer head magic not match

出现这个可能是网络原因导致的，telnet 一下 ip 和 port 看一下网络是否通，不通的话开一下策略。也有可能是测试 json 文件里的 Protocol 没有正确配置。
也有可能是上下游服务协议没有对齐，比如往 http 服务发 trpc 请求。

### 141 connection close

对端业务层直接关闭连接。一般是上下游协议没对齐，如往 http server 发送 trpc protocol 请求。

### 111/141 connection refused

不同的协议，错误码可能不一样，但是错误信息是一致的，对端没有在你请求的 ip:port 上提供服务，则会出现 connection refused 错误，表明连接直接被对端拒绝，一般是下游服务挂了或者 ipport 不对，没有监听这个 ipport，请确保被调服务是否启动正常。这个错误很明确，100% 就是下游服务没有监听这个 ipport，不要再说服务正常，怀疑此文档，几乎可以确定是下游服务重启了。

### 101 write timeout

写数据超时，一般是调用当前 rpc 之前，上一个 rpc 已经把时间耗光了，这个 rpc 其实根本没时间发送出去，请查看服务的超时配置，超时控制逻辑请看 [文档](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688)。

### 101 read timeout

读数据超时，一般是下游服务没有在规定时间内返回，请查看服务的超时配置，超时控制逻辑请看 [文档](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688)。

### 101 dial timeout

建立连接超时，一般是网络不通，或者类似 write timeout，也有可能是下游服务过载，监听队列爆满导致，请查看服务的超时配置，超时控制逻辑请看 [文档](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688)。

### 101/141 context deadline exceeded

不同的插件可能错误码不一样，不过 deadline exceed 就是代表时间不够用了，与 101 write timeout 类似。

### 131 client Select

client 寻址错误，看 [这里](https://iwiki.woa.com/p/4008319150#6faq)。

### 122 client codec Decode: rsp request id xxx different from req request id

回包 id 与请求 id 不一致，这个回包不是当前这次请求的回包。
trpc go client 默认使用的是独占连接池模式，发包后会挂住等待回包，然后再把连接放回连接池里面等下次再拿出来复用。
正常情况，回包都是一致的，出现这种情况一般是被调方代码有 bug，同一个请求回包了多次，因为 client 这边只会取一次，所以导致下次复用时取到的是上次的回包。

以下两种解决方案（二选一即可）：

1. 被调方排查一下 bug，看是否多次调用了 `WriteResponse` 类似接口，多次回包了。tRPC-Go server 只能通过函数 return，框架自动回包，不会出现这个问题。其他语言如 trpc-cpp、trpc-node 提供了用户自己回包的接口，所以很有可能会出现这个 bug。
2. 主调方改成 IO 复用模式，看这里：[tRPC-Go 客户端连接模式](https://iwiki.woa.com/p/435513714)，加个 client option：`client.WithMultiplexed(true)`。为什么不直接默认 IO 复用呢，因为初期考虑通用性，为了支持所有协议，很多私有协议跟 HTTP 一样，都没有 request id，没办法用 IO 复用。

建议采用上面第一种，因为这是代码 bug 引起的，第二种也可以解决，只是永远把问题隐藏了。

### -1 xxx

错误码不在第 3 节的列表中，说明是业务自己定义的，需要找对应负责人。

## 5.2 所有 socket 网络请求错误概念

### EOF

"End of file" 错误，对端关闭链接，可能是对端服务 panic，也可能是对端服务异常关闭，需要让被调服务查看相关原因。

### reset by peer

对端发送了 reset 信号，表明链接被丢弃。当对端服务异常，或者负载过高的时候可能出现，需要让被调服务查看相关原因。
也有可能是上下游服务协议没有对齐，比如往 http 服务发 trpc 请求。

### broken pipe

对端已经关闭连接，主调方没意识到继续操作 socket 的时候会出现 broken pipe 错误。当对端 crash 的时候可能出现这个错误。
也有可能是包太大，超过 10M 大小限制，先考虑下大包合理性问题，再考虑自己设置包大小限制：`trpc.DefaultMaxFrameSize=1111`。

### connection refused

对端没有在你请求的 ip:port 上提供服务，则会出现 connection refused 错误，表明连接直接被对端拒绝，请确保被调服务是否正常。

## 5.3 超时问题：type: framework, code: 101, msg: xxx timeout

### 我明明设置了很大的超时时间，为什么实际上耗时很短就提示超时失败了？

框架对每次收到的请求都有一个最长处理时间的限制，每次 rpc 后端调用的超时时间都是根据当前剩余最长处理时间和调用超时实时计算的，这种情况大概率是因为多个串行 rpc 调用时，上一个 rpc 已经把时间耗的差不多了，所以留给这次 rpc 的时间不够用了。
所以在多个 rpc 调用时，应该自己合理分配多个 rpc 的超时时间，如果每个 rpc 耗时确实很长，则自己调大消息超时，或者禁用继承链路超时。

### 为什么通过 go 自己启动协程调用网络请求，每次都提示 context cancel 错误？

context 是请求上下文的意思，在当前请求函数退出时，会马上取消 context，所以自己用 go 启动的协程不能继续使用请求入口携带的 ctx，需要自己使用新的 context，如 `trpc.BackgroundContext()`。

### 为什么我用 trpc-cli 工具发包时老是超时？

trpc-cli 工具发包时，默认设置的超时时间是 1s，由于你的服务耗时比较久导致工具调用失败，可以先确定下 ipport 是否正确，再定位一下为什么服务内部耗时这么久，或者调高 trpc-cli 的超时时间：`trpc-cli -timeout 5000 -func ...`。

### 101 timeout 错误，如何定位？

1. 首先先阅读并理解 [超时控制](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688) 的概念，了解链路超时，消息超时的定义。
2. 确定下游地址是否正确，包括 环境 namespace，服务名 servicename，直连时的 ipport。
3. 确定下游服务是否收到请求，是否处理时间过长，确定网络是否正常。
4. 超时问题可以使用 [trpc-filter/debuglog](https://git.woa.com/trpc-go/trpc-filter/tree/master/debuglog) 插件来方便定位。
5. 通过 debuglog 日志，可以看到每一个 rpc 的具体耗时时间，大致就能看出问题在哪里了，确定一下时间主要耗在哪里。
6. 可以通过 [tjg 调用链](https://git.woa.com/trpc-go/trpc-opentracing-tjg)，来排查上下游的执行问题。
7. 还定位不到，下游服务就 [打开 trace 日志](https://git.woa.com/trpc-go/trpc-go/tree/master/log)，估计是上下游协议没对齐，下游直接丢包了。
8. 确定各插件版本是否是最新版，上下游老版本的名字服务寻址都有 bug，不管是 go 还是 cpp，都要升级更新一下。
9. 确定网络环境是否正常，换台机器（或容器）看看。
