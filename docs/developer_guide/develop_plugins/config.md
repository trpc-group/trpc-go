[TOC]

# 1. Introduction

This document refers to the business configuration plugin for developing remote configuration centers, not framework configurations. Check [here](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/framework_conf.md) for the framework configuration document.

The framework abstracts component-based configuration interfaces: `trpc-go/config`, integrates basic configuration center pull capabilities, and provides a simple way to read various content sources and file types of configurations.

Specific configuration implementations are registered as plugins. This document describes how to develop configuration plugins.

# 2. Principle

Please take the following steps to implement a custom configuration plugin.

1.  Pull remote configuration:

    implementing the DataProvider interface and the KVConfig interface.

    ```go

    // DataProvider universal content source interface
    // By implementing methods such as Name, Read, and Watch, you can read configurations from any content source (file, TConf, ETCD, configmap)
    // And parse it into a standard format that can be processed by encoders and decoders (JSON, TOML, YAML), etc.
    type DataProvider interface {
        Name() string // Get the provide name registered in trpc_go.yaml
        Read(string) ([]byte, error) // Read configuration from provider
        Watch(ProviderCallback) // Listen for configuration changes
    }
    ```

    ```go
    // KVConfig kv configuration
    type KVConfig interface {
        KV
        Watcher
        Name() string // Same as DataProvider.Name
    }
    // KV configuration center key-value pair interface
    type KV interface {
        // Put sets or updates the value corresponding to configuration item key
        Put(ctx context.Context, key, val string, opts ...Option) error

        // Get gets the value corresponding to the configuration item key
        Get(ctx context.Context, key string, opts ...Option) (Response, error)

        // Del deletes the configuration item key
        Del(ctx context.Context, key string, opts ...Option) error
    }

    // Watcher Configuration Center Watch event interface
    type Watcher interface {
        // Watch listens for change events of the configuration item key
        Watch(ctx context.Context, key string, opts ...Option) (<-chan Response, error)
    }
    ```

2.  Parse service configuration:

    use a specific PluginConfig structure to customize plugin configurations

    ```go
    // Taking rainbow stone as an example
    // PluginConfig trpc-conf plugin configuration
    type PluginConfig struct {
        Providers []*Config yaml:"providers"
    }
    // Config provider configuration
    type Config struct {
        Name    string yaml:"name"
        AppID   string yaml:"appid"
        Group   string yaml:"group"
        Timeout int    yaml:"timeout" default:"2000"
    }
    // Set up loading plugin
    func (p *rainbowPlugin) Setup(name string, decoder plugin.Decoder) error {
        cfg := &PluginConfig{}
        err := decoder.Decode(cfg) // Use the yaml decoder to parse the configuration file into the PluginConfig structure while loading the plugin
        // After parsing the configuration, initialize the provider in turn...
    }
    ```

    ```yaml
    // The trpc_go.yaml configuration file structure is as follows
    config:
      rainbow: # Rainbow Stone Configuration Center
        providers:
          - name: rainbow
            appid: 46cdd160-b8c1-4af9-8353-6dfe9e59a9bd
            group: trpc_go_ugc_weibo_video
            timeout: 2000
    ```

3.  Use Codec to parse the configuration:
    ```go
     // Codec Encoder and decoder
     type Codec interface {
         Name() string
         Unmarshal([]byte, interface{}) error
     }
     // RegisterCodec registers the encoder and decoder. When the plugin is started, the Codec is registered with the global codecMap, and WithCodec is carried during Load
     func RegisterCodec(c Codec)
    ```

Developing configuration plugins mainly involves implementing relevant interfaces, registering them in the `config` library, and using them as needed during configuration loading.

Through the above interfaces, we can implement:

1. Protocol configuration parsing plugin
2. Content source pull plugin
3. Both 1 and 2

> If it is only a protocol parsing plugin, you can register the implementation of Codec directly in any init without registering plugins.

