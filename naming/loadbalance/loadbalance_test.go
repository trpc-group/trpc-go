package loadbalance

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
)

var testNode *registry.Node = &registry.Node{
	ServiceName: "testservice",
	Address:     "loadbalance.ip.1:16721",
	Network:     "tcp",
}

type testLoadbalance struct{}

// Select acquires a node.
func (tlb *testLoadbalance) Select(serviceName string, list []*registry.Node, opt ...Option) (*registry.Node, error) {
	return testNode, nil
}

func TestLoadbalanceRegister(t *testing.T) {
	Register("tlb", &testLoadbalance{})
	assert.NotNil(t, Get("tlb"))
}

func TestLoadbalanceGet(t *testing.T) {
	Register("tlb", &testLoadbalance{})
	assert.NotNil(t, Get("tlb"))
	assert.Nil(t, Get("not_exist"))
}

func TestLoadbalanceSelect(t *testing.T) {
	Register("tlb", &testLoadbalance{})
	lb := Get("tlb")
	n, err := lb.Select("test-service", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, n, testNode)
}

func TestSetDefaultLoadBalancer(t *testing.T) {
	noop := &testLoadbalance{}
	SetDefaultLoadBalancer(noop)
	assert.Equal(t, DefaultLoadBalancer, noop)
}
