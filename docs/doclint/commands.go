package doclint

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GwcCommandRef is one documented gwc command line found in a Markdown shell block.
type GwcCommandRef struct {
	DocPath string
	Line    int
	Raw     string
	Args    []string
}

// GwcCommandPlan is the executable or explicitly skipped plan for one documented command.
type GwcCommandPlan struct {
	Ref        GwcCommandRef
	Args       []string
	SkipReason string
}

// GwcCommandResult records one attempted or skipped documented command.
type GwcCommandResult struct {
	Plan   GwcCommandPlan
	Output string
	Err    error
}

// GwcCommandRunSummary is the result of the command-execution doclint lane.
type GwcCommandRunSummary struct {
	Executed []GwcCommandResult
	Skipped  []GwcCommandResult
}

// CommandRunner executes one command from one working directory with extra environment variables.
type CommandRunner func(context.Context, string, []string, string, []string) (string, error)

// GwcCommandRunOptions configures the documented-command execution lane.
type GwcCommandRunOptions struct {
	OutputRoot  string
	Timeout     time.Duration
	MaxCommands int
	Runner      CommandRunner
}

var finiteGwcCommands = map[string]bool{
	"doctor":   true,
	"env":      true,
	"examples": true,
	"files":    true,
	"build":    true,
}

var longRunningGwcCommands = map[string]string{
	"dev":       "dev starts a long-running server unless -dry-run is present",
	"serve":     "serve starts a long-running server",
	"dashboard": "dashboard starts an interactive status UI",
}

var heavyGwcCommands = map[string]string{
	"bench":     "bench is a benchmark lane, not a doc smoke command",
	"release":   "release writes a full artifact tree and can run smoke validation",
	"test":      "test may run broad wasm/browser lanes",
	"verify":    "verify may run tests and full app validation",
	"wasm":      "wasm subcommands may build and compare artifacts",
	"deploy":    "deploy needs pre-existing release artifacts and target policy",
	"prerender": "prerender can execute app-specific rendering work",
}

var interactiveGwcCommands = map[string]string{
	"start":     "start is interactive unless a preset flow is fully specified",
	"bootstrap": "bootstrap can enter interactive scaffolding flows",
	"seed":      "seed is app-specific and can mutate local data stores",
	"import":    "import converts user-supplied source files",
	"upgrade":   "upgrade mutates project files",
	"migrate":   "migrate can rewrite source files",
}

