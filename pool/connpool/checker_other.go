//go:build !aix && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !solaris && !linux
// +build !aix,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd,!solaris,!linux

package connpool

import (
	"errors"
	"net"
	"time"
)

func checkConnErr(conn net.Conn, buf []byte) error {
	conn.SetReadDeadline(time.Now().Add(time.Millisecond))
	n, err := conn.Read(buf)
	// Idle connections should not read data, it is an unexpected read error.
	if err == nil || n > 0 {
		return errors.New("unexpected read from socket")
	}
	// The idle connection is normal and returns timeout.
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		conn.SetReadDeadline(time.Time{})
		return nil
	}
	// other connection errors, including connection closed.
	return err
}

func checkConnErrUnblock(conn net.Conn, buf []byte) error {
	// Currently non-blocking mode is not supported.
	return nil
}
