package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// componentTemplatesFS embeds the headless component catalog. The templates compile as a
// real package (tools/gwc/templates) so the catalog can never ship a component that does
// not build.
//
//go:embed templates/alert.go templates/breadcrumb.go templates/disclosure.go templates/switch_toggle.go templates/tabs.go
var componentTemplatesFS embed.FS

// runAddCommand routes the `gwc add` headless-component registry command.
var runAddCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runAdd(parseArgs)
}

// componentTemplate is one catalog entry: an a11y-correct component copied into the user's
// repo (the shadcn "own the code" model).
type componentTemplate struct {
	name    string
	summary string
	file    string
}

// componentRegistry is the catalog `gwc add` can install, sorted by name.
var componentRegistry = []componentTemplate{
	{name: "alert", summary: "Accessible status message (WAI-ARIA alert: role=alert + aria-live)", file: "templates/alert.go"},
	{name: "breadcrumb", summary: "Accessible navigation trail (WAI-ARIA breadcrumb: nav + aria-current)", file: "templates/breadcrumb.go"},
	{name: "disclosure", summary: "Accessible show/hide region (WAI-ARIA disclosure: aria-expanded/-controls)", file: "templates/disclosure.go"},
	{name: "switch", summary: "Accessible on/off control (WAI-ARIA switch: role=switch + aria-checked)", file: "templates/switch_toggle.go"},
	{name: "tabs", summary: "Accessible tabs (WAI-ARIA tablist + tabpanel, roving tab stop)", file: "templates/tabs.go"},
}

// lookupComponent finds a catalog entry by name.
func lookupComponent(parseName string) (componentTemplate, bool) {
	for _, parseEntry := range componentRegistry {
		if parseEntry.name == parseName {
			return parseEntry, true
		}
	}
	return componentTemplate{}, false
}

// runAdd parses `gwc add [name] [-dir DIR] [-pkg PKG] [-force]` and copies a catalog
// component into the project, or lists the catalog when no name is given.
func (parseL launcher) runAdd(parseArgs []string) error {
	parseName := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseName = parseArgs[0]
		parseArgs = parseArgs[1:]
	}

	parseFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseDir := parseFlags.String("dir", "components", "Directory to write the component into")
	parsePkg := parseFlags.String("pkg", "", "Package name for the generated file; defaults to the destination directory's base name")
	parseForce := parseFlags.Bool("force", false, "Overwrite an existing file")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	if parseName == "" {
		printComponentCatalog()
		return nil
	}

	parseEntry, parseOk := lookupComponent(parseName)
	if !parseOk {
		return fmt.Errorf("unknown component %q; run `gwc add` to list the catalog", parseName)
	}

	parseSource, parseErr := componentTemplatesFS.ReadFile(parseEntry.file)
	if parseErr != nil {
		return fmt.Errorf("read embedded template %q: %w", parseEntry.file, parseErr)
	}

	parsePkgName := strings.TrimSpace(*parsePkg)
	if parsePkgName == "" {
		parsePkgName = sanitizeGoIdent(filepath.Base(*parseDir))
		if parsePkgName == "" {
			parsePkgName = "components"
		}
	}

	parseRendered, parseErr := renderComponentSource(string(parseSource), parsePkgName, parseEntry.name)
	if parseErr != nil {
		return parseErr
	}

	if parseErr := os.MkdirAll(*parseDir, 0755); parseErr != nil {
		return fmt.Errorf("create %s: %w", *parseDir, parseErr)
	}
	parseOutPath := filepath.Join(*parseDir, parseEntry.name+".go")
	if _, parseStatErr := os.Stat(parseOutPath); parseStatErr == nil && !*parseForce {
		return fmt.Errorf("%s already exists; pass -force to overwrite", parseOutPath)
	}
	if parseErr := os.WriteFile(parseOutPath, []byte(parseRendered), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", parseOutPath, parseErr)
	}

	fmt.Printf("GWC add: wrote %s (package %s). You own this file — edit freely.\n", parseOutPath, parsePkgName)
	return nil
}

// renderComponentSource rewrites an embedded template for the destination package: it
// drops the catalog package doc, sets the package clause, prepends a provenance header,
// and gofmt-formats the result.
func renderComponentSource(parseSource, parsePkgName, parseName string) (string, error) {
	parseLines := strings.Split(parseSource, "\n")
	parsePkgIndex := -1
	for parseI, parseLine := range parseLines {
		if strings.HasPrefix(strings.TrimSpace(parseLine), "package ") {
			parsePkgIndex = parseI
			break
		}
	}
	if parsePkgIndex < 0 {
		return "", fmt.Errorf("template %q has no package clause", parseName)
	}

	parseBody := append([]string(nil), parseLines[parsePkgIndex:]...)
	parseBody[0] = "package " + parsePkgName

	var parseBuilder strings.Builder
	fmt.Fprintf(&parseBuilder, "// Code added by `gwc add %s`. You own this file — edit freely.\n\n", parseName)
	parseBuilder.WriteString(strings.Join(parseBody, "\n"))

	parseFormatted, parseErr := format.Source([]byte(parseBuilder.String()))
	if parseErr != nil {
		return "", fmt.Errorf("format component %q: %w", parseName, parseErr)
	}
	return string(parseFormatted), nil
}

// printComponentCatalog lists the installable components.
func printComponentCatalog() {
	parseEntries := append([]componentTemplate(nil), componentRegistry...)
	sort.Slice(parseEntries, func(parseA, parseB int) bool { return parseEntries[parseA].name < parseEntries[parseB].name })
	fmt.Println("GWC component catalog (copy into your repo with `gwc add <name>`):")
	for _, parseEntry := range parseEntries {
		fmt.Printf("  %-12s %s\n", parseEntry.name, parseEntry.summary)
	}
}
