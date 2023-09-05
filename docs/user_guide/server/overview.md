tRPC-Go Server Development Guide

# Introduction

This article outlines the considerations involved in developing a server-side program, such as:

- What protocol should the service use?
- How to define the service?
- How to choose plugins?
- How to test the service?

# Service Selection

## Built-in protocol service

The tRPC framework provides built-in support for **tRPC services**, **tRPC streaming services**, **generic HTTP RPC services** and **generic HTTP standard services**. “Generic HTTP”specifically refers to the underlying protocols of services using "http", "https", "http2", and "http3".

- What is the difference between **generic HTTP standard services** and generic **HTTP RPC services** ? Generic HTTP RPC services are a set of RPC models based on the generic HTTP protocol, designed by the tRPC framework, with protocol details encapsulated internally and completely transparent to users. Generic HTTP RPC services define services through PB IDL protocol and generate RPC stub code through scaffolding. Generic HTTP standard services are used exactly the same as the golang http standard library, with users defining handle request functions, registering http routes, and filling in http headers themselves. Standard HTTP services do not require IDL protocol files.
- What is the difference between **generic HTTP RPC services** and **tRPC services**? The only difference between generic HTTP RPC services and tRPC services is the difference in protocols. Generic HTTP RPC services use the generic HTTP protocol, while tRPC services use the tRPC private protocol. The difference is only visible within the framework and there is almost no difference in business development usage.
- What is the difference between **tRPC services** and **tRPC streaming services**? For a single RPC call, tRPC services require the client to initiate a request, wait for the server to process it, and then return the result to the client. On the other hand, tRPC streaming services support continuous data transmission between the client and server after establishing a stream connection. The two services differ in protocol format and IDL syntax.

## Scheduled task service

