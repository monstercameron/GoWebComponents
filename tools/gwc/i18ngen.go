package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// runI18nCommand routes the `gwc i18n <gen|check>` typed-message-key codegen.
var runI18nCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runI18n(parseArgs)
}

const i18nGenFile = "i18n_keys_gen.go"

// i18nMessage is one discovered message: its namespace, key, and ordered {param} names.
type i18nMessage struct {
	namespace string
	key       string
	params    []string
}

// runI18n parses `gwc i18n gen|check -bundle FILE [-pkg DIR] [-out FILE]` and writes or
// verifies the generated typed message accessors.
func (parseL launcher) runI18n(parseArgs []string) error {
	parseAction := "gen"
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseAction = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if parseAction != "gen" && parseAction != "check" {
		return fmt.Errorf("unknown i18n action %q (use gen or check)", parseAction)
	}

	parseFlags := flag.NewFlagSet("i18n "+parseAction, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseBundle := parseFlags.String("bundle", "", "Base-locale bundle JSON ({\"namespace\":{\"key\":\"text {param}\"}}); required")
	parsePkgDir := parseFlags.String("pkg", "", "Package directory for the generated file; defaults to the bundle's directory")
	parseCheck := parseFlags.Bool("check", false, "Do not write; exit non-zero if "+i18nGenFile+" is stale (CI gate)")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseBundlePath := strings.TrimSpace(*parseBundle)
	if parseBundlePath == "" {
		return errors.New("gwc i18n gen requires -bundle pointing at the base-locale JSON file")
	}
	parseBundleAbs, parseErr := filepath.Abs(parseBundlePath)
	if parseErr != nil {
		return fmt.Errorf("resolve i18n bundle: %w", parseErr)
	}

	parseOutDir := strings.TrimSpace(*parsePkgDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Dir(parseBundleAbs)
	}
	parseOutAbs, parseErr := filepath.Abs(parseOutDir)
	if parseErr != nil {
		return fmt.Errorf("resolve i18n pkg: %w", parseErr)
	}

	parsePkgName, parseErr := packageNameForDir(parseOutAbs)
	if parseErr != nil {
		return parseErr
	}

	parseMessages, parseErr := collectMessages(parseBundleAbs)
	if parseErr != nil {
		return parseErr
	}
	if len(parseMessages) == 0 {
		return fmt.Errorf("no messages found in %s", parseBundleAbs)
	}

	parseGenerated, parseErr := generateI18nFile(parsePkgName, parseMessages)
	if parseErr != nil {
		return parseErr
	}

	parseOutPath := filepath.Join(parseOutAbs, i18nGenFile)
	if *parseCheck || parseAction == "check" {
		if !routesFileMatches(parseOutPath, parseGenerated) {
			fmt.Printf("GWC i18n: STALE — %s out of date; run `gwc i18n gen` and commit\n", i18nGenFile)
			return fmt.Errorf("%s is stale", i18nGenFile)
		}
		fmt.Printf("GWC i18n: %s up to date (%d message(s))\n", i18nGenFile, len(parseMessages))
		return nil
	}

	if parseErr := os.WriteFile(parseOutPath, []byte(parseGenerated), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", i18nGenFile, parseErr)
	}
	fmt.Printf("GWC i18n: wrote %s with %d typed message accessor(s)\n", i18nGenFile, len(parseMessages))
	return nil
}

// packageNameForDir returns the Go package name declared by the .go files in dir,
// defaulting to the sanitized directory base name when the directory has no Go files yet.
func packageNameForDir(parseDir string) (string, error) {
	parseEntries, parseErr := os.ReadDir(parseDir)
	if parseErr != nil {
		return "", fmt.Errorf("read i18n pkg dir: %w", parseErr)
	}
	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if parseEntry.IsDir() || !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") || parseName == i18nGenFile {
			continue
		}
		if parsePkg := readPackageClause(filepath.Join(parseDir, parseName)); parsePkg != "" {
			return parsePkg, nil
		}
	}
	parseBase := sanitizeGoIdent(filepath.Base(parseDir))
	if parseBase == "" {
		parseBase = "main"
	}
	return parseBase, nil
}

