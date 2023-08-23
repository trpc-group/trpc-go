# RESTful

The tRPC framework uses PB to define services, but providing REST-style APIs based on the HTTP protocol is still a widespread demand. Unifying RPC and REST is not an easy task. The HTTP RPC protocol of the tRPC-Go framework hopes to define the same set of PB files, so that services can be called through both RPC (i.e., through the NewXXXClientProxy provided by the stub code) and native HTTP requests. However, such HTTP calls do not comply with RESTful specifications, for example, custom routes cannot be defined, wildcards are not supported, and the response body is empty when errors occur (error information can only be put into the response header). Therefore, we additionally support RESTful protocols, and no longer try to forcibly unify RPC and REST. If the service is specified as RESTful, it is not supported to call it with stub code, but only supports http client call. The benefits of doing so are that you can provide API that complies with the RESTful specifications through protobuf annotation in the same set of PB files, and can use various plugins and filter capabilities of the tRPC framework.

## Usage

- Define a PB file that contains the service definition and RESTful annotations.
```protobuf
// file : examples/features/restful/server/pb/helloworld.proto
// Greeter service
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option (trpc.api.http) = {
      // http method is GET and path is /v1/greeter/hello/{name}
      // {name} is a path parameter , it will be mapped to HelloRequest.name
      get: "/v1/greeter/hello/{name}"
    };
  }
  ............
  ............
}
```

- Generate the stub code.
```shell
trpc create -p helloworld.proto --rpconly --gotag --alias -f -o=.
```

- Implement the service.
```go
// file : examples/features/restful/server/main.go
type greeterService struct{
    pb.UnimplementedGreeter
}

func (g greeterService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    log.InfoContextf(ctx, "[restful] Received SayHello request with req: %v", req)
    // handle request
    rsp := &pb.HelloReply{
        Message: "[restful] SayHello Hello " + req.Name,
    }
    return rsp, nil
}
```

- Register the service.
```go
// file : examples/features/restful/server/main.go
// Register Greeter service
pb.RegisterGreeterService(server, new(greeterService))
```

- config
```yaml
# file : examples/features/restful/server/trpc_go.yaml
server:                                            # server configuration.
  app: test                                        # business application name.
  server: helloworld                               # service process name.
  bin_path: /usr/local/trpc/bin/                   # paths to binary executables and framework configuration files.
  conf_path: /usr/local/trpc/conf/                 # paths to business configuration files.
  data_path: /usr/local/trpc/data/                 # paths to business data files.
  service:                                         # business service configuration，can have multiple.
    - name: trpc.test.helloworld.Greeter           # the route name of the service.
      ip: 127.0.0.1                                # the service listening ip address, can use the placeholder ${ip}, choose one of ip and nic, priority ip.
      port: 9092                                   # the service listening port, can use the placeholder ${port}.
      network: tcp                                 # the service listening network type,  tcp or udp.
      protocol: restful                            # application layer protocol. NOTE restful service this is restful.
```

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

```shell
$ go run client/main.go -conf client/trpc_go.yaml
```

* Server output

```
2023-05-10 20:31:11.628 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=16: CPU quota undefined
2023-05-10 20:31:11.629 INFO    server/service.go:164   process:2140, restful service:trpc.test.helloworld.Greeter launch success, tcp:127.0.0.1:9092, serving ...
2023-05-10 20:31:23.336 INFO    server/main.go:28       [restful] Received SayHello request with req: name:"trpc-restful"
2023-05-10 20:31:23.355 INFO    server/main.go:36       [restful] Received Message request with req: name:"messages/trpc-restful-wildcard" sub:{subfield:"wildcard"}
2023-05-10 20:31:23.356 INFO    server/main.go:44       [restful] Received UpdateMessage request with req: message_id:"123" message:{message:"trpc-restful-patch"}
2023-05-10 20:31:23.357 INFO    server/main.go:52       [restful] Received UpdateMessageV2 request with req: message_id:"123" message:"trpc-restful-patch-v2"
```

* Client output

```
2023-05-11 11:09:20.911 INFO    client/main.go:55       helloRsp : [restful] SayHello Hello trpc-restful
2023-05-11 11:09:20.912 INFO    client/main.go:66       messageWildcardRsp : [restful] Message name:messages/trpc-restful-wildcard,subfield:wildcard
2023-05-11 11:09:20.912 INFO    client/main.go:84       updateMessageRsp : [restful] UpdateMessage message_id:123,message:trpc-restful-patch
2023-05-11 11:09:20.914 INFO    client/main.go:102      updateMessageV2Rsp : [restful] UpdateMessageV2 message_id:123,message:trpc-restful-patch-v2
```
