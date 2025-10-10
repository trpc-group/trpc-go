# FastHTTP RPC

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