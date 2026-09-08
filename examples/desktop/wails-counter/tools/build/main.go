package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/desktop"
)

// runCommand executes a child in the example module and preserves failures.
func runCommand(parseDir string, parseEnv []string, parseName string, parseArgs ...string) error {
	parseCommand := exec.Command(parseName, parseArgs...)
	parseCommand.Dir = parseDir
	parseCommand.Env = parseEnv
	parseCommand.Stdout = os.Stdout
	parseCommand.Stderr = os.Stderr
	if parseErr := parseCommand.Run(); parseErr != nil {
		return fmt.Errorf("%s %v: %w", parseName, parseArgs, parseErr)
	}
	return nil
}

// cleanEnv removes target overrides before selecting an explicit child target.
func cleanEnv(parseBase []string) []string {
	parseResult := make([]string, 0, len(parseBase))
	for _, parseValue := range parseBase {
		parseKey, _, _ := strings.Cut(parseValue, "=")
		parseKey = strings.ToUpper(parseKey)
		if parseKey == "GOOS" || parseKey == "GOARCH" || parseKey == "CGO_ENABLED" || parseKey == "GOFLAGS" {
			continue
		}
		parseResult = append(parseResult, parseValue)
	}
	return parseResult
}

// copyFile copies the matching Go wasm runtime asset into the embedded dist.
func copyFile(parseSource, parseTarget string) error {
	parseBytes, parseErr := os.ReadFile(parseSource)
	if parseErr != nil {
		return fmt.Errorf("read %s: %w", parseSource, parseErr)
	}
	if parseErr = os.WriteFile(parseTarget, parseBytes, 0o644); parseErr != nil {
		return fmt.Errorf("write %s: %w", parseTarget, parseErr)
	}
	return nil
}

// copyFrontend copies immutable source assets into the ignored embed tree.
func copyFrontend(parseRoot, parseDist, parseTarget string) error {
	if parseTarget == "desktop" {
		if parseErr := os.WriteFile(filepath.Join(parseDist, "desktop.js"), []byte(desktop.BootstrapSource), 0o644); parseErr != nil {
			return parseErr
		}
	}
	for _, parseName := range []string{"bootstrap.js", "index.html", "app.css", "tester.css"} {
		parseSourceName := parseName
		if parseName == "bootstrap.js" && parseTarget == "web" {
			parseSourceName = "bootstrap.web.js"
		}
		if parseErr := copyFile(filepath.Join(parseRoot, "assets", "src", parseSourceName), filepath.Join(parseDist, parseName)); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// main builds the Windows host with explicit targets, regardless of persisted Go defaults.
func main() {
	parseFeatures := flag.String("features", "all", "native feature ceiling")
	parseTarget := flag.String("target", "desktop", "build target: web or desktop")
	flag.Parse()
	if strings.TrimSpace(*parseFeatures) == "" || strings.ContainsAny(*parseFeatures, "\r\n") {
		panic("features must be a non-empty single-line value")
	}
	parsePolicy, parseFeatureErr := desktop.ParseFeaturePolicy(*parseFeatures)
	if parseFeatureErr != nil {
		panic(parseFeatureErr)
	}
	parseCanonicalFeatures := strings.ToLower(strings.TrimSpace(*parseFeatures))
	if parseCanonicalFeatures != "all" && parseCanonicalFeatures != "none" {
		parseNames := []string{}
		for _, parseFeature := range parsePolicy.FeatureNames() {
			parseNames = append(parseNames, string(parseFeature))
		}
		parseCanonicalFeatures = strings.Join(parseNames, ",")
	}
	if *parseTarget != "desktop" && *parseTarget != "web" {
		panic("target must be web or desktop")
	}
	if *parseTarget == "web" && strings.TrimSpace(strings.ToLower(*parseFeatures)) != "all" {
		panic("features are only valid for the desktop target")
	}
	parseRoot, parseErr := os.Getwd()
	if parseErr != nil {
		panic(parseErr)
	}
	parseDist := filepath.Join(parseRoot, "assets", "dist")
	if *parseTarget == "web" {
		parseDist = filepath.Join(parseRoot, "assets", "web")
	}
	if parseErr = os.MkdirAll(parseDist, 0o755); parseErr != nil {
		panic(parseErr)
	}
	if parseErr = copyFrontend(parseRoot, parseDist, *parseTarget); parseErr != nil {
		panic(parseErr)
	}
	parseEnv := append(cleanEnv(os.Environ()), "GOFLAGS=-mod=mod", "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	parseWasmEnv := append(cleanEnv(parseEnv), "GOFLAGS=-mod=mod", "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
	parseWasmArgs := []string{"build"}
	if *parseTarget == "desktop" {
		parseWasmArgs = append(parseWasmArgs, "-tags", "gwc_desktop")
	}
	parseDistRelative, parseRelErr := filepath.Rel(parseRoot, parseDist)
	if parseRelErr != nil {
		panic(parseRelErr)
	}
	parseWasmArgs = append(parseWasmArgs, "-o", filepath.Join(filepath.ToSlash(parseDistRelative), "app.wasm"), "./frontend")
	if parseErr = runCommand(parseRoot, parseWasmEnv, "go", parseWasmArgs...); parseErr != nil {
		panic(parseErr)
	}
	if *parseTarget == "web" {
		parseGoRootOutput, parseGoRootErr := exec.Command("go", "env", "GOROOT").Output()
		if parseGoRootErr != nil {
			panic(fmt.Errorf("resolve GOROOT: %w", parseGoRootErr))
		}
		parseWasmExec := filepath.Join(strings.TrimSpace(string(parseGoRootOutput)), "lib", "wasm", "wasm_exec.js")
		if _, parseStatErr := os.Stat(parseWasmExec); parseStatErr != nil {
			panic(fmt.Errorf("matching wasm_exec.js missing at %s: %w", parseWasmExec, parseStatErr))
		}
		if parseErr = copyFile(parseWasmExec, filepath.Join(parseDist, "wasm_exec.js")); parseErr != nil {
			panic(parseErr)
		}
		return
	}
	if parseErr = runCommand(parseRoot, parseEnv, "go", "run", "github.com/wailsapp/wails/v3/cmd/wails3", "generate", "bindings", "-b", "-d", filepath.Join("assets", "dist", "bindings"), "./cmd/desktop"); parseErr != nil {
		panic(parseErr)
	}
	parseGoRootOutput, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		panic(fmt.Errorf("resolve GOROOT: %w", parseErr))
	}
	parseWasmExec := filepath.Join(strings.TrimSpace(string(parseGoRootOutput)), "lib", "wasm", "wasm_exec.js")
	if _, parseErr = os.Stat(parseWasmExec); parseErr != nil {
		panic(fmt.Errorf("matching wasm_exec.js missing at %s: %w", parseWasmExec, parseErr))
	}
	if parseErr = copyFile(parseWasmExec, filepath.Join(parseDist, "wasm_exec.js")); parseErr != nil {
		panic(parseErr)
	}
	if parseErr = runCommand(parseRoot, parseEnv, "go", "build", "-tags", "gwc_desktop", "-ldflags", "-X main.buildFeatures="+parseCanonicalFeatures, "-o", filepath.Join("bin", "wails-counter.exe"), "./cmd/desktop"); parseErr != nil {
		panic(parseErr)
	}
}
