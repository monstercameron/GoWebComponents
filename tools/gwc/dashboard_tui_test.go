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

func TestScanDashboardProviders(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CEREBRAS_API_KEY", "process-cerebras")
	if err := os.WriteFile(filepath.Join(root, ".env.local"), []byte("OPENAI_API_KEY=file-openai\nOPENAI_MODEL=gpt-5.4\n"), 0o644); err != nil {
		t.Fatalf("write .env.local: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("ANTHROPIC_API_KEY=file-anthropic\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	providers := scanDashboardProviders(root)

	openAI := findDashboardProvider(providers, "openai")
	if openAI.ID != "openai" || !openAI.AuthConfigured || openAI.AuthSource != ".env.local" || openAI.DefaultModel != "gpt-5.4" || openAI.ModelSource != ".env.local" {
		t.Fatalf("unexpected OpenAI provider state: %+v", openAI)
	}

	anthropic := findDashboardProvider(providers, "anthropic")
	if !anthropic.AuthConfigured || anthropic.AuthSource != ".env" {
		t.Fatalf("unexpected Anthropic provider state: %+v", anthropic)
	}

	cerebras := findDashboardProvider(providers, "cerebras")
	if !cerebras.AuthConfigured || cerebras.AuthSource != "process environment" {
		t.Fatalf("unexpected Cerebras provider state: %+v", cerebras)
	}

	groq := findDashboardProvider(providers, "groq")
	if groq.AuthConfigured || groq.APIKeyEnv != "GROQ_API_KEY" {
		t.Fatalf("unexpected Groq provider state: %+v", groq)
	}
}

func TestCollectDashboardSnapshotAndHelpers(t *testing.T) {
	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(dashboardStatusPayload{
			ListeningURL:      "http://127.0.0.1:8090",
			HotReloadEnabled:  true,
			HotReloadEligible: true,
			ClientCount:       1,
			Clients: []dashboardClientSession{
				{ID: "client-1", RemoteAddr: "127.0.0.1:5000", UserAgent: "dashboard-test", ConnectedAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()},
			},
		})
	}))
	defer statusServer.Close()

	snapshot := collectDashboardSnapshot(t.TempDir(), statusServer.URL)
	if snapshot.Status == nil || snapshot.Status.ClientCount != 1 || len(snapshot.Status.Clients) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.StatusError != "" {
		t.Fatalf("expected empty status error, got %q", snapshot.StatusError)
	}
	if got := dashboardDisconnectURL("http://127.0.0.1:8090/__gwc/status"); got != "http://127.0.0.1:8090/__gwc/clients/disconnect" {
		t.Fatalf("unexpected disconnect URL: %q", got)
	}
	if got := dashboardTruncate("abcdefghijklmnopqrstuvwxyz", 10); got != "abcdefg..." {
		t.Fatalf("unexpected truncate result: %q", got)
	}
}

func TestDashboardModelView(t *testing.T) {
	model := dashboardModel{
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

	view := model.View()
	for _, want := range []string{
		"GWC Dashboard",
		"client-1",
		"OpenAI (configured)",
		"needs ANTHROPIC_API_KEY",
		"Disconnected 1 client(s); 0 remaining.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected dashboard view to contain %q, got %q", want, view)
		}
	}
}

func findDashboardProvider(providers []dashboardProviderStatus, id string) dashboardProviderStatus {
	for _, provider := range providers {
		if provider.ID == id {
			return provider
		}
	}
	return dashboardProviderStatus{}
}
