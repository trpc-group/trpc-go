[TOC]

# tRPC-Go Framework Configuration



## Foreword

The tRPC-Go framework configuration is a configuration file defined by the framework and used for framework initialization. As mentioned in [tRPC Architecture Overview](todo), the core of the tRPC framework adopts a plug-in architecture, which components all core functions, and connects all component functions in series through the interface-based programming thinking. Each component is associated with the plug-in SDK through configuration. The tRPC framework provides the `trpc_go.yaml` framework configuration file by default, which gathers the configuration of all basic components into the framework configuration file and passes it to the components when the service starts. In this way, each components don't have to manage their own configuration independently.

Through the introduction of this article, I hope to help users understand the following:
- Components of framework configuration
- How to obtain the implication, value range, and default value of configuration parameters
- How to generate and manage configuration files
- How to use the framework configuration, whether it can be configured dynamically

## How To Use

First of all, the tRPC-Go framework does not support the dynamic update of the framework configuration. After modifying the framework configuration, the user needs to **restart the service** to take effect. There are three ways to set the frame configuration.

### Use Configuration File

**The system recommend.** This method use the framework configuration file. When `NewServer()` starts, it will first parse the framework configuration file, automatically initialize all configured plug-ins, and start the service. It is recommended that other initialization logic be placed after `trpc.NewServer()` to ensure that the framework functions have been initialized. The default framework configuration file name of tRPC-Go is `trpc_go.yaml`, and the default path is the working path of the current program startup. The configuration path can be specified by the `-conf` command line parameter.

``` go
// by using framework configuration file, initialize tRPC service program
func NewServer(opt ...server.Option) *server.Server
```

### Build Configuration Data

