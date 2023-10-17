English | [中文](attachment.zh_CN.md)

#  tRPC-Go Attachment (Large Binary Data) Transmission

tRPC protocol now supports sending attachments over simple RPC.
Attachments are binary data sent along with messages, and they will not be serialized and compressed by the framework.
So the overhead the cost of serialization, deserialization, and related memory copy can be reduced.
[code example](/examples/features/attachment).

## Alternative Solutions

- Consider avoiding carrying large binary data in messages.
  For small binary data, the overhead of serialization, deserialization, and memory copy is not significant, and simple tRPC without attachment is sufficient.

- Consider splitting large binary data using tRPC streaming, where binary data is divided into chunks and streamed over multiple messages.
  For more details, refer to the [example of streaming data](/examples/features/stream).

- Consider using other protocols such as [streaming http](https://gist.github.com/CMCDragonkai/6bfade6431e9ffb7fe88).
  For more usage examples, refer to [client-server sending and receiving HTTP chunked](/http/README.md#client-and-server-sending-and-receiving-http-chunked).