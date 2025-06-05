# tRPC-Go 分场景压测

## 压测结果报表

压测结果展示在 [DataTalk 平台](https://beacon.woa.com/datatalk/pg/dashboard/256532)，你可能需要申请权限进行查看，后续迁移到 trpc.woa.com 平台进行展示。

##  测试链路

根据被测服务在调用链路的位置分为两种情况：

- 压测工具直接发送请求给被测服务，被测服务直接返回结果给压测工具，调用链路如下：
```
压测工具 ⇄ 被测服务
```

- 压测工具直接发送请求给被测服务，被测服务发送请求给下游服务，然后下游将结果返回给被测服务，被测服务再将结果返回给压测工具。
这里的被测服务，又被叫做中转服务。
调用链路如下：

```
压测工具 ⇄ 被测服务（中转服务） ⇄ 下游服务
```

### 被测试服务和下游服务代码

[trpc-go 分场景压测代码](https://git.woa.com/amdahliu/trpc-benchmark/tree/bench/trpc-go-benchmark/scenario-based-stress-testing)

### 压测工具

- 普通 rpc 测试工具：[rpc_press](https://git.woa.com/trpc-cpp/trpc-cpp/tree/master/trpc/tools/rpc_press)
- 流式 rpc 测试工具： [stream_pressure_client](https://git.woa.com/trpc-cpp/trpc-cpp-performance-testing/tree/master/test/stream)

## 压测环境和压测流水线

"压测工具"， "被测服务"和"下游服务"位于不同的机器。

### 虚拟机测试环境

| 机器   | ip            | cpu  | 内存  | 机型  | 虚拟机 |
|------|---------------|------|-----|-----|-----|
| 压测工具 | 9.146.137.169 | 8 核  | 16G | Intel(R) Xeon(R) Platinum 8255C CP |  KVM   |
| 被测服务 | 9.146.137.171 | 8 核  | 16G | Intel(R) Xeon(R) Platinum 8255C CP |  KVM   |
| 下游服务 | 9.146.137.152 | 8 核  | 16G | Intel(R) Xeon(R) Platinum 8255C CP |  KVM   |

- [异步模式-8核16G虚拟机-压测流水线](https://devops.woa.com/console/pipeline/pcgtrpcproject/p-1a33a0e53e604514a6172b7c336ee0f4/history/history/8?page=1&pageSize=20)

- [同步模式-8核16G-虚拟机-压测流水线](https://devops.woa.com/console/pipeline/pcgtrpcproject/p-c22b7d39357042fe8cb015e4478c1c07/history/history/13?page=1&pageSize=20)

- [流式-8核16G-虚拟机-压测流水线](https://devops.woa.com/console/pipeline/pcgtrpcproject/p-7f282a2e7a4e48aa84fd2d1457efed5f/history/history/2?page=1&pageSize=20)

### 物理机测试环境

