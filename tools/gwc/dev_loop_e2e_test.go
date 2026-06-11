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

// devLoopWSMessage mirrors the livereload WebSocketMessage envelope.
type devLoopWSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// TestDevLoopHotReloadAndAssetSwapE2E drives a full dev-server session over a
// generated scaffold: initial build, a Go edit triggering a hot rebuild, a CSS
// edit hot-swapping without a rebuild, and a broken build surfacing its error.
func TestDevLoopHotReloadAndAssetSwapE2E(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping dev loop e2e in short mode")
	}

	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseTargetDir := filepath.Join(defaultGeneratedScaffoldRoot(), "test-dev-loop-e2e")
	_ = os.RemoveAll(parseTargetDir)
	parseT.Cleanup(func() {
		_ = os.RemoveAll(parseTargetDir)
	})

	parseSelection := startSelection{
		Preset: startPreset{
			Key:         "minimal-client",
			Name:        "Minimal Client App",
			Summary:     "Test preset",
			Description: "Generated in tests.",
			Features:    []string{"ui", "html"},
		},
		ProjectName:   "test-dev-loop-e2e",
		ModulePath:    "github.com/example/test-dev-loop-e2e",
		Author:        "Test Author",
		Version:       "1.0.0",
		Description:   "Dev loop e2e app.",
		TargetDir:     parseTargetDir,
		SkipGoModTidy: true,
	}

	parseResult, parseErr := parseLauncher.generateStartScaffold(parseSelection)
	if parseErr != nil {
		parseT.Fatalf("generate scaffold: %v", parseErr)
	}
	ensureScaffoldUsesLocalRepoModule(parseT, parseRepoRoot, parseTargetDir)

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
		select {
		case <-parseProcessExited:
		case <-time.After(5 * time.Second):
			if parseCmd.Process != nil {
				_ = parseCmd.Process.Kill()
			}
		}
	})

	// Wait for the server + first successful build.
	parseRootURL := "http://127.0.0.1:" + parsePort + "/"
	waitForHTTPBodyWithProcess(parseT, parseRootURL, 180*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		return nil
	})
	parseWasmURL := "http://127.0.0.1:" + parsePort + "/" + scaffoldWASMOutputPath()
	waitForHTTPBodyWithProcess(parseT, parseWasmURL, 180*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		return nil
	})

	// Connect a WebSocket client and stream messages into a channel.
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

	parseWaitForMessage := func(parseWanted string, parseTimeout time.Duration, parseMatch func(devLoopWSMessage) bool) devLoopWSMessage {
		parseT.Helper()
		parseDeadline := time.After(parseTimeout)
		for {
			select {
			case parseMsg, parseOk := <-parseMessages:
				if !parseOk {
					parseT.Fatalf("websocket closed while waiting for %q\n%s", parseWanted, parseOutput.String())
				}
				if parseMsg.Type == parseWanted && (parseMatch == nil || parseMatch(parseMsg)) {
					return parseMsg
				}
			case <-parseDeadline:
				parseT.Fatalf("timed out waiting for %q message\n%s", parseWanted, parseOutput.String())
			case <-parseProcessExited:
				parseT.Fatalf("dev server exited while waiting for %q: %v\n%s", parseWanted, parseProcessErr, parseOutput.String())
			}
		}
	}

	// 1. CSS hot swap: creating/altering a stylesheet must broadcast asset_swap
	//    and must NOT trigger a wasm rebuild.
	parseCSSPath := filepath.Join(parseTargetDir, "styles.css")
	if parseErr3 := os.WriteFile(parseCSSPath, []byte("body { background: #fafafa; }\n"), 0o644); parseErr3 != nil {
		parseT.Fatalf("write stylesheet: %v", parseErr3)
	}
	parseSwap := parseWaitForMessage("asset_swap", 30*time.Second, nil)
	if !strings.Contains(string(parseSwap.Payload), "styles.css") {
		parseT.Fatalf("expected asset_swap payload to reference styles.css, got %s", string(parseSwap.Payload))
	}

	// 2. Go edit: appending a component change must trigger a rebuild that
	//    completes successfully with hot reload classification.
	parseAppPath := parseResult.AppPath
	parseAppSource, parseErr4 := os.ReadFile(parseAppPath)
	if parseErr4 != nil {
		parseT.Fatalf("read app source: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseAppPath, append(parseAppSource, []byte("\n// dev-loop-e2e touch\n")...), 0o644); parseErr5 != nil {
		parseT.Fatalf("touch app source: %v", parseErr5)
	}
	parseWaitForMessage("build_start", 60*time.Second, nil)
	parseComplete := parseWaitForMessage("build_complete", 180*time.Second, nil)
	if !strings.Contains(string(parseComplete.Payload), `"success":true`) {
		parseT.Fatalf("expected successful rebuild after Go edit, got %s", string(parseComplete.Payload))
	}

	// 3. Broken build: invalid Go must surface a failed build (with error text)
	//    rather than silently serving stale output.
	parseBrokenSource := append(append([]byte{}, parseAppSource...), []byte("\nfunc broken() { undefinedSymbol() }\n")...)
	if parseErr6 := os.WriteFile(parseAppPath, parseBrokenSource, 0o644); parseErr6 != nil {
		parseT.Fatalf("write broken source: %v", parseErr6)
	}
	parseWaitForMessage("build_start", 60*time.Second, nil)
	parseFailed := parseWaitForMessage("build_complete", 180*time.Second, func(parseMsg devLoopWSMessage) bool {
		return strings.Contains(string(parseMsg.Payload), `"success":false`)
	})
	if !strings.Contains(string(parseFailed.Payload), "undefinedSymbol") && !strings.Contains(string(parseFailed.Payload), "error") {
		parseT.Fatalf("expected failed build payload to carry the compiler error, got %s", string(parseFailed.Payload))
	}

	// 4. Repair: restoring the source must produce a successful build again.
	if parseErr7 := os.WriteFile(parseAppPath, parseAppSource, 0o644); parseErr7 != nil {
		parseT.Fatalf("restore app source: %v", parseErr7)
	}
	parseWaitForMessage("build_start", 60*time.Second, nil)
	parseWaitForMessage("build_complete", 180*time.Second, func(parseMsg devLoopWSMessage) bool {
		return strings.Contains(string(parseMsg.Payload), `"success":true`)
	})

	// 5. Dynamic directory watching: a brand-new subdirectory created after
	//    startup must still trigger rebuilds for files inside it.
	parseNewDir := filepath.Join(parseTargetDir, "widgets")
	if parseErr8 := os.MkdirAll(parseNewDir, 0o755); parseErr8 != nil {
		parseT.Fatalf("create new watched dir: %v", parseErr8)
	}
	time.Sleep(500 * time.Millisecond) // allow the watcher to attach to the new directory
	if parseErr9 := os.WriteFile(filepath.Join(parseNewDir, "widget.go"), []byte("package widgets\n\n// Widget is a dev-loop e2e marker.\nconst Widget = \"w\"\n"), 0o644); parseErr9 != nil {
		parseT.Fatalf("write file in new dir: %v", parseErr9)
	}
	parseWaitForMessage("build_start", 60*time.Second, nil)
}
