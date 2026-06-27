package hotreload

import (
	"reflect"
	"sort"
	"strings"
)

// SchemaFingerprint returns a stable fingerprint of a state snapshot's SHAPE — its sorted
// keys and each value's concrete type — independent of the actual values. Two snapshots with
// the same keys and per-key types produce the same fingerprint; adding/removing a key or
// changing a field's type changes it.
func SchemaFingerprint(parseSnapshot map[string]any) string {
	parseKeys := make([]string, 0, len(parseSnapshot))
	for parseKey := range parseSnapshot {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)

	var parseBuilder strings.Builder
	for parseIndex, parseKey := range parseKeys {
		if parseIndex > 0 {
			parseBuilder.WriteByte(';')
		}
		parseBuilder.WriteString(parseKey)
		parseBuilder.WriteByte(':')
		parseBuilder.WriteString(typeOf(parseSnapshot[parseKey]))
	}
	return parseBuilder.String()
}

// typeOf renders a value's concrete type for the fingerprint ("nil" for a nil value).
func typeOf(parseValue any) string {
	if parseValue == nil {
		return "nil"
	}
	return reflect.TypeOf(parseValue).String()
}

// SchemaChanged reports whether a persisted snapshot's shape differs from the current code's
// shape. This is the C1 ghost-bug guard: when it is true, a state-preserving hot reload must
// surface a visible "state reset" rather than silently restoring the persisted snapshot into
// a mismatched type (which produces stale-state ghost bugs). When false, the snapshot is
// shape-compatible and safe to restore.
func SchemaChanged(parsePersisted, parseCurrent map[string]any) bool {
	return SchemaFingerprint(parsePersisted) != SchemaFingerprint(parseCurrent)
}