The scheduled task service adopts the ordinary service model and provides the ability to perform scheduled tasks, such as regularly loading cache or verifying transactions. A scheduled task service only supports one scheduled task. If there are multiple scheduled tasks, multiple scheduled task services need to be created. For a description of the functionality of the scheduled task service, please refer to the [tRPC-Go Build Scheduled Task Service](https://github.com/trpc-ecosystem/go-database/tree/main/timer).

The scheduled task service is not an RPC service and does not provide service calls to clients. The scheduled task service and RPC service do not affect each other, and an application can have multiple RPC services and multiple scheduled task services at the same time.

## Consumer service

The use case for the consumer service is for a program to consume messages from a message queue as a consumer. Like the scheduled task service, it adopts the ordinary service model and reuses the framework's service governance capabilities, such as automatic monitoring reporting, service tracing, and call chains. The service does not provide service calls to clients.

Currently, tRPC-Go supports message queues such as [kafka](https://github.com/trpc-ecosystem/go-database/tree/main/kafka), etc. The development and configuration of each message queue may vary, please refer to their respective documentation.

# Define Naming Service

After selecting the protocol for the service, we need to define the **Naming Service**, which determines the address of the service provider and the routing identifier used in the naming system for addressing. 

The Naming Service is responsible for network communication and protocol parsing. A Naming Service ultimately represents an `[ip, port, protocol]` combination for addressing. The Naming Service is defined through the "service" configuration in the "server" section of the framework configuration file.

We usually use a string to represent a Naming Service. The naming format depends on how the service model is defined in the service management platform where the service is located. This article takes the common practice of using the four-segment format `trpc.{app}.{server}.{service}` as an example.

```yaml
server:  # server configuration
  service:  # services which are provided by business server, there can be more than one
    - name: trpc.test.helloworld.Greeter1  # the route name of the service, this is an array, note the "-" sign in front of "name"
      ip: 127.0.0.1  # service listening IP address, choose either IP or NIC, IP has priority
      port: 8000  # service listening port
      network: tcp  # network listening type,tcp/udp
      protocol: trpc  # application layer protocol, trpc/http
      timeout: 1000  # maximum processing time for a request, in milliseconds
```

In this example, the routing identifier for the service is "trpc.test.helloworld.Greeter1", the protocol type is "trpc", and the address is "127.0.0.1:8000". When the program starts, it will automatically read this configuration and generate a Naming Service. If the server chooses the "service registration" plugin, the application will automatically register the "name" and "ipport" information of the Naming Service to the naming service, so that the client can use this name for addressing.

# Define Proto Service

Proto Service is a logical combination of a set of interfaces. It needs to define the package, proto service, RPC name, and data types for interface requests and responses. At the same time, Proto Service needs to be combined with Naming Service to complete the service assembly. Although there are slight differences in the registration interface provided to developers between "IDL protocol type" and "non-IDL protocol type" for service assembly, the implementation of both is consistent within the framework.

## IDL protocol type

IDL language can be described by interfaces in a neutral way, which use tools to convert IDL files into stub code in a specified language, allowing programmers to focus on business logic development. tRPC services, tRPC streaming services, and generic HTTP RPC services are all IDL protocol type services. For IDL protocol type services, the definition of Proto Service is usually divided into the following three steps:

**The following examples are all based on tRPC services**

Step 1: Use IDL language to describe the RPC interface specification and generate an IDL file. Taking tRPC service as an example, the definition of its IDL file is as follows:

```protobuf
syntax = "proto3";

package trpc.test.helloworld;
option go_package="github.com/some-repo/examples/helloworld";

service Greeter {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
    string msg = 1;
}

message HelloReply {
    string msg = 1;
}
```

Step 2: The corresponding stub code for the server and client can be generated by [trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline):

```shell
trpc create -p helloworld.proto
```

Step 3: Register the Proto Service to Naming Service to complete the service assembly:

```go
type greeterServerImpl struct{}

// SayHello is the interface processing function.
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{ Msg: "Hello, I am tRPC-Go server." }, nil
}

func main() {
    // Create Naming Service by reading the server.service configuration item in the framework configuration.
    s := trpc.NewServer()
    // Register the implementation instance of Proto Service to Naming Service.
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    // ...
}
```

For programs with only one Proto Service and Naming Service, the server generated by `trpc.NewServer()` can be used directly to map with Proto Service.

## non-IDL protocol type

For non-IDL protocol types, there still needs to be a definition and registration process for Proto Service. Usually, the framework and plugins have different levels of encapsulation for each protocol, and developers need to follow the usage documentation for each protocol when developing. Taking the generic HTTP standard service as an example, its code is as follows:

```go
// interface processing function
func handle(w http.ResponseWriter, r *http.Request) error {
    // construct Http Head
    w.WriteHeader(403)
    // construct Http Body
    w.Write([]byte("response body"))

    return nil
}

func main() {
    // create Naming Service by reading the server.service configuration item in the framework configuration
    s := trpc.NewServer()

    thttp.HandleFunc("/xxx/xxx", handle)
    // register the implementation instance of Proto Service to Naming Service
    thttp.RegisterNoProtocolService(s)
    s.Serve()
}
```

## Multi-service registry

For programs that are not in single-service mode (only one naming service and one proto service), users need to explicitly specify the mapping relationship between naming service and proto service. 

For registration of multiple services, we define two Proto Services for tRPC services as an example: `trpc.test.helloworld.Greeter` and `trpc.test.helloworld.Hello`:

```protobuf
syntax = "proto3";
package trpc.test.helloworld;
option go_package="github.com/some-repo/examples/helloworld";
service Greeter {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

service Hello {
    rpc SayHi (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
    string msg = 1;
}

message HelloReply {
    string msg = 1;
}
```

Correspondingly, two Naming Services also need to be defined: `trpc.test.helloworld.Greeter` and `trpc.test.helloworld.Hell`o`:

``` yaml
server:  # server configuration
  service:  # services which are provided by business server, there can be more than one
    - name: trpc.test.helloworld.Greeter  # the route name of the service, this is an array, note the "-" sign in front of "name"
      ip: 127.0.0.1  # service listening IP address, choose either IP or NIC, IP has priority
      port: 8000  # service listening port
      network: tcp  # network listening type,tcp/udp
      protocol: trpc  # application layer protocol, trpc/http
      timeout: 1000  # maximum processing time for a request, in milliseconds
    - name: trpc.test.helloworld.Hello  # the route name of the service, this is an array, note the "-" sign in front of "name"
      ip: 127.0.0.1  # service listening IP address, choose either IP or NIC, IP has priority
      port: 8001  # service listening port
      network: tcp  # network listening type,tcp/udp
      protocol: trpc  # application layer protocol, trpc/http
      timeout: 1000  # maximum processing time for a request, in milliseconds
```

Register the Proto Service to the Naming Service, and the name of the Naming Service needs to be specified in a multi-service scenario.

```go
func main() {
    // create Naming Service by reading the server.service configuration item in the framework configuration
    s := trpc.NewServer()
    // construct Greeter service
    pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &greeterServerImpl{})
    // construct Hello service
    pb.RegisterHelloService(s.Service("trpc.test.helloworld.Hello"), &helloServerImpl{})
    ...
}
```

## Interface management

For the built-in tRPC services, tRPC streaming services, and generic HTTP RPC services in the framework, it is recommended to follow some certain development specifications.

These three types of services all use PB files to define interfaces. In order to facilitate upstream and downstream to obtain interface information more transparently, we recommend **separating PB files from services, making them language-independent, and managing them through an independent central repository for unified version management**. Use a shared platform to manage PB files.

# Service Development

For the setup of common service types, please refer to the following links:

- [Setup tRPC service](todo)
- [Setup tRPC streaming service](todo)
- [Setup generic HTTP RPC service](todo)
- [Setup generic HTTP standard service](todo)
- [Setup gRPC service](todo)
- [Setup scheduled task service](todo)
- [Setup consumer service](todo)

Some third-party codec plugins: [trpc-ecosystem/go-codec](https://github.com/trpc-ecosystem/go-codec).

## Common APIs

For log, metrics, and config, the framework provides standard calling interfaces. Service development can only interface with the service governance system by using these standard interfaces. For example, for logs, if the standard log interface is not used and "fmt.Printf()" is used directly, log information cannot be reported to the remote log center.

tRPC-Go server configuration supports two ways of configuring services: "**through framework configuration files**" and "**function call parameters**". The priority of "function call parameters" is greater than that of "through framework configuration files". It is recommended to use the framework configuration file to configure the server first, which has the advantage of decoupling configuration and code and facilitating management.

## Error codes

tRPC-Go recommends using `errors.New()` encapsulated by tRPC-Go to return business error codes when writing server-side business logic, so that the framework can automatically report business error codes to the monitoring system. If the business customizes the error, it can only rely on the business to actively call the Metrics SDK to report the error code. For the API usage of error codes, please refer to [here](/errs).

# Framework Configuration

For the server, it is necessary to configure the "global" and "server" parts of the framework configuration. For the specific meanings, value ranges, and other information of configuration parameters, please refer to the [Framework Configuration document](/docs/user_guide/framework_conf.md). The configuration of the "plugins" part depends on the selected plugin, please refer to the "Plugin Selection" section below.

# Plugin Selection

The core of tRPC framework is to modularize framework functional plugins, and the framework core does not include specific implementations. For the use of plugins, we need to **import plugins in the main file** and **configure plugins in the framework configuration file** at the same time. It should be emphasized here that **the selection of plugins must be determined at the stage of development**. Please refer to the example in the [Polaris Naming Service](https://github.com/trpc-ecosystem/go-naming-polarismesh) for how to use plugins.

The tRPC plugin ecosystem provides a rich set of plugins. How can programs choose the appropriate plugins? Here we provide some ideas for reference. We can roughly divide plugins into three categories: independent plugins, service governance plugins, and storage interface plugins.

- Independent plugins: For example, protocol, compression, serialization, local memory cache, and other plugins, their operation does not depend on external system components. The idea of this type of plugin is relatively simple, mainly based on the needs of business functions and the maturity of the plugin to make choices.
- Service governance plugins: Most service governance plugins, such as remote logs, naming services, configuration centers, etc., need to interface with external systems and have a great dependence on the microservice governance system. For the selection of these plugins, we need to clarify on what operating platform the service will ultimately run, what governance components the platform provides, which capabilities the service must interface with the platform, and which ones do not.
- Storage interface plugins: Storage plugins mainly encapsulate the interface calls of mature databases, message queues, and other components in the industry and within the company. For this part of the plugin, we first need to consider the technical selection of the business, which database is more suitable for the needs of the business. Then, based on the technical selection, we can see if tRPC supports it. If not, we can choose to use the native SDK of the database or recommend that everyone contribute plugins to the tRPC community.

## Built-in Plugins

The framework has built-in some necessary plugins for services, which ensures that the framework can still provide normal RPC call capabilities with default plugins without setting any plugins. Users can replace the default plugins themselves.

The table below lists the default plugins provided by the framework as a server and the default behavior of the plugins.

| Plugin Type | Plugin Name  | Default Plugin | Plugin Behavior  |
| ---------- | --------- | --------  | ------------------------------------- |
| log      | Console  | Yes     | Default debug level or above logs are printed to the console, and the level can be set through configuration or API   |
| metric   | Noop     | Yes     | No metric information is reported     |
| config   | File     | Yes     | Supports users to use the interface to obtain configuration items from a specified local file   |
| registry | Noop     | Yes     | No registration or deregistration of services is performed   |

# Filter

tRPC-Go provides an interceptor (filter) mechanism, which sets up event tracking in the context of RPC requests and responses, allowing businesses to insert custom processing logic at these points. Functions such as call chain tracking and authentication and authorization are usually implemented using interceptors. Please refer to the [trpc-ecosystem/go-filter](https://github.com/trpc-ecosystem/go-filter) for commonly used interceptors.

The business can customize filter. Filter are usually combined with plugins to implement functions. Plugins provide configuration, while interceptors are used to insert processing logic into the RPC call context. For the principle, triggering timing, execution order, and example code of custom interceptors, please refer to [Developing Filter Plugins in tRPC-Go](/docs/developer_guide/develop_plugins/interceptor.md).

# Testing Related

tRPC-Go has considered the testability of the framework from the beginning of the design. When generating stub code through pb, mock code is generated by default.

# Advanced Features

## Timeout control

tRPC-Go provides three timeout mechanisms for RPC calls: link timeout, message timeout, and call timeout. For an introduction to the principles and related configurations of these three timeout mechanisms, please refer to [tRPC-Go Timeout Control](/docs/user_guide/timeout_control.md).

This feature requires protocol support (the protocol needs to carry timeout metadata downstream). The tRPC protocol, generic HTTP RPC protocol all support timeout control.

## Link transmission

The tRPC-Go framework provides a mechanism for passing fields between the client and server and passing them down the entire call chain. For the mechanism and usage of link transmission, please refer to [tRPC-Go Link Transmission](/docs/user_guide/metadata_transmission.md).

This feature requires protocol support for metadata distribution. The tRPC protocol, generic HTTP RPC protocol all support link transmission. 

## Reverse proxy

tRPC-Go provides a mechanism for programs that act as reverse proxies to complete the transparent transmission of binary body data without serialization and deserialization processing to improve forwarding efficiency. For the principles and example programs of reverse proxies, please refer to [tRPC-Go Reverse Proxy](/docs/user_guide/reverse_proxy.md).

## Custom compression method

tRPC-Go allows businesses to define and register compression and decompression algorithms for custom RPC message body compression and decompression. For specific examples, please refer to [here](/codec/compress_gzip.go).

## Custom serialization method

tRPC-Go allows businesses to define and register serialization and deserialization algorithms for custom RPC message body serialization and deserialization. For specific examples, please refer to [here](/codec/serialization_json.go).

## Setting the maximum number of service coroutines

tRPC-Go supports service-level synchronous/asynchronous packet processing modes. For asynchronous mode, a coroutine pool is used to improve coroutine usage efficiency and performance. Users can set the maximum number of service coroutines through framework configuration and Option configuration. For details, please refer to the service configuration in the [tPRC-Go Framework Configuration](/docs/user_guide/framework_conf.md) section.
