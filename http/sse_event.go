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

package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"trpc.group/trpc-go/trpc-go"

	"github.com/r3labs/sse/v2"
)

func handleSSE(body io.Reader, handle SSEHandler) error {
	// According to the implementation of SSE, the buffer in the event stream reader
	// stores the entire data of the SSE response, rather than a single event like `data: xxx`.
	// Therefore, we should use trpc.DefaultMaxFrameSize to limit the size of the buffer,
	// instead of codec.DefaultReaderSize, which stands for the size of a single frame.
	reader := sse.NewEventStreamReader(body, trpc.DefaultMaxFrameSize)
	for {
		bs, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				return nil // Normal ending, return directly.
			}
			return fmt.Errorf("sse reader read event error: %w", err)
		}
		event, err := processEvent(bs)
		if err != nil {
			return fmt.Errorf("parsing sse event error: %w", err)
		}
		if err := handle.Handle(event); err != nil {
			return fmt.Errorf("sse handler handle event error: %w", err)
		}
	}
}

// The following is a modification from client.go in
// "github.com/r3labs/sse/v2", since they are unexported.

var (
	headerID    = []byte("id:")
	headerData  = []byte("data:")
	headerEvent = []byte("event:")
	headerRetry = []byte("retry:")
)

func processEvent(msg []byte) (event *sse.Event, err error) {
	var e sse.Event
	if len(msg) == 0 {
		return nil, errors.New("event message was empty")
	}

	// Normalize the crlf to lf to make it easier to split the lines.
	// Split the line by "\n" or "\r", per the spec.
	for _, line := range bytes.FieldsFunc(msg, func(r rune) bool { return r == '\n' || r == '\r' }) {
		switch {
		case bytes.HasPrefix(line, headerID):
			e.ID = trimHeader(len(headerID), line)
		case bytes.HasPrefix(line, headerData):
			// The spec allows for multiple data fields per event, concatenated them with "\n".
			e.Data = append(e.Data, append(trimHeader(len(headerData), line), byte('\n'))...)
			// The spec says that a line that simply contains the string "data"
			// should be treated as a data field with an empty body.
		case bytes.Equal(line, bytes.TrimSuffix(headerData, []byte(":"))):
			e.Data = append(e.Data, byte('\n'))
		case bytes.HasPrefix(line, headerEvent):
			e.Event = trimHeader(len(headerEvent), line)
		case bytes.HasPrefix(line, headerRetry):
			e.Retry = trimHeader(len(headerRetry), line)
		default:
			// Ignore any garbage that doesn't match what we're looking for.
		}
	}

	// Trim the last "\n" per the spec.
	e.Data = bytes.TrimSuffix(e.Data, []byte("\n"))

	return &e, err
}

func trimHeader(size int, data []byte) []byte {
	if data == nil || len(data) < size {
		return data
	}

	data = data[size:]
	// Remove optional leading whitespace.
	if len(data) > 0 && data[0] == ' ' {
		data = data[1:]
	}
	// Remove trailing new line.
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	return data
}
