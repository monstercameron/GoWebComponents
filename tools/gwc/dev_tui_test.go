package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testExitCommand(parseCode int) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", fmt.Sprintf("exit %d", parseCode))
	}
	return exec.Command("sh", "-c", fmt.Sprintf("exit %d", parseCode))
}

func TestTailBufferWriteAndString(parseT *testing.T) {
	var parseNilBuffer *tailBuffer
	parseWritten, parseErr := parseNilBuffer.Write([]byte("ignored"))
	if parseErr != nil || parseWritten != len("ignored") {
		parseT.Fatalf("expected nil buffer write to succeed, wrote=%d err=%v", parseWritten, parseErr)
	}
	if parseNilBuffer.String() != "" {
		parseT.Fatalf("expected nil buffer string to be empty")
	}

	parseBuffer := &tailBuffer{limit: 5}
	if _, parseErr2 := parseBuffer.Write([]byte("hello")); parseErr2 != nil {
		parseT.Fatalf("write first chunk: %v", parseErr2)
	}
	if _, parseErr3 := parseBuffer.Write([]byte(" world ")); parseErr3 != nil {
		parseT.Fatalf("write second chunk: %v", parseErr3)
	}
	if parseGot := parseBuffer.String(); parseGot != "orld" {
		parseT.Fatalf("expected truncated trimmed tail, got %q", parseGot)
	}
}

