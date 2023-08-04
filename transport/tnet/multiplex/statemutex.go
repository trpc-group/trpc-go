//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex

import (
	"sync"
)

// stateRWMutex is similar to sync.RWMutex, but it has a closed state.
type stateRWMutex struct {
	mu       sync.RWMutex
	isClosed bool
}

// rLock locks rw for reading, returns false if mutex is closed.
func (rw *stateRWMutex) rLock() bool {
	rw.mu.RLock()
	if rw.isClosed {
		rw.mu.RUnlock()
		return false
	}
	return true
}

// rUnlock unlocks rw for reading.
func (rw *stateRWMutex) rUnlock() {
	rw.mu.RUnlock()
}

// lock locks rw for writing, returns false if mutex is closed.
func (rw *stateRWMutex) lock() bool {
	rw.mu.Lock()
	if rw.isClosed {
		rw.mu.Unlock()
		return false
	}
	return true
}

// unlock unlocks rw for writing.
func (rw *stateRWMutex) unlock() {
	rw.mu.Unlock()
}

// closeLocked closes the mutex. It should be called only after stateRWMutex.lock.
func (rw *stateRWMutex) closeLocked() {
	rw.isClosed = true
}
