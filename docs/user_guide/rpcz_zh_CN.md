[TOC]

# RPCZ

RPCZ 是一个监控 RPC 的工具，记录了一次  rpc 中发生的各种事件，如序列化/反序列，压缩解压缩和执行拦截器。
RPCZ 可以帮助用户调试服务，它允许用户自行配置需要被记录的事件，用户可以通过 admin 工具可以查看记录的事件，能帮助用户快速准确的定位问题。
除此之外，由于 RPCZ 记录了 RPC 中各种事件的持续时间和收发数据包的大小，因此可以帮助用户分析超时事件，优化服务的性能。

## 术语解释

### 事件（Event）

事件（Event）[1, 2, 3] 用来描述某一特定时刻（`Event.Time`）发生了某件事情（`Event.Name`）。

```go
type Event struct {
	Name string
	Time time.Time
}
```
在一个普通 RPC 调用中会发生一系列的事件，例如发送请求的 Client 端按照时间先后顺序，一般会发生如下一系列事件：

1. 开始运行前置拦截器
2. 结束运行前置拦截器
3. 开始序列化
4. 结束序列化
5. 开始压缩
6. 结束压缩
7. 开始编码协议头部字段
8. 结束编码协议头部字段
9. 开始发送二进制文件到网络
10. 结束发送二进制文件到网络
11. 开始从网络中接收二进制文件
12. 结束从网络中接收二进制文件
13. 开始解码协议头部字段
14. 结束解码协议头部字段
15. 开始解压缩
16. 结束解压缩
17. 开始反序列化
18. 结束反序列化
19. 开始运行后置拦截器
20. 结束运行后置拦截器

而处理请求的 server 端，按照时间先后顺序，一般会发生如下一系列事件：

1. 开始解码协议头部字段
2. 结束解码协议头部字段
3. 开始解压缩
4. 结束解压缩
5. 开始反序列化
6. 结束反序列化
7. 开始运行前置拦截器
8. 结束运行前置拦截器
9. 开始运行用户自定义处理函数
10. 结束运行用户自定义处理函数
11. 开始运行后置拦截器
12. 结束运行后置拦截器
13. 开始序列化
14. 结束序列化
15. 开始压缩
16. 结束压缩
17. 开始编码协议头部字段
18. 结束解码协议头部字段
19. 开始发送二进制文件到网络
20. 结束发送二进制文件到网络

### Span

Span[4, 5] 用来描述某段时间间隔（具有开始时间和结束时间）的单个操作，例如客户端发送远程请求，服务端处理请求或函数调用。
根据划分的时间间隔大小不同，一个大的 Span 可以包含多个小的 Span，就像一个函数中可能调用多个其他函数一样，会形成树结构的层次关系。
因此一个 Span 除了包含名字、内部标识 span-id[6]，开始时间、结束时间和这段时间内发生的一系列事件（Event）外，还可能包含许多子 Span。

rpcz 中存在两种类型的 Span。
1. client-Span：描述 client 从开始发送请求到接收到回复这段时间间隔内的操作（涵盖上一节 Event 中描述的 client 端发生一系列事件）。

2. server-Span：描述 server 从开始接收请求到发送完回复这段时间间隔内的操作（涵盖上一节 Event 中描述的 server 端发生一系列事件）。
   server-Span 运行用户自定义处理函数的时候，可能会创建 client 调用下游服务，此时 server-Span 会包含多个子 client-Span。

```
server-Span
    client-Span-1
    client-Span-2
    ......
    client-Span-n
```

Span 被存储在 context 中，rpcz 会自动调用 ContextWithSpan 往 context 中存 Span，在函数调用过程中需要保证 context 中的 Span 不会丢失。

## Span 的生命周期

考察 Span 对象的生命周期，rpcz 中对 Span 的绝大多数操作，都需要考虑并发安全。
除此之外采用了 sync.Pool 和 预先分配的循环数组来降低 Span 的内存分配时对性能的影响。

### Span 的构造

