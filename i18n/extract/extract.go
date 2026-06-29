// Package extract provides build-time i18n message extraction and locale
// completeness checking for the GoWebComponents i18n framework.
//
// The extractor finds all calls of the form receiver.T("namespace", "key", ...)
// by walking the Go AST. Calls where either the namespace or key argument is
// not a plain string literal are recorded as dynamic usages so that authors
// know which translations cannot be statically verified.
package extract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Message represents a single statically-extractable translation reference.
// Namespace and Key correspond to the first two string-literal arguments of a
// .T() call; File and Line record where the call appears in source.
type Message struct {
	Namespace string
	Key       string
	File      string
	Line      int
}

// DynamicUsage records a .T() call whose namespace or key argument could not
// be resolved to a plain string literal at extraction time. Reason describes
// which argument was non-literal.
type DynamicUsage struct {
	File   string
	Line   int
	Reason string
}

// ExtractResult is the combined output of extracting translation references
// from one or more Go source files.
type ExtractResult struct {
	Messages []Message
	Dynamic  []DynamicUsage
}

// LocaleReport summarises completeness for a single locale. Missing contains
// every (Namespace, Key) pair found in code that is absent from or empty in
// the locale's catalog. Stale lists catalog keys (formatted as
// "namespace.key") that exist in the catalog but are not referenced in code.
type LocaleReport struct {
	Locale  string
	Missing []Message
	Stale   []string
}

// ExtractFromSource parses a single Go source string and returns all
// translation references found within it. parseFilename is used only for
// position reporting; parseSource must be valid Go source text.
func ExtractFromSource(parseFilename string, parseSource string) (ExtractResult, error) {
	parseFset := token.NewFileSet()
	parseF, parseErr := parser.ParseFile(parseFset, parseFilename, parseSource, 0)
	if parseErr != nil {
		return ExtractResult{}, fmt.Errorf("extract: parse %q: %w", parseFilename, parseErr)
	}
	return parseWalkFile(parseFset, parseF), nil
}

// ExtractFromDir walks parseRoot recursively, parses every .go file found
// (including build-tagged files; excluding vendor, node_modules, .git, bin,
// and testdata directories), and returns the aggregated extraction results.
// Identical (Namespace, Key) pairs are deduplicated, keeping the first
// occurrence.
func ExtractFromDir(parseRoot string) (ExtractResult, error) {
	parseSkipDirs := map[string]bool{
		"vendor":       true,
		"node_modules": true,
		".git":         true,
		"bin":          true,
		"testdata":     true,
	}

	parseAgg := ExtractResult{}
	parseSeen := map[string]bool{}

	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseDe fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseDe.IsDir() {
			if parseSkipDirs[parseDe.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parseDe.Name(), ".go") {
			return nil
		}

		parseBytes, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return fmt.Errorf("extract: read %q: %w", parsePath, parseReadErr)
		}

		// Use parser.SkipObjectResolution to keep things fast; build
		// constraints are intentionally ignored (no parser.Mode flag sets
		// them) so both _wasm.go and _native.go files are always scanned.
		parseFset := token.NewFileSet()
		parseF, parseParseErr := parser.ParseFile(parseFset, parsePath, parseBytes, parser.SkipObjectResolution)
		if parseParseErr != nil {
			// Skip files that do not parse rather than aborting the walk.
			return nil
		}

		parseFileResult := parseWalkFile(parseFset, parseF)

		for _, parseMsg := range parseFileResult.Messages {
			parseDedupeKey := parseMsg.Namespace + "\x00" + parseMsg.Key
			if !parseSeen[parseDedupeKey] {
				parseSeen[parseDedupeKey] = true
				parseAgg.Messages = append(parseAgg.Messages, parseMsg)
			}
		}
		parseAgg.Dynamic = append(parseAgg.Dynamic, parseFileResult.Dynamic...)

		return nil
	})

	if parseErr != nil {
		return ExtractResult{}, parseErr
	}
	return parseAgg, nil
}

