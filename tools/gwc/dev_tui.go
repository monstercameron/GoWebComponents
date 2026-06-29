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

func (parseB *tailBuffer) Write(parseP []byte) (int, error) {
	if parseB == nil {
		return len(parseP), nil
	}
	parseB.data = append(parseB.data, parseP...)
	if parseB.limit > 0 && len(parseB.data) > parseB.limit {
		parseB.data = append([]byte(nil), parseB.data[len(parseB.data)-parseB.limit:]...)
	}
	return len(parseP), nil
}

func (parseB *tailBuffer) String() string {
	if parseB == nil || len(parseB.data) == 0 {
		return ""
	}
	return string(bytes.TrimSpace(parseB.data))
}

var devStatusProgramRunner = func(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(model, tea.WithAltScreen()).Run()
}

func runDevStatusTUI(parsePlan devPlanSummary, parseStatusURL string, parseCmd *exec.Cmd) error {
	if !isInteractiveFile(os.Stdin) || !isInteractiveFile(os.Stdout) {
		return errors.New("dev -tui requires an interactive terminal with TUI support; run `go run ./tools/gwc dev` without -tui from non-interactive shells")
	}

	parseTail := &tailBuffer{limit: 16 * 1024}
	parseCmd.Stdout = io.Discard
	parseCmd.Stderr = parseTail
	if parseErr := parseCmd.Start(); parseErr != nil {
		return parseErr
	}

	parseModel := devStatusModel{
		plan:        parsePlan,
		statusURL:   parseStatusURL,
		process:     parseCmd,
		processTail: parseTail,
	}

	parseFinalModel, parseErr2 := devStatusProgramRunner(parseModel)
	if parseErr2 != nil {
		terminateLauncherProcessTree(parseCmd)
		return parseErr2
	}

	parseFinal, _ := parseFinalModel.(devStatusModel)
	if parseFinal.userQuit {
		terminateLauncherProcessTree(parseCmd)
		return nil
	}
	if parseFinal.exitedErr != nil {
		parseTailText := strings.TrimSpace(parseFinal.processTail.String())
		if parseTailText != "" {
			return fmt.Errorf("gwc dev exited: %w\n\n%s", parseFinal.exitedErr, parseTailText)
		}
		return fmt.Errorf("gwc dev exited: %w", parseFinal.exitedErr)
	}
	return nil
}

func (parseM devStatusModel) Init() tea.Cmd {
	return tea.Batch(parseM.pollStatusCmd(), parseM.waitProcessCmd())
}

func (parseM devStatusModel) pollStatusCmd() tea.Cmd {
	parseStatusURL := parseM.statusURL
	return func() tea.Msg {
		parseClient := &http.Client{Timeout: 1200 * time.Millisecond}
		parseResp, parseErr := parseClient.Get(parseStatusURL)
		if parseErr != nil {
			return devStatusMsg{err: parseErr}
		}
		defer parseResp.Body.Close()
		if parseResp.StatusCode != http.StatusOK {
			return devStatusMsg{err: fmt.Errorf("status endpoint returned %s", parseResp.Status)}
		}
		var parsePayload devStatusPayload
		if parseErr2 := json.NewDecoder(parseResp.Body).Decode(&parsePayload); parseErr2 != nil {
			return devStatusMsg{err: parseErr2}
		}
		return devStatusMsg{payload: parsePayload}
	}
}

func (parseM devStatusModel) waitProcessCmd() tea.Cmd {
	parseProcess := parseM.process
	return func() tea.Msg {
		parseErr := parseProcess.Wait()
		return devProcessExitMsg{err: parseErr}
	}
}

func devStatusTick() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg { return struct{}{} })
}

func (parseM devStatusModel) Update(parseMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch parseTyped := parseMsg.(type) {
	case tea.KeyMsg:
		switch parseTyped.String() {
		case "q", "ctrl+c":
			parseM.userQuit = true
			return parseM, tea.Quit
		}
	case devStatusMsg:
		if parseTyped.err != nil {
			parseM.lastErr = parseTyped.err
		} else {
			parseM.status = &parseTyped.payload
			parseM.lastErr = nil
		}
		return parseM, devStatusTick()
	case devProcessExitMsg:
		parseM.exitedErr = parseTyped.err
		return parseM, tea.Quit
	case struct{}:
		return parseM, parseM.pollStatusCmd()
	}
	return parseM, nil
}

