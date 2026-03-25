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

func testExitCommand(code int) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", fmt.Sprintf("exit %d", code))
	}
	return exec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
}

func TestTailBufferWriteAndString(t *testing.T) {
	var nilBuffer *tailBuffer
	written, err := nilBuffer.Write([]byte("ignored"))
	if err != nil || written != len("ignored") {
		t.Fatalf("expected nil buffer write to succeed, wrote=%d err=%v", written, err)
	}
	if nilBuffer.String() != "" {
		t.Fatalf("expected nil buffer string to be empty")
	}

	buffer := &tailBuffer{limit: 5}
	if _, err := buffer.Write([]byte("hello")); err != nil {
		t.Fatalf("write first chunk: %v", err)
	}
	if _, err := buffer.Write([]byte(" world ")); err != nil {
		t.Fatalf("write second chunk: %v", err)
	}
	if got := buffer.String(); got != "orld" {
		t.Fatalf("expected truncated trimmed tail, got %q", got)
	}
}

func TestDevStatusModelPollStatusCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"projectRoot":"/tmp/project","clientCount":2}`))
		}))
		t.Cleanup(server.Close)

		msg := (devStatusModel{statusURL: server.URL}).pollStatusCmd()()
		payload, ok := msg.(devStatusMsg)
		if !ok {
			t.Fatalf("expected devStatusMsg, got %T", msg)
		}
		if payload.err != nil {
			t.Fatalf("expected no poll error, got %v", payload.err)
		}
		if payload.payload.ProjectRoot != "/tmp/project" || payload.payload.ClientCount != 2 {
			t.Fatalf("unexpected payload: %#v", payload.payload)
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)

		msg := (devStatusModel{statusURL: server.URL}).pollStatusCmd()()
		payload := msg.(devStatusMsg)
		if payload.err == nil || !strings.Contains(payload.err.Error(), "status endpoint returned") {
			t.Fatalf("expected status error, got %v", payload.err)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{"))
		}))
		t.Cleanup(server.Close)

		msg := (devStatusModel{statusURL: server.URL}).pollStatusCmd()()
		payload := msg.(devStatusMsg)
		if payload.err == nil {
			t.Fatalf("expected decode error")
		}
	})

	t.Run("request error", func(t *testing.T) {
		msg := (devStatusModel{statusURL: "http://127.0.0.1:1"}).pollStatusCmd()()
		payload := msg.(devStatusMsg)
		if payload.err == nil {
			t.Fatalf("expected request error")
		}
	})
}

func TestDevStatusModelWaitProcessCmd(t *testing.T) {
	cmd := testExitCommand(0)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	msg := (devStatusModel{process: cmd}).waitProcessCmd()()
	exit, ok := msg.(devProcessExitMsg)
	if !ok {
		t.Fatalf("expected devProcessExitMsg, got %T", msg)
	}
	if exit.err != nil {
		t.Fatalf("expected zero-exit process, got %v", exit.err)
	}
}

func TestDevStatusModelUpdateAndInit(t *testing.T) {
	cmd := testExitCommand(0)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	defer func() { _ = cmd.Wait() }()

	model := devStatusModel{
		statusURL: "http://127.0.0.1:1",
		process:   cmd,
	}
	if initCmd := model.Init(); initCmd == nil {
		t.Fatalf("expected init command batch")
	}

	updated, cmdOut := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	keyModel := updated.(devStatusModel)
	if !keyModel.userQuit || cmdOut == nil {
		t.Fatalf("expected quit signal after q key; model=%#v cmd=%v", keyModel, cmdOut)
	}

	updated, tickCmd := model.Update(devStatusMsg{err: errors.New("poll failed")})
	errModel := updated.(devStatusModel)
	if errModel.lastErr == nil || tickCmd == nil {
		t.Fatalf("expected stored poll error and tick command")
	}

	updated, tickCmd = model.Update(devStatusMsg{payload: devStatusPayload{ProjectRoot: "/repo"}})
	okModel := updated.(devStatusModel)
	if okModel.status == nil || okModel.status.ProjectRoot != "/repo" || okModel.lastErr != nil || tickCmd == nil {
		t.Fatalf("expected status payload and cleared error, got %#v", okModel)
	}

	updated, cmdOut = model.Update(devProcessExitMsg{err: errors.New("exit failed")})
	exitModel := updated.(devStatusModel)
	if exitModel.exitedErr == nil || cmdOut == nil {
		t.Fatalf("expected process exit error and quit command, got %#v", exitModel)
	}

	updated, cmdOut = model.Update(struct{}{})
	if cmdOut == nil {
		t.Fatalf("expected poll command after tick")
	}
	_ = updated
}

func TestDevStatusViewAndHelpers(t *testing.T) {
	t.Run("view states", func(t *testing.T) {
		base := devStatusModel{
			plan: devPlanSummary{
				ProjectRoot:  "/repo",
				AppMode:      "app",
				ServerMode:   "dev",
				ListeningURL: "http://127.0.0.1:3000",
			},
			statusURL: "http://127.0.0.1:3000/__gwc/dev/status",
		}
		starting := base.View()
		if !strings.Contains(starting, "starting dev server") {
			t.Fatalf("expected startup placeholder, got %q", starting)
		}

		base.lastErr = errors.New("dial tcp: refused")
		waiting := base.View()
		if !strings.Contains(waiting, "waiting for") || !strings.Contains(waiting, "dial tcp") {
			t.Fatalf("expected waiting error view, got %q", waiting)
		}

		base.lastErr = nil
		base.status = &devStatusPayload{
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
		ready := base.View()
		for _, want := range []string{
			"blocked on error",
			"compile failed",
			"stale output",
			"template changed",
			"Current error:",
			"undefined: foo",
			"Fix the current compile error",
		} {
			if !strings.Contains(ready, want) {
				t.Fatalf("expected view to contain %q; got %q", want, ready)
			}
		}
	})

	t.Run("phase labels and hints", func(t *testing.T) {
		if got := devStatusPhaseLabel(""); got != "unknown" {
			t.Fatalf("expected unknown for empty phase, got %q", got)
		}
		if got := devStatusPhaseLabel("waiting_for_changes"); got != "waiting for more file changes" {
			t.Fatalf("unexpected phase label: %q", got)
		}
		if got := devStatusPhaseLabel("custom_phase"); got != "custom phase" {
			t.Fatalf("expected fallback underscore normalization, got %q", got)
		}
		if got := devStatusServingLabel(false); got != "fresh output" {
			t.Fatalf("unexpected fresh serving label: %q", got)
		}
		if got := devStatusServingLabel(true); got != "stale output" {
			t.Fatalf("unexpected stale serving label: %q", got)
		}
		if got := devStatusRecoveryHint(nil, errors.New("booting")); !strings.Contains(got, "Wait for the livereload status endpoint") {
			t.Fatalf("unexpected recovery hint for errors: %q", got)
		}
		if got := devStatusRecoveryHint(nil, nil); got != "" {
			t.Fatalf("expected empty hint for nil status, got %q", got)
		}
		if got := devStatusRecoveryHint(&devStatusPayload{LastBuild: &devStatusBuildStatus{Phase: "compiling"}}, nil); !strings.Contains(got, "rebuilding now") {
			t.Fatalf("unexpected compiling hint: %q", got)
		}
		if got := devStatusRecoveryHint(&devStatusPayload{LastBuild: &devStatusBuildStatus{Phase: "waiting_for_reload"}}, nil); !strings.Contains(got, "Refresh the browser") {
			t.Fatalf("unexpected waiting_for_reload hint: %q", got)
		}
	})
}

func TestRunDevStatusTUINonInteractiveGuard(t *testing.T) {
	origStdin := os.Stdin
	origStdout := os.Stdout
	inFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("create temp stdin: %v", err)
	}
	outFile, err := os.CreateTemp(t.TempDir(), "stdout-*")
	if err != nil {
		t.Fatalf("create temp stdout: %v", err)
	}
	os.Stdin = inFile
	os.Stdout = outFile
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		_ = inFile.Close()
		_ = outFile.Close()
	})

	err = runDevStatusTUI(devPlanSummary{}, "http://127.0.0.1:3000/__gwc/dev/status", testExitCommand(0))
	if err == nil || !strings.Contains(err.Error(), "requires an interactive terminal") {
		t.Fatalf("expected non-interactive guard error, got %v", err)
	}
}
