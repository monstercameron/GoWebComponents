//go:build !js

// Command gowasmexec is the shell-free js/wasm test executor used as `go test -exec`.
// Go hands the compiled wasm binary and its -test.* flags to this program; it forwards
// every argument verbatim to Node's wasm_exec_node.js. The former go_js_wasm_exec.bat
// routed arguments through cmd.exe, where `^` is an escape character, so `-test.run=^$`
// degraded to `$` and matched every test.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runWasmExec starts nodePath with the wasm_exec_node.js script and the verbatim arguments
// and returns the child's exit code.
func runWasmExec(nodePath string, scriptPath string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	parseResolvedNodePath, parseLookErr := exec.LookPath(nodePath)
	if parseLookErr != nil {
		fmt.Fprintf(stderr, "gowasmexec: resolve node: %v\n", parseLookErr)
		return 1
	}
	parseCmd := exec.Command(parseResolvedNodePath, buildWasmExecArgs(scriptPath, args)...)
	parseCmd.Env = buildWasmExecEnv(parseResolvedNodePath)
	parseCmd.Stdin = stdin
	parseCmd.Stdout = stdout
	parseCmd.Stderr = stderr
	parseErr := parseCmd.Run()
	if parseErr == nil {
		return 0
	}
	var parseExit *exec.ExitError
	if errors.As(parseErr, &parseExit) {
		return parseExit.ExitCode()
	}
	fmt.Fprintf(stderr, "gowasmexec: %v\n", parseErr)
	return 1
}

// buildWasmExecEnv keeps the Node child environment below wasm_exec's 8 KiB argv/env limit.
func buildWasmExecEnv(parseNodePath string) []string {
	parseEnv := []string{}
	parseKeep := map[string]bool{
		"HOME":        true,
		"SystemRoot":  true,
		"TEMP":        true,
		"TMP":         true,
		"TMPDIR":      true,
		"USERPROFILE": true,
		"WINDIR":      true,
	}
	for _, parseItem := range os.Environ() {
		parseName, _, parseOK := strings.Cut(parseItem, "=")
		if parseOK && (parseKeep[parseName] || strings.HasPrefix(parseName, "GOWASMEXEC_")) {
			parseEnv = append(parseEnv, parseItem)
		}
	}
	parseNodeDir := filepath.Dir(parseNodePath)
	parseSystemRoot := os.Getenv("SystemRoot")
	parsePath := parseNodeDir
	if parseSystemRoot != "" {
		parsePath += string(os.PathListSeparator) + filepath.Join(parseSystemRoot, "System32")
	}
	parseEnv = append(parseEnv, "PATH="+parsePath)
	return parseEnv
}

// buildWasmExecArgs prepends the Node script path to the untouched wasm arguments.
func buildWasmExecArgs(scriptPath string, args []string) []string {
	return append([]string{scriptPath}, args...)
}

// getGoRoot returns GOROOT from the environment or `go env GOROOT`.
func getGoRoot() (string, error) {
	if parseValue := strings.TrimSpace(os.Getenv("GOROOT")); parseValue != "" {
		return parseValue, nil
	}
	parseOut, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		return "", fmt.Errorf("go env GOROOT: %w", parseErr)
	}
	return strings.TrimSpace(string(parseOut)), nil
}

func main() {
	parseRoot, parseErr := getGoRoot()
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "gowasmexec: %v\n", parseErr)
		os.Exit(1)
	}
	parseScript := filepath.Join(parseRoot, "lib", "wasm", "wasm_exec_node.js")
	os.Exit(runWasmExec("node", parseScript, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
