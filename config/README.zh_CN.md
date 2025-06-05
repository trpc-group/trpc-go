[English](README.md) | 中文

# config

`config` 包允许你以简单的方式读取各种类型的配置，并监听配置的变更。
被读取的配置可以来自不同的数据源，你可以开发 config 类型的插件，从感兴趣的数据源中加载配置。
而当读取的配置由于某些原因丢失时，config 包允许你回退使用默认值。

## 如何使用 `config` 包

假设下面是你的配置，该配置以 yaml 的格式进行编码。
你把这个配置存放在当前目录下一个名为 "custom.yaml" 文件中。

```yaml
custom :  # Customize configuration.
  test : customConfigFromServer
  test_obj :
    key1 : value1
    key2 : true
    key3 : 1234
```

下面是一个使用 `config` 包来读取配置的程序：

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

`Load` 函数从 "file" 类型的数据源加载配置，被加载配置位于 "custom.yaml"，并使用“yaml”类型的 codec 对被加载的配置进行解码。
"file" 类型的数据源对应着 `config` 包默认提供的 `FileProvider` (其名字为“file”)，你可以使用 `config.RegisterProvider` 来注册新的数据源。
“yaml”类型的 codec 对应着 `config` 包默认提供的 `YamlCodec` (其名字为“yaml”)，除此之外 `config` 包还提供了 `JSONCodec` 和 `TomlCodec`，你也可以使用 `config.RegisterCodec` 来注册新的 codec。

上面的程序在成功调用 `Load` 函数之后，将返回一个 `Config` 接口，并使用 `GetInt` 和 `GetString` 获取更细粒度的配置内容，查询键中包含的 '.' 符号为 `config` 包默认的的分割符号。

截至撰写本文时，它打印：

```ascii
custom.test_obj.key3: 1234
2023-09-15 14:24:11.000 DEBUG   config/trpc_config.go:525       trpc config: search key custom.test_obj.key4 failed: trpc/config: config not exist
custom.test_obj.key4: ok
```

可以看到成功地获取到了 `custom.test_obj.key3` 的值“1234”，但是获取 `custom.test_obj.key4` 失败了，回退使用提供的默认值“ok”。

### 监听配置变化

假设当前协程会修改配置文件 "custom.yaml" 中的内容，比如将其中的“custom.test_obj.key1”的值变更为“unknown”，而你希望在另外一个协程中监听到配置的变更。
那么在上面的程序中继续添加如下代码，可以模拟出你的使用场景：

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

它将会打印出变更后的配置，可以看到“custom.test_obj.key1”的值变更为了“unknown”。

```ascii
config is changed to: custom :  # Customize configuration.
  test : customConfigFromServer
  test_obj :
    key1 : unknown
    key2 : true
    key3 : 1234
```

这里的关键是先使用 `config.GetProvider("file")` 获取到 `FileProvider`，然后调用 `FileProvider` 的 `Watch` 方法来监听配置变更。

### 更多使用例子

- [服务器端读取自定义的配置文件，并将配置文件的参数以文本形式发送给客户端的例子](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/config)
- [使用七彩石配置中心的例子](https://git.woa.com/trpc-go/trpc-config-rainbow)
- [How to mock `Watch`](mockconfig/README.md)
- [如何开发配置插件](https://git.woa.com/trpc-go/trpc-go/tree/master/docs/developer_guide/develop_plugins/config.zh_CN.md)
