//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v7/interop"
)

// elementValueForRef returns the live js.Value for a DOMRef, and whether it is
// usable. It bridges runtime.DOMNode (via its Value() accessor) to syscall/js
// without ui importing the platform adapter directly.
func elementValueForRef(parseRef DOMRef) (js.Value, bool) {
	parseNode := parseRef.Node()
	if parseNode == nil {
		return js.Value{}, false
	}
	parseValuer, parseOk := parseNode.(interface{ Value() js.Value })
	if !parseOk {
		return js.Value{}, false
	}
	parseValue := parseValuer.Value()
	if !parseValue.Truthy() {
		return js.Value{}, false
	}
	return parseValue, true
}

func elementForRef(parseRef DOMRef) (interop.Element, bool) {
	parseValue, parseOk := elementValueForRef(parseRef)
	if !parseOk {
		return interop.Element{}, false
	}
	return interop.WrapElement(parseValue), true
}

// UseElementGeometry returns the referenced element's bounding box, measured
// after mount and kept up to date with a ResizeObserver (G26). It runs as a
// layout effect (post-mutation, pre-paint) so the first measurement is available
// without a flash. Returns the zero Rect before mount / on native.
//
//	r := ui.UseDOMRef()
//	box := ui.UseElementGeometry(r)   // box.Width, box.Height, ...
//	return shorthand.Div(shorthand.Ref(r))
func UseElementGeometry(parseRef DOMRef) interop.Rect {
	parseRect := UseState(interop.Rect{})
	UseLayoutEffect(func() func() {
		parseElement, parseOk := elementForRef(parseRef)
		if !parseOk {
			return nil
		}
		if parseMeasured, parseErr := parseElement.BoundingClientRect(); parseErr == nil {
			parseRect.Set(parseMeasured)
		}
		parseSubscription, parseErr := parseElement.ObserveResize(func(parseEntry interop.ResizeEntry) {
			if parseRemeasured, parseErr2 := parseElement.BoundingClientRect(); parseErr2 == nil {
				parseRect.Set(parseRemeasured)
				return
			}
			parseRect.Set(parseEntry.ContentRect)
		})
		if parseErr != nil {
			return nil
		}
		return func() { parseSubscription.Cancel() }
	}, useElementHookSentinel)
	return parseRect.Get()
}

// UseIntersection reports whether the referenced element currently intersects the
// viewport (or the configured root), updating reactively via IntersectionObserver
// (G27). Returns false before mount / on native.
//
//	visible := ui.UseIntersection(ref)   // lazy-load when true
func UseIntersection(parseRef DOMRef, parseOptions ...interop.IntersectionObserverOptions) bool {
	parseVisible := UseState(false)
	UseEffect(func() func() {
		parseElement, parseOk := elementForRef(parseRef)
		if !parseOk {
			return nil
		}
		parseSubscription, parseErr := parseElement.ObserveIntersection(func(parseEntry interop.IntersectionEntry) {
			parseVisible.Set(parseEntry.IsIntersecting)
		}, parseOptions...)
		if parseErr != nil {
			return nil
		}
		return func() { parseSubscription.Cancel() }
	}, useElementHookSentinel)
	return parseVisible.Get()
}

// UseAnimationRestart re-triggers a CSS keyframe animation on the referenced
// element whenever deps change, using the browser double-rAF idiom (remove class
// → force reflow → next frame re-add) so the animation replays even when the
// class was already present (G27). No-op before mount / on native.
//
//	ui.UseAnimationRestart(ref, "page-enter", props.ActivePath)
func UseAnimationRestart(parseRef DOMRef, parseClassName string, parseDeps ...any) {
	if len(parseDeps) == 0 {
		parseDeps = []any{useElementHookSentinel}
	}
	UseLayoutEffect(func() func() {
		parseValue, parseOk := elementValueForRef(parseRef)
		if !parseOk || parseClassName == "" {
			return nil
		}
		parseClassList := parseValue.Get("classList")
		if !parseClassList.Truthy() {
			return nil
		}
		parseClassList.Call("remove", parseClassName)
		// Force a reflow so the removal is committed before re-adding.
		_ = parseValue.Get("offsetWidth")

		var parseCancelInner func()
		parseCancelOuter := interop.RequestAnimationFrame(func(float64) {
			parseCancelInner = interop.RequestAnimationFrame(func(float64) {
				defer runtime.RecoverContainedPanic("ui", "UseAnimationRestart")
				parseClassList.Call("add", parseClassName)
			})
		})
		return func() {
			if parseCancelOuter != nil {
				parseCancelOuter()
			}
			if parseCancelInner != nil {
				parseCancelInner()
			}
		}
	}, parseDeps...)
}

