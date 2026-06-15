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

package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/r3labs/sse/v2"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

func handleSSE(body io.Reader, handle SSEHandler, msg codec.Msg) error {
	reader := sse.NewEventStreamReader(body, trpc.DefaultMaxFrameSize)
	for {
		bs, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("sse reader read event error: %w", err)
		}
		event, err := processEvent(bs)
		if err != nil {
			return fmt.Errorf("parsing sse event error: %w", err)
		}
		if err := handle.Handle(event); err != nil {
			var e *errs.Error
			if errors.As(err, &e) && e.Type == errs.ErrorTypeBusiness {
				msg.WithClientRspErr(err)
				return nil
			}
			return fmt.Errorf("sse handler handle event error: %w", err)
		}
	}
}

var (
	headerID      = []byte("id:")
	headerData    = []byte("data:")
	headerEvent   = []byte("event:")
	headerRetry   = []byte("retry:")
	headerComment = []byte(":")
)

func processEvent(msg []byte) (*sse.Event, error) {
	if len(msg) == 0 {
		return nil, errors.New("event message was empty")
	}

	var event sse.Event
	for _, line := range bytes.FieldsFunc(msg, func(r rune) bool { return r == '\n' || r == '\r' }) {
		switch {
		case bytes.HasPrefix(line, headerID):
			event.ID = trimHeader(len(headerID), line)
		case bytes.HasPrefix(line, headerData):
			event.Data = append(event.Data, append(trimHeader(len(headerData), line), byte('\n'))...)
		case bytes.Equal(line, bytes.TrimSuffix(headerData, []byte(":"))):
			event.Data = append(event.Data, byte('\n'))
		case bytes.HasPrefix(line, headerEvent):
			event.Event = trimHeader(len(headerEvent), line)
		case bytes.HasPrefix(line, headerRetry):
			event.Retry = trimHeader(len(headerRetry), line)
		case bytes.HasPrefix(line, headerComment):
			event.Comment = trimHeader(len(headerComment), line)
		default:
		}
	}
	event.Data = bytes.TrimSuffix(event.Data, []byte("\n"))
	return &sse.Event{
		ID:      safeCopy(event.ID),
		Data:    safeCopy(event.Data),
		Event:   safeCopy(event.Event),
		Retry:   safeCopy(event.Retry),
		Comment: safeCopy(event.Comment),
	}, nil
}

func trimHeader(size int, data []byte) []byte {
	if data == nil || len(data) < size {
		return data
	}
	data = data[size:]
	if len(data) > 0 && data[0] == ' ' {
		data = data[1:]
	}
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	return data
}

func safeCopy(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	dst := make([]byte, len(b))
	copy(dst, b)
	return dst
}
