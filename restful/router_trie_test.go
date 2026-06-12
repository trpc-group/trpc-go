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

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestTranscoder(t *testing.T, name, pattern string) *transcoder {
	t.Helper()
	pat, err := Parse(pattern)
	require.NoError(t, err)
	return &transcoder{name: name, pat: pat}
}

func TestPatternRawURLPath(t *testing.T) {
	pat, err := Parse("/api/users/{id}")
	require.NoError(t, err)
	require.Equal(t, "/api/users/{id}", pat.RawURLPath())

	var nilPattern *Pattern
	require.Empty(t, nilPattern.RawURLPath())
}

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []pathSegment
	}{
		{
			name:    "static",
			pattern: "/api/users",
			want: []pathSegment{
				{typ: segmentTypeStatic, value: "api"},
				{typ: segmentTypeStatic, value: "users"},
			},
		},
		{
			name:    "parameter",
			pattern: "/api/users/{id}",
			want: []pathSegment{
				{typ: segmentTypeStatic, value: "api"},
				{typ: segmentTypeStatic, value: "users"},
				{typ: segmentTypeParam, value: "id"},
			},
		},
		{
			name:    "wildcard",
			pattern: "/api/**",
			want: []pathSegment{
				{typ: segmentTypeStatic, value: "api"},
				{typ: segmentTypeWildcard, value: "**"},
			},
		},
		{
			name:    "constrained parameter keeps slash-delimited constraint",
			pattern: "/proxy/{path=trpc/go/**}",
			want: []pathSegment{
				{typ: segmentTypeStatic, value: "proxy"},
				{typ: segmentTypeStatic, value: "trpc"},
				{typ: segmentTypeStatic, value: "go"},
				{typ: segmentTypeWildcard, value: "**"},
			},
		},
		{
			name:    "skips empty parts",
			pattern: "/api//users/",
			want: []pathSegment{
				{typ: segmentTypeStatic, value: "api"},
				{typ: segmentTypeStatic, value: "users"},
			},
		},
		{
			name:    "root",
			pattern: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parsePattern(tt.pattern))
		})
	}
}

func TestRouterTrieSearch(t *testing.T) {
	trie := newRouterTrie()
	routes := []struct {
		method  string
		pattern string
		name    string
	}{
		{method: "GET", pattern: "/api/users", name: "ListUsers"},
		{method: "POST", pattern: "/api/users", name: "CreateUser"},
		{method: "GET", pattern: "/api/users/{id}", name: "GetUser"},
		{method: "GET", pattern: "/api/users/{id}/posts/{post_id}", name: "GetPost"},
		{method: "GET", pattern: "/api/**", name: "WildcardAPI"},
		{method: "GET", pattern: "/proxy/{path=trpc/go/**}", name: "Proxy"},
	}
	for _, route := range routes {
		require.NoError(t, trie.insert(route.method, newTestTranscoder(t, route.name, route.pattern)))
	}

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{method: "GET", path: "/api/users", want: "ListUsers"},
		{method: "POST", path: "/api/users", want: "CreateUser"},
		{method: "GET", path: "/api/users/123", want: "GetUser"},
		{method: "GET", path: "/api/users/123/posts/456", want: "GetPost"},
		{method: "GET", path: "/api/anything/else", want: "WildcardAPI"},
		{method: "GET", path: "/proxy/trpc/go/pkg/restful", want: "Proxy"},
		{method: "PATCH", path: "/api/users", want: ""},
		{method: "GET", path: "/api/users/123/missing/456", want: "WildcardAPI"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			got := trie.search(tt.method, tt.path)
			if tt.want == "" {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tt.want, got.name)
		})
	}
}

func TestRouterTriePriority(t *testing.T) {
	trie := newRouterTrie()
	routes := []struct {
		pattern string
		name    string
	}{
		{pattern: "/api/**", name: "Wildcard"},
		{pattern: "/api/users/{id}", name: "Param"},
		{pattern: "/api/users/me", name: "Static"},
	}
	for _, route := range routes {
		require.NoError(t, trie.insert("GET", newTestTranscoder(t, route.name, route.pattern)))
	}

	tests := []struct {
		path string
		want string
	}{
		{path: "/api/users/me", want: "Static"},
		{path: "/api/users/123", want: "Param"},
		{path: "/api/other/path", want: "Wildcard"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := trie.search("GET", tt.path)
			require.NotNil(t, got)
			require.Equal(t, tt.want, got.name)
		})
	}
}

func TestRouterFindTranscoderUsesTriePriority(t *testing.T) {
	router := NewRouter()
	for _, route := range []struct {
		pattern string
		name    string
	}{
		{pattern: "/api/users/{id}", name: "Param"},
		{pattern: "/api/users/me", name: "Static"},
	} {
		require.NoError(t, router.AddImplBinding(&Binding{
			Name:       route.name,
			Input:      func() ProtoMessage { return nil },
			Filter:     func(interface{}, context.Context, interface{}) (interface{}, error) { return nil, nil },
			HTTPMethod: "GET",
			Pattern:    Enforce(route.pattern),
		}, nil))
	}

	tr, fields := router.findTranscoder("GET", "/api/users/me")
	require.NotNil(t, tr)
	require.Equal(t, "Static", tr.name)
	require.Empty(t, fields)
}

func TestRouterTrieFallsBackForPatternMismatch(t *testing.T) {
	router := NewRouter()
	require.NoError(t, router.AddImplBinding(&Binding{
		Name:       "OneSegmentWildcard",
		Input:      func() ProtoMessage { return nil },
		Filter:     func(interface{}, context.Context, interface{}) (interface{}, error) { return nil, nil },
		HTTPMethod: "GET",
		Pattern:    Enforce("/files/*"),
	}, nil))

	tr, fields := router.findTranscoder("GET", "/files/a/b")
	require.Nil(t, tr)
	require.Nil(t, fields)
}

func TestRouterTrieNodeOperations(t *testing.T) {
	node := newTrieNode()
	require.Nil(t, node.getChild(""))

	api := node.addChild("api")
	require.Equal(t, api, node.getChild("api"))
	require.Nil(t, node.getChild("app"))

	users := node.addChild("users")
	require.Equal(t, users, node.getChild("users"))
	require.Equal(t, "au", node.indices)
}
