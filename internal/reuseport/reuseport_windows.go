//go:build windows
// +build windows

package reuseport

import (
	"net"
	"syscall"
)

var ListenerBacklogMaxSize = maxListenerBacklog()

func maxListenerBacklog() int {
	return syscall.SOMAXCONN
}

func NewReusablePortListener(proto, addr string) (net.Listener, error) {
	return net.Listen(proto, addr)
}

func NewReusablePortPacketConn(proto, addr string) (net.PacketConn, error) {
	return net.ListenPacket(proto, addr)
}

// Listen function is an alias for NewReusablePortListener.
func Listen(proto, addr string) (l net.Listener, err error) {
	return NewReusablePortListener(proto, addr)
}

// ListenPacket is an alias for NewReusablePortPacketConn.
func ListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return NewReusablePortPacketConn(proto, addr)
}
