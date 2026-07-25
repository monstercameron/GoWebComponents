package main

import (
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// runAgenticScaffoldCommand routes the non-interactive agentic scaffold command.
var runAgenticScaffoldCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runAgenticScaffold(parseArgs)
}

type agenticScaffoldConfig struct {
	rootPath  string
	kind      string
	name      string
	dir       string
	pkg       string
	module    string
	valueType string
	noInput   bool
	dryRun    bool
	json      bool
}

type agenticScaffoldFile struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	Bytes     int    `json:"bytes,omitempty"`
}

type agenticScaffoldSummary struct {
	OK       bool                   `json:"ok"`
	Root     string                 `json:"root"`
	Kind     string                 `json:"kind"`
	Name     string                 `json:"name"`
	DryRun   bool                   `json:"dryRun"`
	NoInput  bool                   `json:"noInput"`
	Files    []agenticScaffoldFile  `json:"files,omitempty"`
	Next     []string               `json:"next,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type agenticScaffoldPlan struct {
	files []agenticScaffoldPlannedFile
	next  []string
	meta  map[string]interface{}
}

type agenticScaffoldPlannedFile struct {
	path    string
	content []byte
}

// runAgenticScaffold parses scaffold arguments and writes planned files.
func (parseL launcher) runAgenticScaffold(parseArgs []string) error {
	parseKind := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseKind = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	parseFlags := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseKindFlag := parseFlags.String("kind", parseKind, "Scaffold kind: component, hook, example, app")
	parseRoot := parseFlags.String("root", "", "Project root; defaults to the current working directory")
	parseDir := parseFlags.String("dir", "", "Output directory relative to -root")
	parseName := parseFlags.String("name", "", "Generated symbol or project name")
	parsePackage := parseFlags.String("package", "", "Go package name for component and hook scaffolds")
	parseModule := parseFlags.String("module", "", "Module path for app scaffolds")
	parseValueType := parseFlags.String("type", "", "State value type for hook scaffolds (e.g. string, int, bool, MyStruct); defaults to string")
	parseNoInput := parseFlags.Bool("no-input", false, "Confirm that the scaffold must run without prompts")
	parseDryRun := parseFlags.Bool("dry-run", false, "Report planned files without writing")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := resolveAgenticScaffoldConfig(agenticScaffoldConfig{
		rootPath:  *parseRoot,
		kind:      *parseKindFlag,
		name:      *parseName,
		dir:       *parseDir,
		pkg:       *parsePackage,
		module:    *parseModule,
		valueType: *parseValueType,
		noInput:   *parseNoInput,
		dryRun:    *parseDryRun,
		json:      *parseJSON,
	})
	if parseErr != nil {
		if *parseJSON {
			parseDiagnostic := buildAgenticCommandError("GWC-SCAFFOLD-CONFIG", parseErr)
			if parseWriteErr := writeAgenticEnvelope("scaffold", false, nil, []agenticDiagnostic{parseDiagnostic}, parseErr); parseWriteErr != nil {
				return parseWriteErr
			}
		}
		return parseErr
	}
	parseSummary, parseErr := applyAgenticScaffold(parseL, parseConfig)
	if parseConfig.json {
		parseDiagnostics := []agenticDiagnostic(nil)
		if parseErr != nil {
			parseDiagnostics = append(parseDiagnostics, buildAgenticCommandError("GWC-SCAFFOLD-APPLY", parseErr))
		}
		if parseWriteErr := writeAgenticEnvelope("scaffold", parseErr == nil, parseSummary, parseDiagnostics, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
	}
	if !parseConfig.json {
		parseLines := []string{
			fmt.Sprintf("kind: %s", parseSummary.Kind),
			fmt.Sprintf("root: %s", parseSummary.Root),
			fmt.Sprintf("files: %d", len(parseSummary.Files)),
		}
		if parseSummary.DryRun {
			parseLines = append(parseLines, "dry-run: true")
		}
		printAgenticHumanSummary("GWC scaffold", parseErr == nil, parseLines)
	}
	return parseErr
}

// resolveAgenticScaffoldConfig validates scaffold flags and fills defaults.
func resolveAgenticScaffoldConfig(parseConfig agenticScaffoldConfig) (agenticScaffoldConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return agenticScaffoldConfig{}, fmt.Errorf("resolve scaffold root from cwd: %w", parseErr)
		}
		parseRootPath = parseCWD
	}
	parseAbsRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return agenticScaffoldConfig{}, fmt.Errorf("resolve scaffold root: %w", parseErr)
	}
	parseKind := normalizeAgenticScaffoldKind(parseConfig.kind)
	if parseKind == "" {
		return agenticScaffoldConfig{}, fmt.Errorf("scaffold kind is required: component, hook, example, or app")
	}
	parseName := strings.TrimSpace(parseConfig.name)
	if parseName == "" {
		return agenticScaffoldConfig{}, fmt.Errorf("scaffold name is required")
	}
	if parseKind == "app" {
		if !isSafeScaffoldName(parseName) {
			return agenticScaffoldConfig{}, fmt.Errorf("app name contains unsupported characters: %q", parseName)
		}
	} else if !isValidGoIdentifier(parseName) || !unicode.IsUpper([]rune(parseName)[0]) {
		return agenticScaffoldConfig{}, fmt.Errorf("scaffold name must be an exported Go identifier, got %q", parseName)
	}
	parsePackage := strings.TrimSpace(parseConfig.pkg)
	parseDir := strings.TrimSpace(parseConfig.dir)
	if parseDir == "" {
		parseDir = defaultAgenticScaffoldDir(parseKind, parseName)
	}
	if filepath.IsAbs(parseDir) {
		return agenticScaffoldConfig{}, fmt.Errorf("scaffold dir must be relative to root: %s", parseDir)
	}
	if parsePackage == "" {
		parsePackage = packageNameFromDir(parseDir)
	}
	if parseKind != "app" && !isValidGoIdentifier(parsePackage) {
		return agenticScaffoldConfig{}, fmt.Errorf("package must be a valid Go identifier, got %q", parsePackage)
	}
	parseModule := strings.TrimSpace(parseConfig.module)
	if parseModule == "" && parseKind == "app" {
		parseModule = "example.com/" + slugifyScaffoldName(parseName)
	}
	parseValueType := strings.TrimSpace(parseConfig.valueType)
	if parseValueType == "" {
		parseValueType = "string"
	}
	return agenticScaffoldConfig{
		rootPath:  parseAbsRoot,
		kind:      parseKind,
		name:      parseName,
		dir:       filepath.Clean(parseDir),
		pkg:       parsePackage,
		module:    parseModule,
		valueType: parseValueType,
		noInput:   parseConfig.noInput,
		dryRun:    parseConfig.dryRun,
		json:      parseConfig.json,
	}, nil
}

// applyAgenticScaffold writes or reports the requested scaffold files.
func applyAgenticScaffold(parseL launcher, parseConfig agenticScaffoldConfig) (agenticScaffoldSummary, error) {
	parseSummary := agenticScaffoldSummary{
		OK:      true,
		Root:    parseConfig.rootPath,
		Kind:    parseConfig.kind,
		Name:    parseConfig.name,
		DryRun:  parseConfig.dryRun,
		NoInput: parseConfig.noInput,
	}
	if parseConfig.kind == "app" {
		return applyAgenticAppScaffold(parseL, parseConfig, parseSummary)
	}
	parsePlan, parseErr := planAgenticScaffold(parseConfig)
	if parseErr != nil {
		parseSummary.OK = false
		return parseSummary, parseErr
	}
	parseSummary.Metadata = parsePlan.meta
	parseSummary.Next = parsePlan.next
	for _, parseFile := range parsePlan.files {
		parseRel := filepath.ToSlash(parseFile.path)
		parseAbs := filepath.Join(parseConfig.rootPath, filepath.FromSlash(parseRel))
		if _, parseErr := os.Stat(parseAbs); parseErr == nil {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("refusing to overwrite existing scaffold file: %s", parseAbs)
		} else if !errors.Is(parseErr, os.ErrNotExist) {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("stat scaffold file %s: %w", parseAbs, parseErr)
		}
		parseSummary.Files = append(parseSummary.Files, agenticScaffoldFile{Path: parseRel, Operation: "create", Bytes: len(parseFile.content)})
	}
	if parseConfig.dryRun {
		return parseSummary, nil
	}
	for _, parseFile := range parsePlan.files {
		parseAbs := filepath.Join(parseConfig.rootPath, filepath.FromSlash(parseFile.path))
		if parseErr := os.MkdirAll(filepath.Dir(parseAbs), 0o755); parseErr != nil {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("create scaffold directory: %w", parseErr)
		}
		if parseErr := os.WriteFile(parseAbs, parseFile.content, 0o644); parseErr != nil {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("write scaffold file %s: %w", parseAbs, parseErr)
		}
	}
	return parseSummary, nil
}

// applyAgenticAppScaffold delegates app generation to the existing starter generator.
func applyAgenticAppScaffold(parseL launcher, parseConfig agenticScaffoldConfig, parseSummary agenticScaffoldSummary) (agenticScaffoldSummary, error) {
	parseTargetDir := filepath.Join(parseConfig.rootPath, parseConfig.dir)
	parseSelection := startSelection{
		ProjectName:       parseConfig.name,
		ModulePath:        parseConfig.module,
		Author:            "GWC agent",
		Version:           "0.1.0",
		Description:       "Non-interactive starter scaffold generated by gwc scaffold.",
		TargetDir:         parseTargetDir,
		ProjectMode:       scaffoldProjectModeStandalone,
		SkipGoModTidy:     true,
		SkipRuntimeAssets: true,
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client",
			Summary:     "Minimal client starter generated without prompts.",
			Description: "Minimal GoWebComponents client starter.",
			Features:    nil,
		},
	}
	parseSummary.Metadata = map[string]interface{}{"module": parseConfig.module, "targetDir": parseTargetDir}
	parseSummary.Files = []agenticScaffoldFile{
		{Path: filepath.ToSlash(filepath.Join(parseConfig.dir, "go.mod")), Operation: "create"},
		{Path: filepath.ToSlash(filepath.Join(parseConfig.dir, "main.go")), Operation: "create"},
		{Path: filepath.ToSlash(filepath.Join(parseConfig.dir, "index.html")), Operation: "create"},
		{Path: filepath.ToSlash(filepath.Join(parseConfig.dir, "gwc-start.json")), Operation: "create"},
	}
	parseSummary.Next = []string{
		fmt.Sprintf("go run ./tools/gwc build -app %s -root %s", filepath.ToSlash(filepath.Join(parseTargetDir, "main.go")), filepath.ToSlash(parseTargetDir)),
	}
	if parseConfig.dryRun {
		return parseSummary, nil
	}
	parseResult, parseErr := startGenerateScaffold(parseL, parseSelection)
	if parseErr != nil {
		parseSummary.OK = false
		return parseSummary, parseErr
	}
	parseSummary.Metadata["appPath"] = parseResult.AppPath
	parseSummary.Metadata["htmlPath"] = parseResult.HTMLPath
	return parseSummary, nil
}

// planAgenticScaffold builds the file plan for component, hook, and example scaffolds.
func planAgenticScaffold(parseConfig agenticScaffoldConfig) (agenticScaffoldPlan, error) {
	switch parseConfig.kind {
	case "component":
		parseRel := filepath.ToSlash(filepath.Join(parseConfig.dir, snakeCaseScaffoldName(parseConfig.name)+".go"))
		parseContent, parseErr := format.Source([]byte(renderAgenticComponentScaffold(parseConfig)))
		if parseErr != nil {
			return agenticScaffoldPlan{}, fmt.Errorf("format component scaffold: %w", parseErr)
		}
		return agenticScaffoldPlan{
			files: []agenticScaffoldPlannedFile{{path: parseRel, content: parseContent}},
			next:  []string{fmt.Sprintf("Import and render %s.%s from a parent component.", parseConfig.pkg, parseConfig.name)},
			meta:  map[string]interface{}{"package": parseConfig.pkg},
		}, nil
	case "hook":
		parseRel := filepath.ToSlash(filepath.Join(parseConfig.dir, "use_"+snakeCaseScaffoldName(parseConfig.name)+".go"))
		parseContent, parseErr := format.Source([]byte(renderAgenticHookScaffold(parseConfig)))
		if parseErr != nil {
			return agenticScaffoldPlan{}, fmt.Errorf("format hook scaffold: %w", parseErr)
		}
		return agenticScaffoldPlan{
			files: []agenticScaffoldPlannedFile{{path: parseRel, content: parseContent}},
			next:  []string{fmt.Sprintf("Call %s.Use%sState from a component body.", parseConfig.pkg, parseConfig.name)},
			meta:  map[string]interface{}{"package": parseConfig.pkg},
		}, nil
	case "example":
		parseDir := filepath.ToSlash(filepath.Join(parseConfig.dir, slugifyScaffoldName(parseConfig.name)))
		parseRel := filepath.ToSlash(filepath.Join(parseDir, "main.go"))
		parseExampleConfig := parseConfig
		parseExampleConfig.pkg = "main"
		parseContent, parseErr := format.Source([]byte(renderAgenticExampleScaffold(parseExampleConfig)))
		if parseErr != nil {
			return agenticScaffoldPlan{}, fmt.Errorf("format example scaffold: %w", parseErr)
		}
		return agenticScaffoldPlan{
			files: []agenticScaffoldPlannedFile{{path: parseRel, content: parseContent}},
			next:  []string{fmt.Sprintf("GOOS=js GOARCH=wasm go build -o bin/%s.wasm ./%s", slugifyScaffoldName(parseConfig.name), parseDir)},
			meta:  map[string]interface{}{"package": "main"},
		}, nil
	default:
		return agenticScaffoldPlan{}, fmt.Errorf("unsupported scaffold kind %q", parseConfig.kind)
	}
}

// renderAgenticComponentScaffold renders a starter component source file.
func renderAgenticComponentScaffold(parseConfig agenticScaffoldConfig) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// %s renders a starter component scaffold.
func %s() ui.Node {
	return html.Div(html.Props{Class: "gwc-component"}, html.Text(%q))
}
`, parseConfig.pkg, parseConfig.name, parseConfig.name, parseConfig.name)
}

