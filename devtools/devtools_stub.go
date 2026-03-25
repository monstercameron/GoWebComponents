//go:build !js || !wasm
// +build !js !wasm

package devtools

import "time"

// SnapshotNow returns an empty snapshot on non-browser targets.
func SnapshotNow() Snapshot {
	if parseReplay, parseOk := CurrentTraceReplay(); parseOk {
		return parseReplay.Snapshot
	}
	return snapshotNowLive()
}

func snapshotNowLive() Snapshot {
	return Snapshot{}
}

// UseSnapshot returns an empty snapshot on non-browser targets.
func UseSnapshot(parseRefreshInterval time.Duration) Snapshot {
	return SnapshotNow()
}

// Panel is unavailable on non-browser targets and returns nil.
func Panel(parseProps PanelProps) interface{} {
	return nil
}

// ErrorOverlay is unavailable on non-browser targets and returns nil.
func ErrorOverlay(parseProps ErrorOverlayProps) interface{} {
	return nil
}
