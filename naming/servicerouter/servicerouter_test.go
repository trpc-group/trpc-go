package servicerouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceRouterRegister(t *testing.T) {
	Register("noop", &NoopServiceRouter{})
	assert.NotNil(t, Get("noop"))
	unregisterForTesting("noop")
}

func TestSetDefaultServiceRouter(t *testing.T) {
	noop := &NoopServiceRouter{}
	SetDefaultServiceRouter(noop)
	assert.Equal(t, noop, DefaultServiceRouter)
	nodes, err := noop.Filter("noop_service", nil)
	assert.Nil(t, err)
	assert.Len(t, nodes, 0)
}
