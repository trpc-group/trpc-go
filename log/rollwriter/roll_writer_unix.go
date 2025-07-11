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

//go:build !windows
// +build !windows

package rollwriter

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// doReopenFile reopens the file.
func (w *RollWriter) doReopenFile(path, _ string) error {
	atomic.StoreInt64(&w.openTime, time.Now().Unix())
	f, err := w.os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("os.OpenFile %s err: %w", path, err)
	}
	last := w.getCurrFile()
	w.setCurrFile(f)
	if last != nil {
		w.delayCloseAndRenameFile(&closeAndRenameFile{file: last})
	}
	st, err := w.os.Stat(path)
	if err != nil {
		return fmt.Errorf("os.Stat %s err: %w", path, err)
	}
	atomic.StoreInt64(&w.currSize, st.Size())
	return nil
}

// backupFile backs this file up and reopen a new one if file size is too large.
func (w *RollWriter) backupFile() {
	if !(w.opts.MaxSize > 0 && atomic.LoadInt64(&w.currSize) >= w.opts.MaxSize) {
		return
	}
	atomic.StoreInt64(&w.currSize, 0)

	// Rename the old file.
	backup := w.currPath + "." + time.Now().Format(backupTimeFormat)
	if _, err := w.os.Stat(w.currPath); !os.IsNotExist(err) {
		if err := w.os.Rename(w.currPath, backup); err != nil {
			fmt.Printf("os.Rename from %s to backup %s err: %+v\n", w.currPath, backup, err)
		}
	}

	// Reopen a new one.
	if err := w.doReopenFile(w.currPath, ""); err != nil {
		fmt.Printf("w.doReopenFile %s err: %+v\n", w.currPath, err)
	}
	w.notify()
}
