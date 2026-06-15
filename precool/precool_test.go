package precool_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/precool"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status precool.Status
		want   string
	}{
		{status: precool.Unknown, want: "unknown"},
		{status: precool.Success, want: "proc_success"},
		{status: precool.Failure, want: "proc_failure"},
		{status: precool.Ongoing, want: "proc_ongoing"},
		{status: precool.Status(99), want: "unknown"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.status.String())
	}
}