// readPackageClause returns the package name declared by a Go file, or "" if it cannot be
// determined without a full parse.
func readPackageClause(parsePath string) string {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return ""
	}
	for parseLine := range strings.SplitSeq(string(parseData), "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseAfter, parseOk := strings.CutPrefix(parseTrimmed, "package "); parseOk {
			return strings.TrimSpace(parseAfter)
		}
	}
	return ""
}

// collectMessages reads the base-locale bundle JSON and returns every (namespace, key)
// message with its ordered {param} names, sorted for deterministic output.
func collectMessages(parseBundlePath string) ([]i18nMessage, error) {
	parseData, parseErr := os.ReadFile(parseBundlePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read i18n bundle: %w", parseErr)
	}
	var parseCatalog map[string]map[string]string
	if parseErr := json.Unmarshal(parseData, &parseCatalog); parseErr != nil {
		return nil, fmt.Errorf("parse i18n bundle JSON (want {\"namespace\":{\"key\":\"text\"}}): %w", parseErr)
	}

	var parseMessages []i18nMessage
	for parseNamespace, parseEntries := range parseCatalog {
		for parseKey, parseText := range parseEntries {
			parseMessages = append(parseMessages, i18nMessage{
				namespace: parseNamespace,
				key:       parseKey,
				params:    templateParams(parseText),
			})
		}
	}
	sort.Slice(parseMessages, func(parseA, parseB int) bool {
		if parseMessages[parseA].namespace != parseMessages[parseB].namespace {
			return parseMessages[parseA].namespace < parseMessages[parseB].namespace
		}
		return parseMessages[parseA].key < parseMessages[parseB].key
	})
	return parseMessages, nil
}

// templateParams returns the ordered, de-duplicated {param} names in a message template.
func templateParams(parseText string) []string {
	var parseParams []string
	parseSeen := map[string]struct{}{}
	parseRest := parseText
	for {
		parseOpen := strings.IndexByte(parseRest, '{')
		if parseOpen < 0 {
			break
		}
		parseClose := strings.IndexByte(parseRest[parseOpen:], '}')
		if parseClose < 0 {
			break
		}
		parseName := strings.TrimSpace(parseRest[parseOpen+1 : parseOpen+parseClose])
		parseRest = parseRest[parseOpen+parseClose+1:]
		if parseName == "" {
			continue
		}
		if _, parseDup := parseSeen[parseName]; parseDup {
			continue
		}
		parseSeen[parseName] = struct{}{}
		parseParams = append(parseParams, parseName)
	}
	return parseParams
}

// generateI18nFile renders the typed message-accessor Go source, formatted with
// go/format so it is gofmt-clean. Each accessor takes an i18n.Runtime and one string
// argument per {param}, so a missing namespace, key, or param is a compile error.
func generateI18nFile(parsePkgName string, parseMessages []i18nMessage) (string, error) {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("// Code generated by `gwc i18n gen`; DO NOT EDIT.\n\n")
	fmt.Fprintf(&parseBuilder, "package %s\n\n", parsePkgName)
	parseBuilder.WriteString("import \"github.com/monstercameron/GoWebComponents/v6/i18n\"\n\n")
	parseBuilder.WriteString("// Typed, compile-checked accessors for every message in the base-locale bundle.\n")
	parseBuilder.WriteString("// A typo in a namespace, key, or interpolation parameter is now a compile error.\n\n")

	parseSeen := map[string]struct{}{}
	for _, parseMessage := range parseMessages {
		parseFuncName := dedupeName(i18nFuncName(parseMessage.namespace, parseMessage.key), parseSeen)
		parseParamIdents := make([]string, len(parseMessage.params))
		parseArgList := make([]string, len(parseMessage.params))
		for parseI, parseParam := range parseMessage.params {
			parseIdent := safeI18nParamIdent(parseParam, parseI)
			parseParamIdents[parseI] = parseIdent
			parseArgList[parseI] = parseIdent + " string"
		}

		parseSignature := "parseR i18n.Runtime"
		if len(parseArgList) > 0 {
			parseSignature += ", " + strings.Join(parseArgList, ", ")
		}
		fmt.Fprintf(&parseBuilder, "// %s returns the %q/%q message.\n", parseFuncName, parseMessage.namespace, parseMessage.key)
		fmt.Fprintf(&parseBuilder, "func %s(%s) string {\n", parseFuncName, parseSignature)
		if len(parseMessage.params) == 0 {
			fmt.Fprintf(&parseBuilder, "\treturn parseR.T(%q, %q)\n}\n\n", parseMessage.namespace, parseMessage.key)
			continue
		}
		fmt.Fprintf(&parseBuilder, "\treturn parseR.T(%q, %q, i18n.Arguments{\n", parseMessage.namespace, parseMessage.key)
		for parseI, parseParam := range parseMessage.params {
			fmt.Fprintf(&parseBuilder, "\t\t%q: %s,\n", parseParam, parseParamIdents[parseI])
		}
		parseBuilder.WriteString("\t})\n}\n\n")
	}

	parseFormatted, parseErr := format.Source([]byte(parseBuilder.String()))
	if parseErr != nil {
		return "", fmt.Errorf("format generated i18n keys: %w", parseErr)
	}
	return string(parseFormatted), nil
}

