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

package fastop_test

import (
	"net/http"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/internal/http/fastop"
)

func TestCanonicalHeaderGet(t *testing.T) {
	header := make(http.Header)
	headerKey := textproto.CanonicalMIMEHeaderKey("X-Custom-Header")

	retrievedValue := fastop.CanonicalHeaderGet(header, headerKey)
	require.Empty(t, retrievedValue)

	headerValue := "TestValue"
	header.Add(headerKey, headerValue)

	retrievedValue = fastop.CanonicalHeaderGet(header, headerKey)
	if retrievedValue != headerValue {
		t.Errorf("CanonicalHeaderGet() = %v; want %v", retrievedValue, headerValue)
	}
}

func TestCanonicalHeaderAdd(t *testing.T) {
	header := make(http.Header)
	headerKey := textproto.CanonicalMIMEHeaderKey("X-Custom-Header")
	headerValue := "TestValue"

	// Add header.
	modifiedHeader := fastop.CanonicalHeaderAdd(header, headerKey, headerValue)
	if len(modifiedHeader[headerKey]) != 1 || modifiedHeader[headerKey][0] != headerValue {
		t.Errorf("CanonicalHeaderAdd() did not add the value correctly, got: %v, want: %v",
			modifiedHeader[headerKey], headerValue)
	}

	// Add another value to the same header.
	secondValue := "AnotherValue"
	modifiedHeader = fastop.CanonicalHeaderAdd(header, headerKey, secondValue)
	if len(modifiedHeader[headerKey]) != 2 || modifiedHeader[headerKey][1] != secondValue {
		t.Errorf("CanonicalHeaderAdd() did not add the second value correctly, got: %v, want: %v",
			modifiedHeader[headerKey][1], secondValue)
	}
}

func TestCanonicalHeaderSet(t *testing.T) {
	header := make(http.Header)
	headerKey := textproto.CanonicalMIMEHeaderKey("X-Custom-Header")
	firstValue := "FirstValue"
	secondValue := "SecondValue"

	// Set initial header.
	fastop.CanonicalHeaderSet(header, headerKey, firstValue)

	// Overwrite header.
	modifiedHeader := fastop.CanonicalHeaderSet(header, headerKey, secondValue)
	if len(modifiedHeader[headerKey]) != 1 || modifiedHeader[headerKey][0] != secondValue {
		t.Errorf("CanonicalHeaderSet() did not set the value correctly, got: %v, want: %v",
			modifiedHeader[headerKey][0], secondValue)
	}
}
