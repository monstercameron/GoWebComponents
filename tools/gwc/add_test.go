package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryCatalogComponentRendersValidGo proves the `gwc add` smoke contract: every catalog
// component, once copied into a destination package, is syntactically valid Go that parses
// clean (the package-rewrite never corrupts the source). The real cross-module `go build` runs
// in the catalog-smoke CI workflow; this is its fast, offline guard so a broken template fails
// the unit lane immediately.
func TestEveryCatalogComponentRendersValidGo(parseT *testing.T) {
	if len(componentRegistry) != 5 {
		parseT.Fatalf("expected the 5-component v4 catalog, got %d", len(componentRegistry))
	}
	for _, parseEntry := range componentRegistry {
		parseSource, parseErr := componentTemplatesFS.ReadFile(parseEntry.file)
		if parseErr != nil {
			parseT.Fatalf("component %q: unreadable: %v", parseEntry.name, parseErr)
		}
		parseRendered, parseErr := renderComponentSource(string(parseSource), "uikit", parseEntry.name)
		if parseErr != nil {
			parseT.Fatalf("component %q: render failed: %v", parseEntry.name, parseErr)
		}
		if _, parseErr := parser.ParseFile(token.NewFileSet(), parseEntry.name+".go", parseRendered, parser.AllErrors); parseErr != nil {
			parseT.Fatalf("component %q: rendered source is not valid Go: %v", parseEntry.name, parseErr)
		}
		if !strings.Contains(parseRendered, "package uikit") {
			parseT.Fatalf("component %q: destination package not applied", parseEntry.name)
		}
	}
}

// TestComponentRegistryTemplatesEmbedAndCompileMarkers proves every catalog entry has an
// TestScaffoldComponentGeneratesValidGo proves `gwc add component <Name>` emits parseable,
// idiomatic Go: a typed FooProps struct and a Foo(props FooProps) ui.Node component.
func TestScaffoldComponentGeneratesValidGo(parseT *testing.T) {
	parseSource, parseErr := scaffoldComponentSource("widgets", "Foo")
	if parseErr != nil {
		parseT.Fatalf("scaffoldComponentSource: %v", parseErr)
	}
	if _, parseErr := parser.ParseFile(token.NewFileSet(), "foo.go", parseSource, parser.AllErrors); parseErr != nil {
		parseT.Fatalf("generated component does not parse: %v\n%s", parseErr, parseSource)
	}
	for _, parseWant := range []string{"package widgets", "type FooProps struct", "func Foo(props FooProps) ui.Node", "ui.UseState(0)"} {
		if !strings.Contains(parseSource, parseWant) {
			parseT.Fatalf("generated component missing %q:\n%s", parseWant, parseSource)
		}
	}
}

// TestScaffoldRouteGeneratesValidGo proves `gwc add route <Name> -path /p` emits parseable Go
// with a route contract var (which `gwc routes gen` will pick up) and its route component.
func TestScaffoldRouteGeneratesValidGo(parseT *testing.T) {
	parseSource, parseErr := scaffoldRouteSource("routes", "UserDetail", "/users/:id")
	if parseErr != nil {
		parseT.Fatalf("scaffoldRouteSource: %v", parseErr)
	}
	if _, parseErr := parser.ParseFile(token.NewFileSet(), "userdetail.go", parseSource, parser.AllErrors); parseErr != nil {
		parseT.Fatalf("generated route does not parse: %v\n%s", parseErr, parseSource)
	}
	for _, parseWant := range []string{`var UserDetailRoute = router.MustDefineRoute("/users/:id")`, "func UserDetail() ui.Node"} {
		if !strings.Contains(parseSource, parseWant) {
			parseT.Fatalf("generated route missing %q:\n%s", parseWant, parseSource)
		}
	}
}

// TestRunAddGeneratorWritesFiles proves the generator dispatch writes the file to disk under the
// chosen directory with the lower-cased entity filename.
func TestRunAddGeneratorWritesFiles(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseLauncher := launcher{}
	if parseErr := parseLauncher.runAdd([]string{"component", "Foo", "-dir", parseDir, "-pkg", "widgets"}); parseErr != nil {
		parseT.Fatalf("gwc add component: %v", parseErr)
	}
	if _, parseErr := os.Stat(filepath.Join(parseDir, "foo.go")); parseErr != nil {
		parseT.Fatalf("expected foo.go written: %v", parseErr)
	}
	if parseErr := parseLauncher.runAdd([]string{"route", "UserDetail", "-path", "/users/:id", "-dir", parseDir}); parseErr != nil {
		parseT.Fatalf("gwc add route: %v", parseErr)
	}
	if _, parseErr := os.Stat(filepath.Join(parseDir, "userdetail.go")); parseErr != nil {
		parseT.Fatalf("expected userdetail.go written: %v", parseErr)
	}
}

