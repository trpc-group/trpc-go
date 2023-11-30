// Package addrutil provides some utility functions for net address.
package addrutil

import (
	"net"
	"strings"
)

// AddrToKey combines local and remote address into a string.
func AddrToKey(local, remote net.Addr) string {
	return strings.Join([]string{local.Network(), local.String(), remote.String()}, "_")
}
