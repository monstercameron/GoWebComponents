//go:build js && wasm

package ui

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// UseOverlayStack registers a surface in the shared overlay manager and returns its derived stack state.
func UseOverlayStack(parseOptions OverlayStackOptions) OverlayStack {
	parseId := parseOptions.ID
	if parseId == "" {
		parseId = UseId() + "-layer"
	}
	parseRegistration := overlayManagerRegistration{
		ID:                  parseId,
		Kind:                normalizeOverlayKind(parseOptions.Kind),
		BaseZIndex:          normalizeOverlayBaseZIndex(parseOptions.BaseZIndex),
		TrapFocus:           parseOptions.TrapFocus,
		CloseOnEscape:       parseOptions.CloseOnEscape,
		CloseOnOutsideClick: parseOptions.CloseOnOutsideClick,
	}
	parseVersion := UseState(0)

	UseEffect(func() func() {
		return globalOverlayStackManager.subscribe(func() {
			parseVersion.Update(func(parsePrevious int) int { return parsePrevious + 1 })
		})
	}, parseId)

	UseEffect(func() func() {
		if !parseOptions.Open {
			globalOverlayStackManager.remove(parseId)
			return nil
		}
		globalOverlayStackManager.upsert(parseRegistration)
		return func() {
			globalOverlayStackManager.remove(parseId)
		}
	}, parseId, parseOptions.Open, parseRegistration.Kind, parseRegistration.BaseZIndex, parseRegistration.TrapFocus, parseRegistration.CloseOnEscape, parseRegistration.CloseOnOutsideClick)

	_ = parseVersion.Get()
	return globalOverlayStackManager.snapshot(parseId, parseRegistration, parseOptions.Open)
}

// Overlay renders a stack-aware layered surface with coordinated z-order, dismissal routing, and focus ownership.
func Overlay(parseProps OverlayProps) Node {
	parseSurfaceID := parseProps.SurfaceID
	if parseSurfaceID == "" {
		parseSurfaceID = UseId() + "-overlay"
	}
	parseKind := normalizeOverlayKind(parseProps.Kind)
	parseModal := parseProps.Modal
	isParseRestoreFocus := parseProps.RestoreFocus || parseModal
	isParseTrapFocus := parseProps.TrapFocus || parseModal
	isParseCloseOnEscape := parseProps.CloseOnEscape || parseModal
	isParseBackgroundInert := parseProps.BackgroundInert || parseModal
	isParseShowBackdrop := parseProps.Backdrop || parseModal || parseProps.BackdropClass != ""
	parseRole := overlayRole(parseKind, parseProps.Role)
	parseContainerSelector := "#" + parseSurfaceID
	parseStack := UseOverlayStack(OverlayStackOptions{
		ID:                  parseSurfaceID,
		Open:                parseProps.Open,
		Kind:                parseKind,
		BaseZIndex:          parseProps.BaseZIndex,
		TrapFocus:           isParseTrapFocus,
		CloseOnEscape:       isParseCloseOnEscape,
		CloseOnOutsideClick: parseProps.CloseOnOutsideClick,
	})

	useManagedOverlayFocus(managedOverlayFocusOptions{
		Open:                  parseProps.Open,
		Active:                parseProps.Open && isParseTrapFocus && parseStack.TrapFocusActive,
		ContainerSelector:     parseContainerSelector,
		InitialFocusSelector:  parseProps.InitialFocusSelector,
		FallbackFocusSelector: parseProps.FallbackFocusSelector,
		RestoreFocus:          isParseRestoreFocus,
	})
	useOverlayEscape(parseProps.Open && isParseCloseOnEscape && parseStack.HandlesEscape, parseProps.OnDismiss)
	useOverlayScrollLock(parseProps.Open && parseProps.LockScroll)
	useOverlayBackgroundInert(parseProps.AppRootSelector, parseProps.Open && isParseBackgroundInert)
	if !isParseShowBackdrop {
		useOverlayOutsideDismiss(parseProps.Open && parseProps.CloseOnOutsideClick && parseStack.HandlesOutsideClick, parseContainerSelector, parseProps.OnDismiss)
	}

	if !parseProps.Open {
		return nil
	}

	parseChildren := make([]Node, 0, len(parseProps.Children)+1)
	if parseProps.Child != nil {
		parseChildren = append(parseChildren, parseProps.Child)
	}
	parseChildren = append(parseChildren, parseProps.Children...)

	parseStopClick := UseEvent(func(parseEvent MouseEvent) {
		parseEvent.StopPropagation()
	})
	var parseDismissHandler Handler
	if isParseShowBackdrop && parseProps.CloseOnOutsideClick && parseProps.OnDismiss != nil {
		parseDismissHandler = UseEvent(func() {
			parseProps.OnDismiss()
		})
	}

	parseSurfaceStyle := cloneOverlayStyle(parseProps.SurfaceStyle)
	parseSurfaceStyle["z-index"] = fmt.Sprintf("%d", parseStack.SurfaceZIndex)

	parseSurfaceProps := map[string]interface{}{
		"id":                           parseSurfaceID,
		"role":                         parseRole,
		"tabIndex":                     -1,
		"class":                        parseProps.SurfaceClass,
		"style":                        parseSurfaceStyle,
		"onclick":                      parseStopClick.value,
		"data-overlay-kind":            string(parseKind),
		"data-overlay-depth":           fmt.Sprintf("%d", parseStack.Depth),
		"data-overlay-handles-escape":  fmt.Sprintf("%t", parseStack.HandlesEscape),
		"data-overlay-handles-outside": fmt.Sprintf("%t", parseStack.HandlesOutsideClick),
		"data-overlay-trap-owner":      fmt.Sprintf("%t", parseStack.TrapFocusActive),
	}
	if parseProps.LabelledBy != "" {
		parseSurfaceProps["aria-labelledby"] = parseProps.LabelledBy
	}
	if parseProps.DescribedBy != "" {
		parseSurfaceProps["aria-describedby"] = parseProps.DescribedBy
	}
	if parseModal {
		parseSurfaceProps["aria-modal"] = "true"
	}
	if parseProps.AnchorSelector != "" {
		parseSurfaceProps["data-overlay-anchor"] = parseProps.AnchorSelector
	}
	if parseProps.Positioning != "" {
		parseSurfaceProps["data-overlay-positioning"] = parseProps.Positioning
	}

	parseSurface := runtime.CreateElementOwned("div", parseSurfaceProps, toInterfaces(parseChildren)...)
	parseOverlay := Node(parseSurface)
	if isParseShowBackdrop {
		parseBackdropStyle := cloneOverlayStyle(parseProps.BackdropStyle)
		parseBackdropStyle["z-index"] = fmt.Sprintf("%d", parseStack.BackdropZIndex)
		parseBackdropProps := map[string]interface{}{
			"class": parseProps.BackdropClass,
			"style": parseBackdropStyle,
		}
		if parseDismissHandler.value != nil {
			parseBackdropProps["onclick"] = parseDismissHandler.value
		}
		parseOverlay = runtime.CreateElementOwned("div", parseBackdropProps, parseSurface)
	}

	if parseProps.Target.Selector != "" || parseProps.Target.Node != nil {
		return Portal(PortalProps{Target: parseProps.Target, Child: parseOverlay})
	}
	return parseOverlay
}
