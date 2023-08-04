// Package circuitbreaker is a pluggable circuit breaker module.
package circuitbreaker

import (
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// DefaultCircuitBreaker is the default circuit breaker.
var DefaultCircuitBreaker CircuitBreaker = &NoopCircuitBreaker{}

// SetDefaultCircuitBreaker sets the default circuit breaker.
func SetDefaultCircuitBreaker(cb CircuitBreaker) {
	DefaultCircuitBreaker = cb
}

// CircuitBreaker is the interface that defines the circuit breaker which determines whether a node
// is available and report the result of an RPC on the node.
type CircuitBreaker interface {
	Available(node *registry.Node) bool
	Report(node *registry.Node, cost time.Duration, err error) error
}

var (
	circuitbreakers = make(map[string]CircuitBreaker)
	lock            = sync.RWMutex{}
)

// Register registers a named circuit breaker.
func Register(name string, s CircuitBreaker) {
	lock.Lock()
	circuitbreakers[name] = s
	lock.Unlock()
}

// Get gets a named circuit breaker.
func Get(name string) CircuitBreaker {
	lock.RLock()
	c := circuitbreakers[name]
	lock.RUnlock()
	return c
}

func unregisterForTesting(name string) {
	lock.Lock()
	delete(circuitbreakers, name)
	lock.Unlock()
}

// NoopCircuitBreaker is a noop circuit breaker.
type NoopCircuitBreaker struct{}

// Available always returns true.
func (*NoopCircuitBreaker) Available(*registry.Node) bool {
	return true
}

// Report does nothing.
func (*NoopCircuitBreaker) Report(*registry.Node, time.Duration, error) error {
	return nil
}