// UsePointerEvents attaches pointer listeners to the referenced element for as
// long as the component is mounted, with managed js.Func lifetime (G35). It is
// the Go-native path for drag/pan/zoom canvases that previously dropped to an
// eval'd JS blob.
//
//	ui.UsePointerEvents(canvasRef, ui.PointerHandlers{
//	    OnPointerDown: func(e ui.Event) { ... },
//	    OnPointerMove: func(e ui.Event) { ... },
//	})
func UsePointerEvents(parseRef DOMRef, parseHandlers PointerHandlers) {
	// Latest-ref: listeners are bound once at mount but must run the handlers from the most
	// recent render, not the closures captured when they were bound.
	parseLatest := UseRef(parseHandlers)
	parseLatest.Set(parseHandlers)
	UseEffect(func() func() {
		parseValue, parseOk := elementValueForRef(parseRef)
		if !parseOk {
			return nil
		}
		parseUnbinds := []func(){
			bindElementEvent(parseValue, "pointerdown", dispatchLatestPointerHandler(parseHandlers.OnPointerDown, parseLatest, func(parseH PointerHandlers) func(Event) { return parseH.OnPointerDown })),
			bindElementEvent(parseValue, "pointermove", dispatchLatestPointerHandler(parseHandlers.OnPointerMove, parseLatest, func(parseH PointerHandlers) func(Event) { return parseH.OnPointerMove })),
			bindElementEvent(parseValue, "pointerup", dispatchLatestPointerHandler(parseHandlers.OnPointerUp, parseLatest, func(parseH PointerHandlers) func(Event) { return parseH.OnPointerUp })),
			bindElementEvent(parseValue, "pointercancel", dispatchLatestPointerHandler(parseHandlers.OnPointerCancel, parseLatest, func(parseH PointerHandlers) func(Event) { return parseH.OnPointerCancel })),
		}
		return func() {
			for _, parseUnbind := range parseUnbinds {
				if parseUnbind != nil {
					parseUnbind()
				}
			}
		}
	}, useElementHookSentinel)
}

// UseWheel attaches a wheel listener to the referenced element with managed
// js.Func lifetime (G35). No-op before mount / on native.
func UseWheel(parseRef DOMRef, parseHandler func(Event)) {
	// Latest-ref: the listener runs the handler from the most recent render.
	parseLatest := UseRef(parseHandler)
	parseLatest.Set(parseHandler)
	UseEffect(func() func() {
		parseValue, parseOk := elementValueForRef(parseRef)
		if !parseOk {
			return nil
		}
		return bindElementEvent(parseValue, "wheel", dispatchLatestHandler(parseHandler, parseLatest))
	}, useElementHookSentinel)
}

// dispatchLatestHandler wraps parseBound so the listener invokes whatever handler the latest-ref
// currently holds. It returns nil when parseBound is nil, so an unset handler is never bound.
func dispatchLatestHandler(parseBound func(Event), parseLatest Ref[func(Event)]) func(Event) {
	if parseBound == nil {
		return nil
	}
	return func(parseEvent Event) {
		if parseCurrent := parseLatest.Get(); parseCurrent != nil {
			parseCurrent(parseEvent)
		}
	}
}

// dispatchLatestPointerHandler is dispatchLatestHandler for one field of the latest PointerHandlers.
func dispatchLatestPointerHandler(parseBound func(Event), parseLatest Ref[PointerHandlers], parsePick func(PointerHandlers) func(Event)) func(Event) {
	if parseBound == nil {
		return nil
	}
	return func(parseEvent Event) {
		if parseCurrent := parsePick(parseLatest.Get()); parseCurrent != nil {
			parseCurrent(parseEvent)
		}
	}
}

// bindElementEvent attaches one managed listener to an element js.Value and
// returns an unbind that removes it and releases the js.Func. Returns nil when
// handler is nil.
func bindElementEvent(parseValue js.Value, parseEventType string, parseHandler func(Event)) func() {
	if parseHandler == nil {
		return nil
	}
	parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "element "+parseEventType+" callback")
		var parseEvent js.Value
		if len(parseArgs) > 0 {
			parseEvent = parseArgs[0]
		}
		parseHandler(runtime.NewGoEvent(parseEvent))
		return nil
	})
	parseValue.Call("addEventListener", parseEventType, parseListener)
	return func() {
		parseValue.Call("removeEventListener", parseEventType, parseListener)
		parseListener.Release()
	}
}

const useElementHookSentinel = "gwc-element-hook"
