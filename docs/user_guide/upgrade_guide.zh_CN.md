## 背景介绍

有很多用户在使用较低版本的 tRPC-Go，用户在升级到新版本时，可能会遇到编译错误或者运行错误。本文收集了过去用户在升级新版本中反馈较多的问题，为用户升级 tRPC-Go 提供指引。

## 一步到位升级建议

首先建议用户直接升级到 tRPC-Go 的 LTS 版本：v0.18.x，在 tRPC-Go 的版本迭代中，中间版本引入了一些 Bug 和兼容性问题，尤其是 v0.9.0 以下版本，但是最终都已经被修复。为了避免踩到这些历史 Bug，建议用户直接升级至 tRPC-Go LTS 版本，目前是 v0.18.x。

执行以下指令可以将 tRPC-Go 框架更新到 LTS 版本 v0.18.x。

```golang
go get git.code.oa.com/trpc-go/trpc-go@v0.18
```

更新完框架版本后，按照以下 4 个指引检查代码，对代码做相应修改，实现“无痛升级”。例如在指引 1 中，你需要特别注意更新拦截器和 codec 插件版本。

### 指引 1: 修改 server 拦截器和桩代码签名【重要】

_如果你是从 v0.9.0 以下升级，需要关注。_

v0.9.0 变更了服务端拦截器签名，v0.9.0 之前的服务端拦截器 rsp 是在入参中，而 v0.9.0 之后的服务端拦截器 rsp 移动到了出参。为了适配 v0.9.0 之后的拦截器签名，v0.9.0 之后桩代码中的处理函数签名也相应做了变更。

#### 修改 server 拦截器签名

```go
// v0.9.0 前的拦截器格式：
func ServerFilter(ctx, req, rsp, next) error { 
    // 前置逻辑，这里的 rsp 是 nil
    err := next(ctx, req, rsp)
    // 后置逻辑，这里不能操作 rsp，会触发空指针 panic，或者断言失败
}

// v0.9.0 后的拦截器格式：
func ServerFilter(ctx, req, next) (rsp, error) { 
    // 前置逻辑
    rsp, err := next(ctx, req)
    // 后置逻辑，这里可以随意更改 rsp，甚至返回一个新的 rsp 结构体
}
```

如果你升级 tRPC-Go 框架到最新版本，并使用 v0.9.0 之前的服务端拦截器签名，启动服务时会出现如下提示

```raw
filter: xxx is too old, please change to new ServerFilter, any question please refer to ChangeLog v0.9.0
```

