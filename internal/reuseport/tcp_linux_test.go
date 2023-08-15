//go:build linux
// +build linux

package reuseport

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxListenerBackLog(t *testing.T) {
	oldMaxConnFileName := maxConnFileName
	defer func() {
		maxConnFileName = oldMaxConnFileName
	}()

	tests := []struct {
		name     string
		fileName string
		want     int
	}{
		{
			name:     "file not exist",
			fileName: "./testdata/NotExistFile.txt",
			want:     syscall.SOMAXCONN,
		},
		{
			name:     "file content invalid, no eof",
			fileName: "./testdata/NoEof.txt",
			want:     syscall.SOMAXCONN,
		},
		{
			name:     "empty line",
			fileName: "./testdata/EmptyLine.txt",
			want:     syscall.SOMAXCONN,
		},
		{
			name:     "num zero",
			fileName: "./testdata/NumZero.txt",
			want:     syscall.SOMAXCONN,
		},
		{
			name:     "num 65536",
			fileName: "./testdata/NumMax.txt",
			want:     65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxConnFileName = tt.fileName
			assert.Equalf(t, tt.want, maxListenerBacklog(), "maxListenerBacklog()")
		})
	}
}