**The system does not recommend.** This method does not require a framework configuration file, but user needs to assemble the startup parameter `Config` by himself. Please refer to [here](http://godoc.woa.com/git.woa.com/trpc-go/trpc-go#Config) for the data structure of `Config`. The disadvantage of using this method is that the flexibility of configuration changes is poor. Any configuration modification requires code changes, and the decoupling of configuration and program code cannot be achieved.

``` go
// user build cfg framework configuration data, and initialize tRPC service program
func NewServerWithConfig(cfg *Config, opt ...server.Option) *server.Server;
```

### Modify Configuration with Option

Both of these methods provide `Option` parameters to change local parameters. For the parameters provided by `Option`, please refer to [here](http://godoc.woa.com/git.woa.com/trpc-go/trpc-go/server#Option). `Option` configuration takes precedence over framework configuration file configuration and `Config` configuration data. An example of using `Option` to modify the framework configuration is as follows.

``` go
import(
    trpc "git.code.woa.com/trpc-go/trpc-go"
    server "git.code.woa.com/trpc-go/trpc-go/server"
)
func main() {
    s := trpc.NewServer(server.WithEnvName("test"), server.WithAddress("127.0.0.1:8001"))
    ......
}
```

> In the rest of this article, we will only discuss the framework configuration file pattern. Option and the meaning of the parameters in the construction configuration data mode can refer to the introduction of configuration in Section 3.


## Configuration Design

### The Overall Structure

The framework configuration file design is mainly divided into four groups.

| Group | Description |
| ------ | ------ |
| global | The global configuration defines general configurations such as environment-related. |
| server | The server configuration defines the general configuration of the program as a server, including application name, program name, configuration path, interceptor list, `Naming Service` list, etc. |
| client | The client configuration defines the general configuration of the program as a client, including interceptor list, the configuration of the `Naming Service` list to be accessed, etc. It is recommended to use the configuration center first for client configuration, and then the `client` configuration in the framework configuration file. |
| plugins | The plugin configuration collects all the configurations that use plugins. Since `plugins` use a `map` to manage out of order, the framework will randomly pass the plugin configurations to the SDK one by one at startup, and start the plugins. The plugin configuration format is determined by the plugin itself. |

### Configuration Details

``` yaml
# In the following configurations, unless otherwise specified: String type defaults to ""; Integer type defaults to 0; Boolean type defaults to false; [String] type defaults to [].

# Global configuration
global:
  # Required, usually use Production or Development
  namespace: String
  # Optional, environment name, please refer to doc [Multi-environment](todo)
  env_name: String
  # Optional, container name
  container_name: String
  # Optional, when the server IP is not configured, use this field as the default IP
  local_ip: String(ipv4 or ipv6)
  # Optional, whether to enable the set function for service discovery, the default is N (note that its type is String, not Boolean)
  enable_set: String(Y, N)
  # Optional, the name of the set group
  full_set_name: String([set name].[set region].[set group name])
  # Optional, the size of the network receiving buffer (unit B). <=0 means disabled, the default value 4096 is used when leave the field blank
  read_buffer_size: Integer

# Server configuration
server:
  # Required, the application name of the service
  app: String
  # Required, the service name of the service
  server: String
  # Optional, the path to the binary file
  bin_path: String
  # Optional, the path to the data file
  data_path: String
  # Optional, the path to the configuration file
  conf_path: String
  # Optional, network type, when the service is not configured with network, this field is valid, and the default is tcp
  network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
  # Optional, protocol type, when the service is not configured with protocol, this field is valid, and the default is trpc
  protocol: String(trpc, grpc, http, etc.)
  # Optional, interceptor configuration shared by all services
  filter: [String]
  # Required, the service list
  service:
    - # Optional, whether to prohibit inheriting the upstream timeout time, used to close the full link timeout mechanism, the default is false
      disable_request_timeout: Boolean
      # Optional, the IP address of the service monitors, if it is empty, it will try to get the network card IP, if it is still empty, use global.local_ip
      ip: String(ipv4 or ipv6)
      # Required, the service name, used for service discovery
      name: String
      # Optional, the network card bound to the service, it will take effect only when the IP is empty
      nic: String
      # Optional, the port bound to the service, it is required when the address is empty
      port: Integer
      # Optional, the address that the service listens to, use ip:port when it is empty, and ignore ip:port when it is not empty
      address: String
      # Optional, network type, when it is empty, use server.network
      network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
      # Optional, protocol type, when it is empty, use server.protocol
      protocol: String(trpc, grpc, http, etc.)
      # Optional, the timeout time for the service to process the request, in milliseconds
      timeout: Integer
      # Optional, long connection idle time, in milliseconds
      idletime: Integer
      # Optional, which regitration center to use such as polaris
      registry: String
      # Optional, list of interceptors, lower priority than server.filter
      filter: [String]
      # Optional, the TLS private key that the server needs to provide, when both tls_key and tls_cert are not empty, the TLS service will be enabled
      tls_key: String
      # Optional, the TLS public key that the server needs to provide, when both tls_key and tls_cert are not empty, the TLS service will be enabled
      tls_cert: String
      # Optional, if you enable reverse authentication, you need to provide the CA certificate of the client
      ca_cert: String
      # Optional, whether to enable asynchronous processing in the server, the default is true
      server_async: Boolean
      # Optional, when the service is in asynchronous processing mode, the maximum number of coroutines limited, if not set or <=0, use the default value: 1<<31 - 1. Asynchronous mode takes effect, synchronous mode does not take effect
      max_routines: Integer
      # Optional, enable the server to send packets in batches (writev system call), the default is false
      writev: Boolean
  # Optional, management functions frequently used by the service
  admin:
    # Optional, the IP bound by admin, the default is localhost
    ip: String
    # Optional, network card name, when the IP field is empty, it will try to get the IP from the network card
    nic: String
    # Optional, the port bound by admin, if it is 0, which is the default value, the admin function will not be enabled
    port: Integer
    # Optional, read timeout time, the unit is ms, the default is 3000ms
    read_timeout: Integer
    # Optional, write timeout time, the unit is ms, the default is 3000ms
    write_timeout: Integer
    # Optional, whether to enable TLS, currently not supported, setting it to true will directly report an error
    enable_tls: Boolean

# Client configuration
client:
  # Optional, if it is empty, use global.namespace
  namespace: String
  # Optional, network type, when the service is not configured with network, this field shall prevail
  network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
  # Optional, protocol type, when the service is not configured with protocol, this field shall prevail
  protocol: String(trpc, grpc, http, etc.)
  # Optional, interceptor configuration shared by all services
  filter: [String]
  # Optional, client timeout time, when the service is not configured with timeout, this field shall prevail, the unit is millisecond
  timeout: Integer
  # Optional, service discovery strategy, when the service is not configured with discovery, this field shall prevail
  discovery: String
  # Optional, load balancing strategy, when the service is not configured with loadbalance, this field shall prevail
  loadbalance: String
  # Optional, circuit breaker policy, when the service is not configured with circuitbreaker, this field shall prevail
  circuitbreaker: String
  # Required, list of called services
  service:
    - # Callee service name
      # If pb is used, the callee must be be consistent with the service name deifned in pb
      # Fill in at least one name and callee, if it is empty, use the name field
      callee: String
      # callee service name, Commonly used for service discovery
      # pay attention to distinguish [naming service and proto service](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/client/overview.md)
      # Fill in at least one name and callee, if it is empty, use the callee field
      name: String
      # Optional, environment name, used for service routing
      env_name: String
      # Optional, set name, used for service routing
      set_name: String
      # Optional, whether to disable service routing, the default is false, the concept of service routing can refer to here: todo
      disable_servicerouter: Boolean
      # Optional, when empty, use client.namespace
      namespace: String
      # Optional, target service, when not empty, the selector will take the information in target as the standard
      target: String(type:endpoint[,endpoint...])
      # Optional, the password of the callee service
      password: String
      # Optional, the service discovery strategy
      discovery: String
      # Optional, the load balancing strategy
      loadbalance: String
      # Optional, the circuit breaker strategy
      circuitbreaker: String
      # Optional, network type, when it is empty, use client.network
      network: String(tcp, tcp4, tcp6, udp, udp4, udp6)
      # Optional, timeout time, when it is empty, use client.timeout, the unit is millisecond
      timeout: Integer
      # Optional, protocol type, when it is empty, use client.protocol
      protocol: String(trpc, grpc, http, etc.)
      # Optional, serialization protocol, the default is -1, which is without setting
      serialization: Integer(0=pb, 1=JCE, 2=json, 3=flat_buffer, 4=bytes_flow)
      # Optional, compression protocol, the default is 0, which is no compression
      compression: Integer(0=no_compression, 1=gzip, 2=snappy, 3=zlib)
      # Optional, client private key, must be used with tls_cert
      tls_key: String
      # Optional, client public key, must be used with tls_key
      tls_cert: String
      # Optional, the server CA certificate path, when it is none, skip the authentication of the server
      ca_cert: String
      # Optional, service name when verifying TLS
      tls_server_name: String
      # Optional, list of interceptors, lower priority than client.filter
      filter: [String]
# Plugin configuration, please check the plugin document link in [Plugin Ecology](todo)
# If you want to customize the plugin, please refer to [Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/overview.md)
plugins:
  # Plugin type
  ${type}:
    # Plugin name
    ${name}:
      # Plugin detailed configuration, please refer to the description of each plugin for details
      Object
```

## Create Configuration

In Section 2, we introduced that the startup of the program initializes the framework by reading the framework configuration file. So how to generate framework configuration file? This section introduces the following three common methods.

### Create Configurations through Tools

The framework configuration file can be automatically generated by the corresponding trpc_go.yaml file when generating the server stub code through the tRPC scaffolding tool. The services defined in the PB file are automatically added to the configuration file. The tRPC scaffolding tool command is as follows.

``` go
# generate the stub code and the framework configuration file "trpc_go.yaml" through PB file
trpc create --protofile=helloworld.proto
```

It should be emphasized that the configuration generated by the tool is only a template configuration, and users need to modify the configuration content according to their own needs.

### Create Configurations through the Operation Platform

For large complex systems, the best practice is to manage the framework configuration files uniformly through the service operation platform, and the platform will generate the framework configuration files uniformly and deliver them to the machines where the program will run.
Let's take the PCG 123 platform as an example to introduce how the usual operation platform manages the framework configuration. The 123 platform is responsible for the orchestration of services and knows the basic information of the services. At the same time, the 123 platform integrates all the service governance capabilities required for service operation and can automatically generate framework configuration templates. For the configuration related to the specific environment in the configuration, the 123 platform uses `placeholders` (such as ${app} ${server}, etc.) to automatically fill in the framework configuration. When 123 publishes the service, the framework configuration will be automatically generated, and the placeholder will be automatically replaced with the specific value when the service starts.
The default configuration provided by the 123 platform can be found in [here](https://git.woa.com/wod_csc_paas/123_process_script/blob/master/trpc_go/trpc_go.yaml).


### Use Environment Variables to Substitue the Configurations

tRPC-Go also provides the golang template to generate framework configuration: it supports automatic replacement of framework configuration placeholders by reading environment variables. The environment variable method can be used in combination with chapter 5.1 or 5.2. Create a configuration file template through tools or operating platforms, and then replace the environment variable placeholders in the configuration file with environment variables.

For the use of environment variables, first use `${var}` to represent variable parameters in the configuration file, such as:

``` go
server:
  app: ${app}
  server: ${server}
  service: 
    - name: trpc.test.helloworld.Greeter
      ip: ${ip}
      port: ${port}
```

When the framework starts, it will first read the text content of the configuration file `trpc_go.yaml`. When the placeholder is recognized, the framework will automatically read the corresponding value from the environment variable.

As shown in the above configuration content, the environment variables need to be pre-set with the following data:

``` go
export app=test
export server=helloworld
export ip=1.1.1.1
export port=8888
```

Since the framework configuration will parse the `$` symbol, when configuring the user, do not include the `$` character except for placeholders, such as passwords such as redis/mysql, do not include `$`.


## Example

Please refer to the 123 platform to provide a complete set of configurations (see [here](https://git.woa.com/wod_csc_paas/123_process_script/blob/master/trpc_go/trpc_go.yaml) for the default configuration). Placeholders are used in this configuration. If you use the 123 platform to publish services, the system will automatically add the placeholders when the service startups. To replace it with a specific value, the user only needs to modify the last paragraph of the service name field. If you are not using the 123 platform, please replace the placeholders in the configuration by yourself.


## FAQ

todo

