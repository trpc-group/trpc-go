// Package main demonstrates how to apply server filters configured in trpc_go.yaml
// to custom routes registered via restful.RegisterRouter when wrapping the
// pb-defined RESTful router inside a custom http.ServeMux.
//
// Run:
//
//	go run main.go
//
// In another shell:
//
//	curl http://127.0.0.1:9092/v1/greeter/hello/world  # pb route
//	curl http://127.0.0.1:9092/custom/ping             # custom mux route
//
// Both should print the in-process trace filter's "FILTER HIT" log line. Without
// restful.WrapHandlerWithServerFilters, only the pb call would print.
package main

import (
	"context"
	"net/http"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/examples/features/restful/pb"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/restful"
)

const serviceName = "trpc.test.helloworld.Greeter"

// traceFilter is a minimal server filter that prints which RPC name it observes.
// It is registered under the name "trace" in init() so trpc_go.yaml can list it
// under server.filter.
var traceFilter filter.ServerFilter = func(ctx context.Context, req any,
	next filter.ServerHandleFunc) (any, error) {
	rpcName := codec.Message(ctx).ServerRPCName()
	log.InfoContextf(ctx, "[trace] FILTER HIT rpc_name=%q", rpcName)
	return next(ctx, req)
}

func init() {
	filter.Register("trace", traceFilter, nil)
}

func main() {
	s := trpc.NewServer()
	pb.RegisterGreeterService(s.Service(serviceName), &greeterImpl{})

	// Build an http.ServeMux that routes:
	//   /v1/greeter/...  -> restful.Router (pb)
	//   /custom/ping     -> our own HandlerFunc
	r := restful.GetRouter(serviceName)
	mux := http.NewServeMux()
	mux.Handle("/", r)
	mux.HandleFunc("/custom/ping", func(w http.ResponseWriter, _ *http.Request) {
		// Write error is intentionally ignored: a client disconnect on this
		// demo ping endpoint is expected and needs no extra handling.
		_, _ = w.Write([]byte("pong"))
	})

	// Replace the framework router with the wrapped mux. WrapHandlerWithServerFilters
	// makes every custom (non-pb) route pass through the server filter chain
	// configured in trpc_go.yaml ([trace] in this example).
	restful.RegisterRouter(serviceName, restful.WrapHandlerWithServerFilters(serviceName, mux))

	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}

// greeterImpl is a tiny pb service implementation for the demo.
type greeterImpl struct {
	pb.UnimplementedGreeter
}

func (*greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "hello " + req.GetName()}, nil
}
