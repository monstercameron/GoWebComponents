//go:build playwrightgo

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	playwright "github.com/playwright-community/playwright-go"
)

var liveReloadClientChromiumOnce sync.Once
var liveReloadClientChromiumErr error

type liveReloadClientWSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func ensureLiveReloadClientChromium(parseT *testing.T) {
	parseT.Helper()
	liveReloadClientChromiumOnce.Do(func() {
		liveReloadClientChromiumErr = releasePlaywrightInstall(&playwright.RunOptions{
			Browsers: []string{"chromium"},
			Verbose:  false,
		})
	})
	if liveReloadClientChromiumErr != nil {
		parseT.Fatalf("install playwright chromium: %v", liveReloadClientChromiumErr)
	}
}

func withLiveReloadClientPage(parseT *testing.T, parseFn func(playwright.Page)) {
	parseT.Helper()
	ensureLiveReloadClientChromium(parseT)
	parsePw, parseErr := releasePlaywrightRun(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr != nil {
		parseT.Fatalf("run playwright: %v", parseErr)
	}
	defer func() {
		if parseErr2 := parsePw.Stop(); parseErr2 != nil {
			parseT.Errorf("stop playwright: %v", parseErr2)
		}
	}()

	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() {
		if parseErr2 := parseBrowser.Close(); parseErr2 != nil {
			parseT.Errorf("close chromium: %v", parseErr2)
		}
	}()

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new browser page: %v", parseErr)
	}
	defer func() {
		if parseErr2 := parsePage.Close(); parseErr2 != nil {
			parseT.Errorf("close page: %v", parseErr2)
		}
	}()

	parseFn(parsePage)
}

