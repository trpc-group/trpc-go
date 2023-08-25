# 前言

首先，欢迎大家来阅读 tRPC-Go 架构设计文档，这是一个非常好的机会，能和大家分享一下 tRPC-Go 设计中的一些思考。有很多同学注意到 tRPC-Go 之后，就会想 tRPC-Go 有哪些创新之处，和外部的开源框架有哪些优势，我为什么要花大代价去学习一门新框架，等等。

本篇文章主要讲的是 tRPC-Go 特色部分的架构设计，tRPC 所有语言整体上都遵循一致的设计，相同部分可看架构概述。

也许 tRPC-Go 并不是业界的明星产品，但它应该是一个解决问题的不错的选择。tRPC 大家族提供了多语言版本的框架，并且在顶层设计上都遵循一致的架构设计，框架特性、周边生态建设也力求同步推进，对于满足公司团队不同技术栈的选择、对周边组件的支持力度、技术支持，都提供了一种还不错的保障。

"一支穿云箭，千军万马来相见"，有幸在框架治理中感受到了开源协同的力量。tRPC-Go 在大家的讨论中诞生，也希望在公司更大范围的讨论中继续壮大。

# 背景

为了让大家更好地了解 tRPC-Go 的架构设计，本文中将尽可能地覆盖必要的内容，本文档基于 tRPC-Go 框架 v0.3.6 编写。由于笔者精力有限，后续文档也可能会过时，也希望大家一起参与进来。

本文后续小节将按照如下方式进行组织：

- 首先，介绍下 tRPC-Go 的整体架构设计，方便大家先有个大概的认识；
- 然后，介绍下 tRPC-Go 的 server 工作流程，方便大家从全局把握 server 工作原理；
- 然后，介绍下 tRPC-Go 的 client 工作流程，方便大家从全局把握 client 工作原理；
- 然后，介绍下 tRPC-Go 对性能方面的优化，将一些可调优的优化选项告知大家；
- 然后，想和大家分享下某些部分的设计及可优化点，供后续持续优化、迭代；

这是 tRPC-Go 架构设计的第一篇文章，重点关注框架，后续会在模块设计的文档页中更细致的介绍模块与模块、框架之间的协作。

# 架构设计

## 整体形态

tRPC-Go 整体架构设计如下：
![overall](/.resources/developer_guide/architecture_design/overall_zh_CN.png)

tRPC-Go 框架主要包括这几个比较核心的模块：

- client：提供了一个并发安全的通用的 client 实现，主要负责服务发现、负载均衡、路由选择、熔断、编解码、自定义拦截器相关的操作，各部分均支持插件式扩展；
- server：提供了一个服务实现，支持多 service 启动、注册、取消注册、热重启、平滑退出；
- codec：提供了编解码相关的接口，允许框架扩展业务协议、序列化方式、数据压缩方式等；
- config：提供了配置读取相关的接口，支持读取本地配置文件、远程配置中心配置等，允许插件式扩展不同格式的配置文件、不同的配置中心，支持 reload、watch 配置更新；
- log：提供了通用的日志接口、zaplog 实现，允许通过插件的方式来扩展日志实现，允许日志输出到多个目的地；
- naming：提供了名字服务节点注册 registry、服务发现 selector、负载均衡 loadbalance、熔断 circuitbreaker 等，本质上是一个基于名字服务的负载均衡实现；
- pool：提供了连接池实现，基于栈的方式来管理空闲连接，支持定期检查连接状态、清理连接；
- tracing：提供了分布式跟踪能力，当前是基于 filter 来实现的，并未在主框架中实现；
- filter：提供了自定义拦截器的定义，允许通过扩展 filter 的方式来丰富处理能力，如 tracing、recovery、模调、logreplay 等等；
- transport：提供了传输层相关的定义及默认实现，支持 tcp、udp 传输模式；
- metrics：提供了监控上报能力，支持常见的单维上报，如 counter、gauge 等，也支持多维上报，允许通过扩展 Sink 接口实现对接不同的监控平台；
- trpc：提供了默认的 trpc 协议、框架配置、框架版本管理等相关信息。

## 交互流程

tRPC-Go 整体交互流程如下：
![interaction_process](/.resources/developer_guide/architecture_design/interaction_process_zh_CN.png)

# 工作原理

## 服务端

### 启动

Server 启动过程，大致包括以下流程：

1. `trpc.NewServer()`初始化服务实例；
2. 读取框架配置文件 (-conf 指定)，并反序列化到`trpc.Config`，这里的配置包含了 server、service、client 以及众插件的配置信息；
3. 遍历配置文件中的 service 列表及各种插件配置完成初始化逻辑；
    1. service 启动监听，完成服务注册，任意一个失败则全部取消注册并退出；
    2. 各插件完成初始化，任意一个失败，则进程 panic 退出；
    3. 监听信号 SIGUSR2，收到则执行热重启逻辑；
    4. 监听 SIGINT 等信号，收到进程正常退出；
