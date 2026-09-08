package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runServerCommand routes the `gwc server <gen|check>` server-function codegen.
var runServerCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runServer(parseArgs)
}

const (
	serverClientGenFile = "serverfn_gen_client.go"
	serverServerGenFile = "serverfn_gen_server.go"
	serverDirective     = "//gwc:server"
)

// serverFunc is one discovered //gwc:server function and its typed RPC contract.
type serverFunc struct {
	name     string
	reqType  string
	respType string
}

// runServer parses `gwc server gen|check [-pkg DIR]` and writes or verifies the generated
// client stubs and server registration for //gwc:server functions.
func (parseL launcher) runServer(parseArgs []string) error {
	parseAction := "gen"
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseAction = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if parseAction != "gen" && parseAction != "check" {
		return fmt.Errorf("unknown server action %q (use gen or check)", parseAction)
	}

	parseFlags := flag.NewFlagSet("server "+parseAction, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parsePkg := parseFlags.String("pkg", "", "Package directory to scan for //gwc:server functions; defaults to the current directory")
	parseCheck := parseFlags.Bool("check", false, "Do not write; exit non-zero if generated files are stale (CI gate)")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parsePkgDir := strings.TrimSpace(*parsePkg)
	if parsePkgDir == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return fmt.Errorf("resolve server pkg from cwd: %w", parseErr)
		}
		parsePkgDir = parseCWD
	}
	parseAbs, parseErr := filepath.Abs(parsePkgDir)
	if parseErr != nil {
		return fmt.Errorf("resolve server pkg: %w", parseErr)
	}

	parsePkgName, parseFuncs, parseErr := collectServerFuncs(parseAbs)
	if parseErr != nil {
		return parseErr
	}
	if len(parseFuncs) == 0 {
		return fmt.Errorf("no %s functions found in %s", serverDirective, parseAbs)
	}

	parseClientSrc, parseErr := generateServerClientFile(parsePkgName, parseFuncs)
	if parseErr != nil {
		return parseErr
	}
	parseServerSrc, parseErr := generateServerRegistrationFile(parsePkgName, parseFuncs)
	if parseErr != nil {
		return parseErr
	}

	parseClientPath := filepath.Join(parseAbs, serverClientGenFile)
	parseServerPath := filepath.Join(parseAbs, serverServerGenFile)

	if *parseCheck || parseAction == "check" {
		if !routesFileMatches(parseClientPath, parseClientSrc) || !routesFileMatches(parseServerPath, parseServerSrc) {
			fmt.Printf("GWC server: STALE — generated server-function files out of date; run `gwc server gen` and commit\n")
			return errors.New("server-function generated files are stale")
		}
		fmt.Printf("GWC server: generated files up to date (%d function(s))\n", len(parseFuncs))
		return nil
	}

	if parseErr := os.WriteFile(parseClientPath, []byte(parseClientSrc), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", serverClientGenFile, parseErr)
	}
	if parseErr := os.WriteFile(parseServerPath, []byte(parseServerSrc), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", serverServerGenFile, parseErr)
	}
	fmt.Printf("GWC server: wrote %s + %s for %d server function(s)\n", serverClientGenFile, serverServerGenFile, len(parseFuncs))
	return nil
}

