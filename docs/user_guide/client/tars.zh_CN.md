## 1 背景

方便业务方在 trpc-go 服务中调用 taf 服务。

## 2 原理

trpc-go 服务调用 taf 服务的原理见 [官方文档](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars).

## 3 实现

* 注意，如果之前安装过 trpc4tars，请升级最新版本，之前迁移过 woa 域名，请更新到最新版本，否则可能报错

1. 编译桩代码生成工具 trpc4tars

```shell
go get git.code.oa.com/trpc-go/trpc-codec/tars && go install git.code.oa.com/trpc-go/trpc-codec/tars/tools/trpc4tars
```

2. 将依赖的 jce 文件复制到服务目录中去，生成桩代码

```shell
# 假设服务 module 为 git.woa.com/app/server
trpc4tars -module="git.woa.com/app/server" *.jce
# 桩代码默认生成在服务目录下的 tars-protocol 目录中
ls tars-protocol
```

3. 具体调用请参考 [官方示例](https://git.woa.com/trpc-go/trpc-codec/blob/master/tars/examples)

## 4 示例

* 代码示例

```go
import (
    "context"
    "fmt"

    "git.code.oa.com/trpc-go/trpc-codec/tars/model"
    "git.code.oa.com/trpc-go/trpc-codec/tars/examples/TafTestServer/tars-protocol/NFA"
    "git.code.oa.com/trpc-go/trpc-codec/tars/examples/TafTestServer/tars-protocol/comm"
    "git.code.oa.com/trpc-go/trpc-go/client"

    pselector "git.code.oa.com/trpc-go/trpc-naming-polaris/selector"
)

func init() {
    pselector.RegisterDefault()
}

func main() {
    obj := "NFA.TafTestServer.TafTestObj"
    prx := NFA.NewTafTestProxy(obj)

    var a int32 = 8
    var b int32 = 8
    var result int32
    ctx := context.Background()

    ret, err :=prx.Add(ctx, a, b, &result,
        //单纯 client 需要指定 target，因为北极星 Discover 没有注册，在 trpc 服务中可以不指定 target，默认会用北极星 Discover
        client.WithTarget("polaris://"+obj),
        // Development-开发环境，由本身服务在 123 平台所属特性环境（特性环境需要是继承之 sumeru-213 环境或者 sumeru-147 环境）决定调用 147 还是 213
        // Production-正式环境

        client.WithNamespace("Development"),
    )
    if err != nil {
        fmt.Printf("call Add(polaris) fail, err: %v, ret: %d\n", err, ret)
    } else {
        fmt.Printf("call Add(polaris) ok, ret=%d, A=%d, B=%d, result=%d\n", ret, a, b, result)
        fmt.Printf("Add|outCtxInfo: %+v\n", inCtxInfo)
    }
}
```

## 5 添加 mm 监控插件

trpc 服务在调用 tars 服务时需要在 client 端加载 mm 插件才能在 [mm 监控](http://taf.wsd.com/) 上查看到调用数据，加载方法是：

* `import` 插件包：

```go
import _ "git.code.oa.com/trpc-go/trpc-filter/mm"
```

要保证 `go.mod` 中引用的是最新版的 mm 插件，并保证最后项目中 `trpc-naming-polaris` 的版本大于等于 `v0.3.0`

* 配置 `trpc_go.yaml`，分别在 client 以及 plugins 部分添加 mm 插件，最小配置如下：

```go
client:
  filter:
    - mm
plugins:
  metrics:
    mm:
```

更多细节见 [mm 插件 README](https://git.woa.com/trpc-go/trpc-filter/tree/master/mm)

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
