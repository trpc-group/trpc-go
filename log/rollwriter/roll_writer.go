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

// Package rollwriter provides a high performance rolling file log.
// Package rollwriter does not print logs, but implements io.Writer.
// It can coordinate with any logs which depends on io.Writer, such as golang standard log.
// Main features:
//  1. support rolling logs by file size.
//  2. support rolling logs by datetime.
//  3. support scavenging expired or useless logs.
//  4. support compressing logs.
package rollwriter

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lestrrat-go/strftime"

	"trpc.group/trpc-go/trpc-go/log/internal/timeunit"
)

const (
	backupTimeFormat = "bk-20060102-150405.00000"
	compressSuffix   = ".gz"
)

// ensure we always implement io.WriteCloser.
var _ io.WriteCloser = (*RollWriter)(nil)

// RollWriter is a file log writer which support rolling by size or datetime.
// It implements io.WriteCloser.
type RollWriter struct {
	filePath string
	opts     *Options

	pattern       *strftime.Strftime
	currDir       string
	currPath      string
	currSize      int64
	currFile      atomic.Value
	openTime      int64
	closed        uint32
	filenameRegex *regexp.Regexp

	mu         sync.Mutex
	notifyOnce sync.Once
	notifyCh   chan bool
	closeOnce  sync.Once
	closeCh    chan *closeAndRenameFile

	os customizedOS
}

// NewRollWriter creates a new RollWriter.
func NewRollWriter(filePath string, opt ...Option) (*RollWriter, error) {
	opts := &Options{
		MaxSize:    0,     // default no rolling by file size
		MaxAge:     0,     // default no scavenging on expired logs
		MaxBackups: 0,     // default no scavenging on redundant logs
		Compress:   false, // default no compressing
	}

	// opt has the highest priority and should overwrite the original one.
	for _, o := range opt {
		o(opts)
	}

	if filePath == "" {
		return nil, errors.New("empty file path is invalid")
	}

	logFilePath := filePath

	hasTimeFormatTag := timeunit.ContainsTimeFormatTag(logFilePath)
	// Validate the filename and roll type configuration. timeFormat must not be empty when roll_type is set to time.
	if hasTimeFormatTag && opts.TimeFormat == "" {
		// If the filename contains a time format tag without using time-based rolling, return an error.
		return nil, fmt.Errorf("invalid filename '%s': cannot use time format tag without RollByTime", logFilePath)
	}

	// If a time format is specified, append the time format tag to the log file name.
	// default filename trpc.log.%Y%m%d%H%M, so set logFilePath to "trpc.log.{time_format}"
	if !hasTimeFormatTag && opts.TimeFormat != "" {
		logFilePath = filePath + "." + timeunit.TimeFormatTag
	}

	// Generate a regex pattern to match filenames based on the updated file path and specified time format.
	filenameRegex, err := timeunit.GenerateTimeFormatRegex(filepath.Base(logFilePath), opts.TimeFormat)
	if err != nil {
		return nil, err
	}

	// Update the file name with the specified time format.
	updatedFilePath := timeunit.UpdateFileNameWithTimeFormat(logFilePath, opts.TimeFormat)

	pattern, err := strftime.New(updatedFilePath)
	if err != nil {
		return nil, fmt.Errorf("creating Strftime object: %w, invalid time pattern: %s", err, logFilePath)
	}

	w := &RollWriter{
		filePath:      filePath,
		opts:          opts,
		pattern:       pattern,
		currDir:       filepath.Dir(filePath),
		os:            defaultCustomizedOS,
		filenameRegex: filenameRegex,
	}

	if err = w.os.MkdirAll(w.currDir, 0755); err != nil {
		return nil, err
	}

	return w, nil
}

// Write writes logs. It implements io.Writer.
func (w *RollWriter) Write(v []byte) (n int, err error) {
	if atomic.LoadUint32(&w.closed) == 1 {
		return 0, errors.New("roll writer has been closed")
	}

	// reopen file every 10 seconds.
	if w.getCurrFile() == nil || time.Now().Unix()-atomic.LoadInt64(&w.openTime) > 10 {
		w.mu.Lock()
		w.reopenFile()
		w.mu.Unlock()
	}

	// return when failed to open the file.
	if w.getCurrFile() == nil {
		return 0, errors.New("open file fail")
	}

	// write logs to file.
	n, err = w.getCurrFile().Write(v)
	atomic.AddInt64(&w.currSize, int64(n))

	// rolling on full
	if w.opts.MaxSize > 0 && atomic.LoadInt64(&w.currSize) >= w.opts.MaxSize {
		w.mu.Lock()
		w.backupFile()
		w.mu.Unlock()
	}
	return n, err
}

