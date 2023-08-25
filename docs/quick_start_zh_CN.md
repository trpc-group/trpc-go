[TOC]


# tRPC-Go 快速上手



# 前言

Hello tRPC-Go !

现在，你已经对 tRPC-Go 有所 [了解](https://git.woa.com/trpc-go/trpc-wiki/blob/main/quick_start_zh-CN.md)，了解其工作机制最简单的方法就是看一个简单的例子。
Hello World 将带领你创建一个简单的后台服务，向你展示：

- 通过编写 protobuf，定义一个简单的带有 SayHello 方法的 RPC 服务。
- 通过 trpc 工具，生成服务端代码。
- 通过 rpc 方式，调用服务。

这个例子完整的代码在我们源码库的 [examples/helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld) 目录下。

# 环境搭建

在上手之前，必须先保证 Go 环境可用，没有 Go 环境时请先看 [环境搭建](todo)。

# 服务端开发

**注意：本文档旨在让用户简单快速熟悉服务开发流程，这里的开发步骤都是在本地执行的，实际业务开发时，一般都是通过更好的平台管理工具来提高效率，如用 [123 发布服务](todo)，用 rick 管理 pb 接口（详情见 [tRPC-Go 接口管理](todo) 以及 [Rick 平台简介](todo))。**

## 创建服务仓库

