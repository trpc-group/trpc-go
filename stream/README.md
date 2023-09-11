English | [中文](README_CN.md)

# Building Stream Services with tRPC-Go

## Introduction

What is Stream:

In a regular RPC, the client sends a request to the server, waits for the server to process the request, and returns a response to the client.

In contrast, with stream RPC, the client and server can establish a continuous connection to send and receive data continuously, allowing the server to provide continuous responses.

tRPC streaming is divided into three types:

- Server-side streaming RPC
- Client-side streaming RPC
- Bidirectional streaming RPC

Why do we need streaming? Are there any issues with Simple RPC? When using Simple RPC, the following issues may arise:

- Instantaneous pressure caused by large data packets.
- When receiving data packets, all packets must be received correctly before the response is received and business processing can take place (it is not possible to receive and process data on the client and server simultaneously).

Why use Streaming RPC:

- With Simple RPC, for large data packets such as a large file that needs to be transmitted, the packets must be manually divided and reassembled, and any issues with packets arriving out of order must be resolved. In contrast, with streaming, the client can read the file and transmit it directly without the need to split the file into packets or worry about packet order.
- n real-time scenarios such as multi-person chat rooms, the server must push real-time messages to multiple clients upon receiving a message.

## Principle

See [here](https://github.com/trpc-group/trpc/blob/main/docs/cn/trpc_protocol_design.md) for the tRPC streaming design principle.

## Example

### Client-side streaming

#### Define the protocol file

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

#### Generate service code

First install [trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline).

Then generate the streaming service stub code

```shell
trpc create -p helloworld.proto
```

#### Server code

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

// SayHello Client streaming, SayHello passes pb.Greeter_SayHelloServer as a parameter, returns error
// pb.Greeter_SayHelloServer provides interfaces such as Recv() and SendAndClose() for streaming interaction.
func (s *greeterServerImpl) SayHello(gs pb.Greeter_SayHelloServer) error {
    var names []string
    for {
        // The server uses a for loop to recv and receive data from the client
        in, err := gs.Recv()
        if err == nil {
            log.Infof("receive hi, %s\n", in.Name)
        }
        // If EOF is returned, it means that the client stream has ended and the client has sent all the data
        if err == io.EOF {
            log.Infof("recveive error io eof %v\n", err)
            // SendAndClose send and close the stream
            gs.SendAndClose(&pb.HelloReply{Message: "hello " + strings.Join(names, ",")})
            return nil
        }
        // Indicates that an exception occurred in the stream and needs to be returned
        if err != nil {
            log.Errorf("receive from %v\n", err)
            return err
        }
        names = append(names, in.Name)
    }
}

func main() {
    // Create a service object, the bottom layer will automatically read the service configuration and initialize the plug-in, which must be placed in the first line of the main function, and the business initialization logic must be placed after NewServer.
    s := trpc.NewServer()
    // Register the current implementation into the service object.
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    // Start the service and block here.
    if err := s.Serve(); err != nil {
        panic(err)
    }
}
```

#### Client code

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
    // Different from a single RPC, calling SayHello does not need to pass in a request, and returns cstream for send and recv
    cstream, err := proxy.SayHello(ctx, opts...)
    if err != nil {
        log.Error("Error in stream sayHello")
        return
    }
    for i := 0; i < 10; i++ {
        // Call Send to continuously send data
        err = cstream.Send(&pb.HelloRequest{Name: "trpc-go" + strconv.Itoa(i)})
        if err != nil {
            log.Errorf("Send error %v\n", err)
            return err
        }
    }
    // The server only returns once, so call CloseAndRecv to receive
    reply, err := cstream.CloseAndRecv()
    if err == nil && reply != nil {
        log.Infof("reply is %s\n", reply.Message)
    }
    if err != nil {
        log.Errorf("receive error from server :%v", err)
    }
}
```

### Server-side streaming

#### Define the protocol file

```protobuf
service Greeter {
  // Add stream in front of HelloReply.
  rpc SayHello (HelloRequest) returns (stream HelloReply) {}
}
```

#### Server code

```go
// SayHello Server-side streaming, SayHello passes in a request and pb.Greeter_SayHelloServer as parameters, and returns an error
// b.Greeter_SayHelloServer provides Send() interface for streaming interaction
func (s *greeterServerImpl) SayHello(in *pb.HelloRequest, gs pb.Greeter_SayHelloServer) error {
    name := in.Name
    for i := 0; i < 100; i++ {
        // Continuously call Send to send the response
        gs.Send(&pb.HelloReply{Message: "hello " + name + strconv.Itoa(i)})
    }
    return nil
}
```

#### Client code

```go
func main() {
    proxy := pb.NewGreeterClientProxy(opts...)
    // The client directly fills in the parameters, and the returned cstream can be used to continuously receive the response from the server
    cstream, err := proxy.SayHello(ctx, &pb.HelloRequest{Name: "trpc-go"}, opts...)
    if err != nil {
        log.Error("Error in stream sayHello")
        return
    }
    for {
        reply, err := cstream.Recv()
        // Note that errors.Is(err, io.EOF) cannot be used here to determine the end of the stream
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

### Bidirectional streaming

#### Define the protocol file

```protobuf
service Greeter {
  rpc SayHello (stream HelloRequest) returns (stream HelloReply) {}
}
```

#### Server code

```go
// SayHello Bidirectional streaming，SayHello passes pb.Greeter_SayHelloServer as a parameter, returns error
// pb.Greeter_SayHelloServer provides interfaces such as Recv() and SendAndClose() for streaming interaction 
func (s *greeterServerImpl) SayHello(gs pb.Greeter_SayHelloServer) error {
    var names []string
    for {
        // Call Recv in a loop
        in, err := gs.Recv()
        if err == nil {
            log.Infof("receive hi, %s\n", in.Name)
        }
      
        if err == io.EOF {
            log.Infof("recveive error io eof %v\n", err)
            // EOF means that the client stream message has been sent
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

#### Client code

```go
func main() {
    proxy := pb.NewGreeterClientProxy(opts...)
    cstream, err := proxy.SayHello(ctx, opts...)
    if err != nil {
        log.Error("Error in stream sayHello %v", err)
        return
    }
    for i := 0; i < 10; i++ {
        // Keep sending messages.
        cstream.Send(&pb.HelloRequest{Name: "jesse" + strconv.Itoa(i)})
    }
    // Call CloseSend to indicate that the stream has ended.
    err = cstream.CloseSend()
    if err != nil {
        log.Infof("error is %v \n", err)
        return
    }
    for {
        // Continuously call Recv to receive server response.
        reply, err := cstream.Recv()
        if err == nil && reply != nil {
            log.Infof("reply is %s\n", reply.Message)
        }
        // Note that errors.Is(err, io.EOF) cannot be used here to determine the end of the stream.
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

## Flow control

What happens if the sender's transmission speed is too fast for the receiver to handle? This can lead to receiver overload, memory overflow, and other issues.

To solve this problem, tRPC implements a flow control feature similar to http2.0.

- RPC flow control is based on a single stream, not overall connection flow control.
- Similar to HTTP2.0, the entire flow control is based on trust in the sender.
- The tRPC sender can set the initial window size (for a single stream). During tRPC stream initialization, the window size is sent to the receiver.
- After receiving the initial window size, the receiver records it locally. For each DATA frame sent by the sender, the sender subtracts the size of the payload (excluding the frame header) from the current window size.
- If the available window size becomes less than 0 during this process, the sender cannot send the frame without splitting it (unlike HTTP2.0) and the upper layer API becomes blocked.
- After consuming 1/4 of the initial window size, the receiver sends feedback in the form of a feedback frame, carrying an incremental window size. After receiving the incremental window size, the sender adds it to the current available window size.
- For frame priority, feedback frames are given higher priority than data frames to prevent blocking due to priority issues.

Flow control is enabled by default, with a default window size of 65535. If the sender continuously sends data larger than 65535 (after serialization and compression), and the receiver does not call Recv, the sender will block. To set the maximum window size for the client to receive, use the client option `WithMaxWindowSize`.

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

If you want to set the server receiving window size, use server option `WithMaxWindowSize`

```go
s := trpc.NewServer(server.WithMaxWindowSize(1 * 1024 * 1024))
pb.RegisterGreeterService(s, &greeterServiceImpl{})
if err := s.Serve(); err != nil {
    log.Fatal(err)
}
```

## Warning

### Streaming services only support synchronous mode

When a pb file defines both ordinary RPC methods and stream methods for the same service, setting the asynchronous mode will not take effect. Only synchronous mode can be used. This is because streams only support synchronous mode. Therefore, if you want to use asynchronous mode, you must define a service with only ordinary RPC methods.

### The streaming client must use `err == io.EOF` to determine the end of the stream

It is recommended to use `err == io.EOF` to determine the end of a stream instead of `errors.Is(err, io.EOF)`. This is because the underlying connection may return `io.EOF` after disconnection, which will be encapsulated by the framework and returned to the business layer. If the business layer uses `errors.Is(err, io.EOF)` and receives a true value, it may mistakenly believe that the stream has been closed properly, when in fact the underlying connection has been disconnected and the stream has ended abnormally.

## Filter

Stream filter refers to [trpc-go/filter](/filter).
