package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestRunDashboardEmitsJSONSnapshot verifies JSON dashboard output and config resolution.
func TestRunDashboardEmitsJSONSnapshot(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseStatusServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(parseW).Encode(dashboardStatusPayload{
			ListeningURL: "http://127.0.0.1:8090",
			ClientCount:  1,
			Clients:      []dashboardClientSession{{ID: "client-1"}},
		})
	}))
	defer parseStatusServer.Close()

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := (launcher{}).runDashboard([]string{"-root", parseRoot, "-status-url", parseStatusServer.URL, "-json"}); parseErr2 != nil {
		parseT.Fatalf("run dashboard json: %v", parseErr2)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	var parseSnapshot dashboardSnapshot
	if parseErr3 := json.Unmarshal([]byte(parseOutput), &parseSnapshot); parseErr3 != nil {
		parseT.Fatalf("decode dashboard snapshot: %v\n%s", parseErr3, parseOutput)
	}
	parseAbsRoot, parseErr := filepath.Abs(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("Abs root: %v", parseErr)
	}
	if parseSnapshot.ProjectRoot != parseAbsRoot {
		parseT.Fatalf("project root = %q, want %q", parseSnapshot.ProjectRoot, parseAbsRoot)
	}
	if parseSnapshot.StatusURL != parseStatusServer.URL {
		parseT.Fatalf("status URL = %q, want %q", parseSnapshot.StatusURL, parseStatusServer.URL)
	}
	if parseSnapshot.Status == nil || parseSnapshot.Status.ClientCount != 1 {
		parseT.Fatalf("unexpected status payload: %#v", parseSnapshot.Status)
	}
}

// TestRunDashboardPrintsTextSnapshot verifies non-interactive dashboard output.
func TestRunDashboardPrintsTextSnapshot(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseStatusServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(parseW).Encode(dashboardStatusPayload{
			ListeningURL: "http://127.0.0.1:8090",
			ClientCount:  2,
		})
	}))
	defer parseStatusServer.Close()

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := (launcher{}).runDashboard([]string{"-root", parseRoot, "-status-url", parseStatusServer.URL}); parseErr2 != nil {
		parseT.Fatalf("run dashboard text: %v", parseErr2)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseWant := range []string{"GWC Dashboard", "project root:", "status url:", "clients:      2"} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected dashboard output to contain %q, got %q", parseWant, parseOutput)
		}
	}
}

// TestRunDashboardTUIUsesProgramRunner verifies the TUI runner receives the expected model.
func TestRunDashboardTUIUsesProgramRunner(parseT *testing.T) {
	parseOriginalRunner := dashboardProgramRunner
	parseT.Cleanup(func() {
		dashboardProgramRunner = parseOriginalRunner
	})

	isParseCalled := false
	dashboardProgramRunner = func(parseModel tea.Model) (tea.Model, error) {
		isParseCalled = true
		parseDashboardModel, parseOk := parseModel.(dashboardModel)
		if !parseOk {
			parseT.Fatalf("expected dashboardModel, got %T", parseModel)
		}
		if parseDashboardModel.projectRoot != "C:/repo" || parseDashboardModel.statusURL != "http://127.0.0.1:8090/__gwc/status" {
			parseT.Fatalf("unexpected model: %#v", parseDashboardModel)
		}
		return parseModel, nil
	}

	parseConfig := dashboardConfig{
		projectRoot:   "C:/repo",
		statusURL:     "http://127.0.0.1:8090/__gwc/status",
		disconnectURL: "http://127.0.0.1:8090/__gwc/clients/disconnect",
	}
	parseInitial := dashboardSnapshot{GeneratedAt: "2026-03-26T00:00:00Z"}
	if parseErr := runDashboardTUI(parseConfig, parseInitial); parseErr != nil {
		parseT.Fatalf("runDashboardTUI: %v", parseErr)
	}
	if !isParseCalled {
		parseT.Fatal("expected dashboard program runner to be called")
	}
}