func TestLiveReloadClientSurvivesSeveralHotReloads(parseT *testing.T) {
	parseClientScript, parseErr := os.ReadFile(filepath.Join("..", "livereload", "scripts", "livereload-client.txt"))
	if parseErr != nil {
		parseT.Fatalf("read livereload client script: %v", parseErr)
	}

	parseConnected := make(chan *websocket.Conn, 1)
	parseMessages := make(chan liveReloadClientWSMessage, 32)
	parseUpgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var parseConnMu sync.Mutex
	var parseConn *websocket.Conn

	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/ws":
			parseWSConn, parseUpgradeErr := parseUpgrader.Upgrade(parseW, parseR, nil)
			if parseUpgradeErr != nil {
				parseT.Errorf("upgrade websocket: %v", parseUpgradeErr)
				return
			}
			parseConnMu.Lock()
			parseConn = parseWSConn
			parseConnMu.Unlock()
			select {
			case parseConnected <- parseWSConn:
			default:
			}
			for {
				var parseMsg liveReloadClientWSMessage
				if parseReadErr := parseWSConn.ReadJSON(&parseMsg); parseReadErr != nil {
					return
				}
				parseMessages <- parseMsg
			}
		case "/wasm_exec.js":
			parseW.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			_, _ = parseW.Write([]byte("window.__wasmExecLoads = (window.__wasmExecLoads || 0) + 1;\n"))
		case "/app.wasm":
			parseW.Header().Set("Content-Type", "application/wasm")
			_, _ = parseW.Write([]byte("\x00asm"))
		default:
			parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = parseW.Write([]byte(liveReloadClientFixtureHTML(string(parseClientScript))))
		}
	}))
	defer parseServer.Close()
	parseT.Cleanup(func() {
		parseConnMu.Lock()
		defer parseConnMu.Unlock()
		if parseConn != nil {
			_ = parseConn.Close()
		}
	})

	withLiveReloadClientPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr2 := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr2 != nil {
			parseT.Fatalf("goto live reload fixture: %v", parseErr2)
		}
		if _, parseErr2 := parsePage.WaitForFunction(`() => window.GoLiveReload && window.__hotReloadFixtureReady === true`, nil); parseErr2 != nil {
			parseT.Fatalf("wait for live reload fixture: %v", parseErr2)
		}

		parseWSConn := waitForLiveReloadClientConnection(parseT, parseConnected)
		for parseCycle := 1; parseCycle <= 3; parseCycle++ {
			parseExpectedCount := parseCycle * 11
			if _, parseErr2 := parsePage.Evaluate(fmt.Sprintf(`() => {
				window.__hotReloadState.count = %d;
				window.__hotReloadState.label = "cycle-%d";
				window.__renderHotReloadFixture();
			}`, parseExpectedCount, parseCycle)); parseErr2 != nil {
				parseT.Fatalf("seed browser state for cycle %d: %v", parseCycle, parseErr2)
			}

			writeLiveReloadClientMessage(parseT, parseWSConn, "build_start", map[string]any{
				"classification": map[string]any{
					"reloadType": "hot",
					"reason":     fmt.Sprintf("test hot reload cycle %d", parseCycle),
					"plan": map[string]any{
						"summary":       "planned preserve-state hot reload",
						"preserveState": true,
					},
				},
				"status": map[string]any{
					"phase":       "compiling",
					"reloadType":  "hot",
					"staleOutput": true,
				},
			})
			writeLiveReloadClientMessage(parseT, parseWSConn, "state_export", map[string]any{"reason": "hot_reload"})
			parseSnapshot := waitForLiveReloadClientSnapshot(parseT, parseMessages, parseExpectedCount)
			writeLiveReloadClientMessage(parseT, parseWSConn, "build_complete", map[string]any{
				"success":       true,
				"duration":      "1ms",
				"reloadType":    "hot",
				"phase":         "waiting_for_reload",
				"phaseSummary":  "build finished; waiting for the browser to load the fresh artifact",
				"stateSnapshot": parseSnapshot,
				"manifest": map[string]any{
					"reloadType":   "hot",
					"changedFiles": []string{fmt.Sprintf("app_cycle_%d.go", parseCycle)},
					"components": []map[string]any{{
						"name":          "App",
						"qualifiedName": fmt.Sprintf("example.com/app.App%d", parseCycle),
						"packageName":   "main",
						"file":          fmt.Sprintf("app_cycle_%d.go", parseCycle),
					}},
				},
				"timings": map[string]any{"compileMs": 1, "artifactBytes": 4},
			})

			parseWaitExpr := fmt.Sprintf(`() => window.__hotReloadRestoreEvents.length >= %d && window.__hotReloadState.count === %d && window.GoLiveReload.getStoredState() === null`, parseCycle, parseExpectedCount)
			if _, parseErr2 := parsePage.WaitForFunction(parseWaitExpr, nil); parseErr2 != nil {
				parseT.Fatalf("wait for restored state after cycle %d: %v", parseCycle, parseErr2)
			}
		}

		parseRawSummary, parseErr2 := parsePage.Evaluate(`() => ({
			count: window.__hotReloadState.count,
			exports: window.__hotReloadSnapshotExports.map((entry) => entry.count),
			restores: window.__hotReloadRestoreEvents.map((entry) => entry.count),
			outcome: window.GoLiveReload.getLastHotReloadOutcome() && window.GoLiveReload.getLastHotReloadOutcome().outcome,
			wasmRuns: window.__wasmRuns || 0,
			stored: window.GoLiveReload.getStoredState()
		})`)
		if parseErr2 != nil {
			parseT.Fatalf("read hot reload summary: %v", parseErr2)
		}
		parseSummary, parseOk := parseRawSummary.(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected object summary, got %#v", parseRawSummary)
		}
		parseFinalCount, parseOk := liveReloadClientNumber(parseSummary["count"])
		if !parseOk || parseFinalCount != 33 {
			parseT.Fatalf("expected final count 33 after three hot reloads, got %#v", parseSummary)
		}
		parseWasmRuns, parseOk := liveReloadClientNumber(parseSummary["wasmRuns"])
		if !parseOk || parseWasmRuns < 3 {
			parseT.Fatalf("expected at least three fake wasm runs, got %#v", parseSummary)
		}
		assertLiveReloadClientNumericList(parseT, parseSummary["exports"], []float64{11, 22, 33}, "exports")
		assertLiveReloadClientNumericList(parseT, parseSummary["restores"], []float64{11, 22, 33}, "restores")
		if parseSummary["stored"] != nil {
			parseT.Fatalf("expected stored hot reload state to be cleared after final restore, got %#v", parseSummary)
		}
	})
}

