package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runCheckCommand routes the agentic check command.
var runCheckCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runCheck(parseArgs)
}

// agenticCheckRunCommand executes external check commands and is overridden by tests.
var agenticCheckRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
	return launcherRunCommand(parseCommand, parseArgs, parseCwd, parseEnv)
}

type checkConfig struct {
	rootPath        string
	pattern         string
	skipTests       bool
	skipConventions bool
	json            bool
}

type checkCommandSummary struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	OK      bool   `json:"ok"`
	Output  string `json:"output,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

type checkSummary struct {
	OK          bool                  `json:"ok"`
	Root        string                `json:"root"`
	Diagnostics []agenticDiagnostic   `json:"diagnostics,omitempty"`
	Commands    []checkCommandSummary `json:"commands,omitempty"`
}

// runCheck parses check flags, runs checks, and emits human or JSON diagnostics.
func (parseL launcher) runCheck(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("check", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to check; defaults to current working directory")
	parsePattern := parseFlags.String("pattern", "./...", "go test package pattern")
	parseSkipTests := parseFlags.Bool("skip-tests", false, "Skip go test execution")
	parseSkipConventions := parseFlags.Bool("skip-conventions", false, "Skip source convention checks")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr := resolveCheckConfig(checkConfig{
		rootPath:        *parseRoot,
		pattern:         *parsePattern,
		skipTests:       *parseSkipTests,
		skipConventions: *parseSkipConventions,
		json:            *parseJSON,
	})
	if parseErr != nil {
		if *parseJSON {
			parseDiagnostic := buildAgenticCommandError("GWC-CHECK-CONFIG", parseErr)
			if parseWriteErr := writeAgenticEnvelope("check", false, nil, []agenticDiagnostic{parseDiagnostic}, parseErr); parseWriteErr != nil {
				return parseWriteErr
			}
		}
		return parseErr
	}
	parseSummary := buildCheckSummary(parseConfig)
	parseErr = nil
	if !parseSummary.OK {
		parseErr = fmt.Errorf("check found %d diagnostic(s)", len(parseSummary.Diagnostics))
	}
	if parseConfig.json {
		if parseWriteErr := writeAgenticEnvelope("check", parseSummary.OK, parseSummary, parseSummary.Diagnostics, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
	} else {
		parseLines := []string{
			fmt.Sprintf("root: %s", parseSummary.Root),
			fmt.Sprintf("diagnostics: %d", len(parseSummary.Diagnostics)),
		}
		for _, parseDiagnostic := range parseSummary.Diagnostics {
			parseLocation := parseDiagnostic.File
			if parseDiagnostic.Line > 0 {
				parseLocation = fmt.Sprintf("%s:%d", parseLocation, parseDiagnostic.Line)
			}
			parseLines = append(parseLines, fmt.Sprintf("%s %s %s", parseDiagnostic.Code, parseLocation, parseDiagnostic.Message))
		}
		printAgenticHumanSummary("GWC check", parseSummary.OK, parseLines)
	}
	return parseErr
}

// resolveCheckConfig validates check input flags.
func resolveCheckConfig(parseConfig checkConfig) (checkConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return checkConfig{}, fmt.Errorf("resolve check root from cwd: %w", parseErr)
		}
		parseRootPath = parseCWD
	}
	parseAbsRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return checkConfig{}, fmt.Errorf("resolve check root: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsRoot)
	if parseErr != nil {
		return checkConfig{}, fmt.Errorf("stat check root: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return checkConfig{}, fmt.Errorf("check root is not a directory: %s", parseAbsRoot)
	}
	parsePattern := strings.TrimSpace(parseConfig.pattern)
	if parsePattern == "" {
		parsePattern = "./..."
	}
	return checkConfig{
		rootPath:        parseAbsRoot,
		pattern:         parsePattern,
		skipTests:       parseConfig.skipTests,
		skipConventions: parseConfig.skipConventions,
		json:            parseConfig.json,
	}, nil
}

// buildCheckSummary runs enabled checks and returns structured diagnostics.
func buildCheckSummary(parseConfig checkConfig) checkSummary {
	parseSummary := checkSummary{
		OK:   true,
		Root: parseConfig.rootPath,
	}
	if parseConfig.skipTests {
		parseSummary.Commands = append(parseSummary.Commands, checkCommandSummary{Name: "go-test", OK: true, Skipped: true})
	} else {
		parseOutput, parseErr := agenticCheckRunCommand("go", []string{"test", parseConfig.pattern}, parseConfig.rootPath, buildNativeGoEnv())
		parseCommand := "go test " + parseConfig.pattern
		parseCommandSummary := checkCommandSummary{Name: "go-test", Command: parseCommand, OK: parseErr == nil, Output: strings.TrimSpace(parseOutput)}
		parseSummary.Commands = append(parseSummary.Commands, parseCommandSummary)
		if parseErr != nil {
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, agenticDiagnostic{
				Code:       "GWC-CHECK-GO-TEST",
				Severity:   "error",
				Message:    parseErr.Error(),
				Suggestion: "Run the reported failing package test locally and fix the compile or test failure.",
				Attributes: map[string]string{"command": parseCommand},
			})
		}
	}
	if !parseConfig.skipConventions {
		parseDiagnostics := collectCheckConventionDiagnostics(parseConfig.rootPath)
		if len(parseDiagnostics) > 0 {
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, parseDiagnostics...)
		}
	}
	sort.SliceStable(parseSummary.Diagnostics, func(parseI int, parseJ int) bool {
		if parseSummary.Diagnostics[parseI].File == parseSummary.Diagnostics[parseJ].File {
			return parseSummary.Diagnostics[parseI].Line < parseSummary.Diagnostics[parseJ].Line
		}
		return parseSummary.Diagnostics[parseI].File < parseSummary.Diagnostics[parseJ].File
	})
	return parseSummary
}

// collectCheckConventionDiagnostics scans Go files for agent-actionable convention violations.
func collectCheckConventionDiagnostics(parseRootPath string) []agenticDiagnostic {
	parseDiagnostics := []agenticDiagnostic{}
	_ = filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			parseRel := relativeSlashPath(parseRootPath, parsePath)
			parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
				Code:     "GWC-CHECK-WALK",
				Severity: "error",
				Message:  parseWalkErr.Error(),
				File:     parseRel,
			})
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseRootPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parseEntry.Name(), ".go") {
			return nil
		}
		parseDiagnostics = append(parseDiagnostics, collectCheckFileConventionDiagnostics(parseRootPath, parsePath)...)
		return nil
	})
	return parseDiagnostics
}

// collectCheckFileConventionDiagnostics scans one Go source file.
func collectCheckFileConventionDiagnostics(parseRootPath string, parsePath string) []agenticDiagnostic {
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parsePath, nil, parser.ParseComments)
	parseRel := relativeSlashPath(parseRootPath, parsePath)
	if parseErr != nil {
		return []agenticDiagnostic{{
			Code:       "GWC-CHECK-PARSE",
			Severity:   "error",
			Message:    parseErr.Error(),
			File:       parseRel,
			Suggestion: "Fix Go syntax before running agentic checks.",
		}}
	}
	parseDiagnostics := []agenticDiagnostic{}
	for _, parseDecl := range parseFile.Decls {
		switch parseTypedDecl := parseDecl.(type) {
		case *ast.FuncDecl:
			if parseTypedDecl.Name != nil && parseTypedDecl.Name.IsExported() && !hasDocCommentForName(parseTypedDecl.Doc, parseTypedDecl.Name.Name) {
				parsePosition := parseFileSet.Position(parseTypedDecl.Name.Pos())
				parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
					Code:       "GWC-CONVENTION-GODOC",
					Severity:   "warning",
					Message:    fmt.Sprintf("exported function %s should have GoDoc whose first word is %s", parseTypedDecl.Name.Name, parseTypedDecl.Name.Name),
					File:       parseRel,
					Line:       parsePosition.Line,
					Column:     parsePosition.Column,
					Suggestion: fmt.Sprintf("Add `// %s ...` immediately before the declaration.", parseTypedDecl.Name.Name),
				})
			}
		case *ast.GenDecl:
			parseDiagnostics = append(parseDiagnostics, collectCheckGenDeclDiagnostics(parseFileSet, parseRel, parseTypedDecl)...)
		}
	}
	return parseDiagnostics
}

