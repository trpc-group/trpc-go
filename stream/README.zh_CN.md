[English](README.md) | 中文

# tRPC-Go 搭建流式服务

# 前言

什么是流式：

单次 RPC 需要客户端发起请求，等待服务端处理完毕，再返回给客户端。
而流式 RPC 相比单次 RPC 而言，客户端和服务端建立流后可以持续不断发送数据，而服务端也可以持续不断接收数据，可以持续进行响应。

tRPC 的流式，分为三种类型：

- Server-side streaming RPC：服务端流式 RPC
- Client-side streaming RPC：客户端流式 RPC
- Bidirectional streaming RPC：双向流式 RPC

流式为什么要存在呢，是 Simple RPC 有什么问题吗？使用 Simple RPC 时，有如下问题：

- 数据包过大造成的瞬时压力
- 接收数据包时，需要所有数据包都接受成功且正确后，才能够回调响应，进行业务处理（无法客户端边发送，服务端边处理）

为什么用 Streaming RPC：

- 大数据包，例如有一个大文件需要传输，如果使用 simple RPC，得自己分包，自己组合，解决不同包的乱序问题。使用流式可以客户端读出来后，直接传输，无需分包，无需关心乱序
- 实时场景，比如多人聊天室，服务端接收到消息后，需要往多个客户端进行实时消息推送

# 原理

tRPC 流式设计原理见 [这里](https://github.com/trpc-group/trpc/blob/main/docs/cn/trpc_protocol_design.md)。

# 示例

## 客户端流式

### 定义协议文件

```protobuf
syntax = "proto3";

package trpc.test.helloworld;
option go_package="github.com/some-repo/examples/helloworld";

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello (stream HelloRequest) returns (HelloReply);
}
// The request message containing the user's name.
message HelloRequest {
  string name = 1;
}
// The response message containing the greetings
message HelloReply {
  string message = 1;
}
```

### 生成服务代码

首先安装 [trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline)

然后生成流式服务桩代码

```shell
trpc create -p helloworld.proto
```

### 服务端代码

```go
package main

import (
    "fmt"
    "io"
    "strings"
    
    "trpc.group/trpc-go/trpc-go/log"
    trpc "trpc.group/trpc-go/trpc-go"
    _ "trpc.group/trpc-go/trpc-go/stream"
    pb "github.com/some-repo/examples/helloworld"
)

type greeterServerImpl struct{}

// SayHello 客户端流式，SayHello 传入 pb.Greeter_SayHelloServer 作为参数，返回 error
// pb.Greeter_SayHelloServer 提供 Recv() 和 SendAndClose() 等接口，用作流式交互 
func (s *greeterServerImpl) SayHello(gs pb.Greeter_SayHelloServer) error {
    var names []string
    for {
        // 服务端使用 for 循环进行 Recv，接收来自客户的数据
        in, err := gs.Recv()
        if err == nil {
            log.Infof("receive hi, %s\n", in.Name)
        }
        // 如果返回 EOF，说明客户端流已经结束，客户端已经发送完所有数据
        if err == io.EOF {
            log.Infof("recveive error io eof %v\n", err)
            // SendAndClose 发送并关闭流
            gs.SendAndClose(&pb.HelloReply{Message: "hello " + strings.Join(names, ",")})
            return nil
        }
        // 说明流发生异常，需要返回
        if err != nil {
            log.Errorf("receive from %v\n", err)
            return err
        }
        names = append(names, in.Name)
    }
}

func main() {
    // 创建一个服务对象，底层会自动读取服务配置及初始化插件，必须放在 main 函数首行，业务初始化逻辑必须放在 NewServer 后面
    s := trpc.NewServer()
    // 注册当前实现到服务对象中
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    // 启动服务，并阻塞在这里
    if err := s.Serve(); err != nil {
        panic(err)
    }
}
```

### 客户端代码

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "strconv"
    
    "trpc.group/trpc-go/trpc-go/client"
    "trpc.group/trpc-go/trpc-go/log"
    pb "github.com/some-repo/examples/helloworld"
)

