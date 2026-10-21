package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// benchmarkMachine records the environment a benchmark report was generated on so that
// reports from different hosts, or from emulated runs, can be told apart.
type benchmarkMachine struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	HostArch    string `json:"hostArch,omitempty"`
	CPUModel    string `json:"cpuModel,omitempty"`
	LogicalCPUs int    `json:"logicalCpus"`
	RAMBytes    uint64 `json:"ramBytes,omitempty"`
	GoVersion   string `json:"goVersion,omitempty"`
	GOMAXPROCS  int    `json:"gomaxprocs"`
	GOFLAGS     string `json:"goflags,omitempty"`
	BuildTags   string `json:"buildTags,omitempty"`
	GitSHA      string `json:"gitSha,omitempty"`
	GitDirty    bool   `json:"gitDirty,omitempty"`
	// GoEnvGoarch is what `go env GOARCH` printed in the child environment the benchmarks ran in.
	GoEnvGoarch string `json:"goEnvGoarch,omitempty"`
	// ArchSource says where Arch came from: "benchmark-header" (the goarch line of the first
	// completed native target) or "go-env" (the child toolchain default, nothing observed).
	ArchSource string `json:"archSource,omitempty"`
	// RootGoMod holds the go and toolchain directives of <root>/go.mod, so a v6 and a v7 run can
	// be told apart by the toolchain their sources asked for.
	RootGoMod string `json:"rootGoMod,omitempty"`
	// RootGitSha is the commit of the measured root: git HEAD when the root itself is a
	// repository, else the first field of <root>/.gwc-source-sha, else empty.
	RootGitSha string `json:"rootGitSha,omitempty"`
}

// benchmarkSourceShaFile names the optional file that records the commit of an archive export.
const benchmarkSourceShaFile = ".gwc-source-sha"

// benchmarkQueryCommand runs a short read-only command and returns trimmed stdout, or an
// empty string when the command is unavailable; machine metadata must never fail a run.
var benchmarkQueryCommand = func(parseCwd string, parseName string, parseArgs ...string) string {
	parseCmd := exec.Command(parseName, parseArgs...)
	parseCmd.Dir = parseCwd
	parseOut, parseErr := parseCmd.Output()
	if parseErr != nil {
		return ""
	}
	return strings.TrimSpace(string(parseOut))
}

// benchmarkQueryCommandEnv is benchmarkQueryCommand with an explicit child environment; tests
// replace it.
var benchmarkQueryCommandEnv = func(parseCwd string, parseEnv []string, parseName string, parseArgs ...string) string {
	parseCmd := exec.Command(parseName, parseArgs...)
	parseCmd.Dir = parseCwd
	parseCmd.Env = parseEnv
	parseOut, parseErr := parseCmd.Output()
	if parseErr != nil {
		return ""
	}
	return strings.TrimSpace(string(parseOut))
}

// queryBenchmarkCommand runs a read-only query in parseCwd with parseEnv, or with the inherited
// environment when parseEnv is nil.
func queryBenchmarkCommand(parseCwd string, parseEnv []string, parseName string, parseArgs ...string) string {
	if parseEnv == nil {
		return benchmarkQueryCommand(parseCwd, parseName, parseArgs...)
	}
	return benchmarkQueryCommandEnv(parseCwd, parseEnv, parseName, parseArgs...)
}

// buildBenchmarkMachine collects best-effort machine metadata for a report rooted at parseRoot.
// Arch is the effective GOARCH of the toolchain (which can differ from the host when GOARCH is
// overridden via go env -w, making native runs emulated), HostArch is the real machine arch.
func buildBenchmarkMachine(parseRoot string, parseGoVersion string) *benchmarkMachine {
	return buildBenchmarkMachineEnv(parseRoot, parseGoVersion, nil)
}

// buildBenchmarkMachineEnv is buildBenchmarkMachine with every toolchain query run under parseEnv
// (the environment the child go test uses); a nil parseEnv inherits the parent environment.
func buildBenchmarkMachineEnv(parseRoot string, parseGoVersion string, parseEnv []string) *benchmarkMachine {
	parseMachine := &benchmarkMachine{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		LogicalCPUs: runtime.NumCPU(),
		GOMAXPROCS:  runtime.GOMAXPROCS(0),
		GoVersion:   strings.TrimSpace(parseGoVersion),
		CPUModel:    readBenchmarkCPUModel(),
		RAMBytes:    readBenchmarkRAMBytes(),
	}
	if parseEffective := queryBenchmarkCommand(parseRoot, parseEnv, "go", "env", "GOARCH"); parseEffective != "" {
		parseMachine.Arch = parseEffective
		parseMachine.GoEnvGoarch = parseEffective
		parseMachine.ArchSource = "go-env"
	}
	parseMachine.HostArch = queryBenchmarkCommand(parseRoot, parseEnv, "go", "env", "GOHOSTARCH")
	if parseMachine.HostArch == "" {
		parseMachine.HostArch = runtime.GOARCH
	}
	parseMachine.GOFLAGS = queryBenchmarkCommand(parseRoot, parseEnv, "go", "env", "GOFLAGS")
	parseMachine.BuildTags = parseBenchmarkBuildTags(parseMachine.GOFLAGS)
	parseMachine.GitSHA = benchmarkQueryCommand(parseRoot, "git", "rev-parse", "HEAD")
	if parseMachine.GitSHA != "" {
		parseMachine.GitDirty = benchmarkQueryCommand(parseRoot, "git", "status", "--porcelain") != ""
	}
	parseMachine.RootGoMod = getBenchmarkRootGoMod(parseRoot)
	parseMachine.RootGitSha = getBenchmarkRootGitSha(parseRoot)
	return parseMachine
}

