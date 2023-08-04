package http

// OptServerTransport modifies ServerTransport.
type OptServerTransport func(*ServerTransport)

// WithReusePort returns an OptServerTransport which enables reuse port.
func WithReusePort() OptServerTransport {
	return func(st *ServerTransport) {
		st.reusePort = true
	}
}

// WithEnableH2C returns an OptServerTransport which enables H2C.
func WithEnableH2C() OptServerTransport {
	return func(st *ServerTransport) {
		st.enableH2C = true
	}
}
