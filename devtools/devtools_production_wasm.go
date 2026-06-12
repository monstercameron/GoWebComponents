//go:build js && wasm && production

package devtools

import (
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

// SnapshotNow returns a minimal snapshot in production wasm builds.
// The devtools polling loop is excluded from production; only kernel and
// extension state (which do not require js/wasm imports) are available.
func SnapshotNow() Snapshot {
	if parseReplay, parseOk := CurrentTraceReplay(); parseOk {
		return parseReplay.Snapshot
	}
	return snapshotNowLive()
}

func snapshotNowLive() Snapshot {
	return Snapshot{
		Kernel:     snapshotKernelState(),
		Extensions: InspectComposedExtensionSections(),
	}
}

// UseSnapshot returns a static snapshot in production wasm builds.
// No polling interval is installed.
func UseSnapshot(_ time.Duration) Snapshot {
	return SnapshotNow()
}

// Panel is a no-op in production wasm builds and returns nil.
func Panel(_ PanelProps) ui.Node {
	return nil
}

// ErrorOverlay is a no-op in production wasm builds and returns nil.
func ErrorOverlay(_ ErrorOverlayProps) ui.Node {
	return nil
}
