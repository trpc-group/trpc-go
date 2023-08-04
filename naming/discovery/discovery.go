// Package discovery is a pluggable service discovery module.
package discovery

import (
	"sync"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// DefaultDiscovery is the default discovery determined by configuration file.
var DefaultDiscovery Discovery = &IPDiscovery{}

// SetDefaultDiscovery sets the default discovery.
func SetDefaultDiscovery(d Discovery) {
	DefaultDiscovery = d
}

// Discovery is the interface that returns nodes by service name.
type Discovery interface {
	List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}

var (
	discoveries = make(map[string]Discovery)
	lock        = sync.RWMutex{}
)

// Register registers a named discovery.
func Register(name string, s Discovery) {
	lock.Lock()
	discoveries[name] = s
	lock.Unlock()
}

// Get gets a named discovery.
func Get(name string) Discovery {
	lock.RLock()
	d := discoveries[name]
	lock.RUnlock()
	return d
}

func unregisterForTesting(name string) {
	lock.Lock()
	delete(discoveries, name)
	lock.Unlock()
}
