tRPC-Go Plugin Development Guide



# Introduction

tRPC-Go is designed with a plugin architecture, which implements the integration of the framework core and various ecological systems (including services of different protocols and various service governance components) through plugins, providing openness and scalability to the framework. This article explains how to develop a plugin from three aspects: plugin model, plugin registration, and interface implementation.

# Plugin Model

The tRPC-Go framework adopts the idea of interface-based programming, abstracting the framework functions into a series of functional components, defining standard interfaces for the components, and implementing specific functions through plugins. The framework is responsible for connecting these plugin components and assembling the complete framework functions. We can divide the plugin model into the following three layers:

- Framework design layer: The framework only defines standard interfaces without any plugin implementation, completely decoupled from the platform;
- plugin implementation layer: The plugins can be implemented by encapsulating them according to the framework standard interfaces;
- User usage layer: Business development only needs to import the plugins they need, take them as needed, and use them directly.

**Framework**: The standard interfaces provided by the framework are divided into the following five categories according to functional components:

- Configuration: Provides standard interfaces for obtaining configuration data from various data sources such as local files and configuration centers, and provides configuration parsing in various formats such as JSON and YAML. The framework also provides a watch mechanism to achieve dynamic configuration updates.
- Logging: Provides a unified interface for logging and log reporting. The logging plugin completes the integration with the remote logging system by implementing the log reporting interface.
- Protocol: Provides standard interfaces for protocol encoding and decoding, allowing protocol processing to be extended through plugins, such as business protocols, serialization types, and data compression methods.
- Naming Service: Includes standard interfaces for service registration, service discovery, policy routing, load balancing, and circuit breaking, used to implement service routing and addressing.
- Filter: Provides a common interceptor interface, allowing users to set up burials in the context of service calls to implement functions such as model call monitoring, cross-cutting logs, link tracking, and overload protection.

**Framework**: The framework manages all plugins through the "plugin factory", and each plugin needs to be registered with the plugin factory. The plugin factory adopts a two-level management mode: the first level is the plugin type (such as log, conf, selector, etc.), and the second level is the plugin name (such as there are many plugins for conf, including Rainbow Stone, Tconf, local configuration, database, etc.). The framework only provides the plugin management mode and initialization process. For the type of plugin, the framework does not impose any restrictions, and users can add plugin types themselves.

![plugin_factory](/.resources/developer_guide/develop_plugins/overview/plugin_factory.png)

**Plugin**: Plugins are bridges that connect the framework core and external service governance components. On one hand, the plugin needs to implement the plugin according to the framework standard interface, register it with the framework core, and complete the plugin instantiation. On the other hand, the plugin needs to call the SDK/API of the external service governance service to implement service governance functions such as service discovery, load balancing, monitoring, and call chains. The following diagram shows a typical implementation of a naming system.

![plugin](/.resources/developer_guide/develop_plugins/overview/plugin.png)

# Plugin Registration

All plugins in the framework are managed through the plugin factory, and all plugins need to be registered with the plugin factory. The framework provides two ways to register plugins: with configuration files and without configuration files.

## With Configuration Files

For most plugins, configuration is required for instantiation. For example, the remote logging plugin needs to know the logging method (file, console, or remote logging service) when initializing. For these types of plugins, the framework has created a configuration area in the framework configuration file for plugin configuration settings. The configuration file format is as follows:

```yaml
# Plugin configuration
plugins:
  # Plugin type
  log:
    # Plugin name
    logger1:
      # Plugin detailed configuration, please refer to the instructions of each plugin for details
      ....
    logger2:
      # Plugin detailed configuration, please refer to the instructions of each plugin for details
      ....
  # Plugin type
  config:
    # Plugin name
    rainbow:
      # Plugin detailed configuration, please refer to the instructions of each plugin for details
      ....
    tconf:
      # Plugin detailed configuration, please refer to the instructions of each plugin for details
      ....
```

Here, two plugin types and four plugins are defined: under the log plugin type, there are logger1 and logger2 log plugins. Under the configuration plugin type, there are Rainbow Stone and Tconf two configuration center plugins.

The configuration of the plugin is defined by each plugin itself, and the framework does not use or understand the meaning of the plugin configuration. Each plugin needs to implement the parsing of plugin configuration and the initialization of the plugin itself. plugin registration needs to implement the following interface:

```go
// Factory is the unified abstraction of the plugin factory. External plugins need to implement this interface to generate specific plugins through this factory interface and register them in specific plugin types.
type Factory interface {
    // Type is the type of the plugin, such as selector, log, config, tracing, etc.
    Type() string
    // Setup loads the plugin based on the configuration node. Users need to define the specific plugin configuration data structure themselves.
    Setup(name string, configDec Decoder) error
}

// Register registers the plugin factory. You can specify the plugin name yourself, and support registering different factory instances with the same implementation but different configurations.
func Register(name string, p Factory)
```

Then, call the registration function in the init() function of the plugin to add the plugin to the corresponding plugin type under the plugin factory. For example:

``` go
// Register the plugin to the plugin factory, with the plugin type defined in tconfPlugin's Type() function.
plugin.Register("tconf", &tconfPlugin{})
```

Before using the plugin, you need to import the plugin package in the main package, for example:

``` go
package main

import (
    _ "git.code.oa.com/trpc-go/trpc-config-tconf"
)
```

