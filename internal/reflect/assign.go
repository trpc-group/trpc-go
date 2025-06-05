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

// Package reflect provides internal implementations for reflection.
package reflect

import (
	"errors"
	"fmt"
	"reflect"
)

// Assign assigns the src value to the dst value using reflect.
func Assign(dst, src interface{}) error {
	reqDstVal := reflect.ValueOf(dst)
	if reqDstVal.Kind() != reflect.Ptr || reqDstVal.IsNil() {
		return errors.New("req must be a non-nil pointer")
	}

	reqSrcVal := reflect.ValueOf(src)
	if reqSrcVal.Kind() == reflect.Ptr {
		reqSrcVal = reqSrcVal.Elem() // Dereference pointer to get the value.
	}

	if !reqSrcVal.Type().AssignableTo(reqDstVal.Elem().Type()) {
		return fmt.Errorf("type mismatch: req dst type: %s, req src type: %s",
			reqDstVal.Elem().Type().String(), reqSrcVal.Type().String())
	}
	reqDstVal.Elem().Set(reqSrcVal)
	return nil
}
