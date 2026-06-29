//go:build !js || !wasm

package ui

// UseOverlayStack registers and manages an overlay layer for the given stack options.
func UseOverlayStack(parseOverlayOptions OverlayStackOptions) OverlayStack {
	parseOverlayID := parseOverlayOptions.ID
	if parseOverlayID == "" {
		parseOverlayID = "overlay-layer"
	}
	parseOverlayRegistration := overlayManagerRegistration{
		ID:                  parseOverlayID,
		Kind:                normalizeOverlayKind(parseOverlayOptions.Kind),
		BaseZIndex:          normalizeOverlayBaseZIndex(parseOverlayOptions.BaseZIndex),
		TrapFocus:           parseOverlayOptions.TrapFocus,
		CloseOnEscape:       parseOverlayOptions.CloseOnEscape,
		CloseOnOutsideClick: parseOverlayOptions.CloseOnOutsideClick,
	}
	return globalOverlayStackManager.snapshot(parseOverlayID, parseOverlayRegistration, parseOverlayOptions.Open)
}

// Overlay renders overlay children as a Fragment on the server.
func Overlay(parseOverlayProps OverlayProps) Node {
	parseOverlayChildren := make([]Node, 0, len(parseOverlayProps.Children)+1)
	if parseOverlayProps.Child != nil {
		parseOverlayChildren = append(parseOverlayChildren, parseOverlayProps.Child)
	}
	parseOverlayChildren = append(parseOverlayChildren, parseOverlayProps.Children...)
	return Fragment(parseOverlayChildren...)
}
