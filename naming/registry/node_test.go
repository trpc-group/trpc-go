package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNodeString(t *testing.T) {
	n := &Node{
		ServiceName: "name",
		Address:     "127.0.0.1:8080",
		CostTime:    time.Second,
	}
	assert.Equal(t, n.String(), fmt.Sprintf("service:%s, addr:%s, cost:%s",
		n.ServiceName, n.Address, n.CostTime))
}
