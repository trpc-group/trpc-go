# Config Tag

This example demonstrates the use of tag in the config of tRPC-Go.

## Usage

* add tag label in trpc_go.yaml

```yaml
client:                                     # Backend config for client.
  namespace: Development                    # environment type, two types: production and development.
  service:                                  # Backend config for service.
    - callee: trpc.test.helloworld.Greeter  # callee name in the pb protocol file, can be omitted if it matches 'name' below.
      name: trpc.test.helloworld.Greeter1   # Service name for service discovery.
      tag: "timeout_800"                    # Tag for the backend service, used to fine-tune routing when the callee and name are the same.
      target: ip://127.0.0.1:8000           # Server address, e.g., ip://ip:port or polaris://servicename, can be omitted if using naming discovery with name.
      network: tcp                          # Network type for the request (tcp or udp)
      protocol: trpc                        # Application-layer protocol (trpc or http)
      timeout: 800                          # Request timeout(ms)
    - callee: trpc.test.helloworld.Greeter  # callee name in the pb protocol file, can be omitted if it matches 'name' below.
      name: trpc.test.helloworld.Greeter1   # Service name for service discovery.
      tag: "timeout_1500"                   # Tag for the backend service, used to fine-tune routing when the callee and name are the same.
      target: ip://127.0.0.1:8000           # Server address, e.g., ip://ip:port or polaris://servicename, can be omitted if using naming discovery with name.
      network: tcp                          # Network type for the request (tcp or udp)
      protocol: trpc                        # Application-layer protocol (trpc or http)
      timeout: 1500                         # Request timeout(ms)
```

* run server

```go
func main() {
    // Create a server and register a service.
    s := trpc.NewServer()
    pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter1"), greeter)
    // Start serving.
    if err := s.Serve(); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

* request with different tag configs

```go
func main() {
    cfg, err := trpc.LoadConfig("../trpc_go.yaml")
    if err != nil {
        log.Fatalf("load config fail: %+v", err)
    }
    trpc.SetGlobalConfig(cfg)
    if err := trpc.Setup(cfg); err != nil {
        log.Fatalf("setup error: %+v", err)
    }
    
    // Create a client call proxy, see client development documentation for terminology.
    proxy := pb.NewGreeterClientProxy(client.WithServiceName("trpc.test.helloworld.Greeter1"))
    // Populate request parameters.
    req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    // The target address is the address listened by the previously started service, and the timeout_800 tag is used for addressing configuration.
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTag("timeout_800"))
    if err != nil {
        log.Errorf("could not greet: %v", err)
    } else {
        log.Debugf("response: %v", rsp)
    }
    // The target address is the address listened by the previously started service, and the timeout_800 tag is used for addressing configuration.
    rsp, err = proxy.SayHello(context.Background(), req, client.WithTag("timeout_1500"))
    if err != nil {
        log.Errorf("could not greet: %v", err)
    } else {
        log.Debugf("response: %v", rsp)
    }
}
```

## Explanation

For more Information, please refer to:

* [Building a Generic HTTP Standard Service with tRPC-Go](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)