- 小仓模式下，每个服务单独创建一个 git project，如：`git.woa.com/trpc-go/helloworld`，demo 见 [这里](https://git.woa.com/trpc-go/helloworld)。
  到工蜂平台创建一个自己的 git 仓库 clone 到本地，如：`git clone git@git.woa.com:trpc-go/helloworld.git`。
  大仓模式下，每个服务一个子目录，不需要以下的 go.mod 文件，可跳过该 3.1 小节。
  或者不提交 git 的话，随便创建一个本地目录`helloworld`即可。
- 初始化 go mod 文件：

```shell
cd helloworld  # 进入服务内部，以后所有的操作都在这个目录下面执行go mod init git.woa.com/yourrtx/helloworld # yourrtx 替换为你的名字即可
```

## 定义服务接口

tRPC 采用 protobuf 来描述一个服务，我们用 protobuf 定义服务方法，请求参数和响应参数，cd 到前面创建好的目录里面并创建以下 pb 文件，`vim helloworld.proto`：

```go
syntax = "proto3";

// package 内容格式推荐为 trpc.{app}.{server}，以 trpc 为固定前缀，标识这是一个 trpc 服务协议，app 为你的应用名，server 为你的服务进程名
package trpc.test.helloworld;

// 注意：这里 go_package 指定的是协议生成文件 pb.go 在 git 上的地址，不要和上面的服务的 git 仓库地址一样
option go_package="git.woa.com/trpcprotocol/test/helloworld";

// 定义服务接口
service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// 请求参数
message HelloRequest {
  string msg = 1;
}

// 响应参数
message HelloReply {
  string msg = 1;
}
```

如上，这里我们定义了一个`Greeter`服务，这个服务里面有个`SayHello`方法，接收一个包含`msg`字符串的`HelloRequest`参数，返回`HelloReply`数据。
这里需要注意以下几点：

- `syntax`必须是`proto3`，tRPC 都是基于 proto3 通信的。
- `package`内容格式推荐为`trpc.{app}.{server}`，以 trpc 为固定前缀，标识这是一个 trpc 服务协议，app 为你的应用名，server 为你的服务进程名。注意：这里的格式仅仅只是 tRPC 框架的推荐！不是强制！不过，不同的平台（如 rick）考虑到权限控制以及服务管理等因素会强制这个要求，你要使用这个平台，那么就必须遵守该平台的约定。框架与平台无关，你可以自己考虑是否使用该平台。
- `package`后面必须有`option go_package="git.woa.com/trpcprotocol/{app}/{server}";`，指明你的 pb.go 生成文件的 git 存放地址，协议与服务分离，方便其他人直接引用，git 地址用户可以自己随便定，也可以使用 tRPC-Go 提供的公用 group：[trpcprotocol](https://git.woa.com/groups/trpcprotocol/-/projects/list)。
- rick 接口管理详情见 [tRPC-Go 接口管理](todo) 以及 [Rick 平台简介](todo) 。
- 定义`rpc`方法时，一个`server`（服务进程）可以有多个`service`（对`rpc`逻辑分组），一般是一个`server`一个`service`，一个`service`中可以有多个`rpc`调用。
- 编写 protobuf 时必须遵循 [公司 Protobuf 规范](https://git.woa.com/standards/protobuf)。

## 生成服务代码

- 通过`trpc`命令行生成服务代码，前提是先 [安装 trpc 工具](https://git.woa.com/trpc-go/trpc-go-cmdline)（trpc-go-dev 镜像已安装，不过需要自己升级 trpc 工具到最新版）：

> 注：code.oa 项目域名的正确访问需要配置 goproxy ([https://goproxy.woa.com](https://goproxy.woa.com/) )，并且保证 `go env` 的输出中 `GONOPROXY` 以及 `GOPRIVATE` 变量中不包含 `git.code.oa.com`，对于 `trpc.tech v2` 版的 trpc-go 试用，可以参考文章：https://km.woa.com/group/51889/articles/show/527221
>
> 主要是需要额外添加命令 `--domain=trpc.tech --versionsuffix=v2` （为保持兼容性，默认还是引的 code.oa 的 trpc-go）

```shell
# 首次使用，用该命令生成完整工程，当前目录下不要出现跟 pb 同名的目录名，如 pb 名为 helloworld.proto，则当前目录不要出现 helloworld 的目录名
trpc create --protofile=helloworld.proto 

# 只生成 rpcstub，常用于已经创建工程以后更新协议字段时，重新生成桩代码
trpc create --protofile=helloworld.proto --rpconly

# 使用 http 协议
trpc create --protofile=helloworld.proto --protocol=http
```

- 生成代码如下，代码在`main.go`和`greeter.go`文件：

```go
package main

import (
    "context"
	
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
	
    pb "git.code.oa.com/trpcprotocol/test/helloworld"
    trpc "git.code.oa.com/trpc-go/trpc-go"
)

type greeterServerImpl struct{}

// SayHello 函数入口，用户逻辑写在该函数内部即可
// error 代表的是 exception，异常情况比如数据库连接错误，调用下游服务错误的时候，如果返回 error，rsp 的内容将不再被返回
// 如果业务遇到需要返回的错误码，错误信息，而且同时需要保留 HelloReply，请设计在 HelloReply 里面，并将 error 返回 nil
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    // implement business logic here ...
    // ...
	
    rsp.Msg = "Hello, I am tRPC-Go server."
	
    return nil
}

func main() {
    // 创建一个服务对象，底层会自动读取服务配置及初始化插件，必须放在 main 函数首行，业务初始化逻辑必须放在 NewServer 后面
    s := trpc.NewServer()
	
    // 注册当前实现到服务对象中
    pb.RegisterGreeterService(s, &greeterServerImpl{})
	
    // 启动服务，并阻塞在这里
    if err := s.Serve(); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

以上代码均为工具自动生成，正如你所见，服务器有一个`greeterServerImpl`结构，他通过实现`SayHello`方法，实现了 protobuf 定义的服务。
此时，通过填充`SayHello`方法的`rsp`结构，即可向请求方回应数据了。

现在，试一下，修改上面`rsp.Msg`的值，返回你自己的数据吧。

**注：** 以上 pb 文件生成的桩代码一般通过 rick 平台管理，详情见 [tRPC-Go 接口管理](todo) 以及 [Rick 平台简介](todo)。

## 修改框架配置

```yaml
global:  # 全局配置
  namespace: Development  # 环境类型，分正式 Production 和非正式 Development 两种类型

server:  # 服务端配置
  app: test  # 业务的应用名
  server: helloworld  # 进程服务名
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.test.helloworld.Greeter  # service 的路由名称
      ip: 127.0.0.1  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000  # 服务监听端口 可使用占位符 ${port}
      network: tcp  # 网络监听类型  tcp udp
      protocol: trpc  # 应用层协议 trpc http
      timeout: 1000  # 请求最长处理时间 单位 毫秒
```

框架配置提供了服务启动的基本参数，包括 ip、端口、协议等等。框架配置详细指南看 [这里](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/framework_conf_zh_cn.md)。
这里我们配置了一个监听`127.0.0.1:8000`的`trpc 协议`的服务。

## 本地启动服务

直接编译好二进制，本地执行启动命令即可：

```
# 不要用 go build main.go，因为 main.go 可能依赖了当前目录下其他文件中的逻辑go build./helloworld &
```

当屏幕上出现以下日志时就说明服务启动成功了：

```shell
xxxx-xx-xx xx:xx:xx.xxx    INFO    server/service.go:132    process:xxxx, trpc service:trpc.test.helloworld.Greeter launch success, address:127.0.0.1:8000, serving ...
```

## 自测联调工具

- 通过 tRPC-Go 提供的客户端发包工具`trpc-cli`命令行进行自测，前提是先 [安装 trpc-cli 工具](https://git.woa.com/trpc-go/trpc-cli)：

```shell
trpc-cli -func /trpc.test.helloworld.Greeter/SayHello -target ip://127.0.0.1:8000 -body '{"msg":"hello"}'
```

trpc-cli 工具支持很多参数，使用时注意指定。

- `func`为 pb 协议定义的 `/package.service/method`，如上面的 helloworld.proto，则为`/trpc.test.helloworld.Greeter/SayHello`，`千万千万注意：不是 yaml 里面配置的 service`。
- `target`为被调服务的目标地址，格式为`selectorname://servicename`，详细信息可以查看 [tRPC-Go 客户端开发向导](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/client/overview_zh_CN.md)，这里只是本地自测，没有接入名字服务，直接指定 ipport 寻址，使用 ip selector 就可以了，格式是`ip://${ip}:${port}`，如`ip://127.0.0.1:8000`。
- `body`为请求包体数据的 json 结构字符串，内部 json 字段要跟 pb 定义的字段完全一致，注意大小写不要写错。

假如想体验 tRPC-Go 的整个链路和所有插件使用，可以参考 : [全链路工程 helloworld demo](http://git.woa.com/trpc-go/helloworld.git)。

开发过程中可以查询框架的 [API 文档](http://godoc.woa.com/git.woa.com/trpc-go/trpc-go)。

**注：** `trpc` 和 `trpc-cli` 是两个不同的工具，前者主要用于生成 proto 对应的 stub 代码，后者主要用于作为 client 发送请求，各自 wiki 以及 git 地址如下：

- `trpc`：[wiki (trpc-go-cmdline 工具）](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/cmdline_tool_zh_cn.md)、https://git.woa.com/trpc-go/trpc-go-cmdline
- `trpc-cli`：[wiki (tRPC-Go 接口测试）](todo)、https://git.woa.com/trpc-go/trpc-cli

更多工具见 [tRPC-Go 环境搭建](todo) 中的【3.5 安装常用工具】一节。

# 客户端开发

使用 tRPC-Go 开发一个客户端调用后端服务非常简单，通过 pb 生成的代码已经包含了调用方法，调用远程接口就像调用本地函数一样。
现在我们来开发一个客户端来调用前面的服务吧：

```shell
mkdir client
cd client
go mod init git.woa.com/trpc-go/client  # 你自己的 git 地址
vim main.go
```

```go
package main 

import (
    "context"
	
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
	
    pb "git.code.oa.com/trpcprotocol/test/helloworld" // 被调服务的协议生成文件 pb.go 的 git 地址，没有 push 到 git 的话，可以在 gomod 里面 replace 本地路径进行引用，如 gomod 里面加上一行：replace "git.code.oa.com/trpcprotocol/test/helloworld" => ./你的本地桩代码路径
)

func main() {
    proxy := pb.NewGreeterClientProxy() // 创建一个客户端调用代理，名词解释见客户端开发文档
    req :=  &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."} // 填充请求参数
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTarget("ip://127.0.0.1:8000")) // 调用目标地址为前面启动的服务监听的地址
    if err != nil {
        log.Errorf("could not greet: %v", err)
        return
    }
    log.Debugf("response: %v", rsp)
}
```

```shell
go build
./client
```

正常情况，客户端代码不会如此简单，一般都是在服务内部调用下游服务，更加详细的客户端代码请看用户指南里面的 [客户端开发](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/client/overview_zh_CN.md)，或者可以直接看 [example/helloworld](https://git.woa.com/trpc-go/trpc-go/blob/master/examples/helloworld/greeter.go#L31) 的代码。

# 部署上线

```
首先大家需要了解，框架是完全独立的，跟任何平台都没有绑定关系，可以支持在任何平台部署。
```

## 123 平台部署

123 平台是 PCG 容器发布平台，PCG 员工后续新服务都会统一到这个平台 [发布服务](todo)。
注意：使用 123 平台部署需要引入北极星插件，具体参考插件文档：[北极星服务注册与发现](https://git.woa.com/trpc-go/trpc-naming-polaris)。

## 织云部署

织云是一个比较古老的二进制发布平台。首先需要编译好二进制再拖到平台上 [发布](http://tapd.oa.com/zhiyun/markdown_wikis/view/#1010125021009540855)。

- build
  执行 go build -v 会生成一个二进制文件
- 织云发布
  选择： `后台 server 包`
  启动命令： `./app -conf ../conf/trpc_go.yaml &`

登录 [织云](http://yun.isd.com/index.php/package/create/) 平台进行打包发布，可参考：[织云部署](http://tapd.oa.com/zhiyun/markdown_wikis/view/#1010125021009540855)。

## stke 部署

有些团队在大范围 [使用 stke 进行部署](todo)，也可以按需定制流水线在 stke 进行部署。要注意某些能力的支持程度，如北极星能否完成注册。

## GDP/ODP 部署

GDP/ODP 是 IEG 云原生开发者平台，提供了 trpc 的线上部署、持续运营功能
[腾讯游戏微服务平台](https://gdp.woa.com/)
创建业务的项目，通过 trpc 模板创建好服务，即可发布访问，具体使用方式可以咨询 GDP&ODP 助手

# FAQ

**更多问题请查找：** [tRPC-Go 常见问题](todo)

# OWNER

## nickzydeng