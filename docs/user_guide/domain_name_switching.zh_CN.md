# trpc-go trpc.tech v2 迁移指南 (用户版)

(2023.8.21) 注：如果没有必要，不建议域名切换，因为处理新旧共存问题会有相当大的负担（并且 v2 主库及各插件的更新有延迟，并且 v2 的各插件存在潜在的共存未做好的风险）（一线开发踩出来的血路你愿意再走一遍？），可以在根据 [环境搭建](https://iwiki.woa.com/pages/viewpage.action?pageId=99485252) 的配置代理一节来配置 goproxy，从而继续使用 code.oa 的域名（即使切换，大量存量 code.oa 代码也离不开 goproxy 了，所以现在不管怎么样都实质离不开 goproxy 了，所以暂停切换 v2），官方回复见：【新业务使用 trpc-go 框架，是否继续使用 trpc.tech v2，希望有个官方回答？ 】<http://mk.woa.com/q/291729/answer/116248>

全文分为**用户版**和**主库及插件版**，适用于不同的读者，trpc-go 的用户仅需重点关注**用户版**的内容。

## 前言

trpc-go 目前已经进行了 trpc.tech 的试切，并发布了 beta 版本（<https://git.woa.com/trpc-go/trpc-go/tree/v2.0.0-beta）（还有一个> v2.0.0-alpha 的 tag 是废弃掉的，不要使用，从功能上来讲，v2.0.0-beta 和 v0.10.0 完全相同，后续会定期将 v0.x.x 同步到 v2.x.x 上），这半篇是用户使用这个新的仓库的指南

trpc-go 在 main branch 上 go.mod 的 module name 变动如下：

- 新的域名为 trpc.tech
- 新的 group name 仍为 trpc-go
- 所带的版本后缀为 v2

注：只要 module name 发生了变更，不论是域名还是版本后缀，其本质上都相当于是两个仓库，本文所提到的两仓库并存的注意事项统统适用。

## 创建一个新服务

对于一个现存的 helloworld.proto 文件，可以通过一下指令来创建该 pb 文件对应的 trpc.tech v2 项目（trpc-go-cmdline 工具版本需 >= v0.7.8）：

```shell
trpc create -p helloworld.proto -o out --domain=trpc.tech --versionsuffix=v2
```

和以往的命令相比，多了 `--domain=trpc.tech --versionsuffix=v2` 这两个参数。

其中可以生成 trpc.tech v2 版本的桩代码以及服务端的示例代码，其中服务端的示例代码中会自动包含相关插件的 trpc.tech v2 版本，main.go 文件中大致的效果如下：

```go
package main

import (
    // 一些插件
    // ..
    pb "git.woa.com/trpc-go/multi-trpc-go-module-name/case1/stubs/server1"
    trpc "trpc.tech/trpc-go/trpc-go/v2"
    "trpc.tech/trpc-go/trpc-go/v2/log"
)

func main() {
    s := trpc.NewServer()
    pb.RegisterHelloTrpcGoService(s, &helloTrpcGoImpl{})
    if err := s.Serve(); err != nil {
     log.Fatal(err)
    }
}
```

## 旧服务调用新服务

（一个实际可运行的测试示例见 <https://git.woa.com/wineguo/multi-trpc-go-module-name/tree/master/case3）>
旧服务在使用新服务的桩代码调用新服务时需要注意两点：

1. 需要在 trpc.tech v2 新版 trpc-go 中再次进行配置的读取以及插件的加载
2. 再次加载的配置中所需要用到的插件需要匿名 import 他们的 trpc.tech v2 版本

示例代码如下：

```golang
package main

import (
    _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"
    // 2. 配置中对应使用的插件需要匿名 import trpc.tech v2 版本, 此处以 debuglog 插件作为示例
    _ "trpc.tech/trpc-go/trpc-filter/debuglog/v2"
    // ... 

    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/log"
    pb "git.woa.com/trpc-go/multi-trpc-go-module-name/case3/stubs/server1"
    pb3 "git.woa.com/trpc-go/multi-trpc-go-module-name/case3/stubs/server3v2"
    trpcv2 "trpc.tech/trpc-go/trpc-go/v2"
)

func main() {
    // 当前服务是旧服务(非 trpc.tech v2)
    s := trpc.NewServer()
    // 1. 在 trpc-go trpc.tech v2 中再加载一次配置
    // 使用 trpc.ServerConfigPath 参数时, 表示共享当前服务的配置, 这也是在单一版本时的情况, 客户端和服务端共享一个配置文件, 假如希望这个客户端使用不同的配置文件, 可以把这里的参数改为期望的路径
    cfg, err := trpcv2.LoadConfig(trpc.ServerConfigPath)
    if err != nil {
        panic("load config fail: " + err.Error())
    }
    trpcv2.SetGlobalConfig(cfg)
    if err := trpcv2.Setup(cfg); err != nil { // woa v2 中的插件加载
        panic("setup plugin fail: " + err.Error())
    }

    // pb3 实际上是一个 trpc.tech v2 的新版服务提供的桩代码
    // 当前服务中会使用这个客户端来对其进行调用
    proxy3 := pb3.NewHelloTrpcGoClientProxy()
    pb.RegisterHelloTrpcGoService(s, &helloTrpcGoImpl{
        proxy3: proxy3,
    })
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

## 新服务调用旧服务

（一个实际可运行的测试示例见 <https://git.woa.com/wineguo/multi-trpc-go-module-name/tree/master/case5）>

与“旧服务调用新服务”部分类似，也是注意两点：

1. 需要在旧版 trpc-go 中再次进行配置的读取以及插件的加载
2. 再次加载的配置中所需要用到的插件需要匿名 import 他们的旧版本

```go
package main

import (
    trpcv1 "git.code.oa.com/trpc-go/trpc-go"
    pb3 "git.woa.com/trpc-go/multi-trpc-go-module-name/case1/stubs/server3"
    pb "git.woa.com/trpc-go/multi-trpc-go-module-name/case5/stubs/server1v2"
    trpc "trpc.tech/trpc-go/trpc-go/v2"
    "trpc.tech/trpc-go/trpc-go/v2/log"

    // 插件需要额外匿名 import 旧版的 (这里以 debuglog 为例，其他实际用到的也都需要额外 import)
    _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"
    _ "trpc.tech/trpc-go/trpc-filter/debuglog/v2"
)

func main() {
    // 当前服务本身是一个 trpc.tech v2 的新版服务
    s := trpc.NewServer()

    // 1. 需要在 trpc-go 旧版中再加载一次配置
    // 使用 trpc.ServerConfigPath 参数时，表示共享当前服务的配置，这也是在单一版本时的情况，客户端和服务端共享一个配置文件，假如希望这个客户端使用不同的配置文件，可以把这里的参数改为期望的路径
    cfg, err := trpcv1.LoadConfig(trpc.ServerConfigPath)
    if err != nil {
        panic("load config fail: " + err.Error())
    }
    trpcv1.SetGlobalConfig(cfg)
    if err := trpcv1.Setup(cfg); err != nil { // 插件在旧版中的加载
        panic("setup plugin fail: " + err.Error())
    }

    proxy3 := pb3.NewHelloTrpcGoClientProxy()
    pb.RegisterHelloTrpcGoService(s, &helloTrpcGoImpl{
        proxy3: proxy3,
    })
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

## rick 平台操作指引

rick 平台（<https://trpc.rick.woa.com/rick/pb/list）也提供了相应选项来生成带有> trpc.tech v2 trpc-go 的桩代码

其中涉及到 validate/restful/go_tag/swagger/alias 等插件的，目前还保持原来的逻辑以维持兼容性，即：相同的 `import` 语句对应的仍然为相同的 go package（假如对应多份的话会引起兼容性问题），因此如果需要使这些插件使用 v2 版本的 go package，请参考以下规则进行替换：

![proto_dependency_switching](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/domain_name_switching/proto_dependency_switching.png)

相关文档见：[tRPC-Go 代码生成插件 proto 依赖切换](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMmQcAOcIkSOWgvNFkY3?scode=AJEAIQdfAAoLH9SMkiAGkAxgZOAFM) （相关码客问题见：<http://mk.woa.com/q/287559/answer/108332）>

修改 proto 文件后，可以点击桩代码（或服务）的更新按钮，选择生成选项进行相应更新

![rick-generate-pb-1](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/domain_name_switching/rick-generate-pb-1.png)

![rick-generate-pb-2](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/domain_name_switching/rick-generate-pb-2.png)

点击选项三即可生成带有 trpc.tech v2 trpc-go 依赖的桩代码

v2 的切换目前处于测试阶段，有问题欢迎反馈

## 总结

对于用户来讲，新旧 trpc-go 版本共存时只需要考虑两大点：

1. 新旧互调时配置的再次加载，保证共存的新旧框架中都有已加载的配置
2. 新旧框架中的插件需要各自 import 对应的版本，新框架读取的配置中所用到的插件需要匿名 import 对应的  trpc.tech v2 版本，旧框架读取的配置中所用到的插件需要匿名 import 原先的版本

此外，假如一份 xx.proto 文件同时拥有新旧版本的桩代码，那么使用旧的桩代码去调用一个新版的服务（或者反过来）都是可行的，在数据包层级，新旧版本可以互相兼容。

更多测试见：<https://git.woa.com/wineguo/multi-trpc-go-module-name>

# trpc-go trpc.tech v2 迁移指南 (主库及插件版)

## 前言

本文介绍了 trpc-go 本身及相关生态（插件、拦截器等）切换 trpc.tech v2 的方法及注意事项。

## trpc-go 主库切换

原主分支为 master，为保险起见，创建一个 main 分支，在测试时期两分支并存，发版时则从 master 同步到 main，main 验证没问题时，主分支直接切换为 main

- `restful/errors/errors.proto` 文件里面的 package errors 更名为 package errors.v2，然后重新生成桩代码

- trpc-go 根目录下的 trpc.pb.go 对应的 trpc.proto 文件找到，package 更名为 trpc.v2，然后重新生成桩代码

- module name 改为 `"trpc.tech/trpc-go/trpc-go/v2"`

- 内部所有的 import 路径从 `"git.code.oa.com/trpc-go/trpc-go"` 改为 `"trpc.tech/trpc-go/trpc-go/v2"`

- 打上 tag `v2.0.x`

目前已经迁移完毕，见 <https://git.woa.com/trpc-go/trpc-go/tree/v2.0.0-beta>

## 主要插件及拦截器切换

核心是以下两点：

1. trpc-go group 下面的所有 repo 的 go.mod 中的 module 的 domain 必须切为 trpc.tech 并添加 v2 版本后缀，打上 v2.x.x tag
2. 这些 repo 所有的 code.oa 间接依赖需要迁为 git.woa.com v2（注意：不在 trpc-go group 下的不需要改为 trpc.tech，只需要改为 woa）

以 trpc-database/ckv 这个 MR（<https://git.woa.com/trpc-go/trpc-database/merge_requests/977）为例：>

操作分为三步：

1. 自身 module name 修改为 trpc.tech v2
2. 所有非 trpc-go group 下的 code.oa 依赖需要切为 woa v2（依赖进行递归切换）
3. 所有 trpc-go group 下的依赖切为对应的 trpc.tech v2

![modify go mod](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/domain_name_switching/modify_go_mod.png)

如果依赖的 code.oa 来自桩代码，那么需要用 trpc-go-cmdline 工具执行 --domain=trpc.tech --versionsuffix=v2 来进行桩代码的重新生成，如果桩代码在 rick 平台上，则 可以使用 rick 平台的生成选项三来生成依赖 trpc.tech v2 trpc-go 的桩代码（注意假如原始桩代码依赖的是 v0.8.5 之前的 trpc-go，那么这样的切换还面临 0.9.0 带来的 rsp 在出参的不兼容变动）

![rick-generate-pb-3.png](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/domain_name_switching/rick-generate-pb-3.png)

同时需要注意 repo 假如有某些全局注册操作，要考虑新旧之间是否能够正常共存（比如引的 proto 需不需要改名，package name 需不需要改名等）

然后在 main 上面打 v2.x.x 的 tag（刚开始可以打一个 v2.0.0-beta 表示测试版）

最后期望的效果是：切换完之后的 trpc.tech v2 的仓库，他的所有的间接依赖（递归）都不包含任何 code.oa 域名的 module，如果包含，那么就切换的不够彻底

在同一个仓库下改 module name 后为什么要打 v2.x.x 的 tag 见这篇文章方案二中的讨论：<https://km.woa.com/articles/show/560552>

目前 trpc-go group 下面的大部分 repo 已经迁移完成并打 v2.0.0-beta 的 tag，剩下的是存在一些外部依赖，需要业务方协助共同推进改造

更多的改造示例可以参考 trpc-filter, trpc-codec, trpc-database 等 repo 相关的 MR

- <https://git.woa.com/trpc-go/trpc-filter/merge_requests?state=merged&sort=created_desc&page=1&search=>
- <https://git.woa.com/trpc-go/trpc-codec/merge_requests?state=merged&sort=created_desc&page=1&search=>
- <https://git.woa.com/trpc-go/trpc-database/merge_requests?state=merged&sort=created_desc&page=1&search=>

## 总结

主库及插件的切换需要考虑依赖顺序，主库先做，然后各插件根据依赖顺序依次进行。公共的改造内容分为两点：

1. module name 改为 woa 域名，并加 v2 后缀
2. 打上 v2.x.x 的 tag

同时在迁移后需要测试新旧版本是否可以共存，并进行共存所需的其他改造，比如重命名冲突的 flag、重命名冲突的 pb 等

相关 km 文章：

- [tRPC-Go code.oa 域名下线后的切换方案](https://km.woa.com/articles/show/560552)
- [tRPC-Go trpc.tech v2 迁移指南](https://km.woa.com/articles/show/562863)
