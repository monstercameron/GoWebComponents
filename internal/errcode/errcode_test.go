package errcode

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestRegistryEntriesAreComplete checks every entry for the fields the docs
// generator and runtime reports depend on.
func TestRegistryEntriesAreComplete(parseT *testing.T) {
	parseAll := All()
	if len(parseAll) < 30 {
		parseT.Fatalf("expected the full code set, got %d", len(parseAll))
	}
	if !sort.SliceIsSorted(parseAll, func(parseA, parseB int) bool { return parseAll[parseA].Code < parseAll[parseB].Code }) {
		parseT.Fatal("All() is not sorted by code")
	}
	for _, parseEntry := range parseAll {
		if !strings.HasPrefix(string(parseEntry.Code), "GWC-") || parseEntry.Family == "" || parseEntry.Headline == "" || parseEntry.Remediation == "" || parseEntry.Severity == "" {
			parseT.Fatalf("incomplete entry: %+v", parseEntry)
		}
		parseWant := ReferencePagePath + "#" + strings.ToLower(string(parseEntry.Code))
		if parseEntry.Docs != parseWant {
			parseT.Fatalf("%s docs = %q, want %q", parseEntry.Code, parseEntry.Docs, parseWant)
		}
	}
}

// TestRegistryHasNoDuplicateCodes guards against two literals sharing a code.
func TestRegistryHasNoDuplicateCodes(parseT *testing.T) {
	parseSeen := map[Code]bool{}
	for _, parseEntry := range registryEntries {
		if parseSeen[parseEntry.Code] {
			parseT.Fatalf("duplicate code %s", parseEntry.Code)
		}
		parseSeen[parseEntry.Code] = true
	}
}

// TestLookupAndAllCopy covers hits, misses, and that All hands out a copy.
func TestLookupAndAllCopy(parseT *testing.T) {
	parseEntry, parseOk := Lookup(RuntimePanicRender)
	if !parseOk || parseEntry.Family != FamilyPanic {
		parseT.Fatalf("Lookup(RuntimePanicRender) = %+v, %v", parseEntry, parseOk)
	}
	if _, parseOk := Lookup("GWC-NOPE"); parseOk {
		parseT.Fatal("Lookup of an unknown code succeeded")
	}
	parseAll := All()
	parseAll[0].Headline = "mutated"
	if parseFirst := All()[0]; parseFirst.Headline == "mutated" {
		parseT.Fatal("All() exposed the registry slice")
	}
}

// TestErrorWrapsAndMatchesByCode covers Error, Unwrap, Is, and CodeOf.
func TestErrorWrapsAndMatchesByCode(parseT *testing.T) {
	parseCause := errors.New("boom")
	parseErr := fmt.Errorf("outer: %w", New(RouterLoaderFailed, "/users", parseCause))
	if parseCode := CodeOf(parseErr); parseCode != RouterLoaderFailed {
		parseT.Fatalf("CodeOf = %q", parseCode)
	}
	if !errors.Is(parseErr, New(RouterLoaderFailed)) {
		parseT.Fatal("errors.Is should match by code")
	}
	if errors.Is(parseErr, New(RouterRedirectLoop)) {
		parseT.Fatal("errors.Is matched a different code")
	}
	if !errors.Is(parseErr, parseCause) {
		parseT.Fatal("Unwrap should expose the wrapped cause")
	}
	parseText := New(RouterLoaderFailed, "/users").Error()
	if !strings.HasPrefix(parseText, "[GWC-ROUTER-LOADER-FAILED] Route loader failed") || !strings.HasSuffix(parseText, ": /users") {
		parseT.Fatalf("Error() = %q", parseText)
	}
	if parseCode := CodeOf(errors.New("plain")); parseCode != "" {
		parseT.Fatalf("CodeOf(plain) = %q", parseCode)
	}
	if New("GWC-UNKNOWN").Error() != "GWC-UNKNOWN" {
		parseT.Fatal("unknown code should print bare")
	}
}
