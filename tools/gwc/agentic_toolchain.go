package main

import (
	"bytes"
	"encoding/json"
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
	"strconv"
	"strings"
	"time"
)

var runFmtCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runFmt(parseArgs)
}

var runCleanCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runClean(parseArgs)
}

var runWatchCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runWatch(parseArgs)
}

var runSizeCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runSize(parseArgs)
}

var runDocsCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runDocs(parseArgs)
}

var runDeadcodeCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runDeadcode(parseArgs)
}

var runDepsCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runDeps(parseArgs)
}

type fmtFileResult struct {
	Path        string `json:"path"`
	Changed     bool   `json:"changed"`
	WouldChange bool   `json:"wouldChange,omitempty"`
}

type fmtSummary struct {
	OK          bool                `json:"ok"`
	Root        string              `json:"root"`
	Check       bool                `json:"check"`
	Files       []fmtFileResult     `json:"files,omitempty"`
	Diagnostics []agenticDiagnostic `json:"diagnostics,omitempty"`
}

func (parseL launcher) runFmt(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root"}, []string{"check", "json"})
	parseFlags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to format")
	parseCheck := parseFlags.Bool("check", false, "Report formatting changes without writing")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	parseSummary := fmtSummary{OK: true, Root: parseRootPath, Check: *parseCheck}
	parseErr = filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-FMT-WALK", parseWalkErr))
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
		parseRel := relativeSlashPath(parseRootPath, parsePath)
		parseOriginal, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-FMT-READ", parseReadErr))
			return nil
		}
		parseFormatted, parseDiagnostics, parseFormatErr := formatAgenticGoSource(parseRel, parseOriginal, !*parseCheck)
		parseSummary.Diagnostics = append(parseSummary.Diagnostics, parseDiagnostics...)
		if parseFormatErr != nil {
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-FMT-PARSE", parseFormatErr))
			return nil
		}
		if !bytes.Equal(parseOriginal, parseFormatted) {
			parseSummary.Files = append(parseSummary.Files, fmtFileResult{Path: parseRel, Changed: !*parseCheck, WouldChange: *parseCheck})
			if *parseCheck {
				parseSummary.OK = false
				return nil
			}
			parseInfo, parseStatErr := os.Stat(parsePath)
			if parseStatErr != nil {
				return parseStatErr
			}
			if parseWriteErr := os.WriteFile(parsePath, parseFormatted, parseInfo.Mode().Perm()); parseWriteErr != nil {
				return parseWriteErr
			}
		}
		for _, parseDiagnostic := range parseDiagnostics {
			if parseDiagnostic.Severity == "error" || *parseCheck {
				parseSummary.OK = false
				break
			}
		}
		return nil
	})
	if parseErr != nil {
		parseSummary.OK = false
		parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-FMT-FAILED", parseErr))
	}
	if *parseJSON {
		_ = writeAgenticEnvelope("fmt", parseSummary.OK, parseSummary, parseSummary.Diagnostics, nil)
	}
	if !*parseJSON {
		printAgenticHumanSummary("GWC fmt", parseSummary.OK, []string{fmt.Sprintf("files: %d", len(parseSummary.Files)), fmt.Sprintf("diagnostics: %d", len(parseSummary.Diagnostics))})
	}
	if !parseSummary.OK {
		return errors.New("fmt reported changes or diagnostics")
	}
	return nil
}

func formatAgenticGoSource(parseRel string, parseOriginal []byte, isFixDocs bool) ([]byte, []agenticDiagnostic, error) {
	parseNormalized := bytes.ReplaceAll(parseOriginal, []byte("\r\n"), []byte("\n"))
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parseRel, parseNormalized, parser.ParseComments)
	if parseErr != nil {
		return nil, nil, parseErr
	}
	parseDiagnostics := collectMissingGoDocDiagnostics(parseFileSet, parseFile, parseRel)
	parseNext := parseNormalized
	if isFixDocs && len(parseDiagnostics) > 0 {
		parseNext = insertGoDocStubs(parseNext, parseDiagnostics)
	}
	parseFormatted, parseErr := format.Source(parseNext)
	if parseErr != nil {
		return nil, parseDiagnostics, parseErr
	}
	return parseFormatted, parseDiagnostics, nil
}

