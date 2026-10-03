package escalate

import (
	"testing"
	"unicode/utf8"
)

// TestClassifySQLNeverCallsAWriteARead pins the direction that matters: a write
// classified as a read becomes a command a caller believes is safe to retry.
func TestClassifySQLNeverCallsAWriteARead(parseT *testing.T) {
	parseCases := map[string]string{
		"cte delete across lines": "WITH stale AS (SELECT id FROM t)\nDELETE\nFROM t WHERE id IN stale",
		"cte insert tab":          "WITH s AS (SELECT 1) INSERT\tINTO t SELECT * FROM s",
		"cte update after paren":  "WITH s AS (SELECT 1)\n(UPDATE t SET a = 1)",
		"pragma set":              "PRAGMA journal_mode = WAL",
		"pragma set no spaces":    "PRAGMA foreign_keys=ON",
		"stacked write":           "SELECT 1; DROP TABLE t",
		"stacked write newline":   "SELECT 1;\nDELETE FROM t",
	}
	for parseName, parseSQL := range parseCases {
		if parseGot := ClassifySQL(parseSQL); parseGot != AccessWrite {
			parseT.Errorf("%s: ClassifySQL(%q) = %s, want write", parseName, parseSQL, parseGot)
		}
	}
	parseReads := map[string]string{
		"plain select":    "SELECT * FROM t",
		"pragma query":    "PRAGMA table_info(t)",
		"trailing semi":   "SELECT 1;",
		"cte select":      "WITH s AS (SELECT 1) SELECT * FROM s",
		"semi in literal": "SELECT 'a;b' FROM t",
	}
	for parseName, parseSQL := range parseReads {
		if parseGot := ClassifySQL(parseSQL); parseGot != AccessRead {
			parseT.Errorf("%s: ClassifySQL(%q) = %s, want read", parseName, parseSQL, parseGot)
		}
	}
}

// TestProposedNamesStayValidUTF8 pins that a non-ASCII table name cannot yield a
// proposed identifier with a split multi-byte rune.
func TestProposedNamesStayValidUTF8(parseT *testing.T) {
	parseFindings, parseErr := AnalyzeSource("x.go", "package x\nfunc f() { db.Query(\"SELECT * FROM \u00fcber_tabelle WHERE a = ?\", 1) }\n", Options{})
	if parseErr != nil {
		parseT.Fatalf("AnalyzeSource: %v", parseErr)
	}
	if len(parseFindings) != 1 {
		parseT.Fatalf("findings = %d, want 1", len(parseFindings))
	}
	if !utf8.ValidString(parseFindings[0].ProposedSignature) || parseFindings[0].ProposedCommand != "List\u00dcberTabelle" {
		parseT.Fatalf("ProposedCommand = %q, want %q", parseFindings[0].ProposedCommand, "List\u00dcberTabelle")
	}
}
