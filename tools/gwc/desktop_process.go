package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var desktopRunProcess = runDesktopProcess

var desktopRequiredSmokeChecks = []string{"dom", "local-counter", "native-call", "native-error", "native-event", "backend-cancellation", "unmount-cancellation", "adapter-cleanup", "routing", "route-remount", "wasm-mime", "csp", "durable-native-state", "two-window-invalidation", "window-local-state", "native-handshake", "origin-guard", "api-tester-ui", "api-window-info", "api-session-report", "desktop-file-contract", "native-sdk-contract"}

// desktopSmokeProbe requires both the expected process exit and complete structured evidence.
func desktopSmokeProbe(parseConfig desktopConfig, parseArtifact string, parseFault string) error {
	parseArgs := []string{"--smoke-test"}
	if parseFault != "" {
		parseArgs = append(parseArgs, "--smoke-fault", parseFault)
	}
	if parseConfig.timeout <= 0 || parseConfig.timeout > 60*time.Second {
		parseConfig.timeout = 60 * time.Second
	}
	parseOutput, parseErr := desktopExecute(parseConfig, parseArtifact, parseArgs, parseConfig.root, desktopNativeEnv())
	isFailure := parseFault == "missing-binding" || parseFault == "missing-wasm"
	if isFailure {
		var parseExit *exec.ExitError
		if !errors.As(parseErr, &parseExit) || parseExit.ExitCode() != 1 {
			return fmt.Errorf("desktop fault %s requires exit 1, got %v (%s)", parseFault, parseErr, parseOutput)
		}
	} else if parseErr != nil {
		return fmt.Errorf("desktop smoke failed: %w (%s)", parseErr, parseOutput)
	}
	return validateDesktopSmokeOutput(parseOutput, parseFault)
}

// validateDesktopSmokeOutput checks the final JSON report instead of accepting incidental log text.
func validateDesktopSmokeOutput(parseOutput string, parseFault string) error {
	var parseReport struct {
		OK     *bool    `json:"ok"`
		Checks []string `json:"checks"`
		Error  string   `json:"error"`
	}
	isReported := false
	for _, parseLine := range strings.Split(parseOutput, "\n") {
		var parseNext struct {
			OK     *bool    `json:"ok"`
			Checks []string `json:"checks"`
			Error  string   `json:"error"`
		}
		if json.Unmarshal([]byte(parseLine), &parseNext) == nil && parseNext.OK != nil && parseNext.Checks != nil {
			parseReport = parseNext
			isReported = true
		}
	}
	if !isReported {
		return fmt.Errorf("desktop smoke produced no JSON report: %s", parseOutput)
	}
	isFailure := parseFault == "missing-binding" || parseFault == "missing-wasm"
	if *parseReport.OK == isFailure || (isFailure && parseReport.Error == "") || (!isFailure && parseReport.Error != "") {
		return fmt.Errorf("desktop smoke report contradicts expected outcome: %s", parseOutput)
	}
	parseRequired := desktopRequiredSmokeChecks
	if isFailure {
		parseRequired = []string{"visible-boot-error", "native-controls-unavailable"}
	} else if parseFault == "streaming" {
		parseRequired = append(append([]string(nil), parseRequired...), "wasm-fallback")
	}
	if !isFailure {
		parseRequired = removeDesktopSmokeChecks(parseRequired, "durable-native-state", "two-window-invalidation", "window-local-state", "api-window-info")
		if desktopHasSmokeCheck(parseReport.Checks, "persistent-storage-disabled") {
			parseRequired = append(parseRequired, "persistent-storage-disabled")
		} else {
			parseRequired = append(parseRequired, "durable-native-state", "two-window-invalidation", "window-local-state")
		}
		if desktopHasSmokeCheck(parseReport.Checks, "native-window-disabled") {
			parseRequired = append(parseRequired, "native-window-disabled")
		} else {
			parseRequired = append(parseRequired, "api-window-info")
		}
	}
	for _, parseCheck := range parseRequired {
		isFound := false
		for _, parseActual := range parseReport.Checks {
			if parseActual == parseCheck {
				isFound = true
				break
			}
		}
		if !isFound {
			return fmt.Errorf("desktop smoke missing check %q", parseCheck)
		}
	}
	return nil
}

// desktopHasSmokeCheck reports whether a WebView emitted one named evidence marker.
func desktopHasSmokeCheck(parseChecks []string, parseWant string) bool {
	for _, parseCheck := range parseChecks {
		if parseCheck == parseWant {
			return true
		}
	}
	return false
}

// removeDesktopSmokeChecks removes checks that are conditional on host policy.
func removeDesktopSmokeChecks(parseChecks []string, parseRemove ...string) []string {
	parseResult := make([]string, 0, len(parseChecks))
	for _, parseCheck := range parseChecks {
		if !desktopHasSmokeCheck(parseRemove, parseCheck) {
			parseResult = append(parseResult, parseCheck)
		}
	}
	return parseResult
}

// desktopExecute bounds each child operation and preserves the caller's cancellation.
func desktopExecute(parseConfig desktopConfig, parseName string, parseArgs []string, parseDir string, parseEnv []string) (string, error) {
	parseContext := parseConfig.context
	if parseContext == nil {
		parseContext = context.Background()
	}
	parseTimeout := parseConfig.timeout
	if parseTimeout <= 0 {
		parseTimeout = 5 * time.Minute
	}
	parseContext, parseCancel := context.WithTimeout(parseContext, parseTimeout)
	defer parseCancel()
	return desktopRunProcess(parseContext, parseName, parseArgs, parseDir, parseEnv)
}

// desktopProcessOutput retains only the last megabyte of child diagnostics.
type desktopProcessOutput struct {
	parseMutex sync.Mutex
	parseBytes []byte
}

