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

package server

import (
	"context"
	"runtime/pprof"

	"trpc.group/trpc-go/trpc-go/filter"
)

// ProfilerTagger is an interface that defines profiler tags, which can be used to tag goroutine.
type ProfilerTagger interface {
	Tag(ctx context.Context, req interface{}) (*ProfileLabel, error)
}

// StreamProfilerTagger is an interface that defines profiler tags for stream service,
// which can be used to tag goroutine.
type StreamProfilerTagger interface {
	// Tag tags a goroutine during the filter stage of an RPC call.
	// The granularity of the statistics is per RPC call.
	Tag(ctx context.Context, info *StreamServerInfo) (*ProfileLabel, error)
	// TagRecvMsg tags a goroutine each time a message is received.
	TagRecvMsg(ctx context.Context) (*ProfileLabel, error)
	// TagSendMsg tags a goroutine each time a message is sent.
	TagSendMsg(ctx context.Context, m interface{}) (*ProfileLabel, error)
}

// NewProfileLabel creates a new ProfileLabel object and initializes the labels field as an empty map.
func NewProfileLabel() *ProfileLabel {
	return &ProfileLabel{labels: make(map[string]string)}
}

// ProfileLabel is a struct that contains labels for storing key-value pairs.
type ProfileLabel struct {
	labels map[string]string
}

// Store stores the specified key-value pair in the ProfileLabel.
func (p *ProfileLabel) Store(key, value string) {
	p.labels[key] = value
}

// Load retrieves the value associated with the specified key from the ProfileLabel.
func (p *ProfileLabel) Load(key string) (string, bool) {
	value, ok := p.labels[key]
	return value, ok
}

// Len returns the number of key-value pairs stored in the ProfileLabel.
func (p *ProfileLabel) Len() int {
	return len(p.labels)
}

// toLabels converts the ProfileLabel to a string slice,
// where each key-value pair is represented as two consecutive strings.
func (p *ProfileLabel) toLabels() []string {
	labels := make([]string, 0, p.Len()*2)
	for k, v := range p.labels {
		labels = append(labels, k, v)
	}
	return labels
}

// profilerTaggerFilter returns a filter that assigns labels to goroutine.
func profilerTaggerFilter(
	tagger ProfilerTagger,
) filter.ServerFilter {
	return func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
		profileLabel, err := tagger.Tag(ctx, req)
		if err != nil {
			return nil, err
		}
		var labels []string
		if profileLabel != nil {
			labels = profileLabel.toLabels()
		}
		pprof.Do(ctx, pprof.Labels(labels...), func(ctx context.Context) {
			rsp, err = next(ctx, req)
		})
		return rsp, err
	}
}

// streamProfilerTaggerFilter returns a stream filter that assigns labels to goroutine.
func streamProfilerTaggerFilter(
	tagger StreamProfilerTagger,
) StreamFilter {
	return func(ss Stream, info *StreamServerInfo, handler StreamHandler) error {
		ctx := ss.Context()
		profileLabel, err := tagger.Tag(ctx, info)
		if err != nil {
			return err
		}
		var labels []string
		if profileLabel != nil {
			labels = profileLabel.toLabels()
		}
		pprof.Do(ctx, pprof.Labels(labels...), func(ctx context.Context) {
			ws := &wrappedStream{ss, tagger, labels}
			err = handler(ws)
		})
		return err
	}
}

type wrappedStream struct {
	Stream
	tagger StreamProfilerTagger
	labels []string
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	ctx := w.Context()
	profileLabel, err := w.tagger.TagRecvMsg(ctx)
	if err != nil {
		return err
	}
	var labels []string
	if profileLabel != nil {
		labels = profileLabel.toLabels()
	}
	// Merge labels from stream profiler labels filters.
	labels = append(w.labels, labels...)
	pprof.Do(ctx, pprof.Labels(labels...), func(ctx context.Context) {
		err = w.Stream.RecvMsg(m)
	})
	return err
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	ctx := w.Context()
	profileLabel, err := w.tagger.TagSendMsg(ctx, m)
	if err != nil {
		return err
	}
	var labels []string
	if profileLabel != nil {
		labels = profileLabel.toLabels()
	}
	// Merge labels from stream profiler labels filters.
	labels = append(w.labels, labels...)
	pprof.Do(ctx, pprof.Labels(labels...), func(ctx context.Context) {
		err = w.Stream.SendMsg(m)
	})
	return err
}
