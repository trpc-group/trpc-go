# Introduction

Configuration management plays an extremely important role in the microservices governance system. The tRPC framework provides a set of standard interfaces for business program development, supporting the retrieval of configuration from multiple data sources, parsing configuration, and perceiving configuration changes. The framework shields the details of data source docking, simplifying development. This article aims to provide users with the following information:

* What is business configuration and how does it differ from framework configuration.
* Some core concepts of business configuration such as: provider, codec, etc.
* How to use standard interfaces to retrieve business configurations.
* How to perceive changes in configuration items.

# Concept

## What is Business Configuration?

Business configuration refers to configuration used by the business, defined by the business program in terms of format, meaning, and parameter range. The tRPC framework does not use business configuration nor care about its meaning. The framework only focuses on how to retrieve configuration content, parse configuration, discover configuration changes, and notify the business program.

The difference between business configuration and framework configuration lies in the subject using the configuration and the management method. Framework configuration is used for tRPC framework and defined by the framework in terms of format and meaning. Framework configuration only supports local file reading mode and is read during program startup to initialize the framework. Framework configuration does not support dynamic updates; if the framework configuration needs to be updated, the program needs to be restarted.

On the other hand, business configuration supports retrieval from multiple data sources such as local files, configuration centers, databases, etc. If the data source supports configuration item event listening, tRPC framework provides a mechanism to achieve dynamic updating of configurations.

## Managing Business Configuration

For managing business configuration, we recommend the best practice of using a configuration center. Using a configuration center has the following advantages:

* Avoiding source code leaking sensitive information
* Dynamically updating configurations for services
* Allowing multiple services to share configurations and avoiding multiple copies of the same configuration
* Supporting gray releases, configuration rollbacks, and having complete permission management and operation logs
* Business configuration also supports local files. For local files, most use cases involve clients being used as independent tools or programs in the development and debugging phases. The advantage is that it can work without relying on an external system.

## What is Multiple Data Sources?

A data source is the source from which configuration is retrieved and where it is stored. Common data sources include: file, etcd, configmap, etc. The tRPC framework supports setting different data sources for different business configurations. The framework uses a plugin-based approach to extend support for more data sources. In the implementation principle section later, we will describe in detail how the framework supports multiple data sources.

## What is Codec?

In business configuration, Codec refers to the format of configurations retrieved from configuration sources. Common configuration file formats include: YAML, JSON, TOML, etc. The framework uses a plugin-based approach to extend support for more decoding formats.

# Implementation Principle
To better understand the use of configuration interfaces and how to dock with data sources, let's take a brief look at how the configuration interface module is implemented. The following diagram is a schematic diagram of the configuration module implementation (not a code implementation class diagram):

![trpc](/.resources/user_guide/business_configuration/trpc_en.png)

The config interface in the diagram provides a standard interface for business code to retrieve configuration items, and each data type has an independent interface that supports returning default values.

We have already introduced Codec and DataProvider in section 2, and these two modules provide standard interfaces and registration functions to support plugin-based encoding/decoding and data source. Taking multi-data sources as an example, DataProvider provides the following three standard interfaces:

* Read(): provides how to read the original data of the configuration (raw bytes).
* Watch(): provides a callback function that the framework executes when the data source's data changes.

```go
type DataProvider interface {
    Name() string
    Read(string) ([]byte, error)
    Watch(ProviderCallback)
}
```

Finally, let's see how to retrieve a business configuration by specifying the data source and decoder:

```go
// Load etcd configuration file: config.WithProvider("etcd")
c, _ := config.Load("test.yaml", config.WithCodec("yaml"), config.WithProvider("etcd"))
// Read String type configuration
c.GetString("auth.user", "admin")
```

In this example, the data source is the etcd configuration center, and the business configuration file in the data source is "test.yaml". When the ConfigLoader obtains the "test.yaml" business configuration, it specifies to use YAML format to decode the data content. Finally, the c.GetString("server.app", "default") function is used to obtain the value of the auth.user configuration item in the test.yaml file.

# Interface Usage

This article only introduces the corresponding interfaces from the perspective of using business configurations. If users need to develop data source plugins or Codec plugins, please refer to tRPC-Go Development Configuration Plugin. For specific interface parameters, please refer to the tRPC-Go API manual.

The tRPC-Go framework provides two sets of interfaces for "reading configuration items" and "watching  configuration item changes".

## Reading Configuration Items

Step 1: Selecting Plugins

Before using the configuration interface, it is necessary to configure data source plugins and their configurations in advance. Please refer to the Plugin Ecology for plugin usage. The tRPC framework supports local file data sources by default.

