package extract

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// ExtractFromSource tests
// ---------------------------------------------------------------------------

// TestExtractFromSource_BasicCalls verifies that plain .T("ns","key") calls
// and calls with extra interpolation arguments are all captured correctly.
func TestExtractFromSource_BasicCalls(t *testing.T) {
	parseSrc := `package app

import "github.com/monstercameron/GoWebComponents/v6/i18n"

func parseExample() {
	parseRt := i18n.UseI18n()
	_ = parseRt.T("home", "title")
	_ = parseRt.T("home", "subtitle")
	_ = parseRt.T("nav", "links", map[string]interface{}{"count": 3})
}
`
	parseResult, parseErr := ExtractFromSource("example.go", parseSrc)
	if parseErr != nil {
		t.Fatalf("ExtractFromSource returned error: %v", parseErr)
	}

	parseWant := []struct{ ns, key string }{
		{"home", "title"},
		{"home", "subtitle"},
		{"nav", "links"},
	}
	if len(parseResult.Messages) != len(parseWant) {
		t.Fatalf("got %d messages, want %d; messages: %+v", len(parseResult.Messages), len(parseWant), parseResult.Messages)
	}
	for parseI, parseW := range parseWant {
		parseGot := parseResult.Messages[parseI]
		if parseGot.Namespace != parseW.ns || parseGot.Key != parseW.key {
			t.Errorf("message[%d]: got (%q,%q), want (%q,%q)", parseI, parseGot.Namespace, parseGot.Key, parseW.ns, parseW.key)
		}
		if parseGot.File != "example.go" {
			t.Errorf("message[%d].File = %q, want %q", parseI, parseGot.File, "example.go")
		}
	}
	if len(parseResult.Dynamic) != 0 {
		t.Errorf("expected no dynamic usages, got %+v", parseResult.Dynamic)
	}
}

// TestExtractFromSource_LineNumbers verifies that the recorded line numbers
// point to the actual call sites.
func TestExtractFromSource_LineNumbers(t *testing.T) {
	// The T call is on line 5 (1-indexed).
	parseSrc := "package app\n\nfunc parseF() {\n\t_ = r.T(\"ns\", \"k\")\n}\n"
	parseResult, parseErr := ExtractFromSource("lines.go", parseSrc)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if len(parseResult.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(parseResult.Messages))
	}
	if parseResult.Messages[0].Line != 4 {
		t.Errorf("Line = %d, want 4", parseResult.Messages[0].Line)
	}
}

// TestExtractFromSource_DynamicNamespace checks that a non-literal namespace
// argument lands in Dynamic, not Messages.
func TestExtractFromSource_DynamicNamespace(t *testing.T) {
	parseSrc := `package app

func parseF(parseVar string) {
	_ = rt.T(parseVar, "key")
}
`
	parseResult, parseErr := ExtractFromSource("dyn.go", parseSrc)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if len(parseResult.Messages) != 0 {
		t.Errorf("expected 0 concrete messages, got %+v", parseResult.Messages)
	}
	if len(parseResult.Dynamic) != 1 {
		t.Fatalf("expected 1 dynamic usage, got %d: %+v", len(parseResult.Dynamic), parseResult.Dynamic)
	}
	parseDyn := parseResult.Dynamic[0]
	if parseDyn.File != "dyn.go" {
		t.Errorf("Dynamic[0].File = %q, want %q", parseDyn.File, "dyn.go")
	}
	if parseDyn.Reason == "" {
		t.Error("Dynamic[0].Reason must not be empty")
	}
}

// TestExtractFromSource_DynamicKey checks that a non-literal key argument
// (e.g. a function call) lands in Dynamic.
func TestExtractFromSource_DynamicKey(t *testing.T) {
	parseSrc := `package app

func computeKey() string { return "k" }

func parseF() {
	_ = rt.T("ns", computeKey())
}
`
	parseResult, parseErr := ExtractFromSource("dynkey.go", parseSrc)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if len(parseResult.Messages) != 0 {
		t.Errorf("expected 0 concrete messages, got %+v", parseResult.Messages)
	}
	if len(parseResult.Dynamic) != 1 {
		t.Fatalf("expected 1 dynamic usage, got %d: %+v", len(parseResult.Dynamic), parseResult.Dynamic)
	}
}

// TestExtractFromSource_BothDynamic checks that when both args are non-literal
// a single dynamic entry is recorded.
func TestExtractFromSource_BothDynamic(t *testing.T) {
	parseSrc := `package app

func parseF(parseNs, parseKey string) {
	_ = rt.T(parseNs, parseKey)
}
`
	parseResult, parseErr := ExtractFromSource("bothdyn.go", parseSrc)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if len(parseResult.Messages) != 0 {
		t.Errorf("expected 0 concrete messages, got %+v", parseResult.Messages)
	}
	if len(parseResult.Dynamic) != 1 {
		t.Fatalf("expected 1 dynamic usage, got %d", len(parseResult.Dynamic))
	}
}

