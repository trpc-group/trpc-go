package healthcheck

var watchers = make(map[string][]func(Status))

// Watch registers a service status watcher.
// NOTE: No lock is used in this function, so it is not concurrency safe.
func Watch(serviceName string, onStatusChanged func(Status)) {
	watchers[serviceName] = append(watchers[serviceName], onStatusChanged)
}

// GetWatchers returns all registered watchers.
// NOTE: No lock is used in this function, so it is not concurrency safe.
// NOTE: The result is read-only, DO NOT modify.
func GetWatchers() map[string][]func(Status) {
	return watchers
}