Step 2: Plugin Initialization

Since the data source is implemented using a plugin, tRPC framework needs to initialize all plugins in the server initialization function by reading the "trpc_go.yaml" file. The read operation of business configuration must be carried out after completing trpc.NewServer().

```go
import (
    trpc "trpc.group/trpc-go/trpc-go"
)
// Plugin system will be initialized when the server is instantiated, and all configuration read operations need to be performed after this.
trpc.NewServer()
```

Step 3: Loading Configuration
Load configuration file from data source and return config data structure. The data source type and Codec format can be specified, with the framework defaulting to "file" data source and "YAML" Codec. The interface is defined as follows:

```go
// Load configuration file: path is the path of the configuration file
func Load(path string, opts ...LoadOption) (Config, error)
// Change Codec type, default is "YAML" format
func WithCodec(name string) LoadOption
// Change data source, default is "file"
func WithProvider(name string) LoadOption
```

The sample code is as follows:

```go
// Load etcd configuration file: config.WithProvider("etcd")
c, _ := config.Load("test1.yaml", config.WithCodec("yaml"), config.WithProvider("etcd"))
// Load local configuration file, codec is json, data source is file
c, _ := config.Load("../testdata/auth.yaml", config.WithCodec("json"), config.WithProvider("file"))
// Load local configuration file, default Codec is yaml, data source is file
c, _ := config.Load("../testdata/auth.yaml")
```

Step 4: Retrieving Configuration Items
Get the value of a specific configuration item from the config data structure. Default values can be set, and the framework provides the following standard interfaces:

```go
// Config general interface
type Config interface {
    Load() error
    Reload()
    Get(string, interface{}) interface{}
    Unmarshal(interface{}) error
    IsSet(string) bool
    GetInt(string, int) int
    GetInt32(string, int32) int32
    GetInt64(string, int64) int64
    GetUint(string, uint) uint
    GetUint32(string, uint32) uint32
    GetUint64(string, uint64) uint64
    GetFloat32(string, float32) float32
    GetFloat64(string, float64) float64
    GetString(string, string) string
    GetBool(string, bool) bool
    Bytes() []byte
}
```

The sample code is as follows:

```go
// Read bool type configuration
c.GetBool("server.debug", false)
// Read String type configuration
c.GetString("server.app", "default")
```

## Watching configuration item changes

The framework provides a Watch mechanism for business programs to define and execute their own logic based on received configuration item change events in KV-type configuration centers. The monitoring interface is designed as follows:

```go
// Get retrieves kvconfig by name
func Get(name string) KVConfig

// KVConfig is the interface for KV configurations
type KVConfig interface {
    KV
    Watcher
    Name() string
}

// Watcher is the interface for monitoring
type Watcher interface {
    // Watch monitors the changes of the configuration item key
    Watch(ctx context.Context, key string, opts ...Option) (<-chan Response, error)
}

// Response represents the response from the configuration center
type Response interface {
    // Value gets the value corresponding to the configuration item
    Value() string
    // MetaData provides additional metadata information
    // Configuration Option options can be used to carry extra functionality implementation of different configuration centers, such as namespace, group, lease, etc.
    MetaData() map[string]string
    // Event gets the type of the Watch event
    Event() EventType
}

// EventType represents the types of events monitored for configuration changes
type EventType uint8
const (
    // EventTypeNull represents an empty event
    EventTypeNull EventType = 0
    // EventTypePut represents a set or update configuration event
    EventTypePut EventType = 1
    // EventTypeDel represents a delete configuration item event
    EventTypeDel EventType = 2
)
```

The following example demonstrates how a business program monitors the "test.yaml" file on etcd,
prints configuration item change events, and updates the configuration:

```go
import (
    "sync/atomic"
    ...
)
type yamlFile struct {
    Server struct {
        App string
    }
}
var cfg atomic.Value // Concurrent-safe Value
// Listen to remote configuration changes on etcd using the Watch interface in trpc-go/config
c, _ := config.Get("etcd").Watch(context.TODO(), "test.yaml")
go func() {
    for r := range c {
        yf := &yamlFile{}
        fmt.Printf("Event: %d, Value: %s", r.Event(), r.Value())
        if err := yaml.Unmarshal([]byte(r.Value()), yf); err == nil {
            cfg.Store(yf)
        }
    }
}()
// After the configuration is initialized, the latest configuration object can be obtained through the Load method of atomic.Value.
cfg.Load().(*yamlFile)
```

# Data Source Integration

Refer to [trpc-ecosystem/go-config-etcd](https://github.com/trpc-ecosystem/go-config-etcd).