// DiffLocale computes the completeness of parseCatalog for parseLocale
// relative to parseExtracted. parseCatalog maps namespace -> key ->
// translated-string. A message is considered missing when its entry is absent
// from the catalog or when its translated string is empty.
func DiffLocale(parseExtracted ExtractResult, parseLocale string, parseCatalog map[string]map[string]string) LocaleReport {
	parseReport := LocaleReport{Locale: parseLocale}

	// Build a set of all (ns, key) pairs that are referenced in code.
	parseCodeKeys := map[string]bool{}
	for _, parseMsg := range parseExtracted.Messages {
		parseCodeKeys[parseMsg.Namespace+"\x00"+parseMsg.Key] = true
	}

	// Determine missing: code references not present (or empty) in catalog.
	for _, parseMsg := range parseExtracted.Messages {
		parseNsMap, parseNsOk := parseCatalog[parseMsg.Namespace]
		if !parseNsOk {
			parseReport.Missing = append(parseReport.Missing, parseMsg)
			continue
		}
		parseVal, parseKeyOk := parseNsMap[parseMsg.Key]
		if !parseKeyOk || parseVal == "" {
			parseReport.Missing = append(parseReport.Missing, parseMsg)
		}
	}

	// Determine stale: catalog keys not referenced in code.
	for parseNs, parseNsMap := range parseCatalog {
		for parseKey := range parseNsMap {
			parseLookup := parseNs + "\x00" + parseKey
			if !parseCodeKeys[parseLookup] {
				parseReport.Stale = append(parseReport.Stale, parseNs+"."+parseKey)
			}
		}
	}
	sort.Strings(parseReport.Stale)

	return parseReport
}

// IsComplete reports whether parseReport has no missing keys.
func IsComplete(parseReport LocaleReport) bool {
	return len(parseReport.Missing) == 0
}

// CheckLocales runs DiffLocale for every locale in parseCatalogs and returns
// the per-locale reports sorted by locale name. parseComplete is false when
// any locale has at least one missing key.
func CheckLocales(parseExtracted ExtractResult, parseCatalogs map[string]map[string]map[string]string) (parseReports []LocaleReport, parseComplete bool) {
	parseLocales := make([]string, 0, len(parseCatalogs))
	for parseLocale := range parseCatalogs {
		parseLocales = append(parseLocales, parseLocale)
	}
	sort.Strings(parseLocales)

	parseComplete = true
	for _, parseLocale := range parseLocales {
		parseReport := DiffLocale(parseExtracted, parseLocale, parseCatalogs[parseLocale])
		parseReports = append(parseReports, parseReport)
		if !IsComplete(parseReport) {
			parseComplete = false
		}
	}
	return parseReports, parseComplete
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

// parseWalkFile inspects every call expression in parseF and records .T(...)
// calls whose receiver selector is "T".
func parseWalkFile(parseFset *token.FileSet, parseF *ast.File) ExtractResult {
	parseResult := ExtractResult{}

	ast.Inspect(parseF, func(parseNode ast.Node) bool {
		parseCall, parseOk := parseNode.(*ast.CallExpr)
		if !parseOk {
			return true
		}

		parseSel, parseSelOk := parseCall.Fun.(*ast.SelectorExpr)
		if !parseSelOk || parseSel.Sel.Name != "T" {
			return true
		}

		if len(parseCall.Args) < 2 {
			return true
		}

		parsePos := parseFset.Position(parseCall.Pos())

		parseNs, parseNsOk := parseStringLit(parseCall.Args[0])
		parseKey, parseKeyOk := parseStringLit(parseCall.Args[1])

		switch {
		case !parseNsOk && !parseKeyOk:
			parseResult.Dynamic = append(parseResult.Dynamic, DynamicUsage{
				File:   parsePos.Filename,
				Line:   parsePos.Line,
				Reason: "namespace and key are both non-literal",
			})
		case !parseNsOk:
			parseResult.Dynamic = append(parseResult.Dynamic, DynamicUsage{
				File:   parsePos.Filename,
				Line:   parsePos.Line,
				Reason: "namespace is non-literal",
			})
		case !parseKeyOk:
			parseResult.Dynamic = append(parseResult.Dynamic, DynamicUsage{
				File:   parsePos.Filename,
				Line:   parsePos.Line,
				Reason: "key is non-literal",
			})
		default:
			parseResult.Messages = append(parseResult.Messages, Message{
				Namespace: parseNs,
				Key:       parseKey,
				File:      parsePos.Filename,
				Line:      parsePos.Line,
			})
		}

		return true
	})

	return parseResult
}

// parseStringLit returns the string value of parseExpr if it is a basic
// string literal, and reports whether extraction succeeded.
func parseStringLit(parseExpr ast.Expr) (string, bool) {
	parseLit, parseOk := parseExpr.(*ast.BasicLit)
	if !parseOk || parseLit.Kind != token.STRING {
		return "", false
	}
	// BasicLit.Value includes surrounding quotes; strip them.
	parseRaw := parseLit.Value
	if len(parseRaw) >= 2 {
		// Handle both "..." and `...` forms.
		if (parseRaw[0] == '"' && parseRaw[len(parseRaw)-1] == '"') ||
			(parseRaw[0] == '`' && parseRaw[len(parseRaw)-1] == '`') {
			return parseRaw[1 : len(parseRaw)-1], true
		}
	}
	return "", false
}
