// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package config

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_search(t *testing.T) {
	type args struct {
		unmarshalledData map[string]interface{}
		keys             []string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty keys",
			args: args{
				keys: nil,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if !errors.Is(err, ErrConfigNotExist) {
					t.Errorf("received unexpected error got: %+v, want: +%v", err, ErrCodecNotExist)
					return false
				}
				return true
			},
		},
		{
			name: "key doesn't match",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": []string{"x", "y"},
				},
				keys: []string{"not-1"},
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if !errors.Is(err, ErrConfigNotExist) {
					t.Errorf("received unexpected error got: %+v, want: +%v", err, ErrCodecNotExist)
					return false
				}
				return true
			},
		},
		{
			name: "value of unmarshalledData isn't map type",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": []string{"x", "y"},
				},
				keys: []string{"1", "2"},
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if !errors.Is(err, ErrConfigNotExist) {
					t.Errorf("received unexpected error got: %+v, want: +%v", err, ErrCodecNotExist)
					return false
				}
				return true
			},
		},
		{
			name: "value of unmarshalledData is map[interface{}]interface{} type",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": map[interface{}]interface{}{"x": "y"},
				},
				keys: []string{"1", "x"},
			},
			want:    "y",
			wantErr: assert.NoError,
		},
		{
			name: "value of unmarshalledData is map[string]interface{} type",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": map[string]interface{}{"x": "y"},
				},
				keys: []string{"1", "x"},
			},
			want:    "y",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := search(tt.args.unmarshalledData, tt.args.keys)
			if !tt.wantErr(t, err, fmt.Sprintf("search(%v, %v)", tt.args.unmarshalledData, tt.args.keys)) {
				return
			}
			assert.Equalf(t, tt.want, got, "search(%v, %v)", tt.args.unmarshalledData, tt.args.keys)
		})
	}
}

func TestYamlCodec_Unmarshal(t *testing.T) {
	t.Run("interface", func(t *testing.T) {
		var tt interface{}
		tt = map[string]interface{}{}
		require.Nil(t, GetCodec("yaml").Unmarshal([]byte("[1, 2]"), &tt))
	})
	t.Run("map[string]interface{}", func(t *testing.T) {
		tt := map[string]interface{}{}
		require.NotNil(t, GetCodec("yaml").Unmarshal([]byte("[1, 2]"), &tt))
	})
}
