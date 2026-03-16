//go:build js && wasm
// +build js,wasm

package ui

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// UseOverlayStack registers a surface in the shared overlay manager and returns its derived stack state.
func UseOverlayStack(options OverlayStackOptions) OverlayStack {
	id := options.ID
	if id == "" {
		id = UseId() + "-layer"
	}
	registration := overlayManagerRegistration{
		ID:                  id,
		Kind:                normalizeOverlayKind(options.Kind),
		BaseZIndex:          normalizeOverlayBaseZIndex(options.BaseZIndex),
		TrapFocus:           options.TrapFocus,
		CloseOnEscape:       options.CloseOnEscape,
		CloseOnOutsideClick: options.CloseOnOutsideClick,
	}
	version := UseState(0)

	UseEffect(func() func() {
		return globalOverlayStackManager.subscribe(func() {
			version.Update(func(previous int) int { return previous + 1 })
		})
	}, id)

	UseEffect(func() func() {
		if !options.Open {
			globalOverlayStackManager.remove(id)
			return nil
		}
		globalOverlayStackManager.upsert(registration)
		return func() {
			globalOverlayStackManager.remove(id)
		}
	}, id, options.Open, registration.Kind, registration.BaseZIndex, registration.TrapFocus, registration.CloseOnEscape, registration.CloseOnOutsideClick)

	_ = version.Get()
	return globalOverlayStackManager.snapshot(id, registration, options.Open)
}

// Overlay renders a stack-aware layered surface with coordinated z-order, dismissal routing, and focus ownership.
func Overlay(props OverlayProps) Node {
	surfaceID := props.SurfaceID
	if surfaceID == "" {
		surfaceID = UseId() + "-overlay"
	}
	kind := normalizeOverlayKind(props.Kind)
	modal := props.Modal
	restoreFocus := props.RestoreFocus || modal
	trapFocus := props.TrapFocus || modal
	closeOnEscape := props.CloseOnEscape || modal
	backgroundInert := props.BackgroundInert || modal
	showBackdrop := props.Backdrop || modal || props.BackdropClass != ""
	role := overlayRole(kind, props.Role)
	containerSelector := "#" + surfaceID
	stack := UseOverlayStack(OverlayStackOptions{
		ID:                  surfaceID,
		Open:                props.Open,
		Kind:                kind,
		BaseZIndex:          props.BaseZIndex,
		TrapFocus:           trapFocus,
		CloseOnEscape:       closeOnEscape,
		CloseOnOutsideClick: props.CloseOnOutsideClick,
	})

	useManagedOverlayFocus(managedOverlayFocusOptions{
		Open:                  props.Open,
		Active:                props.Open && trapFocus && stack.TrapFocusActive,
		ContainerSelector:     containerSelector,
		InitialFocusSelector:  props.InitialFocusSelector,
		FallbackFocusSelector: props.FallbackFocusSelector,
		RestoreFocus:          restoreFocus,
	})
	useOverlayEscape(props.Open && closeOnEscape && stack.HandlesEscape, props.OnDismiss)
	useOverlayScrollLock(props.Open && props.LockScroll)
	useOverlayBackgroundInert(props.AppRootSelector, props.Open && backgroundInert)
	if !showBackdrop {
		useOverlayOutsideDismiss(props.Open && props.CloseOnOutsideClick && stack.HandlesOutsideClick, containerSelector, props.OnDismiss)
	}

	if !props.Open {
		return nil
	}

	children := make([]Node, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, props.Children...)

	stopClick := UseEvent(func(event MouseEvent) {
		event.StopPropagation()
	})
	var dismissHandler Handler
	if showBackdrop && props.CloseOnOutsideClick && props.OnDismiss != nil {
		dismissHandler = UseEvent(func() {
			props.OnDismiss()
		})
	}

	surfaceStyle := cloneOverlayStyle(props.SurfaceStyle)
	surfaceStyle["z-index"] = fmt.Sprintf("%d", stack.SurfaceZIndex)

	surfaceProps := map[string]interface{}{
		"id":                           surfaceID,
		"role":                         role,
		"tabIndex":                     -1,
		"class":                        props.SurfaceClass,
		"style":                        surfaceStyle,
		"onclick":                      stopClick.value,
		"data-overlay-kind":            string(kind),
		"data-overlay-depth":           fmt.Sprintf("%d", stack.Depth),
		"data-overlay-handles-escape":  fmt.Sprintf("%t", stack.HandlesEscape),
		"data-overlay-handles-outside": fmt.Sprintf("%t", stack.HandlesOutsideClick),
		"data-overlay-trap-owner":      fmt.Sprintf("%t", stack.TrapFocusActive),
	}
	if props.LabelledBy != "" {
		surfaceProps["aria-labelledby"] = props.LabelledBy
	}
	if props.DescribedBy != "" {
		surfaceProps["aria-describedby"] = props.DescribedBy
	}
	if modal {
		surfaceProps["aria-modal"] = "true"
	}
	if props.AnchorSelector != "" {
		surfaceProps["data-overlay-anchor"] = props.AnchorSelector
	}
	if props.Positioning != "" {
		surfaceProps["data-overlay-positioning"] = props.Positioning
	}

	surface := runtime.CreateElement("div", surfaceProps, toInterfaces(children)...)
	overlay := Node(surface)
	if showBackdrop {
		backdropStyle := cloneOverlayStyle(props.BackdropStyle)
		backdropStyle["z-index"] = fmt.Sprintf("%d", stack.BackdropZIndex)
		backdropProps := map[string]interface{}{
			"class": props.BackdropClass,
			"style": backdropStyle,
		}
		if dismissHandler.value != nil {
			backdropProps["onclick"] = dismissHandler.value
		}
		overlay = runtime.CreateElement("div", backdropProps, surface)
	}

	if props.Target.Selector != "" || props.Target.Node != nil {
		return Portal(PortalProps{Target: props.Target, Child: overlay})
	}
	return overlay
}
