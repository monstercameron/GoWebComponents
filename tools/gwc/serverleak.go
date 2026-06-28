package main

import (
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// serverOnlyImports are standard-library packages that only make sense on the server;
// importing one into the browser (js && wasm) build is a leak — the code cannot run there
// and it drags server concerns into the client bundle.
var serverOnlyImports = map[string]string{
	"os/exec":       "runs OS processes — unavailable in the browser",
	"database/sql":  "direct database access belongs on the server",
	"net/smtp":      "sending mail belongs on the server",
	"os/user":       "OS user lookup is server-only",
	"net/http/cgi":  "CGI is server-only",
	"net/http/fcgi": "FastCGI is server-only",
	"plugin":        "Go plugins are server-only",
	"os/signal":     "OS signal handling is server-only",
}

// serverOnlyPrefixes classify THIRD-PARTY import paths that are unmistakably server-side —
// databases, RPC servers, cloud SDKs, container/orchestration clients. The previous analyzer
// only knew six stdlib packages, so a `gorm.io/gorm` or `google.golang.org/grpc` import in a
// browser file passed silently; matching by well-known prefix closes that gap. Each entry is a
// path prefix (matched on a package-path boundary) mapped to a human reason.
var serverOnlyPrefixes = []struct {
	Prefix string
	Reason string
}{
	{"gorm.io/", "GORM is a server-side ORM"},
	{"google.golang.org/grpc", "gRPC servers/clients are server-side"},
	{"github.com/aws/aws-sdk-go", "the AWS SDK is a server-side cloud client"},
	{"cloud.google.com/go", "the Google Cloud SDK is server-side"},
	{"github.com/Azure/azure-sdk-for-go", "the Azure SDK is server-side"},
	{"go.mongodb.org/mongo-driver", "the MongoDB driver is server-side"},
	{"github.com/jackc/pgx", "pgx is a server-side Postgres driver"},
	{"github.com/lib/pq", "lib/pq is a server-side Postgres driver"},
	{"github.com/go-sql-driver/mysql", "the MySQL driver is server-side"},
	{"github.com/mattn/go-sqlite3", "go-sqlite3 is a cgo server-side driver"},
	{"github.com/jmoiron/sqlx", "sqlx is a server-side database helper"},
	{"github.com/redis/go-redis", "the Redis client is server-side"},
	{"github.com/go-redis/redis", "the Redis client is server-side"},
	{"k8s.io/client-go", "the Kubernetes client is server-side"},
	{"github.com/docker/docker", "the Docker client is server-side"},
	{"github.com/hashicorp/vault", "the Vault client is server-side"},
	{"github.com/nats-io/nats.go", "NATS messaging is server-side"},
}

// classifyServerOnlyImport reports whether an import path is server-only (and why), matching
// the curated stdlib set first and then the third-party server-package prefixes. The boundary
// check on prefixes avoids matching a path that merely shares a leading substring.
func classifyServerOnlyImport(parseImportPath string) (string, bool) {
	if parseReason, parseBad := serverOnlyImports[parseImportPath]; parseBad {
		return parseReason, true
	}
	for _, parseEntry := range serverOnlyPrefixes {
		if parseImportPath == parseEntry.Prefix ||
			strings.HasPrefix(parseImportPath, parseEntry.Prefix) && (strings.HasSuffix(parseEntry.Prefix, "/") || importPathHasPrefix(parseImportPath, parseEntry.Prefix)) {
			return parseEntry.Reason, true
		}
	}
	return "", false
}

// importPathHasPrefix reports whether parsePath is parsePrefix or a subpackage of it, treating
// the path as slash-separated segments so "grpcweb" is not matched by the "grpc" prefix.
func importPathHasPrefix(parsePath string, parsePrefix string) bool {
	if parsePath == parsePrefix {
		return true
	}
	return strings.HasPrefix(parsePath, parsePrefix+"/")
}

// leakedImport is one reached server-only import together with the chain of local packages
// that led to it, so the diagnostic can show "client file → helper pkg → gorm" rather than a
// bare path. The chain is empty for a direct import.
type leakedImport struct {
	path   string
	reason string
	chain  []string
}

// collectServerLeakDiagnostics walks the transitive import graph that reaches the browser
// (js && wasm) build and reports every server-only package it can reach. It starts from files
// EXPLICITLY constrained to js && wasm (so shared and server files are never false-positived),
// follows their imports into other local packages of the same module, and classifies every
// import path encountered — stdlib server packages, well-known third-party server SDKs, and
// local packages that are themselves constrained off js/wasm. This replaces the old
// direct-imports-only, six-entry deny-list with a real import-graph analysis.
//
// It walks the module from source (resolving local packages by directory) rather than driving
// golang.org/x/tools/go/packages: source walking is deterministic and offline, so the CI gate
// never depends on module download/resolution, and a fixture with no go.mod still analyzes
// direct imports correctly.
func collectServerLeakDiagnostics(parseRootPath string) []agenticDiagnostic {
	parseModuleRoot, parseModulePath := findModule(parseRootPath)
	parseGraph := buildModulePackageGraph(parseModuleRoot, parseModulePath)

	parseDiagnostics := []agenticDiagnostic{}
	parseFset := token.NewFileSet()
	_ = filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseRootPath {
				return filepath.SkipDir
			}
			return nil
		}
		parseName := parseEntry.Name()
		if !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			return nil
		}
		if !fileIsClientOnly(parsePath) {
			return nil
		}
		parseFile, parseErr := parser.ParseFile(parseFset, parsePath, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		for _, parseImport := range parseFile.Imports {
			parseImportPath, parseUnquoteErr := strconv.Unquote(parseImport.Path.Value)
			if parseUnquoteErr != nil {
				continue
			}
			parsePos := parseFset.Position(parseImport.Pos())
			for _, parseLeak := range resolveImportLeaks(parseImportPath, parseGraph) {
				parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
					Code:       "GWC-CHECK-SERVER-LEAK",
					Severity:   "error",
					Message:    serverLeakMessage(parseLeak),
					File:       relativeSlashPath(parseRootPath, parsePath),
					Line:       parsePos.Line,
					Suggestion: "Move the importing file behind a `//go:build !js || !wasm` constraint (run `gwc check --fix`), or call the server code through a //gwc:server function instead of importing it into the client.",
					Attributes: map[string]string{"leaked": parseLeak.path},
				})
			}
		}
		return nil
	})
	return dedupeLeakDiagnostics(parseDiagnostics)
}

