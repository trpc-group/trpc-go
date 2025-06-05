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

package graceful

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Graceful restart is hard to do in unit test.
// We simulate it by starting a new goroutine.
// The real test should be done in e2e test.
func TestGracefulRestart_TCP(t *testing.T) {
	done := make(chan struct{})
	addr, err := serve(done, "parent")
	require.Nil(t, err)

	conn, err := net.Dial(addr.Network(), addr.String())
	require.Nil(t, err)
	testConn(t, conn, "parent")

	forkExec = func(argv0 string, argv []string, attr *syscall.ProcAttr) (pid int, err error) {
		require.NotEmpty(t, attr.Files)
		require.Nil(t, os.Setenv(gracefulRestartFdEnvKey, strconv.Itoa(int(attr.Files[len(attr.Files)-1]))))
		defer os.Setenv(gracefulRestartFdEnvKey, "")
		go func() {
			_, err = serve(nil, "child")
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(time.Millisecond * 100)
		return 0, nil
	}
	defer func() { forkExec = syscall.ForkExec }()
	require.Nil(t, Restart(nil))
	close(done)
	// this test result in the closing of server connection.
	testConn(t, conn, "parent")

	time.Sleep(time.Millisecond * 100)
	// In a real application, fd is closed automatically when parent process exit.
	// In unit test, we must close it manually.
	require.Nil(t, syscall.Close(writerToChildProcess.Load().T.fd))

	// this test is served by new server conn.
	testConn(t, conn, "child")

	_, err = net.Dial(addr.Network(), addr.String())
	require.Nil(t, err)
}

// Graceful restart is hard to do in unit test.
// We simulate it by starting a new goroutine.
// The real test should be done in e2e test.
func TestGracefulRestart_UDP(t *testing.T) {
	done := make(chan struct{})
	addr, err := serveUDP(done, "parent")
	require.Nil(t, err)

	conn, err := net.Dial(addr.Network(), addr.String())
	require.Nil(t, err)
	testConn(t, conn, "parent")

	forkExec = func(argv0 string, argv []string, attr *syscall.ProcAttr) (pid int, err error) {
		require.NotEmpty(t, attr.Files)
		require.Nil(t, os.Setenv(gracefulRestartFdEnvKey, strconv.Itoa(int(attr.Files[len(attr.Files)-1]))))
		defer os.Setenv(gracefulRestartFdEnvKey, "")
		go func() {
			_, err = serveUDP(nil, "child")
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(time.Millisecond * 100)
		return 0, nil
	}
	defer func() { forkExec = syscall.ForkExec }()
	require.Nil(t, Restart(nil))
	close(done)
	// this test result in the closing of server connection.
	testConn(t, conn, "parent")

	time.Sleep(time.Millisecond * 100)
	// In a real application, fd is closed automatically when parent process exit.
	// In unit test, we must close it manually.
	require.Nil(t, syscall.Close(writerToChildProcess.Load().T.fd))

	// this test is served by new server conn.
	testConn(t, conn, "child")

	_, err = net.Dial(addr.Network(), addr.String())
	require.Nil(t, err)
}

func serve(done chan struct{}, prefix string) (net.Addr, error) {
	init1()
	l, err := Listen("tcp", ":0", true)
	if err != nil {
		return nil, err
	}
	go func() {
		defer l.Close()
		for {
			select {
			case <-done:
				return
			default:
			}
			conn, err := l.Accept()
			if err != nil {
				fmt.Println("Accept failed:", err)
				return
			}
			go func() {
				defer conn.Close()
				buf := make([]byte, 1)
				for {
					select {
					case <-done:
						return
					default:
					}
					_, err := conn.Read(buf)
					if err != nil {
						fmt.Println("conn.Read failed:", err)
						return
					}
					if _, err := conn.Write(append([]byte(prefix), buf...)); err != nil {
						fmt.Println("conn.Write failed:", err)
						return
					}
				}
			}()
		}
	}()
	return l.Addr(), nil
}

func serveUDP(done chan struct{}, prefix string) (net.Addr, error) {
	init1()
	conn, err := ListenPacket("udp", ":0", false)
	if err != nil {
		return nil, err
	}
	go func() {
		time.Sleep(time.Millisecond * 100)
		defer conn.Close()
		for {
			select {
			case <-done:
				return
			default:
			}
			buf := make([]byte, 1)
			_, addr, err := conn.ReadFrom(buf)
			if err != nil {
				fmt.Println("conn.ReadFrom failed:", err)
				return
			}
			if _, err := conn.WriteTo(append([]byte(prefix), buf...), addr); err != nil {
				fmt.Println("conn.WriteTo failed:", err)
				return
			}
		}
	}()
	return conn.LocalAddr(), nil
}

func testConn(t *testing.T, conn net.Conn, prefix string) {
	_, err := conn.Write([]byte("a"))
	require.Nil(t, err)
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	require.Nil(t, err)
	require.Equal(t, prefix+"a", string(buf[:n]))
}
