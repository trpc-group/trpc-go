package rpcz

import (
	"sync"
	"time"
)

// Ender tells an operation to end its work.
// Ender will return to caller while creating a Span by NewSpanContext or Span.NewChild,
// to remind the caller to call Ender.End when this Span is completed.
// Usually using type EndFunc func() is better than Ender interface.
// However, return a function, method, function literals: https://go.dev/ref/spec#Function_literals or closures
// can't avoid memory allocated on heap if them are not inlineable. https://github.com/golang/go/issues/28727
type Ender interface {
	// End does not wait for the work to stop.
	// End  can only be called once.
	// After the first call, subsequent calls to End is undefined behavior.
	End()
}

// Span represents a unit of work or operation.
// It tracks specific operations that a request makes,
// painting a picture of what happened during the time in which that operation was executed.
type Span interface {
	// AddEvent adds a event.
	AddEvent(name string)

	// Event returns the time when event happened.
	Event(name string) (time.Time, bool)

	// SetAttribute sets Attribute with (name, value).
	SetAttribute(name string, value interface{})

	// Attribute gets the attribute value by the attribute name,
	// and the returned result is a shallow copy of the attribute value.
	// In general, you should not directly modify the return value,
	// otherwise it may affect the associated Span,
	// cause other goroutine to read dirty data from Span's Attribute.
	Attribute(name string) (interface{}, bool)

	// Name returns the name of Span.
	Name() string

	// ID returns SpanID.
	ID() SpanID

	// StartTime returns start time of the span.
	StartTime() time.Time

	// EndTime returns end time of the span.
	EndTime() time.Time

	// NewChild creates a child span from current span.
	// Ender ends this span if related operation is completed.
	NewChild(name string) (Span, Ender)

	// Child returns the child of current span.
	Child(name string) (Span, bool)
}

type recorder interface {
	record(child *span)
}

type span struct {
	m sync.Mutex // guards endTime, childSpans, events and attributes.

	id     SpanID
	name   string
	parent recorder

	// startTime is the time at which this span was started.
	startTime time.Time

	// endTime is the time at which this span was ended. It contains the zero
	// value of time.Time until the span is ended.
	endTime time.Time

	// childSpans stores child span created from current span.
	childSpans []*span

	// events track event, used at codec and kinds of filter.
	events []Event

	// attributes records attributes
	attributes []Attribute
}

func (s *span) AddEvent(name string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.events = append(s.events, Event{Name: name, Time: time.Now()})
}

func (s *span) Event(name string) (time.Time, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].Name == name {
			return s.events[i].Time, true
		}
	}
	return time.Time{}, false
}

func (s *span) SetAttribute(name string, value interface{}) {
	s.m.Lock()
	defer s.m.Unlock()
	s.attributes = append(s.attributes, Attribute{Name: name, Value: value})
}

func (s *span) Attribute(name string) (interface{}, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.attribute(name)
}

// attribute should be guarded by mutex, and its concurrency is guaranteed by caller,
// and the returned interface{} is also not goroutine-safe.
func (s *span) attribute(name string) (interface{}, bool) {
	for i := len(s.attributes) - 1; i >= 0; i-- {
		if s.attributes[i].Name == name {
			return s.attributes[i].Value, true
		}
	}
	return nil, false
}

func (s *span) Name() string {
	return s.name
}

func (s *span) ID() SpanID {
	return s.id
}

func (s *span) StartTime() time.Time {
	return s.startTime
}

func (s *span) EndTime() time.Time {
	s.m.Lock()
	defer s.m.Unlock()
	return s.endTime
}

// End sets span's endTime and record span to its parent.
func (s *span) End() {
	if s.trySetEndTime(time.Now()) {
		s.parent.record(s)
	}
}

func (s *span) trySetEndTime(t time.Time) bool {
	s.m.Lock()
	defer s.m.Unlock()
	if !s.endTime.IsZero() {
		return false
	}
	s.endTime = t
	return true
}

func (s *span) record(childSpan *span) {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.hasChild(childSpan) {
		putSpanToPool(childSpan)
	}
}

func (s *span) hasChild(childSpan *span) bool {
	for _, cs := range s.childSpans {
		if cs == childSpan {
			return true
		}
	}
	return false
}

func (s *span) NewChild(name string) (Span, Ender) {
	cs := newSpan(name, s.id, s)

	s.m.Lock()
	s.childSpans = append(s.childSpans, cs)
	s.m.Unlock()
	return cs, cs
}

func (s *span) Child(name string) (Span, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, cs := range s.childSpans {
		if cs.name == name {
			return cs, true
		}
	}
	return nil, false
}

func (s *span) isEnded() bool {
	s.m.Lock()
	defer s.m.Unlock()
	return !s.endTime.IsZero()
}

func (s *span) convertedToReadOnlySpan() *ReadOnlySpan {
	s.m.Lock()
	defer s.m.Unlock()
	readOnlySpan := &ReadOnlySpan{
		ID:        s.id,
		Name:      s.name,
		StartTime: s.startTime,
		EndTime:   s.endTime,
	}

	readOnlySpan.ChildSpans = make([]*ReadOnlySpan, len(s.childSpans))
	for i, cs := range s.childSpans {
		readOnlySpan.ChildSpans[i] = cs.convertedToReadOnlySpan()
	}

	readOnlySpan.Attributes = make([]Attribute, len(s.attributes))
	copy(readOnlySpan.Attributes, s.attributes)

	readOnlySpan.Events = make([]Event, len(s.events))
	copy(readOnlySpan.Events, s.events)

	return readOnlySpan
}

var spanPool = sync.Pool{New: func() interface{} { return new(span) }}

// newSpan creates root or child span, if parent is nil, creates a root span.
func newSpan(name string, id SpanID, parent recorder) *span {
	s := spanPool.Get().(*span)
	s.id = id
	s.name = name
	s.parent = parent
	s.startTime = time.Now()
	return s
}

func putSpanToPool(s *span) {
	s.m.Lock()
	defer s.m.Unlock()

	s.id = nilSpanID
	s.name = ""
	s.parent = nil
	s.startTime = time.Time{}
	s.endTime = time.Time{}
	s.events = s.events[:0]
	s.attributes = s.attributes[:0]

	for _, cs := range s.childSpans {
		if cs.isEnded() {
			putSpanToPool(cs)
		}
	}
	s.childSpans = s.childSpans[:0]

	spanPool.Put(s)
}
