package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpDiscovery(t *testing.T) {
	d := &IPDiscovery{}
	list, err := d.List("ipdiscovery.ip.62:8989", nil)
	assert.Nil(t, err)
	assert.Equal(t, len(list), 1)
	assert.Equal(t, list[0].Address, "ipdiscovery.ip.62:8989")
}