// i18nFuncName builds an exported accessor name from a namespace and key by PascalCasing
// each dotted/underscored/hyphenated segment.
func i18nFuncName(parseNamespace, parseKey string) string {
	parseName := pascalSegments(parseNamespace) + pascalSegments(parseKey)
	if parseName == "" || !unicode.IsLetter(rune(parseName[0])) {
		parseName = "Msg" + parseName
	}
	return parseName
}

// pascalSegments splits on '.', '_', '-', '/', and spaces, then PascalCases each segment.
func pascalSegments(parseInput string) string {
	var parseBuilder strings.Builder
	for _, parseField := range strings.FieldsFunc(parseInput, func(parseR rune) bool {
		return parseR == '.' || parseR == '_' || parseR == '-' || parseR == '/' || parseR == ' '
	}) {
		parseClean := sanitizeGoIdent(parseField)
		if parseClean == "" {
			continue
		}
		parseBuilder.WriteString(strings.ToUpper(parseClean[:1]))
		parseBuilder.WriteString(parseClean[1:])
	}
	return parseBuilder.String()
}

// sanitizeGoIdent strips characters that are not letters/digits/underscore. A leading
// digit is preserved (callers that need a valid identifier head prefix it).
func sanitizeGoIdent(parseInput string) string {
	var parseBuilder strings.Builder
	for _, parseR := range parseInput {
		switch {
		case unicode.IsLetter(parseR), unicode.IsDigit(parseR), parseR == '_':
			parseBuilder.WriteRune(parseR)
		}
	}
	return parseBuilder.String()
}

// i18nGoKeywords are reserved words that cannot be used as parameter identifiers.
var i18nGoKeywords = map[string]struct{}{
	"break": {}, "case": {}, "chan": {}, "const": {}, "continue": {}, "default": {},
	"defer": {}, "else": {}, "fallthrough": {}, "for": {}, "func": {}, "go": {},
	"goto": {}, "if": {}, "import": {}, "interface": {}, "map": {}, "package": {},
	"range": {}, "return": {}, "select": {}, "struct": {}, "switch": {}, "type": {},
	"var": {},
}

// safeI18nParamIdent turns a {param} name into a valid, non-keyword Go identifier,
// falling back to a positional name when the param is unusable.
func safeI18nParamIdent(parseParam string, parseIndex int) string {
	parseIdent := sanitizeGoIdent(parseParam)
	if parseIdent == "" {
		return fmt.Sprintf("arg%d", parseIndex)
	}
	if !unicode.IsLetter(rune(parseIdent[0])) && parseIdent[0] != '_' {
		parseIdent = "_" + parseIdent
	}
	if _, parseIsKeyword := i18nGoKeywords[parseIdent]; parseIsKeyword {
		return parseIdent + "_"
	}
	return parseIdent
}

// dedupeName ensures a generated function name is unique by appending a numeric suffix on
// collision (two keys that PascalCase to the same identifier).
func dedupeName(parseName string, parseSeen map[string]struct{}) string {
	parseCandidate := parseName
	for parseI := 2; ; parseI++ {
		if _, parseTaken := parseSeen[parseCandidate]; !parseTaken {
			parseSeen[parseCandidate] = struct{}{}
			return parseCandidate
		}
		parseCandidate = fmt.Sprintf("%s%d", parseName, parseI)
	}
}
