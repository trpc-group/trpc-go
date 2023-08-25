tRPC-Go Client Development Guide



# Introduction

The tRPC-Go framework and plugins provide interface calls for services, allowing users to call downstream services as if they were calling local functions without worrying about the underlying implementation details. This article first introduces the capabilities that the framework can provide for service calls and the means that users can use to control the behavior of each link in the service call by sorting out the entire processing flow of service calls. Next, this article will explain how to develop and configure a client call from key links such as service calls, configuration, addressing, interceptors, and protocol selection. The article will provide development guidance for typical scenarios of service calls, especially for scenarios where the program serves as both a server and a client.

# Framework Capabilities

This section first introduces the types of service calls supported by the framework. Then, by sorting out the entire processing flow of service calls, we can understand what capabilities the framework provides for service calls and which key link behaviors can be customized. This provides a knowledge foundation for the development of clients later.

## Call Types

The tRPC-Go framework provides multiple types of service calls. We roughly divide service calls into four categories according to the protocol: built-in protocol calls, third-party protocol calls, storage protocol calls, and message queue producer calls. The interfaces for calling these protocols are all different. For example, the tRPC protocol provides a service interface defined by PB files, the interface provided by the HTTP standard service is "put(), get(), post(), delete()", and the interface provided by MySQL is "query(), exec(), transaction()". When developing a client, users need to refer to the protocol documentation to obtain interface information. You can refer to the following development guide documents:

**Built-in protocols**:
tRPC-Go provides service calls for the following built-in protocols:

