package errcode

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// guardedPackageDirs lists the core package directories (relative to this
// package) whose non-test panics must carry a registered code. interop and the
// remaining packages are excluded for now.
var guardedPackageDirs = []string{
	"../../ui",
	"../../router",
	"../../state",
	"../../internal/runtime",
	"../../fetch",
	"../../serverfn",
}

// guardOptOutMarker exempts one panic line; it must be followed by a reason.
const guardOptOutMarker = "//errcode:ok"

// uncodedPanicAllowlist names the panic sites that predate the error-code
// registry and still panic with a bare string or error, keyed "dir/file.go:Func".
// The list may only shrink: convert a site to errcode.New (or ActionableFrameworkPanic
// with a Code) and delete its entry. TestUncodedPanicAllowlistHasNoStaleEntries
// fails when an entry no longer matches a real uncoded panic.
var uncodedPanicAllowlist = map[string]string{}

// isCodedPanicArg reports whether the panic argument is built by the registry
// (errcode.New), by the actionable-panic helpers that resolve a code, by the
// markUnhandledPanic helpers that stamp a GWC-RUNTIME-PANIC-* phase code, or is
// a framework control-flow signal (NewSuspension). A bare identifier whose name
// contains "Recovered" or "Original" is a re-panic of an already-reported value.
func isCodedPanicArg(parseArg ast.Expr) bool {
	if parseIdent, parseIsIdent := parseArg.(*ast.Ident); parseIsIdent {
		return strings.Contains(parseIdent.Name, "Recovered") || strings.Contains(parseIdent.Name, "Original")
	}
	parseCall, parseOk := parseArg.(*ast.CallExpr)
	if !parseOk {
		return false
	}
	switch parseFun := parseCall.Fun.(type) {
	case *ast.SelectorExpr:
		if parseIdent, parseIsIdent := parseFun.X.(*ast.Ident); parseIsIdent && parseIdent.Name == "errcode" && parseFun.Sel.Name == "New" {
			return true
		}
		return isCodedPanicHelperName(parseFun.Sel.Name)
	case *ast.Ident:
		return isCodedPanicHelperName(parseFun.Name)
	}
	return false
}

// isCodedPanicHelperName reports whether a helper function name is one of the
// framework helpers whose result already carries a registered code.
func isCodedPanicHelperName(parseName string) bool {
	return parseName == "ActionableFrameworkPanic" || parseName == "NewSuspension" ||
		strings.HasPrefix(parseName, "actionable") || strings.HasPrefix(parseName, "markUnhandledPanic")
}

// scanUncodedPanics returns "dir/file.go:Func" -> source positions for every
// non-test panic in the guarded packages whose argument is not coded and whose
// line carries no opt-out marker.
func scanUncodedPanics(parseT *testing.T) map[string][]string {
	parseT.Helper()
	parseFound := map[string][]string{}
	for _, parseDir := range guardedPackageDirs {
		parseFiles, parseErr := filepath.Glob(filepath.Join(filepath.FromSlash(parseDir), "*.go"))
		if parseErr != nil {
			parseT.Fatalf("glob %s: %v", parseDir, parseErr)
		}
		for _, parsePath := range parseFiles {
			if strings.HasSuffix(parsePath, "_test.go") {
				continue
			}
			parseSource, parseReadErr := os.ReadFile(parsePath)
			if parseReadErr != nil {
				parseT.Fatalf("read %s: %v", parsePath, parseReadErr)
			}
			parseLines := strings.Split(string(parseSource), "\n")
			parseSet := token.NewFileSet()
			parseFile, parseParseErr := parser.ParseFile(parseSet, parsePath, parseSource, parser.ParseComments)
			if parseParseErr != nil {
				parseT.Fatalf("parse %s: %v", parsePath, parseParseErr)
			}
			parseKeyPrefix := filepath.Base(parseDir) + "/" + filepath.Base(parsePath) + ":"
			if filepath.Base(parseDir) == "runtime" {
				parseKeyPrefix = "internal/runtime/" + filepath.Base(parsePath) + ":"
			}
			for _, parseDecl := range parseFile.Decls {
				parseName := "<init>"
				if parseFunc, parseIsFunc := parseDecl.(*ast.FuncDecl); parseIsFunc {
					parseName = parseFunc.Name.Name
				}
				ast.Inspect(parseDecl, func(parseNode ast.Node) bool {
					parseCall, parseIsCall := parseNode.(*ast.CallExpr)
					if !parseIsCall {
						return true
					}
					parseIdent, parseIsIdent := parseCall.Fun.(*ast.Ident)
					if !parseIsIdent || parseIdent.Name != "panic" || len(parseCall.Args) != 1 {
						return true
					}
					if isCodedPanicArg(parseCall.Args[0]) {
						return true
					}
					parsePos := parseSet.Position(parseCall.Pos())
					if parsePos.Line-1 < len(parseLines) && strings.Contains(parseLines[parsePos.Line-1], guardOptOutMarker) {
						return true
					}
					parseKey := parseKeyPrefix + parseName
					parseFound[parseKey] = append(parseFound[parseKey], parsePos.String())
					return true
				})
			}
		}
	}
	return parseFound
}