// buildBenchSuiteMachine collects the machine block for a suite run: the Go version comes from
// `go version` run in parseRoot under the child environment, because the root may select a
// different toolchain than the one gwc itself was started with.
func buildBenchSuiteMachine(parseRoot string, parseEnv []string) *benchmarkMachine {
	parseVersion := parseBenchmarkGoVersion(queryBenchmarkCommand(parseRoot, parseEnv, "go", "version"))
	if parseVersion == "" {
		parseVersion = queryBenchmarkCommand(parseRoot, parseEnv, "go", "env", "GOVERSION")
	}
	return buildBenchmarkMachineEnv(parseRoot, parseVersion, parseEnv)
}

// parseBenchmarkGoVersion extracts the version token from `go version go1.26.3 windows/arm64`.
func parseBenchmarkGoVersion(parseText string) string {
	parseFields := strings.Fields(parseText)
	if len(parseFields) >= 3 && parseFields[0] == "go" && parseFields[1] == "version" {
		return parseFields[2]
	}
	return ""
}

// applyBenchmarkObservedPlatform records the goos and goarch printed in the benchmark header of
// a completed native target; what ran outranks what the toolchain was configured to do.
func applyBenchmarkObservedPlatform(parseMachine *benchmarkMachine, parseGoOS string, parseGoArch string) {
	if parseMachine == nil || parseGoArch == "" {
		return
	}
	parseMachine.Arch = parseGoArch
	parseMachine.ArchSource = "benchmark-header"
	if parseGoOS != "" {
		parseMachine.OS = parseGoOS
	}
}

// getBenchmarkRootGoMod returns the go and toolchain directives of <root>/go.mod joined by "; ".
func getBenchmarkRootGoMod(parseRoot string) string {
	parseContent, parseErr := os.ReadFile(filepath.Join(parseRoot, "go.mod"))
	if parseErr != nil {
		return ""
	}
	parseParts := []string{}
	for _, parseLine := range strings.Split(string(parseContent), "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseLine, "go ") || strings.HasPrefix(parseLine, "toolchain ") {
			parseParts = append(parseParts, parseLine)
		}
	}
	return strings.Join(parseParts, "; ")
}

// getBenchmarkRootGitSha returns the commit of the measured root without borrowing the commit of
// an enclosing repository: git HEAD only when parseRoot has its own .git, else the first field of
// <root>/.gwc-source-sha (written for archive exports), else empty.
func getBenchmarkRootGitSha(parseRoot string) string {
	if _, parseErr := os.Stat(filepath.Join(parseRoot, ".git")); parseErr == nil {
		return benchmarkQueryCommand(parseRoot, "git", "rev-parse", "HEAD")
	}
	parseContent, parseErr2 := os.ReadFile(filepath.Join(parseRoot, benchmarkSourceShaFile))
	if parseErr2 != nil {
		return ""
	}
	parseFields := strings.Fields(string(parseContent))
	if len(parseFields) == 0 {
		return ""
	}
	return parseFields[0]
}

// parseBenchmarkBuildTags extracts the -tags value from a GOFLAGS string.
func parseBenchmarkBuildTags(parseFlags string) string {
	for _, parseField := range strings.Fields(parseFlags) {
		if parseValue, isFound := strings.CutPrefix(parseField, "-tags="); isFound {
			return parseValue
		}
	}
	return ""
}

// formatBenchmarkMachine renders a one-line machine summary for console and drift output.
func formatBenchmarkMachine(parseMachine *benchmarkMachine) string {
	if parseMachine == nil {
		return ""
	}
	parseArch := parseMachine.OS + "/" + parseMachine.Arch
	if parseMachine.HostArch != "" && parseMachine.HostArch != parseMachine.Arch {
		parseArch += " (host " + parseMachine.HostArch + ", emulated)"
	}
	parseParts := []string{parseArch}
	if parseMachine.CPUModel != "" {
		parseParts = append(parseParts, parseMachine.CPUModel)
	}
	return strings.Join(parseParts, ", ")
}