// renderAgenticHookScaffold renders a starter hook source file. The state value type
// defaults to string and can be overridden with the -type flag.
func renderAgenticHookScaffold(parseConfig agenticScaffoldConfig) string {
	parseValueType := parseConfig.valueType
	if parseValueType == "" {
		parseValueType = "string"
	}
	return fmt.Sprintf(`package %s

import "github.com/monstercameron/GoWebComponents/v5/ui"

// Use%sState returns scaffolded component-local state for %s.
func Use%sState(parseInitial %s) ui.State[%s] {
	return ui.UseState(parseInitial)
}
`, parseConfig.pkg, parseConfig.name, parseConfig.name, parseConfig.name, parseValueType, parseValueType)
}

// renderAgenticExampleScaffold renders a runnable wasm example source file.
func renderAgenticExampleScaffold(parseConfig agenticScaffoldConfig) string {
	return fmt.Sprintf(`//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

// App renders the %s example.
func App() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})
	return html.Div(html.Props{Class: "gwc-example"},
		html.H1(html.Props{}, html.Text(%q)),
		html.P(html.Props{}, html.Textf("count: %%d", parseCount.Get())),
		html.Button(html.Props{OnClick: parseIncrement}, html.Text("Increment")),
	)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	utils.WaitForever()
}
`, parseConfig.name, parseConfig.name)
}

