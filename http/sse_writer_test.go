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

package http

import (
	"bytes"
	"errors"
	"testing"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/assert"
)

type errorWriter struct {
	failAt int
	count  int
}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	e.count++
	if e.count == e.failAt {
		return 0, errors.New("write error")
	}
	return len(p), nil
}

func TestWriteSSEEvent(t *testing.T) {
	event := sse.Event{
		ID:    []byte("1"),
		Event: []byte("message"),
		Retry: []byte("1000"),
		Data:  []byte("This is a test message"),
	}
	var buf bytes.Buffer
	err := WriteSSE(&buf, event)
	assert.NoError(t, err)

	expected := "id:1\nevent:message\nretry:1000\ndata:This is a test message\n\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteSSEEventError(t *testing.T) {
	event := sse.Event{
		ID:    []byte("1"),
		Event: []byte("message"),
		Retry: []byte("1000"),
		Data:  []byte("test data"),
	}
	err := WriteSSE(&errorWriter{failAt: 1}, event)
	assert.Error(t, err)
}

func TestWriteId(t *testing.T) {
	var buf bytes.Buffer
	err := writeID(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())

	err = writeID(&buf, []byte("123"))
	assert.NoError(t, err)

	expected := "id:123\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteIdError(t *testing.T) {
	for i := 1; i <= 3; i++ {
		err := writeID(&errorWriter{failAt: i}, []byte("123"))
		assert.Error(t, err)
	}
}

func TestWriteEvent(t *testing.T) {
	var buf bytes.Buffer
	er := writeEvent(&buf, nil)
	assert.NoError(t, er)
	assert.Equal(t, "", buf.String())

	err := writeEvent(&buf, []byte("test-event"))
	assert.NoError(t, err)

	expected := "event:test-event\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteEventError(t *testing.T) {
	for i := 1; i <= 3; i++ {
		err := writeEvent(&errorWriter{failAt: i}, []byte("test-event"))
		assert.Error(t, err)
	}
}

func TestWriteRetry(t *testing.T) {
	var buf bytes.Buffer
	err := writeRetry(&buf, []byte("5000"))
	assert.NoError(t, err)

	expected := "retry:5000\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteRetryError(t *testing.T) {
	for i := 1; i <= 3; i++ {
		err := writeRetry(&errorWriter{failAt: i}, []byte("5000"))
		assert.Error(t, err)
	}
}

func TestWriteRetryInvalid(t *testing.T) {
	var buf bytes.Buffer
	err := writeRetry(&buf, []byte("invalid"))
	assert.NoError(t, err)

	expected := ""
	assert.Equal(t, expected, buf.String())
}

func TestWriteRetryZero(t *testing.T) {
	var buf bytes.Buffer
	err := writeRetry(&buf, []byte("0"))
	assert.NoError(t, err)

	expected := ""
	assert.Equal(t, expected, buf.String())
}

func TestWriteData(t *testing.T) {
	var buf bytes.Buffer
	err := writeData(&buf, []byte("test data"))
	assert.NoError(t, err)

	expected := "data:test data\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteDataError(t *testing.T) {
	for i := 1; i <= 3; i++ {
		err := writeData(&errorWriter{failAt: i}, []byte("test data"))
		assert.Error(t, err)
	}
}
