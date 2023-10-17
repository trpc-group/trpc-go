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

package httprule

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTemplate(t *testing.T) {
	for _, test := range []struct {
		input   string
		wantTpl *PathTemplate
		wantErr error
		desc    string
	}{
		{
			input: "/foo/{bar}",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foo"),
					variable{fp: []string{"bar"}, segments: []segment{wildcard{}}},
				},
			},
			wantErr: nil,
			desc:    "ParseTemplateTest01",
		},
		{
			input: "/foo/*:bar",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foo"),
					wildcard{},
				},
				verb: "bar",
			},
			wantErr: nil,
			desc:    "ParseTemplateTest02",
		},
		{
			input: "/foo/**:bar",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foo"),
					deepWildcard{},
				},
				verb: "bar",
			},
			wantErr: nil,
			desc:    "ParseTemplateTest03",
		},
		{
			input: "/foo/*/bar:baz",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foo"),
					wildcard{},
					literal("bar"),
				},
				verb: "baz",
			},
			wantErr: nil,
			desc:    "ParseTemplateTest04",
		},
		{
			input: "/foobar/{foo.bar.baz}:qux",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foobar"),
					variable{fp: []string{"foo", "bar", "baz"}, segments: []segment{wildcard{}}},
				},
				verb: "qux",
			},
			wantErr: nil,
			desc:    "ParseTemplateTest05",
		},
		{
			input: "/foo/{bar=baz/**}",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foo"),
					variable{fp: []string{"bar"}, segments: []segment{literal("baz"), deepWildcard{}}},
				},
			},
			wantErr: nil,
			desc:    "ParseTemplateTest06",
		},
		{
			input: "/foo/{bar}/**",
			wantTpl: &PathTemplate{
				segments: []segment{
					literal("foo"),
					variable{fp: []string{"bar"}, segments: []segment{wildcard{}}},
					deepWildcard{},
				},
			},
			desc: "ParseTemplateTest07",
		},
	} {
		gotTpl, gotErr := Parse(test.input)
		require.Equal(t, true, gotErr == test.wantErr, test.desc)
		require.Equal(t, true, reflect.DeepEqual(gotTpl, test.wantTpl), test.desc)
	}
}

func TestFieldPaths(t *testing.T) {
	for _, test := range []struct {
		input string
		want  [][]string
		desc  string
	}{
		{
			input: "/foo/{bar}/baz/{qux}/*/**",
			want:  [][]string{{"bar"}, {"qux"}},
			desc:  "GetFieldPathsTest01",
		},
		{
			input: "/foo/bar",
			want:  nil,
			desc:  "GetFieldPathsTest02",
		},
	} {
		tpl, err := Parse(test.input)
		require.Nil(t, err, test.desc)
		require.Equal(t, test.want, tpl.FieldPaths(), test.desc)
	}
}

func TestVerb(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
		desc  string
	}{
		{
			input: "/a/b/c:d:e:f:g",
			want:  "g",
			desc:  "test verb 01",
		},
		{
			input: "/a/b/{c}:d:e:f:g",
			want:  "d:e:f:g",
			desc:  "test verb 02",
		},
		{
			input: "/a/b/*:d:e:f:g",
			want:  "d:e:f:g",
			desc:  "test verb 03",
		},
		{
			input: "/a/b/**:d:e:f:g",
			want:  "d:e:f:g",
			desc:  "test verb 04",
		},
		{
			input: "/a/b/*/**",
			want:  "",
			desc:  "test verb 05",
		},
	} {
		tpl, err := Parse(test.input)
		require.Nil(t, err, test.desc)
		require.Equal(t, tpl.verb, test.want, test.desc)
	}
}

func TestValidateTemplate(t *testing.T) {
	for _, test := range []struct {
		input   string
		wantErr error
		desc    string
	}{
		{
			input:   "/v1/{name=a/{nested}}",
			wantErr: errNestedVar,
			desc:    "validate tpl test 01",
		},
		{
			input:   "/v1/{name}/{name}",
			wantErr: errDupFieldPath,
			desc:    "validate tpl test 02",
		},
		{
			input:   "/v1/{name=**}/a/b/c",
			wantErr: errDeepWildcard,
			desc:    "validate tpl test 03",
		},
		{
			input:   "/v1/**/a/b/c",
			wantErr: errDeepWildcard,
			desc:    "validate tpl test 04",
		},
		{
			input:   "/v1/a/b/c/{name=**/b}",
			wantErr: errDeepWildcard,
			desc:    "validate tpl test 05",
		},
		{
			input:   "/v1/a:b:c/{name=**}",
			wantErr: nil,
			desc:    "validate tpl test 06",
		},
	} {
		_, gotErr := Parse(test.input)
		require.Equal(t, errors.Unwrap(gotErr), test.wantErr, test.desc)
	}
}
