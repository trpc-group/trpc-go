### Attachment

tRPC protocol supports sending attachments over simple RPC. 
Attachments are binary data sent along with messages, and they will not be serialized and compressed by the framework.
So the overhead the cost of serialization, deserialization, and related memory copy can be reduced.

#### Usage

1. Start server. `go run server/main.go -conf server/trpc_go.yaml`
2. Start client. `go run client/main.go`

If the server starts successfully, you will see a INFO log containing "trpc service:trpc.examples.echo.Echo launch success, tcp:127.0.0.1:8000, serving" in the terminal.
If the client call is successful, you will see a INFO logs containing "received attachment: server attachment" in the terminal.