- [Calling the tRPC service](https://git.woa.com/trpc-go/trpc-wiki/blob/main/quick_start.md)
- [Calling the generic HTTP RPC service](todo)
- [Calling the generic HTTP standard services](todo)

**Third-party protocols**:
tRPC-Go provides a rich set of protocol plugins for clients to integrate with third-party protocol services. At the same time, the framework also supports user-defined protocol plugins. For protocol plugin development, please refer to [here](todo protocol). Common third-party protocols include:

- [Calling the gRPC service](todo)
- [Calling the tars service](todo)

**Storage protocols**:
tRPC-Go encapsulates access to common databases by performing database operations through service access. For details, please refer to [tRPC-Go call  the Storage Service](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/storage.md)。

- [redis](https://git.woa.com/trpc-go/trpc-database/tree/master/redis)
- [mySQL](https://git.woa.com/trpc-go/trpc-database/tree/master/mysql)
- [ckv](https://git.woa.com/trpc-go/trpc-database/tree/master/ckv)
- [dcache](https://git.woa.com/trpc-go/trpc-database/tree/master/dcache)

**Message Queue**:
tRPC-Go encapsulates the producer operations of common message queues by producing messages through service access. For details, please refer to [tRPC-Go Producers Publish Messages](todo)。

- [kafka](https://git.woa.com/trpc-go/trpc-database/tree/master/kafka)
- [hippo](https://git.woa.com/trpc-go/trpc-database/tree/master/hippo)
- [rabbitMQ](https://git.woa.com/trpc-go/trpc-database/tree/master/rabbitmq)

Although the interfaces for calling each protocol are different, the framework adopts a unified service call process, allowing all service calls to reuse the same service governance capabilities, including interceptor, service addressing, monitoring reporting, and other capabilities.

## Calling Flow

Next, let's take a look at what a complete service call process looks like. The following figure shows the entire process from when the client initiates a service call request to when it receives a service response. The first row from left to right represents the process of the service request. The second row from right to left represents the process of the client processing the service response message.

![call_flow](/.resources/user_guide/client/overview/call_flow.png)

The framework provides a service call proxy (also known as "ClientProxy") for each service, which encapsulates the interface functions ("stub functions") of the service call, including the input parameters, output parameters, and error return codes of the interface. From the user's perspective, calling the stub function is the same as calling a local function.

As described in the tRPC framework overview, the framework adopts the idea of interface-based programming. The framework only provides standard interfaces, and plugins implement specific functions. As can be seen from the flowchart, the core process of service calls includes interceptor execution, service addressing, protocol processing, and network connection. Each part is implemented through plugins, and users need to select and configure plugins to complete the entire call process.

Users can choose and configure plugins through the framework configuration file and Option function options. At the same time, the framework also supports users to develop plugins to customize service call behavior. Interceptors are the most typical use case, such as custom interceptors to implement service call authentication and authorization, call quality monitoring and reporting, etc.

Service addressing is a very important part of the service call process. The addressing plugin (selector) provides service instance strategy routing selection, load balancing, and circuit breaker processing capabilities in scenarios where services are used on a large scale. It is a part that needs special attention in client development.

## Governance Capabilities

In addition to providing interface calls for various protocols, tRPC-Go also provides rich service governance capabilities for service calls, realizing integration with service governance components. Developers only need to focus on the business logic itself. The framework can implement the following service governance capabilities through plugins:

- Service addressing
- [Multi-environment routing](todo)
- [Calling timeout control](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/timeout_control.md)
- [Filter mechanism](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/interceptor.md), implementing functions such as [authentication and authorization](todo), call chain tracking, monitoring reporting, [retry hedging](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/retry_hedging.md), etc.
- [Remote logging](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/log.md)
- [Configuration center](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/business_configuration.md)
- ...

# Client Development

This section mainly explains from the perspective of code development how to initialize the client, how to call service interfaces, and how to control the behavior of service calls through parameter configuration.

## Development Modes

Client development is mainly divided into the following two modes:

- Mode 1: The program serves as both a server and a client. tRPC-Go service calls downstream client requests are the most common scenario.
- Mode 2: Non-service pure client small tools requests, commonly used in the development and operation and maintenance of small tools scenarios.

### Service Calls Client Internally

For Mode 1, when creating and starting the service, the framework configuration file will be read, and the initialization of all configured plugins will be automatically completed in trpc.NewServer(). The code example is as follows:

```go
import (
    "git.woa.com/trpc-go/trpc-go/errs"
    // The Git address for the generated protocol file pb.go of the called service and the protocol interface management can be found here: todo
    pb "git.woa.com/trpcprotocol/app/server"
)

// SayHello is the entry function for server requests. Generally, client calls are made within a service to call downstream services
// SayHello carries ctx information, and when calling downstream services within this function, ctx needs to be passed through all the way
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    // Create a client call proxy. This operation is lightweight and does not create a connection. It can be created for each request or a global proxy can be initialized. It is recommended to put it in the service impl struct for easy mock testing. For detailed demos, please refer to the framework source code examples/helloworld.
    proxy := pb.NewGreeterClientProxy()
    // In normal cases, do not specify any option parameters in the code. Use configuration for greater flexibility. If you specify options, the options have the highest priority
    reply, err := proxy.SayHi(ctx, req)
    if err != nil {
        log.ErrorContextf(ctx, "say hi fail:%v", err)
        return errs.New(10000, "xxxxx") 
    }
    rsp.Xxx = reply.Xxx
    return nil
}

func main(){
    // Create a service object, which will automatically read the service configuration and initialize the plugins. It must be placed at the beginning of the main function. The business initialization logic must be placed after NewServer
    s := trpc.NewServer()
    // Register the current implementation to the service object
    pb.RegisterService(s, &greeterServerImpl{})
    // Start the service and block here
    if err := s.Serve(); err != nil {
        panic(err)
    }
}
```

### Pure Small Client Tools

For Mode 2, client small tools do not have configuration files and need to set options to initiate backend calls. Also, there is no ctx, so trpc.BackgroundContext() must be used. Since there is no initialization of plugins through configuration files, some addressing methods need to be manually registered, such as the North Star. The code sample is as follows:

```go
import (
    "git.woa.com/trpc-go/trpc-go/client"
    pb "git.woa.com/trpcprotocol/app/server"
    pselector "git.woa.com/trpc-go/trpc-naming-polaris/selector" // You need to import the required naming service plugin code yourself
    trpc "git.woa.com/trpc-go/trpc-go"
)

// Generally, small tools start from the main function
func main {
    // Since there is no configuration file to help initialize the plugin, you need to manually initialize the North Star
    pselector.RegisterDefault()
    // Create a client call proxy
    proxy := pb.NewGreeterClientProxy() 
    // You must create ctx yourself through trpc.BackgroundContext() and pass in option parameters through code. For specific option parameters, see section 3.3
    rsp, err := proxy.SayHi(trpc.BackgroundContext(), req, client.WithTarget("ip://ip:port")) 
    if err != nil {
        log.Errorf("say hi fail:%v", err)
        return
    }
    return
}
```

Actually, most small tools can be executed using the timer mode, so it is recommended to use the timer to implement them and execute the tools in Mode 1. This way, all server functions will be automatically equipped.

## Interface Call

In the client, the framework defines a "ClientProxy" for each service, which provides stub functions for service invocation. Users can call the stub functions just like calling ordinary functions. The proxy is a lightweight structure that does not create any connection or other resources internally. The proxy call is thread-safe, and users can initialize a global proxy for each service or generate a proxy for each service call.

For different protocols, the services they provide are different. Users need to refer to the client development documentation of their respective protocols during the specific development process (see the Service Type section). Although the definitions of the interfaces are different, they all have common parts: "ClientProxy" and "Option function options". Based on the protocol type and the way the stub code is generated, they are divided into two categories: "IDL type service invocation" and "non-IDL type service invocation".

**1. IDL type service call**

For IDL-type services (such as tRPC services and generic HTTP RPC services), tools are usually used to generate client stub functions, including "ClientProxy creation function" and "interface invocation function". The function definition is roughly as follows:

```go
// Initialization function of ClientProxy
var NewHelloClientProxy = func(opts ...client.Option) HelloClientProxy{...}

// Service interface definition
type HelloClientProxy interface {
    SayHello(ctx context.Context, req *HelloRequest, opts ...client.Option) (rsp *HelloReply, err error)
}
```

The stub code provides users with the ClientProxy creation function, service interface function, and corresponding parameter definitions. Users can use these two sets of functions to call downstream services. The interface call is completed using synchronous calls. The option parameter can configure the behavior of the service call, which will be introduced in later sections. An example of a complete service call is as follows:

```go
import (
    "context"
    "git.woa.com/trpc-go/trpc-go/client"
    "git.woa.com/trpc-go/trpc-go/log"
    pb "git.woa.com/trpcprotocol/test/helloworld"
)

func main() {
    // Create ClientProxy
    proxy := pb.NewGreeterClientProxy()
    // Fill in the request parameters
    req :=  &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    // Call the service request interface
    rsp, err := proxy.SayHello(context.Background(), req, client.WithTarget("ip://127.0.0.1:8000"))
    if err != nil {
        return
    }
    // Get the request response data
    log.Debugf("response: %v", rsp)
}
```

**2. non-IDL type service call**

For non-IDL type services, the "ClientProxy" is also used to encapsulate the service invocation interface. The ClientProxy creation function and interface invocation function are usually provided by the protocol plugin. Different plugins have slightly different encapsulations of functions, and developers need to follow the usage documentation of their respective protocols. Taking the generic HTTP standard service as an example, the interface is defined as follows:

```go
// NewClientProxy creates a new ClientProrxy, with the http service name as a required parameter
var NewClientProxy = func(name string, opts ...client.Option) Client

// Generic HTTP standard service, providing four common interfaces: get, put, delete, and post
type Client interface {
    Get(ctx context.Context, path string, rspbody interface{}, opts ...client.Option) error
    Post(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
    Put(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
    Delete(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
}
```

An example of a complete service call is as follows:

```go
import (
    "git.woa.com/trpc-go/trpc-go/client"
    "git.woa.com/trpc-go/trpc-go/codec"
    "git.woa.com/trpc-go/trpc-go/http"
)

func main() {
    // Create ClientProxy
    httpCli := http.NewClientProxy("trpc.http.inews_importable",
        client.WithSerializationType(codec.SerializationTypeForm))
    // Fill in the request parameters
    req := url.Values{} // Set form parameters
    req.Add("certify", "1")
    req.Add("clientIP", ip)
    // Call the service request interface
    rsp: = &A{}
    err = httpCli.Post(ctx, "/i/getUserUid", req, rsp)
    if err != nil {
        return
    }
    // Get the request response data
    log.Debugf("response: %v", rsp)
}
```

## Option

The tRPC-Go framework provides two levels of Option function options to set Client parameters, which are "ClientProxy-level configuration" and "interface invocation-level configuration". The Option implementation uses the Functional Options Pattern design pattern, the principle of which can be found [here](https://lingchao.xin/post/functional-options-pattern-in-go.html). Option configuration is usually used as a pure client-side tool.

```go
// ClientProxy-level Option setting, the configuration takes effect every time the clientProxy is used to call the service
clientProxy := NewXxxClientProxy(option1, option2...)
// Interface invocation-level Option setting, the configuration only takes effect for this service call
rsp, err := proxy.SayHello(ctx, req, option1, option2...)
```

For scenarios where the program is both a server and a client, the system recommends using the framework configuration file to configure the Client. This can achieve decoupling of configuration and program, and facilitate configuration management. For scenarios where Option and configuration files are used in combination, the priority of configuration settings is: `interface invocation-level Option` > `ClientProxy-level Option` > `framework configuration file`.

The framework provides rich Option parameters. This article focuses on some configurations that are often used in development. For more Option configurations, please refer to [here](http://godoc.oa.com/git.woa.com/trpc-go/trpc-go/client#Option).

**1. We can set the protocol, serialization type, compression method, and service address of the service using the following parameters:**

```go
proxy.SayHello(ctx, req,
    client.WithProtocol("http"),
    client.WithSerializationType(codec.SerializationTypeJSON),
    client.WithCompressType(codec.CompressTypeGzip),
    client.WithTLS(certFile, keyFile, caFile, serverName),
    client.WithTarget("ip://127.0.0.1:8000"))
```

**2. We can get the call address of the downstream called service using the following parameters:**

```go
node := &registry.Node{}
proxy.SayHello(ctx, req, client.WithSelectorNode(node))
```

**3. We can set the pass-through information using the following parameters:**

```go
proxy.SayHello(ctx, req, client.WithMetaData(version.Header, []byte(version.Version)))
```

**4. We can get the pass-through information returned by the downstream service using the following parameters:**

```go
trpcRspHead := &trpc.ResponseProtocol{} // Different protocols correspond to different head structures
proxy.SayHello(ctx, req, client.WithRspHead(trpcRspHead))
// trpcRspHead.TransInfo is the pass-through information returned by the downstream service
```

**5. We can set the service call to be a one-way call using the following parameters:**

```go
proxy.SayHello(ctx, req, client.WithSendOnly())
```

## Commonly Used APIs

tRPC-Go uses GoDoc to manage the tRPC-Go framework API documentation. By consulting the [tRPC-Go API documentation](https://godoc.woa.com/git.woa.com/trpc-go/trpc-go), you can obtain the interface specifications, parameter meanings, and usage examples of the APIs.

For log, metrics, and config, the framework provides standard calling interfaces. Service development can only interface with the service governance system by using these standard interfaces. For example, for logs, if the standard log interface is not used and "fmt.Printf()" is used directly, log information cannot be reported to the remote log center.

- The use of logs, please refer to [here](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/log.md).
- Metrics API is [here](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/metrics.md).
- For business configuration usage, please refer to [here](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/business_configuration.md).

## Error codes

tRPC-Go has planned the data type and meaning of error codes, and has also explained the problem of common error codes. Please refer to the [tRPC-Go error code manual](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/error_codes.md) for details.

# Client Configuration

The client configuration can be configured through the "client" section in the framework configuration file, and the configuration is divided into "global service configuration" and "specified service configuration". For the specific meanings, value ranges, and default values of the configuration, please refer to [tRPC-Go Framework Configuration](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/framework_conf.md)。

The following is a typical example of client configuration:

```yaml
client:                                    # Client-side backend configuration
  timeout: 1000                            # The longest processing time for requests for all backends, in ms
  namespace: Development                   # The environment for all backends, Production for production environment, Development for testing environment
  filter:                                  # An array of interceptor configurations for all backends
    - m007                                 # Report to 007 monitoring for all backend interface requests
    - debuglog                             # It is strongly recommended to use this debuglog to print logs, which is very convenient for troubleshooting. For details, please refer to: https://git.woa.com/trpc-go/trpc-filter/tree/master/debuglog
  service:                                 # Configuration for a single backend, default values are available and can be completely unconfigured
    - callee: trpc.test.helloworld.Greeter # The service name of the backend service protocol file. If callee is the same as the name below, only one needs to be configured
      name: trpc.test.helloworld.Greeter1  # The service name of the backend service name routing. If it is registered with the Polaris name service, the target below does not need to be configured
      target: ip://127.0.0.1:8000          # The address of the backend service, ip://ip:port polaris://servicename cl5://sid cmlb://appid ons://zkname
      network: tcp                         # The network type of the backend service, tcp udp, default tcp
      protocol: trpc                       # Application layer protocol trpc http tars oidb ..., default trpc
      timeout: 800                         # The longest processing time for the current request, default 0 is not timed out
      serialization: 0                     # Serialization method 0-pb 1-jce 2-json 3-flatbuffer, do not configure by default
      compression: 1                       # Compression method 1-gzip 2-snappy 3-zlib, do not configure by default
      filter:                              # An array of interceptor configurations for a single backend
        - tjg                              # Only report to tjg for the current backend
```

The configuration items that need to be focused on are:

**1. The difference between "callee" and "name":**

"Callee" represents the Proto Service of the downstream service, in the format of "{package}.{proto service}". "Name" represents the Naming Service of the downstream service, used for service addressing. Please refer to [tRPC Terminology](todo) for the definition and difference between Proto Service and Naming Service.

According to the tRPC-Go development specification, in most cases, "callee" and "name" are the same, and users can only configure "name". For the scenario where a Proto Service is mapped to multiple Naming Services, users need to set both "callee" and "name".


Related issues can be referred to: [Framework Issue: 11. The relationship between package/service/method in pb and servicename in trpc_go.yaml with service registration and discovery, and request routing?](todo)

**2. Setting of "target":**

tRPC-Go provides two sets of addressing configurations: "Naming Service-based addressing" and "Target-based addressing". The "target" configuration can be left unconfigured, and the framework defaults to using name addressing. When "target" is configured, the framework will use Target-based addressing. Target-based addressing is mainly used to be compatible with old addressing methods, such as "cl5", "ons", "cmlb", etc. The format of "target" is:`selector://service_identifier`, such as `cl5://sid, cmlb://appid, ons://zkname`, `ip://127.0.0.1:1000`.

**3. Configuration of the protocol**

The configuration of the service protocol mainly includes the fields of "network", "protocol", "serialization", and "compression". "Network" and "protocol" should be based on the server configuration.

**4. Configuration of TLS**

For the tRPC protocol, the https, http2, and http3 protocols all support TLS configuration. A typical TLS configuration example is as follows:

```yaml
client:
  service:                               # Service of the downstream service
    - name: trpc.test.helloworld.Greeter # Routing name of the service
      network: tcp                       # Network listening type tcp udp
      protocol: trpc                     # Application layer protocol trpc http
      timeout: 1000                      # The longest processing time for requests, in milliseconds
      tls_key: client.pem                # Client key file address path. The key file should not be directly submitted to git. It should be pulled from the configuration center and stored locally at the specified path when the program starts.
      tls_cert: client.cert              # Client certificate file address path
      ca_cert: ca.cert                   # CA certificate file address path, used to verify the server certificate, call the TLS service, such as https server
      tls_server_name: xxx               # Client verifies the server service name. When calling https, it defaults to hostname
```

For pure client tools, it needs to be specified through options:


```go
proxy.SayHello(ctx, req, client.WithTLS(certFile, keyFile, caFile, serverName))
```

**5. Configuration of Filter**

The framework supports two-level filter configuration: global configuration and single service configuration, and the execution priority is: global setting > single service configuration. If there are duplicate filter between the two, only the one with the highest priority will be executed. A specific example is as follows:

```yaml
client:                                   # Client-side backend configuration
  timeout: 1000                           # The longest processing time for requests for all backends, in ms
  namespace: Development                  # The environment for all backends, Production for production environment, Development for testing environment
  filter:                                 # An array of filter configurations for all backends
    - m007                                # Report to 007 monitoring for all backend interface requests
    - debuglog                            # debuglog prints logs
  service:                                # Configuration for a single backend, default values are available and can be completely unconfigured
    - name: trpc.test.helloworld.Greeter1 # The service name of the backend service name routing. If it is registered with the Polaris name service, the target below does not need to be configured
      network: tcp                        # The network type of the backend service, tcp udp, default tcp
      protocol: trpc                      # Application layer protocol trpc http tars oidb ..., default trpc
      timeout: 800                        # The longest processing time for the current request, default 0 is not timed out
      filter:                             # An array of filter configurations for a single backend
        - tjg                             # Only report to tjg for the current backend
        - debuglog
```

For this example, the global filters are m007 and debuglog, and the filters called by the Greeter service are "tjg" and "debuglog". According to the rules described above, the interceptor execution order of Greeter is: m007 > debuglog > tjg

It is recommended to use the [debuglog filter](https://git.woa.com/trpc-go/trpc-filter/tree/master/debuglog) to print logs, which is very convenient for troubleshooting.

# Service Addressing

Service addressing is a very important part of service invocation. The framework implements service discovery, strategy routing, load balancing, and circuit breakers through plugins. The framework does not include any specific implementation, and users can introduce corresponding plugins as needed. Many functions of service addressing (such as set-based addressing, multi-environment routing, etc.) are closely related to the functions provided by the naming service. Users need to combine the naming service documentation and the corresponding plugin documentation to obtain detailed information. Polaris is a widely used naming service within the company, and the following descriptions in this section are based on the Polaris plugin.

## Namespace and Environment

The framework uses two concepts, namespace and env_name, to implement the isolation of service invocation. Namespace is usually used to distinguish between production and non-production environments, and services in two namespaces are completely isolated. env_name is only used in non-production environments to provide users with personal test environments. At the same time, the framework can also cooperate with the naming service to implement the sharing of services in multiple environments based on specific rules. Please refer to the Multi-Environment Routing section for details.

The system recommends setting the client's namespace and env_name through the framework configuration file, and using the client's namespace and env_name by default when calling services.

```yaml
global:
  # Required, usually use Production or Development
  namespace: String
  # Optional, environment name
  env_name: String
```

The framework also supports specifying the namespace and env_name of the service when calling the service. We call it specified environment service invocation. Specified environment service invocation requires turning off the service routing function (which is turned on by default). It can be set through the Option function:

```go
opts := []client.Option{
    // Namespace, if not filled in, the namespace of the environment where this service is located is used by default
    client.WithNamespace("Development"),
    // Service name
    client.WithServiceName("trpc.test.helloworld.Greeter"),
    // Set the environment of the called service
    client.WithCalleeEnvName("62a30eec"),
    // Turn off service routing
    client.WithDisableServiceRouter()
}
```

It can also be set through the framework configuration file:

```yaml
client:                                   # Client-side backend configuration
  namespace: Development                  # The environment for all backends
  service:                                # Configuration for a single backend
    - name: trpc.test.helloworld.Greeter1 # The service name of the backend service name routing
      disable_servicerouter: true         # Whether to disable service routing for a single client
      env_name: eef23fdab                 # Set the environment name of the downstream service in multiple environments. It only takes effect when disable_servicerouter is true
      namespace: Development              # The environment of the peer service
```

## Addressing Methods

As mentioned in Section 4, the framework provides two sets of addressing configurations: "Naming Service-based addressing" and "Target-based addressing". It can be set through the Option function option, and the system defaults to and recommends "Naming Service-based addressing". The Option function definition and example of Naming Service-based addressing are as follows:

### Addressing based on Naming Service

```go
// Definition of the Naming Service-based addressing interface
func WithServiceName(s string) Option

// Example code
func main() {
    opts := []client.Option{
        client.WithServiceName("trpc.app.server.service"),
    }
    rsp, err := clientProxy.SayHello(ctx, req, opts...)
}
```

### Addressing based on Target

Target-based addressing is mainly used to be compatible with the old addressing methods, such as "cl5", "cmlb", "ons", and "ip". The Option function definition and example are as follows:

```go
// Definition of the Target-based addressing interface, target format: selector://service identifier
func WithTarget(t string) Option

// Example code
func main() {
    opts := []client.Option{
        client.WithNamespace("Development"),
        client.WithTarget("ip://127.0.0.1:8000"),
    }
    rsp, err := clientProxy.SayHello(ctx, req, opts...)
}
```

"ip" and "dns" are commonly used selectors in tool-type clients, and the format of the target is: `ip://ip1:port1,ip2:port2`, which supports IP lists. The IP selector randomly selects an IP from the IP list for service invocation. The IP and DNS selectors do not depend on external naming services.

#### `ip://<ip>:<port>`

Specifies direct IP addressing, such as ip://127.1.1.1:8080. Multiple IPs can also be set, and the format is ip://ip1:port1,ip2:port2.

#### `dns://<domain>:<port>`

Specifies domain name addressing, which is commonly used for HTTP requests, such as dns://www.qq.com:80.

#### `cl5://<modid>:<cmdid>`

Compatible with the old cl5 addressing method, please refer to [here](https://git.woa.com/trpc-go/trpc-selector-cl5). Polaris has already connected to cl5, and Polaris can be used directly for addressing, such as polaris://modid:cmdid.

#### `cmlb://<appid>`

https://git.woa.com/trpc-go/trpc-selector-cmlb

#### `ons://<zkname>`

https://git.woa.com/trpc-go/trpc-selector-ons

## Multi-Environment Routing

Multi-environment routing is mainly used in the scenario of parallel development of multiple sets of test environments in development and testing environments, to achieve shared service invocation in different environments. Multi-environment routing is jointly implemented by the tRPC-Go framework, Polaris, and the 123 platform. For details, please refer to [tRPC-Go Multi-Environment Routing](todo)。

## Plugin Design

Service addressing includes service discovery, load balancing, service routing, circuit breakers, and other parts. The service discovery process can be simplified as follows:

![server_discovery](/.resources/user_guide/client/overview/server_discovery.png)

The framework combines these four modules through the "selector" and provides two plugin methods to implement service addressing:

- Overall interface: The name service is registered as a whole to the framework as a selector plugin. The advantage of the overall interface is that it is relatively simple to register with the framework. The framework does not care about the specific implementation of each module in the name service process. The plugin can control the entire process of name service addressing as a whole, which is convenient for performance optimization and logic control.
- Modular interface: Use the selector provided by the framework by default, and register service discovery, load balancing, service routing, circuit breakers, etc. to the framework, and the framework combines these modules. The advantage of the modular interface is that it is more flexible. Users can choose different modules according to their needs and freely combine them, but it will increase the complexity of plugin implementation.

Most of the clients running on the 123 platform use the Polaris selector plugin. The framework also supports users to develop new name service plugins. Please refer to [tRPC-Go Development Name Service Plugin](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/naming.md) for the development of name service plugins.

## Common Plugins

The service addressing of tRPC-Go is plugin-based, and users can use it as needed. Before using it, it is necessary to import the corresponding plugin. The common addressing plugin homepages include:

- Polaris
https://git.woa.com/trpc-go/trpc-naming-polaris
- cl5
https://git.woa.com/trpc-go/trpc-selector-cl5
- cmlb
https://git.woa.com/trpc-go/trpc-selector-cmlb
- ons
https://git.woa.com/trpc-go/trpc-selector-ons
- ip(IP direct connection scenario)
https://git.woa.com/trpc-go/trpc-go/blob/master/naming/selector/ip_selector.go#L14
- dns(Domain name resolution scenario)
https://git.woa.com/trpc-go/trpc-go/blob/master/naming/selector/ip_selector.go#L15

# Plugin Selection

For the use of plugins, we need to import the plugin in the main file and configure the plugin in the framework configuration file at the same time. Please refer to the example in [Polaris Naming Service](https://git.woa.com/tRPC-Go/trpc-naming-polaris) for how to use plugins.

The tRPC plugin ecosystem provides a rich set of plugins. How can programs choose the appropriate plugins? Here we provide some ideas for reference. We can roughly divide plugins into three categories: independent plugins, service governance plugins, and storage interface plugins.

- Independent plugins: For example, protocol, compression, serialization, local memory cache, and other plugins, their operation does not depend on external system components. The idea of this type of plugin is relatively simple, mainly based on the needs of business functions and the maturity of the plugin to make choices.
- Service governance plugins: Most service governance plugins, such as remote logs, naming services, configuration centers, etc., need to interface with external systems and have a great dependence on the microservice governance system. For the selection of these plugins, we need to clarify on what operating platform the service will ultimately run, what governance components the platform provides, which capabilities the service must interface with the platform, and which ones do not. The [tRPC-Go landing practice](todo) lists the practical solutions for the various BGs within the company and tRPC to interface, which can be used for reference.
- Storage interface plugins: Storage plugins mainly encapsulate the interface calls of mature databases, message queues, and other components in the industry and within the company. For this part of the plugin, we first need to consider the technical selection of the business, which database is more suitable for the needs of the business. Then, based on the technical selection, we can see if tRPC supports it. If not, we can choose to use the native SDK of the database or recommend that everyone contribute plugins to the tRPC community.

For detailed information about plugins, including plugin functions, usage, examples, configurations, limitations, and other information, please refer to the [tRPC Plugin Ecosystem](todo).

# Filter

tRPC-Go provides a filter mechanism, which sets up points in the context of service requests and responses, allowing businesses to insert custom processing logic at these points. The tRPC-Go [Plugin Ecosystem](todo) provides a rich set of filters, including call chain and monitoring plugins, which are also implemented through filters.

For the principle, triggering timing, execution order, and example code of custom filters, please refer to [Developing Filter Plugins in tRPC-Go](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/interceptor.md)。

# Calling Scenarios

For scenarios where the program is purely a client, the service invocation method is relatively simple. Usually, synchronous invocation is used to wait for the invocation to return directly, or a goroutine is created to synchronously invoke and wait for the return result in the goroutine. This will not be discussed here.

For scenarios where the program is both a server and a client (when the service receives an upstream request, it needs to call downstream services), it will be relatively more complex. This article provides ideas for user development according to three methods: synchronous processing, asynchronous processing, and multi-concurrency processing.

## Synchronous Processing

The typical scenario for synchronous processing is that when a service receives an upstream service request, it needs to call a downstream service and wait for the downstream service to complete the call before returning the response to the upstream.

For synchronous processing, the program can use the ctx of the downstream service call, which supports functions such as ctx logging and full-link timeout control. The code example is as follows:

```go
func (s *serverImpl) Call(ctx context.Context, req *pb.Req, rsp *pb.Rsp) error {
    ....

    // Synchronously process subsequent service calls, you can use the ctx in the service request
    proxy := redis.NewClientProxy("trpc.redis.test.service") // proxy should not be created every time, this is just an example
    val1, err := redis.String(proxy.Do(ctx, "GET", "key1")) 
    ....
    return nil
}
```

## Asynchronous Processing

The typical scenario for asynchronous processing is that when a service receives an upstream service request, it needs to return the response to the upstream first and then slowly process the downstream service call.

For asynchronous processing, the program can start a goroutine to execute subsequent service calls, but the subsequent service calls cannot use the ctx of the original service request because the original ctx will be automatically canceled after the response is returned. The subsequent service call can use [trpc.BackgroundContext()](https://git.woa.com/trpc-go/trpc-go/blob/master/trpc.go#L154) to create a new ctx, or directly use the [trpc.Go](https://git.woa.com/trpc-go/trpc-go/blob/v0.8.3/trpc_util.go#L152) utility function provided by trpc:

```go
func (s *serverImpl) Call(ctx context.Context, req *pb.Req, rsp *pb.Rsp) error {
    ....

    trpc.Go(ctx, time.Minute, func(ctx context.Context) {  // Here you can directly pass in the ctx of the request entry. trpc.Go will first clone the context and then go and recover. It will include logging, monitoring, recover, and timeout control internally.
        proxy := redis.NewClientProxy("trpc.redis.test.service")  // proxy should not be created every time, this is just an example
        val1, err := redis.String(proxy.Do(ctx, "GET", "key1")) 
    })

    // Do not wait for the downstream response, return the response directly. The ctx will be automatically canceled after the response is returned.
    ....
    return nil
}
```

## Multi-Concurrency Processing

The typical scenario for multi-concurrency calls is that when an online service receives an upstream service request, it needs to call multiple downstream services at the same time and wait for the response of all downstream services.

In this scenario, the business can start multiple goroutines to initiate requests by itself, but this is more troublesome, requiring its own waitgroup and recover. If there is no recover, the goroutines started by itself are easy to cause the service to crash. The framework encapsulates a simple multi-concurrency function [GoAndWait()](https://git.woa.com/trpc-go/trpc-go/blob/master/trpc.go#L174) for users to use.

```go
// GoAndWait encapsulates a safer multi-concurrency call, starts a goroutine and waits for all processing flows to complete, and automatically recovers.
// Return value error: the first non-nil error returned in the multi-concurrency coroutine
func GoAndWait(handlers ...func() error) error
```

Example: Assuming that the service receives a Call() request, the service needs to get the values of key1 and key2 from two backend services redis. Only after the downstream service call is completed, will the response be returned to the upstream.

```go
func (s *serverImpl) Call(ctx context.Context, req *pb.Req, rsp *pb.Rsp) error {
    var value [2]string
    proxy := redis.NewClientProxy("trpc.redis.test.service")
    if err := trpc.GoAndWait(
        func() error {
            // Assuming that the first downstream service call is to get the value of key1 from redis, since GoAndWait will wait for all goroutines to complete before exiting, ctx will not be canceled, so the ctx of the request entry can be used here. If you want to copy a new ctx, you can use `newCtx := trpc.CloneContext(ctx)` before GoAndWait
            val1, err := redis.String(proxy.Do(ctx, "GET", "key1"))
            if err != nil {
                // key1 is not critical data, it doesn't matter if it fails, you can use a fake data as a backup and return success
                value[0] = "fake1"
                return nil
            }
            log.DebugContextf(ctx, "get key1, val1:%s", val1)
            value[0] = val1
            return nil
        },
        func() error {
            // Assuming that the second downstream service call is to get the value of key2 from redis
            val2, err := redis.String(proxy.Do(ctx, "GET", "key2"))
            if err != nil {
                // key2 is critical data. If it cannot be obtained, the logic needs to be terminated in advance, so return failure here
                return errs.New(10000, "get key2 fail: "+err.Error())
            }
            log.DebugContextf(ctx, "get key2, val2:%s", val2)

            value[1] = val2
            return nil
        },
    );     err != nil { // If there is a failure in the multi-concurrency request, return the error code to the upstream service
        return err
    }
}
```

# Advanced Features

## Timeout control

The tRPC-Go framework provides a call timeout mechanism for service calls. For an introduction to the call timeout mechanism and related configurations, please refer to [tRPC-Go Timeout Control](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/timeout_control.md) 。

## Link transmission

The tRPC-Go framework provides a mechanism for passing fields between the client and server and passing them down the entire call chain. For the mechanism and usage of link transmission, please refer to [tRPC-Go Link Transmission](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/metadata_transmission.md). This feature requires protocol support for metadata distribution. The tRPC protocol, generic HTTP RPC protocol, and TAF protocol all support link transmission. For other protocols, please contact the respective protocol responsible person.

## Custom compression 

The tRPC-Go framework supports businesses to define their own compression and decompression methods. For details, please refer to [here](https://git.woa.com/tRPC-Go/tRPC-Go/blob/master/codec/compress_gzip.go).

## Custom serialization

The tRPC-Go framework allows businesses to define their own serialization and deserialization types. For specific examples, please refer to [here](https://git.woa.com/tRPC-Go/tRPC-Go/blob/master/codec/serialization_json.go).

# FAQ

todo

