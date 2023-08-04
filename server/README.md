# tRPC-Go Server [中文主页](README_CN.md)

## Run a server:

```golang
type greeterServerImpl struct{}

func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
	// implement business logic here ...
	// ...

	return nil
}

func main() {
	
	s := trpc.NewServer()
	
	pb.RegisterGreeterServer(s, &greeterServerImpl{})
	
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```

## Key Concepts
- `server`: `server` represents a tRPC server, it offers one or more `service`. One process, one `server`.
- `service`: `service` actually listens on a port and serves incoming requests. `service` can be configured by trpc_go.yaml file.
- `proto service`: `proto service` is an RPC service defined in a .proto file. Normally, a `proto service` is mapped to a `service`.

## Service Mapping

- If multiple `proto service` are defined in a .proto file:
  ```pb
   service hello {
       rpc SayHello(Request) returns (Response) {};
   }
   service bye {
       rpc SayBye(Request) returns (Response) {};
   }
  ```
- And multiple `service` are configured in trpc.go.yaml file:
  ```yaml
  server:
     app: test
     server: helloworld
     close_wait_time: 5000              # min waiting time when closing server for wait deregister finish
     max_close_wait_time: 60000         # max waiting time when closing server for wait requests finish
     service:
       - name: trpc.test.helloworld.Greeter1
         ip: 127.0.0.1
         port: 8000
         protocol: trpc
       - name: trpc.test.helloworld.Greeter2
         ip: 127.0.0.1
         port: 8080                               
         protocol: http
  ```
- Now, create a `server` by calling `svr := trpc.NewServer()`. 
`trpc.NewServer()` will also parse trpc_go.yaml file 
and `service` named "trpc.test.helloworld.Greeter1" and "trpc.test.helloworld.Greeter2"
will be added to this `server`.
- There would be several mappings according to different registration:
- If `pb.RegisterHelloServer(svr, helloImpl)` is called, 
both `service` will be mapped to `hello proto service`.
- If `pb.RegisterByeServer(svr.Service("trpc.test.helloworld.Greeter1"), byeImpl)` is called, 
`greeter1 service` will be mapped to `bye proto service`
- If `pb.RegisterHelloServer(svr.Service("trpc.test.helloworld.Greeter1"), helloImpl)` and
`pb.RegisterByeServer(svr.Service("trpc.test.helloworld.Greeter1"), byeImpl)` are called,
`greeter1 service` will be mapped to `hello proto service`,
`greeter2 service` will be mapped to `bye proto service`.

## Server Processing

- 1. Accepts a new connection and starts a goroutine to read data from the connection.
- 2. Reads the whole request packet, decodes it.
- 3. Gets handler from handler map that is used to handle this request.
- 4. Decompresses request body.
- 5. Sets timeout for handling this request.
- 6. Deserializes request body
- 7. Starts pre filter handling.
- 8. Actually handles this request.
- 9. Starts post filter handling.
- 10. Serializes response body.
- 11. Compresses response body.
- 12. Encodes response.
- 13. Sends response back to client.