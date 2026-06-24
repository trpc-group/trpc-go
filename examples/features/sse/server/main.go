//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package main is the server main package for SSE demo.
package main

import (
	"fmt"
	stdhttp "net/http"
	"strconv"
	"time"

	"github.com/r3labs/sse/v2"

	thttp "trpc.group/trpc-go/trpc-go/http"
)

func main() {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("/events", events)

	fmt.Println("SSE server listening on http://127.0.0.1:8080/events")
	if err := stdhttp.ListenAndServe("127.0.0.1:8080", mux); err != nil {
		fmt.Println(err)
	}
}

func events(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	flusher, ok := w.(stdhttp.Flusher)
	if !ok {
		stdhttp.Error(w, "streaming unsupported", stdhttp.StatusInternalServerError)
		return
	}

	for i := 1; i <= 3; i++ {
		event := sse.Event{
			ID:    []byte(strconv.Itoa(i)),
			Event: []byte("message"),
			Data:  []byte(fmt.Sprintf("event-%d", i)),
		}
		if err := thttp.WriteSSE(w, event); err != nil {
			return
		}
		flusher.Flush()
		time.Sleep(200 * time.Millisecond)
	}
}
