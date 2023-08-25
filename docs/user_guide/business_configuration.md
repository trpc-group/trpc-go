[TOC]

# Introduction

Configuration management plays an extremely important role in the microservices governance system. The tRPC framework provides a set of standard interfaces for business program development, supporting the retrieval of configuration from multiple data sources, parsing configuration, and perceiving configuration changes. The framework shields the details of data source docking, simplifying development. This article aims to provide users with the following information:

* What is business configuration and how does it differ from framework configuration?
* Some core concepts of business configuration such as: provider, codec, etc.
* How to use standard interfaces to retrieve business configurations
* How to perceive changes in configuration items
* How to dock with multiple data sources

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

A data source is the source from which configuration is retrieved and where it is stored. Common data sources include: file, etcd, configmap, tconf, rainbow, etc. The tRPC framework supports setting different data sources for different business configurations. The framework uses a plugin-based approach to extend support for more data sources. In the implementation principle section later, we will describe in detail how the framework supports multiple data sources.

## What is Codec?

In business configuration, Codec refers to the format of configurations retrieved from configuration sources. Common configuration file formats include: YAML, JSON, TOML, etc. The framework uses a plugin-based approach to extend support for more decoding formats.

# Implementation Principle
To better understand the use of configuration interfaces and how to dock with data sources, let's take a brief look at how the configuration interface module is implemented. The following diagram is a schematic diagram of the configuration module implementation (not a code implementation class diagram):

![trpc](/.resources/user_guide/business_configuration/trpc.png)

The config interface in the diagram provides a standard interface for business code to retrieve configuration items, and each data type has an independent interface that supports returning default values.

We have already introduced Codec and DataProvider in section 2, and these two modules provide standard interfaces and registration functions to support plugin-based encoding/decoding and data source. Taking multi-data sources as an example, DataProvider provides the following three standard interfaces:

* Read(): provides how to read the original data of the configuration (raw bytes).
* Watch(): provides a callback function that the framework executes when the data source's data changes.

```
type DataProvider interface {
    Name() string
    Read(string) ([]byte, error)
    Watch(ProviderCallback)
}
```

Finally, let's see how to retrieve a business configuration by specifying the data source and decoder:

```
// Load TConf configuration file: config.WithProvider("tconf")
c, _ := config.Load("test.yaml", config.WithCodec("yaml"), config.WithProvider("tconf"))
// Read String type configuration
c.GetString("auth.user", "admin")
```

In this example, the data source is the tconf configuration center, and the business configuration file in the data source is "test.yaml". When the ConfigLoader obtains the "test.yaml" business configuration, it specifies to use YAML format to decode the data content. Finally, the c.GetString("server.app", "default") function is used to obtain the value of the auth.user configuration item in the test.yaml file.

# Interface Usage

This article only introduces the corresponding interfaces from the perspective of using business configurations. If users need to develop data source plugins or Codec plugins, please refer to tRPC-Go Development Configuration Plugin. For specific interface parameters, please refer to the tRPC-Go API manual.

The tRPC-Go framework provides two sets of interfaces for "reading configuration items" and "watching  configuration item changes".

## Reading Configuration Items

Step 1: Selecting Plugins

Before using the configuration interface, it is necessary to configure data source plugins and their configurations in advance. Please refer to the Plugin Ecology for plugin usage. For the configuration of tconf and Rainbow, please refer to section 5. The tRPC framework supports local file data sources by default.

Step 2: Plugin Initialization

Since the data source is implemented using a plugin, tRPC framework needs to initialize all plugins in the server initialization function by reading the "trpc_go.yaml" file. The read operation of business configuration must be carried out after completing trpc.NewServer().

```
import (
    trpc "git.code.oa.com/trpc-go/trpc-go"
)
// Plugin system will be initialized when the server is instantiated, and all configuration read operations need to be performed after this.
trpc.NewServer()
```

Step 3: Loading Configuration
Load configuration file from data source and return config data structure. The data source type and Codec format can be specified, with the framework defaulting to "file" data source and "YAML" Codec. The interface is defined as follows:

```
// Load configuration file: path is the path of the configuration file
func Load(path string, opts ...LoadOption) (Config, error)
// Change Codec type, default is "YAML" format
func WithCodec(name string) LoadOption
// Change data source, default is "file"
func WithProvider(name string) LoadOption
```

The sample code is as follows:

```
// Load TConf configuration file: config.WithProvider("tconf")
c, _ := config.Load("test1.yaml", config.WithCodec("yaml"), config.WithProvider("tconf"))
// Load local configuration file, codec is json, data source is file
c, _ := config.Load("../testdata/auth.yaml", config.WithCodec("json"), config.WithProvider("file"))
// Load local configuration file, default Codec is yaml, data source is file
c, _ := config.Load("../testdata/auth.yaml")
```

Step 4: Retrieving Configuration Items
Get the value of a specific configuration item from the config data structure. Default values can be set, and the framework provides the following standard interfaces:

```
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

```
// Read bool type configuration
c.GetBool("server.debug", false)
// Read String type configuration
c.GetString("server.app", "default")
```

## Watching configuration item changes

The framework provides a Watch mechanism for business programs to define and execute their own logic based on received configuration item change events in KV-type configuration centers (such as tconf and rainbow). The monitoring interface is designed as follows:

```
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

The following example demonstrates how a business program monitors the "test.yaml" file on tconf,
prints configuration item change events, and updates the configuration:

