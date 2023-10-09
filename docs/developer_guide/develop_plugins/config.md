English | [中文](config.zh_CN.md)

# How to develop a config type plugin

This guide will introduce how to develop a config type plugin that relies on configuration for loading.

The `config` package provides two sets of different configuration interfaces, `config.DataProvider` and `config.KVConfig`. 
This guide uses the development of the `KVConfig` type configuration as an example, and the development of the `DataProvider` type configuration is similar.

Developing this plugin requires implementing the following two sub-functions:

- Implement loading the plugin by relying on configuration. For details, please refer to [plugin](/plugin/README.md).
- Implement the `config.KVConfi`g interface and register the implementation with the `config` package.

The following steps will introduce the relevant development steps using [trpc-config-etcd](https://github.com/trpc-ecosystem/go-config-etcd) as an example.

## Implement loading the plugin by relying on configuration

### 1. Determine the plugin configuration

The following is an example of setting the "Endpoint" and "Dialtimeout" configurations in the "trpc_go.yaml" configuration file:

```yaml
plugins:                 
  config:
    etcd:
      endpoints:
        - localhost:2379
      dialtimeout: 5s
```

The plugin is encapsulated based on [etcd-client](https://github.com/etcd-io/etcd/tree/main/client/v3), so the complete configuration is in [Config](https://github.com/etcd-io/etcd/blob/client/v3.5.9/client/v3/config.go#L26).

### 2. Implement the plugin.Factory interface

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

### 3. Call plugin.Register to register the plugin with the plugin package

```go
func init() {
	plugin.Register(pluginName, NewPlugin())
}
```

## Implement the `config.KVConfi`g interface and register the implementation with the `config` package.

### 1. Implement the `config.KVConfig` interface

The plugin currently only supports Watch and Get read operations and does not support Put and Del write operations.

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

### 2. Register the implementation of `config.KVConfig` with the config package

The `*etcdPlugin.Setup` function has already called `config.Register` and `config.SetGlobalKV`.

```go
config.SetGlobalKV(c)
config.Register(c)
```