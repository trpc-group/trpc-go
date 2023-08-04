package rollwriter

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// functional test
// go test -v -cover
// benchmark test
// go test -bench=. -benchtime=20s -run=Bench

const (
	testTimes    = 100000
	testRoutines = 256
)

func TestRollWriter(t *testing.T) {

	// empty file name.
	t.Run("empty_log_name", func(t *testing.T) {
		logDir := t.TempDir()
		_, err := NewRollWriter("")
		assert.Error(t, err, "NewRollWriter: invalid log path")

		// print log file list.
		printLogFiles(logDir)
	})

	// no rolling.
	t.Run("roll_by_default", func(t *testing.T) {
		logDir := t.TempDir()
		logName := "test.log"
		w, err := NewRollWriter(filepath.Join(logDir, logName))
		assert.NoError(t, err, "NewRollWriter: create logger ok")
		log.SetOutput(w)
		for i := 0; i < testTimes; i++ {
			log.Printf("this is a test log: %d\n", i)
		}

		// check number of rolling log files(current files + backup files).
		time.Sleep(20 * time.Millisecond)
		logFiles := getLogBackups(logDir, logName)
		if len(logFiles) != 1 {
			t.Errorf("Number of log backup files should be 1")
		}
		require.Nil(t, w.Close())

		// print log file list.
		printLogFiles(logDir)
	})

	// roll by size.
	t.Run("roll_by_size", func(t *testing.T) {
		logDir := t.TempDir()
		logName := "test_size.log"
		const (
			maxBackup = 2
			maxSize   = 1
			maxAge    = 1
		)
		w, err := NewRollWriter(filepath.Join(logDir, logName),
			WithMaxSize(maxSize),
			WithMaxAge(maxAge),
			WithMaxBackups(maxBackup),
		)
		assert.NoError(t, err, "NewRollWriter: create logger ok")
		log.SetOutput(w)
		for i := 0; i < testTimes; i++ {
			log.Printf("this is a test log: %d\n", i)
		}

		w.notify()
		// check number of rolling log files.
		var logFiles []os.FileInfo
		require.Eventuallyf(t,
			func() bool {
				logFiles = getLogBackups(logDir, logName)
				return len(logFiles) == maxBackup+1
			},
			5*time.Second,
			time.Second,
			"Number of log files should be %d, current: %d, %+v",
			maxBackup+1, len(logFiles), func() []string {
				names := make([]string, 0, len(logFiles))
				for _, f := range logFiles {
					names = append(names, f.Name())
				}
				return names
			}(),
		)

		// check rolling log file size(allow to exceed a little).
		for _, file := range logFiles {
			if file.Size() > 1*1024*1024+1024 {
				t.Errorf("Log file size exceeds max_size")
			}
		}
		require.Nil(t, w.Close())

		// print log file list.
		printLogFiles(logDir)
	})

	// rolling by time.
	t.Run("roll_by_time", func(t *testing.T) {
		logDir := t.TempDir()
		logName := "test_time.log"
		const (
			maxBackup = 3
			maxSize   = 1
			maxAge    = 1
		)
		w, err := NewRollWriter(filepath.Join(logDir, logName),
			WithRotationTime(".%Y%m%d"),
			WithMaxSize(maxSize),
			WithMaxAge(maxAge),
			WithMaxBackups(maxBackup),
			WithCompress(true),
		)
		assert.NoError(t, err, "NewRollWriter: create logger ok")
		log.SetOutput(w)
		for i := 0; i < testTimes; i++ {
			log.Printf("this is a test log: %d\n", i)
		}

		w.notify()
		// check number of rolling log files.
		var logFiles []os.FileInfo
		require.Eventuallyf(t,
			func() bool {
				logFiles = getLogBackups(logDir, logName)
				return len(logFiles) == maxBackup+1
			},
			5*time.Second,
			time.Second,
			"Number of log files should be %d, current: %d, %+v",
			maxBackup+1, len(logFiles), func() []string {
				names := make([]string, 0, len(logFiles))
				for _, f := range logFiles {
					names = append(names, f.Name())
				}
				return names
			}(),
		)

		// check rolling log file size(allow to exceed a little).
		for _, file := range logFiles {
			if file.Size() > 1*1024*1024+1024 {
				t.Errorf("Log file size exceeds max_size")
			}
		}

		// check number of compressed files.
		compressFileNum := 0
		for _, file := range logFiles {
			if strings.HasSuffix(file.Name(), compressSuffix) {
				compressFileNum++
			}
		}
		if compressFileNum != 3 {
			t.Errorf("Number of compress log files should be 3")
		}
		require.Nil(t, w.Close())

		// print log file list.
		printLogFiles(logDir)
	})
}

