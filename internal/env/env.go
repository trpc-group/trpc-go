// Package env defines environment variables used inside the framework.
package env

// Defines all keys of the environment variables.
const (
	// LogTrace controls whether to output trace log.
	// To enable trace output, set TRPC_LOG_TRACE=1.
	//
	// This environment variable is needed because zap library lacks trace-level log.
	LogTrace = "TRPC_LOG_TRACE"
)
