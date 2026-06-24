//go:build js && wasm

package css

import "syscall/js"

// defaultSink returns the build's default emission sink. On the wasm build it is
// the DOM sink that appends rules to a single managed <style> element in <head>.
func defaultSink() Sink { return domSinkInstance }

var domSinkInstance = &domSink{}

const styleElementID = "gwc-css"

// domSink appends each newly-registered rule to one managed <style> element. It
// is the v1 runtime emission target: idempotent per class (dedup happens in the
// registry), DOM-only, and a no-op when no document is present (e.g. a worker or
// the node-based wasm test runner) so it degrades gracefully.
type domSink struct {
	style js.Value // cached <style> element, set on first Emit
}

func (s *domSink) styleElement() js.Value {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return js.Undefined()
	}
	if s.style.Truthy() {
		return s.style
	}
	existing := document.Call("getElementById", styleElementID)
	if existing.Truthy() {
		s.style = existing
		return s.style
	}
	style := document.Call("createElement", "style")
	style.Set("id", styleElementID)
	style.Call("setAttribute", "data-gwc-css", "")
	head := document.Get("head")
	if !head.Truthy() {
		head = document.Get("documentElement")
	}
	if head.Truthy() {
		head.Call("appendChild", style)
	}
	s.style = style
	return s.style
}

func (s *domSink) Emit(parseClass string, parseCSS string) {
	style := s.styleElement()
	if !style.Truthy() {
		return // no DOM (worker/test host) — nothing to inject into.
	}
	current := style.Get("textContent").String()
	style.Set("textContent", current+parseCSS)
}

// Harvest returns the full text content of the managed <style> element so wasm
// callers and tests can read back what has been injected. Returns "" when no DOM
// or no style element exists.
func Harvest() string {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return ""
	}
	style := document.Call("getElementById", styleElementID)
	if !style.Truthy() {
		return ""
	}
	return style.Get("textContent").String()
}

// CriticalCSS returns the managed <style> element's text wrapped in a
// <style data-gwc-css> block on wasm, mirroring the native extraction name
// (CSS6). Returns "" when no DOM/style element exists.
func CriticalCSS() string {
	parseCSS := Harvest()
	if parseCSS == "" {
		return ""
	}
	return `<style data-gwc-css="">` + parseCSS + `</style>`
}

// SeedFromDocument pre-seeds the registry from a server-rendered
// <style data-gwc-css="..."> block's class list so already-present rules are
// recognized as hits and not re-injected during hydration. Safe to call when no
// such block exists.
func SeedFromDocument() {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	blocks := document.Call("querySelectorAll", "style[data-gwc-css]")
	if !blocks.Truthy() {
		return
	}
	length := blocks.Get("length").Int()
	for i := 0; i < length; i++ {
		block := blocks.Call("item", i)
		attr := block.Call("getAttribute", "data-gwc-css")
		if !attr.Truthy() {
			continue
		}
		for _, class := range splitFields(attr.String()) {
			Seed(class)
		}
	}
}

func splitFields(parseValue string) []string {
	var out []string
	current := ""
	for _, r := range parseValue {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current != "" {
				out = append(out, current)
				current = ""
			}
			continue
		}
		current += string(r)
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}
