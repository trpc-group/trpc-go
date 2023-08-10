# tRPC-Go Client [中文主页](README_CN.md)

## Usage:
```golang
proxy := pb.NewGreeterClientProxy()
rsp, err := proxy.SayHello(ctx, req)
if err != nil {
	log.Errorf("say hello fail:%v", err)
	return err
}
return nil
```

## Concepts:
- `proxy`: `proxy` is the client proxy stub created by trpc-go-cmdline. Client is called within the proxy. Proxy is lightweight, it's ok to create a proxy for each RPC.
- `target`: `target` is the backend address with scheme: selectorname://endpoint. It's polaris+servicename by default. Generally it doesn't need to be configured unless for self testing or compatibility with old naming services like l5, cmdb, ons etc.
- `config`: `config` is the config of the backend service.

## Backend Config Management
- Generally, backend config like backend address or db address is env related. Different env with different configs.
- Config center should be used for updating backend config without restarting the service or gray release of backend config. 
- If config center is not available, trpc_go.yaml should be used to config client (usually for self testing).
  client config should be like:

```yaml
client:                                    # Backend configurations for client calls
  timeout: 1000                            # Maximum processing time for requests to all backends
  namespace: Development                   # Environment for all backends
  filter:                                  # Array of interceptor configurations for all backends
    - m007                                 # Report 007 monitoring for all backend API requests
  service:                                 # Configuration for individual backends
    - callee: trpc.test.helloworld.Greeter # Service name from the backend's protocol file (pbpackage.service)
      name: trpc.test.helloworld.Greeter1  # Service name for backend service routing, registered in the name service
      target: ip://127.0.0.1:8000          # Backend service address, ip://ip:port polaris://servicename cl5://sid cmlb://appid ons://zkname
      network: tcp                         # Network type of the backend service (tcp udp)
      protocol: trpc                       # Application layer protocol (trpc http)
      timeout: 800                         # Maximum processing time for the current request
      serialization: 0                     # Serialization method 0-pb 2-json 3-flatbuffer (default: do not configure)
      compression: 1                       # Compression method 1-gzip 2-snappy 3-zlib (default: do not configure)
      filter:                              # Array of interceptor configurations for the individual backend
        - tjg                              # Filter reporting to tjg is added to the current backend after m007 
```

Difference between `callee` and `name` in the configuration:

* `callee` refers to the service name in the called service's protocol file, following the format of `pbpackage.service`.

For example, if the protocol is:
```protobuf
package trpc.a.b;
service Greeter {
    rpc SayHello(request) returns reply
}
```
Then the `callee` would be `trpc.a.b.Greeter`.

* `name` refers to the service name registered in the naming service (e.g., Polaris), which is the configuration value for `server.service.name` in the trpc_go.yaml file of the called service.

In most cases, `callee` and `name` are the same, and only one of them needs to be configured. However, in some scenarios, such as for storage services where multiple instances of the same pb file are deployed, the service name in the name service may differ from the pb service name. In such cases, both callee and name must be configured in the configuration file.

```yaml
client:
  service:
    - callee: pbpackage.service  # Both callee and name must be configured; callee is the pb service name for client proxy and configuration matching
      name: polaris-service-name # Service name in Polaris for addressing
      protocol: trpc
```

By default, the client stub code generated from pb fills in the pb service name. Therefore, when the client searches for the configuration, it only matches with the `callee` (pb service name).

However, for clients generated using methods like `redis.NewClientProxy("trpc.a.b.c")` (including all plugins under the database package and http), the service name is the string provided by the user. In such cases, the client searches for the configuration using the input parameter of NewClientProxy (e.g., `trpc.a.b.c`).

Starting from v0.10.0, it is possible to search for configurations using both `callee` and `name` as keys. For example, the following two client configurations share the same `callee`:

```yaml
client:
  service:
    - callee: pbpackage.service  # Both callee and name must be configured; callee is the pb service name for client proxy and configuration matching
      name: polaris-service-name1 # Service name in Polaris for addressing
      protocol: trpc
    - callee: pbpackage.service  # Both callee and name must be configured; callee is the pb service name for client proxy and configuration matching
      name: polaris-service-name2 # Service name in Polaris for addressing
      protocol: trpc
```

In code, users can use `client.WithServiceName` to search for configurations using both `callee` and `name` as keys:

```golang
// proxy1 uses the first configuration.
proxy1 := pb.NewClientProxy(client.WithServiceName("polaris-service-name1"))
// proxy2 uses the second configuration.
proxy2 := pb.NewClientProxy(client.WithServiceName("polaris-service-name2"))
```

In versions prior to v0.10.0, the above code would only find the second configuration (when multiple configurations have the same `callee`, the later one overrides the previous ones).

## Client Processing
- 1. Sets backend config.
- 2. Parses the selector by target. If target is not set, follows the addressing process of the framework itself.
- 3. Gets info of backend node selected by the selector.
- 4. Loads node config according to the node info.
- 5. Starts pre filter handling.
- 6. Serializes request body.
- 7. Compresses request body.
- 8. Encodes the whole request packet. 
- 9. Sends the request to server.
- 10. Receives the response, decodes it.
- 11. Decompresses response body.
- 12. Deserializes response body.
- 13. Starts post filter handling.