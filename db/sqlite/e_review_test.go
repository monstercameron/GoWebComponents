package sqlite

import "testing"

// TestDefaultPragmasHonorsCaseInsensitiveOverride verifies a caller pragma that
// differs from a default only by case replaces the default; PRAGMA names are
// case-insensitive, so keeping both lets the default silently win the sort order.
func TestDefaultPragmasHonorsCaseInsensitiveOverride(parseT *testing.T) {
	parseMerged := defaultPragmas(map[string]string{"Foreign_Keys": "OFF"})
	if _, hasDefault := parseMerged["foreign_keys"]; hasDefault {
		parseT.Fatalf("default foreign_keys must be dropped when overridden by a differently cased key: %v", parseMerged)
	}
	if parseMerged["Foreign_Keys"] != "OFF" {
		parseT.Fatalf("caller override lost: %v", parseMerged)
	}
	if parseMerged["busy_timeout"] != "5000" {
		parseT.Fatalf("untouched default lost: %v", parseMerged)
	}
}
