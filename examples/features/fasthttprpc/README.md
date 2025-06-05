# FastHTTP RPC

This example demonstrates the use of HTTP RPC Service in tRPC, and [how to use custom field json alias in proto file](https://iwiki.woa.com/p/490796254#42-%E8%87%AA%E5%AE%9A%E4%B9%89%E5%AD%97%E6%AE%B5-json-%E5%88%AB%E5%90%8D).

# Usage

## 1. Generate stub code from proto file

```bash
trpc create -p ./proto/echo/echo.proto -o ./proto/echo --alias --protocol http --api-version 2 --rpconly --mock=false --nogomod=true
```

## 2. Start server

```bash
cd ./server
go run .
```

The server log will be displayed as follows:

```log
2024-08-19 15:41:08.629 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=32: CPU quota undefined
2024-08-19 15:41:08.630 INFO    server/service.go:202   process: 108853, fasthttp service: trpc.examples.echo.Echo launch success, tcp: 127.0.0.1:8091, serving ...
```

## 3. Send http post request

- curl

```bash
curl -H "Content-Type: application/json" -X POST "http://127.0.0.1:8091/unaryecho" -d '{"message": "hello"}'
```

The client log will be displayed as follows:

```log
{"code":219,"message":"hello"}

```

- stub code

```bash
cd ./client
go run .
```

The client log will be displayed as follows:

```log
2024-08-19 15:42:27.985 INFO    client/main.go:28       response code: 219, response message: hello
2024-08-19 15:42:27.985 INFO    client/main.go:42       response: {"code":219,"message":"hello"}
2024-08-19 15:42:27.985 INFO    client/main.go:60       response: {"code":219,"message":"hello"}
2024-08-19 15:42:27.986 INFO    client/main.go:71       response code: 219, response message: hello
```

# Explanation

For more Information, please refer to:

- [Building a Generic HTTP RPC Service with tRPC-Go](https://iwiki.woa.com/pages/viewpage.action?pageId=490796254)