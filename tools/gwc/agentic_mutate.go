package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runMutateCommand routes the agentic mutate command.
var runMutateCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runMutate(parseArgs)
}

type mutateConfig struct {
	rootPath string
	op       string
	from     string
	to       string
	dryRun   bool
	json     bool
}

type mutateFileChange struct {
	Path        string `json:"path"`
	Operation   string `json:"operation"`
	Occurrences int    `json:"occurrences"`
	Diff        string `json:"diff,omitempty"`
}

type mutateSummary struct {
	OK           bool               `json:"ok"`
	Root         string             `json:"root"`
	Operation    string             `json:"operation"`
	DryRun       bool               `json:"dryRun"`
	FilesChanged int                `json:"filesChanged"`
	Occurrences  int                `json:"occurrences"`
	Changes      []mutateFileChange `json:"changes,omitempty"`
}

// runMutate parses mutate arguments and applies a safe source operation.
func (parseL launcher) runMutate(parseArgs []string) error {
	parseOp := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		parseOp = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	parseFlags := flag.NewFlagSet("mutate", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseOpFlag := parseFlags.String("op", parseOp, "Mutation operation: rename-ident")
	parseRoot := parseFlags.String("root", "", "Project root to mutate; defaults to current working directory")
	parseFrom := parseFlags.String("from", "", "Source identifier for rename-ident")
	parseTo := parseFlags.String("to", "", "Replacement identifier for rename-ident")
	parseDryRun := parseFlags.Bool("dry-run", false, "Report edits without writing files")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := resolveMutateConfig(mutateConfig{
		rootPath: *parseRoot,
		op:       *parseOpFlag,
		from:     *parseFrom,
		to:       *parseTo,
		dryRun:   *parseDryRun,
		json:     *parseJSON,
	})
	if parseErr != nil {
		if *parseJSON {
			parseDiagnostic := buildAgenticCommandError("GWC-MUTATE-CONFIG", parseErr)
			if parseWriteErr := writeAgenticEnvelope("mutate", false, nil, []agenticDiagnostic{parseDiagnostic}, parseErr); parseWriteErr != nil {
				return parseWriteErr
			}
		}
		return parseErr
	}

	parseSummary, parseErr := applyMutateOperation(parseConfig)
	if parseConfig.json {
		parseDiagnostics := []agenticDiagnostic(nil)
		if parseErr != nil {
			parseDiagnostics = append(parseDiagnostics, buildAgenticCommandError("GWC-MUTATE-APPLY", parseErr))
		}
		if parseWriteErr := writeAgenticEnvelope("mutate", parseErr == nil, parseSummary, parseDiagnostics, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
	}
	if !parseConfig.json {
		parseLines := []string{
			fmt.Sprintf("operation: %s", parseSummary.Operation),
			fmt.Sprintf("root: %s", parseSummary.Root),
			fmt.Sprintf("files changed: %d", parseSummary.FilesChanged),
			fmt.Sprintf("occurrences: %d", parseSummary.Occurrences),
		}
		if parseSummary.DryRun {
			parseLines = append(parseLines, "dry-run: true")
		}
		printAgenticHumanSummary("GWC mutate", parseErr == nil, parseLines)
	}
	return parseErr
}

// resolveMutateConfig resolves paths and validates the requested mutation.
func resolveMutateConfig(parseConfig mutateConfig) (mutateConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return mutateConfig{}, fmt.Errorf("resolve mutate root from cwd: %w", parseErr)
		}
		parseRootPath = parseCWD
	}
	parseAbsRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return mutateConfig{}, fmt.Errorf("resolve mutate root: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsRoot)
	if parseErr != nil {
		return mutateConfig{}, fmt.Errorf("stat mutate root: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return mutateConfig{}, fmt.Errorf("mutate root is not a directory: %s", parseAbsRoot)
	}

	parseOp := strings.ToLower(strings.TrimSpace(parseConfig.op))
	switch parseOp {
	case "rename-ident", "rename-component", "rename-prop":
		parseOp = "rename-ident"
	default:
		return mutateConfig{}, fmt.Errorf("unknown mutate operation %q", parseConfig.op)
	}
	parseFrom := strings.TrimSpace(parseConfig.from)
	parseTo := strings.TrimSpace(parseConfig.to)
	if !isValidGoIdentifier(parseFrom) {
		return mutateConfig{}, fmt.Errorf("from must be a valid Go identifier, got %q", parseConfig.from)
	}
	if !isValidGoIdentifier(parseTo) {
		return mutateConfig{}, fmt.Errorf("to must be a valid Go identifier, got %q", parseConfig.to)
	}
	if parseFrom == parseTo {
		return mutateConfig{}, fmt.Errorf("from and to identifiers must differ")
	}
	return mutateConfig{
		rootPath: parseAbsRoot,
		op:       parseOp,
		from:     parseFrom,
		to:       parseTo,
		dryRun:   parseConfig.dryRun,
		json:     parseConfig.json,
	}, nil
}

// applyMutateOperation applies the configured mutation and returns a summary.
func applyMutateOperation(parseConfig mutateConfig) (mutateSummary, error) {
	parseSummary := mutateSummary{
		OK:        true,
		Root:      parseConfig.rootPath,
		Operation: parseConfig.op,
		DryRun:    parseConfig.dryRun,
	}
	parseChanges, parseErr := collectMutateRenameIdentChanges(parseConfig)
	if parseErr != nil {
		parseSummary.OK = false
		return parseSummary, parseErr
	}
	if len(parseChanges) == 0 {
		parseSummary.OK = false
		return parseSummary, fmt.Errorf("no identifiers named %q found under %s", parseConfig.from, parseConfig.rootPath)
	}
	for _, parseChange := range parseChanges {
		parseSummary.FilesChanged++
		parseSummary.Occurrences += parseChange.Occurrences
		parseSummary.Changes = append(parseSummary.Changes, parseChange)
	}
	if parseConfig.dryRun {
		return parseSummary, nil
	}
	for _, parseChange := range parseChanges {
		parseAbsPath := filepath.Join(parseConfig.rootPath, filepath.FromSlash(parseChange.Path))
		parseContent, parseErr := os.ReadFile(parseAbsPath)
		if parseErr != nil {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("read changed file before write: %w", parseErr)
		}
		parseNext, _, parseErr := rewriteMutateRenameIdent(parseAbsPath, parseContent, parseConfig.from, parseConfig.to)
		if parseErr != nil {
			parseSummary.OK = false
			return parseSummary, parseErr
		}
		parseInfo, parseErr := os.Stat(parseAbsPath)
		if parseErr != nil {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("stat changed file before write: %w", parseErr)
		}
		if parseErr := os.WriteFile(parseAbsPath, parseNext, parseInfo.Mode().Perm()); parseErr != nil {
			parseSummary.OK = false
			return parseSummary, fmt.Errorf("write mutated file %s: %w", parseAbsPath, parseErr)
		}
	}
	return parseSummary, nil
}

// collectMutateRenameIdentChanges scans Go files and prepares deterministic rename diffs.
func collectMutateRenameIdentChanges(parseConfig mutateConfig) ([]mutateFileChange, error) {
	parseChanges := []mutateFileChange{}
	parseErr := filepath.WalkDir(parseConfig.rootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseConfig.rootPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parseEntry.Name(), ".go") {
			return nil
		}
		parseBefore, parseErr := os.ReadFile(parsePath)
		if parseErr != nil {
			return fmt.Errorf("read mutate source %s: %w", parsePath, parseErr)
		}
		parseAfter, parseCount, parseErr := rewriteMutateRenameIdent(parsePath, parseBefore, parseConfig.from, parseConfig.to)
		if parseErr != nil {
			return parseErr
		}
		if parseCount == 0 || bytes.Equal(parseBefore, parseAfter) {
			return nil
		}
		parseRel, parseErr := filepath.Rel(parseConfig.rootPath, parsePath)
		if parseErr != nil {
			return parseErr
		}
		parseRel = filepath.ToSlash(parseRel)
		parseChanges = append(parseChanges, mutateFileChange{
			Path:        parseRel,
			Operation:   parseConfig.op,
			Occurrences: parseCount,
			Diff:        buildUnifiedDiff(parseRel, string(parseBefore), string(parseAfter)),
		})
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}
	sort.Slice(parseChanges, func(parseI int, parseJ int) bool {
		return parseChanges[parseI].Path < parseChanges[parseJ].Path
	})
	return parseChanges, nil
}

// rewriteMutateRenameIdent rewrites Go identifiers while leaving comments and strings untouched.
func rewriteMutateRenameIdent(parsePath string, parseSource []byte, parseFrom string, parseTo string) ([]byte, int, error) {
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parsePath, parseSource, parser.ParseComments)
	if parseErr != nil {
		return nil, 0, fmt.Errorf("parse Go source %s: %w", parsePath, parseErr)
	}
	parseCount := 0
	ast.Inspect(parseFile, func(parseNode ast.Node) bool {
		parseIdent, parseOK := parseNode.(*ast.Ident)
		if !parseOK || parseIdent == nil {
			return true
		}
		if parseIdent.Name == parseFrom {
			parseIdent.Name = parseTo
			parseCount++
		}
		return true
	})
	if parseCount == 0 {
		return parseSource, 0, nil
	}
	var parseBuffer bytes.Buffer
	if parseErr := format.Node(&parseBuffer, parseFileSet, parseFile); parseErr != nil {
		return nil, parseCount, fmt.Errorf("format mutated Go source %s: %w", parsePath, parseErr)
	}
	return parseBuffer.Bytes(), parseCount, nil
}

