[English](flatbuffers.md) | 中文

# 背景

[flatbuffers](https://flatbuffers.dev/) 简介：由 Google 推出的序列化库，主要用于游戏、移动端场景，作用类似于 protobuf
其主要优点有：
- 可以飞速地访问序列化后的数据（序列化之后无需反序列化即可访问数据，其 Unmarshal 操作仅仅只是将 byte slice 拿出来了而已，对字段的访问类似于虚表机制：查偏移量然后定位数据），事实上，flatbuffers 的 Marshal 以及 Unmarshal 都很轻量，真正的序列化步骤都推到了构造的时候，所以它的构造占了总时间的很大比例
- 由于它不需要反序列化即可访问字段，因此这很适合只需访问少量字段的情况，比如只是需要一个大消息某几个字段，protobuf 必须把整个消息反序列化才能对这几个字段访问成功，而 flatbuffers 不需要
- 对内存高效使用，不需要频繁分配内存：这一点主要是和 protobuf 进行对比，protobuf 在序列化以及反序列化的时候需要分配内存来放置中间的临时结果，而 flatbuffers 在初始构造之后，序列化以及反序列化时都不再需要另外分配内存
- 性能压测可以发现，flatbuffers 在数据量较大时，性能优于 protobuf

小结：
所有操作前推到构造阶段，使得 Marshal 和 Unmarshal 操作很轻量  

经 benchmark 测试，可得耗时占比：  
- Protobuf 在构造阶段约占 20%（总共包括构造+Marshal+Unmarshal）  
- Flatbuffers 则占 90%  

缺点：
- 修改一个已经构造好后的 flatbuffer 较为麻烦
- 构造数据的 API 较难使用

# 原理

![flatbuffers](/.resources_without_git_lfs/user_guide/server/flatbuffers/flatbuffers_zh_CN.png)

# 示例
首先安装最新版本 [trpc-cmdline](https://github.com/trpc-group/trpc-cmdline) 工具

然后使用该工具来生成 flatbuffers 对应的桩代码，目前已经支持单发单收、服务端/客户端流式、双向流式等

我们通过一个简单的例子来走一遍所有的流程

首先定义 IDL 文件，语法可以从 flatbuffers 官网上进行学习，整体的结构和 protobuf 非常相似，一个例子如下：
```idl
namespace trpc.testapp.greeter; // 相当于 protobuf 中的 package

// 相当于 protobuf 的 go_package 声明
// 注意：attribute 本身是 flatbuffers 的标准语法，里面加 "go_package=xxx" 这种写法则是通过 trpc-cmdline 中实现的自定义支持
attribute "go_package=github.com/trpcprotocol/testapp/greeter";

table HelloReply { // table 相当于 protobuf 中的 message
  Message:string;
}

table HelloRequest {
  Message:string;
}

rpc_service Greeter {
  SayHello(HelloRequest):HelloReply; // 单发单收
  SayHelloStreamClient(HelloRequest):HelloReply (streaming: "client"); // 客户端流式
  SayHelloStreamServer(HelloRequest):HelloReply (streaming: "server"); // 服务端流式
  SayHelloStreamBidi(HelloRequest):HelloReply (streaming: "bidi"); // 双向流式
}

// 含有两个 service 时的示例
rpc_service Greeter2 {
  SayHello(HelloRequest):HelloReply;
  SayHelloStreamClient(HelloRequest):HelloReply (streaming: "client");
  SayHelloStreamServer(HelloRequest):HelloReply (streaming: "server");
  SayHelloStreamBidi(HelloRequest):HelloReply (streaming: "bidi");
}
```
其中，go_package 字段的含义类似 protobuf 中对应部分的含义，见 https://developers.google.com/protocol-buffers/docs/reference/go-generated#package

以上链接中点出 protobuf 中的 package 和 go_package 字段没有关系：

*There is no correlation between the Go import path and the package specifier in the .proto file. The latter is only relevant to the protobuf namespace, while the former is only relevant to the Go namespace.*

但是由于 flatc 的本身局限性，flatbuffers 的 IDL 文件中至少要保证 namespace 的最后一段和 go_package 的最后一段是相同的，即至少保证以下两个加粗部分是相同的：

- namespace trpc.testapp.greeter;
- attribute "go_package=github.com/trpcprotocol/testapp/greeter";

然后使用如下命令可以生成对应的桩代码：

```sh
$ trpc create --fbs greeter.fbs -o out-greeter --mod github.com/testapp/testgreeter
```
其中 --fbs 指定了 flatbuffers 的文件名（带相对路径），-o 指定了输出路径，--mod 指定了生成文件 go.mod 中 package 的内容，假如没有 --mod 的话，它会寻找当前目录下的 go.mod 文件，以该文件中的 package 内容作为 --mod 的内容，这个表示的是服务端本身的模块路径标识，和 IDL 文件中的 go_package 不同，后者标识的是桩代码的模块路径标识

生成的代码目录结构如下：

```sh
├── cmd/client/main.go # 客户端代码
├── go.mod
├── go.sum
├── greeter_2.go       # 第二个 service 的服务端实现
├── greeter_2_test.go  # 第二个 service 的服务端测试
├── greeter.go         # 第一个 service 的服务端实现
├── greeter_test.go    # 第一个 service 的服务端测试
├── main.go            # 服务启动代码
├── stub/github.com/trpcprotocol/testapp/greeter # 桩代码文件
└── trpc_go.yaml       # 配置文件
```
在一个终端内，编译并运行服务端：
```sh
$ go build      # 编译
$ ./testgreeter # 运行
```
在另一个终端内，运行客户端：
```sh
$ go run cmd/client/main.go
```
然后可以在两个终端的 log 中查看相互发送的消息

启动服务的 main.go 文件展示如下：
```go
package main
import (
    "flag"

    _ "trpc.group/trpc-go/trpc-filter/debuglog"
    _ "trpc.group/trpc-go/trpc-filter/recovery"
    trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/log"
    fb "github.com/trpcprotocol/testapp/greeter"
)
func main() {
    flag.Parse()
    s := trpc.NewServer()
    // 如果是多 service 的话需要在第一个参数明确写上 service 名，否则流式会有问题
    fb.RegisterGreeterService(s.Service("trpc.testapp.greeter.Greeter"), &greeterServiceImpl{})
    fb.RegisterGreeter2Service(s.Service("trpc.testapp.greeter.Greeter2"), &greeter2ServiceImpl{})
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```
整体内容基本和 protobuf 的生成文件相同，唯一要注意的是 serverFBBuilderInitialSize 用于设置桩代码中 service 服务端构造 rsp 时 flatbuffers.NewBuilder 的初始大小，其默认值是 1024，建议大小设置得恰好为构造完所有数据所需的大小，这样可以得到最优性能，但是在数据大小多变的情况下，设置这个大小将成为一个负担，所以建议在这里成为性能瓶颈之前保持 1024 这一默认值

服务端逻辑实现部分示例如下：
```go
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *fb.HelloRequest) (*flatbuffers.Builder, error) {
    // 单发单收 flatbuffers 处理逻辑（仅供参考，请根据需要修改）
    log.Debugf("Simple server receive %v", req)
    // 将 Message 替换为你想要操作的字段名
    v := req.Message() // Get Message field of request.
    var m string
    if v == nil {
        m = "Unknown"
    } else {
        m = string(v)
    }
    // 添加字段示例
    // 将 CreateString 中的 String 替换为你想要操作的字段类型
    // 将 AddMessage 中的 Message 替换为你想要操作的字段名
    idx := b.CreateString("welcome " + m) // 创建一个 flatbuffers 中的字符串
    b := &flatbuffers.Builder{}
    fb.HelloReplyStart(b)
    fb.HelloReplyAddMessage(b, idx)
    b.Finish(fb.HelloReplyEnd(b))
    return b, nil
}
```
构造的每一步详细解释如下：
```go
// 导入桩代码的 package
import fb "github.com/trpcprotocol/testapp/greeter"
// 首先创建一个 *flatbuffers.Builder
b := flatbuffers.NewBuilder(0) 
// 想要为结构体填充字段的话
// 首先创建一个该字段类型的对象
// 比如想要填充的字段类型为 String
// 就可以调用 b.CreateString("a string") 来创建这个字符串
// 该方法返回的是在 flatbuffer 中的 index
i := b.CreateString("GreeterSayHello")
// 想要构造一个 HelloRequest 结构体
// 需要调用桩代码中提供的 XXXXStart 方法
// 表示该结构体构造的开始
// 其相对应的结束为 fb.HelloRequestEnd 
fb.HelloRequestStart(b)
// 该填充字段的名字为 message
// 就可以调用 fb.HelloRequestAddMessage(b, i)
// 通过传入 builder 以及之前构造的字符串的 index 来构造这个 message 字段
// 其他字段可以通过这种方式不断进行构造
fb.HelloRequestAddMessage(b, i)
// 当结构体构造结束时调用 XXXEnd 方法
// 该方法会返回这个结构体在 flatbuffer 中的 index
// 然后调用 b.Finish 可以结束这个 flatbuffer 的构造
b.Finish(fb.HelloRequestEnd(b))
```
可见 flatbuffers 的构造 API 相当难用，尤其是在构造嵌套结构时

想要访问收到消息中的字段时，直接如下访问即可：

```go
req.Message() // 访问 req 中的 message 字段
```

# 性能对比
![performanceComparison](/.resources_without_git_lfs/user_guide/server/flatbuffers/performanceComparison_zh_CN.png)
压测环境是：两台 8 核，CPU 2.5G，Memory 16G 的机器
- 实现客户端循环发包工具，可发用 protobuf 进行序列化的包，也可发用 flatbuffers 进行序列化的包
- 固定起 goroutine 的数量是 500，每次压测时间 50s
- 图上的每个点都是 flatbuffers 和 protobuf 交替测试三次取的各自均值（没画标准差是因为发现三个值差别并不大，画上标准差根本看不出来，所以只画了均值）
- 横坐标是字段的数量，vector 中的每个元素单独作为一个字段进行技术，字段类型均匀覆盖了所有基本类型
- 左纵坐标表示 QPS，右纵坐标表示在不同字段数下的 p99 耗时
- 从这个表中可以看出，当没有 map 字段时，当总字段数量变多时，flatbuffers 的性能会优于 protobuf
- 在字段数较少时之所以 flatbuffers 的性能会差是因为 flatbuffers 初始 builder 里 byte slice 大小统一初始化为 1024，因此当字段数较少时仍然需要分配这么大的空间，造成浪费（protobuf 不会这样），因此性能比 protobuf 差，这一点可以通过预先调节初始 byte slice 大小来缓解，但这对业务来说有一定的负担，因此在压测时统一设置初始大小为 1024

![performanceComparison2](/.resources_without_git_lfs/user_guide/server/flatbuffers/performanceComparison2_zh_CN.png)

- Protobuf 的 map 序列化反序列化性能很差，从图中可见一般
- 由于 flatbuffers 中没有 map 类型，使用的是 vector of key value pair 的形式进行替代，key value 的类型保持和 protobuf 中 map 的 key value 类型一致
- 可以看到当字段数量变多时，flatbuffers 的性能提升更加明显

![performanceComparison3](/.resources_without_git_lfs/user_guide/server/flatbuffers/performanceComparison3_zh_CN.png)

- 从图中可见总字段数较多时，flatbuffers 性能都会好于 protobuf，尤其是在 map 存在的情况下
- 横坐标选取的是不含 map 时的字段数量，对于 with map 这条线来说，它每个点对应的横坐标要再大一点
- 这些字段数量依次对应的发包大小为：

| 是否含 map | 序列化方式 |  |  |  |  |  |  |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 否 | flatbuffers | 284 | 708 | 1124 | 1964 | 3644 | 7243 |
| 否 | protobuf | 167 | 519 | 871 | 1573 | 2973 | 5834 |
| 是 | flatbuffers | 292 | 1084 | 1900 | 3540 | 6819 | 13619 |
| 是 | protobuf | 167 | 659 | 1171 | 2192 | 4232 | 8494 |


# FAQ
##  Q1: .fbs 文件中 include 了其他文件，如何生成桩代码？

参考 https://github.com/trpc-group/trpc-cmdline/tree/main/testcase/flatbuffers 中的下面几个使用示例：

- 2-multi-fb-same-namespace: 在同一目录下有多个 .fbs 文件，每个 .fbs 文件的 namespace 都是一样的（flatbuffers 中的 namespace 等同于 protobuf 中的 package 语句），然后其中一个主文件 include 了其他 .fbs 文件
- 3-multi-fb-diff-namespace: 在同一个目录下有多个 .fbs 文件，每个 .fbs 文件的 namespace 不一样，比如定义 RPC 的主文件中引用了不同 namespace 中的类型
- 4.1-multi-fb-same-namespace-diff-dir: 多个 .fbs 文件的 namespace 相同，但是在不同的目录下，主文件 helloworld.fbs 中在 include 其他文件时使用相对路径，可以看下 run4.1.sh，其中并不需要用 --fbsdir 来指定搜索路径
- 4.2-multi-fb-same-namespace-diff-dir: 除了 helloworld.fbs 文件中 include 语句里面只使用文件名以外，其余和 4.1 完全相同，这个例子想要正确运行，需要添加 --fbsdir 来指定搜索路径，见 run4.2.sh：
  ```sh
  trpc create --fbsdir testcase/flatbuffers/4.2-multi-fb-same-namespace-diff-dir/request \
            --fbsdir testcase/flatbuffers/4.2-multi-fb-same-namespace-diff-dir/response \
            --fbs testcase/flatbuffers/4.2-multi-fb-same-namespace-diff-dir/helloworld.fbs \
            -o out-4-2 \
            --mod github.com/testapp/testserver42
  ```
  所以为了尽可能简化命令行参数，建议在 include 语句时写上文件的相对路径（如果不在一个文件夹中的话）
- 5-multi-fb-diff-gopkg: 多个 .fbs 文件，多文件之间有 include 关系，他们的 go_package 不相同。注意：由于 flatc 的限制，目前不支持两个文件在 namespace 相同的情况下 go_package 却不同，并要求一个文件中的 namespace 和 go_package 的最后一段必须相同（比如 trpc.testapp.testserver 和 github.com/testapp/testserver 最后一段 testserver 是相同的）