func main() {
    target := flag.String("ipPort", "", "ip port")
    serviceName := flag.String("serviceName", "", "serviceName")
    
    flag.Parse()
    
    var ctx = context.Background()
    opts := []client.Option{
        client.WithNamespace("Development"),
        client.WithServiceName("trpc.test.helloworld.Greeter"),
        client.WithTarget(*target),
    }
    log.Debugf("client: %s,%s", *serviceName, *target)
    proxy := pb.NewGreeterClientProxy(opts...)
    // 有别于单次 RPC，调用 SayHello 不需要传入 request，返回 cstream 用于 send 和 recv
    cstream, err := proxy.SayHello(ctx, opts...)
    if err != nil {
        log.Error("Error in stream sayHello")
        return
    }
    for i := 0; i < 10; i++ {
        // 调用 Send 进行持续发送数据
        err = cstream.Send(&pb.HelloRequest{Name: "trpc-go" + strconv.Itoa(i)})
        if err != nil {
            log.Errorf("Send error %v\n", err)
            return err
        }
    }
    // 服务端只返回一次，所以调用 CloseAndRecv 进行接收
    reply, err := cstream.CloseAndRecv()
    if err == nil && reply != nil {
        log.Infof("reply is %s\n", reply.Message)
        
    }
    if err != nil {
        log.Errorf("receive error from server :%v", err)
    }
}
```

## 服务端流式

### 定义协议文件

```protobuf
service Greeter {
  // HelloReply 前面加 stream
  rpc SayHello (HelloRequest) returns (stream HelloReply);
}
```

### 服务端代码

```go
// SayHello 服务端流式，SayHello 传入一次 request 和 pb.Greeter_SayHelloServer 作为参数，返回 error
// pb.Greeter_SayHelloServer 提供 Send() 接口，用作流式交互 
func (s *greeterServerImpl) SayHello(in *pb.HelloRequest, gs pb.Greeter_SayHelloServer) error {
    name := in.Name
    for i := 0; i < 100; i++ {
        // 持续调用 Send 进行发送响应
        gs.Send(&pb.HelloReply{Message: "hello " + name + strconv.Itoa(i)})
    }
    return nil
}