// serverLeakMessage renders the human message for one leaked import, showing the transitive
// chain when the leak is reached through local packages.
func serverLeakMessage(parseLeak leakedImport) string {
	parseVia := ""
	if len(parseLeak.chain) > 0 {
		parseVia = " (via " + strings.Join(parseLeak.chain, " → ") + ")"
	}
	return "browser (js && wasm) build reaches server-only package " + parseLeak.path + parseVia + " (" + parseLeak.reason + ")"
}

// resolveImportLeaks classifies a single import path reachable from a client file and returns
// every server-only package it leads to: the import itself when it is server-only, or — when it
// is a local package — the server-only packages found by walking that package transitively.
func resolveImportLeaks(parseImportPath string, parseGraph map[string]*modulePackage) []leakedImport {
	if parseReason, parseBad := classifyServerOnlyImport(parseImportPath); parseBad {
		return []leakedImport{{path: parseImportPath, reason: parseReason}}
	}
	parsePkg, parseLocal := parseGraph[parseImportPath]
	if !parseLocal {
		return nil
	}
	if !parsePkg.buildsUnderWasm {
		return []leakedImport{{
			path:   parseImportPath,
			reason: "this local package is constrained off js/wasm, so it cannot build for the client",
			chain:  []string{parseImportPath},
		}}
	}
	parseVisited := map[string]bool{parseImportPath: true}
	return walkPackageLeaks(parseImportPath, parseGraph, parseVisited, []string{parseImportPath})
}