// collectServerFuncs parses the package directory and returns the package name and every
// validated //gwc:server function, sorted by name.
func collectServerFuncs(parsePkgDir string) (string, []serverFunc, error) {
	parseEntries, parseErr := os.ReadDir(parsePkgDir)
	if parseErr != nil {
		return "", nil, fmt.Errorf("read server pkg dir: %w", parseErr)
	}
	parseFset := token.NewFileSet()
	parsePkgName := ""
	var parseFuncs []serverFunc
	parseSeen := map[string]struct{}{}

	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if parseEntry.IsDir() || !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			continue
		}
		if parseName == serverClientGenFile || parseName == serverServerGenFile {
			continue
		}
		parsePath := filepath.Join(parsePkgDir, parseName)
		parseFile, parseParseErr := parser.ParseFile(parseFset, parsePath, nil, parser.ParseComments)
		if parseParseErr != nil {
			continue
		}
		if parsePkgName == "" {
			parsePkgName = parseFile.Name.Name
		}

		parseHasServerFunc := false
		for _, parseDecl := range parseFile.Decls {
			parseFuncDecl, parseOk := parseDecl.(*ast.FuncDecl)
			if !parseOk || !hasServerDirective(parseFuncDecl) {
				continue
			}
			parseSF, parseErr := validateServerFunc(parseFset, parseFuncDecl)
			if parseErr != nil {
				return "", nil, fmt.Errorf("%s: %s: %w", parseName, parseFuncDecl.Name.Name, parseErr)
			}
			if _, parseDup := parseSeen[parseSF.name]; parseDup {
				continue
			}
			parseSeen[parseSF.name] = struct{}{}
			parseFuncs = append(parseFuncs, parseSF)
			parseHasServerFunc = true
		}

		// A file declaring server functions must be excluded from the js,wasm build, or
		// the generated client stubs would collide with the real functions.
		if parseHasServerFunc && !fileExcludesClientBuild(parsePath) {
			return "", nil, fmt.Errorf("%s declares %s functions but is not constrained to the server; add a `//go:build !js || !wasm` line so the generated client stubs do not collide", parseName, serverDirective)
		}
	}

	sort.Slice(parseFuncs, func(parseA, parseB int) bool { return parseFuncs[parseA].name < parseFuncs[parseB].name })
	return parsePkgName, parseFuncs, nil
}

// hasServerDirective reports whether a function carries the //gwc:server directive in its
// doc comment.
func hasServerDirective(parseFuncDecl *ast.FuncDecl) bool {
	if parseFuncDecl.Doc == nil {
		return false
	}
	for _, parseComment := range parseFuncDecl.Doc.List {
		if strings.TrimSpace(parseComment.Text) == serverDirective {
			return true
		}
	}
	return false
}

// validateServerFunc enforces the server-function contract:
// func Name(context.Context, Req) (Resp, error), with same-package Req/Resp types.
func validateServerFunc(parseFset *token.FileSet, parseFuncDecl *ast.FuncDecl) (serverFunc, error) {
	if parseFuncDecl.Recv != nil {
		return serverFunc{}, errors.New("must be a top-level function, not a method")
	}
	parseType := parseFuncDecl.Type
	if parseType.TypeParams != nil {
		return serverFunc{}, errors.New("must not be generic")
	}
	if parseType.Params.NumFields() != 2 {
		return serverFunc{}, errors.New("must take exactly (context.Context, RequestType)")
	}
	if parseType.Results == nil || parseType.Results.NumFields() != 2 {
		return serverFunc{}, errors.New("must return exactly (ResponseType, error)")
	}

	parseCtxType := renderExpr(parseFset, parseType.Params.List[0].Type)
	if parseCtxType != "context.Context" {
		return serverFunc{}, fmt.Errorf("first parameter must be context.Context, got %s", parseCtxType)
	}
	if parseErrType := renderExpr(parseFset, parseType.Results.List[1].Type); parseErrType != "error" {
		return serverFunc{}, fmt.Errorf("second result must be error, got %s", parseErrType)
	}

	parseReqExpr := parseType.Params.List[1].Type
	parseRespExpr := parseType.Results.List[0].Type
	if exprReferencesOtherPackage(parseReqExpr) || exprReferencesOtherPackage(parseRespExpr) {
		return serverFunc{}, errors.New("request and response types must be defined in this package (no cross-package types)")
	}

	return serverFunc{
		name:     parseFuncDecl.Name.Name,
		reqType:  renderExpr(parseFset, parseReqExpr),
		respType: renderExpr(parseFset, parseRespExpr),
	}, nil
}

// exprReferencesOtherPackage reports whether a type expression contains a package
// qualifier (a SelectorExpr), which the generated files cannot import.
func exprReferencesOtherPackage(parseExpr ast.Expr) bool {
	parseFound := false
	ast.Inspect(parseExpr, func(parseNode ast.Node) bool {
		if _, parseOk := parseNode.(*ast.SelectorExpr); parseOk {
			parseFound = true
			return false
		}
		return true
	})
	return parseFound
}