// TestDashboardModelCommandsAndUpdate verifies model commands, update branches, and disconnect handling.
func TestDashboardModelCommandsAndUpdate(parseT *testing.T) {
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/status", func(parseW http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(parseW).Encode(dashboardStatusPayload{
			ListeningURL: "http://127.0.0.1:8090",
			ClientCount:  1,
			Clients:      []dashboardClientSession{{ID: "client-1"}},
		})
	})
	parseMux.HandleFunc("/disconnect", func(parseW http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(parseW).Encode(dashboardDisconnectResponse{
			Disconnected: 1,
			Remaining:    0,
		})
	})
	parseServer := httptest.NewServer(parseMux)
	defer parseServer.Close()

	parseModel := dashboardModel{
		projectRoot:   parseT.TempDir(),
		statusURL:     parseServer.URL + "/status",
		disconnectURL: parseServer.URL + "/disconnect",
		snapshot: &dashboardSnapshot{
			Status: &dashboardStatusPayload{
				ClientCount: 2,
				Clients: []dashboardClientSession{
					{ID: "client-1", RemoteAddr: "127.0.0.1:5000", ConnectedAt: time.Now(), LastSeenAt: time.Now()},
					{ID: "client-2", RemoteAddr: "127.0.0.1:5001", ConnectedAt: time.Now(), LastSeenAt: time.Now()},
				},
			},
		},
		selectedClient: 1,
	}

	if parseCmd := parseModel.Init(); parseCmd == nil {
		parseT.Fatal("expected Init command")
	}
	parseSnapshotMessage, parseOk := parseModel.pollSnapshotCmd()().(dashboardSnapshotMsg)
	if !parseOk || parseSnapshotMessage.snapshot.Status == nil || parseSnapshotMessage.snapshot.Status.ClientCount != 1 {
		parseT.Fatalf("unexpected snapshot message: %#v", parseSnapshotMessage)
	}
	if parseModel.currentClientID() != "client-2" {
		parseT.Fatalf("currentClientID = %q, want client-2", parseModel.currentClientID())
	}

	parseUpdatedAny, parseCommand := parseModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	parseUpdatedModel, parseOk := parseUpdatedAny.(dashboardModel)
	if !parseOk {
		parseT.Fatalf("expected dashboardModel after x update, got %T", parseUpdatedAny)
	}
	if !strings.Contains(parseUpdatedModel.lastAction, "Disconnecting selected client") {
		parseT.Fatalf("unexpected lastAction after x: %#v", parseUpdatedModel)
	}
	if parseCommand == nil {
		parseT.Fatal("expected disconnect command")
	}
	parseDisconnectMessage, parseOk := parseCommand().(dashboardDisconnectMsg)
	if !parseOk || parseDisconnectMessage.response.Disconnected != 1 {
		parseT.Fatalf("unexpected disconnect message: %#v", parseDisconnectMessage)
	}

	parseUpdatedAny, parseCommand = parseUpdatedModel.Update(parseDisconnectMessage)
	parseUpdatedModel, parseOk = parseUpdatedAny.(dashboardModel)
	if !parseOk {
		parseT.Fatalf("expected dashboardModel after disconnect update, got %T", parseUpdatedAny)
	}
	if !strings.Contains(parseUpdatedModel.lastAction, "Disconnected 1 client(s)") {
		parseT.Fatalf("unexpected lastAction after disconnect: %#v", parseUpdatedModel)
	}
	if parseCommand == nil {
		parseT.Fatal("expected refresh command after disconnect")
	}

	parseUpdatedAny, parseCommand = parseUpdatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	parseUpdatedModel, parseOk = parseUpdatedAny.(dashboardModel)
	if !parseOk || parseCommand == nil {
		parseT.Fatalf("expected refresh command, got model=%T cmd=%v", parseUpdatedAny, parseCommand)
	}

	parseUpdatedAny, _ = parseUpdatedModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	parseUpdatedModel, parseOk = parseUpdatedAny.(dashboardModel)
	if !parseOk || parseUpdatedModel.selectedClient != 1 {
		parseT.Fatalf("expected down key to keep last client selected, got %#v", parseUpdatedModel)
	}
	parseUpdatedAny, _ = parseUpdatedModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	parseUpdatedModel, parseOk = parseUpdatedAny.(dashboardModel)
	if !parseOk || parseUpdatedModel.selectedClient != 0 {
		parseT.Fatalf("expected up key to move selection, got %#v", parseUpdatedModel)
	}

	parseUpdatedAny, parseCommand = parseUpdatedModel.Update(struct{}{})
	parseUpdatedModel, parseOk = parseUpdatedAny.(dashboardModel)
	if !parseOk || parseCommand == nil {
		parseT.Fatalf("expected tick refresh command, got model=%T cmd=%v", parseUpdatedAny, parseCommand)
	}

	parseNoClientAny, _ := (dashboardModel{}).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	parseNoClientModel, parseOk := parseNoClientAny.(dashboardModel)
	if !parseOk || parseNoClientModel.lastAction != "No client selected to disconnect." {
		parseT.Fatalf("unexpected no-client update: %#v", parseNoClientAny)
	}

	parseOutOfRangeAny, _ := (dashboardModel{selectedClient: 5}).Update(dashboardSnapshotMsg{
		snapshot: dashboardSnapshot{Status: &dashboardStatusPayload{Clients: []dashboardClientSession{{ID: "client-1"}}}},
	})
	parseOutOfRangeModel, parseOk := parseOutOfRangeAny.(dashboardModel)
	if !parseOk || parseOutOfRangeModel.selectedClient != 0 {
		parseT.Fatalf("expected snapshot update to clamp selection, got %#v", parseOutOfRangeAny)
	}
}
