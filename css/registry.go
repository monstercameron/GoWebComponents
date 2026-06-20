package css

import (
	"hash/fnv"
	"strconv"
	"sync"
)

// Sink is the documented emission seam. New(...) calls Emit exactly once per
// newly-registered class (the dedup happens before the sink is touched), so a
// Sink never sees the same class twice within one process. The runtime wasm sink
// appends to a managed <style> element; the native sink buffers for SSR/tests;
// a build-time extraction pass would be a third implementation.
type Sink interface {
	Emit(parseClass string, parseCSS string)
}

var (
	registryMu sync.Mutex
	registry        = map[string]bool{} // class -> already emitted
	activeSink Sink = defaultSink()
)

// SetSink swaps the active emission sink and returns the previous one. Mainly for
// tests and for wiring a custom emission target. Passing nil restores the
// build's default sink.
func SetSink(parseSink Sink) Sink {
	registryMu.Lock()
	defer registryMu.Unlock()
	previous := activeSink
	if parseSink == nil {
		activeSink = defaultSink()
	} else {
		activeSink = parseSink
	}
	return previous
}

// Seed marks a class as already-present without emitting it. Client hydration
// uses it to pre-seed the registry from server-rendered class names so existing
// rules are recognized as hits and not re-injected.
func Seed(parseClasses ...string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, class := range parseClasses {
		registry[class] = true
	}
}

// Reset clears the registry and resets the active sink to the build default. For
// tests — it gives each test a clean process-wide registry.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]bool{}
	activeSink = defaultSink()
	newCache.Clear()
	if r, ok := activeSink.(interface{ reset() }); ok {
		r.reset()
	}
}

// registerAndEmit registers the class if new and emits its CSS through the active
// sink exactly once. Returns true if this call performed the emission.
func registerAndEmit(parseClass, parseCSS string) bool {
	registryMu.Lock()
	if registry[parseClass] {
		registryMu.Unlock()
		return false
	}
	registry[parseClass] = true
	sink := activeSink
	registryMu.Unlock()
	sink.Emit(parseClass, parseCSS)
	return true
}

// classPrefix is the stable prefix for every generated class name.
const classPrefix = "c-"

// hashClass content-hashes the canonical serialization into a short, stable,
// collision-resistant class name.
func hashClass(parseCanonical string) string {
	return classPrefix + shortHash(parseCanonical)
}

// shortHash is a 64-bit FNV-1a hash rendered in base-36 — short, URL/class-safe,
// and deterministic across runs and platforms.
func shortHash(parseText string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(parseText))
	return strconv.FormatUint(h.Sum64(), 36)
}
