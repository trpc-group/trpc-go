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

package transport_test

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/transport"
)

func Test_ServerTransport_UDP(t *testing.T) {
	var addr = getFreeAddr("udp")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		)
		assert.Nil(t, err)
		time.Sleep(20 * time.Millisecond)
	}()
	wg.Wait()

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	reqDataBad := append(reqData, []byte("remain")...)
	pc, err := net.Dial("udp", addr)
	require.Nil(t, err)

	// Bad request, server will not response.
	pc.Write(reqDataBad)
	result := make([]byte, 20)
	pc.SetDeadline(time.Now().Add(100 * time.Millisecond))
	_, err = pc.Read(result)
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	pc.SetDeadline(time.Time{})

	// Good request, server will response.
	pc.Write(reqData)
	_, err = pc.Read(result)
	require.Nil(t, err)
}
