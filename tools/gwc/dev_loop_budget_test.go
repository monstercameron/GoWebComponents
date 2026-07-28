package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// devLoopRebuildBudget is deliberately generous so CI hardware variance never
// flakes this test; the point is catching order-of-magnitude regressions in
// dev-loop latency as apps grow, with actual timings logged for trend review.
const devLoopRebuildBudget = 30 * time.Second

const devLoopBudgetComponentCount = 200

// TestDevLoopLargeAppRebuildBudget generates a scaffold padded with hundreds of
// components, then measures the rebuild latency for a single-file change via
// the dev server's timing telemetry and asserts it stays within budget.
func TestDevLoopLargeAppRebuildBudget(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping dev loop budget test in short mode")
	}

	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(parseT.TempDir(), "test-dev-loop-budget")

	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName:   "test-dev-loop-budget",
		ModulePath:    "github.com/example/test-dev-loop-budget",
		Author:        "Test Author",
		Version:       "1.0.0",
		Description:   "Dev loop budget app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	}

	parseResult, parseErr := parseLauncher.generateStartScaffold(parseSelection)
	if parseErr != nil {
		parseT.Fatalf("generate scaffold: %v", parseErr)
	}
	ensureScaffoldUsesLocalRepoModule(parseT, parseRepoRoot, parseTargetDir)

	// Pad the app with generated components in the app package, all reachable
	// from a registry slice so the linker cannot discard them.
	parseAppDir := filepath.Dir(parseResult.AppPath)
	parseRegistryNames := make([]string, 0, devLoopBudgetComponentCount)
	for parseIdx := range devLoopBudgetComponentCount {
		parseName := fmt.Sprintf("GenComponent%04d", parseIdx)
		parseRegistryNames = append(parseRegistryNames, parseName)
		parseSource := fmt.Sprintf(`package main

import (
	html "github.com/monstercameron/GoWebComponents/v5/html"
	ui "github.com/monstercameron/GoWebComponents/v5/ui"
)

// %s is a generated budget-fixture component.
func %s() ui.Node {
	return html.Div(html.Props{Class: "gen-%04d"})
}
`, parseName, parseName, parseIdx)
		if parseWriteErr := os.WriteFile(filepath.Join(parseAppDir, fmt.Sprintf("gen_component_%04d.go", parseIdx)), []byte(parseSource), 0o644); parseWriteErr != nil {
			parseT.Fatalf("write generated component %d: %v", parseIdx, parseWriteErr)
		}
	}
	var parseRegistry strings.Builder
	parseRegistry.WriteString("package main\n\nimport ui \"github.com/monstercameron/GoWebComponents/v5/ui\"\n\n// genRegistry keeps every generated component reachable for the linker.\nvar genRegistry = []func() ui.Node{\n")
	for _, parseName := range parseRegistryNames {
		parseRegistry.WriteString("\t" + parseName + ",\n")
	}
	parseRegistry.WriteString("}\n\nvar _ = genRegistry\n")
	if parseWriteErr := os.WriteFile(filepath.Join(parseAppDir, "gen_registry.go"), []byte(parseRegistry.String()), 0o644); parseWriteErr != nil {
		parseT.Fatalf("write component registry: %v", parseWriteErr)
	}

	parsePort, parseErr := reserveTCPPort()
	if parseErr != nil {
		parseT.Fatalf("reserve tcp port: %v", parseErr)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseCmd := exec.CommandContext(parseCtx, "go", append([]string{"run", "./tools/gwc", "dev"}, append(devArgsFromScaffold(parseResult), "-host", "127.0.0.1", "-port", parsePort)...)...)
	parseCmd.Dir = parseRepoRoot
	var parseOutput bytes.Buffer
	parseCmd.Stdout = &parseOutput
	parseCmd.Stderr = &parseOutput
	if parseErr2 := parseCmd.Start(); parseErr2 != nil {
		parseT.Fatalf("start dev server: %v", parseErr2)
	}
	parseProcessExited := make(chan struct{})
	var parseProcessErr error
	go func() {
		parseProcessErr = parseCmd.Wait()
		close(parseProcessExited)
	}()
	parseT.Cleanup(func() {
		parseCancel()
		terminateProcessTree(parseCmd)
		// The dev server is a grandchild behind two `go run` hops, so the tree kill
		// above routinely misses it. See killListenersOnPort.
		killListenersOnPort(parsePort)
		select {
		case <-parseProcessExited:
		case <-time.After(5 * time.Second):
			if parseCmd.Process != nil {
				_ = parseCmd.Process.Kill()
			}
		}
	})

	parseRootURL := "http://127.0.0.1:" + parsePort + "/"
	waitForHTTPBodyWithProcess(parseT, parseRootURL, 300*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		return nil
	})
	parseWasmURL := "http://127.0.0.1:" + parsePort + "/" + scaffoldWASMOutputPath()
	waitForHTTPBodyWithProcess(parseT, parseWasmURL, 300*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		return nil
	})

	parseWSURL := "ws://127.0.0.1:" + parsePort + "/ws"
	parseConn, _, parseErr := websocket.DefaultDialer.Dial(parseWSURL, nil)
	if parseErr != nil {
		parseT.Fatalf("dial dev server websocket: %v\n%s", parseErr, parseOutput.String())
	}
	parseT.Cleanup(func() { _ = parseConn.Close() })
	parseMessages := make(chan devLoopWSMessage, 256)
	go func() {
		for {
			var parseMsg devLoopWSMessage
			if parseReadErr := parseConn.ReadJSON(&parseMsg); parseReadErr != nil {
				close(parseMessages)
				return
			}
			parseMessages <- parseMsg
		}
	}()

	// Touch one generated component and measure the rebuild via telemetry.
	parseTouchPath := filepath.Join(parseAppDir, "gen_component_0042.go")
	parseTouchSource, parseErr3 := os.ReadFile(parseTouchPath)
	if parseErr3 != nil {
		parseT.Fatalf("read touch target: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseTouchPath, append(parseTouchSource, []byte("\n// budget probe\n")...), 0o644); parseErr4 != nil {
		parseT.Fatalf("touch component: %v", parseErr4)
	}

	parseDeadline := time.After(devLoopRebuildBudget + 60*time.Second)
	for {
		select {
		case parseMsg, parseOk := <-parseMessages:
			if !parseOk {
				parseT.Fatalf("websocket closed while waiting for rebuild\n%s", parseOutput.String())
			}
			if parseMsg.Type != "build_complete" || !strings.Contains(string(parseMsg.Payload), `"success":true`) {
				continue
			}
			var parsePayload struct {
				Timings map[string]int64 `json:"timings"`
			}
			if parseErr5 := json.Unmarshal(parseMsg.Payload, &parsePayload); parseErr5 != nil {
				parseT.Fatalf("decode build_complete payload: %v", parseErr5)
			}
			if parsePayload.Timings == nil {
				parseT.Fatalf("expected timing telemetry in build_complete payload, got %s", string(parseMsg.Payload))
			}
			parseCompileMs := parsePayload.Timings["compileMs"]
			parseArtifactBytes := parsePayload.Timings["artifactBytes"]
			parseT.Logf("large-app (%d components) single-file rebuild: compile=%dms artifact=%.1fMB", devLoopBudgetComponentCount, parseCompileMs, float64(parseArtifactBytes)/1024/1024)
			if time.Duration(parseCompileMs)*time.Millisecond > devLoopRebuildBudget {
				parseT.Fatalf("rebuild exceeded budget: %dms > %v — dev-loop latency regressed", parseCompileMs, devLoopRebuildBudget)
			}
			return
		case <-parseDeadline:
			parseT.Fatalf("timed out waiting for rebuild\n%s", parseOutput.String())
		case <-parseProcessExited:
			parseT.Fatalf("dev server exited: %v\n%s", parseProcessErr, parseOutput.String())
		}
	}
}
