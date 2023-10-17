# Filter

trpc provides APIs to implement and install filter on client/server. Filter intercepts the execution of each RPC call.
Users can use filter to do logging, auth, metrics collection, and other functionality that can be shared across RPCs.

## Try it

```shell
go run server/main.go -conf server/trpc_go.yaml
```

```shell
go run client/main.go
```

## Explanation

To make the framework more scalable, tRPC introduced the interceptor concept, which is inspired by Java's
Aspect-Oriented Programming (AOP) philosophy.
The specific approach is to set breakpoints before and after specified actions in the framework,
and then insert a series of filters at the breakpoint locations to introduce system functions into the framework.

The ultimate goal of interceptors is to decouple business logic and system-level services, allowing for cohesive
development.
By presetting filter breakpoints, the program is provided with a mechanism for dynamically adding or replacing
functionality to the program without modifying the source code.

### Server-side

[`ServerFilter`](https://github.com/trpc-group/trpc-go/blob/main/filter/filter.go#L29) is the type for server-side
filter.It is essentially a function type with signature:
`func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error)`. An implementation of a
filter can usually be divided into the breakpoints.

To install a filter for Server, configure `NewServer` with ServerOption `server.WithFilter()` or
configure `trpc_go.yaml` with sever filter.

### Client-side

Client side is similar to server side.


