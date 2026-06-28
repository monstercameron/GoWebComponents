package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// runRoutesCommand routes the `gwc routes <gen|check>` typed-link codegen.
var runRoutesCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runRoutes(parseArgs)
}

const routesGenFile = "routes_gen.go"

// routeDef is one discovered route contract: the package-level variable that holds
// it, the URL pattern, and the ordered :param names.
type routeDef struct {
	varName string
	pattern string
	params  []string
}

// runRoutes parses `gwc routes gen|check [-pkg DIR]` and writes or verifies the
// generated typed link constructors.
func (parseL launcher) runRoutes(parseArgs []string) error {
	parseAction := "gen"
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseAction = parseArgs[0]
		parseArgs = parseArgs[1:]
	}

	parseFlags := flag.NewFlagSet("routes "+parseAction, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parsePkg := parseFlags.String("pkg", "", "Package directory to scan for route contracts; defaults to the current directory")
	parseCheck := parseFlags.Bool("check", false, "Do not write; exit non-zero if "+routesGenFile+" is stale (CI gate)")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if parseAction != "gen" && parseAction != "check" {
		return fmt.Errorf("unknown routes action %q (use gen or check)", parseAction)
	}

	parsePkgDir := strings.TrimSpace(*parsePkg)
	if parsePkgDir == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return fmt.Errorf("resolve routes pkg from cwd: %w", parseErr)
		}
		parsePkgDir = parseCWD
	}
	parseAbs, parseErr := filepath.Abs(parsePkgDir)
	if parseErr != nil {
		return fmt.Errorf("resolve routes pkg: %w", parseErr)
	}

	parsePkgName, parseRoutes, parseErr := collectRoutes(parseAbs)
	if parseErr != nil {
		return parseErr
	}
	if len(parseRoutes) == 0 {
		return fmt.Errorf("no router.MustDefineRoute/DefineRoute contracts found in %s", parseAbs)
	}

	parseGenerated, parseErr := generateRoutesFile(parsePkgName, parseRoutes)
	if parseErr != nil {
		return parseErr
	}

	parseOutPath := filepath.Join(parseAbs, routesGenFile)
	if parseCheck := *parseCheck; parseCheck || parseAction == "check" {
		if !routesFileMatches(parseOutPath, parseGenerated) {
			fmt.Printf("GWC routes: STALE — %s out of date; run `gwc routes gen` and commit\n", routesGenFile)
			return fmt.Errorf("%s is stale", routesGenFile)
		}
		fmt.Printf("GWC routes: %s up to date (%d route(s))\n", routesGenFile, len(parseRoutes))
		return nil
	}

	if parseErr := os.WriteFile(parseOutPath, []byte(parseGenerated), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", routesGenFile, parseErr)
	}
	fmt.Printf("GWC routes: wrote %s with %d typed link constructor(s)\n", routesGenFile, len(parseRoutes))
	return nil
}

// collectRoutes parses the Go files in pkgDir (excluding tests and the generated
// file) and returns the package name and every package-level route contract.
func collectRoutes(parsePkgDir string) (string, []routeDef, error) {
	parseEntries, parseErr := os.ReadDir(parsePkgDir)
	if parseErr != nil {
		return "", nil, fmt.Errorf("read routes pkg dir: %w", parseErr)
	}
	parseFset := token.NewFileSet()
	parsePkgName := ""
	var parseRoutes []routeDef
	parseSeen := map[string]struct{}{}

	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if parseEntry.IsDir() || !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") || parseName == routesGenFile {
			continue
		}
		parseFile, parseParseErr := parser.ParseFile(parseFset, filepath.Join(parsePkgDir, parseName), nil, 0)
		if parseParseErr != nil {
			continue // skip files that don't parse; they are not the generator's concern
		}
		if parsePkgName == "" {
			parsePkgName = parseFile.Name.Name
		}
		for _, parseDecl := range parseFile.Decls {
			parseGenDecl, parseOk := parseDecl.(*ast.GenDecl)
			if !parseOk || parseGenDecl.Tok != token.VAR {
				continue
			}
			for _, parseSpec := range parseGenDecl.Specs {
				parseValueSpec, parseOk := parseSpec.(*ast.ValueSpec)
				if !parseOk {
					continue
				}
				for parseI, parseValue := range parseValueSpec.Values {
					if parseI >= len(parseValueSpec.Names) {
						break
					}
					parseVarName := parseValueSpec.Names[parseI].Name
					if !parseValueSpec.Names[parseI].IsExported() {
						continue
					}
					parsePattern, parseOk := routeContractPattern(parseValue)
					if !parseOk {
						continue
					}
					if _, parseDup := parseSeen[parseVarName]; parseDup {
						continue
					}
					parseSeen[parseVarName] = struct{}{}
					parseRoutes = append(parseRoutes, routeDef{
						varName: parseVarName,
						pattern: parsePattern,
						params:  patternParams(parsePattern),
					})
				}
			}
		}
	}
	sort.Slice(parseRoutes, func(parseA, parseB int) bool { return parseRoutes[parseA].varName < parseRoutes[parseB].varName })
	return parsePkgName, parseRoutes, nil
}

