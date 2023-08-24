// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build windows
// +build windows

package rollwriter

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"
	"time"
)

const (
	tmpTimeFormat = "tmp-20060102-150405.00000"
	bkTimeFormat  = "bk-20060102-150405.00000"
)

// doReopenFile reopens the file.
//
// On Windows, the given arguments no longer directly refer to concrete files.
// Instead, they are symbolic links to some well-formatted temporary files.
// After the new temporary file and new link are created, the old link will be
// removed. The old temporary file will be renamed exactly to the old link name.
//
// Generally, there are only two crucial cases involved:
//  1. This function is called because the log reaches the maximum size:
//     The arguments are like:
//     * `newLink` = "./trpc.log"
//     * `oldLink` = "./trpc.log.bk-20230117-180000.00127"
//     Then this function will create newLink which points to a new temporary file.
//     And rename the old temporary file as `oldLink` (asynchronously).
//  2. This function is called because the log file needs to be reopened regularly
//     to ensure that the underlying still exists:
//     Two sub-cases are involved:
//     2.1. `newLink` == `oldLink`:
//     In this case, we do not want to create a new temporary file (otherwise you
//     would get tons of temporary files). Instead, we reopen the existing temporary
//     file and do nothing with the link (because the link still refers to the same
//     temporary file).
//     2.2. `newLink` != `oldLink`:
//     In this case, the typical arguments are like:
//     * `newLink` = "./trpc.log.2023011718"
//     * `oldLink` = "./trpc.log.2023011717"
//     Explanation: this function is called because of rolling by time. The newLink
//     is present one hour later than the oldLink.
//     Under this circumstance, we need to create a new temporary file and link it
//     with the `newLink`. The old temporary file should be renamed as the `oldLink`
//     name (asynchronously). Actually the behavior is the same as case 1, but the
//     meaning and semantics of the arguments are different.
func (w *RollWriter) doReopenFile(newLink string, oldLink string) error {
	atomic.StoreInt64(&w.openTime, time.Now().Unix())
	if w.tryResume(newLink, oldLink) {
		return nil
	}
	// Case 2.1. `newLink` == `oldLink`:
	if newLink == oldLink {
		last := w.getCurrFile()
		if last == nil {
			return fmt.Errorf(
				"w.getCurrFile should not be nil when newLink == oldLink (newLink now is %s)",
				newLink)
		}
		f, err := w.os.OpenFile(last.Name(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return fmt.Errorf("os.OpenFile %s err: %w", last.Name(), err)
		}
		w.setCurrFile(f)
		w.delayCloseAndRenameFile(&closeAndRenameFile{file: last}) // rename = "", so no renaming will happen.
		return nil
	}

	// Case 1. and case 2.2. `newLink` != `oldLink`:

	// Example tmp string:
	// 1. roll by size = true/false, roll by time = false
	//    ./tmp-20230117-180000.00127.trpc.log
	// 2. roll by time = true and this function is called by reopenFile
	//    ./tmp-20230117-180000.00127.trpc.log.2023011717
	// Note: use base name of `newLink` as suffix instead of prefix to prevent
	// temporary files from being recognized as valid backup files.
	tmp := path.Join(w.currDir, time.Now().Format(tmpTimeFormat)+"."+filepath.Base(newLink))
	f, err := w.os.OpenFile(tmp, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("os.OpenFile %s err: %w", tmp, err)
	}
	w.removeLink(newLink)
	if err := os.Symlink(tmp, newLink); err != nil {
		return fmt.Errorf("os.Symlink %s to %s err: %w", tmp, newLink, err)
	}
	w.removeLink(oldLink)
	last := w.getCurrFile()
	if last != nil {
		w.delayCloseAndRenameFile(&closeAndRenameFile{file: last, rename: oldLink})
	}
	w.setCurrFile(f)
	return nil
}

func (w *RollWriter) tryResume(newLink, oldLink string) bool {
	// Check if there exists `trpc.log -> tmp.xxxx.log`.
	if oldLink != "" {
		return false
	}
	st, err := os.Lstat(newLink)
	if os.IsNotExist(err) { // The link `trpc.log` does not exist.
		return false
	}
	if !isSymlink(st.Mode()) { // `trpc.log` exists, but it is not a link.
		// Rename it to backup.
		// If the directory contains trpc.log, the log cannot be written correctly.
		// Because it is not possible to create a link with the same name.
		w.os.Rename(newLink, path.Join(w.currDir, time.Now().Format(bkTimeFormat)+"."+filepath.Base(newLink)))
		return false
	}

	// The following fixes the problem:
	// When the service stops, the tmp log is not processed. After restarting, a new tmp file is generated
	// and rolling continues, which can make it difficult to view the log properly.
	fileName, err := os.Readlink(newLink)
	if err != nil {
		fmt.Printf("os.Readlink %s err: %+v\n", newLink, err)
		return false
	}
	f, err := w.os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("w.os.OpenFile %s err: %+v\n", fileName, err)
		return false
	}
	w.setCurrFile(f)
	return true
}

// backupFile backs this file up and reopen a new one if file size is too large.
func (w *RollWriter) backupFile() {
	if !(w.opts.MaxSize > 0 && atomic.LoadInt64(&w.currSize) >= w.opts.MaxSize) {
		return
	}
	atomic.StoreInt64(&w.currSize, 0)
	backup := w.currPath + "." + time.Now().Format(backupTimeFormat)
	if err := w.doReopenFile(w.currPath, backup); err != nil {
		fmt.Printf("w.doReopenFile %s err: %+v\n", w.currPath, err)
	}
	w.notify()
}

func (w *RollWriter) removeLink(path string) {
	st, err := os.Lstat(path)
	if err != nil {
		return
	}
	if !isSymlink(st.Mode()) {
		return
	}
	if err := w.os.Remove(path); err != nil {
		fmt.Printf("os.Remove existing symlink %s err: %+v", path, err)
	}
}

func isSymlink(m os.FileMode) bool {
	return m&os.ModeSymlink != 0
}
