package healthcheck

// Opt modifies HealthCheck.
type Opt func(*HealthCheck)

// WithUnregisteredServiceStatus changes the default status of unregistered service to status.
func WithUnregisteredServiceStatus(status Status) Opt {
	return func(hc *HealthCheck) {
		hc.unregisteredServiceStatus = status
	}
}

// WithStatusWatchers returns an Option which set watchers for HealthCheck.
func WithStatusWatchers(watchers map[string][]func(status Status)) Opt {
	return func(hc *HealthCheck) {
		hc.serviceWatchers = watchers
	}
}
