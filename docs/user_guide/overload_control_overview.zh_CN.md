tRPC-Go 过载保护能力主要由 trpc-robust 与 trpc-overload-ctrl 两个插件来提供，其对比如下：

|插件名 | trpc-robust | trpc-overload-ctrl |
|--|--|--|
|算法|DAGOR（优先级）|LittleLaw（并发数），优先级|
|过载时效果 | 成功率稍低，成功请求的时延与非过载时相当，CPU 不会到达 100% 的高位，而是维持到配置的 80% 的位置 | 成功率高，成功请求的时延较大，CPU 可以保持在 100% 的高位以充分利用资源|
|可解释性 | 只依赖于优先级阈值，可解释性强 | 同时依赖优先级阈值与并发数计算，其中最大并发数的计算在低负载/低 QPS 下会不准确|

当用户关注过载时成功请求的时延不受影响时，可选用 trpc-robust；当用户更关注整体的通过率，可以接受时延上升时，可选用 trpc-overload-ctrl。

以上两个插件均可通过 [tRPC 官方柔性治理平台](https://trpc.woa.com) 进行治理，详细接入方式请参考各自文档：

* [trpc-robust 插件](https://iwiki.woa.com/p/4012215462)
* [trpc-overload-control 插件](https://iwiki.woa.com/p/776262500)

测试文档见：

* [trpc-go robust 柔性测试 3](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMHiTY6uUARr00r4z237?scode=AJEAIQdfAAoHQmKSTzAGkAxgZOAFM)
* [trpc-go robust 柔性测试 2](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMvX5nd8gZT5ufi3NKx2?scode=AJEAIQdfAAoQznEuu8AGkAxgZOAFM)
* [trpc-go robust 柔性测试 1](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMgb0ok0i3QsyDEV1Ko2?scode=AJEAIQdfAAo0KKDKUDAGkAxgZOAFM)
