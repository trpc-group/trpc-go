package selector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPassthroughSelectorSelect(t *testing.T) {
	selector := &passthroughSelector{}
	n, err := selector.Select("passthrough")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "passthrough")
	assert.Equal(t, n.ServiceName, "passthrough")
}

func TestPassthroughSelectorReport(t *testing.T) {
	selector := &passthroughSelector{}
	n, err := selector.Select("passthrough")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "passthrough")
	assert.Equal(t, n.ServiceName, "passthrough")
	assert.Nil(t, selector.Report(n, 0, nil))
}
