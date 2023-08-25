[TOC]

# 1. Introduction

During the development process, you often need to operate on various storage systems, such as ckv, db, hippo, kafka, etc.

In order to reduce redundant code and unify the operation behavior of storage plugins, trpc-go provides an [API library](https://git.woa.com/trpc-go/trpc-database) for related storage systems.

# 2. Principles

As storage plugins can be divided into two categories: non-network calls and network calls, their design principles also differ.

## Non-Network Calls

Non-network calls generally refer to single-machine versions of storage, such as local LRU, cache, etc.

1. Firstly, an interface needs to be defined to indicate the external interface capabilities of the storage. When there is an extension in the future, the user can also refer to different storage objects through this interface.

   ![interface design](/.resources/developer_guide/develop_plugins/storage/interface_design.png)

2. When instantiating a specific plugin, there is often an optional parameter to be filled in. In this case, it is recommended to use the closure-based method to pass in optional parameters, so that users can define their own modification functions. This can be applied to use cases that many developers may not have considered.

   ```go
    // Based on input parameters to set information
    func Dosomething(timeout time.Duration) // Set timeout duration
    func Init(optionA string, optionB string, optionC string) // All input parameters need to be filled in

    // Based on closure to pass in optional parameters
    type Option func(*OptionSet)

    func New(opts ...Option) {
        // Set default values below
        options := OptionSet{
            A: "default-a",
            B: "default-b",
            C: "default-c",
        }

        for _, fun := range opts {
            fun(&options)
        }
    }

    // If you want to provide an option, say to set A
    func WithA(a string) Option {
        return func(opt *OptionSet) {
            opt.A = a
        }
    }

    // Example usage
    a = New(WithA("abc"))
   ```

> Implement the git.code.oa.com/trpc-go/trpc-go/plugin plugin to integrate with the trpc-go framework configuration for ease of use by users.

## Network Calls

Plugins that involve network calls generally refer to non-single-machine versions, such as ckv, hippo, mysql, etc., and require developers to design the client-side of the c-s model for others to use.

1. The design principles mentioned above for non-network calls are also required.
2. Using the Client interface in `git.code.oa.com/trpc-go/trpc-go/client` to operate network calls, the design process is as follows:

   ![network call process](/.resources/developer_guide/develop_plugins/storage/network_call_process.png)

```go
// The related gomod plugins are as follows:
selector plugin: git.code.oa.com/trpc-go/trpc-go/
codec plugin: git.code.oa.com/trpc-go/trpc-go/codec
transport plugin: git.code.oa.com/trpc-go/trpc-go/transport
```

# 3. Implementation

The recommended project structure for storage plugins is as follows:

```go
storagename:                   // Storage plugin package
    mockstoragename:           // Mock storage, provides mock data for storagename externally
        mock_xx.go
    examples:                 // Examples of using storagename
        xxx_demo.go
    README.md:                // Documentation
    CHANGELOG.md:             // Change log
    go.mod:                   // gomod package management tool
    owners.txt:               // Code owners
    client.go:                // Client plugin implementation
    codec.go:                 // Codec plugin implementation
    plugin.go:                // trpc plugin registration logic
    transport.go:             // Transport plugin implementation
    selector.go:              // Selector plugin implementation
    _test.go:                 // Test code
```

# 4. Examples

## Non-network Call—localcache

https://git.woa.com/trpc-go/trpc-database/tree/master/localcache

## Network Call—redis

https://git.woa.com/trpc-go/trpc-database/tree/master/redis

# 5. FQA

**Q1: When Redis configuration is in `tconf`, and the `redis.client` plugin is used to initiate a call, how to specify the configuration items?**

When the Redis configuration is not in the trpc-go framework's YAML configuration, the Redis client cannot obtain information from the framework configuration. In this case, the required configuration can be set through the opts parameter in the `redis.NewClientProxy` method:

```go
// NewClientProxy creates a new Redis backend request proxy. The service name must be passed as a required parameter: trpc.redis.xxx.xxx
var NewClientProxy = func(name string, opts ...client.Option) Client {
        c := &redisCli{
                ServiceName: name,
                Client:      client.DefaultClient,
        }
        c.opts = make([]client.Option, 0, len(opts)+2)
        c.opts = append(c.opts, opts...)
        c.opts = append(c.opts, client.WithProtocol("redis"), client.WithDisableServiceRouter())
        return c
}
```

The configuration information can refer to the framework YAML configuration:

```yaml
client:
  service:
    - name: trpc.redis.xxx.xxx
      namespace: Production
      target: polaris://xxx.test.redis.com
      password: xxxxx
      timeout: 800
    - name: trpc.redis.xxx.xxx
      namespace: Production
      target: redis+polaris://:passwd@polaris_name
      timeout: 800
```

These configuration settings belong to the `Option func(*Options)` closure parameter in type `Option func(*Options)` of https://git.woa.com/trpc-go/trpc-go/blob/master/client/options.go:

```yaml
name: WithServiceName
namespace: WithNamespace
target: WithTarget
password: WithPassword
timeout: WithTimeout
```
