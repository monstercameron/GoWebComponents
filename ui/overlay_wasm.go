//go:build js && wasm
// +build js,wasm

package ui

import "syscall/js"

type managedOverlayFocusOptions struct {
	Open                  bool
	Active                bool
	ContainerSelector     string
	InitialFocusSelector  string
	FallbackFocusSelector string
	RestoreFocus          bool
}

type overlayInertState struct {
	count          int
	hasHidden      bool
	previousHidden string
	hadInert       bool
	previousInert  bool
}

var overlayScrollLockState struct {
	count            int
	previousOverflow string
}

var overlayInertRegistry = map[string]overlayInertState{}

// useManagedOverlayFocus is a core package helper.
func useManagedOverlayFocus(parseOptions managedOverlayFocusOptions) {
	parseManager := UseFocusManager()
	parseWasOpen := UseRef(false)

	UseEffect(func() func() {
		parsePreviouslyOpen := parseWasOpen.Get()
		if parseOptions.Open && !parsePreviouslyOpen && parseOptions.RestoreFocus {
			parseManager.RememberActive()
		}
		var parsePendingRestore *js.Func
		if !parseOptions.Open && parsePreviouslyOpen && parseOptions.RestoreFocus {
			parseIsFired := false
			var parseRestore js.Func
			parseRestore = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if parseIsFired {
					return nil
				}
				parseIsFired = true
				parseManager.Restore()
				parseRestore.Release()
				parsePendingRestore = nil
				return nil
			})
			parsePendingRestore = &parseRestore
			js.Global().Call("setTimeout", parseRestore, 0)
		}
		parseWasOpen.Set(parseOptions.Open)
		return func() {
			if parsePendingRestore != nil {
				parsePendingRestore.Release()
				parsePendingRestore = nil
			}
		}
	}, parseOptions.Open, parseOptions.RestoreFocus)

	UseEffect(func() func() {
		if !parseOptions.Active || parseOptions.ContainerSelector == "" {
			return nil
		}
		parseDocument := js.Global().Get("document")
		if !parseDocument.Truthy() {
			return nil
		}
		parseContainer := queryDocumentSelector(parseDocument, parseOptions.ContainerSelector)
		if !parseContainer.Truthy() {
			return nil
		}

		parseActive := parseDocument.Get("activeElement")
		isParseInside := parseActive.Truthy() && parseContainer.Call("contains", parseActive).Bool()
		if !isParseInside {
			isParseFocused := false
			if parseOptions.InitialFocusSelector != "" {
				isParseFocused = parseManager.FocusSelector(parseOptions.InitialFocusSelector)
			}
			if !isParseFocused && parseOptions.FallbackFocusSelector != "" {
				isParseFocused = parseManager.FocusSelector(parseOptions.FallbackFocusSelector)
			}
			if !isParseFocused {
				isParseFocused = focusElementValue(firstFocusableWithin(parseContainer))
			}
			if !isParseFocused {
				focusElementValue(parseContainer)
			}
		}

		parseListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if len(parseArgs2) == 0 {
				return nil
			}
			parseEvent := parseArgs2[0]
			if parseEvent.Get("key").String() != "Tab" {
				return nil
			}
			parseActive2 := parseDocument.Get("activeElement")
			parseFocusables := focusableValues(parseContainer)
			if len(parseFocusables) == 0 {
				parseEvent.Call("preventDefault")
				focusElementValue(parseContainer)
				return nil
			}
			parseFirst := parseFocusables[0]
			parseLast := parseFocusables[len(parseFocusables)-1]
			parseShift := parseEvent.Get("shiftKey").Bool()
			parseInside2 := parseContainer.Call("contains", parseActive2).Bool()
			if parseShift {
				if !parseInside2 || parseActive2.Equal(parseFirst) {
					parseEvent.Call("preventDefault")
					focusElementValue(parseLast)
				}
				return nil
			}
			if !parseInside2 || parseActive2.Equal(parseLast) {
				parseEvent.Call("preventDefault")
				focusElementValue(parseFirst)
			}
			return nil
		})
		parseDocument.Call("addEventListener", "keydown", parseListener)
		return func() {
			parseDocument.Call("removeEventListener", "keydown", parseListener)
			parseListener.Release()
		}
	}, parseOptions.Active, parseOptions.ContainerSelector, parseOptions.InitialFocusSelector, parseOptions.FallbackFocusSelector)
}

