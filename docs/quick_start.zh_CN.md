[English](quick_start.md) | 中文

# 快速上手


## 安装依赖

- **[Go][]**: **最近的三个 major** [releases][go-releases].
- **[trpc-go-cmdline][]**: 正确按照 [README][trpc-go-cmdline] 来安装 trpc-go-cmdline 以及相关依赖

## 创建完整的项目

* 新建文件 `helloworld.proto` 并将以下内容进行复制粘贴:

```protobuf
syntax = "proto3";
package helloworld;

option go_package = "github.com/some-repo/examples/helloworld";

// HelloRequest is hello request.
message HelloRequest {
  string msg = 1;
}

// HelloResponse is hello response.
message HelloResponse {
  string msg = 1;
}

// HelloWorldService handles hello request and echo message.
service HelloWorldService {
  // Hello says hello.
  rpc Hello(HelloRequest) returns(HelloResponse);
}
```

* 通过以下命令使用 [trpc-go-cmdline][] 来生成一个完整的项目:
```shell
$ trpc create -p helloworld.proto -o out
```

注: `-p` 指定了 proto 文件名（相对路径）, `-o` 指定了输出位置, 
更多帮助信息可执行 `trpc -h` 以及 `trpc create -h` 以进行查看。

* 进入到输出文件夹并启动服务:
```bash
$ cd out
$ go run .
...
... trpc service:helloworld.HelloWorldService launch success, tcp:127.0.0.1:8000, serving ...
...
```

* 在另一个终端进入输出文件夹并执行客户端:
```bash
$ go run cmd/client/main.go 
... simple  rpc   receive: 
```

注：由于服务侧的默认实现为空，并且客户端发送的请求也为空，因此日志上显示收到的都是空字符串。

* 现在你可以修改位于 `hello_world_service.go` 中的服务端实现代码以及位于 `cmd/client/main.go` 中的客户端实现代码来创建一个 echo 服务。你可以参考 [helloworld][] 以获取实现灵感。

* 所有的生成文件解释如下：

```bash
$ tree
.
|-- cmd
|   `-- client
|       `-- main.go  # Generated client code.
|-- go.mod
|-- go.sum
|-- hello_world_service.go  # Generated server service implementation.
|-- hello_world_service_test.go
|-- main.go  # Server entrypoint.
|-- stub  # Stub code.
|   `-- github.com
|       `-- some-repo
|           `-- examples
|               `-- helloworld
|                   |-- go.mod
|                   |-- helloworld.pb.go
|                   |-- helloworld.proto
|                   |-- helloworld.trpc.go
|                   `-- helloworld_mock.go
`-- trpc_go.yaml  # Configuration file for trpc-go.
```

## 创建服务桩代码

* 在执行 trpc-go-cmdline 工具时直接添加 `--rpconly` 即可只生成服务桩代码：
```go
$ trpc create -p helloworld.proto -o out --rpconly
$ tree out
out
|-- go.mod
|-- go.sum
|-- helloworld.pb.go
|-- helloworld.trpc.go
`-- helloworld_mock.go
```

下面列举了 [trpc-go-cmdline][] 的一些常用的命令行选项

* `-f`: 覆盖写入输出目录
* `-d some-dir`: 指定 proto 文件的搜索路径，可以指定多次以添加多个路径
* `--mock=false`: 禁止 mockgen 生成 mock 代码
* `--nogomod=true`: 生成桩代码时不生成 `go.mod` 文件，仅在 `--rpconly=true` 时生效，默认为 `false`

更多的命令行选项可以通过运行 `trpc -h` 以及 `trpc [subcmd] -h` 以获取帮助

## 下一步

尝试 [更多特性][features]，学习更多关于 [trpc-go-cmdline][] 的 [用法][cmdline-doc]。

[Go]: https://golang.org
[go-releases]: https://golang.org/doc/devel/release.html
[trpc-go-cmdline]: https://github.com/trpc-group/trpc-go-cmdline
[cmdline-releases]: https://github.com/trpc-group/trpc-go-cmdline/releases
[helloworld]: /examples/helloworld/
[features]: /examples/features/
[cmdline-doc]: https://github.com/trpc-group/trpc-go-cmdline/tree/main/docs
