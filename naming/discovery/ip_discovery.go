package discovery

import (
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// IPDiscovery discovers node by IPs.
type IPDiscovery struct{}

// List returns the original IP/Port.
func (*IPDiscovery) List(serviceName string, opt ...Option) ([]*registry.Node, error) {
	node := &registry.Node{ServiceName: serviceName, Address: serviceName}
	return []*registry.Node{node}, nil
}
