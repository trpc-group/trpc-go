package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	require.Nil(t, watchers["testService"], "testService watcher")
	Watch("testService", func(status Status) {})
	require.NotNil(t, watchers["testService"], "testService watcher")
}

func TestGetWatchers(t *testing.T) {
	Watch("testService", func(status Status) {})
	ws := GetWatchers()
	require.NotNil(t, ws["testService"])
}
