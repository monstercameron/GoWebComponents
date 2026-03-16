//go:build !js || !wasm
// +build !js !wasm

package devtools

import "time"

// SnapshotNow returns an empty snapshot on non-browser targets.
func SnapshotNow() Snapshot {
	return Snapshot{}
}

// UseSnapshot returns an empty snapshot on non-browser targets.
func UseSnapshot(refreshInterval time.Duration) Snapshot {
	return SnapshotNow()
}

// Panel is unavailable on non-browser targets and returns nil.
func Panel(props PanelProps) interface{} {
	return nil
}
