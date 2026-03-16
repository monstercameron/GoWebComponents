package ui

import "testing"

func TestOverlayStackSnapshotRoutesTopmostHandlers(t *testing.T) {
	manager := newOverlayStackManager()
	manager.upsert(overlayManagerRegistration{
		ID:                  "dialog",
		Kind:                OverlayKindDialog,
		BaseZIndex:          1000,
		TrapFocus:           true,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
	})
	manager.upsert(overlayManagerRegistration{
		ID:                  "popover",
		Kind:                OverlayKindPopover,
		BaseZIndex:          1000,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
	})
	manager.upsert(overlayManagerRegistration{
		ID:            "tooltip",
		Kind:          OverlayKindTooltip,
		BaseZIndex:    1000,
		CloseOnEscape: false,
	})

	dialog := manager.snapshot("dialog", overlayManagerRegistration{ID: "dialog", Kind: OverlayKindDialog, BaseZIndex: 1000}, true)
	popover := manager.snapshot("popover", overlayManagerRegistration{ID: "popover", Kind: OverlayKindPopover, BaseZIndex: 1000}, true)
	tooltip := manager.snapshot("tooltip", overlayManagerRegistration{ID: "tooltip", Kind: OverlayKindTooltip, BaseZIndex: 1000}, true)

	if dialog.Depth != 0 {
		t.Fatalf("expected dialog depth 0, got %d", dialog.Depth)
	}
	if popover.Depth != 1 {
		t.Fatalf("expected popover depth 1, got %d", popover.Depth)
	}
	if tooltip.Depth != 2 {
		t.Fatalf("expected tooltip depth 2, got %d", tooltip.Depth)
	}
	if !tooltip.IsTop {
		t.Fatal("expected tooltip to be topmost layer")
	}
	if !popover.HandlesEscape {
		t.Fatal("expected topmost escape-enabled overlay to handle escape")
	}
	if !popover.HandlesOutsideClick {
		t.Fatal("expected topmost outside-dismissible overlay to handle outside click")
	}
	if !dialog.TrapFocusActive {
		t.Fatal("expected topmost focus-trapping overlay to own focus trap")
	}
	if tooltip.HandlesEscape {
		t.Fatal("did not expect tooltip without escape support to handle escape")
	}
	if dialog.SurfaceZIndex >= popover.SurfaceZIndex {
		t.Fatalf("expected deeper layer to have higher z-index, got dialog=%d popover=%d", dialog.SurfaceZIndex, popover.SurfaceZIndex)
	}
}

func TestOverlayStackSnapshotUsesFallbackForNewOpenLayer(t *testing.T) {
	manager := newOverlayStackManager()
	manager.upsert(overlayManagerRegistration{
		ID:            "dialog",
		Kind:          OverlayKindDialog,
		BaseZIndex:    1200,
		TrapFocus:     true,
		CloseOnEscape: true,
	})

	stack := manager.snapshot("sheet", overlayManagerRegistration{
		ID:                  "sheet",
		Kind:                OverlayKindSheet,
		BaseZIndex:          1200,
		TrapFocus:           true,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
	}, true)

	if stack.Depth != 1 {
		t.Fatalf("expected fallback layer depth 1, got %d", stack.Depth)
	}
	if !stack.IsTop {
		t.Fatal("expected fallback layer to be treated as topmost before effect registration")
	}
	if !stack.HandlesEscape || !stack.HandlesOutsideClick || !stack.TrapFocusActive {
		t.Fatalf("expected fallback layer to own topmost behaviors, got %#v", stack)
	}
	if stack.BackdropZIndex != 1202 || stack.SurfaceZIndex != 1203 {
		t.Fatalf("expected derived z-index pair 1202/1203, got %d/%d", stack.BackdropZIndex, stack.SurfaceZIndex)
	}
}