func collectMissingGoDocDiagnostics(parseFileSet *token.FileSet, parseFile *ast.File, parseRel string) []agenticDiagnostic {
	parseDiagnostics := []agenticDiagnostic{}
	for _, parseDecl := range parseFile.Decls {
		switch parseTyped := parseDecl.(type) {
		case *ast.FuncDecl:
			if parseTyped.Name != nil && parseTyped.Name.IsExported() && !hasDocCommentForName(parseTyped.Doc, parseTyped.Name.Name) {
				parsePosition := parseFileSet.Position(parseTyped.Pos())
				parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
					Code:       "GWC-FMT-GODOC",
					Severity:   "warning",
					Message:    fmt.Sprintf("exported function %s is missing GoDoc-first-word comment", parseTyped.Name.Name),
					File:       parseRel,
					Line:       parsePosition.Line,
					Column:     parsePosition.Column,
					Suggestion: fmt.Sprintf("Add `// %s ...` before the declaration.", parseTyped.Name.Name),
					Edits:      []agenticTextEdit{{File: parseRel, Description: "insert GoDoc stub", NewText: "// " + parseTyped.Name.Name + " documents " + parseTyped.Name.Name + ".\n"}},
				})
			}
		case *ast.GenDecl:
			for _, parseSpec := range parseTyped.Specs {
				parseType, parseOK := parseSpec.(*ast.TypeSpec)
				if !parseOK || !parseType.Name.IsExported() || hasDocCommentForName(parseTyped.Doc, parseType.Name.Name) {
					continue
				}
				parsePosition := parseFileSet.Position(parseTyped.Pos())
				parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
					Code:       "GWC-FMT-GODOC",
					Severity:   "warning",
					Message:    fmt.Sprintf("exported type %s is missing GoDoc-first-word comment", parseType.Name.Name),
					File:       parseRel,
					Line:       parsePosition.Line,
					Column:     parsePosition.Column,
					Suggestion: fmt.Sprintf("Add `// %s ...` before the declaration.", parseType.Name.Name),
				})
			}
		}
	}
	return parseDiagnostics
}

func insertGoDocStubs(parseSource []byte, parseDiagnostics []agenticDiagnostic) []byte {
	parseLines := strings.SplitAfter(string(parseSource), "\n")
	parseByLine := map[int]string{}
	for _, parseDiagnostic := range parseDiagnostics {
		if len(parseDiagnostic.Edits) == 0 || parseDiagnostic.Line <= 0 {
			continue
		}
		parseByLine[parseDiagnostic.Line] += parseDiagnostic.Edits[0].NewText
	}
	if len(parseByLine) == 0 {
		return parseSource
	}
	var parseBuilder strings.Builder
	for parseIndex, parseLine := range parseLines {
		parseLineNo := parseIndex + 1
		if parseInsert := parseByLine[parseLineNo]; parseInsert != "" {
			parseBuilder.WriteString(parseInsert)
		}
		parseBuilder.WriteString(parseLine)
	}
	return []byte(parseBuilder.String())
}

type cleanSummary struct {
	OK      bool              `json:"ok"`
	Root    string            `json:"root"`
	DryRun  bool              `json:"dryRun"`
	Targets map[string]bool   `json:"targets"`
	Removed []cleanPathRecord `json:"removed,omitempty"`
}

type cleanPathRecord struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Operation string `json:"operation"`
}

