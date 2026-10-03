package hotreload_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/hotreload"
)

// TestSchemaChangedDistinguishesKeysThatEmbedSeparators verifies a key that
// contains the fingerprint separators cannot impersonate two separate keys.
func TestSchemaChangedDistinguishesKeysThatEmbedSeparators(parseT *testing.T) {
	parseTwoKeys := map[string]any{"a": 1, "b": "x"}
	parseOneKey := map[string]any{"a:int;b": "x"}
	if !hotreload.SchemaChanged(parseTwoKeys, parseOneKey) {
		parseT.Fatalf("different shapes share fingerprint %q", hotreload.SchemaFingerprint(parseTwoKeys))
	}
}