// TestExtractFromSource_TooFewArgs ensures that a .T() call with fewer than
// two arguments is silently ignored (no panic, no messages, no dynamic).
func TestExtractFromSource_TooFewArgs(t *testing.T) {
	parseSrc := `package app

func parseF() {
	_ = rt.T("only-one")
}
`
	parseResult, parseErr := ExtractFromSource("fewargs.go", parseSrc)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if len(parseResult.Messages) != 0 || len(parseResult.Dynamic) != 0 {
		t.Errorf("expected empty result, got messages=%+v dynamic=%+v", parseResult.Messages, parseResult.Dynamic)
	}
}

// ---------------------------------------------------------------------------
// DiffLocale tests
// ---------------------------------------------------------------------------

// TestDiffLocale_MissingKey verifies that a code reference absent from the
// catalog is reported in Missing.
func TestDiffLocale_MissingKey(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "home", Key: "title", File: "a.go", Line: 1},
		},
	}
	parseCatalog := map[string]map[string]string{} // empty
	parseReport := DiffLocale(parseExtracted, "en", parseCatalog)

	if len(parseReport.Missing) != 1 {
		t.Fatalf("expected 1 missing key, got %d: %+v", len(parseReport.Missing), parseReport.Missing)
	}
	if parseReport.Missing[0].Key != "title" {
		t.Errorf("Missing[0].Key = %q, want %q", parseReport.Missing[0].Key, "title")
	}
}

// TestDiffLocale_EmptyStringCountsAsMissing checks that a catalog entry with
// an empty string is treated as missing.
func TestDiffLocale_EmptyStringCountsAsMissing(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "home", Key: "title", File: "a.go", Line: 1},
		},
	}
	parseCatalog := map[string]map[string]string{
		"home": {"title": ""},
	}
	parseReport := DiffLocale(parseExtracted, "en", parseCatalog)

	if len(parseReport.Missing) != 1 {
		t.Fatalf("expected 1 missing (empty string), got %d: %+v", len(parseReport.Missing), parseReport.Missing)
	}
}

// TestDiffLocale_StaleKey checks that a catalog key not referenced in code is
// listed in Stale.
func TestDiffLocale_StaleKey(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "home", Key: "title", File: "a.go", Line: 1},
		},
	}
	parseCatalog := map[string]map[string]string{
		"home": {
			"title":  "Home",
			"orphan": "Unused translation",
		},
	}
	parseReport := DiffLocale(parseExtracted, "en", parseCatalog)

	if len(parseReport.Missing) != 0 {
		t.Errorf("expected no missing, got %+v", parseReport.Missing)
	}
	if len(parseReport.Stale) != 1 {
		t.Fatalf("expected 1 stale key, got %d: %+v", len(parseReport.Stale), parseReport.Stale)
	}
	if parseReport.Stale[0] != "home.orphan" {
		t.Errorf("Stale[0] = %q, want %q", parseReport.Stale[0], "home.orphan")
	}
}

// TestDiffLocale_FullyCovered verifies that a fully translated catalog yields
// no missing and no stale entries.
func TestDiffLocale_FullyCovered(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "home", Key: "title", File: "a.go", Line: 1},
			{Namespace: "nav", Key: "home", File: "b.go", Line: 2},
		},
	}
	parseCatalog := map[string]map[string]string{
		"home": {"title": "Home"},
		"nav":  {"home": "Home"},
	}
	parseReport := DiffLocale(parseExtracted, "en", parseCatalog)

	if len(parseReport.Missing) != 0 {
		t.Errorf("expected no missing, got %+v", parseReport.Missing)
	}
	if len(parseReport.Stale) != 0 {
		t.Errorf("expected no stale, got %+v", parseReport.Stale)
	}
}

// ---------------------------------------------------------------------------
// IsComplete tests
// ---------------------------------------------------------------------------

// TestIsComplete_True verifies IsComplete returns true when no keys are missing.
func TestIsComplete_True(t *testing.T) {
	parseReport := LocaleReport{Locale: "en"}
	if !IsComplete(parseReport) {
		t.Error("IsComplete should be true for a report with no missing keys")
	}
}

// TestIsComplete_False verifies IsComplete returns false when keys are missing.
func TestIsComplete_False(t *testing.T) {
	parseReport := LocaleReport{
		Locale:  "fr",
		Missing: []Message{{Namespace: "home", Key: "title"}},
	}
	if IsComplete(parseReport) {
		t.Error("IsComplete should be false when Missing is non-empty")
	}
}

// ---------------------------------------------------------------------------
// CheckLocales tests
// ---------------------------------------------------------------------------

// TestCheckLocales_AllComplete verifies that parseComplete is true when every
// locale has full coverage, and that reports are sorted by locale.
func TestCheckLocales_AllComplete(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "home", Key: "title"},
		},
	}
	parseCatalogs := map[string]map[string]map[string]string{
		"en": {"home": {"title": "Home"}},
		"fr": {"home": {"title": "Accueil"}},
	}
	parseReports, parseComplete := CheckLocales(parseExtracted, parseCatalogs)

	if !parseComplete {
		t.Error("expected parseComplete=true")
	}
	if len(parseReports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(parseReports))
	}
	if parseReports[0].Locale != "en" || parseReports[1].Locale != "fr" {
		t.Errorf("reports not sorted by locale: got %q, %q", parseReports[0].Locale, parseReports[1].Locale)
	}
}

