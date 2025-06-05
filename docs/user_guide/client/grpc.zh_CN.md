## 1 前言

目前公司内部有些 grpc-go 存量服务，想逐步往 trpc-go 上迁移。第一个需求是 trpc-go client 使用 grpc 协议调用 trpc-go 现有服务，不需要框架改动。这个在 trpc-codec 中引入的 grpc。

## 2 原理

## 3 实现

grpc client 调用 trpc server，使用自带的 grpc-cli 或者 grpc client stub 桩代码去创建一个 client。使用方式和原生的 grpc client 一样。

**注意：目前 grpc client 不支持 stream 模式调用 trpc-go server**

## 4 示例

[示例地址](https://git.woa.com/trpc-go/trpc-codec/tree/master/grpc/examples)

具体 trpc-go 支持 grpc 协议调用原理和实现思路，参考[tRPC-Go 搭建 grpc 服务](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=284289174)

## 5 FAQ

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
