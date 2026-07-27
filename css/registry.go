package css

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"

	"github.com/monstercameron/GoWebComponents/v5/diagnostics"
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

	// classFingerprints records a content hash of the CSS emitted under each
	// class, so a hash COLLISION — two distinct rule-sets whose class name hashes
	// to the same value — is detected instead of silently applying the first
	// rule-set's styles to the second's element. Only populated by registerAndEmit
	// (not Seed, which knows the class but not its CSS). Guarded by registryMu.
	classFingerprints = map[string]uint64{}

	// The class registry only grows (a class, once emitted, must stay registered
	// so its rule is not re-emitted). Building classes from RUNTIME values — instead
	// of routing live values through the Dynamic escape valve (var(--name) + inline
	// value) — mints a new class per distinct value and grows the registry (and the
	// live <style>) without bound. When the count crosses the churn threshold, warn
	// ONCE so the developer can find the leak. Vars are test seams.
	classRegistryChurnThreshold = 10000
	classChurnWarned            bool // guarded by registryMu
	reportClassRegistryChurn    = func(parseCount int) {
		diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
			Code:     "GWC-CSS-CLASS-CHURN",
			Headline: fmt.Sprintf("css class registry exceeded %d unique classes", classRegistryChurnThreshold),
			Summary:  fmt.Sprintf("the css class registry has grown to %d classes; this usually means classes are minted from runtime values on every render", parseCount),
			Next:     "route live values through css.DynamicLength / css.DynamicVar (a stable var(--…) class + inline value) instead of building a new class per value",
		}))
	}
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

// registeredClasses returns every class currently known to the registry (emitted
// or seeded), sorted for determinism. Build-tag-free so the wasm sink can expose
// HarvestedClasses() with the same shape as the native buffer sink (which keeps its
// own ordered list); the shared registry is the cross-target source of truth for
// "which classes are present".
func registeredClasses() []string {
	registryMu.Lock()
	defer registryMu.Unlock()
	out := make([]string, 0, len(registry))
	for class := range registry {
		out = append(out, class)
	}
	sort.Strings(out)
	return out
}

// Reset clears the registry and resets the active sink to the build default. For
// tests — it gives each test a clean process-wide registry.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]bool{}
	classFingerprints = map[string]uint64{}
	classChurnWarned = false
	activeSink = defaultSink()
	newCache.Clear()
	// foldCache must be cleared with newCache, never separately. It maps an input
	// digest straight to a Sheet, so an entry surviving a Reset would hand back a
	// class whose CSS is no longer in the registry and no longer emitted — styling
	// that silently vanishes while the class name still appears in the markup.
	foldCache.Clear()
	if r, ok := activeSink.(interface{ reset() }); ok {
		r.reset()
	}
}

// registerAndEmit registers the class if new and emits its CSS through the active
// sink exactly once. Returns true if this call performed the emission.
func registerAndEmit(parseClass, parseCSS string) bool {
	parseFingerprint := cssContentFingerprint(parseCSS)
	registryMu.Lock()
	if registry[parseClass] {
		// A class already emitted is normally a benign fold-cache miss for an
		// identical rule-set. Verify the CSS actually matches; a mismatch means a
		// class-name hash COLLISION between two distinct rule-sets, which would
		// otherwise silently give the second element the first's styles.
		parsePrevFingerprint, parseHasFingerprint := classFingerprints[parseClass]
		registryMu.Unlock()
		if parseHasFingerprint && parsePrevFingerprint != parseFingerprint {
			reportClassHashCollision(parseClass)
		}
		return false
	}
	registry[parseClass] = true
	classFingerprints[parseClass] = parseFingerprint
	parseCount := len(registry)
	parseShouldWarn := !classChurnWarned && parseCount >= classRegistryChurnThreshold
	if parseShouldWarn {
		classChurnWarned = true
	}
	sink := activeSink
	registryMu.Unlock()
	if parseShouldWarn {
		reportClassRegistryChurn(parseCount)
	}
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

// cssContentFingerprint is a 64-bit content hash of emitted CSS, used only to
// detect a class-name hash collision (same class, different CSS). It is a
// separate hash from the class name, so a false collision report would require
// BOTH the class hash AND this fingerprint to collide — effectively impossible.
func cssContentFingerprint(parseCSS string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(parseCSS))
	return h.Sum64()
}

// reportClassHashCollision warns (once-loud, fail-visible) that two distinct
// rule-sets produced the same class name — the second rule-set's CSS was NOT
// emitted, so its element renders with the first rule-set's styles. A test seam.
var reportClassHashCollision = func(parseClass string) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Code:     "GWC-CSS-CLASS-COLLISION",
		Headline: fmt.Sprintf("css class name collision on %q", parseClass),
		Summary:  fmt.Sprintf("two distinct rule-sets hashed to class %q with different CSS; the second rule-set's styles were dropped and its element will render with the first's styles", parseClass),
		Next:     "this is an astronomically rare hash collision — vary one of the colliding rule-sets slightly (e.g. add a no-op declaration) to force a different class name",
	}))
}
