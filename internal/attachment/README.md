# tRPC-Go Attachment (Large Binary Data) Transmission

tRPC protocol now supports sending attachments over simple RPC.
Attachments are binary data sent along with messages, and they will not be serialized and compressed by the framework.
So the overhead the cost of serialization, deserialization, and related memory copy can be reduced.

## Alternative Solutions

- Consider avoiding carrying large binary data in messages.
  For small binary data, the overhead of serialization, deserialization, and memory copy is not significant, and simple tRPC without attachment is sufficient.

- Consider splitting large binary data using tRPC streaming, where binary data is divided into chunks and streamed over multiple messages.

- Consider using other protocols such as [streaming http](https://gist.github.com/CMCDragonkai/6bfade6431e9ffb7fe88).