// walkPackageLeaks recurses through a local package's wasm-included imports, accumulating the
// chain, and returns the server-only packages reachable from it. Visited packages are tracked
// to terminate on import cycles.
func walkPackageLeaks(parsePkgPath string, parseGraph map[string]*modulePackage, parseVisited map[string]bool, parseChain []string) []leakedImport {
	parsePkg := parseGraph[parsePkgPath]
	if parsePkg == nil {
		return nil
	}
	parseLeaks := []leakedImport{}
	for _, parseImportPath := range parsePkg.wasmImports {
		if parseReason, parseBad := classifyServerOnlyImport(parseImportPath); parseBad {
			parseLeaks = append(parseLeaks, leakedImport{path: parseImportPath, reason: parseReason, chain: append([]string(nil), parseChain...)})
			continue
		}
		parseChild, parseLocal := parseGraph[parseImportPath]
		if !parseLocal || parseVisited[parseImportPath] {
			continue
		}
		if !parseChild.buildsUnderWasm {
			parseLeaks = append(parseLeaks, leakedImport{
				path:   parseImportPath,
				reason: "this local package is constrained off js/wasm, so it cannot build for the client",
				chain:  append(append([]string(nil), parseChain...), parseImportPath),
			})
			continue
		}
		parseVisited[parseImportPath] = true
		parseLeaks = append(parseLeaks, walkPackageLeaks(parseImportPath, parseGraph, parseVisited, append(append([]string(nil), parseChain...), parseImportPath))...)
	}
	return parseLeaks
}

// dedupeLeakDiagnostics removes duplicate (file, line, leaked-path) findings that the
// transitive walk can produce when several entry files reach the same server package, and
// returns them in stable file/line order.
func dedupeLeakDiagnostics(parseDiagnostics []agenticDiagnostic) []agenticDiagnostic {
	parseSeen := map[string]bool{}
	parseOut := []agenticDiagnostic{}
	for _, parseDiagnostic := range parseDiagnostics {
		parseKey := parseDiagnostic.File + "|" + strconv.Itoa(parseDiagnostic.Line) + "|" + parseDiagnostic.Attributes["leaked"]
		if parseSeen[parseKey] {
			continue
		}
		parseSeen[parseKey] = true
		parseOut = append(parseOut, parseDiagnostic)
	}
	sort.SliceStable(parseOut, func(parseI int, parseJ int) bool {
		if parseOut[parseI].File == parseOut[parseJ].File {
			return parseOut[parseI].Line < parseOut[parseJ].Line
		}
		return parseOut[parseI].File < parseOut[parseJ].File
	})
	return parseOut
}

// modulePackage records the import-graph facts for one local package: whether any of its files
// build into the wasm target, and the de-duplicated imports of those wasm-included files.
type modulePackage struct {
	importPath      string
	buildsUnderWasm bool
	wasmImports     []string
}

// findModule locates the nearest go.mod at or above start and returns its directory and module
// path. When none is found it returns ("", "") and the analyzer treats every import as external
// (still classifying stdlib + third-party server packages).
func findModule(parseStart string) (string, string) {
	parseDir, parseErr := filepath.Abs(parseStart)
	if parseErr != nil {
		return "", ""
	}
	for {
		parseModFile := filepath.Join(parseDir, "go.mod")
		if parseData, parseReadErr := os.ReadFile(parseModFile); parseReadErr == nil {
			return parseDir, parseModulePathFromGoMod(parseData)
		}
		parseParent := filepath.Dir(parseDir)
		if parseParent == parseDir {
			return "", ""
		}
		parseDir = parseParent
	}
}

// parseModulePathFromGoMod extracts the module path from go.mod content.
func parseModulePathFromGoMod(parseData []byte) string {
	for parseLine := range strings.SplitSeq(string(parseData), "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if parseModulePath, parseFound := strings.CutPrefix(parseLine, "module "); parseFound {
			return strings.TrimSpace(parseModulePath)
		}
	}
	return ""
}

