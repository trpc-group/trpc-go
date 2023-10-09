# tRPC-Go 附件（大二进制数据）传输

tRPC 协议支持通过简单 RPC 发送附件。
附件是与消息一起发送的二进制数据，框架不会对它们进行序列化和压缩。
因此可以减少序列化、反序列化和相关内存拷贝的开销。
[代码示例](../../examples/features/attachment)。

## 其他方案

- 考虑避免在消息中携带大二进制数据，对于较小的二进制数据，序列化，反序列化和内存拷贝开销并不大，使用简单的 RPC 是足够的。

- 考虑使用 tRPC 流式分割大二进制数据，其中二进制数据被分块并通过多个消息进行流式传输，更多详细信息，可以参考[流式传输数据的例子](../../examples/features/stream)。

- 考虑使用其他协议如[流式 http](https://gist.github.com/CMCDragonkai/6bfade6431e9ffb7fe88),  更多使用上的例子，可参考 [客户端服务端收发 HTTP chunked](../../http/README.zh_CN.md#%E5%AE%A2%E6%88%B7%E7%AB%AF%E6%9C%8D%E5%8A%A1%E7%AB%AF%E6%94%B6%E5%8F%91-http-chunked) 。