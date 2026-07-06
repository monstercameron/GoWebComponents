//go:build !(js && wasm)

package css

import (
	"html"
	"strings"
	"sync"
)

// defaultSink returns the build's default emission sink. On native (SSR + tests)
// it is the package-wide buffer sink that records emitted rules in order so SSR
// and tests can read them back.
func defaultSink() Sink { return bufferSinkInstance }

var bufferSinkInstance = &bufferSink{}

// bufferSink collects emitted CSS in registration order. It is the in-memory
// buffer the SSR path serializes into the document head and that native tests
// harvest to assert on emitted output.
type bufferSink struct {
	mu      sync.Mutex
	order   []string          // class names in emission order
	byClass map[string]string // class -> css text
}

func (s *bufferSink) Emit(parseClass string, parseCSS string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byClass == nil {
		s.byClass = map[string]string{}
	}
	if _, ok := s.byClass[parseClass]; ok {
		return
	}
	s.byClass[parseClass] = parseCSS
	s.order = append(s.order, parseClass)
}

func (s *bufferSink) harvest() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	for _, class := range s.order {
		b.WriteString(s.byClass[class])
	}
	return b.String()
}

func (s *bufferSink) classes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.order))
	copy(out, s.order)
	return out
}

func (s *bufferSink) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.order = nil
	s.byClass = nil
}

// Harvest returns all CSS emitted into the native buffer sink so far, concatenated
// in emission order. SSR serializes this into a <style> block; tests assert on it.
//
// Harvest reads the active sink only when it is the default buffer sink. If a test
// installed a custom sink via SetSink, harvest its own state instead.
func Harvest() string { return bufferSinkInstance.harvest() }

// HarvestedClasses returns the class names emitted into the native buffer sink, in
// emission order. Used to seed the client registry for hydration.
func HarvestedClasses() []string { return bufferSinkInstance.classes() }

// CriticalCSS returns the critical-CSS <style> block to inline into an SSR
// document head (CSS6). It is the extraction half of the author-in-Go / ship-
// inline pipeline: render the app under the native sink (which collects every
// emitted rule), call CriticalCSS to serialize it into the page, and on the
// client call SeedFromDocument so the same rules are recognized as already
// present and never re-injected. CriticalCSS is an alias of StyleBlock with the
// pipeline-oriented name; both return "" when nothing has been emitted.
func CriticalCSS() string { return StyleBlock() }

// SeedFromDocument is a no-op on native builds: there is no DOM/document to read a
// server-rendered <style data-gwc-css> block from. It exists for cross-target API
// parity so code that hydrates the client registry compiles on both targets (the
// wasm build does the real seeding).
func SeedFromDocument() {}

// StyleBlock returns the harvested CSS wrapped in a <style data-gwc-css> element
// carrying the emitted class names in a data attribute. The SSR head includes
// this so styles are present on first paint and the client can pre-seed its
// registry to suppress re-injection. Returns "" when nothing has been emitted.
func StyleBlock() string {
	cssText := bufferSinkInstance.harvest()
	if cssText == "" {
		return ""
	}
	classes := strings.Join(bufferSinkInstance.classes(), " ")
	return `<style data-gwc-css="` + html.EscapeString(classes) + `">` + cssText + `</style>`
}
