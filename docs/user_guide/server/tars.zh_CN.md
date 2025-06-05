## 1 背景

Taf（开源版本叫 tars）是腾讯从 2008 年到今天一直在使用的后台逻辑层的统一应用框架，目前支持 C++/Java/golang/php/nodejs 等多种语言。该框架为用户提供了涉及到开发、运维、以及测试的一整套解决方案，帮助一个产品或者服务快速开发、部署、测试、上线。它集可扩展协议编解码、高性能 RPC 通信框架、名字路由与发现、发布监控、日志统计、配置管理等于一体，通过它可以快速用微服务的方式构建自己的稳定可靠的分布式应用，并实现完整有效的服务治理。
该框架在腾讯内部，各大核心业务都在使用，颇受欢迎，基于该框架部署运行的服务节点规模达到上万个。
基于公司统一 rpc 框架的战略，目前 tars 框架已经处于维护状态不再开发新功能，已有的存量 tars 服务很多需要往 trpc 框架下迁移。
本文介绍原有 tars 服务，如何借助 trpc-codec/tars 插件在不改变通信协议的情况下往 trpc-go 框架下迁移的方法。

## 2 原理

trpc-codec/tars 插件提供 tars 协议服务，主要通过以下手段：

1. 实现 trpc-go 框架抽象的 codec 接口，用于支持 tars 协议编解码；
2. 实现 trpc4tars 工具，用于根据服务 jce 协议文件生成桩代码（提供结构体声明，请求响应的编解码实现，接口路由的实现）；
   trpc-go 服务通过引入该插件，既可支持作为主调调用 tars 协议服务，也可作为被调提供 tars 协议服务。

## 3 实现

前面原理部分已经提到，tars 插件主要解决了 tars 协议的编解码实现以及 jce 协议代码生成，下面这部分介绍具体如何实现：

### 3.1 tars 协议编解码

实现 [codec.Framer](https://git.woa.com/trpc-go/trpc-go/blob/master/codec/framer_builder.go) 接口，并调用 transport.RegisterFramerBuilder 将 tars 协议的数据帧构造器注册到 trpc-go 框架中，以支持 tars 协议报文的识别
实现 [codec.Codec](https://git.woa.com/trpc-go/trpc-go/blob/master/codec/codec.go) 接口，并调用 codec.Register 将 tars 协议的 codec 注册到 trpc-go 框架中，以支持 tars 协议的编解码

### 3.2 桩代码生成工具

实现 [trpc4tars](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars/tools/trpc4tars) 工具（对标 tarsgo 框架的 tars2go），支持分析接口 jce 文件，然后根据其中的结构和接口的定义自动生成结构定义和 RPC 调用代码（包括客户端和服务端）

## 4 示例

### 4.1 安装 trpc4tars 工具

```shell
go get git.code.oa.com/trpc-go/trpc-codec/tars && go install git.code.oa.com/trpc-go/trpc-codec/tars/tools/trpc4tars
```

### 4.2 创建 trpc-go 服务

如果服务需要同时提供 trpc 协议和 tars 协议，可以先参照 [tRPC-Go 快速上手](https://iwiki.woa.com/pages/viewpage.action?pageId=118272478) 创建好服务
如果服务只需要提供 tars 协议，可以参照 [TafTestServer](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars/examples/TafTestServer) 创建好服务
记得复制示例服务中提供的`makefile`/`makefile.trpc`，然后修改`makefile`将其中的 app/server 替换成你的真实 app 和 server 名称

```makefile
APP               := NFA
TARGET            := TafTestServer
TRPC4TARS_FLAG    :=
GO_BUILD_FLAG     :=

#JCE_SRC += /home/tafjce/NFA/TafTestServer/TafTest.jce

include ./makefile.trpc
```

### 4.3 定义服务接口

tars 协议是通过 jce 文件来定义服务接口的，jce 用法请参照 [tarsgo/tars](https://git.woa.com/tarsgo/tars)
此处我们可以参照 jce 语法规范，定义服务的 jce 文件

```jce
module NFA
{
    struct HelloReq {
        0 optional string msg;
    };

    struct HelloRsp {
        0 optional string msg;
    };

    interface TafTest
    {
        int hello(HelloReq req, out HelloRsp rsp);
    };
};
```

### 4.4 实现接口功能

修改 jce 文件对应的实现文件，以 TafTestServer 示例服务为例，就是修改 taftest_imp.go，找到具体要实现的接口函数，增加业务逻辑

```go
type TafTestImp struct {}

// Init 初始化
func (imp *TafTestImp) Init() (int, error) {
    log.Debug("imp init ok, imp:", imp)
    return 0, nil
}

// Hello 实现接口 hello
func (imp *TafTestImp) Hello(ctx context.Context, req *comm.HelloReq, rsp *comm.HelloRsp) (int32, error) {
    rsp.Msg = req.Msg
    return 0, nil
}
```

### 4.5 本地编译

直接运行 make 命令，makefile 会自动调用 trpc4tars 工具生成桩代码（桩代码目录`tars-protocol`)，并生成服务二进制文件
`make upload2test`命令会自动将服务上传到 123 平台的 147 环境
`make upload`命令会自动将服务上传到 123 平台的 213 环境
> ps: make upload2test/upload 命令的前提是要在 123 平台上先创建好服务

### 4.6 123 平台部署

此处请参考 123 平台的文章 [上线一个 tRPC-Go 服务](https://iwiki.woa.com/pages/viewpage.action?pageId=928901287)
和普通 trpc-go 服务唯一的区别在于，框架配置有所不同 (protocol 配置为 tars)：

```yaml
server:
  app: NFA #业务的应用名
  server: TafTestServer #进程服务名
  service: #业务服务提供的 service，可以有多个
    - name: trpc.NFA.TafTestServer.TafTestObj
      ip: 127.0.0.1
      port: 8000
      protocol: tars #应用层协议，！！！！注意这里要配置为 tars 协议！！！！
      timeout: 3000
      idletime: 300000
      registry: polaris
```

### 4.7 tRPC 服务和 TAF 服务互调指南

<https://iwiki.woa.com/pages/viewpage.action?pageId=58492099>

## 5 FAQ

请参考服务端开发向导的 [FAQ](https://iwiki.woa.com/p/284289102#11-faq) 部分。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
