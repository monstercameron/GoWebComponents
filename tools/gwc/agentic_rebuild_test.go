package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteRebuildUsesHubJSONEndpoints(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseApp := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(parseApp, []byte("package main\n\nfunc main() {}\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write app: %v", parseErr)
	}

	parseOriginalBuild := buildExecuteBuild
	parseT.Cleanup(func() { buildExecuteBuild = parseOriginalBuild })
	buildExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: parseConfig.profile},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			OutputPath:  parseConfig.outputPath,
			SHA256:      "sha123",
		}, nil
	}

	parseSeen := map[string]int{}
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Query().Get("token") != "tok" {
			http.Error(parseW, "bad token", http.StatusForbidden)
			return
		}
		switch parseR.URL.Path {
		case "/__gwc-agent/sessions":
			parseSeen["sessions"]++
			_, _ = parseW.Write([]byte(`{"sessions":[{"id":"sess-1","state":"closed"},{"id":"sess-2","state":"active"}]}`))
		case "/__gwc-agent/command":
			var parseBody map[string]any
			if parseErr := json.NewDecoder(parseR.Body).Decode(&parseBody); parseErr != nil {
				parseT.Errorf("decode command: %v", parseErr)
				http.Error(parseW, "bad json", http.StatusBadRequest)
				return
			}
			switch parseBody["name"] {
			case "bridge.snapshot":
				parseSeen["snapshot"]++
				if parseBody["session"] != "sess-2" {
					parseT.Errorf("snapshot session = %#v", parseBody["session"])
				}
				_, _ = parseW.Write([]byte(`{"snapshot":{"state":"kept"}}`))
			case "bridge.apply-snapshot":
				parseSeen["apply"]++
				if parseBody["session"] != "sess-3" {
					parseT.Errorf("apply session = %#v", parseBody["session"])
				}
				_, _ = parseW.Write([]byte(`{"ok":true}`))
			default:
				parseT.Errorf("unexpected command %#v", parseBody["name"])
				http.Error(parseW, "bad command", http.StatusBadRequest)
			}
		case "/__gwc-agent/reload":
			parseSeen["reload"]++
			var parseBody map[string]any
			if parseErr := json.NewDecoder(parseR.Body).Decode(&parseBody); parseErr != nil {
				parseT.Errorf("decode reload: %v", parseErr)
				http.Error(parseW, "bad json", http.StatusBadRequest)
				return
			}
			if parseBody["buildId"] != "sha123" || parseBody["session"] != "sess-2" {
				parseT.Errorf("reload body = %#v", parseBody)
			}
			_, _ = parseW.Write([]byte(`{"ok":true}`))
		case "/__gwc-agent/successor":
			parseSeen["successor"]++
			if parseR.URL.Query().Get("session") != "sess-2" || parseR.URL.Query().Get("buildId") != "sha123" {
				parseT.Errorf("successor query = %s", parseR.URL.RawQuery)
			}
			_, _ = parseW.Write([]byte(`{"sessionId":"sess-3"}`))
		default:
			http.NotFound(parseW, parseR)
		}
	}))
	defer parseServer.Close()

	parseReport, parseErr := executeRebuild(rebuildConfig{
		appPath:   parseApp,
		rootPath:  parseRoot,
		output:    filepath.Join(parseRoot, "dist", "app.wasm"),
		hub:       parseServer.URL,
		token:     "tok",
		timeoutMs: 1000,
	})
	if parseErr != nil {
		parseT.Fatalf("execute rebuild: %v", parseErr)
	}
	if !parseReport.OK || parseReport.OldSession != "sess-2" || parseReport.SuccessorSession != "sess-3" || !parseReport.StateRestored {
		parseT.Fatalf("unexpected rebuild report: %#v", parseReport)
	}
	for _, parseKey := range []string{"sessions", "snapshot", "reload", "successor", "apply"} {
		if parseSeen[parseKey] != 1 {
			parseT.Fatalf("expected %s endpoint once, got counts %#v", parseKey, parseSeen)
		}
	}
}

func TestExecuteRebuildBuildFailureDoesNotReload(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseApp := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(parseApp, []byte("package main\n\nfunc main() {}\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write app: %v", parseErr)
	}

	parseOriginalBuild := buildExecuteBuild
	parseT.Cleanup(func() { buildExecuteBuild = parseOriginalBuild })
	buildExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		return buildSummary{}, errors.New("compiler says no")
	}

	parseReloads := 0
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/__gwc-agent/command":
			_, _ = parseW.Write([]byte(`{"snapshot":{"state":"kept"}}`))
		case "/__gwc-agent/reload":
			parseReloads++
			_, _ = parseW.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(parseW, parseR)
		}
	}))
	defer parseServer.Close()

	parseReport, parseErr := executeRebuild(rebuildConfig{
		appPath:   parseApp,
		rootPath:  parseRoot,
		output:    filepath.Join(parseRoot, "dist", "app.wasm"),
		session:   "sess-1",
		hub:       parseServer.URL,
		token:     "tok",
		timeoutMs: 1000,
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "compiler says no") {
		parseT.Fatalf("expected compiler error, got report=%#v err=%v", parseReport, parseErr)
	}
	if parseReport.Phase != "build" || !strings.Contains(parseReport.CompilerOutput, "compiler says no") {
		parseT.Fatalf("expected build phase diagnostics, got %#v", parseReport)
	}
	if parseReloads != 0 {
		parseT.Fatalf("expected no reload after failed build, got %d", parseReloads)
	}
}