// ScanDocGwcCommands returns gwc command lines from Markdown shell blocks.
func ScanDocGwcCommands(parseRoot string) ([]GwcCommandRef, error) {
	var parseRefs []GwcCommandRef
	parseErr := filepath.Walk(parseRoot, func(parsePath string, parseInfo os.FileInfo, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseInfo.IsDir() {
			if dirsSkipped[parseInfo.Name()] {
				return filepath.SkipDir
			}
			if parseInfo.Name() == "pkg" && strings.Contains(filepath.ToSlash(parsePath), "browser-compiler") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(parsePath)) != ".md" {
			return nil
		}
		parseRel, parseRelErr := filepath.Rel(parseRoot, parsePath)
		if parseRelErr != nil {
			return parseRelErr
		}
		parseRelSlash := filepath.ToSlash(parseRel)
		if strings.HasPrefix(parseRelSlash, snapshotDocDir) {
			return nil
		}
		parseFileRefs, parseScanErr := scanDocGwcCommandFile(parsePath, parseRelSlash)
		if parseScanErr != nil {
			return parseScanErr
		}
		parseRefs = append(parseRefs, parseFileRefs...)
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}
	sort.Slice(parseRefs, func(parseI, parseJ int) bool {
		if parseRefs[parseI].DocPath == parseRefs[parseJ].DocPath {
			return parseRefs[parseI].Line < parseRefs[parseJ].Line
		}
		return parseRefs[parseI].DocPath < parseRefs[parseJ].DocPath
	})
	return parseRefs, nil
}

func scanDocGwcCommandFile(parsePath, parseDocRel string) ([]GwcCommandRef, error) {
	parseFile, parseErr := os.Open(parsePath)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseFile.Close()

	var parseRefs []GwcCommandRef
	parseScanner := bufio.NewScanner(parseFile)
	parseScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	parseInFence := false
	parseBlockIsShell := false
	parseLineNo := 0
	for parseScanner.Scan() {
		parseLineNo++
		parseLine := parseScanner.Text()
		if parseMatch := fenceOpen.FindStringSubmatch(parseLine); parseMatch != nil {
			if parseInFence {
				parseInFence = false
				parseBlockIsShell = false
			} else {
				parseInFence = true
				parseBlockIsShell = shellLangs[strings.ToLower(parseMatch[1])]
			}
			continue
		}
		if !parseInFence || (!parseBlockIsShell && !looksLikeCommand(parseLine)) {
			continue
		}
		parseArgs, parseOk := parseGwcCommandArgs(parseLine)
		if !parseOk {
			continue
		}
		parseRefs = append(parseRefs, GwcCommandRef{
			DocPath: parseDocRel,
			Line:    parseLineNo,
			Raw:     strings.TrimSpace(parseLine),
			Args:    parseArgs,
		})
	}
	if parseErr := parseScanner.Err(); parseErr != nil {
		return nil, parseErr
	}
	return parseRefs, nil
}

func parseGwcCommandArgs(parseLine string) ([]string, bool) {
	parseFields, parseErr := splitShellFields(stripShellPrompt(parseLine))
	if parseErr != nil || len(parseFields) == 0 {
		return nil, false
	}
	if strings.EqualFold(parseFields[0], "go") &&
		len(parseFields) >= 3 &&
		strings.EqualFold(parseFields[1], "run") &&
		isToolsGwcToken(parseFields[2]) {
		return normalizeCommandPathArgs(parseFields[3:]), true
	}
	if strings.EqualFold(parseFields[0], "gwc") {
		return normalizeCommandPathArgs(parseFields[1:]), true
	}
	return nil, false
}

func stripShellPrompt(parseLine string) string {
	parseTrimmed := strings.TrimSpace(parseLine)
	for _, parsePrefix := range []string{"PS> ", "$ ", "> "} {
		if after, ok := strings.CutPrefix(parseTrimmed, parsePrefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return parseTrimmed
}

func splitShellFields(parseLine string) ([]string, error) {
	var parseFields []string
	var parseBuilder strings.Builder
	var parseQuote rune
	parseFlush := func() {
		if parseBuilder.Len() == 0 {
			return
		}
		parseFields = append(parseFields, parseBuilder.String())
		parseBuilder.Reset()
	}
	for _, parseRune := range parseLine {
		if parseQuote != 0 {
			if parseRune == parseQuote {
				parseQuote = 0
				continue
			}
			parseBuilder.WriteRune(parseRune)
			continue
		}
		switch parseRune {
		case '\'', '"':
			parseQuote = parseRune
		case ' ', '\t':
			parseFlush()
		default:
			parseBuilder.WriteRune(parseRune)
		}
	}
	if parseQuote != 0 {
		return nil, fmt.Errorf("unterminated quoted command field in %q", parseLine)
	}
	parseFlush()
	return parseFields, nil
}

func isToolsGwcToken(parseToken string) bool {
	parseNormalized := strings.TrimPrefix(filepath.ToSlash(strings.Trim(parseToken, "\"'")), "./")
	return parseNormalized == "tools/gwc"
}

func normalizeCommandPathArgs(parseArgs []string) []string {
	parseOut := make([]string, 0, len(parseArgs))
	for _, parseArg := range parseArgs {
		parseOut = append(parseOut, normalizeDocCommandArg(parseArg))
	}
	return parseOut
}

func normalizeDocCommandArg(parseArg string) string {
	parseTrimmed := strings.Trim(parseArg, "\"'")
	if strings.Contains(parseTrimmed, `\`) {
		parseTrimmed = filepath.ToSlash(parseTrimmed)
	}
	if strings.HasPrefix(parseTrimmed, "./") || strings.HasPrefix(parseTrimmed, "../") {
		return filepath.FromSlash(parseTrimmed)
	}
	return parseTrimmed
}

// PlanDocGwcCommands classifies documented commands into finite executable commands and explicit skips.
func PlanDocGwcCommands(parseRefs []GwcCommandRef, parseOutputRoot string) []GwcCommandPlan {
	parsePlans := make([]GwcCommandPlan, 0, len(parseRefs))
	for _, parseRef := range parseRefs {
		parsePlan := planDocGwcCommand(parseRef, parseOutputRoot)
		parsePlans = append(parsePlans, parsePlan)
	}
	return parsePlans
}

func planDocGwcCommand(parseRef GwcCommandRef, parseOutputRoot string) GwcCommandPlan {
	parsePlan := GwcCommandPlan{Ref: parseRef, Args: append([]string(nil), parseRef.Args...)}
	if len(parsePlan.Args) == 0 {
		parsePlan.SkipReason = "command placeholder has no gwc subcommand"
		return parsePlan
	}
	if parseHasPlaceholderArg(parsePlan.Args) {
		parsePlan.SkipReason = "command contains placeholder arguments"
		return parsePlan
	}
	parseCommand := strings.ToLower(parsePlan.Args[0])
	if isHelpCommand(parsePlan.Args) {
		return parsePlan
	}
	if parseReason, parseOk := longRunningGwcCommands[parseCommand]; parseOk {
		if parseCommand == "dev" && hasFlag(parsePlan.Args, "-dry-run") {
			return parsePlan
		}
		parsePlan.SkipReason = parseReason
		return parsePlan
	}
	if parseReason, parseOk := heavyGwcCommands[parseCommand]; parseOk {
		parsePlan.SkipReason = parseReason
		return parsePlan
	}
	if parseReason, parseOk := interactiveGwcCommands[parseCommand]; parseOk {
		parsePlan.SkipReason = parseReason
		return parsePlan
	}
	if !finiteGwcCommands[parseCommand] {
		parsePlan.SkipReason = "unknown or app-specific gwc command"
		return parsePlan
	}
	if parseCommand == "doctor" && hasFlag(parsePlan.Args, "-audit") {
		parsePlan.SkipReason = "doctor -audit is broader than a command smoke check"
		return parsePlan
	}
	if parseCommand == "doctor" {
		parsePlan.SkipReason = "doctor depends on local toolchain and browser prerequisites"
		return parsePlan
	}
	if parseCommand == "examples" {
		if hasFlag(parsePlan.Args, "-export-static-catalog") {
			if parseOutputRoot == "" {
				parsePlan.SkipReason = "examples export-static-catalog needs an output sandbox"
				return parsePlan
			}
			parsePlan.Args = rewriteFlagOutputArg(parsePlan.Args, "-export-static-catalog", filepath.Join(parseOutputRoot, stableCommandOutputName(parseRef)+".json"))
			return parsePlan
		}
		parsePlan.SkipReason = "examples starts a long-running server unless exporting the catalog"
		return parsePlan
	}
	if parseCommand == "build" {
		if hasFlagValue(parsePlan.Args, "-profile", "tinygo") {
			parsePlan.SkipReason = "tinygo profile requires an optional external toolchain"
			return parsePlan
		}
		if parseOutputRoot == "" {
			parsePlan.SkipReason = "build command needs an output sandbox"
			return parsePlan
		}
		parsePlan.Args = rewriteBuildOutputArg(parsePlan.Args, parseOutputRoot, parseRef)
	}
	return parsePlan
}

func parseHasPlaceholderArg(parseArgs []string) bool {
	for _, parseArg := range parseArgs {
		parseLower := strings.ToLower(filepath.ToSlash(parseArg))
		if strings.Contains(parseLower, "<") || strings.Contains(parseLower, ">") ||
			strings.Contains(parseLower, "...") || isPlaceholder(parseLower) ||
			parseLower == "./main.go" || parseLower == ".\\main.go" || parseLower == "main.go" ||
			parseLower == "./static" || parseLower == ".\\static" || parseLower == "./design/landing.html" {
			return true
		}
	}
	return false
}

func isHelpCommand(parseArgs []string) bool {
	return hasFlag(parseArgs, "-h") || hasFlag(parseArgs, "--help") || hasFlag(parseArgs, "help")
}

func hasFlag(parseArgs []string, parseFlag string) bool {
	for _, parseArg := range parseArgs {
		if strings.EqualFold(parseArg, parseFlag) {
			return true
		}
	}
	return false
}

func hasFlagValue(parseArgs []string, parseFlag string, parseValue string) bool {
	for parseIndex, parseArg := range parseArgs {
		if !strings.EqualFold(parseArg, parseFlag) || parseIndex+1 >= len(parseArgs) {
			continue
		}
		if strings.EqualFold(parseArgs[parseIndex+1], parseValue) {
			return true
		}
	}
	return false
}

func rewriteBuildOutputArg(parseArgs []string, parseOutputRoot string, parseRef GwcCommandRef) []string {
	parseOut := append([]string(nil), parseArgs...)
	parseOutputPath := filepath.Join(parseOutputRoot, stableCommandOutputName(parseRef)+".wasm")
	return rewriteFlagOutputArg(parseOut, "-out", parseOutputPath)
}

func rewriteFlagOutputArg(parseArgs []string, parseFlag string, parseOutputPath string) []string {
	parseOut := append([]string(nil), parseArgs...)
	for parseIndex, parseArg := range parseOut {
		if strings.EqualFold(parseArg, parseFlag) && parseIndex+1 < len(parseOut) {
			parseOut[parseIndex+1] = parseOutputPath
			return parseOut
		}
	}
	return append(parseOut, parseFlag, parseOutputPath)
}

func stableCommandOutputName(parseRef GwcCommandRef) string {
	parseName := strings.NewReplacer("/", "_", "\\", "_", ".", "_", ":", "_").Replace(parseRef.DocPath)
	return parseName + "_" + strconv.Itoa(parseRef.Line)
}

// RunDocGwcCommands executes the finite planned commands with a network-off Go environment.
func RunDocGwcCommands(parseRoot string, parseOptions GwcCommandRunOptions) (GwcCommandRunSummary, error) {
	parseRefs, parseErr := ScanDocGwcCommands(parseRoot)
	if parseErr != nil {
		return GwcCommandRunSummary{}, parseErr
	}
	if len(parseRefs) == 0 {
		return GwcCommandRunSummary{}, errors.New("no documented gwc commands found")
	}
	parseOutputRoot := parseOptions.OutputRoot
	if parseOutputRoot == "" {
		parseOutputRoot = filepath.Join(os.TempDir(), "gwc-doclint-commands")
	}
	if parseErr2 := os.MkdirAll(parseOutputRoot, 0o755); parseErr2 != nil {
		return GwcCommandRunSummary{}, parseErr2
	}
	parseTimeout := parseOptions.Timeout
	if parseTimeout <= 0 {
		parseTimeout = 2 * time.Minute
	}
	parseMaxCommands := parseOptions.MaxCommands
	if parseMaxCommands <= 0 {
		parseMaxCommands = 25
	}
	parseRunner := parseOptions.Runner
	if parseRunner == nil {
		parseRunner = ExecCommandRunner
	}
	parsePlans := PlanDocGwcCommands(parseRefs, parseOutputRoot)
	parseSummary := GwcCommandRunSummary{}
	parseExecutedCount := 0
	parseEnv := docCommandGoEnv()
	for _, parsePlan := range parsePlans {
		if parsePlan.SkipReason != "" {
			parseSummary.Skipped = append(parseSummary.Skipped, GwcCommandResult{Plan: parsePlan})
			continue
		}
		if parseExecutedCount >= parseMaxCommands {
			parsePlan.SkipReason = "max command execution budget reached"
			parseSummary.Skipped = append(parseSummary.Skipped, GwcCommandResult{Plan: parsePlan})
			continue
		}
		parseExecutedCount++
		parseCtx, parseCancel := context.WithTimeout(context.Background(), parseTimeout)
		parseOutput, parseRunErr := parseRunner(parseCtx, parseRoot, parseEnv, "go", append([]string{"run", "./tools/gwc"}, parsePlan.Args...))
		parseCancel()
		parseSummary.Executed = append(parseSummary.Executed, GwcCommandResult{
			Plan:   parsePlan,
			Output: parseOutput,
			Err:    parseRunErr,
		})
	}
	return parseSummary, nil
}

// ExecCommandRunner runs one command and returns combined stdout/stderr.
func ExecCommandRunner(parseCtx context.Context, parseDir string, parseEnv []string, parseName string, parseArgs []string) (string, error) {
	parseCmd := exec.CommandContext(parseCtx, parseName, parseArgs...)
	parseCmd.Dir = parseDir
	parseCmd.Env = append(os.Environ(), parseEnv...)
	parseOutput, parseErr := parseCmd.CombinedOutput()
	if parseCtx.Err() != nil {
		return string(parseOutput), parseCtx.Err()
	}
	return string(parseOutput), parseErr
}

func docCommandGoEnv() []string {
	return []string{
		"GOPROXY=off",
		"GOSUMDB=off",
		"GONOSUMDB=*",
	}
}