rpcz 在启动时会初始化一个全局 GlobalRPCZ，用于生成和存储 Span。
在框架内部 Span 只可能在两个位置被构造，
第一个位置是在 server 端的 transport 层的 handle 函数刚开始处理接收到的请求时；
第二个位置是在 client 端的桩代码中调用 Invoke 函数开始发起 rpc 请求时。
虽然两个位置创建的 Span 类型是不同，但是代码逻辑是相似的，都会调用 rpczNewSpanContext，该函数实际上执行了三个操作
1. 调用 SpanFromContext 函数，从 context 中获取 span。
2. 调用 span.NewChild 方法，创建新的 child span。
3. 调用 ContextWithSpan 函数，将新创建的 child span 设置到 context 中。

### Span 在 context 中传递

被创建 Span 在提交前，会一直在存放在 context 中，沿着 rpc 调用的链路传递。
在调用链路上使用 `rpcz.AddEvent` 往当前 context 中的 Span 中添加新的事件。

### Span 的提交

在 server 端的 transport 层的 handle 函数处理完请求后，会调用 `ender.End()` 把 Span 提交到 GlobalRPCZ 当中。
此后虽然 context 中仍然存放着 Span，但是从语义上来说，已经调用过的 End 函数的 Span 不允许再被继续操作，其行为是未定义的。

### 在 admin 中访问 Span

admin 模块调用 `rpcz.Query` 和 `rpcz.BatchQuery` 从 GlobalRPCZ 中读取 Span。
有一点需要注意的是，admin 获取的 Span 是只读类型的 Span（ReadOnlySpan），只读类型的 Span 是由可写入的 Span 导出得到的，这样做的原因是保证并发访问安全。

### 删除多余的 Span

当哈希表中存储的 Span 过多时就需要按照某种淘汰规则，删除多余的 Span。
目前的实现是当 GlobalRPCZ 中的 Span 个数超过最大容量上限时会删除最老的 Span。

## RPCZ 名字的由来

关于 "RPCZ" 的这个名字的由来，后缀 -z 有在英文中一般有两种含义 [7]: 一是用于名词，实现单数变复数，如 Boy**z** are always trouble；二是用于动词实现动词形态的变化 He love**z** me。
总的来说，在单词后面加 -z 的效果类似于加 -s。
所以 "RPCZ" 就是指各种类型的 RPC，从一个分布式全局的调用链路视角来看的确是成立的，各种 RPC 调用存在树状的父子关系，组合成了 "RPCZ"。

"RPCZ" 这一术语最早来源于 google 内部的 RPC 框架 Stubby，在此基础上 google 在开源的 grpc 实现了类似功能的 channelz[8]，channelz 中除了包括各种 channel 的信息，也涵盖 trace 信息。
之后，百度开源的 brpc 在 google 发表的分布式追踪系统 Dapper 论文 [9] 的基础上，实现了一个非分布式的 trace 工具，模仿 channelz 取名为 brpc-rpcz[10]。
接着就是用户在使用 tRPC 中需要类似于 brpc-rpcz 的工具来进行调试和优化，所以 tRPC-Cpp 首先支持类似功能 [11, 12]，仍然保留了 RPCZ 这个名字。

最后就是在 tRPC-Go 支持类似 "RPCZ" 的功能，在实现过程中发现随着分布式追踪系统的发展，社区中出现了 opentracing[13] 和 opentelemetry[14] 的开源系统，公司内部也做起了天机阁 [15]。
tRPC-Go-RPCZ 在 span 和 event 设计上部分借鉴了 opentelemetry-trace 的 go 语言实现，可以认为是 tRPC-Go 框架内部的 trace 系统。
严格来说，tRPC-Go-RPCZ 是非分布式，因为不同服务之间没有在协议层面实现通信。
现在看来，brpc, tRPC-Cpp 和 tRPC-Go 实现的 rpcz，取名叫 spanz 或许更符合后缀 "-z" 本来的含义。

## 如何配置 rpcz

rpcz 的配置包括基本配置，进阶配置和代码配置，更多配置例子见 `config_test.go`。

