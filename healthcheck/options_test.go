package healthcheck_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/healthcheck"
)

func TestWithStatusWatcher(t *testing.T) {
	service1 := "testService1"
	service2 := "testService2"

	serviceStatus := map[string]healthcheck.Status{
		service1: healthcheck.Serving,
		service2: healthcheck.Serving,
	}
	watchFunc := func(status healthcheck.Status) {
		serviceStatus[service1] = status
	}

	hc := healthcheck.New(healthcheck.WithStatusWatchers(map[string][]func(status healthcheck.Status){
		service1: {watchFunc},
	}))
	update, _ := hc.Register(service1)
	update(healthcheck.Serving)
	require.Equal(t, healthcheck.Serving, serviceStatus[service1])
	require.Equal(t, healthcheck.Serving, serviceStatus[service2])

	update(healthcheck.NotServing)
	require.Equal(t, healthcheck.NotServing, serviceStatus[service1])
	require.Equal(t, healthcheck.Serving, serviceStatus[service2])
}
