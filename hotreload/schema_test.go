package hotreload_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/hotreload"
)

// TestSchemaFingerprintStableForSameShape proves identical shapes (keys + types) fingerprint
// the same regardless of the actual values or key insertion order.
func TestSchemaFingerprintStableForSameShape(parseT *testing.T) {
	parseA := map[string]any{"count": 1, "name": "ada", "active": true}
	parseB := map[string]any{"name": "bob", "active": false, "count": 99}
	if hotreload.SchemaFingerprint(parseA) != hotreload.SchemaFingerprint(parseB) {
		parseT.Fatalf("same shape should fingerprint equal:\n %q\n %q",
			hotreload.SchemaFingerprint(parseA), hotreload.SchemaFingerprint(parseB))
	}
	if hotreload.SchemaChanged(parseA, parseB) {
		parseT.Fatal("same shape should not be reported as changed")
	}
}

// TestSchemaChangedDetectsTypeChange proves a field whose type changed (the ghost-bug case)
// is detected, so a reload can reset instead of restoring a mismatched value.
func TestSchemaChangedDetectsTypeChange(parseT *testing.T) {
	parsePersisted := map[string]any{"count": 1}   // was an int
	parseCurrent := map[string]any{"count": "one"} // now a string
	if !hotreload.SchemaChanged(parsePersisted, parseCurrent) {
		parseT.Fatal("a changed field type must be detected as a schema change")
	}
}

// TestSchemaChangedDetectsAddedAndRemovedKeys proves added/removed state keys are detected.
func TestSchemaChangedDetectsAddedAndRemovedKeys(parseT *testing.T) {
	parseOld := map[string]any{"a": 1}
	parseAdded := map[string]any{"a": 1, "b": 2}
	if !hotreload.SchemaChanged(parseOld, parseAdded) {
		parseT.Fatal("an added key must be detected as a schema change")
	}
	if !hotreload.SchemaChanged(parseAdded, parseOld) {
		parseT.Fatal("a removed key must be detected as a schema change")
	}
}

// TestSchemaChangedNilValueDistinctFromTyped proves a nil value and a typed value differ.
func TestSchemaChangedNilValueDistinctFromTyped(parseT *testing.T) {
	if !hotreload.SchemaChanged(map[string]any{"x": nil}, map[string]any{"x": 0}) {
		parseT.Fatal("nil vs a typed value should be a schema change")
	}
}