// buildModulePackageGraph indexes every local package under the module root by its import path,
// recording whether it builds under wasm and the imports of its wasm-included files. With no
// module root it returns an empty graph (all imports are then treated as external).
func buildModulePackageGraph(parseModuleRoot string, parseModulePath string) map[string]*modulePackage {
	parseGraph := map[string]*modulePackage{}
	if parseModuleRoot == "" || parseModulePath == "" {
		return parseGraph
	}
	parseFset := token.NewFileSet()
	_ = filepath.WalkDir(parseModuleRoot, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseModuleRoot {
				return filepath.SkipDir
			}
			return nil
		}
		parseName := parseEntry.Name()
		if !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			return nil
		}
		parseDir := filepath.Dir(parsePath)
		parseRel, parseRelErr := filepath.Rel(parseModuleRoot, parseDir)
		if parseRelErr != nil {
			return nil
		}
		parseImportPath := parseModulePath
		if parseRel != "." {
			parseImportPath = path.Join(parseModulePath, filepath.ToSlash(parseRel))
		}
		parsePkg := parseGraph[parseImportPath]
		if parsePkg == nil {
			parsePkg = &modulePackage{importPath: parseImportPath}
			parseGraph[parseImportPath] = parsePkg
		}
		if !fileBuildsUnderWasm(parsePath) {
			return nil
		}
		parsePkg.buildsUnderWasm = true
		parseFile, parseErr := parser.ParseFile(parseFset, parsePath, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		for _, parseImport := range parseFile.Imports {
			if parseImportPath, parseUnquoteErr := strconv.Unquote(parseImport.Path.Value); parseUnquoteErr == nil {
				parsePkg.wasmImports = appendUnique(parsePkg.wasmImports, parseImportPath)
			}
		}
		return nil
	})
	return parseGraph
}

// appendUnique appends value to slice only if absent, keeping import lists de-duplicated.
func appendUnique(parseSlice []string, parseValue string) []string {
	for _, parseExisting := range parseSlice {
		if parseExisting == parseValue {
			return parseSlice
		}
	}
	return append(parseSlice, parseValue)
}

// fileBuildsUnderWasm reports whether a file's //go:build constraint admits the js/wasm target
// (it is part of the client build). A file with no constraint (shared) builds everywhere, so it
// counts; a file constrained off js/wasm does not.
func fileBuildsUnderWasm(parsePath string) bool {
	parseExpr, parseHasConstraint := fileBuildConstraint(parsePath)
	if !parseHasConstraint {
		return true // shared file builds under every target, including wasm
	}
	return parseExpr.Eval(func(parseTag string) bool { return parseTag == "js" || parseTag == "wasm" })
}

// fileBuildConstraint parses a file's //go:build line, returning the expression and whether one
// was present (stopping at the first line of real code).
func fileBuildConstraint(parsePath string) (constraint.Expr, bool) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, false
	}
	for parseLine := range strings.SplitSeq(string(parseData), "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if !constraint.IsGoBuild(parseLine) {
			if parseLine != "" && !strings.HasPrefix(parseLine, "//") && !strings.HasPrefix(parseLine, "package") {
				return nil, false
			}
			continue
		}
		if parseExpr, parseParseErr := constraint.Parse(parseLine); parseParseErr == nil {
			return parseExpr, true
		}
		return nil, false
	}
	return nil, false
}

// fileIsClientOnly reports whether a file's //go:build constraint makes it browser-only:
// included when js && wasm are set, excluded for an untagged target. A file with no constraint
// (shared) or a server constraint returns false — only explicitly client-only files seed the
// leak walk, so shared utilities are never false-positived.
func fileIsClientOnly(parsePath string) bool {
	parseExpr, parseHasConstraint := fileBuildConstraint(parsePath)
	if !parseHasConstraint {
		return false
	}
	parseClient := parseExpr.Eval(func(parseTag string) bool { return parseTag == "js" || parseTag == "wasm" })
	parseUntagged := parseExpr.Eval(func(string) bool { return false })
	return parseClient && !parseUntagged
}
