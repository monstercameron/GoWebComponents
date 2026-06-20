//go:build !(js && wasm)

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// fakeFocusNode is a runtime.DOMNode that also implements runtime.Focuser,
// recording focus calls — so DOMRef.Focus routing is testable on native.
type fakeFocusNode struct {
	null    bool
	focused int
}

func (parseN *fakeFocusNode) IsNull() bool                { return parseN == nil || parseN.null }
func (parseN *fakeFocusNode) Equals(runtime.DOMNode) bool { return false }
func (parseN *fakeFocusNode) Focus()                      { parseN.focused++ }

func TestDOMRefFocusRoutesThroughFocuser(t *testing.T) {
	parseNode := &fakeFocusNode{}
	parseRef := DOMRef{box: &domRefBox{node: parseNode}}

	parseRef.Focus()
	parseRef.Focus()
	if parseNode.focused != 2 {
		t.Fatalf("expected 2 focus calls routed to the node, got %d", parseNode.focused)
	}
}

func TestDOMRefFocusIsSafeWhenUnmountedOrZero(t *testing.T) {
	defer func() {
		if parseR := recover(); parseR != nil {
			t.Fatalf("Focus panicked: %v", parseR)
		}
	}()

	// Zero-value ref (no box).
	var parseZero DOMRef
	parseZero.Focus()
	if parseZero.Mounted() {
		t.Fatal("zero ref must not be mounted")
	}

	// Box present but no node (pre-mount / post-unmount).
	parseUnmounted := DOMRef{box: &domRefBox{}}
	parseUnmounted.Focus()
	if parseUnmounted.Mounted() {
		t.Fatal("ref without a node must not be mounted")
	}

	// Node present but null.
	parseNull := DOMRef{box: &domRefBox{node: &fakeFocusNode{null: true}}}
	parseNull.Focus()
	if parseNull.Mounted() {
		t.Fatal("null node must report not mounted")
	}

	// A node that does NOT implement Focuser must be a no-op, not a panic.
	parseNonFocuser := DOMRef{box: &domRefBox{node: nonFocusNode{}}}
	parseNonFocuser.Focus()
}

// nonFocusNode is a DOMNode without the Focuser capability.
type nonFocusNode struct{}

func (nonFocusNode) IsNull() bool                { return false }
func (nonFocusNode) Equals(runtime.DOMNode) bool { return false }

func TestUseAutoFocusOnceDepsIsStableAndNonEmpty(t *testing.T) {
	// The once-on-mount sentinel must be non-empty (empty => "every render" in
	// UseEffect, which would steal focus continuously) and stable across reads.
	if len(autoFocusOnceDeps) == 0 {
		t.Fatal("autoFocusOnceDeps must be non-empty so UseAutoFocus runs once, not every render")
	}
	parseFirst := autoFocusOnceDeps
	parseSecond := autoFocusOnceDeps
	if &parseFirst[0] != &parseSecond[0] {
		t.Fatal("sentinel deps must be a shared stable slice")
	}
}