```
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
// Listen to remote configuration changes on tconf using the Watch interface in trpc-go/config
c, _ := config.Get("tconf").Watch(context.TODO(), "test.yaml")
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

Local configuration, Rainbow, and tconf are the three common data source access modes. This section will describe in detail how tRPC integrates with these three data sources.

## Integration with Local Files

The framework natively supports local configuration files. Users do not need to take any special actions. Simply use the interfaces described in Section 4 to retrieve configuration items. The framework does not support user-defined listening for configuration item changes.

## Integration with Rainbow

### Step 1: Operations in the Rainbow Console

Access the Web Console (http://rainbow.oa.com/)

Create a new project (skip this step if you already have a project). The string in the middle of the browser URL is the "appid" field required by the plugin configuration, such as "3482e0a7-3a00-401c-9505-7bdb0a12511c" in http://rainbow.oa.com/console/3482e0a7-3a00-401c-9505-7bdb0a12511c/list.

Create a new group (skip this step if you already have a group). The name of the group will be the "group" field in the plugin configuration.

Add a configuration and publish it.

### Step 2: Plugin Configuration

The "provider" field indicates the group that the configuration belongs to. The plugin supports pulling configurations from multiple providers.

| Configuration Item     | Description                                                                                                                                                   |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name                   | The provider identifier. You can use config.WithProvider("tconf1") to specify which provider the configuration is pulled from.                                |
| appid                  | The ID of the project that the configuration belongs to.                                                                                                      |
| group                  | The name of the group that the configuration belongs to.                                                                                                      |
| type                   | The data format used by rainbow, kv (default) or table.                                                                                                       |
| env_name               | Rainbow multi-environment configuration. This field does not need to be configured if you are not using multi-environment features.                           |
| timeout                | Timeout setting for pulling configuration interfaces, in milliseconds. If not specified, the default timeout is 2 seconds.                                    |
| address                | Rainbow server address. This field does not need to be filled in for intranet environments. For external network environments, please consult rainbow_helper. |
| uin                    | Client identifier. Optional configuration.                                                                                                                    |
| file_cache             | Local cache file settings. Optional configuration.                                                                                                            |
| enable_sign            | Set signature verification. Optional configuration. If enabled, you need to set user_id and user_key.                                                         |
| user_id                | User ID generated by the platform. Used to generate a signature when pulling configurations. Required when enable_sign is set to true.                        |
| user_key               | User key generated by the platform. Used to generate a signature when pulling configurations. Required when enable_sign is set to true.                       |
| enable_client_provider | Use client provider. Optional configuration. Defaults to False if left blank.                                                                                 |

Please add the corresponding plugin configuration to the framework's configuration file, trpc_go.yaml:

```
plugins:
    config:
        rainbow: # rainbow configuration center
            providers:
              - name: rainbow # Provider name, used in code as: `config.WithProvider("rainbow")`
                appid: 3482e0a7-3a00-401c-9505-7bdb0a12511c # App ID
                group: dev # Configuration group
                type: kv   # rainbow data format, kv (default) or table
                env_name: production
                file_cache: /tmp/a.backup
                uin: a3482e0a7
                enable_sign: true
                user_id: 2a9a63844fe24a8aadaxxx5d2f5e903a
                user_key: 599dd5a3480805e22bb6ac22eeaf40d34f8a
                enable_client_provider: true
                timeout: 2000
              - name: rainbow1
                appid: 3482e0a7-3a00-401c-9505-7bdb0a12511c
                group: dev1
```

### Step 3: Register the Plugin
```
import (
    // Automatically register the rainbow plugin based on the plugin configuration
    _ "git.code.oa.com/trpc-go/trpc-config-rainbow"
)
```

### Step 4: Integration Completed

The integration between tRPC and Rainbow is complete. Users can use the interfaces described in Section 3 to retrieve configurations.

## Integration with Tconf

Tconf will be gradually migrated to Rainbow later on, and the data migration will be done by the Tconf backend, which will be transparent to business.

### Step 1: Operations in the Tconf Console

Register your service and create configurations in the Tconf system through the Web Console at http://tconf.pcg.com/.

### Step 2: Plugin Configuration

The "provider" field represents the combination of the appid, env_name, and namespace that the configuration belongs to in the Tconf service module. The plugin supports pulling configurations from multiple providers. In Tconf, a configuration file must belong to a specific appid and env.

| Configuration Item     | Description                                                                                                                                                                                |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| name                   | The provider identifier. When using the configuration in the provider, you can use config.WithProvider("tconf1").                                                                          |
| appid                  | The ID of the project that the configuration belongs to. Optional. If not specified, the app server under "server" in trpc_go.yaml will be used.                                           |
| env_name               | The name of the environment that the configuration belongs to. Optional. If not specified, the env_name under "global" in trpc_go.yaml will be used.                                       |
| namespace              | The namespace to which the current service running environment belongs (Development or Production). Optional. If not specified, the namespace under "global" in trpc_go.yaml will be used. |
| enable_client_provider | Use client provider. Optional configuration. Defaults to False if left blank.                                                                                                              |

Please add the corresponding plugin configuration to the tRPC framework's configuration file, trpc_go.yaml:

```
plugins:
  config:
    tconf:
      providers:
       - name: tconf1
         appid: tconf.config
         env_name: test
         namespace: Development
         enable_client_provider: true
       - name: tconf2
         appid: test.trey.conf
         env_name: test
         namespace: Development
```

### Step 3: Register the Plugin

Because the tconf plugin depends on Polaris for tconf service addressing, both tconf and Polaris need to be registered when registering the plugin.

```
import (
    // Automatically register the tconf plugin based on the plugin configuration
    _ "git.code.oa.com/trpc-go/trpc-config-tconf"
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris" // The tconf plugin depends on Polaris for addressing
)
```

### Step 4: Integration Completed

The integration between tRPC and Tconf is complete. Users can use the interfaces described in Section 3 to retrieve configurations.