func (parseL launcher) runClean(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root"}, []string{"dry-run", "json", "artifacts", "cache", "bin"})
	parseFlags := flag.NewFlagSet("clean", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to clean")
	parseDryRun := parseFlags.Bool("dry-run", false, "List removals without deleting")
	parseArtifacts := parseFlags.Bool("artifacts", false, "Clean generated artifacts")
	parseCache := parseFlags.Bool("cache", false, "Clean launcher caches")
	parseBin := parseFlags.Bool("bin", false, "Clean bin outputs")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	if !*parseArtifacts && !*parseCache && !*parseBin {
		*parseArtifacts, *parseCache, *parseBin = true, true, true
	}
	parseSummary := cleanSummary{OK: true, Root: parseRootPath, DryRun: *parseDryRun, Targets: map[string]bool{"artifacts": *parseArtifacts, "cache": *parseCache, "bin": *parseBin}}
	parseCandidates := buildCleanCandidates(parseRootPath, *parseArtifacts, *parseCache, *parseBin)
	for _, parseCandidate := range parseCandidates {
		parseRel := relativeSlashPath(parseRootPath, parseCandidate)
		parseInfo, parseStatErr := os.Stat(parseCandidate)
		if errors.Is(parseStatErr, os.ErrNotExist) {
			continue
		}
		if parseStatErr != nil {
			return parseStatErr
		}
		parseKind := "file"
		if parseInfo.IsDir() {
			parseKind = "dir"
		}
		parseOperation := "remove"
		if *parseDryRun {
			parseOperation = "would-remove"
		}
		parseSummary.Removed = append(parseSummary.Removed, cleanPathRecord{Path: parseRel, Kind: parseKind, Operation: parseOperation})
		if !*parseDryRun {
			if parseErr := os.RemoveAll(parseCandidate); parseErr != nil {
				return parseErr
			}
		}
	}
	if *parseJSON {
		return writeAgenticEnvelope("clean", true, parseSummary, nil, nil)
	}
	printAgenticHumanSummary("GWC clean", true, []string{fmt.Sprintf("paths: %d", len(parseSummary.Removed))})
	return nil
}

func buildCleanCandidates(parseRoot string, isArtifacts bool, isCache bool, isBin bool) []string {
	parseCandidates := []string{}
	if isBin {
		parseCandidates = append(parseCandidates, filepath.Join(parseRoot, "bin"))
	}
	if isCache {
		parseCandidates = append(parseCandidates, filepath.Join(parseRoot, ".gwc", "cache"), filepath.Join(parseRoot, ".gwc", "tmp"))
	}
	if isArtifacts {
		parseCandidates = append(parseCandidates,
			filepath.Join(parseRoot, "dist"),
			filepath.Join(parseRoot, "release"),
			filepath.Join(parseRoot, "examples", "site-dist"),
			filepath.Join(parseRoot, "app.wasm"),
			filepath.Join(parseRoot, "main.wasm"),
			filepath.Join(parseRoot, "wasm_exec.js"),
		)
	}
	parseSafe := []string{}
	for _, parseCandidate := range parseCandidates {
		parseAbs, parseErr := filepath.Abs(parseCandidate)
		if parseErr != nil || !pathIsWithinRoot(parseRoot, parseAbs) || parseAbs == parseRoot {
			continue
		}
		parseSafe = append(parseSafe, parseAbs)
	}
	sort.Strings(parseSafe)
	return parseSafe
}

type watchSummary struct {
	OK         bool         `json:"ok"`
	Root       string       `json:"root"`
	Lanes      []string     `json:"lanes"`
	Watch      bool         `json:"watch"`
	Once       bool         `json:"once"`
	LastResult *testSummary `json:"lastResult,omitempty"`
}

func (parseL launcher) runWatch(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "app", "main", "debounce", "lane", "target", "features"}, []string{"json", "once"})
	parseFlags := flag.NewFlagSet("watch", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to watch")
	parseApp := parseFlags.String("app", "", "Path to the app main.go file or app directory")
	parseMain := parseFlags.String("main", "", "Deprecated alias for -app")
	parseDebounce := parseFlags.Duration("debounce", 500*time.Millisecond, "Polling debounce interval")
	parseTarget := parseFlags.String("target", "web", "Test target: web or desktop")
	parseFeatures := parseFlags.String("features", "all", "Native feature ceiling for desktop target")
	parseOnce := parseFlags.Bool("once", false, "Run one watched test pass and exit")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	var parseLanes stringListFlag
	parseFlags.Var(&parseLanes, "lane", "Test lane to run; repeat or comma-separate")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseTargetValue, parseTargetErr := normalizeBuildTarget(*parseTarget)
	if parseTargetErr != nil {
		return parseTargetErr
	}
	if parseTargetValue == "web" && strings.TrimSpace(strings.ToLower(*parseFeatures)) != "all" {
		return errors.New("-features is only valid with -target desktop")
	}
	parseFeaturesValue := strings.TrimSpace(*parseFeatures)
	if parseTargetValue == "desktop" {
		parseFeaturesValue, parseTargetErr = normalizeDesktopFeatures(parseFeaturesValue)
		if parseTargetErr != nil {
			return parseTargetErr
		}
	}
	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	parseConfig, parseErr := resolveTestConfig(testConfig{rootPath: parseRootPath, appPath: firstNonEmpty(*parseApp, *parseMain), lanes: parseLanes.Values(), target: parseTargetValue, features: parseFeaturesValue, json: false})
	if parseErr != nil {
		return parseErr
	}
	parseSummary := watchSummary{OK: true, Root: parseConfig.rootPath, Lanes: parseConfig.lanes, Watch: true, Once: *parseOnce}
	parseRunOnce := func() error {
		parseResult, parseRunErr := parseL.executeTest(parseConfig)
		parseSummary.LastResult = &parseResult
		parseSummary.OK = parseRunErr == nil
		return parseRunErr
	}
	parseErr = parseRunOnce()
	if *parseOnce || *parseJSON {
		if *parseJSON {
			_ = writeAgenticEnvelope("watch", parseSummary.OK, parseSummary, nil, parseErr)
		}
		return parseErr
	}
	parseLastFingerprint := fingerprintWatchTree(parseConfig.rootPath)
	for {
		time.Sleep(*parseDebounce)
		parseNextFingerprint := fingerprintWatchTree(parseConfig.rootPath)
		if parseNextFingerprint == parseLastFingerprint {
			continue
		}
		parseLastFingerprint = parseNextFingerprint
		parseErr = parseRunOnce()
		if parseErr != nil {
			fmt.Fprintf(os.Stdout, "gwc watch: test failed: %v\n", parseErr)
		} else {
			fmt.Fprintln(os.Stdout, "gwc watch: tests passed")
		}
	}
}