func (parseM devStatusModel) View() string {
	var parseB strings.Builder
	parseB.WriteString("GWC Dev Status\n\n")
	parseB.WriteString(fmt.Sprintf("Project root: %s\n", parseM.plan.ProjectRoot))
	parseB.WriteString(fmt.Sprintf("App mode:     %s\n", parseM.plan.AppMode))
	parseB.WriteString(fmt.Sprintf("Server mode:  %s\n", parseM.plan.ServerMode))
	parseB.WriteString(fmt.Sprintf("URL:          %s\n", parseM.plan.ListeningURL))
	if strings.TrimSpace(parseM.statusURL) != "" {
		parseB.WriteString(fmt.Sprintf("Status URL:   %s\n", parseM.statusURL))
	}
	parseB.WriteString("\n")

	if parseM.status != nil {
		parseLastBuild := parseM.status.LastBuild
		if parseLastBuild != nil {
			parseB.WriteString(fmt.Sprintf("Phase:        %s\n", devStatusPhaseLabel(parseLastBuild.Phase)))
			if strings.TrimSpace(parseLastBuild.PhaseSummary) != "" {
				parseB.WriteString(fmt.Sprintf("Phase note:   %s\n", parseLastBuild.PhaseSummary))
			}
			parseB.WriteString(fmt.Sprintf("Serving:      %s\n", devStatusServingLabel(parseLastBuild.StaleOutput)))
			if strings.TrimSpace(parseLastBuild.ReloadType) != "" {
				parseB.WriteString(fmt.Sprintf("Reload mode:  %s\n", parseLastBuild.ReloadType))
			}
			if strings.TrimSpace(parseLastBuild.Duration) != "" {
				parseB.WriteString(fmt.Sprintf("Last build:   %s\n", parseLastBuild.Duration))
			}
		}
		parseB.WriteString(fmt.Sprintf("Hot reload:   enabled=%t eligible=%t\n", parseM.status.HotReloadEnabled, parseM.status.HotReloadEligible))
		parseB.WriteString(fmt.Sprintf("Clients:      %d\n", parseM.status.ClientCount))
		if strings.TrimSpace(parseM.status.ServedWASMPath) != "" {
			parseB.WriteString(fmt.Sprintf("WASM path:    %s\n", parseM.status.ServedWASMPath))
		}
		if strings.TrimSpace(parseM.status.LastClassification.Reason) != "" {
			parseB.WriteString(fmt.Sprintf("Reason:       %s\n", parseM.status.LastClassification.Reason))
		}
		if parseM.status.CurrentError != nil && strings.TrimSpace(parseM.status.CurrentError.Error) != "" {
			parseB.WriteString("\nCurrent error:\n")
			parseB.WriteString(parseM.status.CurrentError.Error)
			parseB.WriteString("\n")
		}
	} else if parseM.lastErr != nil {
		parseB.WriteString(fmt.Sprintf("Status:       waiting for %s\n", parseM.statusURL))
		parseB.WriteString(fmt.Sprintf("Detail:       %v\n", parseM.lastErr))
	} else {
		parseB.WriteString("Status:       starting dev server...\n")
	}

	parseHint := devStatusRecoveryHint(parseM.status, parseM.lastErr)
	if strings.TrimSpace(parseHint) != "" {
		parseB.WriteString("\nHint:\n")
		parseB.WriteString(parseHint)
		parseB.WriteString("\n")
	}

	parseB.WriteString("\nPress q to stop the TUI and terminate gwc dev.\n")
	return parseB.String()
}

func devStatusPhaseLabel(parsePhase string) string {
	switch strings.TrimSpace(parsePhase) {
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
		if strings.TrimSpace(parsePhase) == "" {
			return "unknown"
		}
		return strings.ReplaceAll(strings.TrimSpace(parsePhase), "_", " ")
	}
}

func devStatusServingLabel(isStale bool) string {
	if isStale {
		return "stale output"
	}
	return "fresh output"
}

func devStatusRecoveryHint(parseStatus *devStatusPayload, parseLastErr error) string {
	if parseLastErr != nil {
		return "Wait for the livereload status endpoint to come up, or rerun `gwc dev` without -tui if startup keeps failing."
	}
	if parseStatus == nil || parseStatus.LastBuild == nil {
		return ""
	}
	switch parseStatus.LastBuild.Phase {
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
