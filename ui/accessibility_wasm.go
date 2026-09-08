//go:build js && wasm

package ui

import (
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"strings"
	"syscall/js"
)

const focusableSelector = `button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])`

// RememberActive is a core package helper.
func (parseM FocusManager) RememberActive() bool {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	parseActive := parseDocument.Get("activeElement")
	if !parseActive.Truthy() {
		return false
	}
	parseM.remembered.Set(parseActive)
	return true
}

// FocusSelector is a core package helper.
func (parseM FocusManager) FocusSelector(parseSelector string, parseOptions ...FocusOptions) bool {
	if parseSelector == "" {
		return false
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	parseElement := queryDocumentSelector(parseDocument, parseSelector)
	return focusElementValue(parseElement, parseOptions...)
}

// FocusByID is a core package helper.
func (parseM FocusManager) FocusByID(parseId string, parseOptions ...FocusOptions) bool {
	if parseId == "" {
		return false
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	parseElement := parseDocument.Call("getElementById", parseId)
	return focusElementValue(parseElement, parseOptions...)
}

// FocusFirst is a core package helper.
func (parseM FocusManager) FocusFirst(parseContainerSelector string, parseOptions ...FocusOptions) bool {
	if parseContainerSelector == "" {
		return false
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	parseContainer := queryDocumentSelector(parseDocument, parseContainerSelector)
	if !parseContainer.Truthy() {
		return false
	}
	parseFirst := firstFocusableWithin(parseContainer)
	if parseFirst.Truthy() {
		return focusElementValue(parseFirst, parseOptions...)
	}
	return focusElementValue(parseContainer, parseOptions...)
}

// Restore is a core package helper.
func (parseM FocusManager) Restore(parseOptions ...FocusOptions) bool {
	parseRaw := parseM.remembered.Get()
	parseValue, parseOk := parseRaw.(js.Value)
	if !parseOk || !parseValue.Truthy() {
		return false
	}
	return focusElementValue(parseValue, parseOptions...)
}

// UseFocusTrap installs a keyboard focus trap within the given container while active.
func UseFocusTrap(parseOptions FocusTrapOptions) {
	parseManager := UseFocusManager()
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

		if parseOptions.RestoreFocus {
			parseManager.RememberActive()
		}
		isParseFocused := false
		if parseOptions.InitialFocusSelector != "" {
			isParseFocused = parseManager.FocusSelector(parseOptions.InitialFocusSelector)
		}
		if !isParseFocused && parseOptions.FallbackFocusSelector != "" {
			isParseFocused = parseManager.FocusSelector(parseOptions.FallbackFocusSelector)
		}
		if !isParseFocused {
			focusElementValue(firstFocusableWithin(parseContainer))
		}
		if !isParseFocused {
			focusElementValue(parseContainer)
		}

		parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("ui", "UseFocusTrap callback")
			if len(parseArgs) == 0 {
				return nil
			}
			parseEvent := parseArgs[0]
			if parseEvent.Get("key").String() != "Tab" {
				return nil
			}
			parseActive := parseDocument.Get("activeElement")
			parseFocusables := focusableValues(parseContainer)
			if len(parseFocusables) == 0 {
				parseEvent.Call("preventDefault")
				focusElementValue(parseContainer)
				return nil
			}
			parseFirst := parseFocusables[0]
			parseLast := parseFocusables[len(parseFocusables)-1]
			parseShift := parseEvent.Get("shiftKey").Bool()
			parseInside := parseContainer.Call("contains", parseActive).Bool()
			if parseShift {
				if !parseInside || parseActive.Equal(parseFirst) {
					parseEvent.Call("preventDefault")
					focusElementValue(parseLast)
				}
				return nil
			}
			if !parseInside || parseActive.Equal(parseLast) {
				parseEvent.Call("preventDefault")
				focusElementValue(parseFirst)
			}
			return nil
		})
		parseDocument.Call("addEventListener", "keydown", parseListener)

		return func() {
			parseDocument.Call("removeEventListener", "keydown", parseListener)
			parseListener.Release()
			if parseOptions.RestoreFocus {
				var parseRestore js.Func
				parseRestore = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
					defer runtime.RecoverContainedPanic("ui", "UseFocusTrap callback")
					parseManager.Restore()
					parseRestore.Release()
					return nil
				})
				js.Global().Call("setTimeout", parseRestore, 0)
			}
		}
	}, parseOptions.Active, parseOptions.ContainerSelector, parseOptions.InitialFocusSelector, parseOptions.FallbackFocusSelector, parseOptions.RestoreFocus)
}

