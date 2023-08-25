[TOC]

slime version: [v0.3.0](https://git.woa.com/trpc-go/trpc-filter/tree/slime/v0.3.0/slime)
slime [changelog](https://git.woa.com/trpc-go/trpc-filter/tree/slime/v0.3.0/slime/CHANGELOG.md)



# 支持的协议
<span style="color:red">**请不要对非幂等请求开启重试/对冲功能**</span>。
<span style="color:red">并非所有协议都能使用重试/对冲</span>。如果你使用的协议（或相应版本）没有出现在下面的列表中，请联系 cooperyan 进行确认，我们会将结果补充到下面的表格中。
对于非 trpc 协议，可能并不适用第五章的 yaml 配置，这时，你可以直接使用第四章的基础包。

| 协议 | 重试 | 对冲 | 备注 |
|:-:|:-:|:-:|:-|
|trpc ≥ v0.5.0|✓|✓| 原生的 trpc 协议。 |
|trpc SendOnly|✗|✗| 不支持，重试/对冲根据返回的错误码进行判断，而 SendOnly 请求不会回包。 |
|trpc 流式|✗|✗| 暂不支持。 |
|[http](https://git.woa.com/trpc-go/trpc-go/tree/master/http)|✓|✓| slime v0.2.2 后支持。 |
|[tars](https://git.woa.com/trpc-go/trpc-codec/tree/tars/v1.2.9/tars)|✓|✓| slime v0.2.0 后支持。目前需要在 slime 前配置一个额外的 filter 来使配置文件生效，参考这个 [demo](https://git.woa.com/cooperyan/greetings/blob/master/client/trpc-tars/main.go#L37)。 |
|[Kafaka](https://git.woa.com/trpc-go/trpc-database/tree/master/kafka) ≥ v0.1.5|✓|✗| 对冲功能在测试中。 |
|[MySQL](https://git.woa.com/trpc-go/trpc-database/tree/master/mysql) ≥ v0.1.6|★|★| slime v0.2.2 后，<span style="color:red">除 [Query](https://git.woa.com/trpc-go/trpc-database/blob/mysql/v0.1.6/mysql/client.go#L27) 和 [Transaction](https://git.woa.com/trpc-go/trpc-database/blob/mysql/v0.1.6/mysql/client.go#L30) 两个方法外，其他都支持</span>。这两个方法以函数闭包作为参数，slime 无法保证数据的并发安全性，可以使用 5.6 节的 `slime.WithDisabled` 关闭重试/对冲。 |
|[Redis](https://git.woa.com/trpc-go/trpc-database/tree/master/redis) ≥ v0.1.6|✓|✓| slime v0.2.0 后支持。 |
|[trpc-go-union](https://git.woa.com/videocommlib/trpc-go-union) ≥ v0.1.2|✓|✓||
|[oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb)/[oidb1](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb1)/[oidb3](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb3)|✓|✓| slime v0.2.2 后支持。 |
|[ckv](https://git.woa.com/trpc-go/trpc-database/tree/ckv/v0.4.2/ckv)|✗|✗| 不支持。 |
|[es](https://git.woa.com/trpc-go/trpc-database/tree/es/v0.1.0/es)|✗|✗| 不支持。 |
|[goredis](https://git.woa.com/trpc-go/trpc-database/tree/master/goredis)|✗|✗| 所有基于 [gcd/go-utils/comm/joinfilters](https://git.woa.com/gcd/go-utils/tree/master/comm/joinfilters) 的协议都不支持，因为 jointerfilters 不支持拦截器并发。 |
|[TDMQ](https://git.woa.com/trpc-go/trpc-database/tree/master/tdmq) ≥ v0.2.9|✓|✓| slime v0.2.2 后支持。 |


# 前言
重试是一个很朴素的想法，当原请求失败时，发起重试请求。狭义的重试是一个比较保守的策略，只有当上次请求失败后，才会触发新的请求。对响应时间有要求的用户可能希望使用一种更加激进的策略，**对冲策略**。Jeffrey Dean 在 [the tail at scale](https://cacm.acm.org/magazines/2013/2/160173-the-tail-at-scale/pdf)  中首次提到了策略，以解决扇出数很大时，长尾请求对整个请求时延的影响。

简单地讲，对冲策略并不是被动地等待上一次请求超时或失败。在对冲延迟时间（小于超时时间）内，如果未收到成功的回包，就会再触发一个新的请求。与重试策略不同的是，同一时间可能有多个 in-flight 请求。第一个成功的请求会被交给应用层，其他请求的回包会被忽略。

注意，这两种策略具有互斥性，用户只能二选一。

重试策略实现比较简单。对冲策略业界也有了一些实现：
* [gRPC](https://github.com/grpc/grpc-java)：[A6-client-retries.md](https://github.com/grpc/proposal/blob/master/A6-client-retries.md) 详细介绍了 gRPC 的设计方案。gRPC-java 已经实现了该方案。
* [bRPC](https://github.com/apache/incubator-brpc)：在 bRPC 中，hedging request 被称为 backup request。这个[文档](https://github.com/apache/incubator-brpc/blob/master/docs/cn/backup_request.md)作了粗略的介绍，其 c++ 实现也比较简单。
* [finagle](https://github.com/twitter/finagle)：finagle 是一个 java 的 RPC 开源框架，它也实现了 [backup request](https://twitter.github.io/finagle/guide/MethodBuilder.html#backup-requests)。
* [pegasus](https://github.com/apache/incubator-pegasus)：pegasus 是一个 kv 型数据库，它通过 [backup request](https://github.com/apache/incubator-pegasus/issues/251) 来支持从多副本同时读取数据以提高性能。
* [envoy](https://www.envoyproxy.io/docs/envoy/latest/)：envoy 作为一个代理服务，在云原生中有广泛应用。它也支持了 [request hedging](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_routing#request-hedging)。

本文将介绍 tRPC 框架的重试和对冲能力。在第二章，我们简要介绍了重试对冲的基本原理。在第三章，我们列举了一些简单的示例，通过它们，你可以快速地将重试/对冲功能应用到你自己的项目中。后面两章介绍了更多的实现细节。第四章，我们介绍了重试/对冲的基础包，第五章介绍的 slime 是一个基于这些基础能力的管理器，它可以为你提供基于 yaml 的配置能力。最后，我们列举了一些你可能会有疑问的点。

如果你对更多的设计细节感兴趣，可以参考 [A0-client-retries](https://git.woa.com/trpc/trpc-proposal/blob/master/A0-client-retries.md) 这篇 proposal。对实现细节感兴趣，可以直接阅读 [slime](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime) 的源码。

如果你有任何疑问，请通过下面的方式告诉我们，我们会尽快帮你解决：
* 在这篇文档下评论。
* 在 [slime](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime) 下提交 issue（请注明 slime 插件）。
* 在 [proposal](https://git.woa.com/trpc/trpc-proposal) 中提交 issue 讨论。
* 直接联系 cooperyan 或 jessemjchen。

# 原理
在本章中，我们将通过两张图展示对冲和重试的基本原理，并简要介绍一些其他你可能需要关注的能力。

## 重试策略
顾名思义，对错误的回包进行重试。

![ 'image.png'](/.resources/user_guide/retry_hedging/retry.png)

上图中，client 一共尝试了三次：橙、蓝、绿。前两次都失败了，并且在每一次尝试前都会随机退避一段时间，以防止请求毛刺。最终第三次尝试成功了，并返回给了应用层。另外，也可以看到，对于每次尝试，我们都会尽可能地将请求发往不同的节点。

一般，重试策略需要有以下配置：
- 最大重试次数：一旦耗尽，便返回最后一个错误。
- 退避时间：实际的退避时间取的是 random(0, delay)。
- 可重试错误码：如果返回的错误是不可重试的，就立刻停止重试，并将错误返回给应用层。

## 对冲策略
正如我们在前言中介绍的，对冲可以看作是一种更加激进的重试，它比重试更复杂。

![ 'image.png'](/.resources/user_guide/retry_hedging/hedging.png)

上图中，client 一共尝试了 4 次：橙、蓝、绿、紫。
<span style="color:F2B442">橙色</span>是第一次尝试。在由 client 发起后，server2 很快便收到了。但是 server2 的因为网络等问题，直到绿色请求成功，并返回给应用层后，它的正确回包才姗姗来迟。尽管它成功了，但我们必须丢弃它，因为我们已经将另一个成功的回包返回给应用层了。
<span style="color:2765FF">蓝色</span>是第二次尝试。因为橙色请求在对冲时延（hedging delay）后还没有回包，因此我们发起了一次新的尝试。这次尝试选择了 server1（我们会尽可能地为每次尝试选择不同的节点）。蓝色尝试的回包比较快，在对冲时延之前便返回了。但是却失败了。我们**立刻**发起了新一次尝试。
<span style="color:A4D955">绿色</span>是第三次尝试。尽管它的回包可能有点慢（超过了对冲时延，因此又触发了一次新的尝试），但是它成功了！一旦我们收到第一个成功的回包，便立刻将它返回给了应用层。
<span style="color:B937F6">紫色</span>是第四次尝试。刚发起后，我们便收到了绿色成功的回包。对紫色来说，它可能处于很多状态：请求还在 client tRPC 内，这时，我们有机会取消它；请求已经进入了 client 的内核或者已经由网卡发出，无论如何，我们已经没有机会取消它了。紫色请求上的 <span style="color:F20F79">✘</span> 表示我们会尽可能地取消紫色请求。注意，即使紫色请求最终成功地到达了 server2，它的回包也会像橙色一样被丢弃。

可以看到，对冲更像是一种添加了**等待时间**的**并发**重试。需要注意的是，对冲没有退避机制，一旦它收到一个错误回包，就会立刻发起新的尝试。通常，我们建议，只有当你需要解决请求的长尾问题时，才使用对冲策略。普通的错误重试请使用更加简单明了的重试机制。

一般，对冲会有以下配置：
- 最大重试次数：一旦耗尽，便等待并返回最后一个回包，无论它是否成功或失败。
- 对冲时延：在对对冲时延内没有收到回包时便会立刻发起新的尝试。
- 非致命错误：返回致命错误会立刻中止对冲，等待并返回最后一个回包，无论它是否成功或失败。返回非致命错误会立刻触发一次新的尝试（对冲时延计时器会被重置）。

## 拦截器次序
在 tRPC-Go 中，对冲/重试功能是在拦截器中实现的。

一个应用层请求在经过重试/对冲拦截器后，可能会产生多个子请求，每个子请求都执行一遍后续的拦截器。
对于监控类拦截器，你必须注意它们与重试/对冲拦截器的相对位置。如果它们位于重试/对冲之前，那么应用层每一个请求它们只会统计一次；如果它们位于重试/对冲之后，那么，每一次重试对冲请求它们都会统计。

当你使用重试/对冲拦截器时，请务必多思考一下它与其他拦截器的相对关系。

## server pushback
server pushback 用于服务端显式地控制客户端的重试/对冲策略。
当服务端负载比较高，希望客户端降低重试/对冲频率时，可以在回包中指定延迟时间 T，客户端会将下一次重试/对冲子请求延迟 T 时间后执行。
该功能更常用于服务端指示客户端停止重试/对冲，通过将 delay 设置为 `-1` 即可。

一般情况下，你不应该关心是否需要设置 server pushback。在后续规划中，框架会根据服务当前的负载情况，自动决定如何设置 server pushback。

## 负载均衡
因为重试/对冲是以拦截器的方式实现的，而负载均衡发生在拦截器之后，因此，每一个子请求都会触发一次负载均衡。

![ 'image.png'](/.resources/user_guide/retry_hedging/loadbalance.png)

对于对冲请求，你可能希望每个子请发往不同的节点。我们实现了一个机制，允许多个子请求间进行通信，以获取其他子请求已经访问过的节点。负载均衡器可以利用该机制，只返回未访问过的节点。当然，这需要负载均衡器的配合，目前只有两个框架内置的随机负载均衡策略支持。我们会尽快为其他负载均衡器提供支持。
如果你使用的负载均衡器不支持跳过已经访问过的节点，也不用灰心丧气。一般情况下，轮询或随机的负载均衡器本身就在某种意义上实现了子请求发往不同的节点，即使偶尔发往了同一个节点，也不会有什么大问题。而对于特殊的 hash 类负载均衡器（按某个特定的 key 路由到特定的一个节点，而非一类节点），它可能根本无法支持这个功能，事实上，在这类负载均衡器上使用对冲策略是没有意义的。

# 快速上手
clone 仓库 [greetings](https://git.woa.com/cooperyan/greetings)，重试/对冲客户端示例都在 `client/trpc-client-retries` 目录下。
## 重试
请参考 [retry](https://git.woa.com/cooperyan/greetings/tree/master/client/trpc-client-retries/retry)。
## 对冲
我们提供了两个对冲示例。
[hedging](https://git.woa.com/cooperyan/greetings/tree/master/client/trpc-client-retries/hedging) 以一种相对比较夸张的方式（服务端频繁地失败或延时）展示了对冲的效果。
[hedging_long_tail](https://git.woa.com/cooperyan/greetings/tree/master/client/trpc-client-retries/hedging_long_tail) 展示了对冲是如何解决长尾请求的。

### 如何确定对冲延迟？
下图是 [hedging_long_tail](https://git.woa.com/cooperyan/greetings/tree/master/client/trpc-client-retries/hedging_long_tail) 给出的 [CDF](https://en.wikipedia.org/wiki/Cumulative_distribution_function) 曲线。

![ 'image.png'](/.resources/user_guide/retry_hedging/cdf.png)

观察蓝色的 baseline，我们发现，P95 以上时延分布在 5~50ms 之间。为了减小平均 P95 时延，我们可以将 hedging delay 设置为 P95 处的 5ms。
红色的 hedging 是我们开启对冲后的效果。P95 以上的平均耗时减少到了 10ms 左右。

当然，具体的服务应该具体分析。但是有一个原则，只有要解决长尾问题时（比如，对超时进行重试，请参考 4.4 节的说明），你才需要使用对冲策略。而且，对冲时延不要设置得太小，最好取 P90 以上。
> 注意，如果你将对冲延时设置为 P90 以下，你需要同步地更改对冲限流。因为默认限流允许的写放大比例是 110%。

# retry hedging 基础包介绍
本章只是简要介绍重试/对冲的基础包，以作为第四章的基础。尽管我们提供了一些使用范例，但还是请<span style="color:red">尽量避免直接在应用层使用它们</span>。你应该通过 [slime](#5 slime) 来使用重试/对冲功能。

## [retry](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime/retry)
[retry](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime/retry) 包提供了基础的重试策略。

`New` 创建一个新的重试策略，你必须指定最大重式次数和可重试错误码。你也可以通过 `WithRetryableErr` 自定义可重试错误，它和可重试错误码是或关系。

retry 提供了两种默认的退避策略：`WithExpBackoff` 和 `WithLinearBackoff`（相关参数说明请参考 proposal 中 [配置的有效性检验](https://git.woa.com/trpc/trpc-proposal/blob/master/A0-client-retries.md#check-cfg-retry)）。你也可以通过 `WithBackoff` 自定义退避策略。这三种退避策略至少需要提供一种，如果你提供了多个，它们的优先级为：
`WithBackoff` > `WithExpBackoff` > `WithLinearBackoff`

你可能会奇怪，为什么 `WithSkipVisitedNodes(skip bool)` 有一个额外的 `skip` 布尔变量？事实上，我们在这里区分了三种情形：
1. 用户未显式地指定是否跳过已访问过的节点；
2. 用户显式地指定跳过已访问过的节点；
3. 用户显式地指定不要跳过已访问过的节点。

这三种状态会对负载均衡产生不同的影响。
对第一种情形，负载均衡应该尽可能地返回未访问过的节点。如果所有节点都已经访问过了，我们允许它返回一个已经访问过的节点。这是默认策略。
对第二种情形，负载均衡必须返回未访问过的节点。如果所有节点都已经访问过了，它应该返回无可用节点错误。
对第三种情形，负载均衡可以随意返回任何节点。
如 2.5 节中描述的，`WithSkipVisitedNodes` 需要负载均衡的配合。如果负载均衡器未实现该功能，无论用户是否调用了该 option，最终都对应于第三种情形。

`WithThrottle` 可以为该策略指定限流器。

你可以通过以下方式为某次 RPC 请求指定重试策略：
```Go
r, _ := retry.New(4, []int{errs.RetClientNetErr}, retry.WithLinearBackoff(time.Millisecond*5))
rsp, _ := clientProxy.Hello(ctx, req, client.WithFilter(r.Invoke))
```

## [hedging](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime/hedging)
[hedging](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime/hedging) 包提供了基础的对冲策略。

`New` 创建一个新的对冲策略。你必须指定最大重试次数和非致命错误码。你也可以通过 `WithNonFatalError` 自定义非致命错误，它和非致命错误码是或关系。

hedging 包提供两种方式来设置对冲延时。`WithStaticHedgingDelay` 设置一个静态的延迟。`WithDynamicHedgingDelay` 允许你注册一个函数，每次调用时返回一个时间作为对冲延时。这两种方法是互斥的，多次指定时，后者会覆盖前者。

`WithSkipVisitedNodes` 的行为与 retry 一致，请参考上节。

`WithThrottle` 可以为对冲策略指定限流器。

你可以通过以下方式为某次 RPC 请求指定对冲策略：
```Go
h, _ := hedging.New(2, []int{errs.RetClientNetErr}, hedging.WithStaticHedgingDelay(time.Millisecond*5))
rsp, _ := clientProxy.Hello(ctx, req, client.WithFilter(h.Invoke))
```

## [throttle](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime/throttle)
[throttle](https://git.woa.com/trpc-go/trpc-filter/tree/master/slime/throttle) 实现了 proposal [对重试/对冲请求进行限流](https://git.woa.com/trpc/trpc-proposal/blob/master/A0-client-retries.md#throttle) 中的限流方案。

`throttler` interface 提供了三个方法：
```Go
type throttler interface {
	Allow() bool
	OnSuccess()
	OnFailure()
}
```
每次发送重试/对冲子请求（不包括第一次请求），都会调用 `Allow`，如果返回 `false`，那么这个应用层请求的所有后续子请求都不会再执行，视作「最大对冲次数已经耗尽」。
每当收到重试/对冲子请求的回包时，会根据情况调用 `OnSuccess` 或 `OnFailure`。更多细节还请参考 proposal。

对冲/重试会产生写放大，而限流则是为了避免因重试/对冲造成服务雪崩。当你初始化一个如下 throt，并将它绑定到一个 `Hello` RPC 时，
```Go
throt, _ := throttle.NewTokenBucket(10, 0.1)
r, _ := retry.New(3, []int{errs.RetClientNetErr}, retry.WithLinearBackoff(time.Millisecond*5))
tr := r.NewThrottledRetry(throt)
rsp, _ := clientProxy.Hello(ctx, req, client.WithFilter(tr.Invoke))
```
因重试/对冲产生的总 `Hello` 请求数不会超过应用层次数的 110%（每一个成功的请求会使令牌加 0.1，每一个失败的请求会使令牌减少 1，相当于 10 个成功的请求才能换取来一次重试/对冲的机会），突增的重试/对冲请求数（连续失败）不会大于 5（5 = 10 / 2，只有令牌数大于一半时，`Allow` 才会返回 `true`）。

## 关于超时错误
在 tRPC-Go 中，[`RetClientTimeout`](https://git.woa.com/trpc-go/trpc-go/blob/master/errs/errs.go#L29)，即 101 错误，对应应用层超时。重试/对冲遵循该机制，只要 `ctx` 超时，就会立刻返回错误。因此，<span style="color:red">将 101 作为可重试/对冲错误码是没有意义的</span>。对这种情况，我们建议你使用对冲功能，并配置合理的对冲时延（相当于对冲时延即为你期望的超时时间）。注意，对冲时延应该小于应用层超时时间。

# slime

> <span style="color:red">[slime 不支持从 tconf 或七彩石进行初始化](https://git.woa.com/trpc-go/trpc-go/issues/502)。如果你使用它们管理 client 配置，那么请将重试/对冲直接配在本地文件的 `plugins` 下面，或者使用第四章的基础包。</span>

[slime](todo) 在 [retry]() 和 [hedging]() 两个基础包之上，提供了文件配置功能。利用 slime，你可以将重试/对冲策略统一管理在框架配置中。和其他 tRPC-Go 的插件一样，首先匿名导入 slime 包：
```go
import _ "git.code.oa.com/trpc-go/trpc-filter/slime"
```

我们以下面这个 yaml 文件为例，介绍 slime 是如何解析配置文件的。
```yaml
--- # 重试/对冲策略
retry1: &retry1 # 这是 yaml 引用语法，可以允许不同 service 使用相同的重试策略
  # 省略时，将会随机生成一个名字。
  # 如果需要自定义 backoff 或可重试业务错误，必须显式地提供一个名字，它会用于 slime.SetXXX 方法的第一个参数。
  name: retry1
  # 省略时，将取默认值 2。
  # 最大不超过 5。超过时，将自动截断为 5。
  max_attempts: 4
  backoff: # 必须提供 exponential 或 linear 中的一个
    exponential:
      initial: 10ms
      maximum: 1s
      multiplier: 2
  # 省略时，会默认重试以下四种框架错误：
  # 21: RetServerTimeout
  # 111: RetClientConnectFail
  # 131: RetClientRouteErr
  # 141: RetClientNetErr
  # tRPC-Go 的框架错误码请参考：https://git.woa.com/trpc-go/trpc-go/tree/master/errs
  retryable_error_codes: [ 141 ]

retry2: &retry2
  name: retry2
  max_attempts: 4
  backoff:
    linear: [100ms, 500ms]
  retryable_error_codes: [ 141 ]
  skip_visited_nodes: false # 省略、false 和 true 对应三种不同情形

hedging1: &hedging1
  # 省略时，将会随机生成一个名字。
  # 如果需要自定义 hedging_delay 或者非致命错误，必须显式地提供一个名字，它会用于 slime.SetHedgingXXX 方法的第一个参数。
  name: hedging1
  # 省略时，将取默认值 2。
  # 最大不超过 5。超过时，将自动截断为 5。
  max_attempts: 4
  hedging_delay: 0.5s
  # 省略时，以下四种错误默认为非致命错误：
  # 21: RetServerTimeout
  # 111: RetClientConnectFail
  # 131: RetClientRouteErr
  # 141: RetClientNetErr
  non_fatal_error_codes: [ 141 ]

hedging2: &hedging2
  name: hedging2
  max_attempts: 4
  hedging_delay: 1s
  non_fatal_error_codes: [ 141 ]
  skip_visited_nodes: true # 省略、false 和 true 对应三种不同情形，见 4.1 节

--- # 配置
client: &client
  filter: [slime] # filter 要和 plugin 相互配合，缺一不可
  service:
    - name: trpc.app.server.Welcome
      retry_hedging_throttle: # 该 service 下的所有重试/对冲策略都会和该限流绑定
        max_tokens: 100
        token_ratio: 0.5
      retry_hedging: # service 默认使用策略 retry1
        retry: *retry1 # dereference retry1
      methods:
        - callee: Hello # 使用重试策略 retry2 覆盖 service 策略 retry1
          retry_hedging:
            retry: *retry2
        - callee: Hi # 使用对冲策略 hedging1 覆盖 service 策略 retry1
          retry_hedging:
            hedging: *hedging1
        - callee: Greet # retry_hedging 的内容为空，即不使用任何重试/对冲策略
          retry_hedging: {}
        - callee: Yo # 没有 retry_hedging，采用 service 默认策略 retry1
    - name: trpc.app.server.Greeting
      retry_hedging_throttle: {} # 强制关闭限流功能
      retry_hedging: # service 默认使用策略 hedging2
        hedging: *hedging2
    - name: trpc.app.server.Bye
      # 没有配置限流，使用默认限流
      # 没有配置 service 级别的重试/对冲策略
      methods:
        - callee: SeeYou # 为 SeeYou 方法单独配置了重试策略
          retry_hedging:
            retry: *retry1

plugins:
  slime:
    # 这里引用了整个 client。当然，你可以将 client.service 单独配在 default 下。
	# 在使用 tconf 或 rainbow 管理 client 配置时，必须在这里直接配置，不能使用 yaml 引用。
    default: *client
```

> 上面的配置文件用到了 yaml 中的一个重要的特性，即[引用](https://en.wikipedia.org/wiki/YAML#Advanced_components)。对于重复节点，你可以通过引用复用它们。

## 作为 [Entity](https://en.wikipedia.org/wiki/Domain-driven_design#Building_blocks) 的重试/对冲策略

在上面的配置中，我们定义了四个重试/对冲策略，并在 `client` 中引用了它们。每种策略，除了必要的参数外，都有一个新的字段 `name`，用作实体的**唯一**标识。在上一章中，我们提到一些 option，如 `WithDynamicHedgingDelay`，它们无法在文件中配置，需要在代码中使用，这里的 `name` 就是在代码中使用这些 optioin 的关键。在 slime 中，我们提供了下面几种函数，来设置额外的 options。
```Go
func SetHedgingDynamicDelay(name string, dynamicDelay func() time.Duration) error
func SetHedgingNonFatalError(name string, nonFatalErr func(error) bool)
func SetRetryBackoff(name string, backoff func(attempt int) time.Duration) error
func SetRetryRetryableErr(name string, retryableErr func(error) bool) error
```

注意，对于重试策略的 `backoff`，你只能在 `exponential` 和 `linear` 之间二选一。如果你同时提供了两个，我们将以 `exponential` 为准。

## 与框架配置的统一

在插件配置 `plugins` 中，插件类型必须是 `slime`，插件名必须是 `default`。slime 会根据配置文件，将所有的重试/对冲策略加载到一个插件中，即 default。default 则提供了拦截器（[后面](#拦截器)介绍如何配置拦截器），自动对所有配置了重试/对冲的 service 或方法生效。

你可能发现了，`client` 键与客户端框架配置很像，除了它多了一些新的键，如 `retry_hedging`，`methods` 等。我们是刻意这么设计的，为了能够复用原始的框架配置。如果你打算在现有 client 中引入 slime，那么，你只需要在框架配置的 `client` 键下新增一些键值即可。

对冲是一种更加激进的重试策略。配置重试/对冲策略时，你只能在它们之间二选一：
```yaml
retry_hedging:
  retry: *retry1
  # hedging: *hedging1  # 选择了 retry 就不要再填 hedging 了
```
如果你即填了 retry，又填了 hedging，那么，我们会以 hedging 为准。
如果你这么填 `retry_hedging: {}`，那么该策略等同于没有配置重试/对冲。注意，这与 `retry_hedging:` 不同，前者是配置了键 `retry_hedging`，但它的内容是空的，后者相当于没有键 `retry_hedging`。

你可以为整个 service 指定一个重试/对冲策略，在 `service` 下添加 `retry_hedging` 键即可，也可以精细到具体某个方法，在 `method` 中添加 `callee`。
在配置文件中，service `trpc.app.server.Welcome` 使用了 `retry1` 作为重试策略。
`Hello` 使用重试策略 `retry2` 覆盖了 service 重试策略 `retry1`。
`Hi` 使用对冲策略 `hedging1` 覆盖了 service 重试策略 `retry1`。
`Greeter` 则使用**空策略**覆盖了 service 策略 `retry1`。
`Yo` 显式地继承了 service 的策略 `retry1`。
其他未显式配置的方法都默认继承了 service 的策略 `retry1`。
服务 `trpc.app.server.Greeting` 的所有方法都使用对冲策略 `hedging2`。

## 限流
在 slime 中，限流是以 service 为单位的。
slime 默认为每个 service 都开启限流功能，配置为 `max_tokens: 10` 和 `token_ratio: 0.1`。
你也可以像 service `trpc.app.server.Welcome` 一样，自定义 `max_tokens` 和 `token_ratio`。
如果你想关闭限流，需要这样配置：`retry_hedging_throttle: {}`。

## 拦截器
slime 插件在初始化时，会自动注册 slime 拦截器。
要使 slime 插件生效，你必须在 `filter` 中指定 `slime` 拦截器：
```yaml
client:
  filter: [slime]
  service:
    - # 你也可以将拦截器注册在服务内
      # filter: [slime]
```
slime 会产生多个子请求，请注意它与其他拦截器的次序。

## 跳过已访问过的节点
正如我们在 4.1 节中描述的，你也可以在配置中指定是否跳过已经发送过请求的节点。
`retry1` 和 `hedging1` 没有配置 `skip_visited_nodes`，它们对应第一种情形。`retry2` 显式地指定 `skip_visited_nodes` 为 `false`，它对应第三种情形。`hedging2` 显式地指定 `skip_visited_nodes` 为 `true`，它对应第二种情形。

请注意，该功能需要负载均衡器配合。如果负载均衡器没有实现对应能力，那么都会对应到情形三。

## 对某次请求关闭重试/对冲
在 v0.2.0 后，我们支持了一个新功能：用户可以通过创建一个新的 context 来关闭某次请求的重试/对冲。
该功能通常与 trpc-database 配合，让重试/对冲配置只对读请求（或者幂等请求）生效，而跳过写请求。比如，对于 trpc-database/redis：
```go
rc := redis.NewClientProxy(/*omitted args*/)
rsp, err := rc.Do(trpc.BackgroundContext(), "GET", key) // 默认配置了重试/对冲
_, err = rc.Do(slime.WithDisabled(trpc.BackgroundContext()), "SET", key, val) // 通过 context 关闭了本次 SET 调用的重试/对冲
```
注意，该功能只对 slime 生效，slime/retry 和 slime/hedging 并不提供该功能。

# 可视化
v0.2.0 后，slime 提供两种可视化能力，一个是条件日志，一个是监控打点。
## 条件日志
无论是对冲还是重试，它们都有一个名为 `WithConditionalLog` 的选项。[这](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/retry/retry.go#L210)是重试的，[这](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/hedging/hedging.go#L160)是对冲的，这两个（[retry](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/opts.go#L106)，[hedging](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/opts.go#L48)）是 slime 的。
条件日志需要两个参数，一个是 `log.Logger`
```go
type Logger interface {
	Println(string)
}
```
一个是条件函数 `func(stat view.Stat) bool`。

条件函数中的 `view.Stat` 提供了一个应用层请求执行过程的状态。你可以根据这些数据，决定是否输出重试/对冲日志。比如，下面的条件函数告诉 slime，只有当一共重试了三次，且前两次都没有回包，而第三次成功时，才输出日志：
```go
var condition = func(stat view.Stat) bool {
	attempts := stat.Attempts()
	return len(attempts) == 3 &&
		attempts[0].Inflight() &&
		attempts[1].Inflight() &&
		!attempts[2].Inflight() &&
		attempts[2].Error() == nil
}
```

`Logger` 只需要一个简单的 `Println(string)` 方法。你可以基于任何 log 库包装一个出来。比如，下面这个是基于控制台的 log：
```go
type ConsoleLog struct{}

func (l *ConsoleLog) Println(s string) {
	log.Println(s)
}
```
这是一个 slime 在控制台输出的日志：
![ 'image.png'](/.resources/user_guide/retry_hedging/logs.png)
有几点你需要特别关注：
* 一个应用层请求的所有 slime 日志对应 `log.Logger` 中的一次 `Println`，这在 slime 中称为 lazzy log，就像截图中第一行显式的那样。
* slime 的日志通过换行制表符等进行了格式化。
* 最后一条 slime 日志是对所有尝试的汇总。

更多条件日志的细节请参考 [slime/retry](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/retry/retry_test.go) 和 [slime/hedging](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/hedging/hedging_test.go) 的单元测试。

## 监控
与条件日志类似，重试/对冲的监控也是基于 [`view.Stat`](https://git.woa.com/trpc-go/trpc-filter/blob/slime/v0.2.0/slime/view/stat.go) 的。

slime 提供了四个监控项：应用层请求数、实际请求数、应用层耗时、实际耗时。

所有监控项都有三种标签：caller、callee、method。

对于应用层请求数与应用层耗时，它们具有以下额外标签：总尝试次数、最终错误的错误码、是否被限流、未完成的请求数（只有对冲才可能非零）、后端是否显式禁止重试/对冲。

对实际请求数与实际耗时，它们具有以下额外标签：错误码、是否未完成、后端是否显式禁止重试/对冲。

### m007 监控
引入依赖
```go
import "git.code.oa.com/trpc-go/trpc-filter/slime/view/metrics/m007"
```
对 retry，你需要：
```go
r, err := retry.New(3, []int{141}, retry.WithLinearBackoff(time.Millisecond*5), retry.WithEmitter(m007.NewEmitter()))
```
对 hedging，你需要：
```go
h, err := hedging.New(2, []int{141}, hedging.WithStaticHedgingDelay(time.Millisecond*5), hedging.WithEmitter(m007.NewEmitter()))
```
对 slime，你需要：
```go
err = slime.SetHedgingEmitter("hedging_name", m007.NewEmitter())
err = slime.SetRetryEmitter("retry_name", m007.NewEmitter())
```

为了适配 m007 维度功能，每个标签 kv 会以 `_` 拼接，组成 m007 的一个维度。具体请参考这个 [MR](https://git.woa.com/trpc-go/trpc-filter/merge_requests/114) 中的评论。

### prometheus
prometheus 的使用方式与 m007 类似。引入依赖：
```go
import prom "git.code.oa.com/trpc-go/trpc-filter/slime/view/metrics/prometheus"
```
使用 `prom.NewEmitter` 来初始化一个 Emitter。
prometheus 的使用方式可以参考[官方文档](https://prometheus.io/docs/guides/go-application/)。

### trpc tvar
// TODO

# 用户案例
https://mk.woa.com/note/6739?ADTAG=rb_tag_8205

# OWNER
如果你们有使用重试和对冲的计划，请线上联系我们（cooperyan/jessemjchen），我们需要业务的效果和反馈，以进一步优化。
## cooperyan