The import of plugin configuration occurs when the tRPC-Go server is initialized by calling the trpc.NewServer() function. When initializing the tRPC-Go framework, the framework reads all plugin configurations in the "plugins" section of the framework configuration file, and calls the "Setup()" function of each plugin to complete the parsing and initialization of the plugin configuration, thereby completing the plugin registration.

In general, plugins are independent of each other, and the framework initializes plugins one by one in random order. If a plugin depends on other plugins (such as the tconf plugin depends on the Polaris addressing plugin), the plugin can implement the following methods to declare the dependency relationship.

``` go
// Strong dependency on other plugins, returns an array of plugins, each element in the format of: plugin type-plugin name (note: there is a hyphen in the middle).
// This interface is optional and only needs to be implemented when there is a strong dependency relationship.
type Depender interface {
    DependsOn() []string
}

// Weak dependency interface, different from strong dependency, the plugin does not require the dependent plugin to exist.
// If the dependent exists, it will always be initialized after the dependent.
// Optional, only needs to be implemented when there is a weak dependency relationship.
type FlexDepender interface {
    FlexDependsOn() []string
}
```

The dependency relationship is divided into strong dependency and weak dependency. Strong dependency requires the dependent plugin to exist, and the framework will panic if it does not exist. Weak dependency will not panic. The framework will first ensure that all strong dependencies are met, and then check weak dependencies.

For example, in the following example, the initialization of the plugin has a strong dependency on the Polaris addressing plugin and a weak dependency on the Rainbow configuration plugin.

``` go
func (p *Plugin) DependsOn() []string {
    return []string{"selector-polaris"}
}
func (p *Plugin) FlexDependsOn() []string {
    return []string{"config-rainbow"}
}
```

## Without Configuration Files

In the tRPC-Go framework, there are a few plugin types that do not require configuration during initialization. For these plugins, the plugin registration is not completed using the framework configuration file. Instead, the framework module provides independent plugin registration functions for users to register plugins as needed. The system components that use this method include:

- Registration function for Codec

``` go
type Codec interface {
  // the Decode method is used to parse the binary request body from a complete binary network data packet on the server side
  Decode(message Msg, request-buffer []byte) (reqbody []byte, err error)
  // the Encode method is used to package the binary response body into a complete binary network data packet on the server side
  Encode(message Msg, rspbody []byte) (response-buffer []byte, err error)
}
// the Register function registers the Codec by protocol name, and is called by the init function of the specific third-party implementation package. If there is only a client and no server, the serverCodec parameter can be set to nil
func Register(name string, serverCodec Codec, clientCodec Codec)
```

- Registration function for serialization type

``` go
type Serializer interface {
  // the Unmarshal method is used to parse the binary request body into a specific reqbody structure after the binary packet is unpacked on the server side
  Unmarshal(req-body-bytes []byte, reqbody interface{}) error
  // the Marshal method is used to convert the rspbody structure into binary format for packaging into a binary response packet on the server side
  Marshal(rspbody interface{}) (rsp-body-bytes []byte, err error)
}
// the RegisterSerializer function registers the Serializer by serialization type, and is called by the init function of the specific third-party implementation package
func RegisterSerializer(serializationType int, s Serializer)
```

- Registration function for compression method

``` go
type Compressor interface {
  // the Decompress method is used to decompress the binary packet and extract the original binary data on the server side
  Decompress(in []byte) (out []byte, err error)
  // the Compress method is used to compress the binary packet into smaller binary data for packaging into a binary response packet on the server side
  Compress(in []byte) (out []byte, err error)
}
// the RegisterCompressor function registers the Compressor by compression type, and is called by the init function of the specific third-party implementation package
func RegisterCompressor(compressType int, s Compressor)
```

# Interface Implementation

After configuring and registering the plugins, users need to implement the standard interfaces defined by the framework. The standard interfaces defined by the framework mainly include: filter, config, log, selector, and codec. Users can refer to the development documents for each standard interface for their definitions and implementation details.

- For developing filter, please refer to the [tRPC-Go Filter Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/interceptor.md) document.
- For developing configuration plugins, please refer to the [tRPC-Go Configuration Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/config.md) document.
- For developing log plugins, please refer to the [tRPC-Go Log Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/log.md) document.
- For developing naming service plugins, please refer to the [tRPC-Go Naming Service Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/naming.md) document.
- For developing monitoring plugins, please refer to the [tRPC-Go Monitoring Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/metrics.md) document.
- For developing distributed tracing plugins, please refer to the [tRPC-Go Distributed Tracing Plugin Development](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/open_tracing.md) document.
- For developing protocol plugins, please refer to the [tRPC-Go Protocol Development](todo) document.

# Example

Below is an example of registering a custom plugin using a configuration file. The plugin type is "foo" and the plugin name is "default".

```go
const (
    pluginType = "foo"
    pluginName = "default"
)

type Foo struct {
    Nums []int `yaml:"nums"`
    Name string `yaml:"name"`
}

func init() {
    plugin.Register(pluginName, &Foo{})
}

func (f *Foo) Type() string {
    return pluginType
}

func (f *Foo) Setup(name string, cfgDec plugin.Decoder) error {
    if err := cfgDec.Decode(f); err != nil {
        return err
    }
    return nil
}
```

plugin configuration:

```yaml
plugins:
  foo:
    default:
      nums: [1, 2, 3]
      name: "default_foo"
```

