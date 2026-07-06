package kvstate

import (
	"sync"
	"testing"
)

// TestSanitizeIdentRejectsLeadingDigit pins that a sanitized SQL identifier never
// begins with a digit. The result is interpolated UNQUOTED into DDL/DML, and
// "123abc" as an unquoted identifier is a syntax error at CREATE TABLE time.
func TestSanitizeIdentRejectsLeadingDigit(parseT *testing.T) {
	parseCases := map[string]string{
		"123abc":    "_123abc",
		"9":         "_9",
		"user data": "user_data",
		"table":     "table",
		"":          "gwc_state",
		"!!!":       "___",
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
