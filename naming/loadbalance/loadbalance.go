// Package loadbalance is a pluggable loadbalance module.
package loadbalance

import (
	"errors"
	"sync"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

var (
	// ErrNoServerAvailable is an error that indicate there is no available server.
	ErrNoServerAvailable = errors.New("no server is available")
)

// load balance strategies.
const (
	LoadBalanceRandom             = "random"
	LoadBalanceRoundRobin         = "round_robin"
	LoadBalanceWeightedRoundRobin = "weight_round_robin"
	LoadBalanceConsistentHash     = "consistent_hash"
)

// DefaultLoadBalancer is the default LoadBalancer.
var DefaultLoadBalancer LoadBalancer = NewRandom()

// SetDefaultLoadBalancer sets the default LoadBalancer.
func SetDefaultLoadBalancer(b LoadBalancer) {
	DefaultLoadBalancer = b
}

// LoadBalancer is the interface that defines the load balance which returns a node from a node list.
type LoadBalancer interface {
	Select(serviceName string, list []*registry.Node, opt ...Option) (node *registry.Node, err error)
}

var (
	loadbalancers = make(map[string]LoadBalancer)
	lock          = sync.RWMutex{}
)

// Register registers a named LoadBalancer.
func Register(name string, s LoadBalancer) {
	lock.Lock()
	loadbalancers[name] = s
	lock.Unlock()
}

// Get gets a named LoadBalancer.
func Get(name string) LoadBalancer {
	lock.RLock()
	lb := loadbalancers[name]
	lock.RUnlock()
	return lb
}
