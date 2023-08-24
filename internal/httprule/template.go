// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package httprule

import (
	"fmt"
	"strings"
)

// PathTemplate URL path template:
//
// Template = "/" Segments [ Verb ] ;
// Segments = Segment { "/" Segment } ;
// Segment  = "*" | "**" | LITERAL | Variable ;
// Variable = "{" FieldPath [ "=" Segments ] "}" ;
// FieldPath = IDENT { "." IDENT } ;
// Verb     = ":" LITERAL ;
type PathTemplate struct {
	segments []segment
	verb     string
}

// segment type.
type segmentKind int

const (
	kindWildcard segmentKind = iota
	kindDeepWildcard
	kindLiteral
	kindVariable
)

// Segment = "*" | "**" | LITERAL | Variable
type segment interface {
	fmt.Stringer
	handle(*matcher) error
	kind() segmentKind
	fieldPath() []string
	nestedSegments() []segment
}

var _ segment = wildcard{}
var _ segment = deepWildcard{}
var _ segment = literal("")
var _ segment = variable{}

// wildcard represents *.
type wildcard struct{}

// String implements segment.
func (wildcard) String() string {
	return "*"
}

// fieldPath implements segment.
func (wildcard) fieldPath() []string {
	return nil
}

// kind implements segment.
func (wildcard) kind() segmentKind {
	return kindWildcard
}

// nestedSegments implements segment.
func (wildcard) nestedSegments() []segment {
	return nil
}

// deepWildcard represents **.
type deepWildcard struct{}

// String implements segment.
func (deepWildcard) String() string {
	return "**"
}

// fieldPath implements segment.
func (deepWildcard) fieldPath() []string {
	return nil
}

// kind implements segment.
func (deepWildcard) kind() segmentKind {
	return kindDeepWildcard
}

// nestedSegments implements segment.
func (deepWildcard) nestedSegments() []segment {
	return nil
}

// literal, example: /foo.
type literal string

// String implements segment.
func (l literal) String() string {
	return string(l)
}

// fieldPath implements segment.
func (literal) fieldPath() []string {
	return nil
}

// kind implements segment.
func (literal) kind() segmentKind {
	return kindLiteral
}

// nestedSegments implements segment.
func (literal) nestedSegments() []segment {
	return nil
}

// variable is like {var=*}，Variable = "{" FieldPath [ "=" Segments ] "}"
type variable struct {
	fp       []string // FieldPath = IDENT { "." IDENT }
	segments []segment
}

// String imeplemts segment.
func (v variable) String() string {
	ss := make([]string, len(v.segments))
	for i := range v.segments {
		ss[i] = v.segments[i].String()
	}
	return fmt.Sprintf("{%s=%s}", strings.Join(v.fp, "."), strings.Join(ss, "/"))
}

// fieldPath implements segment.
func (v variable) fieldPath() []string {
	return v.fp
}

// kind implements segment.
func (variable) kind() segmentKind {
	return kindVariable
}

// nestedSegments implements segment.
func (v variable) nestedSegments() []segment {
	return v.segments
}

// FieldPaths gets field paths.
func (tpl *PathTemplate) FieldPaths() [][]string {
	var fps [][]string
	for _, segment := range tpl.segments {
		if fp := segment.fieldPath(); fp != nil {
			fps = append(fps, fp)
		}
	}
	return fps
}
