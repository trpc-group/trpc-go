# Fileupload
trpc-go 支持流式 RPC，通过流式 RPC，客户端和服务器可以建立连续连接，连续发送和接收数据，让服务器提供连续的响应。

这里是一个文件上传的例子，通过流式 RPC 进行通信。
## Usage

* 启动server服务端.
```shell
$ go run server/main.go -conf server/trpc_go.yaml 
```

* 启动client客户端.
```shell
$ go run client/main.go -conf client/trpc_go.yaml
```

client控制台会输出如下日志：
```text
Upload status: true, message: File ITerm2_v3.4_icon.png uploaded successfully
```

最后你可以在server目录发现多了一个`ITerm2_v3.4_icon.png`文件，至此，文件上传demo已经大致通过