[TOC]

# tRPC-Go module: config



## Background

There is often a need to load configuration or configuration files in a program. To better manage configuration versions, we may also use a configuration center.

The config package is designed to better support these capabilities, and is a refined and encapsulated package that makes it easy to read configurations.

To better extend support for different configuration formats and configuration centers, config has also been designed with some plugin features.

## Principle

Combined with the class diagram to briefly describe the implementation principle of config.

![module_design_config](/.resources/developer_guide/module_design/config/uml.png)

Configurations need to support loading different types of configuration files and also support remote configuration centers. The config package here provides a set of interfaces that allow getting the value from KVPairs by key or deserializing a configuration into a specific struct.

As config is designed with plugin features, it supports connecting to different configuration formats and centers through plugins such as tconf and rainbow. This makes it very convenient to read configurations from various sources.

Through the notification mechanism provided by config, it is also possible to be notified about configuration center related operations and to achieve dynamic updates of local configurations.

For the implementation details, please refer to: https://git.woa.com/trpc-go/trpc-go/tree/master/config

Here is a brief introduction to the implementation.

## Implementation

The key interface and implementation are described below in conjunction with the class diagram.

### DateProvider

`DataProvider` defines a common interface for pulling data from different data sources, and trpc-go implements `FileProvider` by default to read file data from files

```go
// DataProvider is a common content pull interface
type DataProvider interface {
    //TODO:add ability to watch
    Name() string
    Read(string) ([]byte, error)
    Watch(ProviderCallback)
}
```

### Codec

Codec defines a common interface for parsing different configuration files and supports business customization.

```go
type Codec interface {
    Name() string
    Unmarshal([]byte, interface{}) error
}
```

## How to use

### read configuration

To load the configuration file, call the `Load` method of `ConfigLoader` to read the configuration, for example:

```go
config, err := config.DefaultConfigLoader.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
```

### get a config file value

For example: Get the value of the configuration `plugins.tracing.jaeger.disabled` in the configuration file

```go
out := config.GetBool("plugins.tracing.jaeger.disabled", true)
```

