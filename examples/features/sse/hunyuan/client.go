//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package main is an example of setting up an HTTP client that uses SSE to receive HunYuan API data.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"

	"github.com/google/uuid"
	"github.com/r3labs/sse/v2"
)

func main() {
	// Call the AppCreate API and append the response.
	if err := autoCallAppCreate(); err != nil {
		log.Fatalf("autoCallAppCreate err: %v", err)
	}

	// Call the AppCreate API, and manually handle the response body for proxy.
	if err := manualCallAppCreate(); err != nil {
		log.Fatalf("manualCallAppCreate err: %v", err)
	}
}

// The following example shows how to set up an HTTP client that uses SSE to receive HunYuan API data.
// For more information about the param configuration, please refer to https://iwiki.woa.com/space/HunyuanaideTaiij

// AppCreateRequest HunYuan AppCreate API Request struct.
type AppCreateRequest struct {
	Query          string    `json:"query"`
	ForwardService string    `json:"forward_service"`
	QueryId        string    `json:"query_id"`
	Stream         bool      `json:"stream"`
	Messages       []Message `json:"messages"`
	// ... other query parameters
}

// Message defines which Role presents what Content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AppCreateResponse HunYuan AppCreate API Response struct.
type AppCreateResponse struct {
	Created    int64          `json:"created"`
	ID         string         `json:"id"`
	Model      string         `json:"model"`
	Version    string         `json:"version"`
	Choices    []Choice       `json:"choices"`
	SearchInfo map[string]any `json:"search_info"`
	Processes  map[string]any `json:"processes"`
	Usage      Usage          `json:"usage"`
}

// Choice define the candidate data.
type Choice struct {
	Delta Delta `json:"delta"`
}

// Delta defines the content of the candidate.
type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage defines the extra usage data.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Example of auto call AppCreate API.
func autoCallAppCreate() error {
	// This is an example of AppCreate API base on office network.
	// For more detail about the protocol, url, ip:port and network, etc.,
	// Please pay attention to iWiki of Prepare Environment:
	// https://iwiki.woa.com/p/4008515885#%E7%8E%AF%E5%A2%83%E5%87%86%E5%A4%87
	target := "dns://stream-server-online-openapi.turbotke.production.polaris:8080"
	cli := thttp.NewClientProxy(
		"hunyuan_openapi",
		client.WithNetwork("tcp"),
		client.WithProtocol("http"),
		client.WithTarget(target),
	)

	header := http.Header{}
	// Please replace the *** with real Authorization Token.
	// You can refer to the iWiki of AppCreate API:
	// https://iwiki.woa.com/p/4008515885#AppCreate
	header.Set("Authorization", "Bearer ****")
	header.Set("Accept", "text/event-stream") // Indicate that we want to receive SSE.
	header.Set("Cache-Control", "no-cache")
	header.Set(thttp.Connection, "keep-alive")
	header.Set("Content-Type", "application/json")
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
		Header: header,
	}

	var data []byte
	rspHead := &thttp.ClientRspHeader{
		// Set ManualReadBody to false in order to handle the stream response automatically.
		ManualReadBody: false, // Default is false.
		// Register SSEHandler to the callback in order to handle the stream response
		SSEHandler: &sseHandler{func(e *sse.Event) error {
			log.Debugf("e.Event: %s; e.Data %s\n", e.Event, e.Data)
			var r AppCreateResponse
			if err := json.Unmarshal(e.Data, &r); err != nil {
				return fmt.Errorf("sse unmarshal err: %v", err)
			}
			if len(r.Choices) == 0 {
				return fmt.Errorf("no choices in response: %q", string(e.Data))
			}
			data = append(data, r.Choices[0].Delta.Content...)
			return nil
		}},
	}

	// Construct a request.
	q := AppCreateRequest{
		Query:          "给我推荐几首歌曲",
		ForwardService: "hyaide-application-1480",
		QueryId:        uuid.New().String(),
		Stream:         true,
		Messages:       []Message{},
	}
	qb, err := json.Marshal(q)
	if err != nil {
		return fmt.Errorf("marshal err: %v", err)
	}
	fmt.Printf("marshal query: %q\n", qb)

	req := &codec.Body{Data: qb}
	rsp := &codec.Body{}
	const path = "/openapi/app_platform/app_create"
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	err = cli.Post(ctx, path, req, rsp,
		// Set SerializationType to noop in order to process the raw data.
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead))
	if err != nil {
		return fmt.Errorf("post err: %v", err)
	}

	// The framework will handle SSE automatically.
	fmt.Printf("data: \n%q\n", data)
	return nil
}

type sseHandler struct {
	fn func(e *sse.Event) error
}

func (h *sseHandler) Handle(e *sse.Event) error {
	return h.fn(e)
}

// Example of manual call AppCreate API.
func manualCallAppCreate() error {
	// This is an example of AppCreate API base on office network.
	// For more detail about the protocol, url, ip:port and network, etc.,
	// Please pay attention to iWiki of Prepare Environment:
	// https://iwiki.woa.com/p/4008515885#%E7%8E%AF%E5%A2%83%E5%87%86%E5%A4%87
	target := "dns://stream-server-online-openapi.turbotke.production.polaris:8080"
	cli := thttp.NewClientProxy(
		"hunyuan_openapi",
		client.WithNetwork("tcp"),
		client.WithProtocol("http"),
		client.WithTarget(target),
	)

	header := http.Header{}
	// Please replace the *** with real Authorization Token.
	// You can refer to the iWiki of AppCreate API:
	// https://iwiki.woa.com/p/4008515885#AppCreate
	header.Set("Authorization", "Bearer ****")
	header.Set("Accept", "text/event-stream") // Indicate that we want to receive SSE.
	header.Set("Cache-Control", "no-cache")
	header.Set(thttp.Connection, "keep-alive")
	header.Set("Content-Type", "application/json")
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
		Header: header,
	}

	rspHead := &thttp.ClientRspHeader{
		// Set ManualReadBody to true in order to handle the raw stream data in the response.
		ManualReadBody: true,
	}

	// Construct a request.
	q := AppCreateRequest{
		Query:          "给我推荐几首歌曲",
		ForwardService: "hyaide-application-1480",
		QueryId:        uuid.New().String(),
		Stream:         true,
		Messages:       []Message{},
	}
	qb, err := json.Marshal(q)
	if err != nil {
		return fmt.Errorf("marshal err: %v", err)
	}
	fmt.Printf("marshal query: %q\n", qb)

	req := &codec.Body{Data: qb}
	rsp := &codec.Body{}
	const path = "/openapi/app_platform/app_create"
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	err = cli.Post(ctx, path, req, rsp,
		// Set SerializationType to noop in order to process the raw stream data.
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead))
	if err != nil {
		return fmt.Errorf("post err: %v", err)
	}

	body := rspHead.Response.Body
	defer body.Close()

	// You can do some extra work such as understanding, and proxy the raw stream data to another sse client.
	// Here just use io.Copy to read the raw stream data and print it to stdout.
	if _, err := io.Copy(os.Stdout, body); err != nil {
		return fmt.Errorf("copy body err: %v", err)
	}

	return nil
}