// TestCheckLocales_OneMissing verifies that parseComplete is false when any
// locale has a missing key.
func TestCheckLocales_OneMissing(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "home", Key: "title"},
		},
	}
	parseCatalogs := map[string]map[string]map[string]string{
		"en": {"home": {"title": "Home"}},
		"fr": {}, // missing
	}
	_, parseComplete := CheckLocales(parseExtracted, parseCatalogs)

	if parseComplete {
		t.Error("expected parseComplete=false when a locale has missing keys")
	}
}

// TestCheckLocales_SortedReports verifies that reports are returned in
// ascending lexicographic order of locale regardless of map iteration order.
func TestCheckLocales_SortedReports(t *testing.T) {
	parseExtracted := ExtractResult{
		Messages: []Message{
			{Namespace: "a", Key: "k"},
		},
	}
	parseCatalogs := map[string]map[string]map[string]string{
		"zh": {"a": {"k": "甲"}},
		"ar": {"a": {"k": "ألف"}},
		"en": {"a": {"k": "alpha"}},
	}
	parseReports, _ := CheckLocales(parseExtracted, parseCatalogs)

	parseWantOrder := []string{"ar", "en", "zh"}
	for parseI, parseWant := range parseWantOrder {
		if parseReports[parseI].Locale != parseWant {
			t.Errorf("reports[%d].Locale = %q, want %q", parseI, parseReports[parseI].Locale, parseWant)
		}
	}
}

// ---------------------------------------------------------------------------
// ExtractFromDir smoke test
// ---------------------------------------------------------------------------

// TestExtractFromDir_BuildTaggedFiles proves that files with build tags
// (_wasm.go and _native.go) are both scanned and their .T() calls captured.
func TestExtractFromDir_BuildTaggedFiles(t *testing.T) {
	parseTmpDir := t.TempDir()

	parseWasmSrc := `//go:build wasm

package app

func parseWasmComp() {
	_ = rt.T("wasm", "label")
}
`
	parseNativeSrc := `//go:build !wasm

package app

func parseNativeComp() {
	_ = rt.T("native", "label")
}
`

	parseWasmPath := filepath.Join(parseTmpDir, "comp_wasm.go")
	parseNativePath := filepath.Join(parseTmpDir, "comp_native.go")

	if parseErr := os.WriteFile(parseWasmPath, []byte(parseWasmSrc), 0o644); parseErr != nil {
		t.Fatalf("write wasm file: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseNativePath, []byte(parseNativeSrc), 0o644); parseErr != nil {
		t.Fatalf("write native file: %v", parseErr)
	}

	parseResult, parseErr := ExtractFromDir(parseTmpDir)
	if parseErr != nil {
		t.Fatalf("ExtractFromDir: %v", parseErr)
	}

	parseFoundWasm := false
	parseFoundNative := false
	for _, parseMsg := range parseResult.Messages {
		if parseMsg.Namespace == "wasm" && parseMsg.Key == "label" {
			parseFoundWasm = true
		}
		if parseMsg.Namespace == "native" && parseMsg.Key == "label" {
			parseFoundNative = true
		}
	}
	if !parseFoundWasm {
		t.Error("wasm .T() call not found — build-tagged wasm file was not scanned")
	}
	if !parseFoundNative {
		t.Error("native .T() call not found — build-tagged native file was not scanned")
	}
}

// TestExtractFromDir_Deduplication verifies that identical (ns,key) pairs
// from multiple files appear only once in Messages.
func TestExtractFromDir_Deduplication(t *testing.T) {
	parseTmpDir := t.TempDir()

	parseCommonSrc := func(parsePkg string) string {
		return "package app\n\nfunc " + parsePkg + "F() {\n\t_ = rt.T(\"shared\", \"key\")\n}\n"
	}

	for parseI, parseName := range []string{"a.go", "b.go"} {
		parsePath := filepath.Join(parseTmpDir, parseName)
		parseSuffix := "A"
		if parseI == 1 {
			parseSuffix = "B"
		}
		if parseErr := os.WriteFile(parsePath, []byte(parseCommonSrc(parseSuffix)), 0o644); parseErr != nil {
			t.Fatalf("write %s: %v", parseName, parseErr)
		}
	}

	parseResult, parseErr := ExtractFromDir(parseTmpDir)
	if parseErr != nil {
		t.Fatalf("ExtractFromDir: %v", parseErr)
	}

	parseCount := 0
	for _, parseMsg := range parseResult.Messages {
		if parseMsg.Namespace == "shared" && parseMsg.Key == "key" {
			parseCount++
		}
	}
	if parseCount != 1 {
		t.Errorf("expected 1 deduplicated message for (shared,key), got %d", parseCount)
	}
}
