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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestProvider(t *testing.T) {
	p := newFileProvider()

	// read
	buf, err := p.Read("../testdata/trpc_go.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	// watch
	cb := func(path string, data []byte) {}
	p.Watch(cb)
	os.WriteFile("../testdata/trpc_go.yaml", buf, 664)

	p.disabledWatcher = true
	p.watcher.Close()
	_, err = p.Read("../testdata/trpc_go.yaml1")
	assert.NotNil(t, err)
}

func TestIsModified(t *testing.T) {
	filename := "../testdata/trpc_go.yaml3"
	p := newFileProvider()
	got, ok := p.isModified(fsnotify.Event{Name: filename})
	assert.Zero(t, got)
	assert.False(t, ok)

	got, ok = p.isModified(fsnotify.Event{Op: fsnotify.Write, Name: filename})
	assert.Zero(t, got)
	assert.False(t, ok)

	p.cache[filepath.Clean(filename)] = filename
	got, ok = p.isModified(fsnotify.Event{Op: fsnotify.Write, Name: filename})
	assert.Zero(t, got)
	assert.False(t, ok)

	os.WriteFile(filename, []byte("test"), 664)
	defer os.Remove(filename)
	got, ok = p.isModified(fsnotify.Event{Op: fsnotify.Write, Name: filename})
	assert.NotZero(t, got)
	assert.True(t, ok)

	p.modTime[filename] = got + 10000
	got, ok = p.isModified(fsnotify.Event{Op: fsnotify.Write, Name: filename})
	assert.Zero(t, got)
	assert.False(t, ok)
}
