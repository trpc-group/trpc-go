//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd
// +build linux darwin dragonfly freebsd netbsd openbsd

package reuseport

import (
	"errors"
	"net"
	"os"
	"syscall"
)

var (
	// ListenerBacklogMaxSize setting backlog size
	ListenerBacklogMaxSize    = maxListenerBacklog()
	errUnsupportedTCPProtocol = errors.New("only tcp, tcp4, tcp6 are supported")
)

func getTCP4Sockaddr(tcp *net.TCPAddr) (syscall.Sockaddr, int, error) {
	sa := &syscall.SockaddrInet4{Port: tcp.Port}

	if tcp.IP != nil {
		if len(tcp.IP) == 16 {
			copy(sa.Addr[:], tcp.IP[12:16]) // copy last 4 bytes of slice to array
		} else {
			copy(sa.Addr[:], tcp.IP) // copy all bytes of slice to array
		}
	}

	return sa, syscall.AF_INET, nil
}

func getTCP6Sockaddr(tcp *net.TCPAddr) (syscall.Sockaddr, int, error) {
	sa := &syscall.SockaddrInet6{Port: tcp.Port}

	if tcp.IP != nil {
		copy(sa.Addr[:], tcp.IP) // copy all bytes of slice to array
	}

	if tcp.Zone != "" {
		iface, err := net.InterfaceByName(tcp.Zone)
		if err != nil {
			return nil, -1, err
		}

		sa.ZoneId = uint32(iface.Index)
	}

	return sa, syscall.AF_INET6, nil
}

func getTCPAddr(proto, addr string) (*net.TCPAddr, string, error) {
	var tcp *net.TCPAddr

	// fix bugs https://github.com/kavu/go_reuseport/pull/33
	tcp, err := net.ResolveTCPAddr(proto, addr)
	if err != nil {
		return nil, "", err
	}

	tcpVersion, err := determineTCPProto(proto, tcp)
	if err != nil {
		return nil, "", err
	}
	return tcp, tcpVersion, nil
}

func getTCPSockaddr(proto, addr string) (sa syscall.Sockaddr, soType int, err error) {
	tcp, tcpVersion, err := getTCPAddr(proto, addr)
	if err != nil {
		return nil, -1, err
	}
	switch tcpVersion {
	case "tcp":
		return &syscall.SockaddrInet4{Port: tcp.Port}, syscall.AF_INET, nil
	case "tcp4":
		return getTCP4Sockaddr(tcp)
	default:
		// must be "tcp6"
		return getTCP6Sockaddr(tcp)
	}
}

func determineTCPProto(proto string, ip *net.TCPAddr) (string, error) {
	// If the protocol is set to "tcp", we try to determine the actual protocol
	// version from the size of the resolved IP address. Otherwise, we simple use
	// the protocol given to us by the caller.

	if ip.IP.To4() != nil {
		return "tcp4", nil
	}

	if ip.IP.To16() != nil {
		return "tcp6", nil
	}

	switch proto {
	case "tcp", "tcp4", "tcp6":
		return proto, nil
	default:
		return "", errUnsupportedTCPProtocol
	}
}

// NewReusablePortListener returns net.FileListener that created from
// a file discriptor for a socket with SO_REUSEPORT option.
func NewReusablePortListener(proto, addr string) (l net.Listener, err error) {
	var (
		soType, fd int
		sockaddr   syscall.Sockaddr
	)
	if sockaddr, soType, err = getSockaddr(proto, addr); err != nil {
		return nil, err
	}

	syscall.ForkLock.RLock()
	if fd, err = syscall.Socket(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
		syscall.ForkLock.RUnlock()
		return nil, err
	}
	syscall.ForkLock.RUnlock()

	if err = createReusableFd(fd, sockaddr); err != nil {
		return nil, err
	}
	return createReusableListener(fd, proto, addr)
}

func createReusableListener(fd int, proto, addr string) (l net.Listener, err error) {
	file := os.NewFile(uintptr(fd), getSocketFileName(proto, addr))
	if l, err = net.FileListener(file); err != nil {
		file.Close()
		return nil, err
	}

	if err = file.Close(); err != nil {
		return nil, err
	}
	return l, err
}

func createReusableFd(fd int, sockaddr syscall.Sockaddr) (err error) {
	defer func() {
		if err != nil {
			syscall.Close(fd)
		}
	}()

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, reusePort, 1); err != nil {
		return err
	}

	if err = syscall.Bind(fd, sockaddr); err != nil {
		return err
	}

	// Set backlog size to the maximum
	if err = syscall.Listen(fd, ListenerBacklogMaxSize); err != nil {
		return err
	}

	return nil
}
