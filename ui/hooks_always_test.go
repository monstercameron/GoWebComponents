package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// TestAlwaysIsRuntimeSentinel pins ui.Always to the runtime sentinel so the
// every-render dependency stays recognizable to the hook implementations.
func TestAlwaysIsRuntimeSentinel(parseT *testing.T) {
	if Always != runtime.AlwaysDependency {
		parseT.Fatal("ui.Always must be the runtime AlwaysDependency sentinel")
	}
	if Always == nil || Always == any("always") {
		parseT.Fatal("ui.Always must be a distinct non-nil sentinel")
	}
}
