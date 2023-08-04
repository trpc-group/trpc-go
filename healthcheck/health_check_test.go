package healthcheck_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/healthcheck"
)

func TestHealthCheckService(t *testing.T) {
	hc := healthcheck.New()

	requireStatusUnknown(t, hc.CheckService("service"))
	update, err := hc.Register("service")
	require.Nil(t, err)
	requireStatusUnknown(t, hc.CheckService("service"))
	update(healthcheck.Serving)
	require.Equal(t, healthcheck.Serving, hc.CheckService("service"))
	update(healthcheck.NotServing)
	require.Equal(t, healthcheck.NotServing, hc.CheckService("service"))
}

func TestHealthCheckServer(t *testing.T) {
	hc := healthcheck.New()
	require.Equal(t, healthcheck.Serving, hc.CheckServer())

	update, err := hc.Register("service")
	require.Nil(t, err)
	requireStatusUnknown(t, hc.CheckServer())

	update(healthcheck.Serving)
	require.Equal(t, healthcheck.Serving, hc.CheckServer())
	update(healthcheck.NotServing)
	require.Equal(t, healthcheck.NotServing, hc.CheckServer())

	hc.Unregister("service")
	require.Equal(t, healthcheck.Serving, hc.CheckServer())
}

func TestHealthCheckRegisterTwice(t *testing.T) {
	hc := healthcheck.New()

	_, err := hc.Register("service")
	require.Nil(t, err)
	_, err = hc.Register("service")
	require.NotNil(t, err)

	hc.Unregister("service")
	_, err = hc.Register("service")
	require.Nil(t, err)
}

func TestHealthCheckUnregisteredService(t *testing.T) {
	hc := healthcheck.New()
	requireStatusUnknown(t, hc.CheckService("not_exist"))

	hc = healthcheck.New(healthcheck.WithUnregisteredServiceStatus(healthcheck.Serving))
	require.Equal(t, healthcheck.Serving, hc.CheckService("not_exist"))
}

func TestHealthCheckStatusWatchers(t *testing.T) {
	const serviceName = "service"
	var firstCalled int
	hc := healthcheck.New(healthcheck.WithStatusWatchers(map[string][]func(healthcheck.Status){
		serviceName: {
			func(status healthcheck.Status) {
				switch firstCalled++; firstCalled {
				case 1:
					require.Equal(t, healthcheck.Unknown, status)
				case 2:
					require.Equal(t, healthcheck.Serving, status)
				case 3:
					require.Equal(t, healthcheck.NotServing, status)
				default:
					require.FailNow(t, "status should only be updated 3 times")
				}
			},
		},
	}))
	update, err := hc.Register(serviceName)
	require.Nil(t, err)

	var secondCalled int
	hc.Watch(serviceName, func(status healthcheck.Status) {
		switch secondCalled++; secondCalled {
		case 1:
			require.Equal(t, healthcheck.Serving, status)
		case 2:
			require.Equal(t, healthcheck.NotServing, status)
		default:
			require.FailNow(t, "onStatusChanged should only be called at most 2 times")
		}
	})

	update(healthcheck.Serving)
	update(healthcheck.NotServing)
	require.Equal(t, 3, firstCalled)
	require.Equal(t, 2, secondCalled)
}

func requireStatusUnknown(t *testing.T, status healthcheck.Status) {
	require.NotContains(t, []healthcheck.Status{healthcheck.Serving, healthcheck.NotServing}, status)
}
