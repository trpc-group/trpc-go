# 1 前言

金丝雀环境，通过使新特性只对少数用户可用，可降低向每个人推出新代码和功能的风险。是在现有正式环境外创建了一个全新的独立生产环境，将少部分用户路由到新的金丝雀环境以验证新特性。一旦证明金丝雀版本稳定并交付预期结果，剩余的用户就被路由到新环境。如果金丝雀发布存在问题，那么金丝雀环境的流量将被路由回正式环境。

这项技术以著名的短语“煤矿中的金丝雀”命名，它起源于煤矿工人使用金丝雀作为早期检测系统来识别有毒气体的危险程度。类似地，金丝雀发布是软件的早期检测和反馈系统。

# 2 原理和实现

[设计文档](https://git.woa.com/trpc/trpc-proposal/blob/master/A3-canary.md)

在北极星插件的 service router 中，添加 canaryRouter 金丝雀路由插件，按以下顺序进行寻址：

set 路由

1. 就近路由（set 路由与就近是互斥关系，有 set 就不会执行就近路由）
2. 金丝雀路由
3. 按以上顺序执行完 set 路由和就近路由后返回了一批节点集，然后开始进行金丝雀路由，逻辑如下：

判断被调服务是否存在 internal-canary 标签，有则进入金丝雀路由，没有则退出。
插件入参是 canary，参数值为透传字段的 $value，没有透传字段则为空：

1. 参数值非空，则过滤服务实例列表中带有 canary: $value 的实例，假如不存在，则返回全量。如 tRPC 框架透传字段为 trpc-canary=1，则北极星 sdk 过滤出带有 canary:1 标签的实例。
2. 参数值为空：则过滤服务实例列表中不带有 canary 的 key 的实例，假如不存在，则返回全量。

# 3 示例

需在 trpc 框架配置增加：

```yaml
selector:                                         # 针对 trpc 框架服务发现的配置
  polaris:                                        # 北极星服务发现的配置
    enable_canary: true                           # 开启金丝雀功能，默认 false 不开启
```

```go
package main

import (
    "context"
    "time"

    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/naming/registry"
    "git.code.oa.com/trpc-go/trpc-naming-polaris/servicerouter"

    pb "git.code.oa.com/trpcprotocol/test/helloworld"
)

func main() {
    ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
    defer cancel()

    node := &registry.Node{}
    opts := []client.Option{
        client.WithServiceName("your service"),
        client.WithNamespace("Production"),
        client.WithSelectorNode(node),
        // 指定金丝雀 key
        servicerouter.WithCanary("1"),
    }

    proxy := pb.NewGreeterClientProxy()
    req := &pb.HelloRequest{
        Msg: "trpc-go-client",
    }
    rsp, err := proxy.SayHello(ctx, req, opts...)
    log.Debugf("req: %s, rsp: %s, err: %v, node: %+v", req, rsp, err, node)
}
```

# 4 FAQ

- 目前金丝雀仅在正式环境生效。
- 有不理解的请先仔细阅读设计文档。
- 问题定位，开启框架的 trace 日志，开启方式请查看 [这里](https://git.woa.com/trpc-go/trpc-go/tree/master/log)，贴出 [NAMING-POLARIS] 为前缀的日志。
- 请更新到最新版本北极星插件

## trpc-go 服务如何在其他平台使用金丝雀路由

例如：智研平台/tkex-csig 可参考如下连接：

- https://mk.woa.com/q/295304
- https://mk.woa.com/q/291361

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
