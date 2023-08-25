# tRPC-Go module: client



## Background

The client is used to initiate network calls, and its responsibilities mainly include encoding and decoding, service discovery, load balancing, and circuit breaking. Each step in the client operation can be customized, such as custom encoding and decoding, custom service discovery methods, and custom load balancing algorithms.

To achieve these goals and facilitate future business expansion, a universal client needs to be provided that supports the above operations comprehensively and provides appropriate interfaces to allow for expansion.

This is the original intention and goal of tRPC-Go client design.

## Principle

Let's take a look at the overall design of the client and its relationship with other layers:

![relation.png](/.resources/module_design/client/client_en.png)

Next, let's further describe the client by combining the class diagram here.

The global default uses the same client and is concurrency-safe. The transport protocol, encoding and decoding type, routing selector, service discovery, load balancing, and circuit breaker can be specified through parameters and can be customized. As long as the interface is implemented according to the framework, seamless integration with the framework can be achieved.

For a complete example, please refer to [helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld).

## Interface definition

```go
// Client call structure
type Client interface {
    //Initiate a backend call
    Invoke(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...Option) (err error)
}
```

The definition of Client mainly uses a parameter option Option to initiate a backend call.

```go
// Options Client call parameters
type Options struct {
    Namespace   string        // Namespace needs to be specified for service discovery
    ServiceName string        // Backend service name
    Target      string        // Backend service address. By default, it uses Polaris but other targets can be supported using name://endpoint
    Timeout     time.Duration // Backend call timeout
    endpoint    string        // By default, it is equal to the service name unless the target is specified
    checkerSet  bool          // There are two ways to read packages in Transport: MsgReader and Checker. The two ways are mutually exclusive. checkerSet indicates whether the package reading method is set.
    CallOptions    []transport.RoundTripOption // Parameters for client transport to call
    Transport      transport.ClientTransport
    Codec          codec.Codec
    Selector       selector.Selector
    LoadBalancer   loadbalance.LoadBalancer
    Discovery      discovery.Discovery
    CircuitBreaker circuitbreaker.CircuitBreaker
    Filters filter.Chain
    ReqHead interface{}
    RspHead interface{}
    Node    *registry.Node
}
// Option call parameter utility function
type Option func(*Options)
```

## client initialization

Generating service code using the [trpc](https://git.woa.com/trpc-go/trpc-go-cmdline) code generation tool includes the corresponding client code. Client initialization:

```go
opts := []client.Option{
    client.WithProtocol("trpc"),
    client.WithNetwork("tcp4"),
    client.WithTarget("ip://10.100.72.229:12367"),
}
proxy := pb.NewGreeterClientProxy()
req := &pb.HelloRequest{}
rsp, err := proxy.SayHello(ctx, req, opts...)
```

The code snippet above sets the encoding and decoding using the trpc protocol, the transport protocol using TCP, and specifies the RPC server address as `10.100.72.229:12367` through the target parameter using `Option` parameters. The meaning of target is explained in the selector section below.

## Network transmission

trpc-go supports different transport protocols, currently including `TCP4`, `TCP6`,` UDP4`,` UDP6`, and will support other transport protocols such as `QUIC` and `RDMA` in the future. The framework defaults to supporting trpc and HTTP protocols in the transport layer, and you can specify whether to use TCP or UDP with `client.WithNetwork("udp")`. The code snippet below supports:

```go
opts := []client.Option{
    client.WithNetwork("udp"),
    client.WithTarget("ip://10.100.72.229:12367"),
}
proxy := pb.NewGreeterClientProxy()
req := &pb.HelloRequest{}
rsp, err := clientProxy.SayHello(ctx, req, opts...)
```

## protocol

trpc-go supports trpc and http protocols by default, you can pass:

```
client.WithProtocol("trpc"),
```

to set. At the same time, you can also set a custom protocol. Specific third-party agreements can be found [here](https://git.woa.com/trpc-go/trpc-codec).

## Selector

The selector is a load balancing implementation based on the `NamingService, integrating service discovery, load balancing, and circuit breaking` functions. The client supports different types of routing methods through a pluggable approach and also allows for customization.

The framework defaults to using [Polaris](https://git.woa.com/trpc-go/trpc-naming-polaris) as the selector, and other plugins that have been implemented include [cmlb](https://git.woa.com/trpc-go/trpc-selector-cmlb) and [cl5](https://git.woa.com/trpc-go/trpc-selector-cl5).

The following is an example of using cmlb as the routing method:

```go
opts := []client.Option{
    client.WithNetwork("tcp4"),
    client.WithTarget("cmlb://13702"),
}
proxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "client hello",
}
rsp, err := proxy.SayHello(ctx, req)
```

For specific usage, please refer to [trpc-selector-cmlb](https://git.woa.com/trpc-go/trpc-selector-cmlb) and [trpc-selector-cl5](https://git.woa.com/trpc-go/trpc-selector-cl5).

trpc supports plugins for the entire routing process as well as for individual functions such as service discovery, load balancing, and circuit breaking. The target parameter can support the following formats:

```ini
ip://ip:port
dns://domain:port
cmlb://appid
cl5://sid
ons://zkname
polaris://servicename
```

## Service Discovery

Users can customize the service discovery type based on their needs, including options such as `etcd`, `zookeeper`, and `DNS`. By accessing the service discovery server, you can obtain a list of corresponding service addresses. Assuming that the trpc etcd service discovery plugin has already been implemented, the specific usage is as follows:

```go
opts := []client.Option{
    client.WithServiceName("ETCD-NAMING-TEST1"),
    client.WithDiscoveryName("etcd-discovery"),
}
clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "client hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
log.Printf("req:%v, rsp:%v, err:%v", req, rsp, err)
```

Users have also customized to implement other types of service discovery methods, discovery detailed [interface](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/discovery).

## Load Balancing

When the service discovery returns a list of servers instead of a single address, load balancing algorithms are used to determine which address to use to communicate with the backend. This entire process is called load balancing, and trpc-go currently uses client-side load balancing. Users can customize load balancing algorithms. The algorithms provided by the framework include:

- Round robin: Selects the next address in the server list.
- Smooth weight round robin: Smooth weighted polling algorithm.
- Random: Randomly selects an address from the server list.

Users can also customize other types of load balancing algorithms, such as consistent hashing and dynamic weighting. Detailed [interfaces](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/loadbalance) are available.

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-go/naming/loadbalance/roundrobin"
)
opts := []client.Option{
    client.WithServiceName("ETCD-NAMING-TEST1"),
    client.WithDiscoveryName("etcd-discovery"),
    client.WithBalancerName("round_robin"),
}
clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "client hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
log.Printf("req:%v, rsp:%v, err:%v", req, rsp, err)
```