### 基本配置

在 server 端配置 admin，同时在 admin 里面配置 rpcz :

```yaml
server:
  admin:
    ip: 127.0.0.1
    port: 9028
    rpcz:
      fraction: 1.0
      capacity: 10000
```

- `fraction` : 采样率，其取值范围为`[0.0, 1.0]`，默认值为 0.0 代表不采样，需要手动配置。
- `capacity`:  rpcz 的存储容量，默认值为 10000，表示最多能存储的 span 数量。

### 进阶配置

进阶配置允许你自行过滤感兴趣的 span，在使用进阶配置之前需要先了解 rpcz 的采样机制。

####  采样机制

rpcz 使用采样机制来控制性能开销和过滤你不感兴趣的 Span。
采样可能发生在 Span 的生命周期的不同阶段，最早的采样发生在 Span 创建之前，最晚的采样发生在 Span 提交之前。

##### 采样结果表

只有创建和提交之前都被采样到的 Span 才会最终被收集到 GlobalRPCZ 中，供你通过 admin 接口查询。

| 在 Span 创建之前采样？   | 在 Span 提交之前采样？ | Span 最终是否会被收集？ |
|:-----------------|:--------------:|:--------------:|
| true             |      true      |      true      |
| true             |     false      |     false      |
| false            |      true      |     false      |
| false            |     false      |     false      |

##### 在 Span 创建之前采样

只有当 Span 被采样到才会去创建 Span，否则就不需要创建 Span，也就避免了后续对 Span 的一系列操作，从而可以较大程度上减少性能开销。
采用固定采样率 [16, 17] 的采样策略，该策略只有一个可配置浮点参数 `rpcz.fraction`, 例如`rpcz.fraction` 为 0.0001，则表示每 10000（1/0.0001）个请求会采样一条请求。
当 `rpcz.fraction` 小于 0 时，会向上取 0；当 `rpcz.fraction` 大于 1 时，会向下取 1。

##### 在 Span 提交之前采样

已经创建好的 Span 会记录 rpc 中的各种信息，但是你可能只关心包含某些特定信息的 Span，例如出现 rpc 错误的 Span，高耗时的 Span 以及包含特定属性信息的 Span。
这时，就需要在 Span 最终提交前只对你需要的 Span 进行采样。
rpcz 提供了一个灵活的对外接口，允许你在服务在启动之前，通过配置文件设置 `rpcz.record` 字段来自定义 Span 提交之前采样逻辑。

```yaml
server:
  admin:
    rpcz:
      record_when:
        error_codes: [0,]      
        min_duration: 1000ms   # ms or s
        sampling_fraction: 1        # [0.0, 1.0]
```

- `error_codes`：只采样包含其中任意一个错误码的 span，例如 0(RetOk), 21(RetServerTimeout)。
- `min_duration`:  只采样持续时间超过 `min_duration` 的 span，可用于耗时分析。
- `sampling_fraction` : 采样率，取值范围为 `[0, 1]`

#### 配置举例

##### 对包含错误码为 1(RetServerDecodeFail) 或持续时间大于 1s 的  span 进行提交

```yaml
server:
  admin:
    ip: 127.0.0.1
    port: 9028
    rpcz:
      fraction: 1.0
      capacity: 10000
      record_when:
        error_codes: 1       
        min_duration: 1000ms
        sampling_fraction: 1         
```

##### 对包含错误码为 1(RetServerDecodeFail) 或 21(RetServerTimeout) 的或持续时间大于 2s 的 span 以 1/2 的概率进行提交

```yaml
server:
  admin:
    ip: 127.0.0.1
    port: 9028
    rpcz:
      fraction: 1.0
      capacity: 10000
      record_when:
        error_codes: [1, 21]     
        min_duration: 2s
        sampling_fraction: 0.5      
```

### 代码配置

在读取配置文件之后且在服务启动前，可以通过 `rpcz.GlobalRPCZ` 来配置 rpcz，此时的提交采样逻辑需要实现 `ShouldRecord` 函数。

