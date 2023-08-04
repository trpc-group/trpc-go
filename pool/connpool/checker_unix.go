//go:build aix || darwin || dragonfly || freebsd || netbsd || openbsd || solaris || linux
// +build aix darwin dragonfly freebsd netbsd openbsd solaris linux

package connpool

import (
	"errors"
	"io"
	"net"
	"syscall"

	"trpc.group/trpc-go/trpc-go/internal/report"
)

func checkConnErr(conn net.Conn, buf []byte) error {
	return checkConnErrUnblock(conn, buf)
}

func checkConnErrUnblock(conn net.Conn, buf []byte) error {
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return nil
	}
	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return err
	}

	var sysErr error
	var n int
	err = rawConn.Read(func(fd uintptr) bool {
		// Go sets the socket to non-blocking mode by default, and calling syscall can return directly.
		// Refer to the Go source code: sysSocket() function under src/net/sock_cloexec.go
		n, sysErr = syscall.Read(int(fd), buf)
		// Return true, the blocking and waiting encapsulated by
		// the net library will not be executed, and return directly.
		return true
	})
	if err != nil {
		return err
	}

	// connection is closed, return io.EOF.
	if n == 0 && sysErr == nil {
		report.ConnectionPoolRemoteEOF.Incr()
		return io.EOF
	}
	// Idle connections should not read data.
	if n > 0 {
		return errors.New("unexpected read from socket")
	}
	// Return to EAGAIN or EWOULDBLOCK if the idle connection is in normal state.
	if sysErr == syscall.EAGAIN || sysErr == syscall.EWOULDBLOCK {
		return nil
	}
	return sysErr
}
