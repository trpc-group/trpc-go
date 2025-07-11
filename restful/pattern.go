//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package restful

import "trpc.group/trpc-go/trpc-go/internal/httprule"

// Pattern makes *httprule.PathTemplate accessible.
type Pattern struct {
	*httprule.PathTemplate
}

// Parse parses the url path into a *Pattern. It should only be used by trpc-cmdline.
func Parse(urlPath string) (*Pattern, error) {
	tpl, err := httprule.Parse(urlPath)
	if err != nil {
		return nil, err
	}
	return &Pattern{tpl}, nil
}

// Enforce ensures the url path is legal (will panic if illegal) and parses it into a *Pattern.
func Enforce(urlPath string) *Pattern {
	pattern, err := Parse(urlPath)
	if err != nil {
		panic(err)
	}
	return pattern
}