```go
// ShouldRecord determines if the Span should be recorded.
type ShouldRecord = func(Span) bool
```

##### 只对包含 "SpecialAttribute" 属性的 Span 进行提交

```go
const attributeName = "SpecialAttribute"
rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{
    Fraction: 1.0,
    Capacity: 1000,
    ShouldRecord: func(s rpcz.Span) bool {
        _, ok = s.Attribute(attributeName)
        return ok
    },
})
```

### 查询最近提交的多条 span 的概要信息

查询最近 num 个 span 的概要信息，可以访问如下的 url:

```html
http://ip:port/cmds/rpcz/spans?num=xxx
```

例如执行 `curl http://ip:port/cmds/rpcz/spans?num=2` ，则会返回如下 2 个 span 的概要信息：

```html
1:
  span: (client, 65744150616107367)
    time: (Dec  1 20:57:43.946627, Dec  1 20:57:43.946947)
    duration: (0, 319.792µs, 0)
    attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, <nil>)
2:
  span: (server, 1844470940819923952)
    time: (Dec  1 20:57:43.946677, Dec  1 20:57:43.946912)
    duration: (0, 235.5µs, 0)
    attributes: (RequestSize, 125),(ResponseSize, 18),(RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, success)
```

每个 span 的概要信息和如下的模版匹配：

```html
span: (name, id)
time: (startTime, endTime)
duration: (preDur, middleDur, postDur)
attributes: (name1, value1) (name2, value2)
```

其中每个字段的含义解释如下：

- name：span 的名字
- id：span 唯一标识，可通过它查询具体的某个 span 详细信息
- startTime：span 的创建时间
- endTime：span 的提交时间，当 span 未被提交成功时，该字段值为 "unknown"
- duration：包含个时间段来描述 currentSpan 和 parentSpan 的耗时
  - preDur: currentSpan.startTime - parentSpan.startTime
  - middleDur：currentSpan.endTime - currentSpan.startTime，当 currentSpan.endTime 为 "unknown" 时，middleDur 的值也为 "unknown"
  - postDur：parentSpan.endTime - currentSpan.endTime，当 parentSpan.endTime 或 currentSpan.endTime 为 "unknown" 时，postDur 的值也为 "unknown"
- attributes：span 的属性值，每一个属性由（属性名，属性值）组成，通常会显示下面三个属性
  - RequestSize：请求包大小（byte）
  - ResponseSize：响应包大小（byte）
  - RPCName：对端的服务名 + 接口名 (/trpc.app.server.service/method)
  - Error：错误信息，根据框架返回码判断请求是否成功，success 或 nil 表示成功

如果不指定查询的个数，则下列查询将会默认返回最近提交成功的 [^1] 10 个 span 的概要信息：

```html
http://ip:port/cmds/rpcz/spans
```

[^1]: **最近提交的 span 并不是严格按照时间来排序的，可能存在多个 goroutine 同时提交 span，是按照最近提交成功的 span 来排序。**

### 查询某个 span 的详细信息

查询包含 id  的 span 的详细信息，可以访问如下的 url：

```html
http://ip:port/cmds/rpcz/spans/{id}
```

例如执行 `curl http://ip:port/cmds/rpcz/spans/6673650005084645130` 可查询 span id 为 6673650005084645130 的 span 的详细信息：

