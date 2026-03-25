package ui

import "testing"

func TestOverlayStackSnapshotRoutesTopmostHandlers(parseT *testing.T) {
	parseManager := newOverlayStackManager()
	parseManager.upsert(overlayManagerRegistration{
		ID:                  "dialog",
		Kind:                OverlayKindDialog,
		BaseZIndex:          1000,
		TrapFocus:           true,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
	})
	parseManager.upsert(overlayManagerRegistration{
		ID:                  "popover",
		Kind:                OverlayKindPopover,
		BaseZIndex:          1000,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
	})
	parseManager.upsert(overlayManagerRegistration{
		ID:            "tooltip",
		Kind:          OverlayKindTooltip,
		BaseZIndex:    1000,
		CloseOnEscape: false,
	})

	parseDialog := parseManager.snapshot("dialog", overlayManagerRegistration{ID: "dialog", Kind: OverlayKindDialog, BaseZIndex: 1000}, true)
	parsePopover := parseManager.snapshot("popover", overlayManagerRegistration{ID: "popover", Kind: OverlayKindPopover, BaseZIndex: 1000}, true)
	parseTooltip := parseManager.snapshot("tooltip", overlayManagerRegistration{ID: "tooltip", Kind: OverlayKindTooltip, BaseZIndex: 1000}, true)

	if parseDialog.Depth != 0 {
		parseT.Fatalf("expected dialog depth 0, got %d", parseDialog.Depth)
	}
	if parsePopover.Depth != 1 {
		parseT.Fatalf("expected popover depth 1, got %d", parsePopover.Depth)
	}
	if parseTooltip.Depth != 2 {
		parseT.Fatalf("expected tooltip depth 2, got %d", parseTooltip.Depth)
	}
	if !parseTooltip.IsTop {
		parseT.Fatal("expected tooltip to be topmost layer")
	}
	if !parsePopover.HandlesEscape {
		parseT.Fatal("expected topmost escape-enabled overlay to handle escape")
	}
	if !parsePopover.HandlesOutsideClick {
		parseT.Fatal("expected topmost outside-dismissible overlay to handle outside click")
	}
	if !parseDialog.TrapFocusActive {
		parseT.Fatal("expected topmost focus-trapping overlay to own focus trap")
	}
	if parseTooltip.HandlesEscape {
		parseT.Fatal("did not expect tooltip without escape support to handle escape")
	}
	if parseDialog.SurfaceZIndex >= parsePopover.SurfaceZIndex {
		parseT.Fatalf("expected deeper layer to have higher z-index, got dialog=%d popover=%d", parseDialog.SurfaceZIndex, parsePopover.SurfaceZIndex)
	}
}

func TestOverlayStackSnapshotUsesFallbackForNewOpenLayer(parseT *testing.T) {
	parseManager := newOverlayStackManager()
	parseManager.upsert(overlayManagerRegistration{
		ID:            "dialog",
		Kind:          OverlayKindDialog,
		BaseZIndex:    1200,
		TrapFocus:     true,
		CloseOnEscape: true,
	})

	parseStack := parseManager.snapshot("sheet", overlayManagerRegistration{
		ID:                  "sheet",
		Kind:                OverlayKindSheet,
		BaseZIndex:          1200,
		TrapFocus:           true,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
	}, true)

	if parseStack.Depth != 1 {
		parseT.Fatalf("expected fallback layer depth 1, got %d", parseStack.Depth)
	}
	if !parseStack.IsTop {
		parseT.Fatal("expected fallback layer to be treated as topmost before effect registration")
	}
	if !parseStack.HandlesEscape || !parseStack.HandlesOutsideClick || !parseStack.TrapFocusActive {
		parseT.Fatalf("expected fallback layer to own topmost behaviors, got %#v", parseStack)
	}
	if parseStack.BackdropZIndex != 1202 || parseStack.SurfaceZIndex != 1203 {
		parseT.Fatalf("expected derived z-index pair 1202/1203, got %d/%d", parseStack.BackdropZIndex, parseStack.SurfaceZIndex)
	}
}
