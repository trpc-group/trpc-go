English | [中文](README.zh_CN.md)

# tRPC-Go Server Package


## Introduction

A service process may listen on multiple ports, providing different business services on different ports. Therefore, the server package introduces the concepts of `Server`, `Service`, and `Proto service`. Typically, one process contains one `Server`, and each `Server` can contain one or more `Service`. `Services` are used for name registration, and clients use it's names for routing and sending network requests. Upon receiving a request, the server executes the business logic based on the specified `Protos service`.

- `Server`: Represents a tRPC server instance, i.e., one process.
- `Service`: Represents a logical service, i.e., a real external service that listens on a port. It corresponds one-to-one with services defined in the configuration file, with one `Server` possibly containing multiple `Service`, one for each port.
- `Proto service`: Represents a protocol service defined in a protobuf protocol file. Typically, a `Service` corresponds one-to-one with a `Proto service`, but users can also combine them arbitrarily using the `Register` method.

```golang
// Server is a tRPC server. One process, one server.
// A server may offer one or more services.
type Server struct {
    MaxCloseWaitTime time.Duration
}

// Service is the interface that provides services.
type Service interface {
    // Register registers a proto service.
    Register(serviceDesc interface{}, serviceImpl interface{}) error
    // Serve starts serving.
    Serve() error
    // Close stops serving.
    Close(chan struct{}) error
}
```

## Service Mapping Relationships

Suppose a protocol file provides a `hello service` as follows:

```protobuf
service hello {
    rpc SayHello(HelloRequest) returns (HelloReply) {};
}
```

And a configuration file specifies multiple services, each providing `trpc` and `http` protocol services:

```yaml
server: # Server configuration
  app: test # Application name
  server: helloworld # Process service name
  close_wait_time: 5000 # Minimum waiting time for service unregistration when closing, in milliseconds
  max_close_wait_time: 60000 # Maximum waiting time when closing to allow pending requests to complete, in milliseconds
  service: # Business services providing two services, listening on different ports and offering different protocols
    - name: trpc.test.helloworld.HelloTrpc # Name for the first service
      ip: 127.0.0.1 # IP address the service listens on
      port: 8000 # Port the service listens on (8000)
      protocol: trpc # Provides tRPC protocol service
    - name: trpc.test.helloworld.HelloHttp # Name for the second service
      ip: 127.0.0.1 # IP address the service listens on
      port: 8080 # Port the service listens on (8080)
      protocol: http # Provides HTTP protocol service
```

To register protocol services for different logical services:

```golang
type helloImpl struct{}

func (s *helloImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    rsp := &pb.HelloReply{}
    // implement business logic here ...
    return rsp, nil
}

func main() {
    s := trpc.NewServer()

    // Recommended: Register a proto service for each service separately
    pb.RegisterHiServer(s.Service("trpc.test.helloworld.HelloTrpc"), helloImpl)
    pb.RegisterHiServer(s.Service("trpc.test.helloworld.HelloHttp"), helloImpl)

    // Alternatively, register the same proto service for all services in the server
    pb.RegisterHelloServer(s, helloImpl)
}
```

## Server Execution Flow

1. The transport layer accepts a new connection and starts a goroutine to handle the connection's data.
2. Upon receiving a complete data packet, unpack the entire request.
3. Locate the specific handling function based on the specific proto service name.
4. Decode the request body.
5. Set an overall message timeout.
6. Decompress and deserialize the request body.
7. Call pre-interceptors.
8. Enter the business handling function.
9. Exit the business handling function.
10. Call post-interceptors.
11. Serialize and compress the response body.
12. Package the entire response.
13. Send the response back to the upstream client.
