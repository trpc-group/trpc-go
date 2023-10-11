English | [中文](README.zh_CN.md)
# tRPC-Go Client Package


## Background

User invoke RPC requests through stub code, and then the request enters client package, where the client package is responsible for the service discovery, interceptor execution, serialization, and compression, and finally, it's sent to the network via the transport package. Upon receiving a network response, the client package executes decompression, deserialization, interceptor execution, and ultimately returns the response to the user. Each step in the client package can be customized, allowing users to define their own service discovery methods, interceptors, serialization, compression, etc.

## Client Options

Users can specify different client options when making RPC requests. These options include the target address, network type, and can also specify the implementation details for each step, such as the service discovery method and serialization method.

```golang
proxy := pb.NewGreeterClientProxy()
rsp, err := proxy.Hello(
    context.Background(),
    &pb.HelloRequest{Msg: "world"},
    client.WithTarget("ip://127.0.0.1:9000"), // Specify client options
)
```

Commonly used options include:

- `WithTarget(target string)`: Set the server address.

- `WithNetwork(network string)`: Set the network type.

- `WithNamespace(ns string)`: Set the server namespace (Production/Development).

- `WithServiceName(name string)`: Set the server service name, which is used for service discovery.

- `WithTimeout(timeout time.Duration)`: Set the request timeout.

- `WithNamedFilter(name string, f filter.ClientFilter)`: Set client filter.

- `WithSerializationType(t int)`: Set the serialization type. Built-in serialization methods include protobuf, JSON, flatbuffer, and custom serialization types can be registered using `codec.RegisterSerializer`.

- `WithCompressType(t int)`: Set the compression type. Built-in compression methods include gzip, snappy, zlib, and custom compression types can be registered using `codec.RegisterCompressor`.

- `WithProtocol(s string)`: Set a custom protocol type (default is trpc), which can be registered using `codec.Register`.

## Client Configs

Users can not only pass client options when making RPC requests but also define client configs in a configuration file. Client options and client configs serve partially overlapping purposes, with client options taking precedence over client configs. When both configs and options are present, the content in the options will override the configs. Using client configs is advantageous as it allows for easy modification of settings without frequent code changes.

```yaml
client: # Client configs
  timeout: 1000 # Timeout(ms) for all requests
  namespace: Development # Server environment for all requests
  filter: # Filters for all requests
    - debuglog # Use debuglog to log request and response
  service: # Configs for requests to specific services
    - callee: trpc.test.helloworld.Greeter # callee name in the pb protocol file, can be omitted if it matches 'name' below
      name: trpc.test.helloworld.Greeter1 # Service name for service discovery
      target: ip://127.0.0.1:8000 # Server address, e.g., ip://ip:port or polaris://servicename, can be omitted if using naming discovery with name
      network: tcp # Network type for the request (tcp or udp)
      protocol: trpc # Application-layer protocol (trpc or http)
      timeout: 800 # Request timeout(ms)
      serialization: 0 # Serialization type (0-pb, 2-json, 3-flatbuffer); no need to configure by default
      compression: 1 # Compression type (1-gzip, 2-snappy, 3-zlib); no need to configure by default
```

Difference between `callee` and `name`:

**`callee` refers to the service name from the pb protocol file, formatted as `pbpackage.service`.**

For example, if the pb protocol is:

```protobuf
package trpc.a.b;
service Greeter {
    rpc SayHello(request) returns reply
}
```

Then `callee` would be `trpc.a.b.Greeter`.

**`name` refers to the service name registered in the naming service**. It corresponds to the `server.service.name` field in the target service's configuration file.

In most cases, `callee` and `name` are the same, and you only need to configure one of them. However, in scenarios like storage services where multiple instances of the same pb file are deployed, the service name registered in the naming service may differ from the pb service name. In such cases, you must configure both `callee` and `name`.

```yaml
client:
  service:
    - callee: pbpackage.service # Must configure both callee and name; callee is the pb service name used for matching client proxy and configuration
      name: polaris-service-name # Service name in the naming service used for addressing
      protocol: trpc
```

Client-generated stub code from protobuf by default includes the pb service name in the client. Therefore, when the client searches for configurations, it matches them based on the "callee" key, which is the pb service name.

On the other hand, clients generated using constructs like `redis.NewClientProxy("trpc.a.b.c")` (including all plugins under the "database" category and HTTP) use the user-provided string as the default service name. Consequently, when searching for configurations, the client utilizes the input parameter of NewClientProxy as the key (e.g., `trpc.a.b.c`).

Additionally, the framework supports finding configurations using both "callee" and "name" as keys. For example, in the following two client configurations, they share the same "callee" but have different "name":

```yaml
client:
  service:
    - callee: pbpackage.service # "callee" is the pb service name
      name: polaris-service-name1 # The service name registered in the naming service for addressing
      network: tcp
    - callee: pbpackage.service # "callee" is the pb service name
      name: polaris-service-name2 # Another service name registered in the naming service for addressing
      network: udp
```

In your code, you can use `client.WithServiceName` to find configurations using both `callee` and `name` as keys:

```golang
// proxy1 uses the first configuration, using TCP
proxy1 := pb.NewClientProxy(client.WithServiceName("polaris-service-name1"))
// proxy2 uses the second configuration, using UDP
proxy2 := pb.NewClientProxy(client.WithServiceName("polaris-service-name2"))
```

## Client Invocation Workflow

1. The user submits a request using stub code to invoke an RPC call.
2. The request enters the client package.
3. Client configurations are completed based on the specified options and configuration file information.
4. Service discovery is performed to obtain the actual service address based on the service name.
5. Interceptors are invoked, executing the pre-interceptor phase.
6. The request body is serialized, resulting in binary data.
7. The request body is compressed.
8. The complete request is packaged, including the protocol header.
9. The transport package sends a network request.
10. The transport package receives the network response.
11. The response is unpacked to obtain the protocol header and response body.
12. The response body is decompressed.
13. The response is deserialized to obtain the response structure.
14. Interceptors are invoked, executing the post-interceptor phase.
15. The response is returned to the user.
