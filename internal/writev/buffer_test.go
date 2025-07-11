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

package writev

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	errForceError = errors.New("force error")
	defaultSize   = 128
	defaultMsg    = []byte("Hello World!!")
)

type fakeReadWriter struct {
	read  chan []byte
	sleep bool
	err   bool
}

// Write simulates a write operation.
func (f *fakeReadWriter) Write(p []byte) (n int, err error) {
	if f.err {
		return 0, errForceError
	}
	if f.sleep {
		time.Sleep(time.Millisecond)
	}
	f.read <- p
	return len(p), nil
}

// Read simulates read operation.
func (f *fakeReadWriter) Read() []byte {
	return <-f.read
}

func TestBufferWrite(t *testing.T) {
	done := make(chan struct{}, 1)
	rw := &fakeReadWriter{read: make(chan []byte, defaultSize)}
	buffer := NewBuffer()
	buffer.Start(rw, done)

	_, err := buffer.Write(defaultMsg)
	assert.Nil(t, err)
	rsp := rw.Read()
	assert.Equal(t, defaultMsg, rsp)
	assert.Equal(t, done, buffer.Done())

	buffer.SetQueueStopped(true)
	_, err = buffer.Write(defaultMsg)
	assert.Equal(t, ErrStopped, err)

	close(done)
	time.Sleep(time.Millisecond)
	assert.Equal(t, ErrAskQuit, buffer.Error())
}

func TestBufferWritev(t *testing.T) {
	done := make(chan struct{}, 1)
	rw := &fakeReadWriter{read: make(chan []byte, defaultSize)}
	buffer := NewBuffer()
	buffer.Start(rw, done)
	defer close(done)

	for i := 0; i < 10; i++ {
		_, err := buffer.Write([]byte(fmt.Sprintf("hello world %d\n", i)))
		assert.Nil(t, err)
	}
	for i := 0; i < 10; i++ {
		rsp := rw.Read()
		assert.Equal(t, fmt.Sprintf("hello world %d\n", i), string(rsp))
	}
}

func TestBufferWriteFull(t *testing.T) {
	done := make(chan struct{}, 1)
	rw := &fakeReadWriter{read: make(chan []byte, defaultSize), sleep: true}
	buffer := NewBuffer(WithDropFull(true), WithBufferSize(4))
	buffer.Start(rw, done)
	defer close(done)
	for {
		_, err := buffer.Write(defaultMsg)
		if err != nil {
			assert.NotNil(t, err)
			return
		}
	}
}

func TestBufferQuitHandle(t *testing.T) {
	done := make(chan struct{}, 1)
	rw := &fakeReadWriter{read: make(chan []byte, defaultSize), err: true}
	value := "abc"
	handler := func(buffer *Buffer) {
		value = "efg"
		buffer.SetQueueStopped(true)
	}
	buffer := NewBuffer(WithDropFull(true), WithBufferSize(8), WithQuitHandler(handler))
	buffer.Start(rw, done)
	defer close(done)

	_, err := buffer.Write(defaultMsg)
	assert.Nil(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "efg", value)
	assert.Equal(t, errForceError, buffer.Error())

	_, err = buffer.Write(defaultMsg)
	assert.Equal(t, errForceError, err)
}

func TestBufferReStart(t *testing.T) {
	done := make(chan struct{}, 1)
	rw := &fakeReadWriter{read: make(chan []byte, defaultSize), err: true}

	buffer := NewBuffer()
	buffer.Start(rw, done)
	close(done)
	time.Sleep(time.Millisecond)
	newdone := make(chan struct{}, 1)
	buffer = buffer.Restart(rw, newdone)
	defer close(newdone)

	_, err := buffer.Write(defaultMsg)
	assert.Nil(t, err)
}

type fakeWriter struct {
	rcv uint32
}

// Write simulates a write operation.
func (f *fakeWriter) Write(p []byte) (n int, err error) {
	atomic.AddUint32(&f.rcv, 1)
	return len(p), nil
}

func (f *fakeWriter) getReceives() uint32 {
	return f.rcv
}

func TestConCurrentWrite(t *testing.T) {
	w := &fakeWriter{}
	done := make(chan struct{}, 1)
	buffer := NewBuffer()
	buffer.Start(w, done)
	defer close(done)
	cpus := runtime.NumCPU()

	wg := &sync.WaitGroup{}
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go doWriteBuffer(buffer, wg, 10000)
	}
	wg.Wait()
	// wait for the package to complete.
	for {
		if buffer.queue.IsEmpty() {
			break
		}
		runtime.Gosched()
	}
	time.Sleep(time.Millisecond)
	assert.Equal(t, uint32(10000*cpus), w.getReceives())
}

// BenchmarkWriteBuffer tests the concurrent performance of WriteBuffer.
func BenchmarkWriteBuffer(b *testing.B) {
	w := &fakeWriter{}
	done := make(chan struct{}, 1)
	buffer := NewBuffer()
	buffer.Start(w, done)
	defer close(done)
	cpus := runtime.NumCPU() * 10
	wg := &sync.WaitGroup{}

	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()
	for j := 0; j < cpus; j++ {
		wg.Add(1)
		go doWriteBuffer(buffer, wg, b.N)
	}
	wg.Wait()
	b.StopTimer()
}

func doWriteBuffer(b *Buffer, wg *sync.WaitGroup, num int) {
	defer wg.Done()
	for {
		if _, err := b.Write(defaultMsg); err == nil {
			num--
		}
		if num <= 0 {
			return
		}
	}
}
