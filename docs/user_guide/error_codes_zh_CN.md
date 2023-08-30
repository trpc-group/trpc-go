# 前言
tRPC 的多语言统一的错误由错误码`code`和错误描述`msg`组成，这与 go 语言常规的 error 只有一个字符串不是很匹配，所以 tRPC-Go 这边通过 [errs](https://git.woa.com/trpc-go/trpc-go/tree/master/errs) 包装了一层，方便用户使用，用户在接口失败时，返回错误码应该使用`errs.New(code, msg)`来返回，而不是直接返回标准库的`errors.New(msg)`，如：
```golang
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if failed { // 业务逻辑失败
		return nil, errs.New(your-int-code, "your business error message") // 失败 自己定义错误码，错误信息返回给上游
	}
	return &pb.HelloRepley{xxx}, nil // 成功返回 nil
}
```

# 错误码定义
tRPC-Go 的错误码分框架错误码`framework`，下游框架错误码`callee framework`，业务错误码`business`。
## 框架错误码
当前自身服务的框架自动返回的错误码，如调用下游服务超时，解包失败等，tRPC 使用的所有框架错误码都定义在 [trpc.proto](https://git.woa.com/trpc/trpc-protocol/blob/master/trpc/proto/trpc.proto) 中。
0~100 为服务端的错误，即当前服务在`收到请求包之后，进入处理函数之前`的失败，框架会自动返回给上游，业务是无感知的。在上游服务的视角来看就是下游框架错误码（见 2.2 小节）。
101~200 为客户端的错误，即当前服务调用下游返回的失败。
201~300 为流式错误。

一般日志表现如下：
```golang
type:framework, code:101, msg:xxx timeout
```
## 下游框架错误码
当前服务调用下游时，`下游服务（被调服务）的框架`返回的错误码，这对于下游服务的业务开发来说可能是无感知的，但很明确就是下游服务返回的错误，跟当前自身服务没有关系，当前服务是正常的，不过一般也是由于自己参数错误引起下游失败。
出现这个错误，请联系下游服务负责人。

一般日志表现如下：
```golang
type:callee framework, code:12, msg:rpcname:xxx invalid
```
## 业务错误码
当前服务调用下游时，下游服务的`业务逻辑`通过`errs.New`返回的错误码。注意：该错误类型是下游服务的业务逻辑返回的错误码，是开发自己任意定义的，具体含义需要找对应开发，跟框架无关。
tRPC-Go 推荐：业务错误时，使用 errs.New 来返回业务错误码，而不是在 body 里面自己定义错误码，这样框架就会自动上报业务错误的监控了，自己定义的话，那只能自己调用监控 sdk 自己上报。
建议用户自定义的错误码范围大于 10000，与框架错误码明显区分开。
出现这个错误，请联系下游服务负责人。

一般日志表现如下：
```golang
type:business, code:10000, msg:xxx fail
```
# 错误码含义

**注意：以下错误码说的是框架错误码和下游框架错误码。业务错误码是业务自己任意定义的，具体含义需要问具体开发。错误码只是大致错误类型，具体错误原因一定要仔细看错误详细信息。**

| 错误码 | 错误信息 |
| :----: | :----   |
| 0 | 成功 |
| 1 | 服务端解码错误，一般是上下游服务 pb 字段没有对齐或者没有同步更新，解包失败，上下游服务全部更新到 pb 最新版，保持 pb 同步即可解决 |
| 2 | 服务端编码错误，序列化响应包失败，一般是 pb 字段设置问题，如把不可见字符的二进制数据设置到 string 字段里面了，具体看 error 信息 |
| 11 | 服务端没有调用相应的 service 实现，tRPC-Go 没有该错误码，其他语言 tRPC 服务有 |
| 12 | 服务端没有调用相应的接口实现，调用函数填错，具体看 FAQ |
| 21 | 服务端业务逻辑处理时间过长超时，超过了链路超时时间或者消息超时时间，请联系下游被调服务负责人 |
| 22 | 请求在服务端过载，一般是下游服务端使用了限流插件，超过容量阈值了，请联系下游被调服务负责人 |
| 23 | 请求被服务端限流 |
| 24 | 服务端全链路超时，即上游调用方给的超时时间过短，还来不及进入本服务的业务逻辑 |
| 31 | 服务端系统错误，一般是 panic 引起的错误，大概率是被调服务空指针，数组越界等 bug，请联系下游被调服务负责人 |
| 41 | 鉴权不通过，比如 cors 跨域检查不通过，ptlogin 登陆态校验不通过，knocknock 没有权限，请联系下游被调服务负责人 |
| 51 | 请求参数自动校验不通过 |
| 101 | 请求在客户端调用超时，原因较多，具体看 FAQ |
| 102 | 客户端全链路超时，即当前发起 rpc 的超时时间过短，有可能是上游给的超时时间不够，也有可能是前面的 rpc 调用已经耗尽了大部分时间 |
| 111 | 客户端连接错误，一般是下游没有监听该 ipport，如下游启动失败 |
| 121 | 客户端编码错误，序列化请求包失败，类似上面的 2 |
| 122 | 客户端解码错误，一般是 pb 没有对齐，类似上面的 1 |
| 123 | 请求被客户端限流 |
| 124 | 客户端过载错误 |
| 131 | 客户端选 ip 路由错误，一般是服务名填错，或者该服务名下没有可用实例 |
| 141 | 客户端网络错误，原因较多，具体看 FAQ |
| 151 | 响应参数自动校验不通过 |
| 161 | 上游调用方提前取消请求 |
| 171 | 客户端读取帧数据错误 |
| 201 | 客户端流式队列满 |
| 351 | 客户端流式结束 |
| 999 | 未明确的错误，一般是下游直接用`errors.New(msg)`返回了不带数字的错误了，没有用框架自带的`errs.New(code, msg)` |
| 其他 | 以上列的是框架定义的框架错误码，不在该列表中的错误码说明都是业务错误码，是业务开发自己定义的错误码，需要找被调服务负责人 |

# 实现
错误码具体实现结构如下：
```golang
type Error struct { 
	Type int    // 错误码类型 1 框架错误码 2 业务错误码 3 下游框架错误码
	Code int32  // 错误码
	Msg  string // 错误信息描述
	Desc string // 错误额外描述，主要用于监控前缀，如 trpc 框架错误为 trpc 前缀，http 协议错误为 http 前缀，用户可以通过实现拦截器捕获该 err 并更改该字段实现上报任意前缀的监控
}
```
错误处理流程：
- 当用户通过`errs.New`明确返回业务错误或者框架失败时，此时会将该 err 通过不同的 type 分别填到 trpc 协议里面的框架错误`ret`或者业务错误`func_ret`字段里面。
- 框架打包返回时，会判断是否有错误，有错误则会抛弃 rsp body，所以`如果返回失败时，不要再试图通过 rsp 返回数据`。
- 上游调用方调用时，如果调用失败直接构造`框架错误 err`返回给用户，如果成功则解析出 trpc 协议里面的框架错误或者业务错误，构造`下游框架错误或者业务错误 err`返回给用户。

# FAQ
todo
