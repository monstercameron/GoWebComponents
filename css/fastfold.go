package css

// Fast fold: skipping canonicalize() on a repeat New(...).
//
// # Why this exists
//
// New's cache used to be consulted AFTER canonicalize(rules) had already run.
// canonicalize is the expensive half of a fold — it allocates a bucket map, an
// order slice and a seen-set, sorts, and serializes into a strings.Builder — so
// the "fast path" only ever skipped hashing and CSS-text building. New's own
// comment claimed a repeat fold "becomes a map lookup"; it did not.
//
// Profiling a real application (Atlas Commerce OS, ~29k lines, 398 design.Class
// call sites) put the typed-CSS fold at 46% of the native render path's CPU and
// 63% of its allocations, with a warm four-bundle fold still costing 23.9 µs and
// 26.7 KB — the cache was saving only ~23% of time and ~22% of bytes over a cold
// fold. That is the single largest item in the render profile, larger than
// reconciliation by two orders of magnitude.
//
// So: compute a cheap, allocation-free digest of the INPUT rules and consult the
// cache with that first. canonicalize only runs on a genuine first sighting.
//
// # Why the digest is order-SENSITIVE
//
// This is the trap, and it is worth stating plainly because "fold a set of rules"
// sounds order-independent and is not.
//
// canonicalize resolves conflicting declarations by argument order — it writes
// buckets[key][property] = value as it walks, so a later rule overwrites an
// earlier one for the same property. That is a deliberate, documented property:
// conflicts resolve in the order the caller wrote them, which is visible in the
// source, instead of by stylesheet emission order, which is visible to nobody.
//
// It means New(TextColor(Red), TextColor(Blue)) and New(TextColor(Blue),
// TextColor(Red)) canonicalize DIFFERENTLY. An order-independent digest (a sum or
// xor of per-rule hashes) would give both the same key and hand the second caller
// the first one's class — a wrong-colour bug that only appears when two rules in
// one fold touch the same property, i.e. exactly the case the ordering rule exists
// to serve.
//
// An order-sensitive digest cannot produce a false hit. The cost is that
// New(a, b) and New(b, a) occupy two fast-cache entries when they don't conflict;
// both then canonicalize to the same string and newCache returns the same Sheet,
// so the class is still shared and only the first fold of each ordering pays.
// That is the safe direction to be wrong in.

import "sync"

// foldKey is a content digest of a rule sequence. It is a comparable struct so it
// can key a sync.Map directly with no string building.
//
// Two independent accumulators plus the structural counts make an accidental
// collision between two different real rule-sets not a practical concern: a
// collision needs both 64-bit hashes to agree AND the rule count AND the total
// declaration count AND the byte length. h1/h2 use different offsets and primes so
// they do not move together.
type foldKey struct {
	h1        uint64
	h2        uint64
	ruleCount int
	declCount int
	byteLen   int
}

const (
	fnvOffset1 uint64 = 14695981039346656037
	fnvPrime1  uint64 = 1099511628211
	fnvOffset2 uint64 = 1469598103934665603
	fnvPrime2  uint64 = 16777619
)

// mixString folds a string into both accumulators. Reading s[i] does not allocate
// and does not copy the backing array.
func (k *foldKey) mixString(parseValue string) {
	for i := 0; i < len(parseValue); i++ {
		parseByte := uint64(parseValue[i])
		k.h1 = (k.h1 ^ parseByte) * fnvPrime1
		k.h2 = (k.h2 ^ parseByte) * fnvPrime2
	}
	k.byteLen += len(parseValue)
	// A separator so ("ab","c") and ("a","bc") cannot digest identically — without
	// it, concatenation ambiguity is a real collision source in a property/value
	// stream where both halves are attacker-adjacent author input.
	k.h1 = (k.h1 ^ 0xff) * fnvPrime1
	k.h2 = (k.h2 ^ 0xff) * fnvPrime2
}

// computeFoldKey digests a rule sequence in order, covering everything
// canonicalize looks at: the selector template, the at-rule stack, every
// declaration property and value, and any raw blocks that travel with a rule.
//
// If a field is ever added to Rule or scope that canonicalize reads, it MUST be
// mixed here too. A field that affects the canonical output but not the digest is
// a silent wrong-class bug, not a missed optimisation — that is the one way this
// file can hurt you.
func computeFoldKey(parseRules []Rule) foldKey {
	parseKey := foldKey{h1: fnvOffset1, h2: fnvOffset2}
	for parseIndex := range parseRules {
		parseRule := &parseRules[parseIndex]
		parseKey.ruleCount++
		parseKey.mixString(parseRule.scope.template())
		for _, parseAt := range parseRule.scope.atRules {
			parseKey.mixString(parseAt)
		}
		parseKey.declCount += len(parseRule.decls)
		// Precomputed hash when the constructor supplied one (decl() does, which is
		// every typed property), otherwise hash the declarations here.
		//
		// This is what makes the digest O(rules) rather than O(total bytes), and on
		// real bundles that is the whole difference: a four-bundle container in the
		// Atlas design system flattens to ~190 rules, so byte-wise hashing walked
		// thousands of characters on every fold of a rule-set that had already been
		// folded thousands of times.
		if parseRule.declHash != 0 {
			parseKey.h1 = (parseKey.h1 ^ parseRule.declHash) * fnvPrime1
			parseKey.h2 = (parseKey.h2 ^ parseRule.declHash) * fnvPrime2
			// Byte length still has to move, or two different rules with equal hashes
			// and equal counts would be indistinguishable on the structural fields.
			parseKey.byteLen += int(parseRule.declHash & 0xffff)
		} else {
			for _, parseDecl := range parseRule.decls {
				parseKey.mixString(parseDecl.property)
				parseKey.mixString(parseDecl.value)
			}
		}
		for _, parseRaw := range parseRule.raw {
			parseKey.mixString(parseRaw)
		}
	}
	return parseKey
}

// foldCache maps a rule-sequence digest to its folded Sheet, letting a repeat
// New(...) skip canonicalize entirely. Reset clears it alongside newCache and the
// registry — a stale entry here would survive a Reset and hand out a class whose
// CSS is no longer registered.
var foldCache sync.Map