4. `server.Register(pb.ServiceDesc, serviceImpl)`注册 sevice，这里其实是注册 rpc 方法名及处理函数的映射关系；
5. 服务此时就已经正常启动了，后续等待 client 建立连接请求。

### 请求处理

1. server transport 调用 Accept 等待 client 建立连接；
2. client 发起建立连接请求，server transport Accept 返回一个连接 tcpconn；
3. server transport 根据当前的工作模式（是否 AsyncMod），来决定是对相同连接上的请求串行处理，还是并发处理；
    1. 如果是串行处理，那么一条连接一个 goroutine 来处理，顺序处理连接上到达的请求，这种适用于 client 端非连接复用模式情景；
    2. 如果是并发处理，那么一条连接上的请求每收到一个请求就起一个 goroutine 去处理，当前这种方式虽然实现了并发处理，但是有可能导致 goroutine 爆炸；
4. 开始收包的逻辑，server transport 根据编解码协议、压缩方式、序列化方式不停地读取请求，并将其封装为一个 msg 交给上层处理；
5. 拿到 msg 之后，根据 msg 内部的 rpc 名称，找到对应的注册的处理函数，调用对应的处理函数；
6. 调用对应的处理函数之前，其实还要过一个 filterchain，filterchain 执行到最后就是我们注册的 rpc 的处理函数；
7. 将处理结果进行序列化、压缩、编解码，然后回包给 client。

注意，在从 tcpconn 读取请求时，有可能出现几种情况：

- 正常读取到请求，ok
- 读取到 eof，表明对端连接关闭，close 掉；
- 读取超时，并且超过设定的连接空闲时间，close 掉；
- 读取到数据，但是解包失败，close 掉。

### 退出

服务退出阶段，根据退出情景的不同也可以细化下。

#### 正常退出

服务受到信号 SIGINT 等执行正常退出逻辑：

1. 调用各个 service 的 close 方法，关闭 service 逻辑；
2. 取消各个 service 在名字服务中的注册；
3. 调用各个插件的 close 方法；
4. 退出。

#### 异常退出

1. 如业务代码中起 goroutine，内部 panic，未正常 recover 时则服务 panic；
2. 服务中引入了 serverside 的 filter：recovery，在框架起的业务处理 goroutine 中出现 panic，recovery filter 负责捕获，不异常退出。

#### 热重启

1. 收到 SIGUSR2 信号后，执行热重启逻辑；
2. 父进程首先收集当前已经打开的 listeners，包括 tcplistener、udp packetconn，然后获取其 fd；
3. 父进程 forkexec 创建子进程，创建时通过 ProcAttr 传递 fd 与子进程共享 stdin\stdout\stderr 以及各个 tcplistener fd、packetconn fd；并通过环境变量通知子进程热重启；
4. 此时，子进程启动，进程启动过程中也会走 server 启动的流程，实际上启动监听时会检查环境变量来发现是否热重启模式，如果是则通过传递的 fd 来重建 listener，否则通过 net.Listen 或者 reuseport.Listen 来监听；
5. forkexec 返回后，父进程继续执行后续退出流程（当前已经建立连接上的请求，未等到起处理完成回包后再退出）
6. 父进程执行自定义事务清理逻辑，类似 AtExit 注册的钩子函数（当前未实现）
7. 父进程退出，子进程代替父进程处理。

## 客户端

1. 发送请求时，先组装各种调用参数；
2. 执行 client filter 前置逻辑；
3. 对发送数据进行序列化、压缩、编码逻辑；
4. 服务发现找到被调服务名对应的一组 ip:port 列表；
5. 通过负载均衡算法，找到合适的一个 ip:port 准备发起请求；
6. 通过熔断检查是否允许发起当前次请求（避免因重试给后端造成压力引发雪崩）
7. 一切都 ok 后，好，准备建立到 ip:port 的连接，这个时候会先检查连接池中是否存在对应的空闲连接，没有就要 net.Dial 创建
8. 获取到连接之后，开始发送数据，并等待接收（如果是连接复用模式，可能会在同一条连接上并发发送多个请求，请求响应通过 seqno 关联，当前未实现）；
9. 接收到数据，解码、解压缩、反序列化逻辑，递交给上层处理；
10. 执行 client filter 后置逻辑；

需要注意的是，client 这里也涉及到一个 filterchain 逻辑，可以扩展一系列功能，比如 rpc 的时候上报 tracing 数据、模调数据等。
client 内部使用的连接池，其实是 client transport 中引用的，client 是一个通用的 client，区分 tcp、udp、连接池是 client transport 来管理的，连接池也会定期检查连接可用性。

更多内容在后续模块文档中介绍。

# 总结

这里简单总结了 tRPC-Go 的整体架构设计，以及 client、server 的大致工作流程，中间穿插着提及了相关模块的的功能，这部分内容的更多信息，我们在后续模块设计相关的文档中进行更详细的介绍。