// Write accepts all bytes while keeping bounded diagnostic memory.
func (parseOutput *desktopProcessOutput) Write(parseData []byte) (int, error) {
	parseOutput.parseMutex.Lock()
	defer parseOutput.parseMutex.Unlock()
	parseCount := len(parseData)
	const parseLimit = 1024 * 1024
	if len(parseData) > parseLimit {
		parseData = parseData[len(parseData)-parseLimit:]
	}
	parseOutput.parseBytes = append(parseOutput.parseBytes, parseData...)
	if len(parseOutput.parseBytes) > parseLimit {
		parseOutput.parseBytes = append([]byte(nil), parseOutput.parseBytes[len(parseOutput.parseBytes)-parseLimit:]...)
	}
	return parseCount, nil
}

// runDesktopProcess captures diagnostics and kills the owned process tree on cancellation.
func runDesktopProcess(parseContext context.Context, parseName string, parseArgs []string, parseDir string, parseEnv []string) (string, error) {
	parseCommand := exec.CommandContext(parseContext, parseName, parseArgs...)
	parseCommand.Dir, parseCommand.Env = parseDir, parseEnv
	parseOutput := &desktopProcessOutput{}
	parseCommand.Stdout, parseCommand.Stderr = parseOutput, parseOutput
	parseCommand.Cancel = func() error { terminateLauncherProcessTree(parseCommand); return nil }
	parseCommand.WaitDelay = 2 * time.Second
	parseErr := parseCommand.Run()
	parseText := strings.TrimSpace(string(parseOutput.parseBytes))
	if parseContext.Err() != nil {
		return parseText, fmt.Errorf("%s interrupted: %w", parseName, parseContext.Err())
	}
	if parseErr != nil {
		return parseText, fmt.Errorf("%s failed: %w", parseName, parseErr)
	}
	return parseText, nil
}

// desktopVerifyPin checks the actual checkout and resolved module, not merely command success.
func (parseL launcher) desktopVerifyPin(parseConfig desktopConfig) error {
	parseCheckout := filepath.Join(parseL.repoRoot, "third_party", "wails")
	parseRevision, parseErr := desktopExecute(parseConfig, "git", []string{"-C", parseCheckout, "rev-parse", "HEAD"}, "", desktopNativeEnv())
	if parseErr != nil {
		return fmt.Errorf("pinned Wails revision unavailable: %w (%s)", parseErr, parseRevision)
	}
	if strings.TrimSpace(parseRevision) != desktopWailsRevision {
		return fmt.Errorf("Wails revision %q does not match pin %s", strings.TrimSpace(parseRevision), desktopWailsRevision)
	}
	parseDirty, parseErr := desktopExecute(parseConfig, "git", []string{"-C", parseCheckout, "status", "--porcelain"}, "", desktopNativeEnv())
	if parseErr != nil {
		return fmt.Errorf("inspect Wails source: %w", parseErr)
	}
	if strings.TrimSpace(parseDirty) != "" {
		return fmt.Errorf("pinned Wails checkout has local changes: %s", parseDirty)
	}
	parseOutput, parseErr := desktopExecute(parseConfig, "go", []string{"list", "-m", "-json", "github.com/wailsapp/wails/v3"}, parseConfig.root, desktopNativeEnv())
	if parseErr != nil {
		return fmt.Errorf("resolve Wails module: %w (%s)", parseErr, parseOutput)
	}
	var parseModule struct {
		Version string
		Replace *struct{ Dir string }
	}
	if parseErr = json.Unmarshal([]byte(parseOutput), &parseModule); parseErr != nil {
		return fmt.Errorf("decode Wails module: %w", parseErr)
	}
	if parseModule.Version != desktopWailsVersion || parseModule.Replace == nil {
		return fmt.Errorf("Wails module must use %s and the pinned local replacement", desktopWailsVersion)
	}
	parseWant, parseErr := filepath.Abs(filepath.Join(parseCheckout, "v3"))
	if parseErr != nil {
		return parseErr
	}
	parseActual, parseErr := filepath.Abs(parseModule.Replace.Dir)
	if parseErr != nil {
		return parseErr
	}
	if !strings.EqualFold(filepath.Clean(parseActual), filepath.Clean(parseWant)) {
		return fmt.Errorf("Wails replacement %q does not match pinned source %q", parseActual, parseWant)
	}
	return nil
}

// desktopWriteExclusive never truncates an existing file, including a dangling symlink.
func desktopWriteExclusive(parsePath string, parseData []byte) error {
	parseFile, parseErr := os.OpenFile(parsePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if parseErr != nil {
		return parseErr
	}
	_, parseErr = parseFile.Write(parseData)
	if parseCloseErr := parseFile.Close(); parseErr == nil {
		parseErr = parseCloseErr
	}
	if parseErr != nil {
		_ = os.Remove(parsePath)
	}
	return parseErr
}

// desktopRejectOutputTree rejects links anywhere in the generated asset tree before the helper writes.
func desktopRejectOutputTree(parseRoot string, parseTarget string) error {
	if parseErr := desktopRejectSymlinkPath(parseRoot, parseTarget); parseErr != nil {
		return parseErr
	}
	if _, parseErr := os.Lstat(parseTarget); os.IsNotExist(parseErr) {
		return nil
	} else if parseErr != nil {
		return parseErr
	}
	return filepath.WalkDir(parseTarget, func(parsePath string, parseEntry os.DirEntry, parseErr error) error {
		if parseErr != nil {
			return parseErr
		}
		if parseEntry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("desktop output tree refuses symlink: %s", parsePath)
		}
		return nil
	})
}
