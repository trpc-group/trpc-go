[English](config.md) | 中文

# 怎么开发一个 config 类型的插件

本指南将介绍如何开发一个依赖配置进行加载的 config 类型的插件。 

`config` 包提供两套不同的配置接口，`config.DataProvider` 和  `config.KVConfig`。
本指南以开发 `KVConfig` 类型的配置为例，`DataProvider` 类型的配置与之类似。

开发该插件需要实现以下两个子功能：

- 实现插件依赖配置进行加载，详细说明请参考 [plugin](/plugin/README_CN.md)
- 实现 `config.KVConfig` 接口，并将实现注册到 `config` 包

下面以 [trpc-config-etcd](https://github.com/trpc-ecosystem/go-config-etcd) 为例，来介绍相关开发步骤。

## 实现插件依赖配置进行加载

### 1. 确定插件的配置

下面是在 "trpc_go.yaml" 配置文件中设置 "Endpoint" 和 "Dialtimeout" 的配置示例：

```yaml
plugins:                 
  config:
    etcd:
      endpoints:
        - localhost:2379
      dialtimeout: 5s
```

```go
const (
    pluginName = "etcd"
    pluginType = "config"
)
```

插件是基于[etcd-client](https://github.com/etcd-io/etcd/tree/main/client/v3) 封装的, 因此完整的配置见 [Config](https://github.com/etcd-io/etcd/blob/client/v3.5.9/client/v3/config.go#L26)。

### 2. 实现 `plugin.Factory` 接口

```go
// etcdPlugin etcd Configuration center plugin.
type etcdPlugin struct{}

// Type implements plugin.Factory.
func (p *etcdPlugin) Type() string {
    return pluginType
}

// Setup implements plugin.Factory.
func (p *etcdPlugin) Setup(name string, decoder plugin.Decoder) error {
    cfg := clientv3.Config{}
    err := decoder.Decode(&cfg)
    if err != nil {
        return err
    }
    c, err := New(cfg)
    if err != nil {
        return err
    }
    config.SetGlobalKV(c)
    config.Register(c)
    return nil
}
```

### 3. 调用 `plugin.Register` 把插件自己注册到 `plugin` 包

```go
func init() {
	plugin.Register(pluginName, NewPlugin())
}
```

## 实现 `config.KVConfig` 接口，并将实现注册到 `config` 包

### 1. 实现 `config.KVConfig` 接口

插件暂时只支持 Watch 和 Get 读操作，不支持 Put和 Del 写操作。

```go
// Client etcd client.
type Client struct {
    cli *clientv3.Client
}

// Name returns plugin name.
func (c *Client) Name() string {
    return pluginName
}

// Get Obtains the configuration content value according to the key, and implement the config.KV interface.
func (c *Client) Get(ctx context.Context, key string, _ ...config.Option) (config.Response, error) {
    result, err := c.cli.Get(ctx, key)
    if err != nil {
        return nil, err
    }
    rsp := &getResponse{
        md: make(map[string]string),
    }

    if result.Count > 1 {
    // TODO: support multi keyvalues
        return nil, ErrNotImplemented
    }

    for _, v := range result.Kvs {
        rsp.val = string(v.Value)
    }
    return rsp, nil
}

// Watch monitors configuration changes and implements the config.Watcher interface.
func (c *Client) Watch(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
    rspCh := make(chan config.Response, 1)
    go c.watch(ctx, key, rspCh)
    return rspCh, nil
}

// watch adds watcher for etcd changes.
func (c *Client) watch(ctx context.Context, key string, rspCh chan config.Response) {
    rch := c.cli.Watch(ctx, key)
    for r := range rch {
        for _, ev := range r.Events {
            rsp := &watchResponse{
                val:       string(ev.Kv.Value),
                md:        make(map[string]string),
                eventType: config.EventTypeNull,
            }
            switch ev.Type {
                case clientv3.EventTypePut:
                    rsp.eventType = config.EventTypePut
                case clientv3.EventTypeDelete:
                    rsp.eventType = config.EventTypeDel
                default:
            }
            rspCh <- rsp
        }
    }
}

// ErrNotImplemented not implemented error
var ErrNotImplemented = errors.New("not implemented")

// Put creates or updates the configuration content value to implement the config.KV interface.
func (c *Client) Put(ctx context.Context, key, val string, opts ...config.Option) error {
    return ErrNotImplemented
}

// Del deletes the configuration item key and implement the config.KV interface.
func (c *Client) Del(ctx context.Context, key string, opts ...config.Option) error {
    return ErrNotImplemented
}
```

### 2. 将实现的 `config.KVConfig` 注册到 config 包

`*etcdPlugin.Setup` 函数中已经调用了 `config.Register` 和 `config.SetGlobalKV`。

```go
config.SetGlobalKV(c)
config.Register(c)
```
