//go:build !production

package runtime

// Hook signatures power development-only hot-reload compatibility and agent
// inspection. Keeping the switch constant lets production builds erase every
// per-hook append and its backing allocation.
const hookSignatureRecordingEnabled = true
