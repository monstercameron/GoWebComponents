package kvstate

import (
	"fmt"
	"hash/fnv"
	"sync"
	"testing"
)

// sanitizeHashFor returns the collision-guard suffix sanitizeIdent appends to
// names that needed punctuation replaced.
func sanitizeHashFor(parseName string) string {
	parseHash := fnv.New32a()
	_, _ = parseHash.Write([]byte(parseName))
	return fmt.Sprintf("%08x", parseHash.Sum32())
}

// TestSanitizeIdentKeepsDistinctNamesDistinct pins that punctuation variants no
// longer collapse onto one table, while clean names (existing databases) are
// returned unchanged.
func TestSanitizeIdentKeepsDistinctNamesDistinct(parseT *testing.T) {
	parseInputs := []string{"a-b", "a_b", "a b", "a.b", "a/b", "a$b", "a-b-", "a-b_", "caf\u00e9", "cafe", "!!!", "___"}
	parseSeen := map[string]string{}
	for _, parseIn := range parseInputs {
		parseOut := sanitizeIdent(parseIn)
		if parsePrev, parseDup := parseSeen[parseOut]; parseDup {
			parseT.Fatalf("sanitizeIdent collision: %q and %q both map to %q", parsePrev, parseIn, parseOut)
		}
		parseSeen[parseOut] = parseIn
	}
	for _, parseClean := range []string{"a_b", "gwc", "My_Table9", "_x"} {
		if parseGot := sanitizeIdent(parseClean); parseGot != parseClean {
			parseT.Fatalf("clean name %q changed to %q; existing databases would lose their table", parseClean, parseGot)
		}
	}
}

// TestSanitizeIdentRejectsLeadingDigit pins that a sanitized SQL identifier never
// begins with a digit. The result is interpolated UNQUOTED into DDL/DML, and
// "123abc" as an unquoted identifier is a syntax error at CREATE TABLE time.
func TestSanitizeIdentRejectsLeadingDigit(parseT *testing.T) {
	parseCases := map[string]string{
		"123abc":    "_123abc",
		"9":         "_9",
		"user data": "user_data_" + sanitizeHashFor("user data"),
		"table":     "table",
		"a_b":       "a_b",
		"":          "gwc_state",
		"!!!":       "____" + sanitizeHashFor("!!!"),
	}
	for parseIn, parseWant := range parseCases {
		if parseGot := sanitizeIdent(parseIn); parseGot != parseWant {
			parseT.Fatalf("sanitizeIdent(%q) = %q, want %q", parseIn, parseGot, parseWant)
		}
		parseGot := sanitizeIdent(parseIn)
		if parseGot[0] >= '0' && parseGot[0] <= '9' {
			parseT.Fatalf("sanitizeIdent(%q) = %q begins with a digit", parseIn, parseGot)
		}
	}
}

// TestNextVersionIsAtomicUnderConcurrency pins that concurrent version claims are
// distinct and monotonic. The previous getVersion()+1 / setVersion() pair let two
// writers claim the same version and clobber each other; nextVersion() increments
// under the lock so N concurrent callers receive exactly 1..N with no duplicates.
func TestNextVersionIsAtomicUnderConcurrency(parseT *testing.T) {
	parseShared := &boundAtomState{}
	const parseN = 200
	parseSeen := make([]int64, parseN)
	var parseWg sync.WaitGroup
	parseWg.Add(parseN)
	for parseI := 0; parseI < parseN; parseI++ {
		go func(parseIdx int) {
			defer parseWg.Done()
			parseSeen[parseIdx] = parseShared.nextVersion()
		}(parseI)
	}
	parseWg.Wait()

	parseDistinct := map[int64]bool{}
	var parseMax int64
	for _, parseV := range parseSeen {
		if parseDistinct[parseV] {
			parseT.Fatalf("version %d claimed more than once (lost-update race)", parseV)
		}
		parseDistinct[parseV] = true
		if parseV > parseMax {
			parseMax = parseV
		}
	}
	if len(parseDistinct) != parseN {
		parseT.Fatalf("expected %d distinct versions, got %d", parseN, len(parseDistinct))
	}
	if parseMax != parseN {
		parseT.Fatalf("expected max version %d, got %d", parseN, parseMax)
	}
}