// collectCheckGenDeclDiagnostics scans exported type, const, and var declarations.
func collectCheckGenDeclDiagnostics(parseFileSet *token.FileSet, parseRel string, parseDecl *ast.GenDecl) []agenticDiagnostic {
	parseDiagnostics := []agenticDiagnostic{}
	for _, parseSpec := range parseDecl.Specs {
		parseNames := []*ast.Ident{}
		switch parseTypedSpec := parseSpec.(type) {
		case *ast.TypeSpec:
			parseNames = append(parseNames, parseTypedSpec.Name)
		case *ast.ValueSpec:
			parseNames = append(parseNames, parseTypedSpec.Names...)
		}
		for _, parseName := range parseNames {
			if parseName == nil || !parseName.IsExported() {
				continue
			}
			if hasDocCommentForName(parseDecl.Doc, parseName.Name) {
				continue
			}
			parsePosition := parseFileSet.Position(parseName.Pos())
			parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
				Code:       "GWC-CONVENTION-GODOC",
				Severity:   "warning",
				Message:    fmt.Sprintf("exported declaration %s should have GoDoc whose first word is %s", parseName.Name, parseName.Name),
				File:       parseRel,
				Line:       parsePosition.Line,
				Column:     parsePosition.Column,
				Suggestion: fmt.Sprintf("Add `// %s ...` immediately before the declaration.", parseName.Name),
			})
		}
	}
	return parseDiagnostics
}

// hasDocCommentForName reports whether a doc comment begins with the symbol name.
func hasDocCommentForName(parseDoc *ast.CommentGroup, parseName string) bool {
	if parseDoc == nil || strings.TrimSpace(parseName) == "" {
		return false
	}
	parseText := strings.TrimSpace(parseDoc.Text())
	return strings.HasPrefix(parseText, parseName+" ") || strings.HasPrefix(parseText, parseName+"\n")
}

// relativeSlashPath returns a stable relative slash path.
func relativeSlashPath(parseRootPath string, parsePath string) string {
	parseRel, parseErr := filepath.Rel(parseRootPath, parsePath)
	if parseErr != nil {
		return filepath.ToSlash(parsePath)
	}
	return filepath.ToSlash(parseRel)
}