```

### 客户端代码

```go
func main() {
    proxy := pb.NewGreeterClientProxy(opts...)
    // 客户端直接填入参数，返回 cstream 可以用来持续接收服务端相应
    cstream, err := proxy.SayHello(ctx, &pb.HelloRequest{Name: "trpc-go"}, opts...)
    if err != nil {
        log.Error("Error in stream sayHello")
        return
    }
    for {
        reply, err := cstream.Recv()
        // 注意这里不能使用 errors.Is(err, io.EOF) 来判断流结束
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Infof("failed to recv: %v\n", err)
        }
        log.Infof("Greeting:%s \n", reply.Message)
    }
}
```

## 双向流式

### 定义协议文件

```protobuf
service Greeter {
  rpc SayHello (stream HelloRequest) returns (stream HelloReply) {}
}
```

### 服务端代码

```go
// SayHello 双向流式，SayHello 传入 pb.Greeter_SayHelloServer 作为参数，返回 error
// pb.Greeter_SayHelloServer 提供 Recv() 和 Send() 接口，用作流式交互 
func (s *greeterServerImpl) SayHello(gs pb.Greeter_SayHelloServer) error {
    var names []string
    for {
        // 循环调用 Recv
        in, err := gs.Recv()
        if err == nil {
            log.Infof("receive hi, %s\n", in.Name)
        }
        if err == io.EOF {
            log.Infof("recveive error io eof %v\n", err)
            // EOF 代表客户端流消息已经发送结束，
            gs.Send(&pb.HelloReply{Message: "hello " + strings.Join(names, ",")})
            return nil
        }
        if err != nil {
            log.Errorf("receive from %v\n", err)
            return err
        }
        names = append(names, in.Name)
    }
}
```

### 客户端代码

```go
func main() {
    proxy := pb.NewGreeterClientProxy(opts...)
    cstream, err := proxy.SayHello(ctx, opts...)
    if err != nil {
        log.Error("Error in stream sayHello %v", err)
        return
    }
    for i := 0; i < 10; i++ {
        // 持续发送消息
        cstream.Send(&pb.HelloRequest{Name: "jesse" + strconv.Itoa(i)})
    }
    // 调用 CloseSend 代表流已经结束
    err = cstream.CloseSend()
    if err != nil {
        log.Infof("error is %v \n", err)
        return
    }
    for {
        // 持续调用 Recv，接收服务端响应
        reply, err := cstream.Recv()
        if err == nil && reply != nil {
            log.Infof("reply is %s\n", reply.Message)
        }
        // 注意这里不能使用 errors.Is(err, io.EOF) 来判断流结束
        if err == io.EOF {
            log.Infof("recvice EOF: %v\n", err)
            break
        }
        if err != nil {
            log.Errorf("receive error from server :%v", err)
        }
    }
    if err != nil {
        log.Fatal(err)
    }
}
```

# 流控

如果发送方发送速度过快，接收方来不及处理怎么办？可能会导致接收方过载，内存超限等等
为了解决这个问题，tRPC 实现了和 http2.0 类似的流控功能

- tRPC 的流控针对单个流，不对整个连接进行流量控制
- 和 HTTP2.0 一样，整个 flow control 基于对发送方的信任
- tRPC 发送端可以设置初始的发送窗口大小（针对单个流），在 tRPC 流式初始化过程中，将这个窗口大小通告给接收方
- 接收方接受到初始窗口大小之后，记录在本地，发送端每发送一个 DATA 帧，就把这个发送窗口值减去 Data 帧有效数据的大小（payload，不包括帧头）
- 如果递减过程，如果当前可用窗口小于 0，那么将不能发送，这里不进行帧的拆分（http2.0 进行拆分），上层 API 进行阻塞
- 接收端每消费 1/4 的初始窗口大小进行 feedback，发送一个 feedback 帧，携带增量的 window size，发送端接收到这个增量 window size 之后加到本地可发送的 window 大小
- 帧分优先级，对于 feedback 的帧不做流控，优先级高于 Data 帧，防止因为优先级问题导致 feedback 帧发生阻塞

tPRC-Go 默认启用流控，目前默认窗口大小为 65535，如果连续发送超过 65535 大小的数据（序列化和压缩后），接收方没调用 Recv，则发送方会 block
如果要设置客户端接收窗口大小，使用 client option `WithMaxWindowSize`

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithMaxWindowSize(1 * 1024 * 1024),
    client.WithServiceName("trpc.test.helloworld.Greeter"),
    client.WithTarget(*target),
}
proxy := pb.NewGreeterClientProxy(opts...)
...
```

如果要设置服务端接收窗口大小，使用 server option `WithMaxWindowSize`

```go
s := trpc.NewServer(server.WithMaxWindowSize(1 * 1024 * 1024))
pb.RegisterGreeterService(s, &greeterServiceImpl{})
if err := s.Serve(); err != nil {
    log.Fatal(err)
}
```

# 注意事项

## 流式服务只支持同步模式

当 pb 里面同一个 service 既定义有普通 rpc 方法 和 流式方法时，用户自行设置启用异步模式会失效，只能使用同步模式。原因是流式只支持同步模式，所以如果想要使用异步模式的话，就必须定义一个只有普通 rpc 方法的 service。

## 流式客户端判断流结束必须使用 `err == io.EOF`

判断流结束应该明确用 `err == io.EOF`，而不是 `errors.Is(err, io.EOF)`，因为底层连接断开可能返回 `io.EOF`，框架对其封装后返回给业务层，业务判断时出现 `errors.Is(err, io.EOF) == true`，这个时候可能会误认为流被正常关闭了，实际上是底层连接断开，流是非正常结束的。

# 拦截器

流式拦截器见 [trpc-go/filter](/filter)