// useOverlayEscape is a core package helper.
func useOverlayEscape(isActive bool, parseOnDismiss func()) {
	UseEffect(func() func() {
		if !isActive || parseOnDismiss == nil {
			return nil
		}
		parseDocument := js.Global().Get("document")
		if !parseDocument.Truthy() {
			return nil
		}
		parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("ui", "useOverlayEscape callback")
			if len(parseArgs) == 0 {
				return nil
			}
			parseEvent := parseArgs[0]
			if parseEvent.Get("key").String() == "Escape" {
				parseEvent.Call("preventDefault")
				parseOnDismiss()
			}
			return nil
		})
		parseDocument.Call("addEventListener", "keydown", parseListener)
		return func() {
			parseDocument.Call("removeEventListener", "keydown", parseListener)
			parseListener.Release()
		}
	}, isActive)
}

// useOverlayScrollLock is a core package helper.
func useOverlayScrollLock(isActive bool) {
	UseEffect(func() func() {
		if !isActive {
			return nil
		}
		overlayAcquireScrollLock()
		return func() {
			overlayReleaseScrollLock()
		}
	}, isActive)
}

// useOverlayBackgroundInert is a core package helper.
func useOverlayBackgroundInert(parseSelector string, isActive bool) {
	UseEffect(func() func() {
		if !isActive || parseSelector == "" {
			return nil
		}
		overlayAcquireBackgroundInert(parseSelector)
		return func() {
			overlayReleaseBackgroundInert(parseSelector)
		}
	}, isActive, parseSelector)
}

// focusElementValue is a core package helper.
func focusElementValue(parseElement js.Value, parseOptions ...FocusOptions) bool {
	if !parseElement.Truthy() {
		return false
	}
	parseFocus := parseElement.Get("focus")
	if parseFocus.Type() != js.TypeFunction {
		return false
	}
	isParsePreventScroll := false
	if len(parseOptions) > 0 {
		isParsePreventScroll = parseOptions[0].PreventScroll
	}
	if isParsePreventScroll {
		parseArg := js.Global().Get("Object").New()
		parseArg.Set("preventScroll", true)
		parseElement.Call("focus", parseArg)
		return true
	}
	parseElement.Call("focus")
	return true
}

// firstFocusableWithin is a core package helper.
func firstFocusableWithin(parseContainer js.Value) js.Value {
	parseItems := focusableValues(parseContainer)
	if len(parseItems) == 0 {
		return js.Undefined()
	}
	return parseItems[0]
}

// focusableValues is a core package helper.
func focusableValues(parseContainer js.Value) []js.Value {
	if !parseContainer.Truthy() {
		return nil
	}
	parseList := parseContainer.Call("querySelectorAll", focusableSelector)
	if !parseList.Truthy() {
		return nil
	}
	parseLength := parseList.Get("length").Int()
	parseValues := make([]js.Value, 0, parseLength)
	for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
		parseCandidate := parseList.Index(parseIndex)
		if parseCandidate.Truthy() {
			parseValues = append(parseValues, parseCandidate)
		}
	}
	return parseValues
}

// queryDocumentSelector is a core package helper.
func queryDocumentSelector(parseDocument js.Value, parseSelector string) js.Value {
	if !parseDocument.Truthy() || parseSelector == "" {
		return js.Undefined()
	}
	if strings.HasPrefix(parseSelector, "#") && len(parseSelector) > 1 {
		return parseDocument.Call("getElementById", parseSelector[1:])
	}
	return parseDocument.Call("querySelector", parseSelector)
}