If you need to get plugin configurations from `trpc_go.yaml`, you need to register plugins and perform configuration parsing operations. For specific examples, please see the plugin registration example.

# 3. Implementation

## Interface Meaning

- Name: DataProvider name, bind Name and DataProvider when registering DataProvider in `RegisterProvider`.

- Read: One-time read configuration interface.

- Watch: Register configuration change processing function.

## Code implementation

```go
package config

import (
    "io/ioutil"
    "path/filepath"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "github.com/fsnotify/fsnotify"
)

func init() {
    // Register DataProvider
    RegisterProvider(newFileProvider())
}

func newFileProvider() *FileProvider {
    return &FileProvider{}
}

// FileProvider pulls configuration content from the file system
type FileProvider struct {
}

// Name DataProvider name
func (*FileProvider) Name() string {
    return "file"
}

// Read Reads specified configuration based on path
func (fp *FileProvider) Read(path string) ([]byte, error) {
    // TODO: Load and read configurations based on Path
}

// Watch Registers configuration change processing function
func (fp *FileProvider) Watch(cb ProviderCallback) {
    // TODO: Register configuration change processing function to execute when configuration changes occur
}
```

## Plugin Registration

```go
import (
    "fmt"
    "sync"
    "git.code.oa.com/trpc-go/trpc-go/config"
    "git.code.oa.com/trpc-go/trpc-go/plugin"
    trpc "git.code.oa.com/trpc-go/trpc-go"
)

const (
    pluginName = "file"
    pluginType = "config"
)

func init() {
    // Register plugin
    plugin.Register(pluginName, &filePlugin{})
}

// filePlugin tconf plugin
type filePlugin struct{}

// DependsOn Plugin dependencies
func (p *filePlugin) DependsOn() []string {
    return depends
}

// Type Returns plugin type
func (p *filePlugin) Type() string {
    return pluginType
}

// Setup Loads plugin
func (p *filePlugin) Setup(name string, decoder plugin.Decoder) error {
    // TODO: Load and register plugin in accordance with framework configuration, trpc_go.yaml
}
```

# 4. Example (Using Rainbow Stone Plugin as an example)

## Implement the KVConfig Interface

```go
// Name returns the name
func (k *KV) Name() string {
    return k.name
}

// Get pulls configuration
func (k *KV) Get(ctx context.Context, key string, opts ...config.Option) (config.Response, error) {
// TODO: Fetch target configuration
}

// Put updates configuration
func (k *KV) Put(ctx context.Context, key string, val string, opts ...config.Option) error {
// ...
}

// Del deletes configuration
func (k *KV) Del(ctx context.Context, key string, opts ...config.Option) error {
// ...
}

// Watch monitors configuration changes
func (k *KV) Watch(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
// TODO: Monitor changes in the target configuration and pass the results to the business layer through a channel
}
```

## Implement the Config Interface

```go
// Provider implements the DataProvider for Rainbow Stone
type Provider struct {
    kv *KV
}

// Read reads the configuration for the specified key
func (p *Provider) Read(path string) ([]byte, error) {
// TODO: Read the target path configuration
}

// Watch registers the callback function for configuration changes
func (p *Provider) Watch(cb config.ProviderCallback) {
// TODO: Register the callback function for configuration changes
}

// Name returns the provider name
func (p *Provider) Name() string {
return p.kv.Name()
}
```

## Implement the Codec interface

```go
// Below is a simple implementation of a json codec
// JSONCodec JSON codec
type JSONCodec struct{}

// Name JSON codec
func (*JSONCodec) Name() string {
    return "json"
}
// Unmarshal JSON decode
func (c *JSONCodec) Unmarshal(in []byte, out interface{}) error {
    return json.Unmarshal(in, out)
}
// Register it in init() function to use it.
RegisterCodec(&JSONCodec{})
```

## Code Samples

[tconf](https://git.woa.com/trpc-go/trpc-config-tconf)
[rainbow](https://git.woa.com/trpc-go/trpc-config-rainbow)

