## 1 前言

本节展示如何使 tRPC-Go 服务调用 thrift 协议服务

## 2 原理

见 [tRPC-Go 搭建 thrift 服务](https://iwiki.woa.com/p/4012787971) 中的原理介绍

## 3 示例

在 [tRPC-Go 搭建 thrift 服务](https://iwiki.woa.com/p/4012787971) 的示例部分已经可以生成客户端代码，整体工程目录结构如下：

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

接下来，我们的目标是与存量的服务互通。纯 thrift 服务/客户端，指的是使用纯开源的 [thrift 库](https://github.com/apache/thrift/tree/master/lib/go) 开发，而没有基于 tRPC 框架的一类服务/客户端。

### 3.1 client 调用纯 thrift 服务

还是基于原来 [tRPC-Go 搭建 thrift 服务](https://iwiki.woa.com/p/4012787971) 的例子，我们在 `out-greeter` 目录下新建一个 `server.go` 文件，用于编写纯 thrift 服务端代码，文件的目录结构如下
（如果你已经有存量的纯 thrift 服务，可以直接跳过这部分 server 的编写）：

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
├── server.go                                   # 纯 thrift 服务端代码放在这里
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

编写 `server.go` 文件，内容如下：

```go
// server.go

package main

import (
    "fmt"
    "os"

    thr "git.woa.com/trpcprotocol/testapp/greeter"

    "github.com/apache/thrift/lib/go/thrift"
)

const (
    networkAddr = "127.0.0.1:8000"
)

func main() {
    // 创建传输工厂和协议工厂
    transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())
    protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
    serverSocket, err := thrift.NewTServerSocket(networkAddr)
    if err != nil {
        fmt.Println("Error!", err)
        os.Exit(1)
    }

    // 注册 handler
    handle := &greeterImpl{}
    processor := thr.NewGreeterProcessor(handle)
    // 创建服务端
    server := thrift.NewTSimpleServer4(processor, serverSocket, transportFactory, protocolFactory)

    if err := server.Serve(); err != nil {
        fmt.Println("thrift serve err: ", err)
    }
}

```

然后，可以参考 `cmd/client/main.go` 来编写客户端代码：

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

以上为纯客户端的写法，当在一个服务中写下游的客户端时，需要调用的服务信息可以通过 `trpc_go.yaml` 来进行配置，从而省去以下部分

```go
proxy := thr.NewGreeterClientProxy(
    client.WithTarget("ip://127.0.0.1:8000"),
    client.WithProtocol("thrift"),
)
```

在一个终端内，编译并运行服务端（由于现在我们同一个 package 里面有两个 `main` 函数，因此指定一下 build 的文件）：

```shell
# 在 out-greeter 项目目录下
go build -o thrift-server server.go greeter.go     # 编译
./thrift-server                                    # 运行
```

在另一个终端内，运行客户端：

```shell
# 在 out-greeter 项目目录下
go run cmd/client/main.go 
```

此时会看到如下结果：

```text
plugin log-default setup succeed, time elapsed: 510.709µs
2024-08-30 16:51:52.264 DEBUG   debuglog@v0.1.13/log.go:229     client request:/trpc.app.server.Greeter/SayHello, cost:939.416µs, to:127.0.0.1:8000
2024-08-30 16:51:52.264 DEBUG   client/main.go:29       simple rpc receive: HelloReply({Message:hello, thrift client})
```

### 3.2 纯 thrift 客户端调用 server

我们在 `out-greeter/cmd/client/main.go` 目录下新建一个 `client.go` 文件，用于编写纯 thrift 客户端代码，文件的目录结构如下：

```text
out-greeter
├── cmd
│   └── client
│       ├── client.go                           # 纯 thrift 客户端代码放在这里
│       └── main.go                             # 客户端代码
├── go.mod                                      # 服务端的 go.mod 文件
├── go.sum
├── greeter.go                                  # 第一个 service 的服务端实现
├── greeter_another.go                          # 第二个 service 的服务端实现
├── main.go                                     # 服务端启动代码
├── server.go                                   # 纯 thrift 服务端代码放在这里
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

编写 `client.go` 文件，内容如下：

```go
// cmd/client/client.go

package main

import (
    "context"
    "fmt"
    "os"

    thr "git.woa.com/trpcprotocol/testapp/greeter"

    "github.com/apache/thrift/lib/go/thrift"
)

const (
    networkAddr = "127.0.0.1:8000"
)

func main() {
    // 创建传输层
    transport, err := thrift.NewTSocket(networkAddr)
    if err != nil {
        fmt.Println("Error opening socket:", err)
        os.Exit(1)
    }

    // 创建传输工厂和协议工厂
    transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())
    protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

    // 打开传输
    useTransport, err := transportFactory.GetTransport(transport)
    if err != nil {
        fmt.Println("Error getting transport:", err)
        os.Exit(1)
    }
    if err := useTransport.Open(); err != nil {
        fmt.Println("Error opening transport:", err)
        os.Exit(1)
    }
    defer useTransport.Close()

    // 创建客户端
    client := thr.NewGreeterClientFactory(useTransport, protocolFactory)

    // 调用服务方法
    ctx := context.Background()
    response, err := client.SayHello(ctx, &thr.HelloRequest{Name: "pure thrift client"})
    if err != nil {
        fmt.Println("Error calling SayHello:", err)
        os.Exit(1)
    }

    fmt.Println("Response from server:", response)
}

```

在一个终端内，运行上面已经编译好的服务端：

```shell
# 在 out-greeter 项目目录下
./thrift-server # 运行纯 thrift 服务端
```

在另一个终端内，运行新编写的纯 thrift 客户端：

```shell
# 在 out-greeter 项目目录下
go run cmd/client/client.go 
```

此时会有如下输出：

```text
Response from server: HelloReply({Message:hello, pure thrift client})
```

如果是在 [tRPC-Go 搭建 thrift 服务](https://iwiki.woa.com/p/4012787971) 中搭建的基于 tRPC 框架的 thrift 服务端，
那么在另一个终端内，运行这个服务端：

```shell
# 在 out-greeter 项目目录下
go build -o myserver main.go greeter.go # 编译基于 tRPC 框架的 thrift 服务端
./myserver                              # 运行
```

在另一个终端内，运行客户端：

```shell
# 在 out-greeter 项目目录下
go run cmd/client/client.go 
```

在控制台也能得到正常的输出。

## 4 FAQ

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
