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

// Package main provides a client example for SSE based on https://github.com/r3labs/sse.
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/r3labs/sse/v2"
)

func main() {
	const (
		address = "127.0.0.1:8081"
		pattern = "/events"
	)
	c := sse.NewClient(fmt.Sprintf("http://%s%s", address, pattern))
	events := make(chan *sse.Event)
	var err error
	go func() {
		err = c.Subscribe("test", func(msg *sse.Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	// Wait for the subscription to succeed.
	time.Sleep(200 * time.Millisecond)
	if err != nil {
		fmt.Printf("Subscription failed: %v\n", err)
		return
	}

	// Subscribe and wait for 1 event.
	subscribeSingleEvent(events)
	// Subscribe and wait for 3 events.
	subscribeMultipleEvents(events)
}

// Subscribe and wait for 1 event, wait for a max of 500ms for the event.
func subscribeSingleEvent(events chan *sse.Event) {
	msg, err := wait(events, time.Millisecond*500)
	if err != nil {
		fmt.Printf("Received error: %v\n", err)
		return
	}
	fmt.Printf("Receive msg: %s\n", msg)
}

// Subscribe and wait for 3 events, wait for a max of 500ms for each event.
func subscribeMultipleEvents(events chan *sse.Event) {
	for i := 0; i < 3; i++ {
		msg, err := wait(events, time.Millisecond*500)
		if err != nil {
			fmt.Printf("%d received error: %v\n", i, err)
			continue
		}
		fmt.Printf("Receive msg: %s\n", msg)
	}
}

// wait waits for the sse event and read data into msg. If timeout, return error.
func wait(ch chan *sse.Event, duration time.Duration) ([]byte, error) {
	var err error
	var msg []byte

	select {
	case event := <-ch:
		msg = event.Data
	case <-time.After(duration):
		err = errors.New("timeout")
	}
	return msg, err
}
