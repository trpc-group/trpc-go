//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateRWMutex(t *testing.T) {
	var mu stateRWMutex
	require.True(t, mu.rLock())
	mu.rUnlock()

	require.True(t, mu.lock())
	mu.closeLocked()
	mu.unlock()

	// Lock return false when mutex is already closed.
	require.False(t, mu.rLock())
	require.False(t, mu.lock())
}
