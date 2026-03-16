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

func useManagedOverlayFocus(options managedOverlayFocusOptions) {
	manager := UseFocusManager()
	wasOpen := UseRef(false)

	UseEffect(func() func() {
		previouslyOpen := wasOpen.Get()
		if options.Open && !previouslyOpen && options.RestoreFocus {
			manager.RememberActive()
		}
		if !options.Open && previouslyOpen && options.RestoreFocus {
			var restore js.Func
			restore = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				manager.Restore()
				restore.Release()
				return nil
			})
			js.Global().Call("setTimeout", restore, 0)
		}
		wasOpen.Set(options.Open)
		return nil
	}, options.Open, options.RestoreFocus)

	UseEffect(func() func() {
		if !options.Active || options.ContainerSelector == "" {
			return nil
		}
		document := js.Global().Get("document")
		if !document.Truthy() {
			return nil
		}
		container := queryDocumentSelector(document, options.ContainerSelector)
		if !container.Truthy() {
			return nil
		}

		active := document.Get("activeElement")
		inside := active.Truthy() && container.Call("contains", active).Bool()
		if !inside {
			focused := false
			if options.InitialFocusSelector != "" {
				focused = manager.FocusSelector(options.InitialFocusSelector)
			}
			if !focused && options.FallbackFocusSelector != "" {
				focused = manager.FocusSelector(options.FallbackFocusSelector)
			}
			if !focused {
				focused = focusElementValue(firstFocusableWithin(container))
			}
			if !focused {
				focusElementValue(container)
			}
		}

		listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				return nil
			}
			event := args[0]
			if event.Get("key").String() != "Tab" {
				return nil
			}
			active := document.Get("activeElement")
			focusables := focusableValues(container)
			if len(focusables) == 0 {
				event.Call("preventDefault")
				focusElementValue(container)
				return nil
			}
			first := focusables[0]
			last := focusables[len(focusables)-1]
			shift := event.Get("shiftKey").Bool()
			inside := container.Call("contains", active).Bool()
			if shift {
				if !inside || active.Equal(first) {
					event.Call("preventDefault")
					focusElementValue(last)
				}
				return nil
			}
			if !inside || active.Equal(last) {
				event.Call("preventDefault")
				focusElementValue(first)
			}
			return nil
		})
		document.Call("addEventListener", "keydown", listener)
		return func() {
			document.Call("removeEventListener", "keydown", listener)
			listener.Release()
		}
	}, options.Active, options.ContainerSelector, options.InitialFocusSelector, options.FallbackFocusSelector)
}

func useOverlayOutsideDismiss(active bool, surfaceSelector string, onDismiss func()) {
	UseEffect(func() func() {
		if !active || onDismiss == nil || surfaceSelector == "" {
			return nil
		}
		document := js.Global().Get("document")
		if !document.Truthy() {
			return nil
		}
		listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				return nil
			}
			surface := queryDocumentSelector(document, surfaceSelector)
			if !surface.Truthy() {
				return nil
			}
			target := args[0].Get("target")
			if target.Truthy() && surface.Call("contains", target).Bool() {
				return nil
			}
			onDismiss()
			return nil
		})
		document.Call("addEventListener", "pointerdown", listener, true)
		return func() {
			document.Call("removeEventListener", "pointerdown", listener, true)
			listener.Release()
		}
	}, active, surfaceSelector)
}

func overlayAcquireScrollLock() {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	body := document.Get("body")
	if !body.Truthy() {
		return
	}
	style := body.Get("style")
	if overlayScrollLockState.count == 0 {
		overlayScrollLockState.previousOverflow = style.Get("overflow").String()
		style.Set("overflow", "hidden")
	}
	overlayScrollLockState.count++
}

func overlayReleaseScrollLock() {
	if overlayScrollLockState.count == 0 {
		return
	}
	overlayScrollLockState.count--
	if overlayScrollLockState.count > 0 {
		return
	}
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	body := document.Get("body")
	if !body.Truthy() {
		return
	}
	body.Get("style").Set("overflow", overlayScrollLockState.previousOverflow)
	overlayScrollLockState.previousOverflow = ""
}

func overlayAcquireBackgroundInert(selector string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	element := queryDocumentSelector(document, selector)
	if !element.Truthy() {
		return
	}
	state := overlayInertRegistry[selector]
	if state.count == 0 {
		state.hasHidden = element.Call("hasAttribute", "aria-hidden").Bool()
		if state.hasHidden {
			state.previousHidden = element.Call("getAttribute", "aria-hidden").String()
		}
		inertValue := element.Get("inert")
		if inertValue.Type() != js.TypeUndefined && inertValue.Type() != js.TypeNull {
			state.hadInert = true
			state.previousInert = inertValue.Bool()
			element.Set("inert", true)
		}
		element.Call("setAttribute", "aria-hidden", "true")
	}
	state.count++
	overlayInertRegistry[selector] = state
}

func overlayReleaseBackgroundInert(selector string) {
	state, ok := overlayInertRegistry[selector]
	if !ok || state.count == 0 {
		return
	}
	state.count--
	if state.count > 0 {
		overlayInertRegistry[selector] = state
		return
	}
	delete(overlayInertRegistry, selector)
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	element := queryDocumentSelector(document, selector)
	if !element.Truthy() {
		return
	}
	if state.hasHidden {
		element.Call("setAttribute", "aria-hidden", state.previousHidden)
	} else {
		element.Call("removeAttribute", "aria-hidden")
	}
	if state.hadInert {
		element.Set("inert", state.previousInert)
	}
}
