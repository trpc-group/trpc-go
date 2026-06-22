# Stop

## Generate stub code

```bash
trpc create -p ./proto/echo/echo.proto --api-version 2 --rpconly -o ./proto/echo --protodir . --mock=false --nogomod
```

## Build and Start server

```bash
go build -o server/trpc-server server/main.go

server/trpc-server -conf server/trpc_go.yaml
```

## Build and Start client

```bash
go build -o client/trpc-client client/main.go
client/trpc-client
```

## Stop server gracefully

```bash
ps -ef | grep  trpc-server
kill -SIGTERM pid
```

## Restart server gracefully

```bash
ps -ef | grep  trpc-server
kill -SIGUSR2 pid
```
