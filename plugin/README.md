[中文](README_zh_CN.md)


# Plugin

tRPC-Go is designed with a plugin architecture concept, which allows the framework to connect with various ecosystems through plugins, providing openness and extensibility.
The plugin package is used to manage plugins that need to be loaded based on configurations.
Plugins that do not rely on configuration are relatively simple, such as [codec plugins](../codec/README.md), which will not be discussed here.
Therefore, we will first introduce the design of the plugin package and then explain how to develop a plugin that needs to be loaded based on configuration.

## Design of the `plugin` package

The plugin package manages all plugins through a "plugin factory".
Each plugin needs to be registered with the plugin factory.
The plugin factory adopts a two-level management mode:
The first level is the plugin type, such as log type, conf type, selector type, etc.
The second level is the plugin name, such as local file configuration, remote file configuration, local database configuration, etc. for conf plugins.

```ascii
                          +-----------------+
                      +---+  Plugin Factory +-------+
                      |   +--------+--------+       |
                      |            |                |
                      |            |                |
                      |            |                |
                   +--v--+      +--v--+        +----v----+
     +-------------+conf |      | log |        | selector|
     |             +--+--+      +-----+        +----+----+
     |                |                             |
     |                |                             |
     |                |                             |
     |                |                             |
+----v-----+    +-----v-------+                +----v----+
|local-file|    | remote-file |    ......      | polaris |
+----------+    +-------------+                +---------+
```

For the plugin type, the `plugin` package does not impose any restrictions, and you can add your own plugin types.

### Common plugin types

According to their functions, the framework provides the following five types of common plugins:

- Configuration: Provides a standard interface for obtaining configurations, getting configuration data from various data sources such as local files, configuration centers, etc., providing configuration parsing in multiple formats such as json, yaml, etc., and the framework also provides a watch mechanism to achieve dynamic updates of configurations.
- Logging: Provides a unified logging print and log reporting standard interface. Log plugins can complete the docking with remote log systems by implementing the log reporting interface.
- Protocol: Provides standard interfaces related to protocol encoding and decoding, allowing the expansion of business protocols, serialization types, data compression methods, and other protocol processing through plugins.
- Name Service: Provides standard interfaces including service registration, service discovery, policy routing, load balancing, and fusing, used to implement service routing addressing.
- Filter: Provides a generic filter interface, allowing users to set up buried points in the context of service calls to implement functions such as module monitoring, cross-cutting logging, link tracking, and overload protection.

## How to develop a plugin that needs to be loaded based on configuration

Developing a plugin that needs to be loaded based on configuration usually involves implementing the plugin and configuring the plugin. [A runnable specific example](../examples/features/plugin)

### Implementing the plugin

1. The plugin implements the `plugin.Factory` interface.

```go
// Factory is a unified abstract for the plugin factory. External plugins need to implement this interface to generate specific plugins and register them in specific plugin types.
type Factory interface {
    // Type is the type of the plugin, such as selector, log, config, tracing.
    Type() string
    // Setup loads the plugin based on the configuration node. Users need to define the specific plugin configuration data structure first.
    Setup(name string, configDec Decoder) error
}
```

2. The plugin calls `plugin.Register` to register itself with the `plugin` package.

```go
// Register registers the plugin factory. You can specify the plugin name yourself, and different factory instances can be registered for the same implementation with different configurations.
func Register(name string, p Factory)
```

### Configuring the plugin

1. Import the plugin's package in the `main` package.
2. Configure the plugin under the `plugins` field in the configuration file. The configuration file format is:
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
The above configuration defines two plugin types and four plugins.
There are logger1 and logger2 plugins under the log type.
There are local-file and remote-file plugins under the config type.

### Plugin initialization order

After the tRPC-GO server calls the `trpc.NewServer()` function, it reads all plugin configurations under the "plugins" field in the framework configuration file and then calls the "Setup()" function of each plugin to complete the initialization of the plugin configuration.
In general, plugins are independent of each other, and the framework initializes the plugins in a random order (e.g., plugin A depends on plugin B).
If a plugin depends on other plugins, it can implement the following methods to declare the dependency relationship.

```go
// Depender is the interface for "Strong Dependence".
// If plugin a "Strongly" depends on plugin b, b must exist and
// a will be initialized after b's initialization.
type Depender interface {
// DependsOn returns a list of plugins that are relied upon.
// The list elements are in the format of "type-name" like [ "selector-polaris" ].
DependsOn() []string
}

// FlexDepender is the interface for "Weak Dependence".
// If plugin a "Weakly" depends on plugin b and b does exist,
// a will be initialized after b's initialization.
type FlexDepender interface {
FlexDependsOn() []string
}
```

The dependency relationship is divided into strong dependency and weak dependency.
Strong dependency requires the depended plugin to exist, otherwise, the framework will panic.
Weak dependency will not panic.
The framework will first ensure that all strong dependencies are satisfied, and then check the weak dependencies.

For example, in the following example, the plugin initialization strongly depends on the selector type plugin a and weakly depends on the config type plugin b.

```go
func (p *Plugin) DependsOn() []string {
    return []string{"selector-a"}
}
func (p *Plugin) FlexDependsOn() []string {
    return []string{"config-b"}
}
```