func TestDevStatusModelPollStatusCmd(parseT *testing.T) {
	parseT.Run("success", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, _ *http.Request) {
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"projectRoot":"/tmp/project","clientCount":2}`))
		}))
		parseT2.Cleanup(parseServer.Close)

		parseMsg := (devStatusModel{statusURL: parseServer.URL}).pollStatusCmd()()
		parsePayload, parseOk := parseMsg.(devStatusMsg)
		if !parseOk {
			parseT2.Fatalf("expected devStatusMsg, got %T", parseMsg)
		}
		if parsePayload.err != nil {
			parseT2.Fatalf("expected no poll error, got %v", parsePayload.err)
		}
		if parsePayload.payload.ProjectRoot != "/tmp/project" || parsePayload.payload.ClientCount != 2 {
			parseT2.Fatalf("unexpected payload: %#v", parsePayload.payload)
		}
	})

	parseT.Run("non-200 status", func(parseT3 *testing.T) {
		parseServer2 := httptest.NewServer(http.HandlerFunc(func(parseW2 http.ResponseWriter, _ *http.Request) {
			http.Error(parseW2, "nope", http.StatusServiceUnavailable)
		}))
		parseT3.Cleanup(parseServer2.Close)

		parseMsg2 := (devStatusModel{statusURL: parseServer2.URL}).pollStatusCmd()()
		parsePayload2 := parseMsg2.(devStatusMsg)
		if parsePayload2.err == nil || !strings.Contains(parsePayload2.err.Error(), "status endpoint returned") {
			parseT3.Fatalf("expected status error, got %v", parsePayload2.err)
		}
	})

	parseT.Run("decode error", func(parseT4 *testing.T) {
		parseServer3 := httptest.NewServer(http.HandlerFunc(func(parseW3 http.ResponseWriter, _ *http.Request) {
			_, _ = parseW3.Write([]byte("{"))
		}))
		parseT4.Cleanup(parseServer3.Close)

		parseMsg3 := (devStatusModel{statusURL: parseServer3.URL}).pollStatusCmd()()
		parsePayload3 := parseMsg3.(devStatusMsg)
		if parsePayload3.err == nil {
			parseT4.Fatalf("expected decode error")
		}
	})

	parseT.Run("request error", func(parseT5 *testing.T) {
		parseMsg4 := (devStatusModel{statusURL: "http://127.0.0.1:1"}).pollStatusCmd()()
		parsePayload4 := parseMsg4.(devStatusMsg)
		if parsePayload4.err == nil {
			parseT5.Fatalf("expected request error")
		}
	})
}

func TestDevStatusModelWaitProcessCmd(parseT *testing.T) {
	parseCmd := testExitCommand(0)
	if parseErr := parseCmd.Start(); parseErr != nil {
		parseT.Fatalf("start process: %v", parseErr)
	}

	parseMsg := (devStatusModel{process: parseCmd}).waitProcessCmd()()
	parseExit, parseOk := parseMsg.(devProcessExitMsg)
	if !parseOk {
		parseT.Fatalf("expected devProcessExitMsg, got %T", parseMsg)
	}
	if parseExit.err != nil {
		parseT.Fatalf("expected zero-exit process, got %v", parseExit.err)
	}
}

func TestDevStatusModelUpdateAndInit(parseT *testing.T) {
	parseCmd := testExitCommand(0)
	if parseErr := parseCmd.Start(); parseErr != nil {
		parseT.Fatalf("start process: %v", parseErr)
	}
	defer func() { _ = parseCmd.Wait() }()

	parseModel := devStatusModel{
		statusURL: "http://127.0.0.1:1",
		process:   parseCmd,
	}
	if parseInitCmd := parseModel.Init(); parseInitCmd == nil {
		parseT.Fatalf("expected init command batch")
	}

	parseUpdated, parseCmdOut := parseModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	parseKeyModel := parseUpdated.(devStatusModel)
	if !parseKeyModel.userQuit || parseCmdOut == nil {
		parseT.Fatalf("expected quit signal after q key; model=%#v cmd=%v", parseKeyModel, parseCmdOut)
	}

	parseUpdated, parseTickCmd := parseModel.Update(devStatusMsg{err: errors.New("poll failed")})
	parseErrModel := parseUpdated.(devStatusModel)
	if parseErrModel.lastErr == nil || parseTickCmd == nil {
		parseT.Fatalf("expected stored poll error and tick command")
	}

	parseUpdated, parseTickCmd = parseModel.Update(devStatusMsg{payload: devStatusPayload{ProjectRoot: "/repo"}})
	parseOkModel := parseUpdated.(devStatusModel)
	if parseOkModel.status == nil || parseOkModel.status.ProjectRoot != "/repo" || parseOkModel.lastErr != nil || parseTickCmd == nil {
		parseT.Fatalf("expected status payload and cleared error, got %#v", parseOkModel)
	}

	parseUpdated, parseCmdOut = parseModel.Update(devProcessExitMsg{err: errors.New("exit failed")})
	parseExitModel := parseUpdated.(devStatusModel)
	if parseExitModel.exitedErr == nil || parseCmdOut == nil {
		parseT.Fatalf("expected process exit error and quit command, got %#v", parseExitModel)
	}

	parseUpdated, parseCmdOut = parseModel.Update(struct{}{})
	if parseCmdOut == nil {
		parseT.Fatalf("expected poll command after tick")
	}
	_ = parseUpdated
}

func TestDevStatusViewAndHelpers(parseT *testing.T) {
	parseT.Run("view states", func(parseT2 *testing.T) {
		parseBase := devStatusModel{
			plan: devPlanSummary{
				ProjectRoot:  "/repo",
				AppMode:      "app",
				ServerMode:   "dev",
				ListeningURL: "http://127.0.0.1:3000",
			},
			statusURL: "http://127.0.0.1:3000/__gwc/dev/status",
		}
		parseStarting := parseBase.View()
		if !strings.Contains(parseStarting, "starting dev server") {
			parseT2.Fatalf("expected startup placeholder, got %q", parseStarting)
		}

		parseBase.lastErr = errors.New("dial tcp: refused")
		parseWaiting := parseBase.View()
		if !strings.Contains(parseWaiting, "waiting for") || !strings.Contains(parseWaiting, "dial tcp") {
			parseT2.Fatalf("expected waiting error view, got %q", parseWaiting)
		}

		parseBase.lastErr = nil
		parseBase.status = &devStatusPayload{
			ServedWASMPath:    "bin/main.wasm",
			HotReloadEnabled:  true,
			HotReloadEligible: true,
			ClientCount:       4,
			LastClassification: devStatusClassify{
				Reason: "template changed",
			},
			LastBuild: &devStatusBuildStatus{
				Phase:        "blocked_on_error",
				PhaseSummary: "compile failed",
				StaleOutput:  true,
				ReloadType:   "full",
				Duration:     "2.1s",
			},
			CurrentError: &devStatusBuildStatus{Error: "undefined: foo"},
		}
		parseReady := parseBase.View()
		for _, parseWant := range []string{
			"blocked on error",
			"compile failed",
			"stale output",
			"template changed",
			"Current error:",
			"undefined: foo",
			"Fix the current compile error",
		} {
			if !strings.Contains(parseReady, parseWant) {
				parseT2.Fatalf("expected view to contain %q; got %q", parseWant, parseReady)
			}
		}
	})

	parseT.Run("phase labels and hints", func(parseT3 *testing.T) {
		if parseGot := devStatusPhaseLabel(""); parseGot != "unknown" {
			parseT3.Fatalf("expected unknown for empty phase, got %q", parseGot)
		}
		if parseGot2 := devStatusPhaseLabel("waiting_for_changes"); parseGot2 != "waiting for more file changes" {
			parseT3.Fatalf("unexpected phase label: %q", parseGot2)
		}
		if parseGot3 := devStatusPhaseLabel("custom_phase"); parseGot3 != "custom phase" {
			parseT3.Fatalf("expected fallback underscore normalization, got %q", parseGot3)
		}
		if parseGot4 := devStatusServingLabel(false); parseGot4 != "fresh output" {
			parseT3.Fatalf("unexpected fresh serving label: %q", parseGot4)
		}
		if parseGot5 := devStatusServingLabel(true); parseGot5 != "stale output" {
			parseT3.Fatalf("unexpected stale serving label: %q", parseGot5)
		}
		if parseGot6 := devStatusRecoveryHint(nil, errors.New("booting")); !strings.Contains(parseGot6, "Wait for the livereload status endpoint") {
			parseT3.Fatalf("unexpected recovery hint for errors: %q", parseGot6)
		}
		if parseGot7 := devStatusRecoveryHint(nil, nil); parseGot7 != "" {
			parseT3.Fatalf("expected empty hint for nil status, got %q", parseGot7)
		}
		if parseGot8 := devStatusRecoveryHint(&devStatusPayload{LastBuild: &devStatusBuildStatus{Phase: "compiling"}}, nil); !strings.Contains(parseGot8, "rebuilding now") {
			parseT3.Fatalf("unexpected compiling hint: %q", parseGot8)
		}
		if parseGot9 := devStatusRecoveryHint(&devStatusPayload{LastBuild: &devStatusBuildStatus{Phase: "waiting_for_reload"}}, nil); !strings.Contains(parseGot9, "Refresh the browser") {
			parseT3.Fatalf("unexpected waiting_for_reload hint: %q", parseGot9)
		}
	})
}

func TestRunDevStatusTUINonInteractiveGuard(parseT *testing.T) {
	parseOrigStdin := os.Stdin
	parseOrigStdout := os.Stdout
	parseInFile, parseErr := os.CreateTemp(parseT.TempDir(), "stdin-*")
	if parseErr != nil {
		parseT.Fatalf("create temp stdin: %v", parseErr)
	}
	parseOutFile, parseErr := os.CreateTemp(parseT.TempDir(), "stdout-*")
	if parseErr != nil {
		parseT.Fatalf("create temp stdout: %v", parseErr)
	}
	os.Stdin = parseInFile
	os.Stdout = parseOutFile
	parseT.Cleanup(func() {
		os.Stdin = parseOrigStdin
		os.Stdout = parseOrigStdout
		_ = parseInFile.Close()
		_ = parseOutFile.Close()
	})

	parseErr = runDevStatusTUI(devPlanSummary{}, "http://127.0.0.1:3000/__gwc/dev/status", testExitCommand(0))
	if parseErr == nil || !strings.Contains(parseErr.Error(), "requires an interactive terminal") {
		parseT.Fatalf("expected non-interactive guard error, got %v", parseErr)
	}
}
