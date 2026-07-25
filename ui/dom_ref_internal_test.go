//go:build !(js && wasm)

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// fakeFocusNode is a runtime.DOMNode that also implements the imperative-handle capabilities
// (Focuser/Blurrer/Clicker/ScrollIntoViewer), recording calls — so DOMRef routing is testable
// on native.
type fakeFocusNode struct {
	null       bool
	focused    int
	blurred    int
	clicked    int
	scrolledTo []string
}

func (parseN *fakeFocusNode) IsNull() bool                { return parseN == nil || parseN.null }
func (parseN *fakeFocusNode) Equals(runtime.DOMNode) bool { return false }
func (parseN *fakeFocusNode) Focus()                      { parseN.focused++ }
func (parseN *fakeFocusNode) Blur()                       { parseN.blurred++ }
func (parseN *fakeFocusNode) Click()                      { parseN.clicked++ }
func (parseN *fakeFocusNode) ScrollIntoView(parseBehavior string) {
	parseN.scrolledTo = append(parseN.scrolledTo, parseBehavior)
}

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

func TestDOMRefImperativeHandlesRouteThroughCapabilities(t *testing.T) {
	parseNode := &fakeFocusNode{}
	parseRef := DOMRef{box: &domRefBox{node: parseNode}}

	parseRef.Blur()
	parseRef.Click()
	parseRef.ScrollIntoView("smooth")
	parseRef.ScrollIntoView("")

	if parseNode.blurred != 1 {
		t.Fatalf("expected 1 blur call, got %d", parseNode.blurred)
	}
	if parseNode.clicked != 1 {
		t.Fatalf("expected 1 click call, got %d", parseNode.clicked)
	}
	if len(parseNode.scrolledTo) != 2 || parseNode.scrolledTo[0] != "smooth" || parseNode.scrolledTo[1] != "" {
		t.Fatalf("expected scrollIntoView ['smooth',''], got %v", parseNode.scrolledTo)
	}
}

func TestDOMRefImperativeHandlesSafeWithoutCapability(t *testing.T) {
	defer func() {
		if parseR := recover(); parseR != nil {
			t.Fatalf("imperative handle panicked on a capability-less node: %v", parseR)
		}
	}()
	// nonFocusNode implements none of Blurrer/Clicker/ScrollIntoViewer — all must no-op.
	parseRef := DOMRef{box: &domRefBox{node: nonFocusNode{}}}
	parseRef.Blur()
	parseRef.Click()
	parseRef.ScrollIntoView("smooth")

	// Zero-value ref is also safe.
	var parseZero DOMRef
	parseZero.Blur()
	parseZero.Click()
	parseZero.ScrollIntoView("")
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
