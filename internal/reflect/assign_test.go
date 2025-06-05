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

package reflect_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/reflect"
)

// TestAssignSuccess tests the successful assignment of values.
func TestAssignSuccess(t *testing.T) {
	var dst int
	src := 42

	if err := reflect.Assign(&dst, src); err != nil {
		t.Fatalf("Assign failed: %s", err)
	}
	if dst != src {
		t.Errorf("Expected dst to be %d, got %d", src, dst)
	}
}

// TestAssignTypeMismatch tests the assignment failure due to type mismatch.
func TestAssignTypeMismatch(t *testing.T) {
	var dst int
	src := "hello"

	if err := reflect.Assign(&dst, src); err == nil {
		t.Fatal("Expected type mismatch error, got nil")
	}
}

// TestAssignNonPointerDst tests passing a non-pointer as the destination.
func TestAssignNonPointerDst(t *testing.T) {
	var dst int
	src := 42

	if err := reflect.Assign(dst, src); err == nil {
		t.Fatal("Expected non-pointer error, got nil")
	}
}

// TestAssignNilPointerDst tests passing a nil pointer as the destination.
func TestAssignNilPointerDst(t *testing.T) {
	var dst *int
	src := 42

	if err := reflect.Assign(dst, src); err == nil {
		t.Fatal("Expected nil pointer error, got nil")
	}
}

// TestAssignPointerToPointer tests the assignment of pointer to pointer.
func TestAssignPointerToPointer(t *testing.T) {
	var dst int
	src := 42
	srcPtr := &src
	dstPtr := &dst

	if err := reflect.Assign(dstPtr, srcPtr); err != nil {
		t.Fatalf("Assign failed: %s", err)
	}
	if dstPtr == nil || *dstPtr != *srcPtr {
		t.Errorf("Expected dst to point to %d, got %d", *srcPtr, *dstPtr)
	}
}

// TestAssignDifferentPointerTypes tests the assignment of different pointer types.
func TestAssignDifferentPointerTypes(t *testing.T) {
	var dst *int
	src := "hello"
	srcPtr := &src

	if err := reflect.Assign(&dst, srcPtr); err == nil {
		t.Fatal("Expected type mismatch error, got nil")
	}
}
