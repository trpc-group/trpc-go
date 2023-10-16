//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package codec

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReaderSize(t *testing.T) {
	assert.Equal(t, DefaultReaderSize, GetReaderSize())
	defer SetReaderSize(DefaultReaderSize)

	bufSize := 128 * 1024
	SetReaderSize(bufSize)
	assert.Equal(t, bufSize, GetReaderSize())
	SetReaderSize(0)
	assert.Equal(t, 0, GetReaderSize())
}

func TestNewReaderSize(t *testing.T) {
	orig := strings.NewReader("test")
	newer := NewReaderSize(orig, -1)
	assert.Equal(t, orig, newer)

	newer = NewReaderSize(orig, 0)
	assert.Equal(t, orig, newer)

	newer = NewReaderSize(orig, 32*1024)
	assert.NotEqual(t, orig, newer)
	bufioReader, ok := newer.(*bufio.Reader)
	assert.Equal(t, true, ok)
	assert.Equal(t, 32*1024, bufioReader.Size())
}

func TestNewReader(t *testing.T) {
	orig := strings.NewReader("test")
	newer := NewReader(orig)
	assert.NotEqual(t, orig, newer)
	bufioReader, ok := newer.(*bufio.Reader)
	assert.Equal(t, true, ok)
	assert.Equal(t, DefaultReaderSize, bufioReader.Size())
}

type fakeSafeFramer struct {
	safe bool
}

func (f *fakeSafeFramer) ReadFrame() ([]byte, error) {
	return nil, nil
}

func (f *fakeSafeFramer) IsSafe() bool {
	return f.safe
}

func TestIsSafeFramer(t *testing.T) {
	safeFrame := fakeSafeFramer{safe: true}
	assert.Equal(t, true, IsSafeFramer(&safeFrame))

	noSafeFrame := fakeSafeFramer{}
	assert.Equal(t, false, IsSafeFramer(&noSafeFrame))

	assert.Equal(t, false, IsSafeFramer(10))
}