```
span: (server, 6673650005084645130)
  time: (Dec  2 10:43:55.295935, Dec  2 10:43:55.399262)
  duration: (0, 103.326ms, 0)
  attributes: (RequestSize, 125),(ResponseSize, 18),(RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, success)
  span: (DecodeProtocolHead, 6673650005084645130)
    time: (Dec  2 10:43:55.295940, Dec  2 10:43:55.295952)
    duration: (4.375µs, 12.375µs, 103.30925ms)
  span: (Decompress, 6673650005084645130)
    time: (Dec  2 10:43:55.295981, Dec  2 10:43:55.295982)
    duration: (45.875µs, 791ns, 103.279334ms)
  span: (Unmarshal, 6673650005084645130)
    time: (Dec  2 10:43:55.295982, Dec  2 10:43:55.295983)
    duration: (47.041µs, 334ns, 103.278625ms)
  span: (filter1, 6673650005084645130)
    time: (Dec  2 10:43:55.296161, Dec  2 10:43:55.399249)
    duration: (225.708µs, 103.088ms, 12.292µs)
    event: (your annotation at pre-filter, Dec  2 10:43:55.296163)
    span: (filter2, 6673650005084645130)
      time: (Dec  2 10:43:55.296164, Dec  2 10:43:55.399249)
      duration: (2.75µs, 103.085ms, 250ns)
      event: (your annotation at pre-filter, Dec  2 10:43:55.296165)
      span: (server.WithFilter, 6673650005084645130)
        time: (Dec  2 10:43:55.296165, Dec  2 10:43:55.399249)
        duration: (1.208µs, 103.083625ms, 167ns)
        event: (your annotation at pre-filter, Dec  2 10:43:55.296165)
        span: (, 6673650005084645130)
          time: (Dec  2 10:43:55.296166, Dec  2 10:43:55.399249)
          duration: (792ns, 103.082583ms, 250ns)
          span: (HandleFunc, 6673650005084645130)
            time: (Dec  2 10:43:55.296177, Dec  2 10:43:55.399249)
            duration: (11.583µs, 103.070917ms, 83ns)
            event: (handling EmptyCallF, Dec  2 10:43:55.296179)
            span: (client, 6673650005084645130)
              time: (Dec  2 10:43:55.296187, Dec  2 10:43:55.297871)
              duration: (9.125µs, 1.684625ms, 101.377167ms)
              attributes: (RPCName, /trpc.testing.end2end.TestTRPC/UnaryCall),(Error, <nil>)
              span: (filter1, 6673650005084645130)
                time: (Dec  2 10:43:55.296192, Dec  2 10:43:55.297870)
                duration: (5.292µs, 1.678542ms, 791ns)
                span: (client.WithFilter, 6673650005084645130)
                  time: (Dec  2 10:43:55.296192, Dec  2 10:43:55.297870)
                  duration: (542ns, 1.677875ms, 125ns)
                  span: (selector, 6673650005084645130)
                    time: (Dec  2 10:43:55.296193, Dec  2 10:43:55.297870)
                    duration: (541ns, 1.677209ms, 125ns)
                    span: (CallFunc, 6673650005084645130)
                      time: (Dec  2 10:43:55.296200, Dec  2 10:43:55.297869)
                      duration: (7.459µs, 1.668541ms, 1.209µs)
                      attributes: (RequestSize, 405),(ResponseSize, 338)
                      span: (Marshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.296202, Dec  2 10:43:55.296341)
                        duration: (1.375µs, 138.875µs, 1.528291ms)
                      span: (Compress, 6673650005084645130)
                        time: (Dec  2 10:43:55.296341, Dec  2 10:43:55.296341)
                        duration: (140.708µs, 333ns, 1.5275ms)
                      span: (EncodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.296342, Dec  2 10:43:55.296345)
                        duration: (141.458µs, 3.333µs, 1.52375ms)
                      span: (SendMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.297540, Dec  2 10:43:55.297555)
                        duration: (1.339375ms, 15.708µs, 313.458µs)
                      span: (ReceiveMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.297556, Dec  2 10:43:55.297860)
                        duration: (1.355666ms, 303.75µs, 9.125µs)
                      span: (DecodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.297862, Dec  2 10:43:55.297865)
                        duration: (1.661916ms, 2.5µs, 4.125µs)
                      span: (Decompress, 6673650005084645130)
                        time: (Dec  2 10:43:55.297866, Dec  2 10:43:55.297866)
                        duration: (1.665583ms, 167ns, 2.791µs)
                      span: (Unmarshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.297866, Dec  2 10:43:55.297868)
                        duration: (1.666041ms, 1.709µs, 791ns)
            span: (sleep, 6673650005084645130)
              time: (Dec  2 10:43:55.297876, unknown)
              duration: (1.698709ms, unknown, unknown)
              event: (awake, Dec  2 10:43:55.398954)
            span: (client, 6673650005084645130)
              time: (Dec  2 10:43:55.398979, Dec  2 10:43:55.399244)
              duration: (102.80125ms, 265.417µs, 4.25µs)
              attributes: (RPCName, /trpc.testing.end2end.TestTRPC/UnaryCall),(Error, <nil>)
              span: (filter2, 6673650005084645130)
                time: (Dec  2 10:43:55.398986, Dec  2 10:43:55.399244)
                duration: (6.834µs, 258.25µs, 333ns)
                span: (client.WithFilter, 6673650005084645130)
                  time: (Dec  2 10:43:55.398987, Dec  2 10:43:55.399244)
                  duration: (1.708µs, 256.458µs, 84ns)
                  span: (selector, 6673650005084645130)
                    time: (Dec  2 10:43:55.398988, Dec  2 10:43:55.399244)
                    duration: (417ns, 255.916µs, 125ns)
                    span: (CallFunc, 6673650005084645130)
                      time: (Dec  2 10:43:55.399005, Dec  2 10:43:55.399243)
                      duration: (16.833µs, 238.375µs, 708ns)
                      attributes: (RequestSize, 405),(ResponseSize, 338)
                      span: (Marshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.399006, Dec  2 10:43:55.399017)
                        duration: (1.792µs, 10.458µs, 226.125µs)
                      span: (Compress, 6673650005084645130)
                        time: (Dec  2 10:43:55.399017, Dec  2 10:43:55.399017)
                        duration: (12.583µs, 167ns, 225.625µs)
                      span: (EncodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.399018, Dec  2 10:43:55.399023)
                        duration: (12.958µs, 4.917µs, 220.5µs)
                      span: (SendMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.399041, Dec  2 10:43:55.399070)
                        duration: (36.375µs, 29.083µs, 172.917µs)
                      span: (ReceiveMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.399070, Dec  2 10:43:55.399239)
                        duration: (65.75µs, 168.25µs, 4.375µs)
                      span: (DecodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.399240, Dec  2 10:43:55.399241)
                        duration: (235.417µs, 1.375µs, 1.583µs)
                      span: (Decompress, 6673650005084645130)
                        time: (Dec  2 10:43:55.399242, Dec  2 10:43:55.399242)
                        duration: (237µs, 125ns, 1.25µs)
                      span: (Unmarshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.399242, Dec  2 10:43:55.399243)
                        duration: (237.292µs, 750ns, 333ns)
        event: (your annotation at post-filter, Dec  2 10:43:55.399249)
      event: (your annotation at post-filter, Dec  2 10:43:55.399249)
    event: (your annotation at post-filter, Dec  2 10:43:55.399249)
  span: (Marshal, 6673650005084645130)
    time: (Dec  2 10:43:55.399250, Dec  2 10:43:55.399251)
    duration: (103.314625ms, 1.208µs, 10.167µs)
  span: (Compress, 6673650005084645130)
    time: (Dec  2 10:43:55.399252, Dec  2 10:43:55.399252)
    duration: (103.315958ms, 125ns, 9.917µs)
  span: (EncodeProtocolHead, 6673650005084645130)
    time: (Dec  2 10:43:55.399252, Dec  2 10:43:55.399253)
    duration: (103.316208ms, 750ns, 9.042µs)
  span: (SendMessage, 6673650005084645130)
    time: (Dec  2 10:43:55.399253, Dec  2 10:43:55.399261)
    duration: (103.317333ms, 8.333µs, 334ns)
```

