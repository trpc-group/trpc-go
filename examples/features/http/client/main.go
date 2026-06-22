package main

import (
	"context"
	"net/http"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Perform a POST request using HTTP protocol.
	httpNoProtocolPost()

	// Perform a GET request using HTTP protocol.
	httpNoProtocolGet()
}

// httpNoProtocolPost sends an HTTP POST request using tRPC-Go framework.
// Note: tRPC-Go framework configuration loading is omitted here assuming it's already loaded in a typical RPC handler.
func httpNoProtocolPost() {
	// Create a ClientProxy, set the protocol to HTTP, and use Noop serialization.
	httpCli := thttp.NewClientProxy("trpc.app.server.stdhttp",
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:8080"),
	)

	// Create a ClientReqHeader with the specified HTTP method (POST)
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}

	// Add a custom "request" header to the HTTP request header
	reqHeader.AddHeader("request", "test")

	// Create ClientRspHeader to store the response header
	rspHead := &thttp.ClientRspHeader{}

	// Create a Body containing the request data
	req := &codec.Body{Data: []byte("Hello, I am stdhttp client!")}

	// Create an empty Body to store the response data
	rsp := &codec.Body{}

	// Send a HTTP POST request
	if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Warnf("Error getting thttp response: %d, err", err)
		return
	}

	// Get the "reply" field from the HTTP response header
	replyHead := rspHead.Response.Header.Get("reply")
	log.Infof("Data is \"%s\", response head is \"%s\"", string(rsp.Data), replyHead)
}

// httpNoProtocolGet sends an HTTP GET request using tRPC-Go framework.
func httpNoProtocolGet() {
	// Create a ClientProxy, set the protocol to HTTP, and use Noop serialization
	httpCli := thttp.NewClientProxy("trpc.app.server.stdhttp",
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:8080"),
	)

	// Create a ClientReqHeader with the specified HTTP method (GET)
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodGet,
	}

	// Add a custom "request" header to the HTTP request header
	reqHeader.AddHeader("request", "test")

	// Create ClientRspHeader to store the response header
	rspHead := &thttp.ClientRspHeader{}

	// Create an empty Body to store the response data
	rsp := &codec.Body{}

	// Send an HTTP GET request
	if err := httpCli.Get(context.Background(), "/v1/hello", rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Warnf("Error getting thttp response: %d, err", err)
		return
	}

	// Get the "reply" field from the HTTP response header
	replyHead := rspHead.Response.Header.Get("reply")
	log.Infof("Data is \"%s\", response head is \"%s\"", string(rsp.Data), replyHead)
}
