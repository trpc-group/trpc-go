// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package dat_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/dat"
)

var fps = [][]string{
	{"baz"},
	{"foobar", "foo"},
	{"foobar", "bar"},
	{"foobar", "baz", "baz"},
	{"foo", "bar", "baz", "qux"},
}

func TestBuild(t *testing.T) {
	if got, err := dat.Build(fps); err != nil || got == nil {
		t.Errorf("Build(%v) (got, error) = %v, %v, (want, wantErr) = (not nil, nil)", fps, got, err)
	}
}

func TestCommonPrefixSearch(t *testing.T) {
	trie := mustBuild(t, fps)
	for _, tt := range []struct {
		name  string
		input []string
		want  bool
	}{
		{
			name:  "fail-1",
			input: []string{"foobar", "baz"},
			want:  false,
		},
		{
			name:  "fail-2",
			input: []string{"bar1"},
			want:  false,
		},
		{
			name:  "fail-3",
			input: []string{},
			want:  false,
		},
		{
			name:  "fail-4",
			input: []string{"foobar"},
			want:  false,
		},
		{
			name:  "success-1",
			input: []string{"foobar", "foo"},
			want:  true,
		},
		{
			name:  "success-2",
			input: []string{"foo", "bar", "baz", "qux"},
			want:  true,
		},
		{
			name:  "success-3",
			input: []string{"foo", "bar", "baz", "qux", "any"},
			want:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := trie.CommonPrefixSearch(tt.input); got != tt.want {
				t.Errorf("dat.CommonPrefixSearch(%v) got = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func mustBuild(t *testing.T, fps [][]string) *dat.DoubleArrayTrie {
	t.Helper()
	trie, err := dat.Build(fps)
	if err != nil {
		t.Fatalf("could not build DoubleArrayTrie under test: %v", err)
	}
	return trie
}