// routeContractPattern reports whether expr is a MustDefineRoute/DefineRoute call
// with a string-literal pattern, returning that pattern.
func routeContractPattern(parseExpr ast.Expr) (string, bool) {
	parseCall, parseOk := parseExpr.(*ast.CallExpr)
	if !parseOk || len(parseCall.Args) == 0 {
		return "", false
	}
	parseFuncName := ""
	switch parseFun := parseCall.Fun.(type) {
	case *ast.SelectorExpr:
		parseFuncName = parseFun.Sel.Name
	case *ast.Ident:
		parseFuncName = parseFun.Name
	}
	if parseFuncName != "MustDefineRoute" && parseFuncName != "DefineRoute" {
		return "", false
	}
	parseLit, parseOk := parseCall.Args[0].(*ast.BasicLit)
	if !parseOk || parseLit.Kind != token.STRING {
		return "", false
	}
	parsePattern, parseErr := strconv.Unquote(parseLit.Value)
	if parseErr != nil {
		return "", false
	}
	return parsePattern, true
}

// patternParams returns the ordered :param names in a route pattern.
func patternParams(parsePattern string) []string {
	var parseParams []string
	for parseSegment := range strings.SplitSeq(strings.Trim(parsePattern, "/"), "/") {
		if parseName, parseIsParam := strings.CutPrefix(parseSegment, ":"); parseIsParam {
			parseParams = append(parseParams, parseName)
		}
	}
	return parseParams
}

// generateRoutesFile renders the typed-link-constructor Go source for the routes,
// formatted with go/format so it is gofmt-clean.
func generateRoutesFile(parsePkgName string, parseRoutes []routeDef) (string, error) {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("// Code generated by `gwc routes gen`; DO NOT EDIT.\n\n")
	fmt.Fprintf(&parseBuilder, "package %s\n\n", parsePkgName)
	parseBuilder.WriteString("import \"net/url\"\n\n")
	parseBuilder.WriteString("// Typed, compile-checked link constructors for the route contracts in this package.\n")
	parseBuilder.WriteString("// A missing or misnamed path parameter is now a compile error, not a runtime one.\n")
	parseBuilder.WriteString("// Each route also gets a *WithQuery variant that appends typed search params.\n\n")

	for _, parseRoute := range parseRoutes {
		parseFuncName := "Link" + strings.TrimSuffix(parseRoute.varName, "Route")
		parseArgNames := make([]string, len(parseRoute.params))
		parseArgList := make([]string, len(parseRoute.params))
		for parseI, parseParam := range parseRoute.params {
			parseArg := safeRouteParamIdent(parseParam, parseI)
			parseArgNames[parseI] = parseArg
			parseArgList[parseI] = parseArg + " string"
		}
		parseSig := strings.Join(parseArgList, ", ")
		parseMapLiteral := ""
		if len(parseRoute.params) > 0 {
			parseMapLiteral = "map[string]string{"
			for parseI, parseParam := range parseRoute.params {
				parseMapLiteral += fmt.Sprintf("%q: %s, ", parseParam, parseArgNames[parseI])
			}
			parseMapLiteral += "}"
		} else {
			parseMapLiteral = "nil"
		}

		// Plain link constructor (path params only).
		fmt.Fprintf(&parseBuilder, "// %s builds a URL for the %q route.\n", parseFuncName, parseRoute.pattern)
		fmt.Fprintf(&parseBuilder, "func %s(%s) string {\n\treturn %s.MustHref(%s, nil)\n}\n\n", parseFuncName, parseSig, parseRoute.varName, parseMapLiteral)

		// Combined constructor: path params + typed search params (router.EncodeQuery output).
		parseQuerySig := "query url.Values"
		if parseSig != "" {
			parseQuerySig = parseSig + ", query url.Values"
		}
		fmt.Fprintf(&parseBuilder, "// %sWithQuery builds a URL for the %q route with search params appended.\n", parseFuncName, parseRoute.pattern)
		fmt.Fprintf(&parseBuilder, "func %sWithQuery(%s) string {\n\treturn %s.MustHref(%s, query)\n}\n\n", parseFuncName, parseQuerySig, parseRoute.varName, parseMapLiteral)
	}

	parseFormatted, parseErr := format.Source([]byte(parseBuilder.String()))
	if parseErr != nil {
		return "", fmt.Errorf("format generated routes: %w", parseErr)
	}
	return string(parseFormatted), nil
}

// routeGoKeywords are reserved words that cannot be used as parameter identifiers.
var routeGoKeywords = map[string]struct{}{
	"break": {}, "case": {}, "chan": {}, "const": {}, "continue": {}, "default": {},
	"defer": {}, "else": {}, "fallthrough": {}, "for": {}, "func": {}, "go": {},
	"goto": {}, "if": {}, "import": {}, "interface": {}, "map": {}, "package": {},
	"range": {}, "return": {}, "select": {}, "struct": {}, "switch": {}, "type": {},
	"var": {},
}

// safeRouteParamIdent turns a route :param name into a valid, non-keyword Go
// identifier, falling back to a positional name when the param is unusable.
func safeRouteParamIdent(parseParam string, parseIndex int) string {
	var parseBuilder strings.Builder
	for parseI, parseR := range parseParam {
		switch {
		case parseR >= 'a' && parseR <= 'z', parseR >= 'A' && parseR <= 'Z', parseR == '_':
			parseBuilder.WriteRune(parseR)
		case parseR >= '0' && parseR <= '9' && parseI > 0:
			parseBuilder.WriteRune(parseR)
		default:
			parseBuilder.WriteByte('_')
		}
	}
	parseIdent := parseBuilder.String()
	if parseIdent == "" || parseIdent == "_" {
		return fmt.Sprintf("arg%d", parseIndex)
	}
	if _, parseIsKeyword := routeGoKeywords[parseIdent]; parseIsKeyword {
		return parseIdent + "_"
	}
	return parseIdent
}

// routesFileMatches reports whether the file at path exists and equals want.
func routesFileMatches(parsePath, parseWant string) bool {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return false
	}
	return string(parseData) == parseWant
}