// TestValidPackageName proves the package-name fallback: a base starting with a digit (e.g. a
// temp-dir segment "001" or "2fa") or an empty base is not a legal Go identifier, so it falls
// back to the default; a normal base is lower-cased and kept.
func TestValidPackageName(parseT *testing.T) {
	parseCases := []struct {
		base     string
		fallback string
		want     string
	}{
		{"2fa", "routes", "routes"},
		{"001", "components", "components"},
		{"", "routes", "routes"},
		{"MyRoutes", "routes", "myroutes"},
		{"widgets", "components", "widgets"},
	}
	for _, parseCase := range parseCases {
		if parseGot := validPackageName(parseCase.base, parseCase.fallback); parseGot != parseCase.want {
			parseT.Fatalf("validPackageName(%q, %q) = %q, want %q", parseCase.base, parseCase.fallback, parseGot, parseCase.want)
		}
	}
}

// embedded template that is non-empty and declares the catalog package (a smoke check that
// the embed paths are correct; the templates themselves compile as tools/gwc/templates).
func TestComponentRegistryTemplatesEmbed(parseT *testing.T) {
	for _, parseEntry := range componentRegistry {
		parseSource, parseErr := componentTemplatesFS.ReadFile(parseEntry.file)
		if parseErr != nil {
			parseT.Fatalf("component %q: embedded file %q unreadable: %v", parseEntry.name, parseEntry.file, parseErr)
		}
		if !strings.Contains(string(parseSource), "package components") {
			parseT.Fatalf("component %q: template should be in package components", parseEntry.name)
		}
	}
}

// TestRenderComponentSourceRewritesPackage proves the copied source drops the catalog
// package doc, sets the destination package, adds a provenance header, and is gofmt-clean.
func TestRenderComponentSourceRewritesPackage(parseT *testing.T) {
	parseSource, _ := componentTemplatesFS.ReadFile("templates/disclosure.go")
	parseRendered, parseErr := renderComponentSource(string(parseSource), "ui_kit", "disclosure")
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseRendered, "package ui_kit") {
		parseT.Fatalf("expected destination package, got:\n%s", parseRendered)
	}
	if strings.Contains(parseRendered, "package components") {
		parseT.Fatal("catalog package name should be rewritten away")
	}
	if !strings.HasPrefix(parseRendered, "// Code added by `gwc add disclosure`") {
		parseT.Fatalf("expected provenance header, got:\n%.120s", parseRendered)
	}
	if !strings.Contains(parseRendered, "func Disclosure(") {
		parseT.Fatal("expected the Disclosure component to be present")
	}
}

// TestLookupComponentUnknown proves an unknown name is reported, not silently ignored.
func TestLookupComponentUnknown(parseT *testing.T) {
	if _, parseOk := lookupComponent("does-not-exist"); parseOk {
		parseT.Fatal("expected unknown component to be absent")
	}
	if _, parseOk := lookupComponent("tabs"); !parseOk {
		parseT.Fatal("expected tabs to be in the catalog")
	}
}

// TestRunAddWritesComponent proves the end-to-end copy: the file lands at <dir>/<name>.go
// with the destination package, and a second add without -force is refused.
func TestRunAddWritesComponent(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseDest := filepath.Join(parseDir, "widgets")

	parseLauncher := launcher{}
	if parseErr := parseLauncher.runAdd([]string{"tabs", "-dir", parseDest}); parseErr != nil {
		parseT.Fatalf("runAdd: %v", parseErr)
	}
	parseOut := filepath.Join(parseDest, "tabs.go")
	parseData, parseErr := os.ReadFile(parseOut)
	if parseErr != nil {
		parseT.Fatalf("expected %s to be written: %v", parseOut, parseErr)
	}
	if !strings.Contains(string(parseData), "package widgets") || !strings.Contains(string(parseData), "func Tabs(") {
		parseT.Fatalf("unexpected written content:\n%s", parseData)
	}

	// Without -force, re-adding must refuse rather than clobber.
	if parseErr := parseLauncher.runAdd([]string{"tabs", "-dir", parseDest}); parseErr == nil {
		parseT.Fatal("expected a refusal to overwrite without -force")
	}
	// With -force, it succeeds.
	if parseErr := parseLauncher.runAdd([]string{"tabs", "-dir", parseDest, "-force"}); parseErr != nil {
		parseT.Fatalf("expected -force overwrite to succeed: %v", parseErr)
	}
}
