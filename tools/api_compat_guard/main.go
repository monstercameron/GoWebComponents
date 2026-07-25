package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultBaselinePath = "tools/api_compat_guard/api_compatibility_baseline.json"
	defaultPackageList  = "fetch,flags,router,state,ui,projection,domain,delta,compute,trace,gcpacing,escalate,db/offthread,db/durability"
	defaultTargetList   = "native,wasm"
)

type apiBaseline struct {
	GeneratedAtUTC string                    `json:"generated_at_utc,omitempty"`
	Scope          []string                  `json:"scope,omitempty"`
	Targets        map[string]targetBaseline `json:"targets"`
}

type targetBaseline struct {
	GOOS      string              `json:"goos"`
	GOARCH    string              `json:"goarch"`
	BuildTags []string            `json:"build_tags,omitempty"`
	Packages  map[string][]string `json:"packages"`
}

type targetSpec struct {
	Name      string
	GOOS      string
	GOARCH    string
	BuildTags []string
}

type compatIssue struct {
	Target  string
	Package string
	Symbol  string
}

func main() {
	var baselinePath string
	var moduleRoot string
	var packageList string
	var targetList string
	var update bool

	flag.StringVar(&baselinePath, "baseline", defaultBaselinePath, "path to the API compatibility baseline")
	flag.StringVar(&moduleRoot, "module-root", ".", "module root to scan")
	flag.StringVar(&packageList, "packages", defaultPackageList, "comma-separated public package directories to scan")
	flag.StringVar(&targetList, "targets", defaultTargetList, "comma-separated targets: native, wasm, or name=goos/goarch[:tag+tag]")
	flag.BoolVar(&update, "update", false, "rewrite the baseline with the currently exported API surface")
	flag.Parse()

	if err := run(moduleRoot, baselinePath, packageList, targetList, update); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(moduleRoot string, baselinePath string, packageList string, targetList string, update bool) error {
	root, err := filepath.Abs(moduleRoot)
	if err != nil {
		return fmt.Errorf("resolve module root: %w", err)
	}

	packages := parseCommaList(packageList)
	if len(packages) == 0 {
		return errors.New("at least one package must be provided")
	}

	targets, err := parseTargets(targetList)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return errors.New("at least one target must be provided")
	}

	current, err := collectBaseline(root, packages, targets)
	if err != nil {
		return err
	}

	if update {
		current.GeneratedAtUTC = time.Now().UTC().Format(time.RFC3339)
		current.Scope = packages
		return writeBaseline(resolvePath(root, baselinePath), current)
	}

	expected, err := readBaseline(resolvePath(root, baselinePath))
	if err != nil {
		return err
	}

	issues := compareBaselines(expected, current)
	if len(issues) > 0 {
		return fmt.Errorf("API compatibility guard failed:\n%s\n\nRun `go run ./tools/api_compat_guard -update` only for intentional public API changes.", formatIssues(issues))
	}

	fmt.Printf("API compatibility guard passed for %d target(s), %d package(s).\n", len(targets), len(packages))
	return nil
}

func collectBaseline(moduleRoot string, packages []string, targets []targetSpec) (apiBaseline, error) {
	result := apiBaseline{
		Scope:   append([]string(nil), packages...),
		Targets: make(map[string]targetBaseline, len(targets)),
	}

	for _, target := range targets {
		targetResult := targetBaseline{
			GOOS:      target.GOOS,
			GOARCH:    target.GOARCH,
			BuildTags: append([]string(nil), target.BuildTags...),
			Packages:  make(map[string][]string, len(packages)),
		}

		for _, pkg := range packages {
			symbols, err := scanPackage(moduleRoot, pkg, target)
			if err != nil {
				return apiBaseline{}, fmt.Errorf("scan %s for %s: %w", pkg, target.Name, err)
			}
			targetResult.Packages[pkg] = symbols
		}

		result.Targets[target.Name] = targetResult
	}

	return result, nil
}

func scanPackage(moduleRoot string, packagePath string, target targetSpec) ([]string, error) {
	dir := filepath.Join(moduleRoot, filepath.FromSlash(packagePath))
	ctx := build.Default
	ctx.GOOS = target.GOOS
	ctx.GOARCH = target.GOARCH
	ctx.BuildTags = append([]string(nil), target.BuildTags...)

	buildPkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		return nil, err
	}

	files := append([]string{}, buildPkg.GoFiles...)
	files = append(files, buildPkg.CgoFiles...)
	sort.Strings(files)

	symbols := map[string]struct{}{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if err := scanGoFile(filepath.Join(dir, name), symbols); err != nil {
			return nil, err
		}
	}

	out := make([]string, 0, len(symbols))
	for symbol := range symbols {
		out = append(out, symbol)
	}
	sort.Strings(out)
	return out, nil
}