func liveReloadClientFixtureHTML(parseClientScript string) string {
	return `<!doctype html>
<html>
<head><title>live reload client fixture</title></head>
<body>
<script src="/wasm_exec.js"></script>
<div id="app"><span id="counter"></span><span id="label"></span></div>
<script>
window.__GWC_LIVERELOAD_CONFIG = { wasmPath: "/app.wasm" };
window.__hotReloadState = { count: 0, label: "initial" };
window.__hotReloadSnapshotExports = [];
window.__hotReloadRestoreEvents = [];
window.__hotReloadFixtureReady = false;
window.__lastHotReloadRestoreResult = null;
window.__renderHotReloadFixture = function() {
	const counter = document.getElementById("counter");
	const label = document.getElementById("label");
	if (counter) counter.textContent = String(window.__hotReloadState.count);
	if (label) label.textContent = window.__hotReloadState.label;
};
window.GoWebComponentsHotReloadApp = {
	captureSnapshot: function() {
		const snapshot = {
			count: window.__hotReloadState.count,
			label: window.__hotReloadState.label
		};
		window.__hotReloadSnapshotExports.push(snapshot);
		return JSON.stringify(snapshot);
	},
	restoreSnapshot: function(payload) {
		const parsed = JSON.parse(payload);
		window.__hotReloadState = {
			count: parsed.count,
			label: parsed.label
		};
		window.__hotReloadRestoreEvents.push({
			count: parsed.count,
			label: parsed.label,
			restoreMode: parsed.restoreMode || "",
			changedComponents: Array.isArray(parsed.changedComponents) ? parsed.changedComponents.slice() : []
		});
		window.__lastHotReloadRestoreResult = {
			outcome: parsed.restoreMode === "selective" ? "restored-selective" : "restored",
			message: "fixture restored"
		};
		window.__renderHotReloadFixture();
		return window.__lastHotReloadRestoreResult;
	},
	getLastRestoreResult: function() {
		return window.__lastHotReloadRestoreResult;
	},
	getHotReloadDiagnostics: function() {
		return [];
	},
	getHotReloadActivity: function() {
		return window.__hotReloadRestoreEvents.map(function(entry, index) {
			return {
				domain: "hotreload",
				level: "info",
				classification: "test",
				message: "restore " + (index + 1),
				fields: { count: String(entry.count) }
			};
		});
	},
	prepare: function() {
		window.__hotReloadPrepared = (window.__hotReloadPrepared || 0) + 1;
	}
};
window.hotReloadWasm = function() {
	window.__hotReloadPatchCalls = (window.__hotReloadPatchCalls || 0) + 1;
};
window.Go = function() {
	this.importObject = {};
	this.run = function() {
		window.__wasmRuns = (window.__wasmRuns || 0) + 1;
		if (window.GoLiveReload) {
			const stored = window.GoLiveReload.getStoredState();
			if (stored) {
				window.GoLiveReload.importState(stored);
				window.GoLiveReload.clearStoredState();
			}
		}
	};
};
WebAssembly.instantiateStreaming = function() {
	return Promise.resolve({ instance: {} });
};
document.addEventListener("DOMContentLoaded", function() {
	window.__renderHotReloadFixture();
	window.__hotReloadFixtureReady = true;
});
</script>
<script>
` + parseClientScript + `
</script>
</body>
</html>`
}

func waitForLiveReloadClientConnection(parseT *testing.T, parseConnected <-chan *websocket.Conn) *websocket.Conn {
	parseT.Helper()
	select {
	case parseConn := <-parseConnected:
		return parseConn
	case <-time.After(10 * time.Second):
		parseT.Fatal("timed out waiting for browser live reload websocket connection")
		return nil
	}
}

func writeLiveReloadClientMessage(parseT *testing.T, parseConn *websocket.Conn, parseMsgType string, parsePayload any) {
	parseT.Helper()
	parseErr := parseConn.WriteJSON(map[string]any{
		"protocol":  "gwc.livereload.ws",
		"version":   1,
		"type":      parseMsgType,
		"payload":   parsePayload,
		"timestamp": time.Now(),
	})
	if parseErr != nil {
		parseT.Fatalf("write %s websocket message: %v", parseMsgType, parseErr)
	}
}

func waitForLiveReloadClientSnapshot(parseT *testing.T, parseMessages <-chan liveReloadClientWSMessage, parseExpectedCount int) string {
	parseT.Helper()
	parseDeadline := time.After(10 * time.Second)
	for {
		select {
		case parseMsg, parseOk := <-parseMessages:
			if !parseOk {
				parseT.Fatal("browser websocket message stream closed while waiting for state snapshot")
			}
			if parseMsg.Type != "state_snapshot" {
				continue
			}
			var parsePayload string
			if parseErr := json.Unmarshal(parseMsg.Payload, &parsePayload); parseErr != nil {
				parseT.Fatalf("decode state_snapshot payload: %v; raw=%s", parseErr, string(parseMsg.Payload))
			}
			var parseState struct {
				Count int `json:"count"`
			}
			if parseErr := json.Unmarshal([]byte(parsePayload), &parseState); parseErr != nil {
				parseT.Fatalf("decode snapshot state payload: %v; raw=%s", parseErr, parsePayload)
			}
			if parseState.Count != parseExpectedCount {
				parseT.Fatalf("expected state snapshot count %d, got %d in %s", parseExpectedCount, parseState.Count, parsePayload)
			}
			return parsePayload
		case <-parseDeadline:
			parseT.Fatalf("timed out waiting for state snapshot count %d", parseExpectedCount)
			return ""
		}
	}
}

func assertLiveReloadClientNumericList(parseT *testing.T, parseRaw any, parseExpected []float64, parseLabel string) {
	parseT.Helper()
	parseItems, parseOk := parseRaw.([]any)
	if !parseOk || len(parseItems) != len(parseExpected) {
		parseT.Fatalf("expected %s list %#v, got %#v", parseLabel, parseExpected, parseRaw)
	}
	for parseIndex, parseExpectedValue := range parseExpected {
		parseGot, parseOk2 := liveReloadClientNumber(parseItems[parseIndex])
		if !parseOk2 || parseGot != parseExpectedValue {
			parseT.Fatalf("expected %s[%d] = %.0f, got %#v", parseLabel, parseIndex, parseExpectedValue, parseItems[parseIndex])
		}
	}
}

func liveReloadClientNumber(parseRaw any) (float64, bool) {
	switch parseValue := parseRaw.(type) {
	case float64:
		return parseValue, true
	case int:
		return float64(parseValue), true
	default:
		return 0, false
	}
}
