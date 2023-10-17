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
	"strings"
	"sync"

	"trpc.group/trpc-go/trpc-go/internal/stack"
)

var (
	errNotMatch       = errors.New("not match to the path template")
	errVerbMismatched = errors.New("verb mismatched")
)

// matcher is used to match variable values from template.
type matcher struct {
	components []string             // urlPath: "/foo/bar/baz" => []string{"foo","bar","baz"}
	pos        int                  // pos is the current match position, initialized before every match.
	captured   map[string]string    // values that has already been captured.
	st         *stack.Stack[string] // st is a stack to aid the matching process.
}

// matcher pool.
var matcherPool = sync.Pool{
	New: func() interface{} {
		return &matcher{}
	},
}

// putBackMatch puts the matcher back to the pool.
func putBackMatcher(m *matcher) {
	m.components = nil
	m.pos = 0
	m.captured = nil
	m.st.Reset()
	stackPool.Put(m.st)
	m.st = nil
	matcherPool.Put(m)
}

// stack pool.
var stackPool = sync.Pool{
	New: func() interface{} {
		return stack.New[string]()
	},
}

// handle implements segment.
func (wildcard) handle(m *matcher) error {
	// prevent out-of-bounds error.
	if m.pos >= len(m.components) {
		return errNotMatch
	}

	// "*" could match any component, push it into the stack directly.
	m.st.Push(m.components[m.pos])
	m.pos++

	return nil
}

// handle implements segment.
func (deepWildcard) handle(m *matcher) error {
	// prevent out-of-bounds error.
	if m.pos > len(m.components) {
		return errNotMatch
	}
	// m.pos = len(m.components) is allowed, because "**" could match any number of components.
	if m.pos == len(m.components) {
		m.st.Push("")
		return nil
	}

	var sb strings.Builder
	for ; m.pos < len(m.components); m.pos++ {
		sb.WriteRune('/')
		sb.WriteString(m.components[m.pos])
	}
	m.st.Push(sb.String()[1:])

	return nil
}

// handle implements segment.
func (l literal) handle(m *matcher) error {
	// prevent out-of-bounds error.
	if m.pos >= len(m.components) {
		return errNotMatch
	}

	// literal value should equal to the current component.
	if m.components[m.pos] != l.String() {
		return errNotMatch
	}

	// matched, push it into the stack.
	m.st.Push(m.components[m.pos])
	m.pos++

	return nil
}

// handle implements segment.
func (v variable) handle(m *matcher) error {
	// match segments recursively.
	for _, segment := range v.segments {
		if err := segment.handle(m); err != nil {
			return err
		}
	}

	// concatenate the popped components.
	// the final result is the captured value of v.fieldPath.
	concat := make([]string, len(v.segments))
	for i := len(v.segments) - 1; i >= 0; i-- {
		s, ok := m.st.Pop()
		if !ok {
			return errNotMatch
		}
		concat[i] = s
	}
	m.captured[strings.Join(v.fp, ".")] = strings.Join(concat, "/")

	return nil
}

// Match matches the value of variables according to the given HTTP URL path.
func (tpl *PathTemplate) Match(urlPath string) (map[string]string, error) {
	// must start with '/'
	if !strings.HasPrefix(urlPath, "/") {
		return nil, errNotMatch
	}

	urlPath = urlPath[1:]
	components := strings.Split(urlPath, "/")

	// verb match.
	if tpl.verb != "" {
		if !strings.HasSuffix(components[len(components)-1], ":"+tpl.verb) {
			return nil, errVerbMismatched
		}
		idx := len(components[len(components)-1]) - len(tpl.verb) - 1
		if idx <= 0 {
			return nil, errVerbMismatched
		}
		components[len(components)-1] = components[len(components)-1][:idx]
	}

	// use sync.Pool to reuse memory, since the initialization of matcher/match
	// is of high frequency.
	m := matcherPool.Get().(*matcher)
	defer putBackMatcher(m)
	m.components = components
	m.captured = make(map[string]string)
	// use sync.Pool to reuse memory of stack.Stack.
	m.st = stackPool.Get().(*stack.Stack[string])

	// segments match.
	for _, segment := range tpl.segments {
		if err := segment.handle(m); err != nil {
			return nil, err
		}
	}

	// check whether pos reaches the end.
	if m.pos != len(m.components) {
		return nil, errNotMatch
	}

	return m.captured, nil
}
