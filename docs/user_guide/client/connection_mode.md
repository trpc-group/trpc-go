English | [中文](connection_mode.zh_CN.md)

# tRPC-Go client connection mode


## Introduction

Currently, tRPC-Go client supports various connection modes for the initiator of requests, including short connections, connection pools, and connection multiplexing. The client uses a connection pool mode by default, and users can choose different connection modes according to their needs.

<font color="red">Note: The connection pool here refers to the connection pool implemented in tRPC-Go's transport layer. The database and HTTP plugins replace the transport with open-source libraries using a plugin mode, and do not use this connection pool.</font>

## Principle and Implementation

### Short connection

The client creates a new connection for each request, and the connection is destroyed after the request is completed. In the case of a large number of requests, the throughput of the service will be greatly affected, resulting in significant performance loss.

Use cases: it is suitable for one-time requests or when the called service is an old service that does not support receiving multiple requests on one connection.

### Connection pool

The client maintains a connection pool for each downstream IP, and each request first obtains an IP from the name service, then obtains the corresponding connection pool based on the IP, and retrieves a connection from the connection pool. After the request is completed, the connection is returned to the connection pool. During the request process, this connection is exclusive and cannot be reused. The connections in the connection pool are destroyed and newly created according to a certain strategy. Binding one connection for one invocation may result in a large number of network connections when both upstream and downstream have a large scale, which creates enormous scheduling pressure and computational overhead.

Use cases: This mode can be used in almost all scenarios.
Note: Since the strategy of the connection pool queue is Last In First Out (LIFO), if the backend uses VIP addressing, it is possible to cause uneven distribution of the number of connections among different instances. In this case, it is advisable to address based on the name service as much as possible.

### Connection multiplexing

The client sends multiple requests simultaneously on the same connection, and each request is distinguished by a serial number ID. The client establishes a long connection with each downstream service node, and by default all requests are sent to the server through this long connection. The server needs to support connection reuse mode. Connection multiplexing can greatly reduce the number of connections between services, but due to TCP header blocking, when the number of concurrent requests on the same connection is too high, it may cause some delay (in milliseconds). This problem can be alleviated to some extent by increasing the number of multiplexing connections (default two connections are established for one IP).

Use cases: This mode is suitable for scenarios with extreme requirements for stability and throughput. The server needs to support single-connection asynchronous concurrent processing and the ability to distinguish requests by serial number ID, which requires certain server capabilities and protocol fields.

Warning：

- Because connection multiplexing will only establish one connection for each backend node, if the backend is in vip addressing mode (only one instance from the perspective of the client), connection multiplexing cannot be used, and the connection pool mode must be used.
- The transferred server (note that it is not your current service, but the service called by you) must support connection multiplexing, that is, each request is processed asynchronously on one connection, multiple sending and multiple receiving, otherwise, there will be a large number of timeout failures on the client side. 

## Example

### Short connection

```go
opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
		// If the default connection pool is disabled, the short connection mode will be used
		client.WithDisableConnectionPool(),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
	Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
	log.Error(err.Error())
	return 
}

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```

### Connection pool

```go
// The connection pool mode is used by default, no configuration is required
opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
	Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
	log.Error(err.Error())
	return 
}

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```

custom connection pool

```go
import "trpc.group/trpc-go/trpc-go/pool/connpool"

/*
connection pool parameters
type Options struct {
	MinIdle             int			  	// the minimum number of idle connections, periodically replenished by the background of the connection pool, 0 means no replenishment
	MaxIdle             int           	// the maximum number of idle connections, 0 means no limit, the default value of the framework is 65535
	MaxActive           int           	// the maximum number of concurrent connections available to users, 0 means no limit
	Wait                bool          	// whether to wait when the available connections reach the maximum number of concurrency, the default is false, do not wait
	IdleTimeout         time.Duration 	// idle connection timeout, 0 means no limit, the default value of the framework is 50s
	MaxConnLifetime     time.Duration 	// the maximum lifetime of the connection, 0 means no limit
	DialTimeout         time.Duration 	// establish connection timeout, the default value of the framework is 200ms
	ForceClose          bool          	// whether to forcibly close the connection after the user uses it, the default is false, and put it back into the connection pool
	PushIdleConnToTail  bool			// the way to put it back into the connection pool, the default is false, using LIFO to get idle connections
}
*/

// The parameters of the connection pool can be set through option, please refer to the documentation of trpc-go for details, the connection pool needs to be set as a global variable
var pool = connpool.NewConnectionPool(connpool.WithMaxIdle(65535))
// The connection pool mode is used by default, no configuration is required
opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
		// Set up a custom connection pool
		client.WithPool(pool),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
	Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
	log.Error(err.Error())
	return 
}

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```


#### Setting Idle Connection Timeout

For the client's connection pool mode, the framework sets a default idle timeout of 50 seconds.

* For `go-net`, the connection pool maintains a list of idle connections. The idle timeout only affects the connections in this idle list and is only triggered when the connection is retrieved next time, causing idle connections to be closed due to the idle timeout.
* For `tnet`, the idle timeout is implemented by maintaining a timer on each connection. Even if a connection is being used for a client's call, if the downstream does not return a result within the idle timeout period, the connection will still be triggered by the idle timeout and forcibly closed.

The methods to change the idle timeout are as follows:

* `go-net`

```go
import "trpc.group/trpc-go/trpc-go/pool/connpool"

func init() {
	connpool.DefaultConnectionPool = connpool.NewConnectionPool(
		connpool.WithIdleTimeout(0), // Setting to 0 disables it.
	)
}
```

tnet

```go
import (
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func init() {
	tnettrans.DefaultConnPool = connpool.NewConnectionPool(
	      connpool.WithDialFunc(tnettrans.Dial),
	      connpool.WithIdleTimeout(0), // Setting to 0 disables it.
	      connpool.WithHealthChecker(tnettrans.HealthChecker),
      )
}
```

**Note**: The server also has a default idle timeout, which is 60 seconds. This time is designed to be longer than the 50 seconds, so that under default conditions, it is the client that triggers the idle connection timeout to actively close the connection, rather than the server triggering a forced cleanup. For methods to change the server's idle timeout, see the server usage documentation.


### I/O multiplexing

```go
opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
		// Enable connection multiplexing
		client.WithMultiplexed(true),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
	Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
	log.Error(err.Error())
	return 
}

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```

Set custom Connection multiplexing

```go
/*
type PoolOptions struct {
    connectNumber int  // set the number of connections per address
    queueSize     int  // set the request queue length for each connection
    dropFull      bool // whether to discard when the queue is full
}
*/
// Connection multiplexing parameters can be set through option. For details, please refer to the documentation of trpc-go. Chengdu needs to be set as a global variable.
var m = multiplexed.New(multiplexed.WithConnectNumber(16))

opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
		// Enable connection multiplexing
		client.WithMultiplexed(true),
		client.WithMultiplexedPool(m),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
	Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
	log.Error(err.Error())
	return 
}

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```
