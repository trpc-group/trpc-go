__注：__ 本文请配合 [tRPC-Go 服务路由（tRPC 知识库）- set 路由](https://iwiki.woa.com/p/4008319150#set-%E8%B7%AF%E7%94%B1) 以及该链接中的附录 [naming-polaris readme](https://git.woa.com/trpc-go/trpc-naming-polaris#clientwithservicename-%E5%AF%BB%E5%9D%80%E4%B8%8E-clientwithtarget-%E5%AF%BB%E5%9D%80%E7%9A%84%E5%8C%BA%E5%88%AB%E4%BB%A5%E5%8F%8A-enable_servicerouter-%E7%9A%84%E8%AF%AD%E4%B9%89) 食用。

# 1 前言

Set 部署是指根据业务功能特征对服务以 Set 为单元进行规范化、标准化和规模化部署，从而有效防止故障扩散，实现海量服务的高效运营，实现高效的容量规划。

优点如下：

- 服务名统一，服务配置统一管理。
- 按照小组为单位，容量容易控制。
- 各个小组之间没有调用关系，不干扰。

示例：

当服务 100w 在线的时候，一个服务单节点可以提供服务；
当服务到 500w 在线的时候，一个服务多个节点可以提供服务；
当服务 5000w 在线的时候，就要考虑进行拆分，否则一个服务有问题，会影响所有用户的访问。
这个时候需要考虑服务拆分。服务拆分考虑可以考虑按服务名拆分，一个服务拆分成多个服务，但这会带来很多问题，
比如服务或者应用的名称和原服务不一致，配置文件、发布服务需要单独对待，不能统一管理
而按 set 划分就可以很好的解决这个问题，同个服务通过划分 set 来提供规模化的部署，
单个 set 内的故障不会影响到其他 set，实现故障隔离的同时，简化运维的成本。

![why-set-routing](../../../.resources/practice/pcg/set_routing/why-set-routing.png)

# 2 原理

## 2.1 set 模型

Set 定义最终定为三级结构

命名规范：

Set 名：定义一个大的 Set 名称，可以以业务名称来定义（mmt，yyb，ws）。
Set 地区：可以按照地区来划分，如 hn，hb（华南，华北），也可以以城市来分，如 sh，sz（上海，深圳）等。
Set 组名：实际可以重复的组单元的名称，一般是 0，1，2，3，4，5，…，也可以为`*`，`*`代表通配组。

![set-model-figure](../../../.resources/practice/pcg/set_routing/set-model-figure.png)

地区和组名：

| SET 名 | SET 地区 | SET 组名 | 服务列表 |
|:-------:|:-------:|:-------:|:-------:|
| mtt | SZ | 1 | A, B, C, F |
| mtt | SZ | 2 | A, B, C |
| mtt | SZ | `*`(通配组) | C, D, E, F |
| mtt | SH | 1 | A, B, C |
| mtt | SH | 2 | A, B, C |

SET 分组调用总体原则：

1. 主被调双方都要启用 SET 分组，并且 SET 名（指的是第一段，不包含地区以及组名）要一致。

2. SET 内有被调的（不管节点状态）, 只能调用本 SET 内的，如果没有被调（如果 set 内节点都异常，则认为没有这个 set），则只能调用本地区的公共区域的，公共区域还没有的话则返回寻址失败。

   - `1A` 调用 `1C`，但 `1A` 不能调用 `*C`
   - `1A` 调用 `1F`，但 `1A` 不能调用 `*F`
   - `2A` 调用 `*F`，但 `1A` 不能调用 `*F`
   - `1C`，`2C`，`*C` 均可调用 `*E`

3. 通配组通配组服务可调用 SET 内和通配组的任何服务，如 `*D` 调用 `*C`+`1C`+`2C`

4. 对于不同 SET 下的服务互调，则采用就近原则调用。有两个 SET:MTT 和 SET:XXSQQ，由于 SET_NAME 不一样，则认为没有启动 SET 分组，采用默认的就近原则，因此可实现两者间的互通

5. 如果不满足 1，则不启用 set 规则，由就近原则进行路由

北极星 SDK 可以通过插件来做路由的支持

北极星 SDK，目前支持规则路由和就近原则，需要增加 set 分组插件

北极星新增功能

1. 北极星 SDK 需要支持动态判断是否启用某个路由插件比如，路由链是：规则路由-set 路由 - 就近路由，那么 set 路由逻辑中，假如 set 路由成功，则设置就近路由的 enable 为 false，后续就跳过就近路由

2. 北极星 SDK 需要支持插件可以对服务实例的元数据按各个维度缓存聚合，支持 Set 信息快速获取

3. 北极星 set 相关的数据方面：

    迁移到北极星的 set 信息字段定义（暂定）【存放在服务实例节点的 metadata 信息】:
    internal-enable-set //是否开启 set，目前定为 Y/N
    internal-set-name //set 全名，三段式，.（点号）隔开，全小写和数字

## 2.2 详细调用规则

| 主调      | 被调      | 就近   | 逻辑                                                                                                                                                                      |
|---------|---------|------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 启用 set  | 启用 set  | 停用就近 | 以第一段 set 名为匹配，只要被调节点有第一段 set 名字一样的，认为被调启用了 set。如果主调 set 分组不为`*`（星号），则优先匹配本 set 内（三段匹配），匹配不到，则匹配本地区的（通配符、`*`），再匹配不到，则返回空。如果主调的 set 分组为`*`，则按两段进行匹配。只要启用了 set，则就近路由停用。 |
| 启用 set  | 不启用 set | 启用就近 | 以第一段 set 名为匹配，如果第一段 set 名字不一样，也认为没有启用 set，这个时候返回按就近原则匹配                                                                                                                 |
| 不启用 set | 启用 set  | 启用就近 | 不按 set 逻辑调用，按就近原则调用                                                                                                                                                     |
| 不启用 set | 不启用 set | 启用就近 | 不按 set 逻辑调用，按就近原则调用                                                                                                                                                     |

# 4 使用示例

## 4.1 123 平台服务端配置

123 管理平台服务端 Set 启用

在 123 管理平台，服务详情页面里添加或者修改容器配额

![ 123-config-set-overview](../../../.resources/practice/pcg/set_routing/123-config-set-overview.png)

选择增加配额，在这里填入 set 信息

![123-config-set-quota](../../../.resources/practice/pcg/set_routing/123-config-set-quota.png)

## 4.2 客户端代码调用

如果在 123 管理平台部署的服务，且在页面上设置了 set，那客户端无需任何操作，即可启用 set，调用的时候会启用 set。
相当于使用了 `WithCallerSetName`，这个 option 可以在独立客户端，或者页面没有设置 set 的时候启用。

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris" 
    // 注意一定要使用 naming-polaris 插件，不要注册自己的 selector 或者其他 selector，请检查是否 import 了其他 selector（CL5 等）,
    // 或者 git.code.oa.com/trpc-go/trpc-naming-polaris/selector 等也不要
)