在 span 的详细信息中新增了 `event` 字段，以及内嵌的子 span。

- event:  描述了在某一时刻发生的事情，类似于日志。
  可以由你自行插入的事件，如上面例子中的 `Nov  4 14:39:23.594147: your annotation at pre-filter`。
- span：在 server 处理你自定义函数时，可能会创建新的 client 调用下游服务，此时会创建子 span
  可以看到，所有的 子 span 都是在 `HandleFunc` 内发生的。

需要注意的是，endTime、duration 中的 middleDur 和 postDur 的值可能为 "unknown"，例如上面的 span 中包含如下的子 span：

```
span: (sleep, 6673650005084645130)
time: (Dec  2 10:43:55.297876, unknown)
duration: (1.698709ms, unknown, unknown)
event: (awake, Dec  2 10:43:55.398954)
```

## Span 接口

你可以先调用 `rpcz.SpanFromContext`[^2] 获取当前 `context` 中的 `Span`，然后使用下面的接口来操作 Span。

```go
type Span interface {
	// AddEvent adds a event.
	AddEvent(name string)

	// SetAttribute sets Attribute with (name, value).
	SetAttribute(name string, value interface{})
	
	// ID returns SpanID.
	ID() SpanID

	// NewChild creates a child span from current span.
	// Ender ends this span if related operation is completed. 
	NewChild(name string) (Span, Ender)
}
```

