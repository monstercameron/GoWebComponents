package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type devStatusPayload struct {
	Mode               string                `json:"mode"`
	ListeningURL       string                `json:"listeningURL"`
	StatusURL          string                `json:"statusURL"`
	WebSocketURL       string                `json:"websocketURL"`
	ProjectRoot        string                `json:"projectRoot"`
	WatchRoot          string                `json:"watchRoot"`
	BuildDir           string                `json:"buildDir"`
	ServedWASMPath     string                `json:"servedWASMPath"`
	HotReloadEnabled   bool                  `json:"hotReloadEnabled"`
	HotReloadEligible  bool                  `json:"hotReloadEligible"`
	LastClassification devStatusClassify     `json:"lastClassification"`
	LastBuild          *devStatusBuildStatus `json:"lastBuild"`
	CurrentError       *devStatusBuildStatus `json:"currentError"`
	ClientCount        int                   `json:"clientCount"`
}

type devStatusClassify struct {
	ReloadType string `json:"reloadType"`
	Reason     string `json:"reason"`
}

type devStatusBuildStatus struct {
	Success      bool   `json:"success"`
	Duration     string `json:"duration"`
	Error        string `json:"error"`
	ReloadType   string `json:"reloadType"`
	Phase        string `json:"phase"`
	PhaseSummary string `json:"phaseSummary"`
	StaleOutput  bool   `json:"staleOutput"`
}

type devStatusMsg struct {
	payload devStatusPayload
	err     error
}

type devProcessExitMsg struct {
	err error
}

type devStatusModel struct {
	plan        devPlanSummary
	statusURL   string
	process     *exec.Cmd
	processTail *tailBuffer
	status      *devStatusPayload
	lastErr     error
	exitedErr   error
	userQuit    bool
}

type tailBuffer struct {
	limit int
	data  []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	if b == nil {
		return len(p), nil
	}
	b.data = append(b.data, p...)
	if b.limit > 0 && len(b.data) > b.limit {
		b.data = append([]byte(nil), b.data[len(b.data)-b.limit:]...)
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	if b == nil || len(b.data) == 0 {
		return ""
	}
	return string(bytes.TrimSpace(b.data))
}

var devStatusProgramRunner = func(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(model, tea.WithAltScreen()).Run()
}

func runDevStatusTUI(plan devPlanSummary, statusURL string, cmd *exec.Cmd) error {
	if !isInteractiveFile(os.Stdin) || !isInteractiveFile(os.Stdout) {
		return errors.New("dev -tui requires an interactive terminal with TUI support; run `go run ./tools/gwc dev` without -tui from non-interactive shells")
	}

	tail := &tailBuffer{limit: 16 * 1024}
	cmd.Stdout = io.Discard
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		return err
	}

	model := devStatusModel{
		plan:        plan,
		statusURL:   statusURL,
		process:     cmd,
		processTail: tail,
	}

	finalModel, err := devStatusProgramRunner(model)
	if err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return err
	}

	final, _ := finalModel.(devStatusModel)
	if final.userQuit {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil
	}
	if final.exitedErr != nil {
		tailText := strings.TrimSpace(final.processTail.String())
		if tailText != "" {
			return fmt.Errorf("gwc dev exited: %w\n\n%s", final.exitedErr, tailText)
		}
		return fmt.Errorf("gwc dev exited: %w", final.exitedErr)
	}
	return nil
}

func (m devStatusModel) Init() tea.Cmd {
	return tea.Batch(m.pollStatusCmd(), m.waitProcessCmd())
}

func (m devStatusModel) pollStatusCmd() tea.Cmd {
	statusURL := m.statusURL
	return func() tea.Msg {
		client := &http.Client{Timeout: 1200 * time.Millisecond}
		resp, err := client.Get(statusURL)
		if err != nil {
			return devStatusMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return devStatusMsg{err: fmt.Errorf("status endpoint returned %s", resp.Status)}
		}
		var payload devStatusPayload
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return devStatusMsg{err: err}
		}
		return devStatusMsg{payload: payload}
	}
}

func (m devStatusModel) waitProcessCmd() tea.Cmd {
	process := m.process
	return func() tea.Msg {
		err := process.Wait()
		return devProcessExitMsg{err: err}
	}
}

func devStatusTick() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg { return struct{}{} })
}

func (m devStatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "q", "ctrl+c":
			m.userQuit = true
			return m, tea.Quit
		}
	case devStatusMsg:
		if typed.err != nil {
			m.lastErr = typed.err
		} else {
			m.status = &typed.payload
			m.lastErr = nil
		}
		return m, devStatusTick()
	case devProcessExitMsg:
		m.exitedErr = typed.err
		return m, tea.Quit
	case struct{}:
		return m, m.pollStatusCmd()
	}
	return m, nil
}

