package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScanDashboardProviders(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseT.Setenv("OPENAI_API_KEY", "")
	parseT.Setenv("OPENAI_MODEL", "")
	parseT.Setenv("ANTHROPIC_API_KEY", "")
	parseT.Setenv("CEREBRAS_API_KEY", "process-cerebras")
	if parseErr := os.WriteFile(filepath.Join(parseRoot, ".env.local"), []byte("OPENAI_API_KEY=file-openai\nOPENAI_MODEL=gpt-5.4\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write .env.local: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, ".env"), []byte("ANTHROPIC_API_KEY=file-anthropic\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write .env: %v", parseErr2)
	}

	parseProviders := scanDashboardProviders(parseRoot)

	parseOpenAI := findDashboardProvider(parseProviders, "openai")
	if parseOpenAI.ID != "openai" || !parseOpenAI.AuthConfigured || parseOpenAI.AuthSource != ".env.local" || parseOpenAI.DefaultModel != "gpt-5.4" || parseOpenAI.ModelSource != ".env.local" {
		parseT.Fatalf("unexpected OpenAI provider state: %+v", parseOpenAI)
	}

	parseAnthropic := findDashboardProvider(parseProviders, "anthropic")
	if !parseAnthropic.AuthConfigured || parseAnthropic.AuthSource != ".env" {
		parseT.Fatalf("unexpected Anthropic provider state: %+v", parseAnthropic)
	}

	parseCerebras := findDashboardProvider(parseProviders, "cerebras")
	if !parseCerebras.AuthConfigured || parseCerebras.AuthSource != "process environment" {
		parseT.Fatalf("unexpected Cerebras provider state: %+v", parseCerebras)
	}

	parseGroq := findDashboardProvider(parseProviders, "groq")
	if parseGroq.AuthConfigured || parseGroq.APIKeyEnv != "GROQ_API_KEY" {
		parseT.Fatalf("unexpected Groq provider state: %+v", parseGroq)
	}
}

func TestCollectDashboardSnapshotAndHelpers(parseT *testing.T) {
	parseStatusServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(parseW).Encode(dashboardStatusPayload{
			ListeningURL:      "http://127.0.0.1:8090",
			HotReloadEnabled:  true,
			HotReloadEligible: true,
			ClientCount:       1,
			Clients: []dashboardClientSession{
				{ID: "client-1", RemoteAddr: "127.0.0.1:5000", UserAgent: "dashboard-test", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()},
			},
		})
	}))
	defer parseStatusServer.Close()

	parseSnapshot := collectDashboardSnapshot(parseT.TempDir(), parseStatusServer.URL)
	if parseSnapshot.Status == nil || parseSnapshot.Status.ClientCount != 1 || len(parseSnapshot.Status.Clients) != 1 {
		parseT.Fatalf("unexpected snapshot: %+v", parseSnapshot)
	}
	if parseSnapshot.StatusError != "" {
		parseT.Fatalf("expected empty status error, got %q", parseSnapshot.StatusError)
	}
	if parseGot := dashboardDisconnectURL("http://127.0.0.1:8090/__gwc/status"); parseGot != "http://127.0.0.1:8090/__gwc/clients/disconnect" {
		parseT.Fatalf("unexpected disconnect URL: %q", parseGot)
	}
	if parseGot2 := dashboardTruncate("abcdefghijklmnopqrstuvwxyz", 10); parseGot2 != "abcdefg..." {
		parseT.Fatalf("unexpected truncate result: %q", parseGot2)
	}
}

func TestDashboardModelView(parseT *testing.T) {
	parseModel := dashboardModel{
		projectRoot:   "/repo",
		statusURL:     "http://127.0.0.1:8090/__gwc/status",
		disconnectURL: "http://127.0.0.1:8090/__gwc/clients/disconnect",
		lastAction:    "Disconnected 1 client(s); 0 remaining.",
		snapshot: &dashboardSnapshot{
			GeneratedAt: "2026-03-25T10:00:00Z",
			Providers: []dashboardProviderStatus{
				{ID: "openai", Label: "OpenAI", APIKeyEnv: "OPENAI_API_KEY", AuthConfigured: true, AuthSource: ".env.local", DefaultModel: "gpt-5.4-mini"},
				{ID: "anthropic", Label: "Anthropic", APIKeyEnv: "ANTHROPIC_API_KEY", AuthConfigured: false, DefaultModel: "claude-sonnet-4-5"},
			},
			Status: &dashboardStatusPayload{
				ListeningURL:      "http://127.0.0.1:8090",
				HotReloadEnabled:  true,
				HotReloadEligible: true,
				ClientCount:       1,
				Clients: []dashboardClientSession{
					{ID: "client-1", RemoteAddr: "127.0.0.1:5500", UserAgent: "Mozilla/5.0", ConnectedAt: time.Now(), LastSeenAt: time.Now()},
				},
			},
		},
	}

	parseView := parseModel.View()
	for _, parseWant := range []string{
		"GWC Dashboard",
		"client-1",
		"OpenAI (configured)",
		"needs ANTHROPIC_API_KEY",
		"Disconnected 1 client(s); 0 remaining.",
	} {
		if !strings.Contains(parseView, parseWant) {
			parseT.Fatalf("expected dashboard view to contain %q, got %q", parseWant, parseView)
		}
	}
}

func findDashboardProvider(parseProviders []dashboardProviderStatus, parseId string) dashboardProviderStatus {
	for _, parseProvider := range parseProviders {
		if parseProvider.ID == parseId {
			return parseProvider
		}
	}
	return dashboardProviderStatus{}
}