// renderExpr renders an AST type expression back to Go source.
func renderExpr(parseFset *token.FileSet, parseExpr ast.Expr) string {
	var parseBuf bytes.Buffer
	if parseErr := format.Node(&parseBuf, parseFset, parseExpr); parseErr != nil {
		return ""
	}
	return parseBuf.String()
}

// fileExcludesClientBuild reports whether the file's //go:build constraint excludes a
// js,wasm (browser) build. A file with no constraint is included everywhere and therefore
// does not exclude the client build.
func fileExcludesClientBuild(parsePath string) bool {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return false
	}
	for parseLine := range strings.SplitSeq(string(parseData), "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if !constraint.IsGoBuild(parseLine) {
			if parseLine != "" && !strings.HasPrefix(parseLine, "//") && !strings.HasPrefix(parseLine, "package") {
				break // reached code; stop scanning the header
			}
			continue
		}
		parseExpr, parseErr := constraint.Parse(parseLine)
		if parseErr != nil {
			return false
		}
		// The file is excluded from the client build when its constraint evaluates false
		// for a build that has both the js and wasm tags set.
		return !parseExpr.Eval(func(parseTag string) bool {
			return parseTag == "js" || parseTag == "wasm"
		})
	}
	return false
}

// generateServerClientFile renders the js&&wasm client stubs that call serverfn.Call.
func generateServerClientFile(parsePkgName string, parseFuncs []serverFunc) (string, error) {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("//go:build js && wasm\n\n")
	parseBuilder.WriteString("// Code generated by `gwc server gen`; DO NOT EDIT.\n\n")
	fmt.Fprintf(&parseBuilder, "package %s\n\n", parsePkgName)
	parseBuilder.WriteString("import (\n\t\"context\"\n\n\t\"github.com/monstercameron/GoWebComponents/v6/serverfn\"\n)\n\n")
	parseBuilder.WriteString("// Browser-side stubs: each calls its server function over HTTP with full type safety.\n\n")

	for _, parseFunc := range parseFuncs {
		fmt.Fprintf(&parseBuilder, "// %s invokes the %q server function from the browser.\n", parseFunc.name, parseFunc.name)
		fmt.Fprintf(&parseBuilder, "func %s(parseCtx context.Context, parseReq %s) (%s, error) {\n", parseFunc.name, parseFunc.reqType, parseFunc.respType)
		fmt.Fprintf(&parseBuilder, "\treturn serverfn.Call[%s, %s](parseCtx, %q, parseReq)\n}\n\n", parseFunc.reqType, parseFunc.respType, parseFunc.name)
	}

	parseFormatted, parseErr := format.Source([]byte(parseBuilder.String()))
	if parseErr != nil {
		return "", fmt.Errorf("format generated client stubs: %w", parseErr)
	}
	return string(parseFormatted), nil
}

// generateServerRegistrationFile renders the server-side registration that wires every
// server function onto an http.ServeMux via serverfn.Handle.
func generateServerRegistrationFile(parsePkgName string, parseFuncs []serverFunc) (string, error) {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("//go:build !js || !wasm\n\n")
	parseBuilder.WriteString("// Code generated by `gwc server gen`; DO NOT EDIT.\n\n")
	fmt.Fprintf(&parseBuilder, "package %s\n\n", parsePkgName)
	parseBuilder.WriteString("import (\n\t\"net/http\"\n\n\t\"github.com/monstercameron/GoWebComponents/v6/serverfn\"\n)\n\n")
	parseBuilder.WriteString("// RegisterServerFunctions wires every //gwc:server function in this package onto mux.\n")
	parseBuilder.WriteString("// Call it once when building the server's HTTP handler.\n")
	parseBuilder.WriteString("func RegisterServerFunctions(parseMux *http.ServeMux) {\n")
	for _, parseFunc := range parseFuncs {
		fmt.Fprintf(&parseBuilder, "\tserverfn.Handle(parseMux, %q, %s)\n", parseFunc.name, parseFunc.name)
	}
	parseBuilder.WriteString("}\n")

	parseFormatted, parseErr := format.Source([]byte(parseBuilder.String()))
	if parseErr != nil {
		return "", fmt.Errorf("format generated server registration: %w", parseErr)
	}
	return string(parseFormatted), nil
}
