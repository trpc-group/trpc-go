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

package restful_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
)

func TestPBSerializer(t *testing.T) {
	// Create an instance of the struct with some data.
	sampleData := &helloworld.HelloRequest{
		Name: "nobody",
		SingleNested: &helloworld.NestedOuter{
			Name: "anybody",
		},
		PrimitiveDoubleValue: float64(1.23),
		Time: &timestamppb.Timestamp{
			Seconds: int64(111111111),
		},
		EnumValue: helloworld.NumericEnum_ONE,
		OneofValue: &helloworld.HelloRequest_OneofString{
			OneofString: "oneof",
		},
		MappedStringValue: map[string]string{
			"foo": "bar",
		},
	}

	// Create a new HTTP server with dynamic port allocation.
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		w.Header().Set("Content-Type", accept)
		serializer := restful.GetSerializer(accept)
		data, _ := serializer.Marshal(sampleData)
		w.Write(data)
	})

	// Get a free port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start the server in a goroutine.
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("ListenAndServe(): %v", err)
		}
	}()
	defer server.Shutdown(context.Background())

	// Create an HTTP client and send a request.
	client := &http.Client{}

	// Give the server a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Define the test cases.
	testContentTypes := []string{
		"application/octet-stream",
		"application/protobuf",
		"application/x-protobuf",
		"application/pb",
		"application/proto",
	}

	// Iterate over the test cases.
	for _, testContentType := range testContentTypes {
		t.Run(testContentType, func(t *testing.T) {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/hello", port), nil)
			require.NoError(t, err)
			req.Header.Set("Accept", testContentType)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Assert the Content-Type header.
			require.Equal(t, testContentType, resp.Header.Get("Content-Type"),
				"resp.Content-Type should match the req.Accept:"+testContentType)

			// Read the response body.
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Get the serializer.
			serializer := restful.GetSerializer(testContentType)

			// Unmarshal the response body.
			var receivedData helloworld.HelloRequest
			require.NoError(t, serializer.Unmarshal(body, &receivedData))

			// Assert the deserialized data.
			// Note: "sampleData.sizeCache and receivedData.sizeCache may not be equal,
			// so proto.Equal() should be used for comparison."
			require.True(t, proto.Equal(sampleData, &receivedData),
				"receivedData and sampleData should be equal:"+testContentType)
		})
	}
}

func TestProtoSerializerNames(t *testing.T) {
	// Define the test cases table.
	testContentTypes := []string{
		"application/octet-stream",
		"application/protobuf",
		"application/x-protobuf",
		"application/pb",
		"application/proto",
	}

	// Iterate over the test cases table.
	for _, testContentType := range testContentTypes {
		t.Run(testContentType, func(t *testing.T) {
			// Get the serializer by input.
			serializer := restful.GetSerializer(testContentType)

			// Call the Name function.
			name := serializer.Name()

			// Check if the name matches the expected content type.
			require.Equal(t, testContentType, name, "Serializer name should be"+testContentType)
		})
	}
}

func TestProtoSerializerContentType(t *testing.T) {
	// Define the test cases table.
	testContentTypes := []string{
		"application/octet-stream",
		"application/protobuf",
		"application/x-protobuf",
		"application/pb",
		"application/proto",
	}

	// Iterate over the test cases table.
	for _, testContentType := range testContentTypes {
		t.Run(testContentType, func(t *testing.T) {
			// Get the serializer by input.
			serializer := restful.GetSerializer(testContentType)

			// Call the ContentType function.
			name := serializer.ContentType()

			// Check if the content type matches the expected content type.
			require.Equal(t, testContentType, name, "Serializer name should be"+testContentType)
		})
	}
}