[^2]: 当 `context` 中不含有任何 `span` 的时候会返回一个 `noopSpan`，对 `noopSpan` 的任何操作都是空操作，不会生效。

### 使用 AddEvent 来添加事件

```go
// If no Span is currently set in ctx an implementation of a Span that performs no operations is returned.
span := SpanFromContext(ctx context.Context)

span.AddEvent("Acquiring lock")
mutex.Lock()

span.AddEvent("Got lock, doing work...")
// do some stuff ...

span.AddEvent("Unlocking")
mutex.Unlock()
```

### 使用 SetAttribute 来设置属性

```go
ctx, msg := codec.EnsureMessage(ctx)
span := SpanFromContext(ctx context.Context)
span.SetAttribute("RPCName", msg.ClientRPCName())
span.SetAttribute("Error", msg.ClientRspErr())
```

### 创建新的子 Span

**特别需要注意的是：创建的子 Span 应该由调用者只调用一次 end.End() 函数来结束子 Span 的生命周期，未调用 End 和 多次调用 End 的行为是未定义的**

```go
span := SpanFromContext(ctx context.Context)
cs, end := span.NewChild("Decompress")
reqBodyBuf, err := codec.Decompress(compressType, reqBodyBuf)
end.End()
```

## 参考

- [1] https://en.wikipedia.org/wiki/Event_(UML)
- [2] https://en.wikipedia.org/wiki/Event_(computing)
- [3] https://opentelemetry.io/docs/instrumentation/go/manual/#events
- [4] https://opentelemetry.io/docs/instrumentation/go/api/tracing/#starting-and-ending-a-span
- [5] https://opentelemetry.io/docs/concepts/observability-primer/#spans
- [6] span-id 用 8 字节的数组表示，满足 w3c trace-context specification. https://www.w3.org/TR/trace-context/#parent-id
- [7] https://en.wiktionary.org/wiki/-z#English
- [8] https://github.com/grpc/proposal/blob/master/A14-channelz.md
- [9] Dapper, a Large-Scale Distributed Systems Tracing Infrastructure: http://static.googleusercontent.com/media/research.google.com/en//pubs/archive/36356.pdf
- [10] brpc-rpcz: https://github.com/apache/incubator-brpc/blob/master/docs/cn/rpcz.md
- [11] tRPC-Cpp rpcz wiki. todo
- [12] tRPC-Cpp rpcz proposal. https://git.woa.com/trpc/trpc-proposal/blob/master/L17-cpp-rpcz.md
- [13] opentracing: https://opentracing.io/
- [14] opentelemetry: https://opentelemetry.io/
- [15] https://tpstelemetry.pages.woa.com/
- [16] 天机阁 2.0-sdk-go：https://git.woa.com/opentelemetry/opentelemetry-go-ecosystem/blob/master/sdk/trace/dyeing_sampler.go
- [17] open-telemetry-sdk-go- traceIDRatioSampler: https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/sampling.go