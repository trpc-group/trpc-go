English | [中文](README.zh_CN.md)

# config

The `config` package allows you to easily read various types of configurations and watch changes to configurations.
The configurations being read can come from different data sources, and you can develop config-type plugins to load configurations from data sources that you are interested in.
When the configurations being read are lost for some reason, the config package allows you to fall back to using default values.

## How to use the config package

Assume that the following is your configuration, which is encoded in YAML format.
You store this configuration in a file named "custom.yaml" in the current directory.

```yaml
custom :  # Customize configuration.
  test : customConfigFromServer
  test_obj :
    key1 : value1
    key2 : true
    key3 : 1234
```

Here is a program that uses the config package to read the configuration:

```go
package main

import (
    "fmt"
    "log/slog"
    
    "git.code.oa.com/trpc-go/trpc-go/config"
)

func main() {
    const configPath = "custom.yaml"
    c, err := config.Load(configPath, config.WithCodec("yaml"), config.WithProvider("file"))
    if err != nil {
        slog.Error("loading config failed", "config path", configPath, "error", err)
    }

    fmt.Printf("custom.test_obj.key3: %d\n", c.GetInt("custom.test_obj.key3", 567))
    fmt.Printf("custom.test_obj.key4: %v\n", c.GetString("custom.test_obj.key4", "ok"))
}
```

The `Load` function loads the configuration from a data source of type "file".
The loaded configuration is located in "custom.yaml", and the "yaml" codec is used to decode the loaded configuration.

he "file" data source type corresponds to the `FileProvider` whose name is "file" provided by the `config` package .
You can use the `config.RegisterProvider` function to register a new data source.

The "yaml" codec type corresponds to the `YamlCodec` whose name is "yaml" provided by the config package .
In addition, the `config` package also provides `JSONCodec` and `TomlCodec`, and you can use the `config.RegisterCodec` function to register a new codec.

After the `Load` function is successfully called, the program returns a `Config` interface, and `GetInt` and `GetString` are used to obtain more granular configuration content.
The '.' symbol in the query key is the default separator for the `config` package.

At the time of writing this article, it prints:

```ascii
custom.test_obj.key3: 1234
2023-09-15 14:24:11.000 DEBUG   config/trpc_config.go:525       trpc config: search key custom.test_obj.key4 failed: trpc/config: config not exist
custom.test_obj.key4: ok
```

It can be seen that the value "1234" of "custom.test_obj.key3" is successfully obtained, but the attempt to obtain "custom.test_obj.key4" failed, and the default value "ok" is used as a fallback.

### Watch Configuration Changes

Assume that the current goroutine modifies the contents of the configuration file "custom.yaml", such as changing the value of "custom.test_obj.key1" to "unknown", and you want to watch configuration changes in another goroutine.
Then, add the following code to the program to simulate your use case:

```go
    cfg := make(chan []byte)
    config.GetProvider("file").Watch(func(path string, content []byte) {
        if path == configPath {
            cfg <- content
        }
    })
    
    var g sync.WaitGroup
    g.Add(1)
    go func() {
        defer g.Done()
        select {
        case c := <-cfg:
            fmt.Printf("config is changed to: %s\n", string(c))
        case <-time.After(10 * time.Second):
            slog.Error("receiving message timeout", "timeout", 10*time.Second)
        }
    }()

    if err := os.WriteFile(configPath, []byte(`custom :  # Customize configuration.
  test : customConfigFromServer
  test_obj :
    key1 : unknown
    key2 : true
    key3 : 1234`), 0644); err != nil {
        slog.Error("writing new config failed", "config path", configPath, "error", err)
    }
    g.Wait()
```

It will print the changed configuration, and you can see that the value of "custom.test_obj.key1" has been changed to "unknown".

```ascii
config is changed to: custom :  # Customize configuration.
  test : customConfigFromServer
  test_obj :
    key1 : unknown
    key2 : true
    key3 : 1234
```

The key is to use `config.GetProvider("file")` to obtain the `FileProvider`, and then call the `Watch` method of the `FileProvider` to listen for configuration changes.

### More Examples

- [An example of reading a custom configuration file on the server and sending the configuration parameters to the client in text form](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/config)
- [An example of using the Rainbow configuration center](https://git.woa.com/trpc-go/trpc-config-rainbow)
- [How to mock `Watch`](mockconfig/README.md)
- [How to develop config plugin](https://git.woa.com/trpc-go/trpc-go/blob/master/docs/developer_guide/develop_plugins/config.zh_CN.md)
