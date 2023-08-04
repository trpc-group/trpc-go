// Package selector determines how client chooses a backend node by service name. It contains service
// discovery, load balance and circuit breaker.
package selector

import (
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// Selector is the interface that defines the selector.
type Selector interface {
	// Select gets a backend node by service name.
	Select(serviceName string, opt ...Option) (*registry.Node, error)
	// Report reports request status.
	Report(node *registry.Node, cost time.Duration, err error) error
}

var (
	selectors = make(map[string]Selector)
)

// Register registers a named Selector, such as l5, cmlb and tseer.
func Register(name string, s Selector) {
	selectors[name] = s
}

// Get gets a named Selector.
func Get(name string) Selector {
	s := selectors[name]
	return s
}

func unregisterForTesting(name string) {
	delete(selectors, name)
}
