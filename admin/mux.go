package admin

import (
	"errors"
	"net/http"
	"reflect"
	"sync"
	"unsafe"
)

// unregisterHandlers deletes router from http.DefaultServeMux.
// The import of "net/http/pprof" will automatically register pprof related routes on
// http.DefaultServeMux, which may cause security problems.
// Refer to：https://github.com/golang/go/issues/22085
func unregisterHandlers(patterns []string) error {
	// Need to import muxEntry in net/http pkg.
	type muxEntry struct {
		h       http.Handler
		pattern string
	}

	v := reflect.ValueOf(http.DefaultServeMux)

	// Get lock.
	muField := v.Elem().FieldByName("mu")
	if !muField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `mu`")
	}
	muPointer := unsafe.Pointer(muField.UnsafeAddr())
	mu := (*sync.RWMutex)(muPointer)
	(*mu).Lock()
	defer (*mu).Unlock()

	// Delete value of map.
	mField := v.Elem().FieldByName("m")
	if !mField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `m`")
	}
	mPointer := unsafe.Pointer(mField.UnsafeAddr())
	m := (*map[string]muxEntry)(mPointer)
	for _, pattern := range patterns {
		delete(*m, pattern)
	}

	// Delete value of muxEntry slice.
	esField := v.Elem().FieldByName("es")
	if !esField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `es`")
	}
	esPointer := unsafe.Pointer(esField.UnsafeAddr())
	es := (*[]muxEntry)(esPointer)
	for _, pattern := range patterns {
		// Removes muxEntry of the same pattern.
		var j int
		for _, muxEntry := range *es {
			if muxEntry.pattern != pattern {
				(*es)[j] = muxEntry
				j++
			}
		}
		*es = (*es)[:j]
	}

	// Modify hosts.
	hostsField := v.Elem().FieldByName("hosts")
	if !hostsField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `hosts`")
	}
	hostsPointer := unsafe.Pointer(hostsField.UnsafeAddr())
	hosts := (*bool)(hostsPointer)
	*hosts = false
	for _, v := range *m {
		if v.pattern[0] != '/' {
			*hosts = true
		}
	}

	return nil
}
