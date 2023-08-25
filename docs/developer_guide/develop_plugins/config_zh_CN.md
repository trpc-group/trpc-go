# 1. 前言

本篇文档指的是开发远程配置中心的业务配置插件，不是框架配置，框架配置文档见[这里](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/framework_conf_zh_cn.md)。

框架通过定义组件化的配置接口抽象：`trpc-go/config`，集成基本的配置中心拉取能力，提供了一种简单方式读取多种内容源、多种文件类型的配置，具体配置实现通过插件注册进来，本文介绍的是如何开发配置插件。

# 2. 原理

自定义配置插件主要是实现：

1.  拉取远程配置：实现 DataProvider 接口、实现 KVConfig 接口

    ```go
    // DataProvider 通用内容源接口
    // 通过实现 Name、Read、Watch 等方法，就能从任意的内容源（file、TConf、ETCD、configmap）中读取配置
    // 并通过编解码器解析为可处理的标准格式（JSON、TOML、YAML）等
    type DataProvider interface {
        Name() string                    // 获取 trpc_go.yaml 注册时的 provide name
        Read(string) ([]byte, error)    // 从 provider 中读取配置
        Watch(ProviderCallback)            // 监听配置变化
    }
    ```

    ```go
    // KVConfig kv 配置
    type KVConfig interface {
        KV
        Watcher
        Name() string    // 作用同 DataProvider.Name
    }
    // KV 配置中心键值对接口
    type KV interface {
        // Put 设置或更新配置项 key 对应的值
        Put(ctx context.Context, key, val string, opts ...Option) error
        // Get 获取配置项 key 对应的值
        Get(ctx context.Context, key string, opts ...Option) (Response, error)
        // Del 删除配置项 key
        Del(ctx context.Context, key string, opts ...Option) error
    }
    // Watcher 配置中心 Watch 事件接口
    type Watcher interface {
    // Watch 监听配置项 key 的变更事件
        Watch(ctx context.Context, key string, opts ...Option) (<-chan Response, error)
    }
    ```

2.  服务配置解析：使用特定的 PluginConfig 结构来自定义插件配置

    ```go
    // 以七彩石为例
    // PluginConfig trpc-conf 插件配置
    type PluginConfig struct {
        Providers []*Config `yaml:"providers"`
    }
    // Config provider 配置
    type Config struct {
        Name          string `yaml:"name"`
        AppID         string `yaml:"appid"`
        Group         string `yaml:"group"`
        Timeout       int    `yaml:"timeout" default:"2000"`
    }
    // Setup 加载插件
    func (p *rainbowPlugin) Setup(name string, decoder plugin.Decoder) error {
        cfg := &PluginConfig{}
        err := decoder.Decode(cfg) // 加载插件时通过 yaml 的 decoder 来解析配置文件到 PluginConfig 结构中
        // 解析完配置，依次初始化 provider...
    }
    ```

    ```yaml
    // trpc_go.yaml 配置文件结构如下
    config:
      rainbow: # 七彩石配置中心
        providers:
          - name: rainbow
            appid: 46cdd160-b8c1-4af9-8353-6dfe9e59a9bd
            group: trpc_go_ugc_weibo_video
            timeout: 2000
    ```

3.  通过 Codec 解析配置
    ```go
    // Codec 编解码器
    type Codec interface {
        Name() string
        Unmarshal([]byte, interface{}) error
    }
    // RegisterCodec 注册编解码器，插件启动时将 Codec 注册到全局 codecMap 中，Load 时带上 WithCodec 即可
    func RegisterCodec(c Codec)
    ```

开发配置插件主要是实现相关接口，注册`config`库中，用户根据需要在配置加载的时候按需使用。

通过以上接口我们可以实现：

Through the above interfaces, we can implement:

1. 协议配置解析插件
2. 内容源拉取插件
3. 协议配置解析插件和内容源拉取插件

> 如果只是协议解析插件，可以在任意 init 中直接注册 Codec 的实现到 config 中，无需进行插件注册。

如果需要从`trpc_go.yaml`中获得插件配置，需要进行插件注册和配置解析操作，具体实例请看插件注册示例。

# 3. 实现

## 接口含义

- Name：DataProvider 名字，在 RegisterProvider 注册 DataProvider 时绑定 Name 和 DataProvider。

- Read: 一次性读取配置接口

- Watch: 注册配置变化处理函数

## 代码实现

```go
package config

import (
    "io/ioutil"
    "path/filepath"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "github.com/fsnotify/fsnotify"
)

func init() {
    // 注册 DataProvider
    RegisterProvider(newFileProvider())
}
func newFileProvider() *FileProvider {
    return &FileProvider{}
}
// FileProvider 从文件系统拉取配置内容
type FileProvider struct {
}
// Name DataProvider 名字
func (*FileProvider) Name() string {
    return "file"
}
// Read 根据路径读取指定配置
func (fp *FileProvider) Read(path string) ([]byte, error) {
    // TODO: 根据 Path 读取加载配置
}
// Watch 注册配置变化处理函数
func (fp *FileProvider) Watch(cb ProviderCallback) {
    // TODO: 注册配置变化处理函数，当配置变更时执行
}
```

## 插件注册

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
    // 注册插件
    plugin.Register(pluginName, &filePlugin{})
}
// filePlugin tconf 插件
type filePlugin struct{}
// DependsOn filePlugin 插件依赖
func (p *filePlugin) DependsOn() []string {
    return depends
}
// Type 返回插件类型
func (p *filePlugin) Type() string {
    return pluginType
}
// Setup 加载插件
func (p *filePlugin) Setup(name string, decoder plugin.Decoder) error {
    // TODO: 根据框架配置：trpc_go.yaml 加载注册插件
}
```

# 4. 示例（以七彩石插件为例）

## 如何实现 kvconfig 接口

```go
// Name 返回 name
func (k *KV) Name() string {
    return k.name
}
// Get 拉取配置
func (k *KV) Get(ctx context.Context, key string, opts ...config.Option) (config.Response, error) {
    // TODO: 获取目标配置
}
// Put 更新配置操作
func (k *KV) Put(ctx context.Context, key string, val string, opts ...config.Option) error {
    // ...
}
// Del 删除配置操作
func (k *KV) Del(ctx context.Context, key string, opts ...config.Option) error {
    // ...
}
// Watch 监听配置变更
func (k *KV) Watch(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
    // TODO: 监听目标配置的变更，并通过 channel 将结果传给业务层
}
```

## 如何实现 Config 接口

```go
// Provider 七彩石的 DataProvider 实现
type Provider struct {
    kv *KV
}
// Read 读取指定 key 的配置
func (p *Provider) Read(path string) ([]byte, error) {
    // TODO: 读取目标路径配置
}
// Watch 注册配置变更的回调函数
func (p *Provider) Watch(cb config.ProviderCallback) {
    // TODO: 注册配置变更的回调函数
}
// Name 返回 Provider name
func (p *Provider) Name() string {
    return p.kv.Name()
}
```

## 如何实现 Codec 接口

```go
// 下面就简单的实现了一个 json 的 codec
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
// init 中注册一下即可使用
RegisterCodec(&JSONCodec{})
```

## 实例代码

[tconf](https://git.woa.com/trpc-go/trpc-config-tconf)
[rainbow](https://git.woa.com/trpc-go/trpc-config-rainbow)

