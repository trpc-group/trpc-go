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

package restful_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/restful"
)

func TestPattern(t *testing.T) {
	for _, test := range []struct {
		input   string
		wantErr bool
		desc    string
	}{
		{
			input:   "/",
			wantErr: true,
			desc:    "test blank url path",
		},
		{
			input:   "!@#$%^&",
			wantErr: true,
			desc:    "test invalid url path",
		},
		{
			input:   "/foobar/foo/{bar}",
			wantErr: false,
			desc:    "test valid url path",
		},
	} {
		_, err := restful.Parse(test.input)
		require.Equal(t, test.wantErr, err != nil, test.desc)
		if test.wantErr {
			require.Panics(t, func() { restful.Enforce(test.input) }, test.desc)
		} else {
			require.NotPanics(t, func() { restful.Enforce(test.input) }, test.desc)
		}
	}
}
