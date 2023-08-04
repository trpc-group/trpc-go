// Package frame contains transport-level frame utilities.
package frame

// ShouldCopy judges whether to enable frame copy according to the current framer and options.
func ShouldCopy(isCopyOption, serverAsync, isSafeFramer bool) bool {
	// The following two scenarios do not need to copy frame.
	// Scenario 1: Framer is already safe on concurrent read.
	if isSafeFramer {
		return false
	}
	// Scenario 2: The server is in sync mod, and the caller does not want to copy frame(not stream RPC).
	if !serverAsync && !isCopyOption {
		return false
	}

	// Otherwise:
	// To avoid data overwriting of the concurrent reading Framer, enable copy frame by default.
	return true
}
