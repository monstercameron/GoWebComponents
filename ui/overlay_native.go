//go:build !js || !wasm
// +build !js !wasm

package ui

// UseOverlayStack registers and manages an overlay layer for the given stack options.
func UseOverlayStack(parseOptions OverlayStackOptions) OverlayStack {
	parseId := parseOptions.ID
	if parseId == "" {
		parseId = "overlay-layer"
	}
	parseRegistration := overlayManagerRegistration{
		ID:                  parseId,
		Kind:                normalizeOverlayKind(parseOptions.Kind),
		BaseZIndex:          normalizeOverlayBaseZIndex(parseOptions.BaseZIndex),
		TrapFocus:           parseOptions.TrapFocus,
		CloseOnEscape:       parseOptions.CloseOnEscape,
		CloseOnOutsideClick: parseOptions.CloseOnOutsideClick,
	}
	return globalOverlayStackManager.snapshot(parseId, parseRegistration, parseOptions.Open)
}

// Overlay renders overlay children as a Fragment on the server.
func Overlay(parseProps OverlayProps) Node {
	parseChildren := make([]Node, 0, len(parseProps.Children)+1)
	if parseProps.Child != nil {
		parseChildren = append(parseChildren, parseProps.Child)
	}
	parseChildren = append(parseChildren, parseProps.Children...)
	return Fragment(parseChildren...)
}
