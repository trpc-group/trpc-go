## 1 前言

tRPC-Go 将常用的数据存储层接口按 tRPC-Go 的 client 调用模式封装了一遍，自动集成名字服务，监控，调用链，mock 能力，不需要用户自己二次开发。
所以当你要调用 redis，mysql，mongodb 等接口时，`千万不要自己到github上下载源码，并引入到业务代码中自己调用`。自己引用开源库，就需要自己重新封装一遍，很容易出各种 bug，而且可能还会有一些安全风险。

## 2 原理

封装存储层接口主要利用了 tRPC-Go 的`transport可插拔能力`，将默认的 client transport 替换成自己实现的 transport，在该 transport 内部其实也是引用自开源库，所以`当你遇到存储层接口返回的错误信息时，应当首先到谷歌上搜索具体原因`，跟 tRPC-Go 框架无关。

## 3 实现

tRPC-Go 的所有存储接口调用模式大致如下：

```go
proxy := xxx.NewClientProxy("trpc.${app}.${server}.${service}") 
rsp, err := proxy.Do(ctx, req)
```

由于存储服务接口不是通过 pb 自动生成的桩代码，所以需要在调用`NewClientProxy`时，自己指定存储服务的 service name。框架会通过这个 service name 到配置文件里面寻找 client 的配置信息，service name 的规范建议是`trpc.${app}.${server}.${service}`。框架默认使用北极星寻址，如果存储服务也可以通过北极星寻址的话，那么这个 name 直接填写北极星服务名即可，如果不是北极星寻址的话就应该自己设置`target`。

```yaml
client:
  service:
    - name: trpc.${app}.${server}.${service}  # NewClientProxy 填的 service_name 参数，如果存储服务有北极星名字地址，这里可以直接填北极星名字
      namespace: Production   # 存储服务所处的环境 Production 正式环境 Development 测试环境，cl5 只有正式环境
      target: polaris://service_name  # 存储服务的地址 具体要看存储服务对外的名字服务 如 cl5://sid  cmlb://appid ip://vip:port
      timeout: 1000  # 调用该存储服务允许的超时时间
```

## 4 示例

更多具体存储接口的使用示例都在[这里](https://git.woa.com/trpc-go/trpc-database)

### 4.1 redis

redis 示例见[这里](https://git.woa.com/trpc-go/trpc-database/tree/master/redis)

### 4.2 mysql

mysql 示例见[这里](https://git.woa.com/trpc-go/trpc-database/tree/master/mysql)

### 4.3 ckv

ckv 示例见[这里](https://git.woa.com/trpc-go/trpc-database/tree/master/ckv)

### 4.4 dcache

dcache 示例见[这里](https://git.woa.com/trpc-go/trpc-database/tree/master/dcache)

## 5 FAQ

### Q1：NewClientProxy 可以全局只 new 一次，所有请求共用吗？

可以，没问题，`proxy是并发安全的`，你可以在程序入口定义一个全局变量，如`var mysqlproxy = mysql.NewClientProxy("xxx")`，也可以每次请求都 NewClientProxy。更推荐的做法是定义成 impl struct 里面的成员变量，方便依赖注入和 mock 测试。

### Q2：数据库地址配置如何管理？

所有的数据库地址都推荐使用配置中心管理，可以参考[这里](https://git.woa.com/trpc-go/trpc-config-tconf)，`数据库地址、密码等属于敏感信息，一定不能写到代码里面提交到git上`。
没有配置中心的话，也可以使用框架配置的 client 区块。
readme 里面 demo 的 option 只是一个示例，代表可以这样设置参数，但是正常开发服务完全不需要自己设置任何 option，tconf 会默认注册，业务代码只需 NewClientProxy("dbname") 即可。

### Q3：数据库配置 target 如何填写？

target 格式是 `selector://servicename` 。
框架默认使用北极星寻址，如果数据库已经支持北极星，则 name 直接填写北极星服务名，target 不用填。
每个数据库的地址格式不一样，需要具体看 readme 里面的示例。

### Q4：mysql 如何配置读写分离？

client 配置两个 service（在哪里配置？见上面 Q2），每个 service 对应读和写，然后实例化两个 client proxy 即可，如：

```yaml
client:
  service:
    - name: trpc.mysql.xxx.read
      target: xxxx1
      timeout: 1000
    - name: trpc.mysql.xxx.write
      target: xxxx2
      timeout: 2000
```

```golang
reader := mysql.NewClientProxy("trpc.mysql.xxx.read")
writer := mysql.NewClientProxy("trpc.mysql.xxx.write")

// 读写不同的逻辑调用不同的proxy
reader.QueryToStructs(ctx, "select xxxx")
//...
writer.Exec(ctx, "insert xxxx")

```

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