func (m devStatusModel) View() string {
	var b strings.Builder
	b.WriteString("GWC Dev Status\n\n")
	b.WriteString(fmt.Sprintf("Project root: %s\n", m.plan.ProjectRoot))
	b.WriteString(fmt.Sprintf("App mode:     %s\n", m.plan.AppMode))
	b.WriteString(fmt.Sprintf("Server mode:  %s\n", m.plan.ServerMode))
	b.WriteString(fmt.Sprintf("URL:          %s\n", m.plan.ListeningURL))
	if strings.TrimSpace(m.statusURL) != "" {
		b.WriteString(fmt.Sprintf("Status URL:   %s\n", m.statusURL))
	}
	b.WriteString("\n")

	if m.status != nil {
		lastBuild := m.status.LastBuild
		if lastBuild != nil {
			b.WriteString(fmt.Sprintf("Phase:        %s\n", devStatusPhaseLabel(lastBuild.Phase)))
			if strings.TrimSpace(lastBuild.PhaseSummary) != "" {
				b.WriteString(fmt.Sprintf("Phase note:   %s\n", lastBuild.PhaseSummary))
			}
			b.WriteString(fmt.Sprintf("Serving:      %s\n", devStatusServingLabel(lastBuild.StaleOutput)))
			if strings.TrimSpace(lastBuild.ReloadType) != "" {
				b.WriteString(fmt.Sprintf("Reload mode:  %s\n", lastBuild.ReloadType))
			}
			if strings.TrimSpace(lastBuild.Duration) != "" {
				b.WriteString(fmt.Sprintf("Last build:   %s\n", lastBuild.Duration))
			}
		}
		b.WriteString(fmt.Sprintf("Hot reload:   enabled=%t eligible=%t\n", m.status.HotReloadEnabled, m.status.HotReloadEligible))
		b.WriteString(fmt.Sprintf("Clients:      %d\n", m.status.ClientCount))
		if strings.TrimSpace(m.status.ServedWASMPath) != "" {
			b.WriteString(fmt.Sprintf("WASM path:    %s\n", m.status.ServedWASMPath))
		}
		if strings.TrimSpace(m.status.LastClassification.Reason) != "" {
			b.WriteString(fmt.Sprintf("Reason:       %s\n", m.status.LastClassification.Reason))
		}
		if m.status.CurrentError != nil && strings.TrimSpace(m.status.CurrentError.Error) != "" {
			b.WriteString("\nCurrent error:\n")
			b.WriteString(m.status.CurrentError.Error)
			b.WriteString("\n")
		}
	} else if m.lastErr != nil {
		b.WriteString(fmt.Sprintf("Status:       waiting for %s\n", m.statusURL))
		b.WriteString(fmt.Sprintf("Detail:       %v\n", m.lastErr))
	} else {
		b.WriteString("Status:       starting dev server...\n")
	}

	hint := devStatusRecoveryHint(m.status, m.lastErr)
	if strings.TrimSpace(hint) != "" {
		b.WriteString("\nHint:\n")
		b.WriteString(hint)
		b.WriteString("\n")
	}

	b.WriteString("\nPress q to stop the TUI and terminate gwc dev.\n")
	return b.String()
}

func devStatusPhaseLabel(phase string) string {
	switch strings.TrimSpace(phase) {
	case "checking_current_state":
		return "checking current state"
	case "compiling":
		return "compiling"
	case "waiting_for_reload":
		return "waiting for reload"
	case "serving_output":
		return "serving output"
	case "blocked_on_error":
		return "blocked on error"
	case "waiting_for_changes":
		return "waiting for more file changes"
	default:
		if strings.TrimSpace(phase) == "" {
			return "unknown"
		}
		return strings.ReplaceAll(strings.TrimSpace(phase), "_", " ")
	}
}

func devStatusServingLabel(stale bool) string {
	if stale {
		return "stale output"
	}
	return "fresh output"
}

func devStatusRecoveryHint(status *devStatusPayload, lastErr error) string {
	if lastErr != nil {
		return "Wait for the livereload status endpoint to come up, or rerun `gwc dev` without -tui if startup keeps failing."
	}
	if status == nil || status.LastBuild == nil {
		return ""
	}
	switch status.LastBuild.Phase {
	case "blocked_on_error":
		return "Fix the current compile error; the dev server is still serving the last successful output."
	case "compiling":
		return "The dev loop is rebuilding now. If this takes too long, inspect watcher churn or repeated file writes."
	case "waiting_for_reload":
		return "The artifact is built. Refresh the browser or inspect the live-reload connection if the page does not update."
	default:
		return ""
	}
}
