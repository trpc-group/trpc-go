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

package packetbuffer

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketReader(t *testing.T) {
	buf := New(make([]byte, 65535))
	assert.Equal(t, buf.UnRead(), 0)
	data := []byte("helloworld")
	copy(buf.Bytes(), data)
	buf.Advance(len(data))
	assert.NotEqual(t, buf.UnRead(), 0)
	b := make([]byte, 128)
	n, err := buf.Read(b[0:5])
	assert.Nil(t, err)
	assert.Equal(t, n, 5)
	_, err = buf.Read(b)
	assert.Equal(t, err, io.EOF)
	buf.Reset()
	assert.Equal(t, buf.UnRead(), 0)
	_, err = buf.Read(nil)
	assert.Equal(t, err, io.EOF)
}
