### SendOnly

- Asynchronous Tasks, Generally it's used for udp async sending.
- Notifications and Events

#### Usage

1. Start server. `go run server/main.go -conf server/trpc_go.yaml`
2. Start client. `go run client/main.go`

If the server starts successfully, you will see a INFO log containing "trpc service: trpc.examples.sendonly.greeter launch success" in the terminal.
If the client call is successful, you will see a INFO logs containing "send only successfully, message of response is empty" in the terminal.
