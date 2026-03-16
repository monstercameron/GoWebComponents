//go:build !js || !wasm
// +build !js !wasm

package devtools

import "time"

func SnapshotNow() Snapshot {
	return Snapshot{}
}

func UseSnapshot(refreshInterval time.Duration) Snapshot {
	return SnapshotNow()
}

func Panel(props PanelProps) interface{} {
	return nil
}
