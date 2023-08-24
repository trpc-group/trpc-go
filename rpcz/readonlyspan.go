// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import (
	"fmt"
	"strings"
	"time"
)

// ReadOnlySpan is exported from writable span to achieve concurrency safety.
type ReadOnlySpan struct {
	ID   SpanID
	Name string

	// StartTime is the time at which this span was started.
	StartTime time.Time

	// EndTime is the time at which this span was ended. It contains the zero
	// value of time.Time until the span is ended.
	EndTime time.Time

	// used at route server when client send request.
	ChildSpans []*ReadOnlySpan

	// track event, used at codec and kinds of filter.
	Events []Event

	// Attributes records attributes of span.
	Attributes []Attribute
}

// PrintSketch prints sketched description of span.
func (s *ReadOnlySpan) PrintSketch(indentation string) string {
	return s.printSketch(nil, indentation)
}

// PrintDetail prints detailed description of span.
func (s *ReadOnlySpan) PrintDetail(indentation string) string {
	return s.printDetail(nil, indentation)
}

func (s *ReadOnlySpan) printSketch(parent *ReadOnlySpan, indentation string) string {
	content := fmt.Sprintf("%sspan: (%s, %d)\n", indentation, s.Name, s.ID)
	indentation += "  "
	content += fmt.Sprintf("%stime: (%s, %s)\n", indentation, s.StartTime.Format(time.StampMicro), s.printEndTime())
	content += fmt.Sprintf(
		"%sduration: (%s, %s, %s)\n",
		indentation,
		s.printPrevDuration(parent),
		s.printDuration(),
		s.printPostDuration(parent),
	)
	if attributes := s.printAttributes(); len(attributes) != 0 {
		content += fmt.Sprintf("%sattributes: %s\n", indentation, attributes)
	}
	return content
}

func (s *ReadOnlySpan) printDetail(parent *ReadOnlySpan, indentation string) string {
	content := s.printSketch(parent, indentation)
	content += s.printChildSpansAndEvents(indentation + "  ")
	return content
}

// printChildSpansAndEvents prints ChildSpans and Events in order of StartTime alternatively.
func (s *ReadOnlySpan) printChildSpansAndEvents(indentation string) string {
	var (
		content string
		i, j    int
	)
	for i < len(s.Events) && j < len(s.ChildSpans) {
		if s.Events[i].Time.Before(s.ChildSpans[j].StartTime) {
			content += fmt.Sprintf("%sevent: %s\n", indentation, printEvent(s.Events[i]))
			i++
		} else {
			content += s.ChildSpans[j].printDetail(s, indentation)
			j++
		}
	}
	for ; i < len(s.Events); i++ {
		content += fmt.Sprintf("%sevent: %s\n", indentation, printEvent(s.Events[i]))
	}
	for ; j < len(s.ChildSpans); j++ {
		content += s.ChildSpans[j].printDetail(s, indentation)
	}
	return content
}

func printEvent(e Event) string {
	return fmt.Sprintf("(%s, %v)", e.Name, e.Time.Format(time.StampMicro))
}

func (s *ReadOnlySpan) printAttributes() string {
	var content []string
	for _, a := range s.Attributes {
		if a.Name == TRPCAttributeFilterNames {
			continue
		}
		content = append(content, fmt.Sprintf("(%s, %v)", parsedTRPCAttribute(a.Name), a.Value))
	}
	return strings.Join(content, ",")
}

const (
	unknownEnd   = "unknown"
	zeroDuration = "0"
)

func (s *ReadOnlySpan) printPrevDuration(parent *ReadOnlySpan) string {
	if parent == nil {
		return zeroDuration
	}
	return fmt.Sprint(s.StartTime.Sub(parent.StartTime))
}

func (s *ReadOnlySpan) printDuration() string {
	if s.EndTime.IsZero() {
		return unknownEnd
	}
	return fmt.Sprint(s.EndTime.Sub(s.StartTime))
}

func (s *ReadOnlySpan) printEndTime() string {
	if s.EndTime.IsZero() {
		return unknownEnd
	}
	return fmt.Sprint(s.EndTime.Format(time.StampMicro))
}

func (s *ReadOnlySpan) printPostDuration(parent *ReadOnlySpan) string {
	if parent == nil {
		return zeroDuration
	}
	if parent.EndTime.IsZero() || s.EndTime.IsZero() {
		return unknownEnd
	}
	return fmt.Sprint(parent.EndTime.Sub(s.EndTime))
}
