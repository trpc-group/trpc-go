## Keep Order Client

## Usage

Start server:

```shell
cd examples/features/keeporderclient
cd server
go run . 
```

Start client:

```shell
cd examples/features/keeporderclient
cd client
go run .

# Expect output:
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 1: expect 1, got 1
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 2: expect 1 2, got 1 2
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 3: expect 1 2 3, got 1 2 3
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 4: expect 1 2 3 4, got 1 2 3 4
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 5: expect 1 2 3 4 5, got 1 2 3 4 5
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 6: expect 1 2 3 4 5 6, got 1 2 3 4 5 6
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 7: expect 1 2 3 4 5 6 7, got 1 2 3 4 5 6 7
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 8: expect 1 2 3 4 5 6 7 8, got 1 2 3 4 5 6 7 8
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 9: expect 1 2 3 4 5 6 7 8 9, got 1 2 3 4 5 6 7 8 9
2024-10-17 16:08:47.661 INFO    client/main.go:61       [SUCCESS] count 10: expect 1 2 3 4 5 6 7 8 9 10, got 1 2 3 4 5 6 7 8 9 10
```

Keep point:

* Use multiplexed mode at client side and specify each host with only one connection.

```go
import "trpc.group/trpc-go/trpc-go/pool/multiplexed"

proxy := proto.NewPlayerClientProxy(client.WithMultiplexedPool(multiplexed.New(multiplexed.WithConnectNumber(1))))
```

* Use `proxy.KeepOrderXxx` method which is generated in newer version of trpc-go-cmdline to issue keep-order requests.

```shell
trpc upgrade 
trpc create -p proto/player.proto --rpconly --nogomod --mock=false -o proto --keeporder
```
