---
title: "Quick Start"
linkTitle: "Quick Start"
weight: 10
date: 2023-08-20
description: This is a simple working example to get you started with tRPC in Go.
---

### Prerequisites

- **[Go](https://go.dev/doc/install)**, should be greater or equal than go1.18.
- **[tRPC cmdline tools](https://github.com/trpc-group/trpc-cmdline)**, to generate stub codes from protobuf.

### Get Example Code

Example code is part of tRPC-Go repo.
Clone and change directory to helloworld.
```bash
$ git clone --depth 1 git@github.com:trpc-group/trpc-go.git
$ cd trpc-go/examples/helloworld
```

### Run the Example

1. Compile and execute the server code:
   ```bash
   $ cd server && go run main.go
   ```
2. From a different terminal, compile and execute the client code:
   ```bash
   $ cd client && go run main.go
   ```
   You will see `Hello world!` displayed as a log.

Congratulations! You’ve just run a client-server application with tRPC-Go.

### Update protobuf

As you can see, service `Greeter` are defined in protobuf `./pb/helloworld.proto` as following:
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
It has only one method `Hello`, which takes `HelloRequest` as parameter and returns `HelloReply`.

Now, add a new method `HelloAgain`, with the same request and response:
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

Regenerate tRPC code by `$ make` in `./pb` directory.
The Makefile calls `trpc` which should be installed by prerequisites.

### Update and Run Server and Client

At server side `server/main.go`, add codes to implement `HelloAgain`:
```go
func (g Greeter) HelloAgain(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Infof("got HelloAgain request: %s", req.Msg)
	return &pb.HelloReply{Msg: "Hello " + req.Msg + " again!"}, nil
}
```

At client side `client/main.go`, add codes to call `HelloAgain`:
```go
	rsp, err = c.HelloAgain(context.Background(), &pb.HelloRequest{Msg: "world"})
	if err != nil {
		log.Error(err)
	}
	log.Info(rsp.Msg)
```

Follow the `Run the Example` section to re-run your example and you will see `Hello world again!` in client log.

### What's Next

- Learn how tRPC works in [What is tRPC](../../../what-is-trpc/).
- Read [basics tutorial](../basics_tutorial/) to get deeper into tRPC-Go.
- Explore the [API reference](https://pkg.go.dev/trpc.group/trpc-go).
