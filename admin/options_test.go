package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithSkipServe(t *testing.T) {
	opts := []Option{
		WithVersion(testVersion),
		WithAddr(defaultListenAddr),
		WithTLS(false),
		WithReadTimeout(defaultReadTimeout),
		WithWriteTimeout(defaultWriteTimeout),
		WithConfigPath(testConfigPath),
	}
	t.Run("enable SkipServe option", func(t *testing.T) {
		require.True(t, NewServer(append(opts, WithSkipServe(true))...).config.skipServe)
	})
	t.Run("disable SkipServe option", func(t *testing.T) {
		require.False(t, NewServer(append(opts, WithSkipServe(false))...).config.skipServe)
	})
}
