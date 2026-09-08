//go:build !windows

package wails

import "testing"

// TestNativeBackendUnavailable verifies unsupported platforms advertise no native backend.
func TestNativeBackendUnavailable(parseTest *testing.T) {
	if NewNativeBackend() != nil { parseTest.Fatal("native backend must be unavailable") }
}
