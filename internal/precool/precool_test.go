package precool

import (
	"testing"

	"github.com/stretchr/testify/require"

	coreprecool "trpc.group/trpc-go/trpc-go/precool"
)

func TestNewAndWithUnregisteredServiceStatus(t *testing.T) {
	pc := New()
	require.Equal(t, coreprecool.Unknown, pc.unregisteredServiceStatus)

	pc = New(WithUnregisteredServiceStatus(coreprecool.Failure))
	require.Equal(t, coreprecool.Failure, pc.unregisteredServiceStatus)
}

func TestRegisterAndUnregister(t *testing.T) {
	pc := New()
	fn := func() coreprecool.Status { return coreprecool.Success }

	require.NoError(t, pc.Register("svc", fn))
	require.Error(t, pc.Register("svc", fn))

	pc.Unregister("svc")
	require.NoError(t, pc.Register("svc", fn))
}

func TestCheckService(t *testing.T) {
	pc := New(WithUnregisteredServiceStatus(coreprecool.Failure))
	require.Equal(t, coreprecool.Failure, pc.CheckService("notfound"))

	require.NoError(t, pc.Register("svc", func() coreprecool.Status { return coreprecool.Success }))
	require.Equal(t, coreprecool.Success, pc.CheckService("svc"))
}

func TestCheckServer(t *testing.T) {
	tests := []struct {
		name     string
		statuses map[string]coreprecool.Status
		want     coreprecool.Status
	}{
		{
			name:     "all success",
			statuses: map[string]coreprecool.Status{"a": coreprecool.Success, "b": coreprecool.Success},
			want:     coreprecool.Success,
		},
		{
			name:     "has failure",
			statuses: map[string]coreprecool.Status{"a": coreprecool.Success, "b": coreprecool.Failure},
			want:     coreprecool.Unknown,
		},
		{
			name:     "has ongoing",
			statuses: map[string]coreprecool.Status{"a": coreprecool.Success, "b": coreprecool.Ongoing},
			want:     coreprecool.Ongoing,
		},
		{
			name:     "mixed unknown",
			statuses: map[string]coreprecool.Status{"a": coreprecool.Success, "b": coreprecool.Unknown},
			want:     coreprecool.Failure,
		},
		{
			name:     "empty",
			statuses: map[string]coreprecool.Status{},
			want:     coreprecool.Success,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := New()
			for name, status := range tt.statuses {
				st := status
				require.NoError(t, pc.Register(name, func() coreprecool.Status { return st }))
			}
			require.Equal(t, tt.want, pc.CheckServer())
		})
	}
}