func scanGoFile(path string, symbols map[string]struct{}) error {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			scanGenDecl(typed, symbols)
		case *ast.FuncDecl:
			scanFuncDecl(typed, symbols)
		}
	}

	return nil
}

func scanGenDecl(decl *ast.GenDecl, symbols map[string]struct{}) {
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *ast.TypeSpec:
			if !typed.Name.IsExported() {
				continue
			}
			typeName := typed.Name.Name
			symbols["type "+typeName] = struct{}{}
			scanTypeMembers(typeName, typed.Type, symbols)
		case *ast.ValueSpec:
			kind := "var"
			if decl.Tok == token.CONST {
				kind = "const"
			}
			for _, name := range typed.Names {
				if name.IsExported() {
					symbols[kind+" "+name.Name] = struct{}{}
				}
			}
		}
	}
}

func scanTypeMembers(typeName string, expr ast.Expr, symbols map[string]struct{}) {
	switch typed := expr.(type) {
	case *ast.StructType:
		for _, field := range typed.Fields.List {
			if len(field.Names) == 0 {
				if embedded, ok := exportedEmbeddedFieldName(field.Type); ok {
					symbols["field "+typeName+"."+embedded] = struct{}{}
				}
				continue
			}
			for _, name := range field.Names {
				if name.IsExported() {
					symbols["field "+typeName+"."+name.Name] = struct{}{}
					// The field's TYPE is part of the struct's public contract:
					// changing `F int` -> `F string` is breaking, yet the name-only
					// entry misses it. Record the type too.
					symbols["fieldtype "+typeName+"."+name.Name+" "+normalizeSignature(exprString(field.Type))] = struct{}{}
				}
			}
		}
	case *ast.InterfaceType:
		for _, field := range typed.Methods.List {
			if len(field.Names) == 0 {
				if embedded, ok := exportedEmbeddedFieldName(field.Type); ok {
					symbols["interface "+typeName+"."+embedded] = struct{}{}
				}
				continue
			}
			for _, name := range field.Names {
				if name.IsExported() {
					symbols["interface "+typeName+"."+name.Name] = struct{}{}
					// The method signature is the interface contract; a changed
					// signature is breaking for every implementer.
					symbols["interfacesig "+typeName+"."+name.Name+" "+normalizeSignature(exprString(field.Type))] = struct{}{}
				}
			}
		}
	}
}

func scanFuncDecl(decl *ast.FuncDecl, symbols map[string]struct{}) {
	if !decl.Name.IsExported() {
		return
	}
	// Record the signature alongside the name so a BREAKING change (added/removed
	// param, changed param or result type) is caught. The name-only entry catches
	// removal; the signature entry catches an incompatible reshape that keeps the
	// name. exprString(decl.Type) renders the full func type incl. generic type
	// params, e.g. "func[T any](a int) error".
	sig := normalizeSignature(exprString(decl.Type))
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		symbols["func "+decl.Name.Name] = struct{}{}
		symbols["funcsig "+decl.Name.Name+" "+sig] = struct{}{}
		return
	}
	recv := exprString(decl.Recv.List[0].Type)
	symbols[fmt.Sprintf("method (%s) %s", recv, decl.Name.Name)] = struct{}{}
	symbols[fmt.Sprintf("methodsig (%s) %s %s", recv, decl.Name.Name, sig)] = struct{}{}
}

func exportedEmbeddedFieldName(expr ast.Expr) (string, bool) {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name, typed.IsExported()
	case *ast.StarExpr:
		return exportedEmbeddedFieldName(typed.X)
	case *ast.SelectorExpr:
		return typed.Sel.Name, typed.Sel.IsExported()
	case *ast.IndexExpr:
		return exportedEmbeddedFieldName(typed.X)
	case *ast.IndexListExpr:
		return exportedEmbeddedFieldName(typed.X)
	default:
		return "", false
	}
}

func exprString(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), expr); err != nil {
		return "<invalid>"
	}
	return buf.String()
}

// normalizeSignature collapses all whitespace runs to single spaces so a
// multi-line rendering of a func/field type compares as one stable line.
func normalizeSignature(source string) string {
	return strings.Join(strings.Fields(source), " ")
}

// apiBaselineDriftSymbol marks a package (or target) that the current scan
// covers but the baseline does not. Adding a symbol to a tracked package is a
// non-breaking change and is allowed, but a package with NO baseline at all is a
// silent coverage gap: its future breaking changes would pass unnoticed until
// someone regenerates the baseline. Flag it so the guard fails and the baseline
// is brought back in sync.
const apiBaselineDriftSymbol = "(package absent from baseline — run api_compat_guard -update to track it)"

