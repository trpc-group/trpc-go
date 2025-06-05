## 1 背景

[Apache Thrift](https://thrift.apache.org/) 是一个用于跨语言服务开发的框架。它最初由 Facebook 开发，并在 2007
年[开源](https://github.com/apache/thrift)。
Thrift 允许开发者定义数据类型和服务接口，然后生成代码以支持多种编程语言，从而实现跨语言的 RPC。目前它已支持 C++, Java,
Python, Go 等 28 种语言。

## 2 实现

* trpc-go 主库 `codec` 包实现 thrift 序列化 ( MR 见 [!2940](https://git.woa.com/trpc-go/trpc-go/-/merge_requests/2490) )
* trpc-codec 仓库实现 thrift 编解码，并提供 `trpc4thrift` 工具，用于生成桩代码
  （MR 见 [!722](https://git.woa.com/trpc-go/trpc-codec/-/merge_requests/722)
  、[!724](https://git.woa.com/trpc-go/trpc-codec/-/merge_requests/724) 和
  [!725](https://git.woa.com/trpc-go/trpc-codec/-/merge_requests/725)）

## 3 环境配置

`trpc4thrift` 采用第三方编译工具 [`thriftgo`](https://github.com/cloudwego/thriftgo)
来生成桩代码，工具内部集成了 `thriftgo` 编译器，因此暂时无需额外安装其他工具。
由于目前命令行暂不自动执行 `go mod tidy` 和 `mockgen`，因此用户需要自行安装 `go` 环境和 `mockgen` 二进制文件。

安装 `mockgen`（可以参考 [官方仓库](https://github.com/uber-go/mock)）:

```shell
go install go.uber.org/mock/mockgen@latest
```

然后通过以下命令安装 `trpc4thrift`：

```shell
go install git.code.oa.com/trpc-go/trpc-codec/thrift/tools/trpc4thrift@latest
```

## 4 示例

接下来分别介绍如何使用 `trpc4thrift` 工具生成 Thrift 协议和 tRPC 协议的代码。
你也可以在 [这里](https://git.woa.com/trpc-go/trpc-codec/tree/master/thrift/examples)
（与下面的示例代码不完全相同）查看完整的代码。

### 4.1 Thrift 协议

我们通过一个简单的例子来走一遍所有的流程。

首先定义 IDL 文件，语法可以从 [thrift 官网](https://thrift.apache.org/docs/idl) 上进行学习，整体的语法和 C 语言有相同之处。

基于实现难度以及和 Protobuf 对齐考虑，目前工具对 IDL 定义的 RPC 方法做了一些限制：

1. 只支持单个入参，不支持多个入参或者无参
2. 入参和返回值需要做一层封装，不支持直接使用基本数据类型（如 `i32`, `list`, `map`等），返回值不支持 `void`

例如：

```thrift
struct HelloRequest {
    1: required string name;
}

struct HelloReply {
    1: required string message;
}

// 不合法，入参有多个
service Greeter {
    HelloReply SayHello(1:HelloRequest req, 2:HelloRequest req2);
}

// 不合法，入参是基本数据类型
service Greeter {
    HelloReply SayHello(1:i32 req);
}

// 不合法，返回值是基本数据类型
service Greeter {
    i32 SayHello(1:HelloRequest req);
}

// 不合法，不支持 void，不支持无参
service Greeter {
    void SayHello();
}

// 合法
service Greeter {
    HelloReply SayHello(1:HelloRequest req);
}

```

假设我们有一个 `greeter.thrift` 文件如下：

```thrift
namespace go trpc.app.server // 相当于 protobuf 中的 package

// 相当于 protobuf 的 go_package 声明
// 注意：这个字段不强制要求，为了和 protobuf 保持一致而做兼容
const string go_package = "git.woa.com/trpcprotocol/testapp/greeter"

struct HelloRequest {
    1: required string name
}

struct HelloReply {
    1: required string message
}

// 单发单收服务
service Greeter {
    HelloReply SayHello(1:HelloRequest req)
}

// 含有两个服务时的示例
service GreeterAnother {
    HelloReply SayAnotherHello(1:HelloRequest req)
}

```

其中，`go_package` 字段的含义类似 protobuf
中对应部分的含义，见 [protobuf#package](https://protobuf.dev/reference/go/go-generated/#package) 。

以上链接中点出 protobuf 中的 `package` 和 `go_package` 字段没有关系：

> There is no correlation between the Go import path and the package specifier in the .proto file. The latter is only
> relevant to the protobuf namespace, while the former is only relevant to the Go namespace.

同理，thrift 中的 `namespace go` 和 `go_package` 字段也没有关系，两者在 `trpc4thrift`
中是为了指定桩代码生成路径以及桩代码的包名 (`package name`)，
且 `go_package` 的优先级会比 `namespace go` 更高。

* 如果定义了 `go_package` 字段，则使用 `go_package` 字段作为桩代码路径，`go_package` 最后一个 `/`
  后的内容作为 `package name`；
* 如果没有定义 `go_package` 字段，则使用 `namespace go` 字段作为桩代码路径，`namespace go` 将 `.` 全部替换为 `/`
  后作为 `package name`；

可以参考以下表格来对比两者对桩代码路径和 `package name` 的影响（其中，`-` 表示未定义，`*` 表示任意值）：

| `go_package`                    | `namespace go`      | 桩代码路径                                | `package name`      |
|---------------------------------|---------------------|--------------------------------------|---------------------|
| `git.woa.com/trpc/test/greeter` | *                   | `stub/git.woa.com/trpc/test/greeter` | `greeter`           |
| `trpc/test/greeter`             | *                   | `stub/trpc/test/greeter/greeter`     | `greeter`           |
| `greeter`                       | *                   | `stub/greeter`                       | `greeter`           |
| -                               | `trpc.test.greeter` | `stub/trpc.test.greeter`             | `trpc_test_greeter` |
| -                               | `trpc_test`         | `stub/trpc_test`                     | `trpc_test`         |

然后使用如下命令可以生成对应的桩代码：

```shell
trpc4thrift create -t greeter.thrift -o out-greeter --go-mod git.woa.com/myapp/myserver --protocol thrift
```

其中：

* `-t` 指定了 thrift IDL 的文件名（带相对路径）。
* `-o` 指定了输出路径，如果没有指定 `-o`，则会新建一个与 thrift IDL
  同名的文件夹。注意：输出路径非空会报错，如果希望覆盖已存在的文件夹，可以使用 `-f`。
* `--go-mod` 指定了生成文件 `go.mod` 中 `package`
  的内容，假如没有 `--go-mod` 的话，它会默认使用 `trpc.app.{ServerName}` 作为 `--go-mod` 的内容，其中 `{ServerName}` 为 IDL
  文件中第一个服务的名称。注意，这个 `--go-mod` 表示的是服务端本身的模块路径标识，和 IDL 文件中的 `go_package`
  不同，后者标识的是桩代码的模块路径标识。
* `--protocol` 指定了协议，目前支持 `thrift` 和 `trpc` 两种协议。如果没有指定 `--protocol`，则会默认使用 `thrift`
  协议。注意，在 `thrift` 协议下，注册 Handler 是以 `Method` 作为路由的，区别于 `trpc`
  协议下以 `/trpc.app.server.Service/Method` 作为路由。因此，当使用 `thrift`
  协议时，如果涉及多个 `Service`，那么需要保证所有 `Service` 的 `Method` 名称唯一。例如上面的 `service Greeter`
  和 `service GreeterAnother` 只能全局存在一个 `SayHello` 方法。

```thrift
// --protocol thrift 时以下 IDL 不合法，方法名 SayHello 不为全局唯一
service GreeterA {
    HelloReply SayHello(1:HelloRequest req);
}

service GreeterB {
    HelloReply SayHello(1:HelloRequest req);
}

```

生成的代码目录结构如下：

```text
out-greeter
├── cmd
│   └── client
│       └── main.go                             # 客户端代码
├── go.mod                                      # 服务端的 go.mod 文件
├── go.sum
├── greeter.go                                  # 第一个 service 的服务端实现
├── greeter_another.go                          # 第二个 service 的服务端实现
├── main.go                                     # 服务端启动代码
├── stub                                        # 桩代码目录
│   └── git.woa.com
│       └── trpcprotocol
│           └── testapp
│               └── greeter                     # 因为定义了 go_package，所以这里使用了 go_package 作为桩代码路径
│                   ├── go.mod                  # 桩代码的 go.mod 文件
│                   ├── greeter.thrift          # 原始 thrift IDL 文件
│                   ├── greeter.thrift.go       # thrift 协议相关的桩代码
│                   └── greeter.trpc.go         # trpc 协议相关的桩代码
└── trpc_go.yaml                                # trpc-go 配置文件

```

`trpc-go` 已发测试版 `v0.19.0-beta`，`trpc-codec/thrift` 已发 `v0.0.1`，可以直接拉取 tag 使用。

**注意**：

1. 在桩代码生成后，需要在项目目录和桩代码的目录手动执行 `go mod tidy`，以更新依赖

```shell
cd out-greeter # 项目目录
go mod tidy
cd stub/git.woa.com/trpcprotocol/testapp/greeter # 桩代码目录
go mod tidy
```

2. 如果需要生成 `mock` 代码，请自行安装 `mockgen`
   工具，具体安装方式可以参考 [mockgen 安装](https://github.com/uber-go/mock#installation)。
   然后手动执行以下命令：

```shell
cd stub/git.woa.com/trpcprotocol/testapp/greeter # 桩代码目录
mockgen -source=greeter.trpc.go -destination=greeter_mock.go -package=greeter
```

其中，`-source` 表示要生成 mock 代码的源文件，`-destination` 表示生成的 mock 代码的文件路径，`-package` 表示生成的 mock
代码的包名，这里与 `greeter` 文件夹中的 `package`
保持一致。如果需要更多选项，可以参考 [mockgen 用法](https://github.com/uber-go/mock#running-mockgen)。

还需要注意的是，生成的 `trpc_go.yaml` 配置文件中，`protocol` 为生成桩代码的命令中 `--protocol` 选项保持一致。如果需要使用
`trpc` 协议，
需要 **指定 `--protocol trpc` 并使用工具重新生成代码**，不能简单地将 `trpc_go.yaml` 配置文件中的 `protocol` 字段改为
`trpc`。

```yaml
server: # 服务端配置
  app: app  # 业务的应用名
  server: server  # 进程服务名
  service: # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.Greeter  # service 的路由名称
      ip: 127.0.0.1  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      # nic: eth0
      port: 8000  # 服务监听端口 可使用占位符 ${port}
      network: tcp  # 网络监听类型 tcp udp
      protocol: thrift # 应用层协议 trpc 或 thrift, 这里使用 thrift 协议
      timeout: 1000  # 请求最长处理时间 单位 毫秒

client:  # 客户端调用的后端配置
  service:  # 针对单个后端的配置
    - name: trpc.app.server.Greeter  # 后端服务的 service name
      namespace: Development  # 后端服务的环境
      network: tcp  # 后端服务的网络类型 tcp udp 配置优先
      protocol: thrift # 应用层协议 trpc 或 thrift, 这里使用 thrift 协议

```

接下来，在项目目录下，实现一个简单的 service:

```go
// greeter.go

package main

import (
    "context"

    thr "git.woa.com/trpcprotocol/testapp/greeter"
)

type greeterImpl struct {
    thr.Greeter
}

func (s *greeterImpl) SayHello(
    ctx context.Context,
    req *thr.HelloRequest,
) (*thr.HelloReply, error) {
    rsp := &thr.HelloReply{Message: "hello, " + req.Name}
    return rsp, nil
}

```

然后，在 `cmd/client` 目录下，实现一个简单的 client:

```go
// cmd/client/main.go

package main

import (
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"

    _ "git.code.oa.com/trpc-go/trpc-codec/thrift" // 导入 thrift 编解码器
    _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"

    thr "git.woa.com/trpcprotocol/testapp/greeter"
)

func callGreeterSayHello() {
    proxy := thr.NewGreeterClientProxy(
        client.WithTarget("ip://127.0.0.1:8000"),
    )
    ctx := trpc.BackgroundContext()
    // 一发一收 client 用法示例
    reply, err := proxy.SayHello(ctx, &thr.HelloRequest{Name: "my first trpc4thrift project"})
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    log.Debugf("simple rpc receive: %+v", reply)
}

func main() {
    // 仿照 trpc.NewServer 中的逻辑进行配置的加载
    cfg, err := trpc.LoadConfig(trpc.ServerConfigPath)
    if err != nil {
        panic("load config fail: " + err.Error())
    }
    trpc.SetGlobalConfig(cfg)
    if err := trpc.Setup(cfg); err != nil {
        panic("setup plugin fail: " + err.Error())
    }
    callGreeterSayHello()
}

```

在一个终端内，编译并运行服务端：

```shell
# 在 out-greeter 项目目录下
go build      # 编译
./myserver    # 运行
```

在另一个终端内，运行客户端：

```shell
# 在 out-greeter 项目目录下
go run cmd/client/main.go 
```

在两个终端的控制台就可以看到有对应的日志输出。

启动服务的 `main.go` 文件展示如下：

```go
// main.go

package main

import (
    _ "git.code.oa.com/trpc-go/trpc-codec/thrift" // 导入 thrift 编解码器
    _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"
    _ "git.code.oa.com/trpc-go/trpc-filter/recovery"
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/log"
    thr "git.woa.com/trpcprotocol/testapp/greeter"
)

func main() {
    s := trpc.NewServer()

    thr.RegisterGreeter(s, &greeterImpl{})

    thr.RegisterGreeterAnother(s, &greeterAnotherImpl{})
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}

```

其中，如果使用的是 `thrift` 协议，请**匿名导入** `_ "git.code.oa.com/trpc-go/trpc-codec/thrift"` 注册 `thrift` 协议的编解码。
同理，客户端也需要导入 `thrift` 协议的编解码器，否则无法解析 `thrift` 协议的请求。

```go
// cmd/client/main.go

package main

import (
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"

    _ "git.code.oa.com/trpc-go/trpc-codec/thrift" // 导入 thrift 编解码器
    _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"

    thr "git.woa.com/trpcprotocol/testapp/greeter"
)

func callGreeterSayHello() {
    proxy := thr.NewGreeterClientProxy(
        client.WithTarget("ip://127.0.0.1:8000"),
        client.WithProtocol("thrift"), // 这里也可以手动指定协议
    )
    ctx := trpc.BackgroundContext()
    // 一发一收 client 用法示例
    reply, err := proxy.SayHello(ctx, &thr.HelloRequest{Name: "thrift client"})
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    log.Debugf("simple rpc receive: %+v", reply)
}

func main() {
    // 仿照 trpc.NewServer 中的逻辑进行配置的加载
    cfg, err := trpc.LoadConfig(trpc.ServerConfigPath)
    if err != nil {
        panic("load config fail: " + err.Error())
    }
    trpc.SetGlobalConfig(cfg)
    if err := trpc.Setup(cfg); err != nil {
        panic("setup plugin fail: " + err.Error())
    }
    callGreeterSayHello()
}

```

### 4.2 tRPC 协议

`trpc4thrift` 也支持 `trpc` 协议，使用方式与 `thrift` 协议类似。当使用 `trpc` 协议时，相当于把 `.thrift` 文件当成 `.proto`
来用，生成的桩代码基本相似。
正如前面介绍 `--protocol` 选项中提到的，桩代码中注册 Handler 时，在 `trpc` 协议下以 `/trpc.app.server.Service/Method` 作为
key，区别于 `thrift` 协议下仅仅以 `Method` 作为 key。因此，当使用 `trpc` 协议时，解除了 `Method` 名称全局唯一的限制。

```thrift
// --protocol trpc 时以下 IDL 也合法
service GreeterA {
    HelloReply SayHello(1:HelloRequest req);
}

service GreeterB {
    HelloReply SayHello(1:HelloRequest req);
}

```

我们还是使用第 4.1 节的 IDL，使用以下命令生成 `trpc` 协议的桩代码（注意 `--protocol` 选项）：

```shell
trpc4thrift create -t greeter.thrift -o trpc-greeter --go-mod git.woa.com/myapp/myserver --protocol trpc
```

生成的代码目录结构如下：

```text
out-greeter
├── cmd
│   └── client
│       └── main.go                             # 客户端代码
├── go.mod                                      # 服务端的 go.mod 文件
├── go.sum
├── greeter.go                                  # 第一个 service 的服务端实现
├── greeter_another.go                          # 第二个 service 的服务端实现
├── main.go                                     # 服务端启动代码
├── stub                                        # 桩代码目录
│   └── git.woa.com
│       └── trpcprotocol
│           └── testapp
│               └── greeter                     # 因为定义了 go_package，所以这里使用了 go_package 作为桩代码路径
│                   ├── go.mod                  # 桩代码的 go.mod 文件
│                   ├── greeter.thrift          # 原始 thrift IDL 文件
│                   ├── greeter.thrift.go       # thrift 协议相关的桩代码
│                   └── greeter.trpc.go         # trpc 协议相关的桩代码
└── trpc_go.yaml                                # trpc-go 配置文件

```

可以发现，目录结构和第 4.1 节 `--protocol thrift` 生成的代码目录结构完全一致。文件内容上主要有以下两点区别：

1. `trpc_go.yaml` 文件中，`server` 和 `client` 的 `protocol` 字段被设置为 `trpc`。

```yaml
server: # 服务端配置
  app: app  # 业务的应用名
  server: server  # 进程服务名
  service: # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.Greeter  # service 的路由名称
      ip: 127.0.0.1  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      # nic: eth0
      port: 8000  # 服务监听端口 可使用占位符 ${port}
      network: tcp  # 网络监听类型 tcp udp
      protocol: trpc # 应用层协议 trpc 或 thrift, 这里使用 trpc 协议
      timeout: 1000  # 请求最长处理时间 单位 毫秒

client:  # 客户端调用的后端配置
  service:  # 针对单个后端的配置
    - name: trpc.app.server.Greeter  # 后端服务的 service name
      namespace: Development  # 后端服务的环境
      network: tcp  # 后端服务的网络类型 tcp udp 配置优先
      protocol: trpc # 应用层协议 trpc 或 thrift, 这里使用 trpc 协议

```

2. `stub/git.woa.com/trpcprotocol/testapp/greeter` 目录下，`greeter.thrift.go` 代码中注册 Handler 的 `Name` 字段不同（这点我们在前面已经讲过）

```go
// trpc 协议中使用 `/trpc.app.server.Service/Method``
var GreeterServer_ServiceDesc = server.ServiceDesc{
    ServiceName: "trpc.app.server.Greeter",
    HandlerType: ((*GreeterService)(nil)),
    Methods: []server.Method{
        {
            Name: "/trpc.app.server.Greeter/SayHello",
            Func: GreeterService_SayHello_Handler,
        },
    },
}

// thrift 协议中使用 `Method`
var GreeterServer_ServiceDesc = server.ServiceDesc{
    ServiceName: "trpc.app.server.Greeter",
    HandlerType: ((*GreeterService)(nil)),
    Methods: []server.Method{
        {
            Name: "SayHello",
            Func: GreeterService_SayHello_Handler,
        },
    },
}

```

除此以外，其他代码完全一致。因此，我们可以复用这个简单 service 的业务代码：

```go
// greeter.go

package main

import (
    "context"

    thr "git.woa.com/trpcprotocol/testapp/greeter"
)

type greeterImpl struct {
    thr.Greeter
}

func (s *greeterImpl) SayHello(
    ctx context.Context,
    req *thr.HelloRequest,
) (*thr.HelloReply, error) {
    rsp := &thr.HelloReply{Message: "hello, " + req.Name}
    return rsp, nil
}

```

以及复用在 `cmd/client` 目录下简单 client 的代码，这里就不用导入 thrift 编解码器了：

```go
// cmd/client/main.go

package main

import (
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"

    _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"

    thr "git.woa.com/trpcprotocol/testapp/greeter"
)

func callGreeterSayHello() {
    proxy := thr.NewGreeterClientProxy(
        client.WithTarget("ip://127.0.0.1:8000"),
    )
    ctx := trpc.BackgroundContext()
    // 一发一收 client 用法示例
    reply, err := proxy.SayHello(ctx, &thr.HelloRequest{Name: "my first trpc4thrift project"})
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    log.Debugf("simple rpc receive: %+v", reply)
}

func main() {
    // 仿照 trpc.NewServer 中的逻辑进行配置的加载
    cfg, err := trpc.LoadConfig(trpc.ServerConfigPath)
    if err != nil {
        panic("load config fail: " + err.Error())
    }
    trpc.SetGlobalConfig(cfg)
    if err := trpc.Setup(cfg); err != nil {
        panic("setup plugin fail: " + err.Error())
    }
    callGreeterSayHello()
}

```

在一个终端内，编译并运行服务端：

```shell
# 在 trpc-greeter 项目目录下
go build      # 编译
./myserver    # 运行
```

在另一个终端内，运行客户端：

```shell
# 在 trpc-greeter 项目目录下
go run cmd/client/main.go 
```

在两个终端的控制台就可以看到有对应的日志输出。

## 5 业务代码复用

根据第 4 节的示例，我们分析了 `thrift` 和 `trpc` 协议生成代码的异同，可以发现只有桩代码中 Handler 注册方式，以及
`trpc_go.yaml` 配置文件中 `protocol` 字段。因此，如果需要迁移协议，只需要在其他文件夹中重新生成代码：

```shell
trpc4thrift create -t xxx.thrift -o temp_dir --protocol your_target_protocol
```

然后，替换原项目中的 `stub` 文件夹和 `trpc_go.yaml` 文件即可。

考虑到这里还是有一定的不方便，后续可以考虑增加仅生成桩代码的选项（例如 `--stub-only`）。

## 6 FAQ

### Q1: 报错 `serializer not registered`

在不同版本的 trpc-go 中，Thrift 序列化器的注册方式有所不同。
- 在 trpc-go 版本 < v0.20.0 的情况下，Thrift 序列化器是默认注册的，不需要手动进行任何操作。
- 在 trpc-go 版本 >= v0.20.0 的情况下，Thrift 序列化器被移动到了 trpc-codec 中。为了使用 Thrift 序列化功能，需要通过匿名导入 trpc-codec 的方式注册 Thrift 序列化器。幸运的是，使用 thrift4trpc 工具生成的代码已经自动匿名导入了 trpc-codec，因此不需要额外的手动操作。此外，trpc-codec 的 thrift/v0.0.3 版本引入了 Thrift 序列化器注册功能，因此需要确保使用 trpc-codec 的版本 >= thrift/v0.0.3。代码示例如下：
```go
package main

import (
    _ "git.code.oa.com/trpc-go/trpc-codec/thrift" // 为 Thrift 协议注册 codec 和 serialization
    trpc "git.code.oa.com/trpc-go/trpc-go"
)

func main() {
    // ...
}
```

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
