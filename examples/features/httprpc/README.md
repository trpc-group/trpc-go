## HTTP RPC

This example demonstrates the use of HTTP RPC Service in tRPC, including custom field JSON aliases in proto files.

## Usage

### 1. Generate stub code from proto file

```bash
trpc create -p ./proto/echo/echo.proto -o ./proto/echo --alias --protocol http --api-version 2 --rpconly --mock=false --nogomod=true
```

### 2. Start server

```bash
cd ./server
go run .
```

The server log will be displayed as follows:

```text
2024-02-29 15:20:05.617 INFO    server/service.go:176   process:63060, http service:trpc.examples.echo.Echo launch success, tcp:127.0.0.1:8090, serving ...
```

### 3. Send http post request

- curl

```bash
curl -H "Content-Type: application/json" -X POST "http://127.0.0.1:8090/unaryecho" -d '{"message": "hello"}'
```

or:

```bash
curl -H "Content-Type: application/json" -X POST "http://127.0.0.1:8090/unaryecho" -d '{"message_json": "hello"}'
```

The client log will be displayed as follows:

```text
"2024-02-29 15:24:27.956 INFO    client/main.go:19       response code: 0, response message: hello".
```

- stub code

```bash
cd ./client
go run .
```

The client log will be displayed as follows:

```text
2024-05-21 16:14:13.934 INFO    client/main.go:23       response code: 0, response message: hello
2024-05-21 16:14:13.935 INFO    client/main.go:36       response: {"code":0,"message":"hello"}
```

## Explanation

For more Information, please refer to:

- [Building a Generic HTTP RPC Service with tRPC-Go](/http/README.md#pan-http-rpc-service)
