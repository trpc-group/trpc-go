[English](quick_start.md) | 中文

## 快速开始

### 准备工作

- **[Go](https://go.dev/doc/install)**, 版本应该大于等于 go1.18。
- **[tRPC 命令行工具](https://github.com/trpc-group/trpc-cmdline)**, 用于从 protobuf 生成 Go 桩代码。

### 获取示例代码

示例代码是 tRPC-Go 仓库的一部分。
克隆仓库并进入 helloworld 目录。
```bash
$ git clone --depth 1 git@github.com:trpc-group/trpc-go.git
$ cd trpc-go/examples/helloworld
```

### 执行示例

1. 编译并执行服务端代码：
   ```bash
   $ cd server && go run main.go
   ```
2. 打开另一个终端，编译并执行客户端代码：
   ```bash
   $ cd client && go run main.go
   ```
   你会在客户端日志中发现 `Hello world!` 字样。

恭喜你！你已经成功地在 tRPC-Go 框架中执行了客户端-服务端应用示例。

### 更新 protobuf

可以看到，protobuf `./pb/helloworld.proto` 定义了服务 `Greeter`：
```protobuf
service Greeter {
  rpc Hello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string msg = 1;
}

message HelloReply {
  string msg = 1;
}
```
它只有一个方法 `Hello`。它的参数是 `HelloRequest`，返回一个 `HelloReply`。

现在，我们加入一个新的方法 `HelloAgain`，使用相同的参数和返回值。
```protobuf
service Greeter {
  rpc Hello (HelloRequest) returns (HelloReply) {}
  rpc HelloAgain (HelloRequest) returns (HelloReply) {}
}


message HelloRequest {
  string msg = 1;
}

message HelloReply {
  string msg = 1;
}
```

通过在 `./pb` 目录中执行 `$ make` 方法来重新生成 tRPC 桩代码。
在「准备工作」一节，我们已经安装了 Makefile 所需要的命令行工具 `trpc`。

### 更新并执行服务端和客户端

在服务端 `server/main.go`，加入以下代码来实现 `HelloAgain`：
```go
func (g Greeter) HelloAgain(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    log.Infof("got HelloAgain request: %s", req.Msg)
    return &pb.HelloReply{Msg: "Hello " + req.Msg + " again!"}, nil
}
```

在客户端 `client/main.go`，加入以下代码来调用 `HelloAgain`：
```go
    rsp, err = c.HelloAgain(context.Background(), &pb.HelloRequest{Msg: "world"})
    if err != nil {
        log.Error(err)
    }
    log.Info(rsp.Msg)
```

按「执行示例」一节重新再执行一遍示例，你可能看到 `Hello world again!` 出现在客户端日志中。

### 下一步

- 了解 [tRPC 设计原理](https://github.com/trpc-group/trpc)。
- 阅读 [基础教程](./basics_tutorial.zh_CN.md) 来更深入地了解 tRPC-Go。
- 查阅 [API 手册](https://pkg.go.dev/trpc.group/trpc-go)。
