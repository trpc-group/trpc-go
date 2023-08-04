# Http

This example demonstrates the use of http protocol in tRPC.

## Usage

* Start server.
```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Curl request.
```sh
curl -X POST -d '{"msg":"hello"}' -H "Content-Type:application/json" http://127.0.0.1:8000/trpc.test.helloworld.Greeter/SayHello
```

The server log will be displayed as follows:
```
2023-06-12 11:27:55.440 INFO    server/service.go:164   process:68073, http service:trpc.test.helloworld.Greeter launch success, tcp:127.0.0.1:8000, serving ...
2023-06-12 11:28:00.456 DEBUG   server/main.go:21       SayHello recv req:msg:"hello"
```

## Explanation
For more Information, please refer to:

- [Building a Generic HTTP Standard Service with tRPC-Go](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)
- [Building a Generic HTTP RPC Service with tRPC-Go](https://iwiki.woa.com/pages/viewpage.action?pageId=490796254)
