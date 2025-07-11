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
	"sync"

	"github.com/fsnotify/fsnotify"

	"trpc.group/trpc-go/trpc-go/log"
)

func init() {
	RegisterProvider(newFileProvider())
}

func newFileProvider() *FileProvider {
	fp := &FileProvider{
		cb:              make(chan ProviderCallback),
		disabledWatcher: true,
		cache:           make(map[string]string),
		modTime:         make(map[string]int64),
	}
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		fp.disabledWatcher = false
		fp.watcher = watcher
		go fp.run()
		return fp
	}
	log.Debugf("fsnotify.NewWatcher err: %+v", err)
	return fp
}

// FileProvider is a config provider which gets config from file system.
type FileProvider struct {
	disabledWatcher bool
	watcher         *fsnotify.Watcher
	cb              chan ProviderCallback
	cache           map[string]string
	modTime         map[string]int64
	mu              sync.RWMutex
}

// Name returns file provider's name.
func (*FileProvider) Name() string {
	return "file"
}

// Read reads the specific path file, returns
// it content as bytes.
func (fp *FileProvider) Read(path string) ([]byte, error) {
	if !fp.disabledWatcher {
		if err := fp.watcher.Add(filepath.Dir(path)); err != nil {
			return nil, err
		}
		fp.mu.Lock()
		fp.cache[filepath.Clean(path)] = path
		fp.mu.Unlock()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Tracef("Failed to read file %v", err)
		return nil, err
	}
	return data, nil
}

// Watch watches config changing. The change will
// be handled by callback function.
func (fp *FileProvider) Watch(cb ProviderCallback) {
	if !fp.disabledWatcher {
		fp.cb <- cb
	}
}

func (fp *FileProvider) run() {
	fn := make([]ProviderCallback, 0)
	for {
		select {
		case i := <-fp.cb:
			fn = append(fn, i)
		case e := <-fp.watcher.Events:
			if t, ok := fp.isModified(e); ok {
				fp.trigger(e, t, fn)
			}
		}
	}
}

func (fp *FileProvider) isModified(e fsnotify.Event) (int64, bool) {
	if e.Op&fsnotify.Write != fsnotify.Write {
		return 0, false
	}
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	if _, ok := fp.cache[filepath.Clean(e.Name)]; !ok {
		return 0, false
	}
	fi, err := os.Stat(e.Name)
	if err != nil {
		return 0, false
	}
	if fi.ModTime().Unix() > fp.modTime[e.Name] {
		return fi.ModTime().Unix(), true
	}
	return 0, false
}

func (fp *FileProvider) trigger(e fsnotify.Event, t int64, fn []ProviderCallback) {
	data, err := os.ReadFile(e.Name)
	if err != nil {
		return
	}
	fp.mu.Lock()
	path := fp.cache[filepath.Clean(e.Name)]
	fp.modTime[e.Name] = t
	fp.mu.Unlock()
	for _, f := range fn {
		go f(path, data)
	}
}
