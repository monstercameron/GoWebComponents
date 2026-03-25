//go:build js && wasm && !production
// +build js,wasm,!production

package hotreload

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

func TestConfigureInstallsBridgeObjectAndRestoresSnapshot(parseT *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "dark"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}

	parseGlobal, parseErr2 := interop.GetGlobalThis()
	if parseErr2 != nil {
		parseT.Fatalf("expected browser global, got %v", parseErr2)
	}
	parsePrevBridge := parseGlobal.Get(appBridgeGlobal)
	parsePrevLiveReload := parseGlobal.Get(liveReloadGlobal)

	parseT.Cleanup(func() {
		Disable()
		if parsePrevBridge.Present() {
			_ = parseGlobal.Set(appBridgeGlobal, parsePrevBridge)
		} else {
			_ = parseGlobal.Delete(appBridgeGlobal)
		}
		if parsePrevLiveReload.Present() {
			_ = parseGlobal.Set(liveReloadGlobal, parsePrevLiveReload)
		} else {
			_ = parseGlobal.Delete(liveReloadGlobal)
		}
	})

	Configure(Config{})
	if !Enabled() {
		parseT.Fatal("expected hotreload to be enabled")
	}

	parseBridge, parseErr2 := appBridge()
	if parseErr2 != nil {
		parseT.Fatalf("expected installed bridge, got %v", parseErr2)
	}
	parseCaptureFn := parseBridge.Get("captureSnapshot")
	parsePrepareFn := parseBridge.Get("prepare")
	parseRestoreFn := parseBridge.Get("restoreSnapshot")
	parseStatusFn := parseBridge.Get("getLastRestoreResult")
	parseDiagnosticsFn := parseBridge.Get("getHotReloadDiagnostics")
	parseActivityFn := parseBridge.Get("getHotReloadActivity")
	if !parseCaptureFn.Present() || !parsePrepareFn.Present() || !parseRestoreFn.Present() || !parseStatusFn.Present() || !parseDiagnosticsFn.Present() || !parseActivityFn.Present() {
		parseT.Fatal("expected bridge methods to be installed")
	}

	parsePayload, parseErr2 := parseCaptureFn.Invoke()
	if parseErr2 != nil {
		parseT.Fatalf("expected capture invocation to succeed, got %v", parseErr2)
	}
	if !parsePayload.Present() || parsePayload.String() == "" {
		parseT.Fatal("expected exported hot reload payload")
	}

	if parseErr3 := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "light"); parseErr3 != nil {
		parseT.Fatalf("unexpected mutation error: %v", parseErr3)
	}
	parseResult, parseErr2 := parseRestoreFn.Invoke(parsePayload.String())
	if parseErr2 != nil {
		parseT.Fatalf("expected restore invocation to succeed, got %v", parseErr2)
	}
	parseResultValue, parseErr2 := parseResult.ToGo()
	if parseErr2 != nil {
		parseT.Fatalf("expected restore result to decode, got %v", parseErr2)
	}
	parseResultMap, parseOk := parseResultValue.(map[string]any)
	if !parseOk || parseResultMap["outcome"] != "restored" {
		parseT.Fatalf("expected restore result payload, got %#v", parseResult)
	}

	parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("hot-reload-theme")
	if !parseOk {
		parseT.Fatal("expected restored atom to be present")
	}
	if parseValue != "dark" {
		parseT.Fatalf("expected restored atom value dark, got %#v", parseValue)
	}

	if _, parseErr4 := parsePrepareFn.Invoke(); parseErr4 != nil {
		parseT.Fatalf("expected prepare invocation to succeed, got %v", parseErr4)
	}

	parseStatus, parseErr2 := parseStatusFn.Invoke()
	if parseErr2 != nil {
		parseT.Fatalf("expected status invocation to succeed, got %v", parseErr2)
	}
	parseStatusValue, parseErr2 := parseStatus.ToGo()
	if parseErr2 != nil {
		parseT.Fatalf("expected status payload to decode, got %v", parseErr2)
	}
	parseStatusMap, parseOk := parseStatusValue.(map[string]any)
	if !parseOk || parseStatusMap["outcome"] != "restored" {
		parseT.Fatalf("expected last restore status to mention restored, got %#v", parseStatus)
	}

	parseDiagnostics, parseErr2 := parseDiagnosticsFn.Invoke()
	if parseErr2 != nil {
		parseT.Fatalf("expected diagnostics invocation to succeed, got %v", parseErr2)
	}
	parseDiagnosticsValue, parseErr2 := parseDiagnostics.ToGo()
	if parseErr2 != nil {
		parseT.Fatalf("expected diagnostics payload to decode, got %v", parseErr2)
	}
	parseDiagnosticsList, parseOk := parseDiagnosticsValue.([]any)
	if !parseOk || len(parseDiagnosticsList) != 0 {
		parseT.Fatalf("expected no hot reload diagnostics, got %#v", parseDiagnostics)
	}

	parseActivity, parseErr2 := parseActivityFn.Invoke()
	if parseErr2 != nil {
		parseT.Fatalf("expected activity invocation to succeed, got %v", parseErr2)
	}
	parseActivityValue, parseErr2 := parseActivity.ToGo()
	if parseErr2 != nil {
		parseT.Fatalf("expected activity payload to decode, got %v", parseErr2)
	}
	if _, parseOk2 := parseActivityValue.([]any); !parseOk2 {
		parseT.Fatalf("expected activity list payload, got %#v", parseActivityValue)
	}
}

