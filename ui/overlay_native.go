//go:build !js || !wasm
// +build !js !wasm

package ui

func UseOverlayStack(options OverlayStackOptions) OverlayStack {
	id := options.ID
	if id == "" {
		id = "overlay-layer"
	}
	registration := overlayManagerRegistration{
		ID:                  id,
		Kind:                normalizeOverlayKind(options.Kind),
		BaseZIndex:          normalizeOverlayBaseZIndex(options.BaseZIndex),
		TrapFocus:           options.TrapFocus,
		CloseOnEscape:       options.CloseOnEscape,
		CloseOnOutsideClick: options.CloseOnOutsideClick,
	}
	return globalOverlayStackManager.snapshot(id, registration, options.Open)
}

func Overlay(props OverlayProps) Node {
	children := make([]Node, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, props.Children...)
	return Fragment(children...)
}