此提示不会影响服务的正常启动，但是由于 v0.9.0 之前的拦截器入参 rsp 变成了空指针，可能触发运行时的 bug。所以需要将所有拦截器签名更新到 v0.9.0 之后的格式。如果是使用了官方提供的拦截器插件 [trpc-filter](https://git.woa.com/trpc-go/trpc-filter) 或 codec 插件（<https://git.woa.com/trpc-go/trpc-codec)>，直接将插件版本更新到其最新版本即可，如果无法升级到最新版本，则至少得升级到不低于如下表格中的版本才行。）

|        filter         | version |
|:---------------------:|:--------|
|          bkn          | v0.1.3  |
|         cors          | v0.1.4  |
|       debuglog        | v0.1.4  |
|        degrade        | v0.1.1  |
|   filterextensions    | v0.1.1  |
|        forward        | v0.1.2  |
|        hystrix        | v0.1.0  |
|          ioa          | v0.2.0  |
|          jwt          | v0.2.0  |
| knocknock-auth-client | v0.1.6  |
| knocknock-auth-server | v0.1.3  |
|        limiter        | v0.1.4  |
|          mm           | v0.0.5  |
|         mock          | v0.1.0  |
|        ptlogin        | v0.1.0  |
|       qqconnect       | v0.1.0  |
|       recovery        | v0.1.3  |
|        referer        | v0.1.0  |
|         slime         | v0.2.8  |
|   transinfo-blocker   | v0.1.0  |
|         tvar          | v0.1.0  |
|      validation       | v0.1.2  |
|        wxlogin        | v0.1.0  |

---

| codec | version |
|:-----:|:--------|
|  cmd  | v0.2.0  |

#### 重新生成桩代码

由于桩代码会调用服务端拦截器，随着服务端拦截器的签名变更，桩代码也需要同步更新。使用最新版本的 [trpc cmdline](https://git.woa.com/trpc-go/trpc-go-cmdline) 工具重新生成桩代码时，会生成不同签名的桩代码。

```golang
// 旧版本的桩代码 Service interface 定义，rsp 是在入参中
type GreeterService interface {
    SayHello(ctx context.Context, req *HelloRequest, rsp *HelloReply) error

    SayHi(ctx context.Context, req *HelloRequest, rsp *HelloReply) error
}

// 新版本的桩代码 Service interface 定义，rsp 是在出参中
type GreeterService interface {
    SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error)

    SayHi(ctx context.Context, req *HelloRequest) (*HelloReply, error)
}
```

### 指引 2: plugin setup 阶段发起调用的逻辑需要放到 plugin onFinish 中

_如果你是从 v0.8.2 以下升级，需要关注。_

如果你的 plugin 在 Setup 阶段发起了 RPC 调用，在新版本可能会调用失败。你需要为 plugin 添加 OnFinish 方法，将 NewClientProxy 和发起调用的逻辑移入 OnFinish 方法中，OnFinish 会在所有插件 setup 结束并且 client option 配置完成后执行。

```golang
func (p *myPlugin) OnFinish(name string) error {
    // do request...
}
```

### 指引 3: 客户端拦截器中  SetMetaData 改为 WithClientMetaData

_如果你是从 v0.8.1 以下升级，需要关注。_

```golang
// 旧版本的写法
func clientFilter(ctx context.Context, req interface{}, rsp interface{}, next filter.ClientHandleFunc) error {
    trpc.SetMetaData(ctx, key, []byte(val))
    // 业务逻辑...
}
```

检查你当前的客户端拦截器，看下有没有 trpc.SetMetaData 的逻辑，如果有则需要改为 msg.WithClientMetaData。

```golang
// 需要改成这种写法
func clientFilter(ctx context.Context, req interface{}, rsp interface{}, next filter.ClientHandleFunc) error {
    msg := trpc.Message(ctx)
    clientMetaData := msg.ClientMetaData()
    if clientMetaData == nil {
        msg.WithClientMetaData(map[string][]byte{})
        clientMetaData = msg.ClientMetaData()
    }
    clientMetaData[key] = []byte(val)
    // 业务逻辑...
}
```

### 指引 4: tars 服务需要更新 jce module 名

_如果你是从 v0.8.0 以下升级，需要关注。_

如果你的服务是 trpc-tars 服务，需要使用最新版的 [trpc4tars](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars/tools/trpc4tars) 桩代码生成工具，重新生成 jce 桩代码。并将上游所有的 jce 依赖从 `git.code.oa.com/jce/jce` 改为 `git.woa.com/jce/jce`。

## 升级到特定版本

如果你不想升级 tRPC-Go 框架到最新版本，在这里可以查询你想升级的目标版本常出现的问题。

### v0.8.0

#### tars 服务需要更新 jce module 名

**错误现象**

如果是 tRPC-Go 搭建的 tars 服务，升级至 v0.8.0 以上会出现以下错误的其中之一。

编译错误

```log
have XXX(*"git.code.oa.com/jce/jce".Buffer) error
want XXX(*"git.woa.com/jce/jce".Buffer) error
```

运行错误

```log
type:framework, code:121, msg:client codec Marshal: not jce.Message
```

运行错误

```log
failed to unmarshal body: expected git.woa.com/jce/jce.Message, got git.code.oa.com/jce/jce.Message. You may need to refer to issue https://git.woa.com/trpc-go/trpc-go/issues/897"
```

**解决方案**

使用最新版的 [trpc4tars](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars/tools/trpc4tars) 桩代码生成工具，重新生成 jce 桩代码。并将上游所有的 jce 依赖从 `git.code.oa.com/jce/jce` 改为 `git.woa.com/jce/jce` 。

**参考资料**

引入 MR: [jce/jce&trpc-go/go_reuseport 切换为 woa 域名](https://git.woa.com/trpc-go/trpc-go/merge_requests/1253)

反馈 Issue: [能否同时兼容两种域名的 jce，目前 trpc-go 版本升级导致 jce 序列化异常](https://git.woa.com/trpc-go/trpc-go/issues/897)

码客：[解决 jce 库修改域名后不兼容的问题](https://mk.woa.com/note/7422)

### v0.8.1

#### 需要将客户端拦截器中  SetMetaData 改为 WithClientMetaData

**错误现象**

在升级到 v0.8.1 以上的版本时，在客户端拦截器调用 trpc.SetMetaData 设置的元数据会失效。服务端不会收到通过 trpc.SetMetaData 设置进去的元数据。

**解决方案**

```golang
// 旧版本的写法
func clientFilter(ctx context.Context, req interface{}, rsp interface{}, next filter.ClientHandleFunc) error {
    trpc.SetMetaData(ctx, key, []byte(val))
    // 业务逻辑...
}
```

检查你当前的客户端拦截器，将客户端拦截器中的 trpc.SetMetaData 改为 msg.WithClientMetaData。

```go
// 需要改成这种写法
func clientFilter(ctx context.Context, req interface{}, rsp interface{}, next filter.ClientHandleFunc) error {
    msg := trpc.Message(ctx)
    clientMetaData := msg.ClientMetaData()
    if clientMetaData == nil {
     msg.WithClientMetaData(map[string][]byte{})
     clientMetaData = msg.ClientMetaData()
    }
    clientMetaData[key] = []byte(val)
    // 业务逻辑...
}
```

**错误原因**

客户端拦截器内部通过 trpc.SetMetaData 设置的参数，设置到的是 msg.ServerMetaData，在后续发包过程中 tRPC-Go 框架不会把 ServerMetaData 发送给下游，只会把 ClientMetaData 里的数据发送给下游。这个是合理的逻辑，只不过 v0.8.2 之前的版本会把 ServerMetaData 里的数据也发给下游，这算是框架补上了之前的一个漏洞。

**参考资料**

反馈码客：[关于 trpc-go 版本从 v0.7.2 升级到 v0.9.4 client filter 请求透传问题？](http://mk.woa.com/q/285304)

#### 如果出现 unary request pb header empty 错误，请升级到 v0.9.0 以上

**错误现象**

如果升级到 v0.8.1~0.8.6 版本，客户端发送包头数据都是“零”值的请求，会触发 encode fail:unary request pb header empty 错误，直接使用 tRPC 框架发包一般不会发送“零”值的包头，通常发生在测试、非 trpc 框架（spp，自己构造 trpc 请求）的场景。

**解决方案**

升级框架至 v0.9.0 以上或者构造 trpc 请求的时候将 requestID 赋值，保证 requestID 大于零值，可以避免此错误。

**参考资料**

反馈 Issue: [[Bug Fixes] frameHeadLen 检查的必要性和造成兼容性问题的处理](https://git.woa.com/trpc-go/trpc-go/issues/685)

反馈码客：[trpc-go v0.5.2   升级到 v0.8.1，trpc 服务响应回包报错 encode fail:unary request pb header empty 该如何解决？](http://mk.woa.com/q/280432)

### v0.8.2

#### 不要使用 v0.8.2 版本

**错误现象**

用户升级到 v0.8.2 会发现 client 包的如下导出函数找不到

```go
- Options.LoadClientConfig
- Options.SetNamingOptions
- Options.LoadClientFilterConfig
```

**解决方案**

升级 tRPC-Go 框架到 v0.8.3 以上

**错误原因**

tRPC-Go 框架在 v0.8.2 重构 client option 时删除了以上导出函数，并在 v0.8.3 重新引入。

**参考资料**

引入 MR: [重构 client 模块，抽取 selector filter](https://git.woa.com/trpc-go/trpc-go/merge_requests/1299)

修复 MR: [client options 兼容历史逻辑](https://git.woa.com/trpc-go/trpc-go/merge_requests/1322)

#### 升级 trpc-config-rainbow 插件至 v0.1.22 以上

**错误现象**

用户升级到 v0.8.2 以上版本，可能会发现 rainbow 的客户端远程配置失效

**解决方案**

升级 tRPC-Go 框架至 v0.8.5 以上
升级 trpc-config-rainbow 插件至 v0.1.22 以上

**错误原因**

tRPC-Go 框架在 v0.8.2 重构了 client option，不再每次请求的时候再读取 config，而是提前构造好 config map 和 option map，在发起请求的时候直接读取 map。这就导致提前构造 config map 和 option map 的时候覆盖了 trpc-config-rainbow 插件写入的 config map 信息。v0.1.22 版本的 trpc-config-rainbow 插件会保证框架不再修改 config map 后再写入 config map，保证配置信息不会被覆盖

**参考资料**

反馈 Issue: [v0.8.3 兼容问题。client 无法初始化](https://git.woa.com/trpc-go/trpc-go/issues/672)

引入 MR: [重构 client 模块，抽取 selector filter](https://git.woa.com/trpc-go/trpc-go/merge_requests/1299)

修复 MR: [解决 client config 覆盖问题](https://git.woa.com/trpc-go/trpc-go/merge_requests/1337)

修复 MR: [fix: 增加远程 client 配置首次应用等待时间，适配新版本](https://git.woa.com/trpc-go/trpc-config-rainbow/merge_requests/80)

#### 将 plugin setup 阶段发起调用的逻辑放到 plugin onFinish 中

**错误现象**

用户升级到 v0.8.2 以上版本，可能会发现在 plugin setup 阶段调用 NewClientProxy 失败或者发起请求失败。

**解决方案**

升级 tRPC-Go 至 v0.8.4 以上
为自定义的 plugin 添加 OnFinish 方法，将 NewClientProxy 操作或者发起调用的逻辑移入 OnFinish 方法中，OnFinish 会在所有插件 setup 结束并且 client option 配置完成后才会执行。

```golang
func (p *myPlugin) OnFinish(name string) error {
    // do request...
}
```

**错误原因**

tRPC-Go 框架在 v0.8.2 重构了 client option，在所有插件 setup 结束后，才会解析配置文件中的 config，导致原来的插件在 setup 调用下游出错（因为没解析出框架配置）。OnFinish 会可以保证在所有插件 setup 结束并且 client option 配置完成后才会执行。

**参考资料**

反馈码客：[trpc-go 升级到 v0.9.4 之后，filter setup 阶段 NewClientProxy 报错](https://mk.woa.com/q/283235?ADTAG=search)

引入 MR: [重构 client 模块，抽取 selector filter](https://git.woa.com/trpc-go/trpc-go/merge_requests/1299)

修复 MR: [提供插件初始化完成回调通知](https://git.woa.com/trpc-go/trpc-go/merge_requests/1330)

#### 如果出现 target 设置了不生效，请更新到 v0.10.0

**错误现象**

用户升级到  v0.8.2 ～ v0.9.4 之间，可以会发现自己配置的 target 不生效了，使用了 service name 发起请求。

**解决方案**

更新到 v0.10.0 即可解决。

**错误原因**

tRPC-Go 框架在 v0.8.2 重构了 client 模块，导致 target、serviceName 生效顺序不符合预期。

**参考资料**

引入 MR: [重构 client 模块，抽取 selector filter](https://git.woa.com/trpc-go/trpc-go/merge_requests/1299)

反馈 Issue: [[Bug Fixes] {client}: target、serviceName 生效顺序不符合预期](https://git.woa.com/trpc-go/trpc-go/issues/720)

反馈 Issue: [[Bug Fixes/ Features] 代码中指定 WithServiceName 之后，框架配置的 WithTarget 不生效](https://git.woa.com/trpc-go/trpc-go/issues/736)

#### 如果需要使用流式拦截器请更新到 v0.9.0 以上

**错误现象**

如果从 v0.8.2 ～ v0.8.6 版本升级到 v0.9.0 以上，用户自定义了流式拦截器，会发现流式拦截器签名的不兼容报错。

**解决方案**

避免使用 v0.8.2  ～ v0.8.6 版本的流式拦截器，这个阶段的流式拦截器签名定义不合理，v0.9.0 使用了不兼容变更的方式修改了拦截器签名。

**参考资料**

初版流式拦截器 MR: [feat: 流式支持拦截器 Filters](https://git.woa.com/trpc-go/trpc-go/merge_requests/1329)

最终版流式拦截器 MR: [流式拦截器支持 yaml 配置](https://git.woa.com/trpc-go/trpc-go/merge_requests/1347)

### v0.9.0【重要】

#### 【重大变更】server 拦截器和桩代码签名变更

v0.9.0 变更了服务端拦截器签名，v0.9.0 之前的服务端拦截器 rsp 是在入参中，而 v0.9.0 之后的服务端拦截器 rsp 移动到了出参。为了适配 v0.9.0 之后的拦截器签名，v0.9.0 之后桩代码中的处理函数签名也相应做了变更。

##### 修改 server 拦截器签名

```golang
// v0.9.0 前的拦截器格式：
func ServerFilter(ctx, req, rsp, next) error { 
    // 前置逻辑，这里的 rsp 是 nil
    err := next(ctx, req, rsp)
    // 后置逻辑，这里不能操作 rsp，会触发空指针 panic，或者断言失败
}

// v0.9.0 后的拦截器格式：
func ServerFilter(ctx, req, next) (rsp, error) { 
    // 前置逻辑
    rsp, err := next(ctx, req)
    // 后置逻辑，这里可以随意更改 rsp，甚至返回一个新的 rsp 结构体
}
```

如果你升级 tRPC-Go 框架到最新版本，并使用 v0.9.0 之前的服务端拦截器签名，启动服务时会出现如下提示

```log
filter: xxx is too old, please change to new ServerFilter, any question please refer to ChangeLog v0.9.0
```

此提示不会影响服务的正常启动，但是由于 v0.9.0 之前的拦截器入参 rsp 变成了空指针，可能触发运行时的 bug。所以需要将所有拦截器签名更新到 v0.9.0 之后的格式。如果是使用了官方提供的拦截器插件 [trpc-filter](https://git.woa.com/trpc-go/trpc-filter)，直接将插件版本更新到其最新版本即可。

##### 重新生成桩代码

由于桩代码会调用服务端拦截器，随着服务端拦截器的签名变更，桩代码也需要同步更新。需要使用最新版本的 [trpc cmdline](https://git.woa.com/trpc-go/trpc-go-cmdline) 工具重新生成桩代码时，会生成不同签名的桩代码。

```golang
// 旧版本的桩代码 Service interface 定义，rsp 是在入参中
type GreeterService interface {
    SayHello(ctx context.Context, req *HelloRequest, rsp *HelloReply) error

    SayHi(ctx context.Context, req *HelloRequest, rsp *HelloReply) error
}

// 新版本的桩代码 Service interface 定义，rsp 是在出参中
type GreeterService interface {
    SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error)

    SayHi(ctx context.Context, req *HelloRequest) (*HelloReply, error)
}
```

虽然不重新生成桩代码，直接使用旧版本的桩代码也是兼容 v0.9.0 编译的，但是实际运行中，可能会出现运行时 bug，所以建议使用最新版本（至少高于 v0.7.0）的 trpc cmdline 工具重新生成桩代码。

| 桩代码版本 | 框架版本 | filter 版本 | 结果     |
| ------|------| -------|---------|
| 旧    | 旧   | 旧     | ok      |
| 新    | 旧   | 旧     | 执行出错 |
| 旧    | 新   | 旧     | ok      |
| 旧    | 旧   | 新     | 编译出错 |
| 旧    | 新   | 新     | ok      |
| 新    | 旧   | 新     | 编译出错 |
| 新    | 新   | 旧     | 执行出错 |

#### 不要在旧版 server filter 修改 rsp

**错误现象**

```golang
// v0.9.0 前的旧版本拦截器格式：
func ServerFilter(ctx, req, rsp, next) error { 
    // 这里不能操作 rsp，会触发空指针 panic，或者断言失败
}
```

升级至 v0.9.0 以上版本，虽然框架同时支持新旧两种不同函数签名的 server filter，但是旧版本格式的 server filter rsp 会传入 nil，所以升级至 v0.9.0 以上版本后，不能在旧版本拦截器 server filter 里面操作 rsp 用于篡改回包数据，如果操作 rsp 会出现空指针异常。

**解决方案**

检查自定义的 server filter 是否有读取或写入 rsp，如果有读写操作可能会出现空指针 panic；如果没有读写 rsp 则无需修改，后续可以逐渐将自定义的 filter 重构为新版本的拦截器格式，tRPC-Go 提供的第三方 filter 插件已经全部升级为新版本的拦截器格式了。

**参考资料**

[【公示】v0.9.0 提示 filter xx is too old，并且导致 server filter rsp 断言失败](https://git.woa.com/trpc-go/trpc-go/issues/697)

#### 使用高于 v0.7.0 版本的 trpc-cmdline 工具

**错误现象**

当用户使用 v0.9.0 以上框架版本的时候，如果没有及时更新 [trpc 桩代码工具](https://git.woa.com/trpc-go/trpc-go-cmdline)，会发现生成的桩代码 rsp 还在入参中，和 v0.9.0 的框架最新 server filter 定义 rsp 在出参不符，导致报错：

```log
filter: xxx is too old, please change to new ServerFilter, any question please refer to ChangeLog v0.9.0
```

**解决方案**

在 v0.7.0 后 trpc-cmdline 工具默认生成 rsp 在出参的桩代码。
更新 [trpc 桩代码工具](https://git.woa.com/trpc-go/trpc-go-cmdline) 至最新版本，默认生成的桩代码 rsp 在出参里。

**参考资料**

[【公示】trpc-go-cmdline 生成工具从 v0.7.0 开始将默认生成 rsp 在出参的桩代码](https://git.woa.com/trpc-go/trpc-go/issues/755)

#### 如果新版本 server filter 修改 rsp 不生效，请更新到 v0.9.3 以上

**错误现象**

```golang
func ServerFilter(ctx, req, next) (rsp, error) { 
    _, err := next(ctx, req)
    // 后置逻辑，在这里返回了一个新的 rsp 结构体
    rsp = &pb.HelloReply{Msg: "intercepted response"}
    return rsp, err
}
```

升级至 v0.9.0 ～ v0.9.2 版本，使用了旧版本的桩代码，但是使用新版本的 server filter 拦截器，在拦截器里面操作 rsp 篡改回包数据，会发现修改的 rsp 没生效。这是由于 v0.9.0 定义新 server filter 拦截器签名的时候，在兼容旧的桩代码存在 Bug，没有对用户修改的 rsp 进行拷贝。

**解决方案**

升级框架至 v0.9.3 以上。尽快升级桩代码版本。
对于旧版本的桩代码，框架只对 protobuf 和 json 协议做了兼容，如果用了其他的序列化协议，可能会出 Bug。

**参考资料**

[修复 MR]<https://git.woa.com/trpc-go/trpc-go/merge_requests/1410>

### v0.9.1

#### 如果发现 HTTP CalleeMethod 截断了，请更新到 v0.11.0 以上

**错误现象**

升级至 v0.9.1 ～ v0.10.0 中的版本出现 server 侧 HTTP 的 CalleeMethod 从完整的 path 变为了只截取 "/" 最后一部分的内容（比如原来 CalleeMethod 拿到的是 /the/full/path，现在拿到的是 path）

**解决方案**

升级框架至 v0.11.0 以上

**参考资料**

引入 MR: [feat: tRPC-metrics-rules 实现](https://git.woa.com/trpc-go/trpc-go/merge_requests/1390)

修复 MR: [修复 codec 解析 callee method 不兼容](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFM7kkNAn9aSr6fhw6N5S)

反馈 Issue: [trpc-go/http msg.calleeMethod 只截取了 uri 最后一个/后的信息](https://git.woa.com/trpc-go/trpc-go/issues/747)

#### 尽量避免使用非四段式的 service name

**错误现象**

升级至 v0.9.1 以上的版本，如果客户端发情请求时，配置的 service name 不是 trpc.app.server.service 四段式结构（例如 oidb 的 qqconnect_oidb_0xb60），会出现 007 主调监控上报丢失，同时 CalleeServer 和 CalleeService 为空。

**解决方案**

在 client 配置文件里加上 target 指向被调服务真实的北极星名字，然后 name 用 4 段式写，就可以正常上报。

```yaml
name: trpc.app.server.service
target: polaris://qqconnect_oidb_0xb60
```

**参考资料**

引入 MR: [feat: tRPC-metrics-rules 实现](https://git.woa.com/trpc-go/trpc-go/merge_requests/1390)

反馈 Issue: [升级 trpc-go 版本到 v0.9.3 以后部分 007 主调监控数据丢失](https://git.woa.com/trpc-go/trpc-go/issues/892)

### v0.9.5

#### 如果在 client 端多个下游使用同一 pb service 时，请更新到 v0.10.0

**错误现象**

当使用了多个客户端配置，它们有不同的 service name，相同的 service callee。

```yaml
client:
  service:           
    - name: trpc.myapp.myserver.serviceA
      callee: trpc.same.server.callee
      target: "polaris://trpc.myapp.myserver.serviceA"

    - name: trpc.myapp.myserver.serviceB
      callee: trpc.same.server.callee         
      target: "polaris://trpc.myapp.myserver.serviceB"
```

代码指定了 service name，但是实际生效的却是第二个 target。

```golang
opt := client.WithServiceName("trpc.myapp.myserver.serviceA")
proxy := xsearch.NewProxyClientProxy(opt)
rsp, err := proxy.Search(ctx, req)
```

**解决方案**

更新 tRPC-Go 框架至 v0.10.0 以上

**错误原因**

v0.9.5 在修复 client 配置不生效时，引入了当前 bug，导致导致代码中的 service name 不再生效，变成真正的第二个 yaml service target 生效了。v0.10.0 以后支持使用 callee 和 service name 一起作为 key 来索引配置，修复了该问题。

**参考资料**

反馈码客：[[Bug Fixes] trpc-go 升级到 0.9.5 版本，原来多个同协议服务无法识别路由？](https://mk.woa.com/q/285242)

反馈 Issue: [在 client 端多个下游使用同一 pb service 时，yaml 中实际只有一个配置生效！](https://git.woa.com/trpc-go/trpc-go/issues/759)

修复 MR: [feat: use callee and servicename as a combined key to retrieve client config](https://git.woa.com/trpc-go/trpc-go/merge_requests/1535)

### Golang 升级至 1.18 出错，请升级框架至 v0.9.0 以上

**错误现象**

使用低于 v0.9.0 tRPC-Go 框架的服务，如果升级 golang 至 1.18 及以上，会出现 panic 报错

```golang
fatal error: fault
[signal SIGSEGV: segmentation violation code=xxx addr=xxx pc=xxx]
```

原因是低于 v0.9.0 版本的框架默认引用

```
github.com/json-iterator/go v1.1.10
```

go1.18 修改了 reflect map 的签名，导致低版本 [modern-go/reflect2](https://github.com/modern-go/reflect2/pull/25/files) panic。

json-iterator 依赖了 reflect2，导致 go1.18 编译的可执行文件会 [panic](https://github.com/json-iterator/go/issues/608)。

**解决方案**

升级 tRPC-Go 框架至 v0.9.0 以上。

**参考资料**

<https://git.woa.com/trpc-go/trpc-go/issues/684>
<https://git.woa.com/trpc-go/trpc-go/issues/764>
<https://git.woa.com/trpc-go/trpc-go/issues/768>

## 其他已回归问题

### trpc-go 升级到 v0.15.0，HTTP 请求返回的 rsp 为空？

**错误现象**

trpc-go 升级到 v0.15.0 后，发起 HTTP 请求，例如 Post 请求会出现 rsp 为空。

**解决方案**

升级框架至 v0.16.0 以上即可解决

**相关链接**

<http://mk.woa.com/q/291843>

### trpc-go v0.9.5 以下帧数据过大导致连接被关闭问题

**错误现象**

trpc-go 默认数据帧限制是 10MB，如果发送的数据帧大于 10MB，客户端会出现 EOF 或者 reset by peer 错误

**解决方案**

更新框架至 v0.9.5 以上客户端会有更详细的错误提示，而不是直接将连接关闭了。
要解决这个问题，需要同步修改 client 和 server 的帧大小限制，参考 iwiki。

**相关 MR**
[MR1056](https://git.woa.com/trpc-go/trpc-go/merge_requests/1056)
[MR1467](https://git.woa.com/trpc-go/trpc-go/merge_requests/1467)

## 版本规范

自 v0.10.0 以来（2022-11-03），tRPC-Go 的版本发布已经严格遵循 [semantic versioning](https://iwiki.woa.com/p/655870017) 规范，尽可能保证框架的 API 兼容性，更多版本信息见 [CHANGELOG](https://git.woa.com/trpc-go/trpc-go/blob/master/CHANGELOG.md)。

| 版本类型        | 示例          | 描述                                                                                  |
| -------------- | ------------- | ------------------------------------------------------------------------------------- |
| Major version  | v1.x.x        | 不同 Major version 不保证公共 API 的兼容性，常见于大版本变更。例如 v2.0.0 与 v1.10.0 不保证公共 API 的兼容 |
| Minor version  | vx.4.x        | 不同 Minor version 保证公共 API 的兼容性，常见于发布新 feature。例如 v1.12.0 与 v1.11.0 保证公共 API 的兼容 |
| Patch version  | vx.x.1        | 不同 Patch version 保证公共 API 的兼容性和稳定性，不用于发布新 feature，只做 bug fixes。例如 v0.12.1 修复 v0.12.0 引入的某个 bug。  |