func TestAsyncRollWriter(t *testing.T) {
	logDir := t.TempDir()
	const flushThreshold = 4 * 1024

	// no rolling(asynchronous mod)
	t.Run("roll_by_default_async", func(t *testing.T) {
		logName := "test.log"
		w, err := NewRollWriter(filepath.Join(logDir, logName))
		assert.NoError(t, err, "NewRollWriter: create logger ok")

		asyncWriter := NewAsyncRollWriter(w, WithLogQueueSize(10), WithWriteLogSize(1024),
			WithWriteLogInterval(100), WithDropLog(true))
		log.SetOutput(asyncWriter)
		for i := 0; i < testTimes; i++ {
			log.Printf("this is a test log: %d\n", i)
		}

		// check number of rolling log files.
		time.Sleep(20 * time.Millisecond)
		logFiles := getLogBackups(logDir, logName)
		if len(logFiles) != 1 {
			t.Errorf("Number of log backup files should be 1")
		}
		require.Nil(t, asyncWriter.Close())
	})

	// rolling by size(asynchronous mod)
	t.Run("roll_by_size_async", func(t *testing.T) {
		logName := "test_size.log"
		w, err := NewRollWriter(filepath.Join(logDir, logName),
			WithMaxSize(1),
			WithMaxAge(1),
		)
		assert.NoError(t, err, "NewRollWriter: create logger ok")

		asyncWriter := NewAsyncRollWriter(w, WithWriteLogSize(flushThreshold))
		log.SetOutput(asyncWriter)
		for i := 0; i < testTimes; i++ {
			log.Printf("this is a test log: %d\n", i)
		}

		// check number of rolling log files.
		time.Sleep(200 * time.Millisecond)
		logFiles := getLogBackups(logDir, logName)
		if len(logFiles) != 5 {
			t.Errorf("Number of log backup files should be 5")
		}

		// check rolling log file size(asynchronous mod, may exceed 4K at most)
		for _, file := range logFiles {
			if file.Size() > 1*1024*1024+flushThreshold*2 {
				t.Errorf("Log file size exceeds max_size")
			}
		}
		require.Nil(t, asyncWriter.Close())
	})

	// rolling by time(asynchronous mod)
	t.Run("roll_by_time_async", func(t *testing.T) {
		logName := "test_time.log"
		w, err := NewRollWriter(filepath.Join(logDir, logName),
			WithRotationTime(".%Y%m%d"),
			WithMaxSize(1),
			WithMaxAge(1),
			WithCompress(true),
		)
		assert.NoError(t, err, "NewRollWriter: create logger ok")

		asyncWriter := NewAsyncRollWriter(w, WithWriteLogSize(flushThreshold))
		log.SetOutput(asyncWriter)
		for i := 0; i < testTimes; i++ {
			log.Printf("this is a test log: %d\n", i)
		}
		require.Nil(t, asyncWriter.Sync())

		// check number of rolling log files.
		time.Sleep(200 * time.Millisecond)
		logFiles := getLogBackups(logDir, logName)
		if len(logFiles) != 5 {
			t.Errorf("Number of log files should be 5, current: %d", len(logFiles))
		}

		// check rolling log file size(asynchronous, may exceed 4K at most)
		for _, file := range logFiles {
			if file.Size() > 1*1024*1024+flushThreshold*2 {
				t.Errorf("Log file size exceeds max_size")
			}
		}

		// number of compressed files.
		compressFileNum := 0
		for _, file := range logFiles {
			if strings.HasSuffix(file.Name(), compressSuffix) {
				compressFileNum++
			}
		}
		if compressFileNum != 4 {
			t.Errorf("Number of compress log files should be 4")
		}
		require.Nil(t, asyncWriter.Close())
	})

	// wait 1 second.
	time.Sleep(1 * time.Second)

	// print log file list.
	printLogFiles(logDir)
}

func TestRollWriterRace(t *testing.T) {
	logDir := t.TempDir()

	writer, err := NewRollWriter(
		filepath.Join(logDir, "test.log"),
		WithRotationTime(".%Y%m%d"),
	)
	require.Nil(t, err)
	writer.opts.MaxSize = 1

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			writer.Write([]byte(fmt.Sprintf("this is a test log: 1-%d\n", i)))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			writer.Write([]byte(fmt.Sprintf("this is a test log: 2-%d\n", i)))
		}
	}()
	wg.Wait()

	time.Sleep(time.Second) // Wait till all the files are closed.
	require.Nil(t, writer.Close())
}