func compareBaselines(expected apiBaseline, current apiBaseline) []compatIssue {
	var issues []compatIssue
	for targetName, expectedTarget := range expected.Targets {
		currentTarget, ok := current.Targets[targetName]
		if !ok {
			for pkg, symbols := range expectedTarget.Packages {
				for _, symbol := range symbols {
					issues = append(issues, compatIssue{Target: targetName, Package: pkg, Symbol: symbol})
				}
			}
			continue
		}

		for pkg, expectedSymbols := range expectedTarget.Packages {
			currentSymbols := map[string]struct{}{}
			for _, symbol := range currentTarget.Packages[pkg] {
				currentSymbols[symbol] = struct{}{}
			}
			for _, symbol := range expectedSymbols {
				if _, ok := currentSymbols[symbol]; !ok {
					issues = append(issues, compatIssue{Target: targetName, Package: pkg, Symbol: symbol})
				}
			}
		}

		// Package drift within a tracked target.
		for pkg := range currentTarget.Packages {
			if _, ok := expectedTarget.Packages[pkg]; !ok {
				issues = append(issues, compatIssue{Target: targetName, Package: pkg, Symbol: apiBaselineDriftSymbol})
			}
		}
	}

	// Target drift: a whole target the current scan covers but the baseline omits.
	for targetName, currentTarget := range current.Targets {
		if _, ok := expected.Targets[targetName]; ok {
			continue
		}
		for pkg := range currentTarget.Packages {
			issues = append(issues, compatIssue{Target: targetName, Package: pkg, Symbol: apiBaselineDriftSymbol})
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		left := issues[i]
		right := issues[j]
		if left.Target != right.Target {
			return left.Target < right.Target
		}
		if left.Package != right.Package {
			return left.Package < right.Package
		}
		return left.Symbol < right.Symbol
	})
	return issues
}

func formatIssues(issues []compatIssue) string {
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("- %s/%s missing %s", issue.Target, issue.Package, issue.Symbol))
	}
	return strings.Join(lines, "\n")
}

func readBaseline(path string) (apiBaseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return apiBaseline{}, fmt.Errorf("read baseline: %w", err)
	}
	var baseline apiBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return apiBaseline{}, fmt.Errorf("parse baseline: %w", err)
	}
	// Reject an empty baseline: compareBaselines only walks expected.Targets, so a
	// baseline truncated to "{}" (bad merge, botched -update, overwrite) would
	// iterate zero times and the guard would "pass" even if every exported symbol
	// were deleted — a silent false-pass in a compatibility gate.
	if len(baseline.Targets) == 0 {
		return apiBaseline{}, fmt.Errorf("baseline %s has no targets; refusing to run (an empty baseline passes unconditionally)", path)
	}
	return baseline, nil
}

func writeBaseline(path string, baseline apiBaseline) error {
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("encode baseline: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create baseline directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}
	return nil
}

func resolvePath(root string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func parseCommaList(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func parseTargets(value string) ([]targetSpec, error) {
	items := parseCommaList(value)
	targets := make([]targetSpec, 0, len(items))
	for _, item := range items {
		switch item {
		case "native":
			targets = append(targets, targetSpec{Name: "native", GOOS: "linux", GOARCH: "amd64"})
			continue
		case "wasm":
			targets = append(targets, targetSpec{Name: "wasm", GOOS: "js", GOARCH: "wasm"})
			continue
		}

		parsed, err := parseExplicitTarget(item)
		if err != nil {
			return nil, err
		}
		targets = append(targets, parsed)
	}
	return targets, nil
}

func parseExplicitTarget(value string) (targetSpec, error) {
	name, rest, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(rest) == "" {
		return targetSpec{}, fmt.Errorf("invalid target %q: use native, wasm, or name=goos/goarch[:tag+tag]", value)
	}

	goPair, tagPart, _ := strings.Cut(rest, ":")
	goos, goarch, ok := strings.Cut(goPair, "/")
	if !ok || strings.TrimSpace(goos) == "" || strings.TrimSpace(goarch) == "" {
		return targetSpec{}, fmt.Errorf("invalid target %q: expected name=goos/goarch[:tag+tag]", value)
	}

	var tags []string
	if tagPart != "" {
		tags = strings.Split(tagPart, "+")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		tags = parseCommaList(strings.Join(tags, ","))
	}

	return targetSpec{
		Name:      strings.TrimSpace(name),
		GOOS:      strings.TrimSpace(goos),
		GOARCH:    strings.TrimSpace(goarch),
		BuildTags: tags,
	}, nil
}
