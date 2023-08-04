package frame_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/transport/internal/frame"
)

func TestShouldCopy(t *testing.T) {
	type args struct {
		isCopyOption bool
		serverAsync  bool
		isSafeFramer bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"is safe framer: not copy", args{false, false, true}, false},
		{"not safe framer, sync mod, option not copy: not copy", args{false, false, false}, false},
		{"not safe framer, sync mod, option copy: copy", args{true, false, false}, true},
		{"not safe framer, async mod, option not copy: copy", args{false, true, false}, true},
		{"not safe framer, async mod, option copy: copy", args{true, true, false}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frame.ShouldCopy(
				tt.args.isCopyOption,
				tt.args.serverAsync,
				tt.args.isSafeFramer,
			); got != tt.want {
				t.Errorf("ShouldCopy() = %v, want %v", got, tt.want)
			}
		})
	}
}
