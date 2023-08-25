# tRPC-Go 模块：config



### 背景

程序中经常有加载配置、配置文件的需求，为了对配置版本进行更好管理，我们还可能会使用到配置中心。

config 就是为了更好地支持上述能力，所提炼封装的一个 package，通过该 package 可以非常方便地实现配置的读取。

为了更好地扩展支持到不同的配置格式、配置中心，config 也进行了一定的插件化设计。

### 原理

结合类图来简要描述下 config 的实现原理。

![](/.resources/developer_guide/module_design/config/uml.png)

配置既需要支持对不同格式的配置文件的加载，也需要支持对远程配置中心的支持。这里的 config 提供了一组接口，支持通过 key 来获取 kvpairs 中的 value，也支持获取一个配置并反序列化成指定的结构体。

由于 config 是插件化设计，支持通过插件来对接不同的配置格式，也支持对接不同的配置中心，如 tconf、七彩石等，所以是可以非常方便地来实现对多种配置文件、配置中心配置的读取的。

通过 config 提供的通知机制，还可以感知到配置中心配置下发相关的操作，以实现本地配置的动态更新。

实现相关内容详见：https://git.woa.com/trpc-go/trpc-go/tree/master/config
下面介绍下大致的实现。

### 实现

下面结合类图来对关键接口及实现进行说明。

- #### DataProvider

  DataProvider 定义了从不同数据来源拉取数据的通用接口，trpc-go 默认实现 FileProvider 从文件中读取文件数据

  ```go
  // DataProvider 为通用的内容拉取接口
  type DataProvider interface {
      //TODO:add ability to watch
      Name() string
      Read(string) ([]byte, error)
      Watch(ProviderCallback)
  }
  ```

- #### Codec

  codec 定义了解析不同配置文件的通用接口，支持业务自定义

  ```go
  type Codec interface {
      Name() string
      Unmarshal([]byte, interface{}) error
  }
  ```

### 使用方式

- #### 1、读取配置

加载配置文件，调用 ConfigLoader 的 Load 方法可以读取配置，例如：

```go
config, err := config.DefaultConfigLoader.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
```

- #### 2、获取某个配置文件值

如：获取配置文件中 plugins.tracing.jaeger.disabled 这个配置的值

```go
out := config.GetBool("plugins.tracing.jaeger.disabled", true
```

