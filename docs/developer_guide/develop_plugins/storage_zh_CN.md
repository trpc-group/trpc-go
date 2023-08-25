# 1. 前言

在平时的开发过程中，大家总会对 ckv、db、hippo、kafka 等存储进行操作。为了减少重复代码，统一存储插件的操作行为，trpc-go 提供了相关存储的 api 库，代码路径：https://git.woa.com/trpc-go/trpc-database

# 2. 原理

因为存储插件可以分为非网络调用和网络调用两大类，其设计原理也有所不同。

## 非网络调用

非网络调用一般指的是单机版本的存储，如本地 LRU、cache 等。

1. 需要先定义一个接口，用于注明存储的对外接口能力，同时后续有扩展的时候，使用方也通过此接口来引用不同的存储对象。

   ![接口设计](/.resources/developer_guide/develop_plugins/storage/interface_design.png)

2. 实例化具体的插件时，往往会存在填写 optional 类型的入参，此时建议基于闭包的方式传入可选参数，用户可以自行定义修改函数，可以适用于很多开发者都没有考虑到的用例：

   ```go
    // 基于入参设置信息
    func Dosomething(timeout time.Duration) // 设置超时时间
    func Init(optionA string,optionB string,optionC string) // 需要填写所有入参
    // 基于闭包传入可选参数
    type Option func(*OptionSet)
    func New(opts ...Option) {
        //下面设置默认值
        options:=OptionSet{
            A:"default-a",
            B:"default-b",
            C:"default-c",
        }
        for _,fun:=range opts{
                fun(&options)
        }
    }
    //如果需要提供 option 选项，比方说设置 A
    func WithA(a string) Option{
        return func(opt *OptionSet){
            opt.A=a
        }
    }
    // 使用的时候
    a=New(WithA("abc"))
   ```

> 实现 plugin 插件 git.code.oa.com/trpc-go/trpc-go/plugin, 用于和 trpc-go 框架配置打通，便于使用方引入。

## 网络调用

涉及网络调用的插件一般值的是非单机版本，入 ckv、hippo、mysql 等，需要开发者设计 c-s 模型中的 client 端供他人使用。

1. 需要上述提到的非网络调用的设计原理。2.利用 git.code.oa.com/trpc-go/trpc-go/client 的 Client 接口操作网络调用，其设计的流程如下：
   ![网络调用流程](/.resources/developer_guide/develop_plugins/storage/network_call_process_zh_CN.png)

```go
// 相关插件 gomod 如下
selector 插件：git.code.oa.com/trpc-go/trpc-go/
codec 插件：git.code.oa.com/trpc-go/trpc-go/codec
transport 插件：git.code.oa.com/trpc-go/trpc-go/transport
```

# 3. 实现

存储插件的工程结构建议如下：

```go
  storagename: // 存储插件包
  ---------mockstoragename: // mock 存储，对外提供 storagename 的 mock 数据
  ------------------mock_xx.go
  ---------examples:// storagename 的使用 demo
  ------------------xxx_demo.go
  ---------README.md: // 说明文档
  ---------CHANGELOG.md: // 变更文档
  ---------go.mod: // gomod 包管理工具
  ---------owners.txt: // 代码负责人
  ---------client.go: // client 插件实现
  ---------codec.go: // codec 插件实现
  ---------plugin.go: // trpc 插件注册逻辑
  ---------transport.go: // transport 插件实现
  ---------selector.go: // selector 插件实现
  ---------_test.go: // 测试代码
```

# 4. 示例

## 非网络调用--localcache

https://git.woa.com/trpc-go/trpc-database/tree/master/localcache

## 网络调用--redis

https://git.woa.com/trpc-go/trpc-database/tree/master/redis

# 5. FQA

**Q1: redis 的配置放在 tconf 中，使用插件中的 redis.client 发起调用，应该如何指定配置项？**

当 redis 的配置不在 trpc-go 的框架 yaml 配置时，redis client 不能从框架配置中获取信息，此时可以通过 `redis.NewClientProxy` 方法中的 opts 入参设置所需配置：

```go
// NewClientProxy 新建一个 redis 后端请求代理 必传参数 redis 服务名：trpc.redis.xxx.xxx
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

配置信息可以参考框架 yaml 配置：

```yaml
client:  # 客户端调用的后端配置
  service:  # 针对单个后端的配置
    - name: trpc.redis.xxx.xxx
      namespace: Production
      target: polaris://xxx.test.redis.com
      password: xxxxx
      timeout: 800  # 当前这个请求最长处理时间
    - name: trpc.redis.xxx.xxx
      namespace: Production
      # redis+polaris 表示 target 为 uri，其中 uri 中的 host 会进行北极星解析，uri 方式支持多种参数，
      # 详见：https://www.iana.org/assignments/uri-schemes/prov/redis
      target: redis+polaris://:passwd@polaris_name
      timeout: 800  # 当前这个请求最长处理时间
```

这些配置信息的设置方法属于 https://git.woa.com/trpc-go/trpc-go/blob/master/client/options.go 的 type `Option func(*Options)` 闭包入参：

```yaml
name: WithServiceName
namespace: WithNamespace
target: WithTarget
password: WithPassword
timeout: WithTimeout
```
