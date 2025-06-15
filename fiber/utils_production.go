//go:build production
// +build production

package fiber

// isDebugBuild returns false for production builds
// This allows the compiler to completely eliminate debug calls
func isDebugBuild() bool {
	return false
}

// Production-optimized debugf that becomes a no-op
// The compiler will inline this and eliminate the call entirely
func debugf(namespace, format string, a ...interface{}) {
	// No-op in production builds - compiler will eliminate this function call
} 