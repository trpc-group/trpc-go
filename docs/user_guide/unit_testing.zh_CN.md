## 前言

单元测试可以带来优秀的代码质量、良好的异常处理，所以单元测试的重要性不言而喻，tRPC-Go 从设计之初就考虑了框架的易测性，通过 pb 生成桩代码时，默认会生成 mock 代码，并且所有的 database 封装包也默认集成了 mock 能力。

golang 本身提供了完整的单元测试配套工具，使用配套的工具可以快速搭建单元测试框架。

单元测试应该测什么？单元测试应该测的是你当前服务自身内部逻辑，通过构造足够多的数据用例尽量覆盖函数内部所有分支，推荐使用 [Table Driven](https://github.com/golang/go/wiki/TableDrivenTests) 模式来实现，对于有外部依赖问题如 rpc 调用或者 db 请求的情况推荐全部使用`gomock`来解决。

下面将展示一些简单示例，有关单元测试的更多实践可参考 [PCG 代码委员会 Go 编程指南](https://iwiki.woa.com/p/4008801643)。

## 示例

### 自动生成单元测试框架

#### linux + vim-go

参考 [golang 命令行工具 gotests](https://github.com/cweill/gotests) 。

#### GoLand IDE

参考 [jetbrains 官方 GoLand Test 使用文档](https://www.jetbrains.com/help/go/testing.html)。

### 如何写 mock 代码

在执行过程中，后台服务通常都会依赖 RPC 调用，在本地执行单元测试 RPC 调用是不能调用成功的，我们可以 MOCK 函数中的 RPC 调用。
针对不同的应用场景，有两种不同的 MOCK 方式：直接 mock 调用的函数/变量；mock 接口。
trpc 框架是面向接口实现的服务框架，相对于直接 mock 函数，推荐使用接口 mock 的方式。

#### 接口 mock

接口 mock 主要有两种使用方式：

##### 接口 proxy 由外部传入或者是全局变量

接口 proxy 由外部传入或者是全局变量，这个情况只需把变量替换成 mockproxy 即可。

接口 mock 通过依赖注入的方式，将 mock 接口注入到功能函数/类中，替换掉正常访问 RPC 的接口；要 mock 接口，首先得有接口。trpc-go 框架是面向接口的，框架提供的 rpc 调用基本都提供了接口 mock 的功能，使用 trpc-go 工程生成的工程，默认会生成协议接口的 mock 接口。以读取 redis 为例：

```golang
func GetRedis(ctx context.Context, client redis.Client, key string) (string, error) {
    reply, err := redigo.String(client.Do(ctx, "GET", key))
    if nil != err {
        return "", err
    }
    return reply, nil
}
```

这个函数通过 redis.Client 接口调用 Do 方法获取 key 的值，trpc-go 提供了 redis 接口的 mock:

```golang
func TestGetRedis(t *testing.T) {
    type args struct {
        ctx    context.Context
        client redis.Client
        key    string
    }
    mockCtr := gomock.NewController(t)
    defer mockCtr.Finish()
    redisMock := mockredis.NewMockClient(mockCtr) // 这里生成 redis 的 mock 变量
    redisMock.EXPECT().Do(context.Background(), "GET", "").Return("value", nil)
    
    tests := []struct {
        name    string
        args    args
        want    string
        wantErr bool
    }{
        // TODO: Add test cases.
        {
            name: "t0",
            args: args{
                ctx:    context.Background(),
                client: redisMock,
                key:    "key",
            },
        },
    }

    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := GetRedis(tt.args.ctx, tt.args.client, tt.args.key)
            if (err != nil) != tt.wantErr {
                t.Errorf("GetRedis() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("GetRedis() got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

代码第 9 行，通过 `mockredis.NewMockClient` 方法提供 `redis.Client` 的 mock 接口，在用例执行过程中，将 `redisMock` 实例作为`redis.Client` 作为入参传递给函数。

##### 接口 proxy 在函数内部临时创建

接口 proxy 在函数内部临时创建，这个时候需要使用一些打桩工具，例如 `gostub` 打桩。

```golang
    func GetRedis(ctx context.Context, key string) (string, error) {
        client := redis.NewClientProxy("trpc.redis.xxx.xxx")
        reply, err := redigo.String(client.Do(ctx, "GET", key))
        if nil != err {
            return "", err
        }
        return reply, nil
    }
```

```golang
    func TestGetRedis(t *testing.T) {
        type args struct {
            ctx    context.Context
            key    string
        }
        ctrl := gomock.NewController(t)
        defer ctrl.Finish()
        
        // 为 mock 打桩
        mockcli := mockredis.NewMockClient(ctrl)
        stubs := gostub.Stub(&redis.NewClientProxy, func(name string, opts ...client.Option) redis.Client { return mockcli })
        defer stubs.Reset()
        
        mockcli.EXPECT().Do(context.Background(), "GET", "").Return("value", nil)
        
        tests := []struct {
            name    string
            args    args
            want    string
            wantErr bool
        }{
            // TODO: Add test cases.
            {
                name: "t0",
                args: args{
                    ctx:    context.Background(),
                    key:    "key",
                },
            },
        }
    
        
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got, err := GetRedis(tt.args.ctx, tt.args.key)
                if (err != nil) != tt.wantErr {
                    t.Errorf("GetRedis() error = %v, wantErr %v", err, tt.wantErr)
                    return
                }
                if got != tt.want {
                    t.Errorf("GetRedis() got = %v, want %v", got, tt.want)
                }
            })
        }
    }
```

### 如何生成接口 mock

tRPC-Go 本身已经默认生成了 rpc 和 database 的 mock 接口，这里是针对其他用户自己写的接口的情况，需要用户自己生成 mock 代码，可以参考 [go mock 官方文档](https://github.com/golang/mock) 来生成 mock 接口。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
