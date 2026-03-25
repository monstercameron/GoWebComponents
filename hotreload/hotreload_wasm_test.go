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

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

func TestConfigureInstallsBridgeObjectAndRestoresSnapshot(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	if err := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "dark"); err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	global, err := interop.GetGlobalThis()
	if err != nil {
		t.Fatalf("expected browser global, got %v", err)
	}
	prevBridge := global.Get(appBridgeGlobal)
	prevLiveReload := global.Get(liveReloadGlobal)

	t.Cleanup(func() {
		Disable()
		if prevBridge.Present() {
			_ = global.Set(appBridgeGlobal, prevBridge)
		} else {
			_ = global.Delete(appBridgeGlobal)
		}
		if prevLiveReload.Present() {
			_ = global.Set(liveReloadGlobal, prevLiveReload)
		} else {
			_ = global.Delete(liveReloadGlobal)
		}
	})

	Configure(Config{})
	if !Enabled() {
		t.Fatal("expected hotreload to be enabled")
	}

	bridge, err := appBridge()
	if err != nil {
		t.Fatalf("expected installed bridge, got %v", err)
	}
	captureFn := bridge.Get("captureSnapshot")
	prepareFn := bridge.Get("prepare")
	restoreFn := bridge.Get("restoreSnapshot")
	statusFn := bridge.Get("getLastRestoreResult")
	diagnosticsFn := bridge.Get("getHotReloadDiagnostics")
	activityFn := bridge.Get("getHotReloadActivity")
	if !captureFn.Present() || !prepareFn.Present() || !restoreFn.Present() || !statusFn.Present() || !diagnosticsFn.Present() || !activityFn.Present() {
		t.Fatal("expected bridge methods to be installed")
	}

	payload, err := captureFn.Invoke()
	if err != nil {
		t.Fatalf("expected capture invocation to succeed, got %v", err)
	}
	if !payload.Present() || payload.String() == "" {
		t.Fatal("expected exported hot reload payload")
	}

	if err := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "light"); err != nil {
		t.Fatalf("unexpected mutation error: %v", err)
	}
	result, err := restoreFn.Invoke(payload.String())
	if err != nil {
		t.Fatalf("expected restore invocation to succeed, got %v", err)
	}
	resultValue, err := result.ToGo()
	if err != nil {
		t.Fatalf("expected restore result to decode, got %v", err)
	}
	resultMap, ok := resultValue.(map[string]any)
	if !ok || resultMap["outcome"] != "restored" {
		t.Fatalf("expected restore result payload, got %#v", result)
	}

	value, ok := runtime.GetGlobalRuntime().GetAtomValue("hot-reload-theme")
	if !ok {
		t.Fatal("expected restored atom to be present")
	}
	if value != "dark" {
		t.Fatalf("expected restored atom value dark, got %#v", value)
	}

	if _, err := prepareFn.Invoke(); err != nil {
		t.Fatalf("expected prepare invocation to succeed, got %v", err)
	}

	status, err := statusFn.Invoke()
	if err != nil {
		t.Fatalf("expected status invocation to succeed, got %v", err)
	}
	statusValue, err := status.ToGo()
	if err != nil {
		t.Fatalf("expected status payload to decode, got %v", err)
	}
	statusMap, ok := statusValue.(map[string]any)
	if !ok || statusMap["outcome"] != "restored" {
		t.Fatalf("expected last restore status to mention restored, got %#v", status)
	}

	diagnostics, err := diagnosticsFn.Invoke()
	if err != nil {
		t.Fatalf("expected diagnostics invocation to succeed, got %v", err)
	}
	diagnosticsValue, err := diagnostics.ToGo()
	if err != nil {
		t.Fatalf("expected diagnostics payload to decode, got %v", err)
	}
	diagnosticsList, ok := diagnosticsValue.([]any)
	if !ok || len(diagnosticsList) != 0 {
		t.Fatalf("expected no hot reload diagnostics, got %#v", diagnostics)
	}

	activity, err := activityFn.Invoke()
	if err != nil {
		t.Fatalf("expected activity invocation to succeed, got %v", err)
	}
	activityValue, err := activity.ToGo()
	if err != nil {
		t.Fatalf("expected activity payload to decode, got %v", err)
	}
	if _, ok := activityValue.([]any); !ok {
		t.Fatalf("expected activity list payload, got %#v", activityValue)
	}
}

func TestImportSnapshotSkipsRestoreWhenResetKeyChanges(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	if err := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "dark"); err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	global, err := interop.GetGlobalThis()
	if err != nil {
		t.Fatalf("expected browser global, got %v", err)
	}
	prevBridge := global.Get(appBridgeGlobal)

	t.Cleanup(func() {
		Disable()
		if prevBridge.Present() {
			_ = global.Set(appBridgeGlobal, prevBridge)
		} else {
			_ = global.Delete(appBridgeGlobal)
		}
	})

	Configure(Config{ResetKey: "counter:v1"})
	payload, err := GetSnapshot()
	if err != nil {
		t.Fatalf("expected snapshot export to succeed, got %v", err)
	}
	if !strings.Contains(payload, `"resetKey":"counter:v1"`) {
		t.Fatalf("expected reset key in exported payload, got %s", payload)
	}

	if err := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "light"); err != nil {
		t.Fatalf("unexpected mutation error: %v", err)
	}

	Configure(Config{ResetKey: "counter:v2"})
	if err := ApplySnapshot(payload); err != nil {
		t.Fatalf("expected reset-key mismatch to be ignored without error, got %v", err)
	}

	value, ok := runtime.GetGlobalRuntime().GetAtomValue("hot-reload-theme")
	if !ok {
		t.Fatal("expected atom to remain present")
	}
	if value != "light" {
		t.Fatalf("expected reset-key mismatch to skip restore and keep light, got %#v", value)
	}
	if bridge, err := appBridge(); err != nil {
		t.Fatalf("expected bridge to remain installed, got %v", err)
	} else if got := bridge.Get("resetKey"); !got.Present() || got.String() != "counter:v2" {
		t.Fatalf("expected bridge reset key counter:v2, got %#v", got)
	} else if status, err := bridge.Get("getLastRestoreResult").Invoke(); err != nil {
		t.Fatalf("expected restore status invocation to succeed, got %v", err)
	} else if statusValue, err := status.ToGo(); err != nil {
		t.Fatalf("expected restore status payload to decode, got %v", err)
	} else if statusMap, ok := statusValue.(map[string]any); !ok || statusMap["outcome"] != "skipped-reset-key" {
		t.Fatalf("expected skipped-reset-key status, got %#v", status)
	}
}
