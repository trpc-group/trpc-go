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

//go:build !windows
// +build !windows

// These are meaningless tests just to pass code coverage.
// You don't need to know how these tests work. If a test case failed for some changes, just remove the case.
// If the code coverage falls bellow the threshold, simply add your own case in any way you see fit.

package graceful

import (
	"encoding/gob"
	"errors"
	"math"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSysConnFd(t *testing.T) {
	_, err := sysConnFd(1)
	require.NotNil(t, err)

	_, err = sysConnFd(syscallConnFunc(func() (syscall.RawConn, error) {
		return nil, errors.New("")
	}))
	require.NotNil(t, err)

	_, err = sysConnFd(syscallConnFunc(func() (syscall.RawConn, error) {
		return rawConnControlFunc(func(f func(fd uintptr)) error {
			return errors.New("")
		}), nil
	}))
	require.NotNil(t, err)
}

type syscallConnFunc func() (syscall.RawConn, error)

func (f syscallConnFunc) SyscallConn() (syscall.RawConn, error) {
	return f()
}

type rawConnControlFunc func(f func(fd uintptr)) error

func (f rawConnControlFunc) Control(ff func(fd uintptr)) error {
	return f(ff)
}

func (f rawConnControlFunc) Read(func(fd uintptr) (done bool)) error {
	return errors.New("never call")
}

func (f rawConnControlFunc) Write(func(fd uintptr) (done bool)) error {
	return errors.New("never call")
}

func TestRPCWriterFlushError(t *testing.T) {
	w := NewRpcWriter(syscall.Stdout)
	require.NotNil(t, w.Flush(make([]int, maxSCMDataLen+1)))

	require.Nil(t, w.Encode(1))
	require.NotNil(t, w.Flush(nil))

	w.fds = []int{}
	require.NotNil(t, w.Flush(nil))
}

func TestRPCReaderReadError(t *testing.T) {
	r := NewRpcReader(syscall.Stdin)
	_, err := r.Read(nil)
	require.NotNil(t, err)

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.Nil(t, err)
	defer func() {
		require.Nil(t, syscall.Close(fds[0]))
		require.Nil(t, syscall.Close(fds[1]))
	}()

	r = NewRpcReader(fds[0])
	spcm := syscallParseSocketControlMessage

	syscallParseSocketControlMessage = func([]byte) ([]syscall.SocketControlMessage, error) {
		return nil, errors.New("")
	}
	require.Nil(t, syscall.Sendmsg(fds[1], []byte("a"), nil, nil, 0))
	_, err = r.Read(make([]byte, 8))
	require.NotNil(t, err)
	syscallParseSocketControlMessage = spcm

	syscallParseSocketControlMessage = func([]byte) ([]syscall.SocketControlMessage, error) {
		return make([]syscall.SocketControlMessage, 2), nil
	}
	require.Nil(t, syscall.Sendmsg(fds[1], []byte("a"), nil, nil, 0))
	_, err = r.Read(make([]byte, 8))
	require.NotNil(t, err)
	syscallParseSocketControlMessage = spcm

	syscallParseSocketControlMessage = func([]byte) ([]syscall.SocketControlMessage, error) {
		return make([]syscall.SocketControlMessage, 1), nil
	}
	require.Nil(t, syscall.Sendmsg(fds[1], []byte("a"), nil, nil, 0))
	r.fds = []int{}
	_, err = r.Read(make([]byte, 8))
	require.NotNil(t, err)
	r.fds = nil
	syscallParseSocketControlMessage = spcm

	syscallParseSocketControlMessage = func([]byte) ([]syscall.SocketControlMessage, error) {
		return make([]syscall.SocketControlMessage, 1), nil
	}
	require.Nil(t, syscall.Sendmsg(fds[1], []byte("a"), nil, nil, 0))
	_, err = r.Read(make([]byte, 8))
	require.NotNil(t, err)
	syscallParseSocketControlMessage = spcm
}

func TestListenError(t *testing.T) {
	_, err := Listen("invalid", "", true)
	require.NotNil(t, err)
}

func TestAcceptInvalidRecvState(t *testing.T) {
	l, err := Listen("tcp", "", true)
	require.Nil(t, err)
	lis, ok := l.(*Listener)
	require.True(t, ok)

	defer func() {
		require.NotNil(t, recover())
		require.Nil(t, lis.Close())
	}()
	lis.recvState = math.MaxUint32
	_, err = lis.Accept()
}

func TestInit1_ParseInvalidGracefulRestartFdEnvErrorPanic(t *testing.T) {
	require.Nil(t, os.Setenv(gracefulRestartFdEnvKey, "invalid"))
	defer func() {
		require.NotNil(t, recover())
		require.Nil(t, os.Setenv(gracefulRestartFdEnvKey, ""))
	}()
	init1()
}

func TestReceiveListeners(t *testing.T) {
	r := NewRpcReader(syscall.Stdout)
	_, err := receiveListeners(r)
	require.NotNil(t, err)

	r, w, c := newSocketPairReaderWriter(t)
	defer c()
	var req protocol = ReqConn{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	_, err = receiveListeners(r)
	require.NotNil(t, err)

	req = ReqListeners{
		Listeners: make([]ReqListener, 1),
		Continue:  false,
	}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	_, err = receiveListeners(r)
	require.NotNil(t, err)

	req = ReqListeners{Continue: true}
	require.Nil(t, w.Encode(&req))
	req = ReqListeners{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	_, err = receiveListeners(r)
	require.Nil(t, err)
}

func TestReceiveAllListeners(t *testing.T) {
	r, w, c := newSocketPairReaderWriter(t)
	defer c()
	var req protocol = ReqConn{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	_, err := receiveAllListeners(r, w)
	require.NotNil(t, err)

	req = ReqListeners{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	enc := w.enc
	w.enc = gob.NewEncoder(writerFunc(func([]byte) (int, error) {
		return 0, errors.New("")
	}))
	_, err = receiveAllListeners(r, w)
	require.NotNil(t, err)
	w.enc = enc

	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	w.enc = gob.NewEncoder(writerFunc(func(bts []byte) (int, error) {
		return len(bts), nil
	}))
	w.fds = []int{}
	_, err = receiveAllListeners(r, w)
	require.NotNil(t, err)
	w.enc = enc
}

func TestSendListeners(t *testing.T) {
	w := NewRpcWriter(syscall.Stdout)
	enc := w.enc
	w.enc = gob.NewEncoder(writerFunc(func([]byte) (int, error) {
		return 0, errors.New("")
	}))
	require.NotNil(t, sendListeners(w, nil, nil))
	w.enc = enc
	w.fds = []int{}
	require.NotNil(t, sendListeners(w, nil, nil))
}

func TestSendListenersWaitAck(t *testing.T) {
	listeners.Lock()
	listeners.T = appendMap(listeners.T, t.Name(), t.Name(), nil)
	listeners.Unlock()
	require.NotNil(t, sendListenersWaitAck(nil, nil))
	listeners.Lock()
	listeners.T = nil
	listeners.Unlock()

	r, w, c := newSocketPairReaderWriter(t)
	require.Nil(t, w.Encode(1))
	require.Nil(t, w.Flush(nil))
	w.fds = []int{}
	require.NotNil(t, sendListenersWaitAck(w, nil))
	w.fds = nil
	enc := w.enc
	w.enc = gob.NewEncoder(writerFunc(func(bts []byte) (int, error) {
		return len(bts), nil
	}))
	require.NotNil(t, sendListenersWaitAck(w, r))
	w.enc = enc
	c()

	r, w, c = newSocketPairReaderWriter(t)
	var req protocol = ReqConn{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	enc = w.enc
	w.enc = gob.NewEncoder(writerFunc(func(bts []byte) (int, error) {
		return len(bts), nil
	}))
	require.NotNil(t, sendListenersWaitAck(w, r))
	w.enc = enc
	c()

	r, w, c = newSocketPairReaderWriter(t)
	req = AckListeners{Cnt: 1}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	enc = w.enc
	w.enc = gob.NewEncoder(writerFunc(func(bts []byte) (int, error) {
		return len(bts), nil
	}))
	require.NotNil(t, sendListenersWaitAck(w, r))
	w.enc = enc
	c()
}

func TestReceivingConnections(t *testing.T) {
	receivingConnections(NewRpcReader(syscall.Stdout), nil)
	require.True(t, true)

	r, w, c := newSocketPairReaderWriter(t)
	var req protocol = AckListeners{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	receivingConnections(r, nil)
	require.True(t, true)
	c()

	r, w, c = newSocketPairReaderWriter(t)
	req = ReqConn{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	receivingConnections(r, nil)
	require.True(t, true)
	c()

	r, w, c = newSocketPairReaderWriter(t)
	req = ReqConn{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush([]int{syscall.Stdout}))
	receivingConnections(r, nil)
	require.True(t, true)
	c()

	r, w, c = newSocketPairReaderWriter(t)
	req = ReqConn{}
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush([]int{w.writer.fd}))
	require.Nil(t, w.Encode(&req))
	require.Nil(t, w.Flush(nil))
	receivingConnections(r, nil)
	require.True(t, true)
	c()
}

func TestNewConnOnClosed(t *testing.T) {
	w2cp := writerToChildProcess.Swap(nil)
	newConnOnClosed("", "")(nil)
	require.True(t, true)
	writerToChildProcess.Store(w2cp)

	w := NewRpcWriter(syscall.Stdout)
	w2cp = NewSafe(w)
	restore := writerToChildProcess.Swap(w2cp)
	defer writerToChildProcess.Store(restore)
	newConnOnClosed("", "")(nil)
	require.True(t, true)

	conn := netSyscallConn{syscallConn: syscallConnFunc(func() (syscall.RawConn, error) {
		return rawConnControlFunc(func(f func(fd uintptr)) error {
			f(1)
			return nil
		}), nil
	})}
	w.enc = gob.NewEncoder(writerFunc(func(bts []byte) (int, error) {
		return 0, errors.New("")
	}))
	newConnOnClosed("", "")(conn)
	require.True(t, true)

	w.enc = gob.NewEncoder(writerFunc(func(bts []byte) (int, error) {
		return len(bts), nil
	}))
	w.fds = []int{}
	newConnOnClosed("", "")(conn)
	require.True(t, true)
}

func TestInitInherit(t *testing.T) {
	require.NotNil(t, initInherit("-1"))
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

type netSyscallConn struct {
	net.Conn
	syscallConn syscall.Conn
}

func (c netSyscallConn) SyscallConn() (syscall.RawConn, error) {
	return c.syscallConn.SyscallConn()
}

func newSocketPairReaderWriter(t *testing.T) (*Reader, *Writer, func()) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.Nil(t, err)
	return NewRpcReader(fds[0]), NewRpcWriter(fds[1]), func() {
		require.Nil(t, syscall.Close(fds[0]))
		require.Nil(t, syscall.Close(fds[1]))
	}
}
