package testdata

import (
	"path/filepath"
	"runtime"
)

// basePath is the root directory of this package.
var basePath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basePath = filepath.Dir(currentFile)
}

// BasePath returns the root directory of this package.
func BasePath() string {
	return basePath
}

// Path returns the absolute path the given relative file or directory path,
// relative to the trpc.group/trpc-go/trpc-go/test/testdata directory in the user's GOPATH.
// If relativePath is already absolute, it is returned unmodified.
func Path(relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return relativePath
	}

	return filepath.Join(basePath, relativePath)
}