func TestAsyncRollWriterRace(t *testing.T) {
	logDir := t.TempDir()

	writer, _ := NewRollWriter(
		filepath.Join(logDir, "test.log"),
		WithRotationTime(".%Y%m%d"),
	)
	writer.opts.MaxSize = 1
	w := NewAsyncRollWriter(writer)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			w.Write([]byte(fmt.Sprintf("this is a test log: 1-%d\n", i)))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			w.Write([]byte(fmt.Sprintf("this is a test log: 2-%d\n", i)))
		}
	}()
	wg.Wait()

	time.Sleep(20 * time.Millisecond)
	require.Nil(t, w.Close())
}

func TestAsyncRollWriterSyncTwice(t *testing.T) {
	w := NewAsyncRollWriter(&noopWriteCloser{})
	w.Write([]byte("hello"))
	require.Nil(t, w.Sync())
	require.Nil(t, w.Sync())
	require.Nil(t, w.Close())
}

func TestRollWriterError(t *testing.T) {
	logDir := t.TempDir()
	t.Run("reopen file", func(t *testing.T) {
		r, err := NewRollWriter(path.Join(logDir, "trpc.log"))
		require.Nil(t, err)
		r.os = errOS{openFileErr: errAlwaysFail}
		r.reopenFile()
		require.Nil(t, r.Close())
	})
	t.Run("delay close and rename file", func(t *testing.T) {
		r, err := NewRollWriter(path.Join(logDir, "trpc.log"))
		require.Nil(t, err)
		r.os = errOS{renameErr: errAlwaysFail}
		f, err := os.CreateTemp(logDir, "trpc.log")
		require.Nil(t, err)
		r.delayCloseAndRenameFile(&closeAndRenameFile{file: f, rename: path.Join(logDir, "tmp.log")})
		time.Sleep(30 * time.Millisecond)
		require.Nil(t, r.Close())
	})
	t.Run("match log file", func(t *testing.T) {
		r, err := NewRollWriter(path.Join(logDir, "trpc.log"))
		require.Nil(t, err)
		r.os = errOS{statErr: errAlwaysFail}
		_, err = r.matchLogFile("trpc.log.20230130", "trpc.log")
		require.NotNil(t, err)
		require.Nil(t, r.Close())
	})
	t.Run("remove files", func(t *testing.T) {
		r, err := NewRollWriter(path.Join(logDir, "trpc.log"))
		require.Nil(t, err)
		r.os = errOS{removeErr: errAlwaysFail}
		r.removeFiles([]logInfo{{time.Time{}, &noopFileInfo{}}})
		require.Nil(t, r.Close())
	})
	t.Run("compress file", func(t *testing.T) {
		file := path.Join(logDir, "trpc.log")
		r, err := NewRollWriter(file)
		require.Nil(t, err)
		r.os = errOS{openErr: errAlwaysFail}
		require.NotNil(t, r.compressFile(file, ""))
		r.os = errOS{openFileErr: errAlwaysFail}
		f, err := os.Create(file)
		require.Nil(t, err)
		require.Nil(t, f.Close())
		require.NotNil(t, r.compressFile(file, ""))
		require.Nil(t, r.Close())
	})
}

type noopFileInfo struct{}

func (*noopFileInfo) Name() string {
	return "trpc.log"
}

func (*noopFileInfo) IsDir() bool {
	return false
}

func (*noopFileInfo) Type() os.FileMode {
	return fs.ModeAppend
}

func (*noopFileInfo) Info() (os.FileInfo, error) {
	return &noopFileInfo{}, nil
}

func (*noopFileInfo) Sys() interface{} {
	return nil
}

func (*noopFileInfo) Size() int64 {
	return 0
}

func (*noopFileInfo) Mode() os.FileMode {
	return os.ModePerm
}

func (*noopFileInfo) ModTime() time.Time {
	return time.Time{}
}

var errAlwaysFail = errors.New("always fail")

type errOS struct {
	openErr     error
	openFileErr error
	mkdirAllErr error
	renameErr   error
	statErr     error
	removeErr   error
}

func (o errOS) Open(name string) (*os.File, error) {
	if o.openErr != nil {
		return nil, o.openErr
	}
	return os.Open(name)
}

func (o errOS) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	if o.openFileErr != nil {
		return nil, o.openFileErr
	}
	return os.OpenFile(name, flag, perm)
}

func (o errOS) MkdirAll(path string, perm fs.FileMode) error {
	if o.mkdirAllErr != nil {
		return o.mkdirAllErr
	}
	return os.MkdirAll(path, perm)
}

