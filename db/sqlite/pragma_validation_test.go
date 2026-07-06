package sqlite

import "testing"

// TestPragmaValidationRejectsInjection pins that pragma keys/values which could
// smuggle extra SQL are rejected, while all legitimate pragma forms are allowed.
func TestPragmaValidationRejectsInjection(parseT *testing.T) {
	parseValidKeys := []string{"journal_mode", "busy_timeout", "cache_size", "foreign_keys"}
	for _, parseKey := range parseValidKeys {
		if !isValidPragmaKey(parseKey) {
			parseT.Fatalf("legit pragma key %q rejected", parseKey)
		}
	}
	parseBadKeys := []string{"", "journal_mode; DROP TABLE x", "foo bar", "1abc", `a"b`, "a=b"}
	for _, parseKey := range parseBadKeys {
		if isValidPragmaKey(parseKey) {
			parseT.Fatalf("injection pragma key %q accepted", parseKey)
		}
	}

	parseValidValues := []string{"WAL", "NORMAL", "MEMORY", "ON", "OFF", "5000", "-2000"}
	for _, parseValue := range parseValidValues {
		if !isValidPragmaValue(parseValue) {
			parseT.Fatalf("legit pragma value %q rejected", parseValue)
		}
	}
	parseBadValues := []string{"", "WAL; DROP TABLE x", "5000 OR 1=1", `'x'`, "a(b)", "a b"}
	for _, parseValue := range parseBadValues {
		if isValidPragmaValue(parseValue) {
			parseT.Fatalf("injection pragma value %q accepted", parseValue)
		}
	}
}
