//go:build !js || !wasm

package otel

import "sync"

// OverflowValue is the attribute value that replaces every value seen after a
// BoundedSet is full.
const OverflowValue = "_other"

// DefaultAttributeCap is the number of distinct values a metric attribute may take
// before further values collapse into OverflowValue.
const DefaultAttributeCap = 200

// BoundedSet admits at most a fixed number of distinct values and maps every later
// new value to OverflowValue. It keeps metric attribute cardinality bounded when
// the value comes from request data such as a route or a function name. The zero
// value is not usable; build one with NewBoundedSet.
type BoundedSet struct {
	mu     sync.Mutex
	limit  int
	values map[string]struct{}
}

// NewBoundedSet builds a BoundedSet admitting parseLimit distinct values; a
// non-positive limit selects DefaultAttributeCap.
func NewBoundedSet(parseLimit int) *BoundedSet {
	if parseLimit <= 0 {
		parseLimit = DefaultAttributeCap
	}
	return &BoundedSet{limit: parseLimit, values: make(map[string]struct{}, parseLimit)}
}

// Bound returns parseValue when it is already admitted or there is room to admit
// it, and OverflowValue otherwise. The overflow marker itself is always allowed.
func (parseSet *BoundedSet) Bound(parseValue string) string {
	parseSet.mu.Lock()
	defer parseSet.mu.Unlock()
	if _, parseKnown := parseSet.values[parseValue]; parseKnown {
		return parseValue
	}
	if len(parseSet.values) >= parseSet.limit {
		return OverflowValue
	}
	parseSet.values[parseValue] = struct{}{}
	return parseValue
}

// Len reports how many distinct values have been admitted.
func (parseSet *BoundedSet) Len() int {
	parseSet.mu.Lock()
	defer parseSet.mu.Unlock()
	return len(parseSet.values)
}
