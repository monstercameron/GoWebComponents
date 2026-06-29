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
	"strconv"
	"strings"
)

// runCSSCommand routes `gwc css <gen|check>` typed-token codegen — the styling
// analog of `gwc i18n gen` / `gwc routes gen`: a theme JSON is the source of
// truth, and a typo in a token reference becomes a compile error.
var runCSSCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runCSS(parseArgs)
}

const cssGenFile = "css_tokens_gen.go"

// cssThemeFile is the JSON theme shape the generator reads. Every field is
// optional; only the present scales emit constants. Colors are the primary
// stringly-typed axis this closes, but spacing/fontSizes/radii are accepted too.
type cssThemeFile struct {
	Colors    map[string]string `json:"colors"`
	FontSizes map[string]string `json:"fontSizes"`
	Radii     map[string]string `json:"radii"`
	Spacing   map[string]string `json:"spacing"`
}

// runCSS parses `gwc css gen|check -theme FILE [-pkg DIR] [-out FILE]` and writes
// or verifies the generated typed token constants.
func (parseL launcher) runCSS(parseArgs []string) error {
	parseAction := "gen"
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseAction = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if parseAction != "gen" && parseAction != "check" {
		return fmt.Errorf("unknown css action %q (use gen or check)", parseAction)
	}

	parseFlags := flag.NewFlagSet("css "+parseAction, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseTheme := parseFlags.String("theme", "", "Theme JSON ({\"colors\":{\"slate-900\":\"#0f172a\"},\"fontSizes\":{...},\"radii\":{...},\"spacing\":{...}}); required")
	parsePkgDir := parseFlags.String("pkg", "", "Package directory for the generated file; defaults to the theme file's directory")
	parseCheck := parseFlags.Bool("check", false, "Do not write; exit non-zero if "+cssGenFile+" is stale (CI gate)")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseThemePath := strings.TrimSpace(*parseTheme)
	if parseThemePath == "" {
		return errors.New("gwc css gen requires -theme pointing at the theme JSON file")
	}
	parseThemeAbs, parseErr := filepath.Abs(parseThemePath)
	if parseErr != nil {
		return fmt.Errorf("resolve css theme: %w", parseErr)
	}

	parseOutDir := strings.TrimSpace(*parsePkgDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Dir(parseThemeAbs)
	}
	parseOutAbs, parseErr := filepath.Abs(parseOutDir)
	if parseErr != nil {
		return fmt.Errorf("resolve css pkg: %w", parseErr)
	}

	parsePkgName, parseErr := packageNameForDir(parseOutAbs)
	if parseErr != nil {
		return parseErr
	}

	parseTokens, parseErr := collectThemeTokens(parseThemeAbs)
	if parseErr != nil {
		return parseErr
	}
	if len(parseTokens) == 0 {
		return fmt.Errorf("no tokens found in %s; expected a JSON object with one or more of the keys \"colors\", \"fontSizes\", \"radii\", \"spacing\"", parseThemeAbs)
	}

	parseGenerated, parseErr := generateCSSTokensFile(parsePkgName, parseTokens)
	if parseErr != nil {
		return parseErr
	}

	parseOutPath := filepath.Join(parseOutAbs, cssGenFile)
	if *parseCheck || parseAction == "check" {
		if !routesFileMatches(parseOutPath, parseGenerated) {
			fmt.Printf("GWC css: STALE — %s out of date; run `gwc css gen` and commit\n", cssGenFile)
			return fmt.Errorf("%s is stale", cssGenFile)
		}
		fmt.Printf("GWC css: %s up to date (%d token(s))\n", cssGenFile, len(parseTokens))
		return nil
	}

	if parseErr := os.WriteFile(parseOutPath, []byte(parseGenerated), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", cssGenFile, parseErr)
	}
	fmt.Printf("GWC css: wrote %s with %d typed token constant(s)\n", cssGenFile, len(parseTokens))
	return nil
}

// cssToken is one generated constant: its Go identifier, the u-package token type
// it is typed as, and the underlying theme key it resolves against.
type cssToken struct {
	ident    string // generated Go identifier, e.g. ColorBrand500
	tokenTyp string // u.ColorToken / u.TextScale / u.Radius / u.Spacing
	value    string // the theme key, e.g. "brand-500" (or an int literal for spacing)
	isInt    bool   // Spacing tokens are typed ints, not strings
}

// collectThemeTokens reads the theme JSON and returns every token across the
// colors/fontSizes/radii/spacing scales, sorted for deterministic output.
func collectThemeTokens(parseThemePath string) ([]cssToken, error) {
	parseData, parseErr := os.ReadFile(parseThemePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read css theme: %w", parseErr)
	}
	var parseTheme cssThemeFile
	if parseErr := json.Unmarshal(parseData, &parseTheme); parseErr != nil {
		return nil, fmt.Errorf("parse css theme JSON (want {\"colors\":{\"name\":\"value\"},...}): %w", parseErr)
	}

	var parseTokens []cssToken
	parseSeen := map[string]struct{}{}

	parseAddStr := func(parsePrefix, parseTyp string, parseScale map[string]string) {
		parseNames := make([]string, 0, len(parseScale))
		for parseName := range parseScale {
			parseNames = append(parseNames, parseName)
		}
		sort.Strings(parseNames)
		for _, parseName := range parseNames {
			parseIdent := dedupeName(parsePrefix+pascalSegments(parseName), parseSeen)
			parseTokens = append(parseTokens, cssToken{ident: parseIdent, tokenTyp: parseTyp, value: parseName})
		}
	}

	parseAddStr("Color", "u.ColorToken", parseTheme.Colors)
	parseAddStr("Text", "u.TextScale", parseTheme.FontSizes)
	parseAddStr("Radius", "u.Radius", parseTheme.Radii)

	// Spacing keys are integer scale indices typed as u.Spacing.
	parseSpacingKeys := make([]int, 0, len(parseTheme.Spacing))
	for parseKey := range parseTheme.Spacing {
		if parseN, parseErr := strconv.Atoi(strings.TrimSpace(parseKey)); parseErr == nil {
			parseSpacingKeys = append(parseSpacingKeys, parseN)
		}
	}
	sort.Ints(parseSpacingKeys)
	for _, parseN := range parseSpacingKeys {
		parseIdent := dedupeName(fmt.Sprintf("Spacing%d", parseN), parseSeen)
		parseTokens = append(parseTokens, cssToken{ident: parseIdent, tokenTyp: "u.Spacing", value: strconv.Itoa(parseN), isInt: true})
	}

	return parseTokens, nil
}

// generateCSSTokensFile renders the typed token-constant Go source, formatted with
// go/format. Each constant is typed as the matching css/u token type, so a token
// reference autocompletes and a typo is a compile error.
func generateCSSTokensFile(parsePkgName string, parseTokens []cssToken) (string, error) {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("// Code generated by `gwc css gen`; DO NOT EDIT.\n\n")
	fmt.Fprintf(&parseBuilder, "package %s\n\n", parsePkgName)
	parseBuilder.WriteString("import \"github.com/monstercameron/GoWebComponents/css/u\"\n\n")
	parseBuilder.WriteString("// Typed, compile-checked token constants for the application theme.\n")
	parseBuilder.WriteString("// A typo in a color/size/radius/spacing token is now a compile error, and the\n")
	parseBuilder.WriteString("// full token set autocompletes. Pass these to u.BgC/u.TextC/u.TextSize/etc.\n\n")
	parseBuilder.WriteString("const (\n")
	for _, parseToken := range parseTokens {
		if parseToken.isInt {
			fmt.Fprintf(&parseBuilder, "\t%s %s = %s\n", parseToken.ident, parseToken.tokenTyp, parseToken.value)
			continue
		}
		fmt.Fprintf(&parseBuilder, "\t%s %s = %q\n", parseToken.ident, parseToken.tokenTyp, parseToken.value)
	}
	parseBuilder.WriteString(")\n")

	parseFormatted, parseErr := format.Source([]byte(parseBuilder.String()))
	if parseErr != nil {
		return "", fmt.Errorf("format generated css tokens: %w", parseErr)
	}
	return string(parseFormatted), nil
}
