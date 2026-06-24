## mTLS

This example code demonstrates how to transmit the trpc protocol via mTLS. In this example, we specifically illustrate how the client and server configure and utilize certificates, private keys, and CA certificates to achieve secure mTLS transmission.

## Usage

- Start server.

```bash
go run server/main.go -conf server/trpc_go.yaml
```

- Start client.

```bash
go run client/main.go
```

- Server output

```txt
2024-07-05 17:29:04.243 DEBUG common/common.go:39 SayHi recv req:msg:"test mTLS message"
```

- Client output

```txt
2024-07-05 17:29:04.244 INFO client/main.go:29 get msg: Hi test mTLS message
```

## Explanation

To adapt to sensitive scenarios such as databases and finance, the tRPC architecture provides Token-based Knocknock authentication and mTLS authentication methods for transmitting tRPC protocols. The specific implementation is to configure and use client certificates, private keys, and Certificate Authority (CA) certificates on both the client and server sides.

### Server-side

In order for the server to support mTLS authentication, configure `NewServer`
with the ServerOption `server.WithTLS()`, or use the server configuration
`trpc_go.yaml`.

### Client-side

Client side is similar to server side.