// normalizeAgenticScaffoldKind normalizes supported scaffold kind aliases.
func normalizeAgenticScaffoldKind(parseKind string) string {
	switch strings.ToLower(strings.TrimSpace(parseKind)) {
	case "component", "cmp":
		return "component"
	case "hook":
		return "hook"
	case "example":
		return "example"
	case "app", "starter":
		return "app"
	default:
		return ""
	}
}

// defaultAgenticScaffoldDir returns the default relative output directory.
func defaultAgenticScaffoldDir(parseKind string, parseName string) string {
	switch parseKind {
	case "component":
		return "components"
	case "hook":
		return "hooks"
	case "example":
		return "examples"
	case "app":
		return slugifyScaffoldName(parseName)
	default:
		return "."
	}
}

// packageNameFromDir derives a conservative package name from a relative path.
func packageNameFromDir(parseDir string) string {
	parseBase := filepath.Base(filepath.Clean(parseDir))
	parseBase = strings.ReplaceAll(parseBase, "-", "_")
	parseBase = strings.ReplaceAll(parseBase, ".", "_")
	if parseBase == "" || parseBase == "." {
		return "main"
	}
	return parseBase
}

// snakeCaseScaffoldName converts an exported identifier to a file stem.
func snakeCaseScaffoldName(parseName string) string {
	var parseBuilder strings.Builder
	for parseIndex, parseRune := range parseName {
		if parseIndex > 0 && unicode.IsUpper(parseRune) {
			parseBuilder.WriteRune('_')
		}
		parseBuilder.WriteRune(unicode.ToLower(parseRune))
	}
	return parseBuilder.String()
}

// slugifyScaffoldName converts a project or symbol name to a path segment.
func slugifyScaffoldName(parseName string) string {
	parseLower := strings.ToLower(strings.TrimSpace(parseName))
	var parseBuilder strings.Builder
	parseLastDash := false
	for _, parseRune := range parseLower {
		if unicode.IsLetter(parseRune) || unicode.IsDigit(parseRune) {
			parseBuilder.WriteRune(parseRune)
			parseLastDash = false
			continue
		}
		if !parseLastDash {
			parseBuilder.WriteRune('-')
			parseLastDash = true
		}
	}
	return strings.Trim(parseBuilder.String(), "-")
}

// isSafeScaffoldName reports whether an app name is usable in generated paths.
func isSafeScaffoldName(parseName string) bool {
	parseSlug := slugifyScaffoldName(parseName)
	return parseSlug != "" && !strings.Contains(parseSlug, "..")
}