func fingerprintWatchTree(parseRoot string) string {
	var parseBuilder strings.Builder
	_ = filepath.WalkDir(parseRoot, func(parsePath string, parseEntry fs.DirEntry, parseErr error) error {
		if parseErr != nil {
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parseEntry.Name(), ".go") {
			return nil
		}
		parseInfo, parseStatErr := parseEntry.Info()
		if parseStatErr == nil {
			fmt.Fprintf(&parseBuilder, "%s:%d:%d\n", relativeSlashPath(parseRoot, parsePath), parseInfo.ModTime().UnixNano(), parseInfo.Size())
		}
		return nil
	})
	return fmt.Sprintf("%x", parseBuilder.String())
}

type sizeSummary struct {
	OK              bool                `json:"ok"`
	Artifact        string              `json:"artifact"`
	Bytes           int64               `json:"bytes"`
	AttributedBytes int64               `json:"attributedBytes"`
	Packages        []sizeContribution  `json:"packages,omitempty"`
	Symbols         []sizeSymbol        `json:"symbols,omitempty"`
	Diagnostics     []agenticDiagnostic `json:"diagnostics,omitempty"`
}

type sizeContribution struct {
	Package string `json:"package"`
	Bytes   int64  `json:"bytes"`
}

type sizeSymbol struct {
	Name    string `json:"name"`
	Package string `json:"package"`
	Bytes   int64  `json:"bytes"`
}

