// Package main is the main package.
package main

import (
	"context"
	"net/http"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

type (
	// greeterMessageReq is request struct.
	greeterMessageReq struct {
		Message string `json:"message"`
	}
	// greeterRsp is response struct.
	greeterRsp struct {
		Message string `json:"message"`
	}
)

var greeterHttpProxy = thttp.NewClientProxy("greeterRestfulService")

func main() {
	// init trpc server
	_ = trpc.NewServer()

	// get trpc context
	ctx := trpc.BackgroundContext()

	// get /v1/greeter/hello/{name}
	callGreeterHello(ctx)

	// get /v1/greeter/message/{name=messages/*}
	callGreeterMessageSubfield(ctx)

	// patch /v1/greeter/message/{message_id}
	callGreeterUpdateMessageV1(ctx)

	// patch /v2/greeter/message/{message_id}
	callGreeterUpdateMessageV2(ctx)
}

// callGreeterHello restful request greeter service
func callGreeterHello(ctx context.Context) {
	var rsp greeterRsp
	err := greeterHttpProxy.Get(ctx, "/v1/greeter/hello/trpc-restful", &rsp)
	if err != nil {
		log.Fatalf("get /v1/greeter/hello/trpc-restful http.err:%s", err.Error())
	}
	// want: [restful] SayHello Hello trpc-restful
	log.Infof("helloRsp : %v", rsp.Message)
}

// callGreeterMessageSubfield restful request greeter service
func callGreeterMessageSubfield(ctx context.Context) {
	var rsp greeterRsp
	err := greeterHttpProxy.Get(ctx, "/v1/greeter/message/messages/trpc-restful-wildcard?sub.subfield=wildcard", &rsp)
	if err != nil {
		log.Fatalf("get /v1/greeter/message/messages/trpc-restful-wildcard http.err:%s", err.Error())
	}
	// want: [restful] Message name:messages/trpc-restful-wildcard,subfield:wildcard
	log.Infof("messageWildcardRsp : %v", rsp.Message)
}

// callGreeterUpdateMessageV1 restful request greeter service
func callGreeterUpdateMessageV1(ctx context.Context) {
	var rsp greeterRsp
	var reqBody = greeterMessageReq{
		Message: "trpc-restful-patch",
	}
	header := &thttp.ClientReqHeader{
		Method: http.MethodPatch,
	}
	header.AddHeader("ContentType", "application/json")
	err := greeterHttpProxy.Patch(ctx, "/v1/greeter/message/123", reqBody, &rsp, client.WithReqHead(header))
	if err != nil {
		log.Fatalf("patch /v1/greeter/message/123 http.err:%s", err.Error())
	}
	// want: [restful] UpdateMessage message_id:123,message:trpc-restful-patch
	log.Infof("updateMessageRsp : %v", rsp.Message)
}

// callGreeterUpdateMessageV2 restful request greeter service
func callGreeterUpdateMessageV2(ctx context.Context) {
	var rsp greeterRsp
	var reqBody = greeterMessageReq{
		Message: "trpc-restful-patch-v2",
	}
	header := &thttp.ClientReqHeader{
		Method: http.MethodPatch,
	}
	header.AddHeader("ContentType", "application/json")
	err := greeterHttpProxy.Patch(ctx, "/v2/greeter/message/123", reqBody, &rsp, client.WithReqHead(header))
	if err != nil {
		log.Fatalf("patch /v2/greeter/message/123 http.err:%s", err.Error())
	}
	// want: [restful] UpdateMessage message_id:123,message:trpc-restful-patch
	log.Infof("updateMessageV2Rsp : %v", rsp.Message)
}