// isValidGoIdentifier reports whether a value can safely be used as a Go identifier.
func isValidGoIdentifier(parseValue string) bool {
	if strings.TrimSpace(parseValue) == "" || !token.IsIdentifier(parseValue) || token.Lookup(parseValue).IsKeyword() {
		return false
	}
	return true
}

// shouldSkipMutateDir reports whether a directory should be ignored by source mutations.
func shouldSkipMutateDir(parseName string) bool {
	switch parseName {
	case ".git", ".hg", ".svn", "bin", "build", "dist", "node_modules", "vendor", "testdata":
		return true
	default:
		return false
	}
}

// buildUnifiedDiff renders a compact single-hunk unified diff.
func buildUnifiedDiff(parsePath string, parseBefore string, parseAfter string) string {
	parseBeforeLines := splitDiffLines(parseBefore)
	parseAfterLines := splitDiffLines(parseAfter)
	parsePrefix := 0
	for parsePrefix < len(parseBeforeLines) && parsePrefix < len(parseAfterLines) && parseBeforeLines[parsePrefix] == parseAfterLines[parsePrefix] {
		parsePrefix++
	}
	parseSuffix := 0
	for parseSuffix < len(parseBeforeLines)-parsePrefix && parseSuffix < len(parseAfterLines)-parsePrefix {
		parseBeforeIndex := len(parseBeforeLines) - 1 - parseSuffix
		parseAfterIndex := len(parseAfterLines) - 1 - parseSuffix
		if parseBeforeLines[parseBeforeIndex] != parseAfterLines[parseAfterIndex] {
			break
		}
		parseSuffix++
	}
	parseBeforeStart := max(0, parsePrefix-3)
	parseAfterStart := max(0, parsePrefix-3)
	parseBeforeEnd := len(parseBeforeLines) - parseSuffix
	parseAfterEnd := len(parseAfterLines) - parseSuffix
	if parseBeforeEnd < parseBeforeStart {
		parseBeforeEnd = parseBeforeStart
	}
	if parseAfterEnd < parseAfterStart {
		parseAfterEnd = parseAfterStart
	}
	parseBeforeLen := parseBeforeEnd - parseBeforeStart
	parseAfterLen := parseAfterEnd - parseAfterStart
	var parseBuilder strings.Builder
	parseBuilder.WriteString("--- a/" + parsePath + "\n")
	parseBuilder.WriteString("+++ b/" + parsePath + "\n")
	parseBuilder.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", parseBeforeStart+1, parseBeforeLen, parseAfterStart+1, parseAfterLen))
	for parseIndex := parseBeforeStart; parseIndex < parsePrefix && parseIndex < len(parseBeforeLines); parseIndex++ {
		parseBuilder.WriteString(" " + parseBeforeLines[parseIndex] + "\n")
	}
	for parseIndex := parsePrefix; parseIndex < parseBeforeEnd; parseIndex++ {
		parseBuilder.WriteString("-" + parseBeforeLines[parseIndex] + "\n")
	}
	for parseIndex := parsePrefix; parseIndex < parseAfterEnd; parseIndex++ {
		parseBuilder.WriteString("+" + parseAfterLines[parseIndex] + "\n")
	}
	for parseIndex := len(parseBeforeLines) - parseSuffix; parseIndex < len(parseBeforeLines); parseIndex++ {
		if parseIndex >= 0 && parseIndex < len(parseBeforeLines) {
			parseBuilder.WriteString(" " + parseBeforeLines[parseIndex] + "\n")
		}
	}
	return parseBuilder.String()
}

// splitDiffLines returns lines without a trailing empty line from final newline.
func splitDiffLines(parseText string) []string {
	parseLines := strings.Split(strings.ReplaceAll(parseText, "\r\n", "\n"), "\n")
	if len(parseLines) > 0 && parseLines[len(parseLines)-1] == "" {
		parseLines = parseLines[:len(parseLines)-1]
	}
	return parseLines
}
