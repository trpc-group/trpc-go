package codec_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
)

func TestIsValidCompressType(t *testing.T) {
	tests := []struct {
		name string
		arg  int
		want bool
	}{
		{"valid compress type that is defined in codec", codec.CompressTypeSnappy, true},
		{"valid compress type that isn't defined in codec", 10000, true},
		{"invalid compress type", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := icodec.IsValidCompressType(tt.arg); got != tt.want {
				t.Errorf("IsValidCompressType() = %v, want %v", got, tt.want)
			}
		})
	}
}