func (parseL launcher) runSize(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"artifact", "limit"}, []string{"json"})
	parseFlags := flag.NewFlagSet("size", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseArtifact := parseFlags.String("artifact", "", "Wasm artifact path to attribute")
	parseLimit := parseFlags.Int("limit", 20, "Maximum symbols/packages to return")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseSummary, parseErr := buildSizeSummary(*parseArtifact, *parseLimit)
	if *parseJSON {
		_ = writeAgenticEnvelope("size", parseSummary.OK, parseSummary, parseSummary.Diagnostics, parseErr)
	}
	if !*parseJSON {
		printAgenticHumanSummary("GWC size", parseSummary.OK, []string{fmt.Sprintf("bytes: %d", parseSummary.Bytes), fmt.Sprintf("attributed: %d", parseSummary.AttributedBytes)})
	}
	return parseErr
}

func buildSizeSummary(parseArtifact string, parseLimit int) (sizeSummary, error) {
	parseArtifact = strings.TrimSpace(parseArtifact)
	if parseArtifact == "" {
		parseErr := errors.New("size requires -artifact")
		return sizeSummary{OK: false, Diagnostics: []agenticDiagnostic{buildAgenticCommandError("GWC-SIZE-CONFIG", parseErr)}}, parseErr
	}
	parseAbs, parseErr := filepath.Abs(parseArtifact)
	if parseErr != nil {
		return sizeSummary{OK: false}, parseErr
	}
	parseInfo, parseErr := os.Stat(parseAbs)
	if parseErr != nil {
		return sizeSummary{OK: false, Artifact: parseAbs}, parseErr
	}
	parseSummary := sizeSummary{OK: true, Artifact: parseAbs, Bytes: parseInfo.Size()}
	parseOutput, parseErr := launcherRunCommand("go", []string{"tool", "nm", "-size", parseAbs}, "", os.Environ())
	if parseErr != nil {
		parseSummary.OK = false
		parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-SIZE-NM", parseErr))
		return parseSummary, parseErr
	}
	parsePackageBytes := map[string]int64{}
	for _, parseLine := range strings.Split(parseOutput, "\n") {
		parseFields := strings.Fields(parseLine)
		if len(parseFields) < 4 {
			continue
		}
		parseSize, parseConvErr := strconv.ParseInt(parseFields[1], 10, 64)
		if parseConvErr != nil || parseSize <= 0 {
			continue
		}
		parseName := strings.Join(parseFields[3:], " ")
		parsePackage := packageFromSymbolName(parseName)
		parseSummary.Symbols = append(parseSummary.Symbols, sizeSymbol{Name: parseName, Package: parsePackage, Bytes: parseSize})
		parsePackageBytes[parsePackage] += parseSize
		parseSummary.AttributedBytes += parseSize
	}
	sort.Slice(parseSummary.Symbols, func(parseLeft, parseRight int) bool {
		if parseSummary.Symbols[parseLeft].Bytes == parseSummary.Symbols[parseRight].Bytes {
			return parseSummary.Symbols[parseLeft].Name < parseSummary.Symbols[parseRight].Name
		}
		return parseSummary.Symbols[parseLeft].Bytes > parseSummary.Symbols[parseRight].Bytes
	})
	for parsePackage, parseBytes := range parsePackageBytes {
		parseSummary.Packages = append(parseSummary.Packages, sizeContribution{Package: parsePackage, Bytes: parseBytes})
	}
	sort.Slice(parseSummary.Packages, func(parseLeft, parseRight int) bool {
		if parseSummary.Packages[parseLeft].Bytes == parseSummary.Packages[parseRight].Bytes {
			return parseSummary.Packages[parseLeft].Package < parseSummary.Packages[parseRight].Package
		}
		return parseSummary.Packages[parseLeft].Bytes > parseSummary.Packages[parseRight].Bytes
	})
	if parseLimit > 0 {
		if len(parseSummary.Symbols) > parseLimit {
			parseSummary.Symbols = parseSummary.Symbols[:parseLimit]
		}
		if len(parseSummary.Packages) > parseLimit {
			parseSummary.Packages = parseSummary.Packages[:parseLimit]
		}
	}
	return parseSummary, nil
}