// Close closes the current log file. It implements io.Closer.
func (w *RollWriter) Close() error {
	if !atomic.CompareAndSwapUint32(&w.closed, 0, 1) {
		return errors.New("closing closed roll writer")
	}
	if w.getCurrFile() == nil {
		return nil
	}
	err := w.getCurrFile().Close()
	w.setCurrFile(nil)

	if w.notifyCh != nil {
		close(w.notifyCh)
		w.notifyCh = nil
	}

	if w.closeCh != nil {
		close(w.closeCh)
		w.closeCh = nil
	}

	return err
}

// getCurrFile returns the current log file.
func (w *RollWriter) getCurrFile() *os.File {
	if file, ok := w.currFile.Load().(*os.File); ok {
		return file
	}
	return nil
}

// setCurrFile sets the current log file.
func (w *RollWriter) setCurrFile(file *os.File) {
	w.currFile.Store(file)
}

// reopenFile reopen the file regularly. It notifies the scavenger if file path has changed.
func (w *RollWriter) reopenFile() {
	if w.getCurrFile() == nil || time.Now().Unix()-atomic.LoadInt64(&w.openTime) > 10 {
		atomic.StoreInt64(&w.openTime, time.Now().Unix())
		oldPath := w.currPath
		currPath := w.pattern.FormatString(time.Now())
		if w.currPath != currPath {
			w.currPath = currPath
			w.notify()
		}
		if err := w.doReopenFile(w.currPath, oldPath); err != nil {
			fmt.Printf("w.doReopenFile %s err: %+v\n", w.currPath, err)
		}
	}
}

// notify runs scavengers.
func (w *RollWriter) notify() {
	w.notifyOnce.Do(func() {
		w.notifyCh = make(chan bool, 1)
		go w.runCleanFiles()
	})
	select {
	case w.notifyCh <- true:
	default:
	}
}

// runCleanFiles cleans redundant or expired (compressed) logs in a new goroutine.
func (w *RollWriter) runCleanFiles() {
	for range w.notifyCh {
		if w.opts.MaxBackups == 0 && w.opts.MaxAge == 0 && !w.opts.Compress {
			continue
		}
		w.cleanFiles()
	}
}

// delayCloseAndRenameFile delay closing and renaming the given file.
func (w *RollWriter) delayCloseAndRenameFile(f *closeAndRenameFile) {
	w.closeOnce.Do(func() {
		w.closeCh = make(chan *closeAndRenameFile, 100)
		go w.runCloseFiles()
	})
	w.closeCh <- f
}

// runCloseFiles delay closing file in a new goroutine.
func (w *RollWriter) runCloseFiles() {
	for f := range w.closeCh {
		// delay 20ms
		time.Sleep(20 * time.Millisecond)
		if err := f.file.Close(); err != nil {
			fmt.Printf("f.file.Close err: %+v, filename: %s\n", err, f.file.Name())
		}
		if f.rename == "" || f.file.Name() == f.rename {
			continue
		}
		if err := w.os.Rename(f.file.Name(), f.rename); err != nil {
			fmt.Printf("os.Rename from %s to %s err: %+v\n", f.file.Name(), f.rename, err)
		}
		w.notify()
	}
}

// cleanFiles cleans redundant or expired (compressed) logs.
func (w *RollWriter) cleanFiles() {
	// get the file list of current log.
	files, err := w.getOldLogFiles()
	if err != nil {
		fmt.Printf("w.getOldLogFiles err: %+v\n", err)
		return
	}
	if len(files) == 0 {
		return
	}

	files, redundantInfos := partitionByMaxBackups(files, w.opts.MaxBackups)
	files, expiredInfos := partitionByMaxAge(files, w.opts.MaxAge)
	w.removeFiles(append(redundantInfos, expiredInfos...))

	if w.opts.Compress {
		_, uncompressedFiles := partitionByCompressExt(files, compressSuffix)
		w.compressFiles(uncompressedFiles)
	}
}

// getOldLogFiles returns the log file list ordered by modified time.
func (w *RollWriter) getOldLogFiles() ([]logInfo, error) {
	entries, err := os.ReadDir(w.currDir)
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory %s: %w", w.currDir, err)
	}

	var logFiles []logInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		if modTime, err := w.matchLogFile(e.Name()); err == nil {
			logFiles = append(logFiles, logInfo{modTime, e})
		}
	}
	sort.Sort(byFormatTime(logFiles))
	return logFiles, nil
}

