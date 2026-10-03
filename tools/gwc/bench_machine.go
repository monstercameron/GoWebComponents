package main

import (
	"os/exec"
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
}

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

// buildBenchmarkMachine collects best-effort machine metadata for a report rooted at parseRoot.
// Arch is the effective GOARCH of the toolchain (which can differ from the host when GOARCH is
// overridden via go env -w, making native runs emulated), HostArch is the real machine arch.
func buildBenchmarkMachine(parseRoot string, parseGoVersion string) *benchmarkMachine {
	parseMachine := &benchmarkMachine{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		LogicalCPUs: runtime.NumCPU(),
		GOMAXPROCS:  runtime.GOMAXPROCS(0),
		GoVersion:   strings.TrimSpace(parseGoVersion),
		CPUModel:    readBenchmarkCPUModel(),
		RAMBytes:    readBenchmarkRAMBytes(),
	}
	if parseEffective := benchmarkQueryCommand(parseRoot, "go", "env", "GOARCH"); parseEffective != "" {
		parseMachine.Arch = parseEffective
	}
	parseMachine.HostArch = benchmarkQueryCommand(parseRoot, "go", "env", "GOHOSTARCH")
	if parseMachine.HostArch == "" {
		parseMachine.HostArch = runtime.GOARCH
	}
	parseMachine.GOFLAGS = benchmarkQueryCommand(parseRoot, "go", "env", "GOFLAGS")
	parseMachine.BuildTags = parseBenchmarkBuildTags(parseMachine.GOFLAGS)
	parseMachine.GitSHA = benchmarkQueryCommand(parseRoot, "git", "rev-parse", "HEAD")
	if parseMachine.GitSHA != "" {
		parseMachine.GitDirty = benchmarkQueryCommand(parseRoot, "git", "status", "--porcelain") != ""
	}
	return parseMachine
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