func (o errOS) Rename(oldpath string, newpath string) error {
	if o.renameErr != nil {
		return o.renameErr
	}
	return os.Rename(oldpath, newpath)
}

func (o errOS) Stat(name string) (fs.FileInfo, error) {
	if o.statErr != nil {
		return nil, o.statErr
	}
	return os.Stat(name)
}

func (o errOS) Remove(name string) error {
	if o.removeErr != nil {
		return o.removeErr
	}
	return os.Remove(name)
}

type noopWriteCloser struct{}

func (*noopWriteCloser) Write(p []byte) (n int, err error) { return }

func (*noopWriteCloser) Close() (err error) { return }

// BenchmarkRollWriterBySize benchmarks RollWriter by size.
func BenchmarkRollWriterBySize(b *testing.B) {
	logDir := b.TempDir()

	// init RollWriter.
	writer, _ := NewRollWriter(filepath.Join(logDir, "test.log"))
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(writer),
		zapcore.DebugLevel,
	)
	logger := zap.New(
		core,
		zap.AddCaller(),
	)

	// warm up.
	for i := 0; i < testTimes; i++ {
		logger.Debug(fmt.Sprint("this is a test log: ", i))
	}

	b.SetParallelism(testRoutines / runtime.GOMAXPROCS(0))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Debug("this is a test log")
		}
	})
}

// BenchmarkRollWriterByTime benchmarks RollWriter by time.
func BenchmarkRollWriterByTime(b *testing.B) {
	logDir := b.TempDir()

	// init RollWriter.
	writer, _ := NewRollWriter(
		filepath.Join(logDir, "test.log"),
		WithRotationTime(".%Y%m%d"),
	)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(writer),
		zapcore.DebugLevel,
	)
	logger := zap.New(
		core,
		zap.AddCaller(),
	)

	// warm up.
	for i := 0; i < testTimes; i++ {
		logger.Debug(fmt.Sprint("this is a test log: ", i))
	}

	b.SetParallelism(testRoutines / runtime.GOMAXPROCS(0))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Debug("this is a test log")
		}
	})
}

// BenchmarkAsyncRollWriterBySize benchmarks asynchronous RollWriter.
func BenchmarkAsyncRollWriterBySize(b *testing.B) {
	logDir := b.TempDir()

	// init RollWriter.
	writer, _ := NewRollWriter(
		filepath.Join(logDir, "test.log"),
	)
	asyncWriter := NewAsyncRollWriter(writer)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		asyncWriter,
		zapcore.DebugLevel,
	)
	logger := zap.New(
		core,
		zap.AddCaller(),
	)

	// warm up.
	for i := 0; i < testTimes; i++ {
		logger.Debug(fmt.Sprint("this is a test log: ", i))
	}

	b.SetParallelism(testRoutines / runtime.GOMAXPROCS(0))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Debug("this is a test log")
		}
	})
}

// BenchmarkAsyncRollWriterByTime benchmarks asynchronous RollWriter by time.
func BenchmarkAsyncRollWriterByTime(b *testing.B) {
	logDir := b.TempDir()

	// init RollWriter.
	writer, _ := NewRollWriter(
		filepath.Join(logDir, "test.log"),
		WithRotationTime(".%Y%m%d"),
	)
	asyncWriter := NewAsyncRollWriter(writer)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		asyncWriter,
		zapcore.DebugLevel,
	)
	logger := zap.New(
		core,
		zap.AddCaller(),
	)

	// warm up.
	for i := 0; i < testTimes; i++ {
		logger.Debug(fmt.Sprint("this is a test log: ", i))
	}

	b.SetParallelism(testRoutines / runtime.GOMAXPROCS(0))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Debug("this is a test log")
		}
	})
}

func printLogFiles(logDir string) {
	fmt.Println("================================================")
	fmt.Printf("[%s]:\n", logDir)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Println("get entries failed ", err)
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println("get info failed ", err)
			continue
		}
		fmt.Println("\t", info.Name(), info.Size())
	}
}

func execCommand(name string, args ...string) string {
	fmt.Println(name, args)
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("exec command failed, err: %v\n", err)
	}
	fmt.Println(string(output))

	return string(output)
}

func getLogBackups(logDir, prefix string) []os.FileInfo {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil
	}

	var logFiles []os.FileInfo
	for _, file := range entries {
		if !strings.HasPrefix(file.Name(), prefix) {
			continue
		}
		if info, err := file.Info(); err == nil {
			logFiles = append(logFiles, info)
		}
	}
	return logFiles
}