// matchLogFile checks whether current log file matches all relative log files, if matched, returns
// the modified time.
func (w *RollWriter) matchLogFile(logFilename string) (time.Time, error) {
	// exclude current log file.
	// a.log
	// a.log.20200712
	if filepath.Base(w.currPath) == logFilename {
		return time.Time{}, errors.New("ignore current logfile")
	}

	// match all log files with current log file.
	// match customized log file format.
	// a.log -> a.log.20200712-1232/a.log.20200712-1232.gz
	// a.log.20200712 -> a.log.20200712.20200712-1232/a.log.20200712.20200712-1232.gz
	// a_\\{d8}.log -> a_\\{d8}.log.20200712-1232/a_\\{d8}.log.20200712-1232.gz
	isMatch := w.filenameRegex.MatchString(logFilename)
	if !isMatch {
		return time.Time{}, errors.New("mismatched prefix")
	}

	st, err := w.os.Stat(filepath.Join(w.currDir, logFilename))
	if err != nil {
		return time.Time{}, fmt.Errorf("file stat fail: %w", err)
	}
	return st.ModTime(), nil
}

// removeFiles deletes expired or redundant log files.
func (w *RollWriter) removeFiles(remove []logInfo) {
	// clean expired or redundant files.
	for _, f := range remove {
		file := filepath.Join(w.currDir, f.Name())
		if err := w.os.Remove(file); err != nil {
			fmt.Printf("remove file %s err: %+v\n", file, err)
		}
	}
}

// compressFiles compresses demanded log files.
func (w *RollWriter) compressFiles(compress []logInfo) {
	// compress log files.
	for _, f := range compress {
		fn := filepath.Join(w.currDir, f.Name())
		w.compressFile(fn, fn+compressSuffix)
	}
}

func partitionByMaxBackups(files []logInfo, maxBackups int) (necessary, redundant []logInfo) {
	if maxBackups == 0 || len(files) < maxBackups {
		return files, nil
	}

	preserved := make(map[string]struct{})
	return partition(files, func(f logInfo) bool {
		fn := strings.TrimSuffix(f.Name(), compressSuffix)
		preserved[fn] = struct{}{}
		return len(preserved) <= maxBackups
	})
}

func partitionByMaxAge(files []logInfo, maxAge int) (valid, expired []logInfo) {
	if maxAge <= 0 {
		return files, nil
	}

	diff := time.Duration(int64(24*time.Hour) * int64(maxAge))
	cutoff := time.Now().Add(-1 * diff)
	return partition(files, func(f logInfo) bool {
		return !f.timestamp.Before(cutoff)
	})
}

func partitionByCompressExt(files []logInfo, compressExt string) (incompressible, compressible []logInfo) {
	return partition(files, func(info logInfo) bool {
		return strings.HasSuffix(info.Name(), compressExt)
	})
}

// partition partitions the infos into two parts, such that the infos satisfying match are in the matching,
// and the elements not satisfying match are in the nonMatching.
func partition(infos []logInfo, match func(logInfo) bool) (matching, nonMatching []logInfo) {
	for _, info := range infos {
		if match(info) {
			matching = append(matching, info)
		} else {
			nonMatching = append(nonMatching, info)
		}
	}
	return
}

// compressFile compresses file src to dst, and removes src on success.
func (w *RollWriter) compressFile(src, dst string) (err error) {
	f, err := w.os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}

	gzf, err := w.os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to open compressed file: %v", err)
	}

	gz := gzip.NewWriter(gzf)
	defer func() {
		gz.Close()
		// Make sure files are closed before removing, or else the removal
		// will fail on Windows.
		f.Close()
		gzf.Close()
		if err != nil {
			w.os.Remove(dst)
			err = fmt.Errorf("failed to compress file: %v", err)
			return
		}
		w.os.Remove(src)
	}()

	if _, err := io.Copy(gz, f); err != nil {
		return err
	}
	return nil
}

type closeAndRenameFile struct {
	file   *os.File
	rename string
}

// logInfo is an assistant struct which is used to return file name and last modified time.
type logInfo struct {
	timestamp time.Time
	os.DirEntry
}

// byFormatTime sorts by time descending order.
type byFormatTime []logInfo

// Less checks whether the time of b[j] is early than the time of b[i].
func (b byFormatTime) Less(i, j int) bool {
	return b[i].timestamp.After(b[j].timestamp)
}

// Swap swaps b[i] and b[j].
func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// Len returns the length of list b.
func (b byFormatTime) Len() int {
	return len(b)
}

var defaultCustomizedOS = stdOS{}

type stdOS struct{}

func (stdOS) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (stdOS) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (stdOS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (stdOS) Rename(oldpath string, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (stdOS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (stdOS) Remove(name string) error {
	return os.Remove(name)
}

type customizedOS interface {
	Open(name string) (*os.File, error)
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	MkdirAll(path string, perm fs.FileMode) error
	Rename(oldpath string, newpath string) error
	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
}