node := & registry.Node{} // 用于 debug，可去掉
opts := []client.Option{
    // 注意千万不要使用 client.WithDisableServiceRouter 
    client.WithNamespace("Development"),
    client.WithCallerSetName("a.b.c")
    // 注意不要用 WithTarget 的方式，使用 WithServiceName
    client.WithServiceName("trpc.settestapp.settestserver.Greeter"),
    client.WithSelectorNode(node), // 用于 debug，可去掉
}
proxy := pb.NewGreeterClientProxy(opts...)
```

如果要强制调用服务端的 set，请使用 `WithCalleeSetName`，这个时候会强制取这个 set 的服务端节点，获取不到则返回为空。
和 `WithCallerSetName` 不一样的是，这个取不到对应的 Set 不会走 set 规则，也不会再走就近原则，直接返回空。

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris" 
    // 注意一定要使用 naming-polaris 插件，不要注册自己的 selector 或者其他 selector，请检查是否 import 了其他 selector（CL5 等）
)

node := & registry.Node{} // 用于 debug，可去掉
opts := []client.Option{
    // 注意千万不要使用 client.WithDisableServiceRouter 
    client.WithNamespace("Development"),
    client.WithCalleeSetName("a.b.c")
    // 注意不要用 WithTarget 的方式，使用 WithServiceName
    client.WithServiceName("trpc.settestapp.settestserver.Greeter"),
    client.WithSelectorNode(node), // 用于 debug，可去掉
}
proxy := pb.NewGreeterClientProxy(opts...)
```

# 5 FAQ

## 5.1 selector instance empty

请检查 set 的规则，对应的服务端 set 启用了 set，但根据 set 规则没有相应的节点，或者没有存活的节点。

## 5.2 route set division with set group rule  not match, source set name is xxx, not instances found in this set group,please check

检查下是否使用了 `WithCalleeSetName` 且被调方没有对应的 set。

## 5.3 分 set 部署，但是发生跨 set 调用问题

- 注意检查是否使用了 naming-polaris 插件，不要注册自己的 selector 或者其他 selector，请检查是否 import 了其他 selector（CL5 等）, 或者 `git.code.oa.com/trpc-go/trpc-naming-polaris/selector` 等也不要。
- 注意千万不要使用 `client.WithDisableServiceRouter`。
- 注意不要用 `WithTarget` 的方式，使用 `WithServiceName`，注意检查配置文件 client 的 service 下面是不是配置了 target。
- 检查调用的 set 名是否写对。

## 5.4 我是纯客户端，我想按 set 调用有什么办法吗？

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)

func main() {
    LoadConfig()
}

// 加载 ./trpc_go.yaml 主要是为了让 trpc-naming-polaris 插件启动成功
func LoadConfig() {
    cfg, err := trpc.LoadConfig("./trpc_go.yaml")
    if err != nil {
        panic("parse config fail: " + err.Error())
    }
    // 保存到全局配置里面，方便其他插件获取配置数据
    trpc.SetGlobalConfig(cfg)

    // 加载插件
    err = trpc.Setup(cfg)
    if err != nil {
        panic("setup plugin fail: " + err.Error())
    }
}
```

trpc_go.yaml 必须包含以下配置：

```yaml
plugins:
  selector:
    polaris:
      # address_list: 9.141.66.8:8081,9.141.66.121:8081,9.141.66.27:8081,9.141.66.125:8081,9.136.124.80:8081,9.136.121.211:8081,9.136.124.240:8081,9.136.125.12:8081,9.136.124.229:8081,9.141.66.84:8081 # 名字服务远程地址列表
      protocol: grpc # 北极星交互协议支持 http，grpc，trpc
        discovery:
          refresh_interval: 10000 # 北极星服务发现刷新间隔，123 默认 10000，即 10s
```

## 5.5 启用了 set，能否在同一个 set 内再启用就近原则？

不能，set 和就近属于互斥，且 set 的第二段本来就为地区信息（area），可以将地区信息纳入到 set 信息中，比如 mtt.sz.1 ,mtt.sz.2, mtt.sh.1, mtt.sh.2

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
