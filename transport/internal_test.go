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

package transport

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
)

// TestUDPServerTransportJobQueueFullFail tests the UDP server transport when the job queue is full.
func TestUDPServerTransportJobQueueFullFail(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	opts := []ListenServeOption{
		WithListenNetwork("udp"),
		WithUDPListener(ln),
		WithHandler(&delayHandler{}),
		WithServerFramerBuilder(&framerBuilder{}),
		WithMaxRoutines(1), // Set the number of routines to 1.
	}
	lsopts := &ListenServeOptions{}
	for _, opt := range opts {
		opt(lsopts)
	}

	// Create a non-blocking UDP routine pool to return an error when the queue is full.
	pool := createUDPRoutinePoolNoBlocking(lsopts.Routines)

	sopts := defaultServerTransportOptions()
	addrToConn := make(map[string]*tcpconn)
	s := &serverTransport{addrToConn: addrToConn, m: &sync.RWMutex{}, opts: sopts}
	udpconn, err := s.getUDPListener(lsopts)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		if err := s.serveUDP(ctx, udpconn, pool, lsopts); err != nil {
			return
		}
	}()

	// Perform round trips.
	rNum := 2 // Number of round trips.
	roundTripWG := &sync.WaitGroup{}
	roundTripWG.Add(rNum)
	for i := 0; i < rNum; i++ {
		go func() {
			defer roundTripWG.Done()
			req := &helloRequest{
				Name: "trpc",
				Msg:  "HelloWorld",
			}

			data, err := json.Marshal(req)
			require.Nil(t, err)

			lenData := make([]byte, 4)
			binary.BigEndian.PutUint32(lenData, uint32(len(data)))
			reqData := append(lenData, data...)

			ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer f()

			_, err = RoundTrip(ctx, reqData,
				WithDialNetwork(ln.LocalAddr().Network()),
				WithDialAddress(ln.LocalAddr().String()),
				WithClientFramerBuilder(&framerBuilder{}))
			assert.NotNil(t, err)
		}()
	}
	roundTripWG.Wait()
}

func createUDPRoutinePoolNoBlocking(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		param, ok := args.(*handleUDPParam)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		if param.uc == nil {
			log.Tracef("routine pool udpconn is nil, shouldn't happen!")
			return
		}
		param.uc.handleSync(param.req, param.remoteAddr)
		param.reset()
		handleUDPParamPool.Put(param)
	}, ants.WithNonblocking(true)) // // Use non-blocking mode to return an error when the queue is full.
	if err != nil {
		log.Tracef("routine pool create error:%v", err)
		return nil
	}
	return pool
}

type delayHandler struct{}

func (h *delayHandler) Handle(ctx context.Context, req []byte) ([]byte, error) {
	time.Sleep(time.Second * 1)
	rsp := make([]byte, len(req))
	return rsp, nil
}

type framerBuilder struct {
	errSet bool
	err    error
	safe   bool
}

// SetError sets frameBuilder error.
func (fb *framerBuilder) SetError(err error) {
	fb.errSet = true
	fb.err = err
}

func (fb *framerBuilder) ClearError() {
	fb.errSet = false
	fb.err = nil
}

func (fb *framerBuilder) New(r io.Reader) codec.Framer {
	return &framer{r: r, fb: fb}
}

type framer struct {
	fb *framerBuilder
	r  io.Reader
}

func (f *framer) ReadFrame() ([]byte, error) {
	if f.fb.errSet {
		return nil, f.fb.err
	}
	var lenData [4]byte

	_, err := io.ReadFull(f.r, lenData[:])
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lenData[:])

	msg := make([]byte, len(lenData)+int(length))
	copy(msg, lenData[:])

	_, err = io.ReadFull(f.r, msg[len(lenData):])
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (f *framer) IsSafe() bool {
	return f.fb.safe
}

type helloRequest struct {
	Name string
	Msg  string
}

type helloResponse struct {
	Name string
	Msg  string
	Code int
}