func packageFromSymbolName(parseName string) string {
	parseName = strings.TrimSpace(parseName)
	if parseSlash := strings.LastIndex(parseName, "/"); parseSlash >= 0 {
		parseName = parseName[parseSlash+1:]
	}
	if parseDot := strings.Index(parseName, "."); parseDot > 0 {
		return parseName[:parseDot]
	}
	return "(unknown)"
}

type docsSummary struct {
	OK       bool          `json:"ok"`
	Root     string        `json:"root"`
	Out      string        `json:"out,omitempty"`
	Packages []docsPackage `json:"packages"`
}

type docsPackage struct {
	Package string       `json:"package"`
	Symbols []docsSymbol `json:"symbols"`
}

type docsSymbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Signature string `json:"signature,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

func (parseL launcher) runDocs(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "out"}, []string{"json"})
	parseFlags := flag.NewFlagSet("docs", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to document")
	parseOut := parseFlags.String("out", "", "Optional markdown output path")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	parseSummary, parseErr := buildDocsSummary(parseRootPath, *parseOut)
	if *parseJSON {
		_ = writeAgenticEnvelope("docs", parseErr == nil, parseSummary, nil, parseErr)
	}
	if !*parseJSON {
		printAgenticHumanSummary("GWC docs", parseErr == nil, []string{fmt.Sprintf("packages: %d", len(parseSummary.Packages))})
	}
	return parseErr
}

func buildDocsSummary(parseRoot string, parseOut string) (docsSummary, error) {
	parseManifest, parseErr := buildAgenticModelManifest(parseRoot)
	if parseErr != nil {
		return docsSummary{}, parseErr
	}
	parseOut = strings.TrimSpace(parseOut)
	if parseOut != "" && !filepath.IsAbs(parseOut) {
		parseOut = filepath.Join(parseRoot, parseOut)
	}
	parseByPackage := map[string][]docsSymbol{}
	for _, parseSymbol := range parseManifest.Symbols {
		if !parseSymbol.Exported {
			continue
		}
		parsePackage := firstNonEmpty(parseSymbol.Package, parseSymbol.PackageName)
		parseByPackage[parsePackage] = append(parseByPackage[parsePackage], docsSymbol{
			Name: parseSymbol.Name, Kind: parseSymbol.Kind, File: parseSymbol.File, Line: parseSymbol.Line, Signature: parseSymbol.Signature, Summary: parseSymbol.Summary,
		})
	}
	parseSummary := docsSummary{OK: true, Root: parseRoot, Out: parseOut}
	for parsePackage, parseSymbols := range parseByPackage {
		sort.Slice(parseSymbols, func(parseLeft, parseRight int) bool {
			return parseSymbols[parseLeft].Name < parseSymbols[parseRight].Name
		})
		parseSummary.Packages = append(parseSummary.Packages, docsPackage{Package: parsePackage, Symbols: parseSymbols})
	}
	sort.Slice(parseSummary.Packages, func(parseLeft, parseRight int) bool {
		return parseSummary.Packages[parseLeft].Package < parseSummary.Packages[parseRight].Package
	})
	if parseSummary.Out != "" {
		if parseErr := os.MkdirAll(filepath.Dir(parseSummary.Out), 0o755); parseErr != nil {
			return parseSummary, parseErr
		}
		if parseErr := os.WriteFile(parseSummary.Out, []byte(renderDocsMarkdown(parseSummary)), 0o644); parseErr != nil {
			return parseSummary, parseErr
		}
	}
	return parseSummary, nil
}

func renderDocsMarkdown(parseSummary docsSummary) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("# GWC Project API\n\n")
	for _, parsePackage := range parseSummary.Packages {
		parseBuilder.WriteString("## " + parsePackage.Package + "\n\n")
		for _, parseSymbol := range parsePackage.Symbols {
			parseBuilder.WriteString("### " + parseSymbol.Name + "\n\n")
			if parseSymbol.Signature != "" {
				parseBuilder.WriteString("`" + parseSymbol.Signature + "`\n\n")
			}
			if parseSymbol.Summary != "" {
				parseBuilder.WriteString(parseSymbol.Summary + "\n\n")
			}
		}
	}
	return parseBuilder.String()
}

type deadcodeSummary struct {
	OK      bool             `json:"ok"`
	Root    string           `json:"root"`
	Symbols []deadcodeSymbol `json:"symbols"`
}

type deadcodeSymbol struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Package     string `json:"package,omitempty"`
	Uncertainty string `json:"uncertainty,omitempty"`
}

func (parseL launcher) runDeadcode(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root"}, []string{"json"})
	parseFlags := flag.NewFlagSet("deadcode", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to analyze")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	parseSummary, parseErr := buildDeadcodeSummary(parseRootPath)
	if *parseJSON {
		_ = writeAgenticEnvelope("deadcode", parseErr == nil, parseSummary, nil, parseErr)
	}
	if !*parseJSON {
		printAgenticHumanSummary("GWC deadcode", parseErr == nil, []string{fmt.Sprintf("symbols: %d", len(parseSummary.Symbols))})
	}
	return parseErr
}

func buildDeadcodeSummary(parseRoot string) (deadcodeSummary, error) {
	parseManifest, parseErr := buildAgenticModelManifest(parseRoot)
	if parseErr != nil {
		return deadcodeSummary{}, parseErr
	}
	parseReferenced := map[string]bool{}
	for _, parseEdge := range parseManifest.Edges {
		parseReferenced[parseEdge.To] = true
	}
	parseSummary := deadcodeSummary{OK: true, Root: parseRoot}
	for _, parseSymbol := range parseManifest.Symbols {
		if !parseSymbol.Exported || parseReferenced[parseSymbol.ID] || parseSymbol.Name == "main" || parseSymbol.Name == "App" {
			continue
		}
		parseSummary.Symbols = append(parseSummary.Symbols, deadcodeSymbol{
			ID: parseSymbol.ID, Name: parseSymbol.Name, Kind: parseSymbol.Kind, File: parseSymbol.File, Line: parseSymbol.Line, Package: parseSymbol.Package, Uncertainty: "static analysis; reflection and external callers may reference this symbol",
		})
	}
	sort.Slice(parseSummary.Symbols, func(parseLeft, parseRight int) bool {
		return parseSummary.Symbols[parseLeft].ID < parseSummary.Symbols[parseRight].ID
	})
	return parseSummary, nil
}

type depsSummary struct {
	OK          bool                `json:"ok"`
	Root        string              `json:"root"`
	DryRun      bool                `json:"dryRun"`
	Latest      bool                `json:"latest"`
	Modules     []depsModule        `json:"modules"`
	Diagnostics []agenticDiagnostic `json:"diagnostics,omitempty"`
}

type depsModule struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Update  string `json:"update,omitempty"`
	Main    bool   `json:"main,omitempty"`
}

func (parseL launcher) runDeps(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"root", "module", "to"}, []string{"json", "latest", "dry-run"})
	parseFlags := flag.NewFlagSet("deps", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root containing go.mod")
	parseLatest := parseFlags.Bool("latest", false, "Ask go list for available updates")
	parseModule := parseFlags.String("module", "", "Module path to update")
	parseTo := parseFlags.String("to", "", "Version to update to")
	parseDryRun := parseFlags.Bool("dry-run", false, "Report planned update without writing")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	parseSummary, parseErr := buildDepsSummary(parseRootPath, *parseLatest)
	parseSummary.DryRun = *parseDryRun
	if parseErr == nil && strings.TrimSpace(*parseModule) != "" {
		parseErr = applyDepsUpdate(parseRootPath, *parseModule, *parseTo, *parseDryRun)
		if parseErr != nil {
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-DEPS-UPDATE", parseErr))
		}
	}
	if *parseJSON {
		_ = writeAgenticEnvelope("deps", parseSummary.OK, parseSummary, parseSummary.Diagnostics, parseErr)
	}
	if !*parseJSON {
		printAgenticHumanSummary("GWC deps", parseSummary.OK, []string{fmt.Sprintf("modules: %d", len(parseSummary.Modules))})
	}
	return parseErr
}

func buildDepsSummary(parseRoot string, isLatest bool) (depsSummary, error) {
	parseArgs := []string{"list", "-m", "-json"}
	if isLatest {
		parseArgs = append(parseArgs, "-u")
	}
	parseArgs = append(parseArgs, "all")
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseRoot, os.Environ())
	parseSummary := depsSummary{OK: parseErr == nil, Root: parseRoot, Latest: isLatest}
	if parseErr != nil {
		parseSummary.Diagnostics = append(parseSummary.Diagnostics, buildAgenticCommandError("GWC-DEPS-LIST", parseErr))
		return parseSummary, parseErr
	}
	parseDecoder := json.NewDecoder(strings.NewReader(parseOutput))
	for {
		var parseRecord struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
			Main    bool   `json:"Main"`
			Update  *struct {
				Version string `json:"Version"`
			} `json:"Update"`
		}
		if parseErr := parseDecoder.Decode(&parseRecord); parseErr != nil {
			break
		}
		parseModule := depsModule{Path: parseRecord.Path, Version: parseRecord.Version, Main: parseRecord.Main}
		if parseRecord.Update != nil {
			parseModule.Update = parseRecord.Update.Version
		}
		parseSummary.Modules = append(parseSummary.Modules, parseModule)
	}
	sort.Slice(parseSummary.Modules, func(parseLeft, parseRight int) bool {
		return parseSummary.Modules[parseLeft].Path < parseSummary.Modules[parseRight].Path
	})
	return parseSummary, nil
}

func applyDepsUpdate(parseRoot string, parseModule string, parseTo string, isDryRun bool) error {
	parseModule = strings.TrimSpace(parseModule)
	parseTo = strings.TrimSpace(parseTo)
	if parseModule == "" || parseTo == "" {
		return errors.New("deps update requires -module and -to")
	}
	if isDryRun {
		return nil
	}
	parseGoMod := filepath.Join(parseRoot, "go.mod")
	parseGoSum := filepath.Join(parseRoot, "go.sum")
	parseGoModBytes, parseErr := os.ReadFile(parseGoMod)
	if parseErr != nil {
		return parseErr
	}
	parseGoSumBytes, _ := os.ReadFile(parseGoSum)
	parseOutput, parseErr := launcherRunCommand("go", []string{"get", parseModule + "@" + parseTo}, parseRoot, os.Environ())
	if parseErr == nil {
		_, parseErr = launcherRunCommand("go", []string{"test", "./..."}, parseRoot, buildNativeGoEnv())
	}
	if parseErr != nil {
		_ = os.WriteFile(parseGoMod, parseGoModBytes, 0o644)
		if len(parseGoSumBytes) > 0 {
			_ = os.WriteFile(parseGoSum, parseGoSumBytes, 0o644)
		}
		return fmt.Errorf("dependency update rolled back after failure: %w (%s)", parseErr, parseOutput)
	}
	return nil
}

func resolveAgenticToolRoot(parseRoot string) (string, error) {
	if strings.TrimSpace(parseRoot) == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return "", parseErr
		}
		parseRoot = parseCwd
	}
	parseAbs, parseErr := filepath.Abs(parseRoot)
	if parseErr != nil {
		return "", parseErr
	}
	parseInfo, parseErr := os.Stat(parseAbs)
	if parseErr != nil {
		return "", parseErr
	}
	if !parseInfo.IsDir() {
		return "", fmt.Errorf("root is not a directory: %s", parseAbs)
	}
	return parseAbs, nil
}

func pathIsWithinRoot(parseRoot string, parsePath string) bool {
	parseRel, parseErr := filepath.Rel(parseRoot, parsePath)
	if parseErr != nil {
		return false
	}
	return parseRel != "." && !strings.HasPrefix(parseRel, ".."+string(filepath.Separator)) && parseRel != ".."
}
