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

import "strings"

// trieNode represents a node in the prefix tree.
type trieNode struct {
	indices  string
	paths    []string
	children []*trieNode

	paramChild    *trieNode
	wildcardChild *trieNode

	transcoder *transcoder
	isEnd      bool
}

func newTrieNode() *trieNode {
	return &trieNode{
		indices:  "",
		children: make([]*trieNode, 0),
	}
}

type methodTree struct {
	method string
	root   *trieNode
}

// RouterTrie is a routing trie organized by HTTP method.
type RouterTrie struct {
	trees []*methodTree
}

func newRouterTrie() *RouterTrie {
	return &RouterTrie{
		trees: make([]*methodTree, 0),
	}
}

func (rt *RouterTrie) insert(method string, tr *transcoder) error {
	root := rt.getOrCreateRoot(method)

	segments := parsePattern(tr.pat.RawURLPath())
	current := root
	for i, seg := range segments {
		isLast := i == len(segments)-1

		switch seg.typ {
		case segmentTypeStatic:
			child := current.getChild(seg.value)
			if child == nil {
				child = current.addChild(seg.value)
			}
			current = child
		case segmentTypeParam:
			if current.paramChild == nil {
				current.paramChild = newTrieNode()
			}
			current = current.paramChild
		case segmentTypeWildcard:
			if current.wildcardChild == nil {
				current.wildcardChild = newTrieNode()
			}
			current = current.wildcardChild
		}

		if isLast {
			current.isEnd = true
			current.transcoder = tr
		}
	}

	return nil
}

func (rt *RouterTrie) getOrCreateRoot(method string) *trieNode {
	for _, tree := range rt.trees {
		if tree.method == method {
			return tree.root
		}
	}

	tree := &methodTree{
		method: method,
		root:   newTrieNode(),
	}
	rt.trees = append(rt.trees, tree)
	return tree.root
}

func (node *trieNode) getChild(path string) *trieNode {
	if path == "" {
		return nil
	}

	firstChar := path[0]
	for i := 0; i < len(node.indices); i++ {
		if node.indices[i] == firstChar && node.paths[i] == path {
			return node.children[i]
		}
	}
	return nil
}

func (node *trieNode) addChild(path string) *trieNode {
	child := newTrieNode()
	node.indices += string(path[0])
	node.paths = append(node.paths, path)
	node.children = append(node.children, child)
	return child
}

type segmentType int

const (
	segmentTypeStatic segmentType = iota
	segmentTypeParam
	segmentTypeWildcard
)

type pathSegment struct {
	typ   segmentType
	value string
}

func parsePattern(pattern string) []pathSegment {
	pattern = strings.TrimPrefix(pattern, "/")
	if pattern == "" {
		return nil
	}

	parts := splitPatternParts(pattern)
	segments := make([]pathSegment, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if part == "**" || part == "*" {
			segments = append(segments, pathSegment{typ: segmentTypeWildcard, value: part})
			continue
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramContent := part[1 : len(part)-1]
			if paramContent == "" {
				continue
			}
			if eqIdx := strings.Index(paramContent, "="); eqIdx > 0 {
				prefixPattern := paramContent[eqIdx+1:]
				for _, prefixPart := range strings.Split(prefixPattern, "/") {
					if prefixPart == "" {
						continue
					}
					if prefixPart == "**" || prefixPart == "*" {
						segments = append(segments, pathSegment{typ: segmentTypeWildcard, value: prefixPart})
					} else {
						segments = append(segments, pathSegment{typ: segmentTypeStatic, value: prefixPart})
					}
				}
			} else {
				segments = append(segments, pathSegment{typ: segmentTypeParam, value: paramContent})
			}
			continue
		}
		segments = append(segments, pathSegment{typ: segmentTypeStatic, value: part})
	}

	return segments
}

func splitPatternParts(pattern string) []string {
	var parts []string
	var current strings.Builder
	braceDepth := 0

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '{':
			braceDepth++
			current.WriteByte(c)
		case '}':
			braceDepth--
			current.WriteByte(c)
		case '/':
			if braceDepth == 0 {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func (rt *RouterTrie) search(method, path string) *transcoder {
	root := rt.getRoot(method)
	if root == nil {
		return nil
	}

	path = strings.TrimPrefix(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) == 1 && segments[0] == "" {
		segments = nil
	}

	return rt.dfs(root, segments, 0)
}

func (rt *RouterTrie) getRoot(method string) *trieNode {
	for _, tree := range rt.trees {
		if tree.method == method {
			return tree.root
		}
	}
	return nil
}

func (rt *RouterTrie) dfs(node *trieNode, segments []string, index int) *transcoder {
	if index == len(segments) {
		if node.isEnd && node.transcoder != nil {
			return node.transcoder
		}
		return nil
	}

	currentSegment := segments[index]
	if child := node.getChild(currentSegment); child != nil {
		if tr := rt.dfs(child, segments, index+1); tr != nil {
			return tr
		}
	}

	if node.paramChild != nil {
		if tr := rt.dfs(node.paramChild, segments, index+1); tr != nil {
			return tr
		}
	}

	if node.wildcardChild != nil {
		if tr := rt.dfs(node.wildcardChild, segments, len(segments)); tr != nil {
			return tr
		}
	}

	return nil
}
