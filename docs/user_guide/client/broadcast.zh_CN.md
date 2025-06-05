# tRPC-Go 广播调用

## 1 前言

版本要求：v0.19.0-beta 已支持广播调用

tRPC-Go 框架支持广播调用（这里指的是主调对被调用的多个实例节点一次性发起调用），用户可以在重新生成的桩代码后，通过调用 `proxy.BroadcastXXX` 的方式发起调用，设计文档及背景如下：

[广播调用 - 桩代码版本](https://doc.weixin.qq.com/doc/w3_ASMAUQbxALEzXkHKeRsRFa14xle8A?scode=AJEAIQdfAAotgFRP0WASMAUQbxALE)

**提示：广播调用相对应的术语为单播调用，指 tRPC-Go 基本的一问一答的形式，本文中出现的普通调用指的也是这种调用形式。**

## 2 使用示例

### 2.1 前提基础

使用 tRPC-Go 的广播调用功能需要具备以下条件。

- 更新 trpc-go 到 v0.19.0-beta。
- 更新 trpc-go-cmdline >= v2.8.0。
  - 使用 `go install` 命令更新最新版本。

  ```shell
  go install trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc@latest
  ```

  - 如果已经存在 trpc-go-cmdline 二进制文件，直接执行 `trpc upgrade`。
- 重新生成广播调用版本的桩代码（`trpc create --broadcast ...`），具体可以参考中的 2.2 中的示例。
- 更新 naming-polaris >= v0.6.0，未合入且发布前，可以 replace 临时替换个人开发分支使用，具体步骤：
  - 把以下语句添加到 go.mod 文件中。

  ```gomod
  replace git.code.oa.com/trpc-go/trpc-naming-polaris => git.woa.com/nanjianyang/trpc-naming-polaris broadcast-02
  ```

  - 在终端中执行 go mod tidy。

  ```shell
  go mod tidy
  ```

  - 会自动拉取对应分支。

  ```gomod
  replace git.code.oa.com/trpc-go/trpc-naming-polaris => git.woa.com/nanjianyang/trpc-naming-polaris v0.5.18-0.20240913151003-00441b79db33
  ```

下面介绍详细的步骤。

### 2.2 生成广播调用桩代码

广播调用功能需要在桩代码中新增广播调用的接口，从而能让用户在主调代码中直接调该接口。
可以使用 `trpc create --broadcast ...` 的方式生成带有广播调用的桩代码。
例如以下这个 `helloworld.proto`：

```protobuf
syntax = "proto3";

package trpc.test.helloworld;
option go_package="git.code.oa.com/trpcprotocol/test/helloworld";

service Greeter {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
    string msg = 1;
}

message HelloReply {
    string msg = 1;
}
```

使用命令 `trpc create --broadcast --rpconly -p helloworld.proto`，可以生成带有广播调用接口的桩代码，
相当于在原理桩代码的基础上增加广播调用的接口，即在`xxx.trpc.go` 中新增广播调用相关代码。

```go
// START ======================================= Client Service Definition ======================================= START

// GreeterClientProxy defines service client proxy
type GreeterClientProxy interface {
    SayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error)
    // 新增广播调用接口
    BroadcastSayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) ([]*client.BroadcastRsp[HelloReply], error)
}

type GreeterClientProxyImpl struct {
    client client.Client
    opts   []client.Option
}

var NewGreeterClientProxy = func(opts ...client.Option) GreeterClientProxy {
    return &GreeterClientProxyImpl{client: client.DefaultClient, opts: opts}
}

func (c *GreeterClientProxyImpl) SayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) (*HelloReply, error) {
    ctx, msg := codec.WithCloneMessage(ctx)
    defer codec.PutBackMessage(msg)
    msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
    msg.WithCalleeServiceName(GreeterServer_ServiceDesc.ServiceName)
    msg.WithCalleeApp("test")
    msg.WithCalleeServer("helloworld")
    msg.WithCalleeService("Greeter")
    msg.WithCalleeMethod("SayHello")
    msg.WithSerializationType(codec.SerializationTypePB)
    callopts := make([]client.Option, 0, len(c.opts)+len(opts))
    callopts = append(callopts, c.opts...)
    callopts = append(callopts, opts...)
    rsp := &HelloReply{}
    if err := c.client.Invoke(ctx, req, rsp, callopts...); err != nil {
        return nil, err
    }
    return rsp, nil
}

// 新增广播调用接口实现
func (c *GreeterClientProxyImpl) BroadcastSayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) ([]*client.BroadcastRsp[HelloReply], error) {
    ctx, msg := codec.WithCloneMessage(ctx)
    defer codec.PutBackMessage(msg)
    msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
    msg.WithCalleeServiceName(GreeterServer_ServiceDesc.ServiceName)
    msg.WithCalleeApp("test")
    msg.WithCalleeServer("helloworld")
    msg.WithCalleeService("Greeter")
    msg.WithCalleeMethod("SayHello")
    msg.WithSerializationType(codec.SerializationTypePB)
    callopts := make([]client.Option, 0, len(c.opts)+len(opts))
    callopts = append(callopts, c.opts...)
    callopts = append(callopts, opts...)
    broadcastClient := client.NewBroadcastClient[HelloReply]()
    return broadcastClient.BroadcastInvoke(ctx, req, callopts...)
}
// END ======================================= Client Service Definition ======================================= END
```

需要注意，如果在生成广播调用桩代码的同时需要生成 `mock` 代码（无 `--rpconly`，无 `--mock=false`），默认会使用 `uber-go` 的 `mockgen` 进行替代，生成逻辑中会自动帮用户安装和生成，用户不需要额外操作。
这是因为广播调用功能的实现使用到了泛型的特性，需要使用 `ubermockgen` 才能支持泛型。

广播调用能力是主调的能力，被调的各个节点在被广播调用时与单播调用的感知无差别。
所以，如果用户暂时没有升级被调桩代码版本的计划，无需更新被调桩代码版本的依赖，只需要更新主调中对被调的桩代码版本的依赖。

例如，在 `A -> B` 的场景中，需要重新生成 `B` 的桩代码 `pB`，而 `A` 的桩代码 `pA` 的更新并不是必须的。
`A` 的代码中导入的对 `B` 的桩代码 `pB` 的依赖版本需要更新到重新生成的 `pB` 的版本，从而在 `A -> B` 时可以使用到广播调用的接口。
而 `B`代码本身对 `pB` 依赖版本更新并不是必须的。当然，A 和 B 的桩代码和依赖全部都更新也是可以的。

### 2.3 引入北极星名字服务插件

目前广播调用需要北极星名字服务的支持来获取广播调用的节点集，对应的插件就是 `trpc-naming-polaris`，需要在主调中匿名引入。

```go
import(
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)
```

可以理解为，主调能广播到的节点，前提是能被北极星找得到的节点。
因此，被调是需要被注册到北极星名字服务的，且主调与被调的网络是可达的。
例如，可以将主调服务与被调服务都部署在 `123` 平台上，这样会将服务自动注册到北极星名字服务，且这两个服务的网络是可达的。
这里主调也被注册到名字服务的好处是，可以根据一些路由规则来选定最终需要广播的节点范围，在后面会详细介绍。

为了复用框架本身的多环境路由等能力，目前采用 `WithServiceName` 的方式去使用 `trpc-naming-polaris`，不支持使用 `WithTarget` 的方式进行广播。
有关这两者的区别可以阅读 [tRPC-Go 北极星名字服务插件](https://git.woa.com/trpc-go/trpc-naming-polaris#clientwithservicename-寻址与-clientwithtarget-寻址的区别以及-enable_servicerouter-的语义)

使用北极星插件还需要对北极星进行配置，因为部署在 `123` 平台上的服务已经自动配置好 `trpc_go.yaml` 文件，因此这里不需要做调整。

### 2.4 被调代码示例

被调代码与单播调用相比，不需要做任何特殊处理。
从每个被调节点的视角看，它接受到的广播调用请求与一问一答的形式没有任何区别，并不能感知出是否是广播操作。
如果被调服务之前就存在，不需要做任何修改。
这里给一个示例：

```go
package main

import (
    _ "git.code.oa.com/trpc-go/trpc-filter/validation"
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/log"
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
    pb "git.woa.com/trpcprotocol/nanjdemo/nanjdemo_greeter"
)

func main() {
    s := trpc.NewServer()
    pb.RegisterGreeterService(s, &greeterImpl{})
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

```go
package main

import (
    "context"

    pb "git.woa.com/trpcprotocol/nanjdemo/nanjdemo_greeter"
)

type greeterImpl struct {
    pb.UnimplementedGreeter
}

func (s *greeterImpl) SayHello(
    ctx context.Context,
    req *pb.HelloReq,
) (*pb.HelloRsp, error) {
    rsp := &pb.HelloRsp{Msg: "Receive: " + req.Msg}
    return rsp, nil
}

```

### 2.5 主调代码示例

主调代码的整体流程与普通调用非常相似，例如只需要将之前普通调用的 `proxy.SayHello` 更换成 `proxy.BroadcastSayHello` 即可。
桩代码的生成规则就是在普通调用前增加 `Broadcast` 前缀形成广播调用的接口。

请使用 `WithServiceName` 的方式使用北极星插件，需注意避免有 `WithTarget` 选项或者框架配置文件中存在 `Target` 配置项。
`WithTarget` 的方式优先级高于 `WithServiceName` 方式，会覆盖后者。
目前暂时不支持 `WithTarget` 的方式使用广播调用。

```go
package main

import (
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
    pb "git.woa.com/trpcprotocol/nanjdemo/nanjdemo_greeter"
)

func main() {
    proxy := pb.NewGreeterClientProxy(
        client.WithNamespace("Development"),
        client.WithServiceName("trpc.NanjDemo.nanjDemo.Greeter"),
    )
    ctx := trpc.BackgroundContext()
    // 使用广播调用接口
    replies, err := proxy.BroadcastSayHello(ctx, &pb.HelloReq{Msg: "test"})
    if err != nil {
        // ...
    }
    // 使用 replies
    for _, reply := range replies {
        if reply.Err != nil {
            log.Errorf("error from node %s: %v", reply.Node.Address, reply.Err)
        } else {
            log.Debugf("broadcast rpc receive from node: %s, with: %+v", reply.Node.Address, reply.Rsp)
        }
    }
}
```

怎么使用广播调用的返回值？广播调用接口有两个返回值，先说第二个返回值。
广播调用的第二个返回值为 `error` 类型，表示整个广播调用的情况，具体如下：

- 该 `error` 为 `nil` 表示整体全部调用成功；
- 如果存在请求失败，则为广播调用错误，`error` 为多个失败子调用的 `multierror`；
- 存在失败时，`BroadcastRsp` 中失败的请求对应的 `err` 将附带原始的错误信息。

广播调用的第一个返回值是  `[]*BroadcastRsp[RspType]` ，用于表示广播调用返回结果的合集，其中 `BroadcastRsp` 在框架中定义：

```go
// BroadcastRsp is the generic broadcast response type.
type BroadcastRsp[RspType any] struct {
    Node *registry.Node
    Rsp  *RspType
    Err  error
}
```

- Node 表示节点信息，用于帮助用户确定被广播的节点信息。
- Rsp 表示由 proto 定义的响应
- Err 表示广播调用中每个具体调用返回的错误。

为什么广播调用的第二个返回值已经是 `multierror` 类型，`BroadcastRsp` 中还需要有单独的一个 `Err` 呢？

在框架的广播实现中，广播调用的每个子调用是并发执行的，`error` 很难区分是哪个子调用的对应起来。
所以使用 `Node-Rsp-Err` 的形式将一个子调用的信息聚合起来，方便用户使用和定位错误。

而用户想简便快捷的判断 `error` 则可以直接使用第二个返回值。需要了解其中每个错误，可以：

```go
if err != nil {
    // 检查是否是 multierror
    var merr *multierror.Error
    if errors.As(err, &merr) {
        log.Errorf("Broadcast encountered multiple errors: %v", merr)
        for _, subErr := range merr.Errors {
            log.Errorf("Sub error: %v", subErr)
        }
    }
}
```

补充：这里 `BroadcastRsp` 设计为泛型的原因是跟框架的设计有关，框架中实现广播调用时会帮用户收集 `response`。
但是框架不能获取 `response`的类型，使用反射的开销又很大，所以使用泛型的方式方便框架得到 `response` 的类型 `RspType`。

### 2.6 单向广播调用

tRPC-Go 支持单向广播调用，与普通的单向调用类似，只需要在创建 `proxy` 时或者发送请求时传入参数 `WithSendOnly()` 即可。
与使用普通单向调用类似，单向广播调用时，将不需要接收被调的响应即可返回，因此广播调用的第一个返回值将不会带有响应。
每个子调用的请求发送之后就立即返回，这是一种更快的广播方式，适用于事件通知等场景。

示例：

```go
_, err := proxy.BroadcastSayHello(ctx, &pb.HelloReq{Msg: "test"}, client.WithSendOnly())
```

## 3 广播调用路由规则

### 3.1 基本介绍

为了在广播中服用现有的路由逻辑，广播调用只支持 `WithServiceName` 的方式来路由。
因此，这里介绍的路由规则，均指的是 `WithServiceName` 的方式。
不特殊说明的情况下，后续的讨论主调和被调都部署在 `123` 平台上，即两者都被自动注册到了北极星服务上了。

广播调用的路由规则，主要是用于确定广播调用被调实例节点的范围。
在看广播的调用的路由规则之前，可以先阅读一下：

- [tRPC-Go 服务路由](https://iwiki.woa.com/p/4008319150)
- [tRPC-Go 多环境路由](https://iwiki.woa.com/p/99485673)
- [tRPC-Go Set 路由](https://iwiki.woa.com/p/118669392)
- [trpc-go-北极星名字服务插件](https://git.woa.com/trpc-go/trpc-naming-polaris#trpc-go-北极星名字服务插件)
- [规则路由使用指南](https://iwiki.woa.com/p/102467866)
- [就近路由](https://iwiki.woa.com/p/188713609)

普通调用设置路由规则的方式包括：

- 北极星控制台控制
- `yaml` 文件配置
- 代码设置

这些方式同样适用于广播调用的路由设置，后续会介绍。

使用 `WithServiceName` 寻址时，实际上使用的是框架中实现的 `trpcSelector`。
在匿名导入 `trpc-naming-polaris` 时，会将 `Discovery`、`ServiceRouter`、`Balancer` 全部重新注册成自己的。
不过，广播调用只需要用到 `Discovery` 和 `ServiceRouter`。
这是普通调用的寻址过程：

```raw
"trpc.app.server.service" =>  (trpc-naming-polaris).discovery.Discovery.List
                           =>  (trpc-naming-polaris).servicerouter.ServiceRouter.Filter        
                            =>  (trpc-naming-polaris).loadbalance.WRLoadBalancer.Select => ip:port  # WithServiceName
```

而广播调用相当于把负载均衡的步骤给去掉，还是会走服务发现和服务路由，即

```raw
"trpc.app.server.service" =>  (trpc-naming-polaris).discovery.Discovery.List
                           =>  (trpc-naming-polaris).servicerouter.ServiceRouter.Filter => ip:port slice  # WithServiceName
```

而 `ServiceRouter` 主要包括规则路由和就近路由，相当于：

```raw
+-------------+                                 +-------------+
|   服务发现   |                                 |   服务发现    |
+-------------+                                 +-------------+
       |                                               |           
       v                                               v
+------------------+                            +------------------+          
|     规则路由      |                            |    规则路由        |  
+------------------+                            +------------------+        
       |                                                |  
       v                                                v 
+------------------+                            +------------------+
|     就近路由      |      =================>    |    就近路由        | 
+------------------+                            +------------------+    
       |                                                |  
       v                                                v 
+------------------+                            +------------------+        
|     负载均衡      |                            |    广播节点集      |
+------------------+                            +------------------+    
       |                              
       v
+------------------+                             
|     单个节点      |
+------------------+                             
```

因此可以说，普通调用的路由规则适用于广播调用。
可以理解为，在使用普通调用的时候，能调用到的节点来源哪个节点集，能广播到的范围就是哪个节点集。
普通调用只是在这个节点集的基础上，再进行一次负载均衡。

有一下几种路由规则可能会影响到广播的范围，请在使用广播调用前预先了解哪些范围的实例将会被广播到。

- 环境路由
- 就近路由
- `Set` 路由

当多个路由规则生效时，最终的广播范围是每个路由规则共同限制出来的范围。
当使用 `123` 平台部署主调和被调时，如果不进行额外的配置，默认是开启环境路由和就近路由的，`Set` 路由为关闭状态。
下面详细介绍一下每个规则对广播的影响。

### 3.2 环境路由

- 默认开启。
- 不同基线环境的服务不能互相广播。
- 测试环境的主调不能广播到正式环境的节点。
- 只能广播到本环境的所有节点，本环境的服务没有节点时广播到基线环境的所有节点。

例如，主调和被调在基线环境和对应的特性环境都有节点。
主调在基线环境发起广播，广播的范围为基线环境的节点；
主调在特性环境发起广播，广播范围为特性环境的节点，当特性环境没有节点时，广播范围变成基线环境的节点。

如果想配置环境路由的规则来改变广播的范围，可以参考 [tRPC-Go 多环境路由](https://iwiki.woa.com/p/99485673)。例如希望广播到指定的环境 `62a30eec`：

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    // 设置被调服务环境
    client.WithCalleeEnvName("62a30eec"),
    // 关闭服务路由
    client.WithDisableServiceRouter()
}
proxy := pb.NewGreeterClientProxy(opts...)
```

### 3.3. 就近路由

- 默认开启。
- 默认广播到与主调同城市的所有节点。
- 当同城没有节点时，会广播到同区域的所有节点。
- 当同区域没有可用实例时，会广播到任何可用节点。

例如，被调在深圳和广州都有节点，主调在深圳发起广播，默认会广播到深圳的节点，而不会广播到广州的节点。

就近规则的策略可以参考 [就近路由](https://iwiki.woa.com/p/188713609) 进行调整。

比如把就近路由关闭，可以将广播调用的范围扩大到所有区域所有城市的所有节点。
这个操作可以在 123 平台上进行设置，也可以在代码中使用 `servicerouter.WithDisableNearbyRouter(ctx)` 的方式关闭。示例：

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
}
proxy := pb.NewGreeterClientProxy(opts...)
replies, err = proxy.BroadcastSayHello(servicerouter.WithDisableNearbyRouter(ctx), &pb.HelloReq{Msg: "test"})
```

### 3.4 Set 路由

- 在 123 平台上部署服务时，可以给主调和被调设置好 Set 属性，主调向被调发起广播调用时会遵守 Set 路由规则。
- 当 Set 的第一段一样时，就认为启用了 Set 路由，并根据完整的三段 Set 名进行匹配，找到广播的节点集。
- 可以根据通配符等进行对广播范围的控制，完整的 Set 路由可参考 [tRPC-Go Set 路由](https://iwiki.woa.com/p/118669392)。

例如，被调服务在 `set.sz.1`、`set.sz.2` 和 `set.gz.1` 都各有 2 个节点。
当主调的 `Set` 为 `set.sz.1` 时，广播到被调的节点为 `set.sz.1` 中的 `2` 个节点；
当主调的 `Set` 为 `set.sz.*` 时，广播到的节点为 `set.sz.1` 与 `set.sz.2` 中的 4 个节点。

也可以通过代码的方式指定广播的 `Set`：

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithCallerSetName("a.b.c")
    // 注意不要用 WithTarget 的方式，使用 WithServiceName
    client.WithServiceName("trpc.settestapp.settestserver.Greeter"),
}
proxy := pb.NewGreeterClientProxy(opts...)
```

注意，`Set` 路由和就近路由不能同时起效，启用了 `Set` 路由后就近路由规则会失效。

### 更多广播调用路由方式

不管是环境路由还是 `Set` 路由本质上都是借助了北极星的规则路由，也就是说，如果想自定义广播的范围，可以参考 [北极星规则路由](https://iwiki.woa.com/p/102467866) 进行调整。

示例场景：

- 需要广播到所有区域的所有城市：关闭就近路由，关闭 `Set` 路由。
- 此外，还需要广播到 `Development` 所有环境：使用 `WithDisableServiceRouter()` 关闭服务路由即可。示例：

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    client.WithDisableServiceRouter(),
}
proxy := pb.NewGreeterClientProxy(opts...)
```

## 4 FAQ

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