// TestNoNewUncodedPanics fails on a non-test panic(<expr>) in the guarded
// packages that is neither errcode.New / ActionableFrameworkPanic nor on the
// shrinking allowlist nor marked `//errcode:ok <reason>`.
func TestNoNewUncodedPanics(parseT *testing.T) {
	parseFound := scanUncodedPanics(parseT)
	var parseViolations []string
	for parseKey, parsePositions := range parseFound {
		if _, parseAllowed := uncodedPanicAllowlist[parseKey]; !parseAllowed {
			parseViolations = append(parseViolations, parseKey+" at "+strings.Join(parsePositions, ", "))
		}
	}
	sort.Strings(parseViolations)
	if len(parseViolations) > 0 {
		parseT.Fatalf("uncoded panic(s): use errcode.New(code, ...) or ActionableFrameworkPanic with a Code, or annotate the line with %q and a reason:\n  %s", guardOptOutMarker+" <reason>", strings.Join(parseViolations, "\n  "))
	}
}

// TestUncodedPanicAllowlistHasNoStaleEntries keeps the allowlist honest: once a
// site is converted its entry must be deleted so the list only shrinks.
func TestUncodedPanicAllowlistHasNoStaleEntries(parseT *testing.T) {
	parseFound := scanUncodedPanics(parseT)
	for parseKey := range uncodedPanicAllowlist {
		if _, parseStillPanics := parseFound[parseKey]; !parseStillPanics {
			parseT.Errorf("stale allowlist entry %s: the site is coded or gone, delete it", parseKey)
		}
	}
}

// uncodedErrorDirs lists the package directories whose non-test fmt.Errorf and
// errors.New calls are counted by TestNoNewUncodedErrors.
var uncodedErrorDirs = append(append([]string{}, guardedPackageDirs...), "../../interop")

// uncodedErrorAllowlist holds the number of uncoded fmt.Errorf / errors.New
// calls each guarded package still has, keyed by package directory name
// ("runtime" is internal/runtime). The counts may only shrink: convert a site to
// errcode.New with a registered code (or delete it) and lower the number.
// Sentinels compared with errors.Is, wrapped causes inside a coded error, and
// internal control-flow errors are what remain.
var uncodedErrorAllowlist = map[string]int{
	"fetch":    36,
	"interop":  158,
	"router":   21,
	"runtime":  38,
	"serverfn": 5,
	"state":    0,
	"ui":       4,
}

// countUncodedErrors returns, per package directory name, the number of
// fmt.Errorf and errors.New calls in non-test files.
func countUncodedErrors(parseT *testing.T) map[string]int {
	parseT.Helper()
	parseCounts := map[string]int{}
	for _, parseDir := range uncodedErrorDirs {
		parseFiles, parseErr := filepath.Glob(filepath.Join(filepath.FromSlash(parseDir), "*.go"))
		if parseErr != nil {
			parseT.Fatalf("glob %s: %v", parseDir, parseErr)
		}
		parseName := filepath.Base(parseDir)
		parseCounts[parseName] += 0
		for _, parsePath := range parseFiles {
			if strings.HasSuffix(parsePath, "_test.go") {
				continue
			}
			parseSource, parseReadErr := os.ReadFile(parsePath)
			if parseReadErr != nil {
				parseT.Fatalf("read %s: %v", parsePath, parseReadErr)
			}
			parseFile, parseParseErr := parser.ParseFile(token.NewFileSet(), parsePath, parseSource, 0)
			if parseParseErr != nil {
				parseT.Fatalf("parse %s: %v", parsePath, parseParseErr)
			}
			ast.Inspect(parseFile, func(parseNode ast.Node) bool {
				parseCall, parseIsCall := parseNode.(*ast.CallExpr)
				if !parseIsCall {
					return true
				}
				parseSel, parseIsSel := parseCall.Fun.(*ast.SelectorExpr)
				if !parseIsSel {
					return true
				}
				parsePkg, parseIsIdent := parseSel.X.(*ast.Ident)
				if !parseIsIdent {
					return true
				}
				if (parsePkg.Name == "fmt" && parseSel.Sel.Name == "Errorf") || (parsePkg.Name == "errors" && parseSel.Sel.Name == "New") {
					parseCounts[parseName]++
				}
				return true
			})
		}
	}
	return parseCounts
}

// TestNoNewUncodedErrors is the second ratchet: the number of uncoded
// fmt.Errorf / errors.New calls in each guarded package must not exceed the
// allowlist. Convert the site to errcode.New instead of raising the number.
func TestNoNewUncodedErrors(parseT *testing.T) {
	parseCounts := countUncodedErrors(parseT)
	for parseName, parseCount := range parseCounts {
		parseAllowed := uncodedErrorAllowlist[parseName]
		if parseCount > parseAllowed {
			parseT.Errorf("package %s has %d uncoded fmt.Errorf/errors.New calls, allowlist is %d: use errcode.New(code, ...) with a registered code", parseName, parseCount, parseAllowed)
		}
		if parseCount < parseAllowed {
			parseT.Logf("package %s now has %d uncoded errors; lower uncodedErrorAllowlist[%q] from %d", parseName, parseCount, parseName, parseAllowed)
		}
	}
}

// TestRegistryEntriesHaveDocsText asserts the fields the reference page and the
// remediation line depend on are all non-empty for every registered code.
func TestRegistryEntriesHaveDocsText(parseT *testing.T) {
	for _, parseEntry := range All() {
		if strings.TrimSpace(parseEntry.Headline) == "" {
			parseT.Errorf("%s has an empty headline", parseEntry.Code)
		}
		if strings.TrimSpace(parseEntry.Remediation) == "" {
			parseT.Errorf("%s has an empty remediation", parseEntry.Code)
		}
		if strings.TrimSpace(parseEntry.Docs) == "" || !strings.HasPrefix(parseEntry.Docs, ReferencePagePath+"#") {
			parseT.Errorf("%s has a missing or malformed docs ref %q", parseEntry.Code, parseEntry.Docs)
		}
	}
}
