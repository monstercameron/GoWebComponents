//go:build js && wasm
// +build js,wasm

package ui

import (
	"strings"
	"syscall/js"
)

const focusableSelector = `button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])`

func (m FocusManager) RememberActive() bool {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return false
	}
	active := document.Get("activeElement")
	if !active.Truthy() {
		return false
	}
	m.remembered.Set(active)
	return true
}

func (m FocusManager) FocusSelector(selector string, options ...FocusOptions) bool {
	if selector == "" {
		return false
	}
	document := js.Global().Get("document")
	if !document.Truthy() {
		return false
	}
	element := queryDocumentSelector(document, selector)
	return focusElementValue(element, options...)
}

func (m FocusManager) FocusByID(id string, options ...FocusOptions) bool {
	if id == "" {
		return false
	}
	document := js.Global().Get("document")
	if !document.Truthy() {
		return false
	}
	element := document.Call("getElementById", id)
	return focusElementValue(element, options...)
}

func (m FocusManager) FocusFirst(containerSelector string, options ...FocusOptions) bool {
	if containerSelector == "" {
		return false
	}
	document := js.Global().Get("document")
	if !document.Truthy() {
		return false
	}
	container := queryDocumentSelector(document, containerSelector)
	if !container.Truthy() {
		return false
	}
	first := firstFocusableWithin(container)
	if first.Truthy() {
		return focusElementValue(first, options...)
	}
	return focusElementValue(container, options...)
}

func (m FocusManager) Restore(options ...FocusOptions) bool {
	raw := m.remembered.Get()
	value, ok := raw.(js.Value)
	if !ok || !value.Truthy() {
		return false
	}
	return focusElementValue(value, options...)
}

func UseFocusTrap(options FocusTrapOptions) {
	manager := UseFocusManager()
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

		if options.RestoreFocus {
			manager.RememberActive()
		}
		focused := false
		if options.InitialFocusSelector != "" {
			focused = manager.FocusSelector(options.InitialFocusSelector)
		}
		if !focused && options.FallbackFocusSelector != "" {
			focused = manager.FocusSelector(options.FallbackFocusSelector)
		}
		if !focused {
			focusElementValue(firstFocusableWithin(container))
		}
		if !focused {
			focusElementValue(container)
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
			if options.RestoreFocus {
				var restore js.Func
				restore = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					manager.Restore()
					restore.Release()
					return nil
				})
				js.Global().Call("setTimeout", restore, 0)
			}
		}
	}, options.Active, options.ContainerSelector, options.InitialFocusSelector, options.FallbackFocusSelector, options.RestoreFocus)
}

func useOverlayEscape(active bool, onDismiss func()) {
	UseEffect(func() func() {
		if !active || onDismiss == nil {
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
			event := args[0]
			if event.Get("key").String() == "Escape" {
				event.Call("preventDefault")
				onDismiss()
			}
			return nil
		})
		document.Call("addEventListener", "keydown", listener)
		return func() {
			document.Call("removeEventListener", "keydown", listener)
			listener.Release()
		}
	}, active)
}

func useOverlayScrollLock(active bool) {
	UseEffect(func() func() {
		if !active {
			return nil
		}
		overlayAcquireScrollLock()
		return func() {
			overlayReleaseScrollLock()
		}
	}, active)
}

func useOverlayBackgroundInert(selector string, active bool) {
	UseEffect(func() func() {
		if !active || selector == "" {
			return nil
		}
		overlayAcquireBackgroundInert(selector)
		return func() {
			overlayReleaseBackgroundInert(selector)
		}
	}, active, selector)
}

func focusElementValue(element js.Value, options ...FocusOptions) bool {
	if !element.Truthy() {
		return false
	}
	focus := element.Get("focus")
	if focus.Type() != js.TypeFunction {
		return false
	}
	preventScroll := false
	if len(options) > 0 {
		preventScroll = options[0].PreventScroll
	}
	if preventScroll {
		arg := js.Global().Get("Object").New()
		arg.Set("preventScroll", true)
		element.Call("focus", arg)
		return true
	}
	element.Call("focus")
	return true
}

func firstFocusableWithin(container js.Value) js.Value {
	items := focusableValues(container)
	if len(items) == 0 {
		return js.Undefined()
	}
	return items[0]
}

func focusableValues(container js.Value) []js.Value {
	if !container.Truthy() {
		return nil
	}
	list := container.Call("querySelectorAll", focusableSelector)
	if !list.Truthy() {
		return nil
	}
	length := list.Get("length").Int()
	values := make([]js.Value, 0, length)
	for index := 0; index < length; index++ {
		candidate := list.Index(index)
		if candidate.Truthy() {
			values = append(values, candidate)
		}
	}
	return values
}

func queryDocumentSelector(document js.Value, selector string) js.Value {
	if !document.Truthy() || selector == "" {
		return js.Undefined()
	}
	if strings.HasPrefix(selector, "#") && len(selector) > 1 {
		return document.Call("getElementById", selector[1:])
	}
	return document.Call("querySelector", selector)
}
