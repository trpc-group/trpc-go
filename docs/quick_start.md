# Getting Started with tRPC-Go



## Introduction

Hello tRPC-Go !

Now that you know [something](https://git.woa.com/trpc-go/trpc-wiki/blob/main/overview.md) about tRPC-Go, the easiest way to understand how it works is to look at a simple example.Hello World will walk you through creating a simple backend service, showing you:

- Define an RPC service with a simple SayHello method by writing protobuf.
- Generate server-side code using the tRPC tool.
- Invoke the service using the RPC method.

The complete code for this example can be found in the [examples/helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld) directory of our source code repository.

## Environment Setup

Before getting started, it is necessary to ensure that the Go environment is available. If there is no Go environment, please refer to [Environment Setup](todo).

## Server Development

Warning: This document aims to help users quickly understand the process of server-side development. The development steps mentioned here are meant to be executed locally. In actual business development, efficiency is typically improved by using better platform management tools, such as [publishing services with 123](todo) and managing PB interfaces with Rick(For more details, please refer to [tRPC-Go API management](todo) and [the introduction of Rick platform](todo)).

### Create a service repository

- In the small repository mode, each service creates a separate `Git project`, such as `git.woa.com/trpc-go/helloworld`, with a [demo](https://git.woa.com/trpc-go/helloworld) available here.

  To create a Git repository in tGit, clone it to your local machine, such as git clone `git@git.woa.com:trpc-go/helloworld.git`.

  In the large repository mode, each service is placed in a subdirectory, and it is not necessary to have the go.mod file mentioned in section 3.1.

  Alternatively, if you do not plan to commit to Git, you can create a local directory called `helloworld`.

- Initialize the `go.mod` file:

  ```shell
  cd helloworld  # Enter the service directory, and perform all future operations within this directory.
  go mod init git.woa.com/yourrtx/helloworld # Replace "yourrtx" with your own name.
  ```

### Define the service interface

tRPC uses protobuf to describe a service. We define the service methods, request parameters, and response parameters using protobuf. Enter the directory created earlier and create the following pb file,`vim helloworld.proto`:

```protobuf
syntax = "proto3";

// package The recommended content format is trpc.{app}.{server}, with trpc as the fixed prefix to identify it as a tRPC service protocol, app as your application name, and server as your service process name.
package trpc.test.helloworld;

// Waring：The go_package specified here refers to the address of the protocol generated file pb.go on Git, not the same as the Git repository address of the service above.
option go_package="git.woa.com/trpcprotocol/test/helloworld";

// Define the service interface
service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// Request parameters
message HelloRequest {
  string msg = 1;
}

// Response parameters
message HelloReply {
  string msg = 1;
}
```

Above, we have defined a `Greeter` service, which includes a `SayHello` method that accepts a `HelloRequest` parameter containing a `msg` string and returns a `HelloReply` data. Here are a few points to note:

- The `syntax` must be `proto3` because tRPC communicates exclusively using proto3.
- The recommended format for the `package` contents is `trpc.{app}.{server}`, with trpc as the fixed prefix indicating that this is a tRPC service protocol. `app` refers to the name of your application, while `server` is the name of your service process. Please note that this format is merely a recommendation from the tRPC framework and is not mandatory. However, different platforms (such as Rick) may enforce this requirement due to factors such as access control and service management. If you choose to use such a platform, you must comply with its conventions. The framework is independent of the platform, and you can decide whether or not to use the platform.
- After the `package`, there must be an `option go_package="git.woa.com/trpcprotocol/{app}/{server}";` to indicate the Git repository address for your generated pb.go file. Separating the protocol from the service makes it easy for others to reference it. Users can freely set their Git repository address or use the public group provided by tRPC-Go: [trpcprotocol](https://git.woa.com/groups/trpcprotocol/-/projects/list).
- Details on Rick interface management are available in the  [tRPC-Go API management](todo) and [the introduction of Rick platform](todo).
- When defining an `rpc` method, a `server` can have multiple `services` (grouping RPC logic), with typically one `service` per `server`. A `service` can have multiple `rpc` calls.
- When writing protobuf, you must follow the [company's protobuf specification](https://git.woa.com/standards/protobuf).

### Generate service code

- To generate service code through the `trpc` command line, the prerequisite is to [install the trpc tool](https://git.woa.com/trpc-go/trpc-go-cmdline) (the trpc-go-dev image has been installed, but you need to upgrade the trpc tool to the latest version on your own).

> Warning: Correct access to the code.oa project domain requires configuring goproxy (https://goproxy.woa.com) and ensuring that the `GONOPROXY` and `GOPRIVATE` variables in the `go env` output do not include `git.code.oa.com`. For the `trpc-go v2` version test, you can refer to the article: https://km.woa.com/group/51889/articles/show/527221. Mainly, it is necessary to add the additional command `--domain=trpc.tech --versionsuffix=v2` (to maintain compatibility, the default is still to reference trpc-go from code.oa).

```shell
# For first-time use, when using this command to generate the complete project, do not have a directory name that is the same as the pb file in the current directory. For example, if the pb name is helloworld.proto, do not have a directory name in the current directory called helloworld.
trpc create --protofile=helloworld.proto 

# Only generate rpcstub, commonly used when updating protocol fields after creating the project, to regenerate the stub code.
trpc create --protofile=helloworld.proto --rpconly

# Use the http protocol.
trpc create --protofile=helloworld.proto --protocol=http
```

- The generated code is as follows, with code in the `main.go` and `greeter.go` files:

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

// SayHello The function entry point, where users can write their logic inside the function.
// error represents an exception, such as a database connection error or an error when calling a downstream service. If "error" is returned, the content of "rsp" will no longer be returned.
// If the business encounters an error code and error message that needs to be returned, while also needing to keep "HelloReply", please design it inside "HelloReply" and return "nil" for the "error".
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    // implement business logic here ...
    // ...
	
    rsp.Msg = "Hello, I am tRPC-Go server."
	
    return nil
}

func main() {
    // Create a service object that automatically reads the service configuration and initializes the plugin at the bottom. It must be placed at the beginning of the "main" function. Business initialization logic must be placed after "NewServer".
    s := trpc.NewServer()
	
    // Register the current implementation with the service object.
    pb.RegisterGreeterService(s, &greeterServerImpl{})
	
    // Start the service and block here.
    if err := s.Serve(); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

All of the above code is automatically generated by the tool. As you can see, the server has a `greeterServerImpl` structure, which implements the service defined by protobuf through the implementation of the `SayHello` method. Now, by filling in the `rsp` structure of the `SayHello` method, we can respond to the requester with data.

Now, try modifying the value of rsp.Msg above and return your own data.

Note: The stub code generated by the above pb file is generally managed through the Rick platform. Please see the  [tRPC-Go API management](todo) and [the introduction of Rick platform](todo) for more details.

### Modify framework configuration

`vim trpc_go.yaml`

```yaml
global:  # Global Configuration
  namespace: Development  # Environment type, including two types: official Production and non-official Development.

server:  # Server Configuration
  app: test  # Application name of the business.
  server: helloworld  # Process service name.
  service:  # Services provided by the business. Multiple services can be provided.
    - name: trpc.test.helloworld.Greeter  # Routing name of the service.
      ip: 127.0.0.1  # Service listening IP address. Can use placeholder ${ip}. Choose between ip and nic, with ip as priority.
      port: 8000  # Service listening port. Can use placeholder ${port}.
      network: tcp  # Network listening type. tcp udp
      protocol: trpc  # Application layer protocol. trpc http
      timeout: 1000  # Max processing time per request in milliseconds.
```

The framework configuration provides basic parameters for service startup, including IP, port, protocol, and so on. For a detailed guide to the framework configuration, see [here](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/framework_conf.md).

Here, we have configured a `trpc protocol` service listening on `127.0.0.1:8000`.

### Start the service locally

Simply compile the binary and execute the start command locally:

```shell
# Do not use "go build main.go" because "main.go" may depend on the logic in other files in the current directory.
go build
./helloworld &
```

When the following log appears on the screen, it indicates that the service has started successfully:

```shell
xxxx-xx-xx xx:xx:xx.xxx    INFO    server/service.go:132    process:xxxx, trpc service:trpc.test.helloworld.Greeter launch success, address:127.0.0.1:8000, serving ...
```

### Self-test joint debugging tool

- Test using the tRPC-Go provided client packet tool `trpc-cli` command line, provided that the [trpc-cli](https://git.woa.com/trpc-go/trpc-cli) tool has been installed:

  ```shell
  trpc-cli -func /trpc.test.helloworld.Greeter/SayHello -target ip://127.0.0.1:8000 -body '{"msg":"hello"}'
  ```

The "trpc-cli" tool supports many parameters, so be sure to specify them correctly.

- The `func` is defined in the protobuf protocol for `/package.service/method`. In the example above, this would be `/trpc.test.helloworld.Greeter/SayHello`. `Please note: this is not the service configured in the YAML file`.
- `target` is the target address of the called service. The format is `selectorname://servicename`. For details, please refer to the [tRPC-Go client development guide](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/client/overview.md). This is just a local self-test, without access to the name service. Directly specify the ipport address and use the ip selector That's it, the format is `ip://${ip}:${port}`, such as `ip://127.0.0.1:8000`.
- The `body` is a JSON structure string for the request packet data. The internal JSON fields must match the fields defined in the protobuf file exactly, so be careful not to misspell or use incorrect capitalization.

If you want to experience the entire tRPC-Go chain and use all plugins, you can refer to the ["helloworld" demo project which covers the whole process](https://git.woa.com/trpc-go/helloworld).

During the development process, you can query the [API documentation](https://godoc.woa.com/git.woa.com/trpc-go/trpc-go) of the framework.

Warning: `trpc` and `trpc-cli` are two different tools. The former is mainly used to generate stub code corresponding to protobuf, while the latter is mainly used as a client to send requests. The wiki and git addresses for each are as follows:

- trpc: [wiki (trpc-go-cmdline tool)](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/cmdline_tool.md) https://git.woa.com/trpc-go/trpc-go-cmdline
- trpc-cli: [wiki (tRPC-Go interface testing)](todo) https://git.woa.com/trpc-go/trpc-cli

For more tools, see section [3.5 Install Common Tools] in [tRPC-Go Environment Construction](todo).

## Client development

Developing a client to call backend services using tRPC-Go is very simple. The code generated by protobuf already includes the calling method, and calling a remote interface is as easy as calling a local function. Now, let's develop a client to call the service we created earlier:

```shell
mkdir client
cd client
go mod init git.woa.com/trpc-go/client  # your own git address
vim main.go
```

```go
package main 

import (
    "context"
	
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
	
    pb "git.code.oa.com/trpcprotocol/test/helloworld" // git address of the protocol-generated file pb.go for the called service. If it has not been pushed to git, you can refer to the local path in gomod. For example, add the following line in gomod: replace "git.code.oa.com/trpcprotocol/test/helloworld" => ./your/local/proxy/codes/path
)

func main() {
    proxy := pb.NewGreeterClientProxy() // create a client calling proxy. See the client development document for further explanation.
    req :=  &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."} // fill in the request parameter
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTarget("ip://127.0.0.1:8000")) // call with the target address as the service listening address started earlier.
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

Under normal circumstances, the client code is not so simple. Typically, downstream services are called within the service. For more detailed client code, please refer to the [Client Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/client/overview.md) section in the user guide, or directly refer to the code in the [example/helloworld](example/helloworld) directory.

## Deployment online

First of all, it is important to understand that the framework is completely independent and not bound to any platform, which means that it can be deployed on any platform.

### Deployment on 123 platform

The 123 platform is a PCG container release platform, and all new services for PCG employees will be [released](todo) on this platform.

Note that using the 123 platform for deployment requires introducing the Polaris plugin. For more information, please refer to the plugin documentation: [Polaris Service Registration and Discovery](https://git.woa.com/trpc-go/trpc-naming-polaris).

### Deployment on zhiyun platform

Zhiyun is a relatively old binary release platform. First, the binary needs to be compiled and then dragged to the platform for release.

- Build: Run "go build -v" to generate a binary file
- Zhiyun release: Choose the `backend server package` and start the command: `./app -conf ../conf/trpc_go.yaml &`.

Then login to the [Zhiyun](http://yun.isd.com/index.php/package/create/) platform for packaging and release, which can be referred to as the [Zhiyun Deployment](http://tapd.oa.com/zhiyun/markdown_wikis/view/#1010125021009540855).

### Deployment on stke

Some teams [use STKE for deployment](todo) on a large scale, while others can customize pipelines as needed for deployment on STKE. It is important to pay attention to the level of support for certain capabilities, such as whether Polaris can complete registration.

### Deployment on GDP/ODP

[GDP/ODP](https://gdp.woa.com/) is the IEG Cloud Native Developer Platform, which provides online deployment and continuous operation functions for trpc.

To create a business project, services can be built using the trpc template and released for access. Specific usage information can be consulted with the GDP&ODP assistant.

## FAQ

For more questions, please refer to [tRPC-Go FAQ](todo).

