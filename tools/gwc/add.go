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

	// Composable sub-generators: `gwc add component <Name>` and `gwc add route <Name>` scaffold
	// idiomatic code into an existing app (the A3 10-rung "add-a-component / add-a-route"
	// generators), distinct from copying a catalog component by name.
	if parseName == "component" || parseName == "route" {
		return parseL.runAddGenerator(parseName, parseArgs)
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

// runAddGenerator handles `gwc add component <Name>` and `gwc add route <Name>`, scaffolding
// idiomatic, compiling Go into the project. The component generator emits a typed-props component
// in the README starter idiom; the route generator emits a `router.MustDefineRoute` contract var
// (so `gwc routes gen` produces a typed Link constructor for it) plus its route component.
func (parseL launcher) runAddGenerator(parseKind string, parseArgs []string) error {
	parseEntityName := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseEntityName = parseArgs[0]
		parseArgs = parseArgs[1:]
	}

	parseFlags := flag.NewFlagSet("add "+parseKind, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseDir := parseFlags.String("dir", parseKind+"s", "Directory to write the file into")
	parsePkg := parseFlags.String("pkg", "", "Package name; defaults to the destination directory's base name")
	parsePath := parseFlags.String("path", "", "Route URL pattern, e.g. /users/:id (route generator only)")
	parseForce := parseFlags.Bool("force", false, "Overwrite an existing file")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	if parseEntityName == "" {
		return fmt.Errorf("usage: gwc add %s <Name> [-dir DIR] [-pkg PKG]%s", parseKind, map[string]string{"route": " [-path /pattern]"}[parseKind])
	}
	parseTypeName := exportedGoIdent(parseEntityName)
	if parseTypeName == "" {
		return fmt.Errorf("%q is not a valid Go identifier for a %s name", parseEntityName, parseKind)
	}

	parsePkgName := strings.TrimSpace(*parsePkg)
	if parsePkgName == "" {
		parsePkgName = validPackageName(sanitizeGoIdent(filepath.Base(*parseDir)), parseKind+"s")
	}

	var parseSource string
	var parseErr error
	switch parseKind {
	case "component":
		parseSource, parseErr = scaffoldComponentSource(parsePkgName, parseTypeName)
	case "route":
		parsePattern := strings.TrimSpace(*parsePath)
		if parsePattern == "" {
			parsePattern = "/" + strings.ToLower(parseTypeName)
		}
		parseSource, parseErr = scaffoldRouteSource(parsePkgName, parseTypeName, parsePattern)
	}
	if parseErr != nil {
		return parseErr
	}

	if parseErr := os.MkdirAll(*parseDir, 0755); parseErr != nil {
		return fmt.Errorf("create %s: %w", *parseDir, parseErr)
	}
	parseOutPath := filepath.Join(*parseDir, strings.ToLower(parseTypeName)+".go")
	if _, parseStatErr := os.Stat(parseOutPath); parseStatErr == nil && !*parseForce {
		return fmt.Errorf("%s already exists; pass -force to overwrite", parseOutPath)
	}
	if parseErr := os.WriteFile(parseOutPath, []byte(parseSource), 0644); parseErr != nil {
		return fmt.Errorf("write %s: %w", parseOutPath, parseErr)
	}
	fmt.Printf("GWC add %s: wrote %s (package %s). You own this file — edit freely.\n", parseKind, parseOutPath, parsePkgName)
	return nil
}

// scaffoldComponentSource returns a gofmt-clean, compiling component in the README starter idiom:
// typed props, a UseState hook, a UseEvent handler, and dot-imported shorthand tags.
func scaffoldComponentSource(parsePkgName, parseTypeName string) (string, error) {
	parseSrc := fmt.Sprintf(`// Code generated by `+"`gwc add component %s`"+`. You own this file — edit freely.

package %s

import (
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// %sProps are the inputs to the %s component.
type %sProps struct {
	Title string
}

// %s renders the %s component.
func %s(props %sProps) ui.Node {
	count := ui.UseState(0)
	increment := ui.UseEvent(func() {
		count.Update(func(previous int) int { return previous + 1 })
	})
	return Div(
		H2(props.Title),
		P(Textf("Count: %%d", count.Get())),
		Button(OnClick(increment), "Increment"),
	)
}
`, parseTypeName, parsePkgName, parseTypeName, parseTypeName, parseTypeName, parseTypeName, parseTypeName, parseTypeName, parseTypeName)
	parseFormatted, parseErr := format.Source([]byte(parseSrc))
	if parseErr != nil {
		return "", fmt.Errorf("format generated component %q: %w", parseTypeName, parseErr)
	}
	return string(parseFormatted), nil
}

// scaffoldRouteSource returns a gofmt-clean, compiling route file: a typed route contract var
// (picked up by `gwc routes gen` to emit a Link constructor) plus its route component.
func scaffoldRouteSource(parsePkgName, parseTypeName, parsePattern string) (string, error) {
	parseSrc := fmt.Sprintf(`// Code generated by `+"`gwc add route %s`"+`. You own this file — edit freely.

package %s

import (
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// %sRoute is the typed route contract for %q. Run `+"`gwc routes gen`"+` to generate a
// compile-checked Link%s(...) constructor from it.
var %sRoute = router.MustDefineRoute(%q)

// %s renders the %q route.
func %s() ui.Node {
	return Div(
		H1(%q),
		P("TODO: render the %s route."),
	)
}
`, parseTypeName, parsePkgName, parseTypeName, parsePattern, parseTypeName, parseTypeName, parsePattern, parseTypeName, parsePattern, parseTypeName, parseTypeName, parsePattern)
	parseFormatted, parseErr := format.Source([]byte(parseSrc))
	if parseErr != nil {
		return "", fmt.Errorf("format generated route %q: %w", parseTypeName, parseErr)
	}
	return string(parseFormatted), nil
}

// validPackageName returns a usable lower-case package identifier from a sanitized base, falling
// back when the base is empty or starts with a digit (which is not a legal Go identifier — e.g. a
// directory named "001" from a temp path or "2fa").
func validPackageName(parseBase, parseFallback string) string {
	if parseBase == "" {
		return parseFallback
	}
	if parseBase[0] >= '0' && parseBase[0] <= '9' {
		return parseFallback
	}
	return strings.ToLower(parseBase)
}

// exportedGoIdent converts a name into an exported Go identifier (leading letter upper-cased,
// non-identifier runes dropped), or "" if nothing valid remains.
func exportedGoIdent(parseName string) string {
	parseSanitized := sanitizeGoIdent(parseName)
	if parseSanitized == "" {
		return ""
	}
	return strings.ToUpper(parseSanitized[:1]) + parseSanitized[1:]
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
