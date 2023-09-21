# Quick Start

English | [中文](quick_start_zh_CN.md)

## Prerequisites

- **[Go][]**: any one of the **three latest major** [releases][go-releases].
- **[trpc-go-cmdline][]**: follow the instructions in the [README][trpc-go-cmdline] to install trpc-go-cmdline and its related dependencies correctly.

## Create a Full Project Step by Step

* Copy and paste the following to `helloworld.proto`:

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

* Using [trpc-go-cmdline][] to generate a full project:
```shell
$ trpc create -p helloworld.proto -o out
```

Note: `-p` specifies proto file, `-o` specifies the output directory, 
for more information please run `trpc -h` and `trpc create -h`

* Enter the output directory and start the server:
```bash
$ cd out
$ go run .
...
... trpc service:helloworld.HelloWorldService launch success, tcp:127.0.0.1:8000, serving ...
...
```

* Open the output directory in another terminal and start the client:
```bash
$ go run cmd/client/main.go 
... simple  rpc   receive: 
```

Note: Since the implementation of server service is an empty operation and the client sends empty data, therefore the log shows that the simple rpc receives an empty string.

* Now you may try to modify the service implementation located in `hello_world_service.go` and the client implementation located in `cmd/client/main.go` to create an echo server. You can refer to [helloworld][] for inspiration.

* The generated files are explained below:

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

## Generate of RPC Stub

* Simply add `--rpconly` flag to generate rpc stub instead of a full project:
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

The following lists some frequently used flags for [trpc-go-cmdline][].

* `-f`: Force overwrite the existing code.
* `-d some-dir`: Search paths for pb files (including dependent pb files), can be specified multiple times.
* `--mock=false`: Disable generation of mock stub code.
* `--nogomod=true`: Do not generate go.mod file in the stub code, only effective when --rpconly=true, defaults to false.

For additional flags please run `trpc -h` and `trpc [subcmd] -h`.

## What's Next

Try [more features][features]. Learn more about [trpc-go-cmdline][]'s [documentation][cmdline-doc].

[Go]: https://golang.org
[go-releases]: https://golang.org/doc/devel/release.html
[trpc-go-cmdline]: https://github.com/trpc-group/trpc-go-cmdline
[cmdline-releases]: https://github.com/trpc-group/trpc-go-cmdline/releases
[helloworld]: /examples/helloworld/
[features]: /examples/features/
[cmdline-doc]: https://github.com/trpc-group/trpc-go-cmdline/tree/main/docs