func TestImportSnapshotSkipsRestoreWhenResetKeyChanges(parseT *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "dark"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}

	parseGlobal, parseErr2 := interop.GetGlobalThis()
	if parseErr2 != nil {
		parseT.Fatalf("expected browser global, got %v", parseErr2)
	}
	parsePrevBridge := parseGlobal.Get(appBridgeGlobal)

	parseT.Cleanup(func() {
		Disable()
		if parsePrevBridge.Present() {
			_ = parseGlobal.Set(appBridgeGlobal, parsePrevBridge)
		} else {
			_ = parseGlobal.Delete(appBridgeGlobal)
		}
	})

	Configure(Config{ResetKey: "counter:v1"})
	parsePayload, parseErr2 := GetSnapshot()
	if parseErr2 != nil {
		parseT.Fatalf("expected snapshot export to succeed, got %v", parseErr2)
	}
	if !strings.Contains(parsePayload, `"resetKey":"counter:v1"`) {
		parseT.Fatalf("expected reset key in exported payload, got %s", parsePayload)
	}

	if parseErr3 := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "light"); parseErr3 != nil {
		parseT.Fatalf("unexpected mutation error: %v", parseErr3)
	}

	Configure(Config{ResetKey: "counter:v2"})
	if parseErr4 := ApplySnapshot(parsePayload); parseErr4 != nil {
		parseT.Fatalf("expected reset-key mismatch to be ignored without error, got %v", parseErr4)
	}

	parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("hot-reload-theme")
	if !parseOk {
		parseT.Fatal("expected atom to remain present")
	}
	if parseValue != "light" {
		parseT.Fatalf("expected reset-key mismatch to skip restore and keep light, got %#v", parseValue)
	}
	if parseBridge, parseErr5 := appBridge(); parseErr5 != nil {
		parseT.Fatalf("expected bridge to remain installed, got %v", parseErr5)
	} else if parseGot := parseBridge.Get("resetKey"); !parseGot.Present() || parseGot.String() != "counter:v2" {
		parseT.Fatalf("expected bridge reset key counter:v2, got %#v", parseGot)
	} else if parseStatus, parseErr6 := parseBridge.Get("getLastRestoreResult").Invoke(); parseErr6 != nil {
		parseT.Fatalf("expected restore status invocation to succeed, got %v", parseErr6)
	} else if parseStatusValue, parseErr7 := parseStatus.ToGo(); parseErr7 != nil {
		parseT.Fatalf("expected restore status payload to decode, got %v", parseErr7)
	} else if parseStatusMap, parseOk2 := parseStatusValue.(map[string]any); !parseOk2 || parseStatusMap["outcome"] != "skipped-reset-key" {
		parseT.Fatalf("expected skipped-reset-key status, got %#v", parseStatus)
	}
}