// useOverlayOutsideDismiss is a core package helper.
func useOverlayOutsideDismiss(isActive bool, parseSurfaceSelector string, parseOnDismiss func()) {
	UseEffect(func() func() {
		if !isActive || parseOnDismiss == nil || parseSurfaceSelector == "" {
			return nil
		}
		parseDocument := js.Global().Get("document")
		if !parseDocument.Truthy() {
			return nil
		}
		parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			if len(parseArgs) == 0 {
				return nil
			}
			parseSurface := queryDocumentSelector(parseDocument, parseSurfaceSelector)
			if !parseSurface.Truthy() {
				return nil
			}
			parseTarget := parseArgs[0].Get("target")
			if parseTarget.Truthy() && parseSurface.Call("contains", parseTarget).Bool() {
				return nil
			}
			parseOnDismiss()
			return nil
		})
		parseDocument.Call("addEventListener", "pointerdown", parseListener, true)
		return func() {
			parseDocument.Call("removeEventListener", "pointerdown", parseListener, true)
			parseListener.Release()
		}
	}, isActive, parseSurfaceSelector)
}

// overlayAcquireScrollLock is a core package helper.
func overlayAcquireScrollLock() {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseBody := parseDocument.Get("body")
	if !parseBody.Truthy() {
		return
	}
	parseStyle := parseBody.Get("style")
	if overlayScrollLockState.count == 0 {
		overlayScrollLockState.previousOverflow = parseStyle.Get("overflow").String()
		parseStyle.Set("overflow", "hidden")
	}
	overlayScrollLockState.count++
}

// overlayReleaseScrollLock is a core package helper.
func overlayReleaseScrollLock() {
	if overlayScrollLockState.count == 0 {
		return
	}
	overlayScrollLockState.count--
	if overlayScrollLockState.count > 0 {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseBody := parseDocument.Get("body")
	if !parseBody.Truthy() {
		return
	}
	parseBody.Get("style").Set("overflow", overlayScrollLockState.previousOverflow)
	overlayScrollLockState.previousOverflow = ""
}

// overlayAcquireBackgroundInert is a core package helper.
func overlayAcquireBackgroundInert(parseSelector string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseElement := queryDocumentSelector(parseDocument, parseSelector)
	if !parseElement.Truthy() {
		return
	}
	parseState := overlayInertRegistry[parseSelector]
	if parseState.count == 0 {
		parseState.hasHidden = parseElement.Call("hasAttribute", "aria-hidden").Bool()
		if parseState.hasHidden {
			parseState.previousHidden = parseElement.Call("getAttribute", "aria-hidden").String()
		}
		parseInertValue := parseElement.Get("inert")
		if parseInertValue.Type() != js.TypeUndefined && parseInertValue.Type() != js.TypeNull {
			parseState.hadInert = true
			parseState.previousInert = parseInertValue.Bool()
			parseElement.Set("inert", true)
		}
		parseElement.Call("setAttribute", "aria-hidden", "true")
	}
	parseState.count++
	overlayInertRegistry[parseSelector] = parseState
}

// overlayReleaseBackgroundInert is a core package helper.
func overlayReleaseBackgroundInert(parseSelector string) {
	parseState, parseOk := overlayInertRegistry[parseSelector]
	if !parseOk || parseState.count == 0 {
		return
	}
	parseState.count--
	if parseState.count > 0 {
		overlayInertRegistry[parseSelector] = parseState
		return
	}
	delete(overlayInertRegistry, parseSelector)
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseElement := queryDocumentSelector(parseDocument, parseSelector)
	if !parseElement.Truthy() {
		return
	}
	if parseState.hasHidden {
		parseElement.Call("setAttribute", "aria-hidden", parseState.previousHidden)
	} else {
		parseElement.Call("removeAttribute", "aria-hidden")
	}
	if parseState.hadInert {
		parseElement.Set("inert", parseState.previousInert)
	}
}
