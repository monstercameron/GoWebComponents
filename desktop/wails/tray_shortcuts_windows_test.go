//go:build windows

package wails

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type parseTrayWindowFake struct {
	application.Window
	parseMutex          sync.Mutex
	parseEvents         []desktop.TrayEvent
	parseShortcutEvents []desktop.ShortcutEvent
}

// ID returns a stable caller identity.
func (*parseTrayWindowFake) ID() uint { return 77 }

// EmitEvent records typed tray callbacks.
func (parseWindow *parseTrayWindowFake) EmitEvent(_ string, parseData ...any) bool {
	parseWindow.parseMutex.Lock()
	defer parseWindow.parseMutex.Unlock()
	if len(parseData) == 1 {
		if parseEvent, parseOK := parseData[0].(desktop.TrayEvent); parseOK {
			parseWindow.parseEvents = append(parseWindow.parseEvents, parseEvent)
		}
		if parseEvent, parseOK := parseData[0].(desktop.ShortcutEvent); parseOK {
			parseWindow.parseShortcutEvents = append(parseWindow.parseShortcutEvents, parseEvent)
		}
	}
	return true
}

// TestShortcutGenerationNeverRevivesStaleCallbacks verifies removed IDs retain monotonic tokens.
func TestShortcutGenerationNeverRevivesStaleCallbacks(parseTest *testing.T) {
	parseWindow := &parseTrayWindowFake{}
	parseOwner := &nativeOwnerState{parseShortcuts: map[string]string{"main": "Ctrl+K"}, parseShortcutGenerations: map[string]uint64{"main": 4}}
	parseState := newNativeTrayShortcutState()
	parseState.parseOwners[parseWindow.ID()] = parseOwner
	parseState.emitShortcutEvent(parseWindow, "main", 3)
	parseState.emitShortcutEvent(parseWindow, "main", 4)
	delete(parseOwner.parseShortcuts, "main")
	parseState.emitShortcutEvent(parseWindow, "main", 4)
	parseOwner.parseShortcuts["main"] = "Ctrl+L"
	parseOwner.parseShortcutGenerations["main"]++
	parseState.emitShortcutEvent(parseWindow, "main", 4)
	parseState.emitShortcutEvent(parseWindow, "main", 5)
	if len(parseWindow.parseShortcutEvents) != 2 {
		parseTest.Fatalf("stale shortcut callbacks revived: %#v", parseWindow.parseShortcutEvents)
	}
}

// TestDefaultTrayIconIsValid verifies the synthetic fallback is a bounded decodable PNG.
func TestDefaultTrayIconIsValid(parseTest *testing.T) {
	parseIcon, parseErr := base64.StdEncoding.DecodeString(parseTrayIconPNG)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseConfig, parseErr := png.DecodeConfig(bytes.NewReader(parseIcon))
	if parseErr != nil || parseConfig.Width != 16 || parseConfig.Height != 16 {
		parseTest.Fatalf("invalid default tray icon: %+v: %v", parseConfig, parseErr)
	}
}

// TestTrayShortcutBackendExtensionsRequireCaller verifies optional capabilities never fall back to focus.
func TestTrayShortcutBackendExtensionsRequireCaller(parseTest *testing.T) {
	parseBackend := NewNativeBackend()
	parseTrayBackend, parseOK := parseBackend.(desktop.TrayBackend)
	if !parseOK {
		parseTest.Fatal("Windows backend does not implement tray extension")
	}
	if parseErr := parseTrayBackend.ConfigureTray(context.Background(), desktop.TrayRequest{Action: "upsert", ID: "main"}); parseErr == nil {
		parseTest.Fatal("tray accepted missing caller")
	}
	parseShortcutBackend, parseOK := parseBackend.(desktop.ShortcutBackend)
	if !parseOK {
		parseTest.Fatal("Windows backend does not implement shortcut extension")
	}
	if parseErr := parseShortcutBackend.ConfigureShortcut(context.Background(), desktop.ShortcutRequest{Action: "upsert", ID: "main", Accelerator: "Ctrl+Shift+K"}); parseErr == nil {
		parseTest.Fatal("shortcut accepted missing caller")
	}
}

// TestTrayGenerationNeverRevivesStaleCallbacks verifies remove and recreate keep monotonic identity.
func TestTrayGenerationNeverRevivesStaleCallbacks(parseTest *testing.T) {
	parseWindow := &parseTrayWindowFake{}
	parseOwner := &nativeOwnerState{parseTrays: map[string]*application.SystemTray{"main": nil}, parseTrayGenerations: map[string]uint64{"main": 2}}
	parseState := newNativeTrayShortcutState()
	parseState.parseOwners[parseWindow.ID()] = parseOwner
	parseState.emitTrayEvent(parseWindow, "main", 1, "click", "")
	parseState.emitTrayEvent(parseWindow, "main", 2, "click", "")
	delete(parseOwner.parseTrays, "main")
	parseState.emitTrayEvent(parseWindow, "main", 2, "click", "")
	parseOwner.parseTrays["main"] = nil
	parseOwner.parseTrayGenerations["main"]++
	parseState.emitTrayEvent(parseWindow, "main", 2, "click", "")
	parseState.emitTrayEvent(parseWindow, "main", 3, "click", "")
	if len(parseWindow.parseEvents) != 2 {
		parseTest.Fatalf("stale callbacks revived: %#v", parseWindow.parseEvents)
	}
}

// TestTrayCallbackDoesNotWaitForOperationLock verifies UI callbacks cannot deadlock native dispatch.
func TestTrayCallbackDoesNotWaitForOperationLock(parseTest *testing.T) {
	parseWindow := &parseTrayWindowFake{}
	parseState := newNativeTrayShortcutState()
	parseState.parseOwners[parseWindow.ID()] = &nativeOwnerState{parseTrays: map[string]*application.SystemTray{"main": nil}, parseTrayGenerations: map[string]uint64{"main": 1}}
	parseState.parseOperationMutex.Lock()
	parseDone := make(chan struct{})
	go func() {
		parseState.emitTrayEvent(parseWindow, "main", 1, "click", "")
		close(parseDone)
	}()
	select {
	case <-parseDone:
	case <-time.After(time.Second):
		parseTest.Fatal("tray callback waited for operation lock")
	}
	parseState.parseOperationMutex.Unlock()
}

// TestTrayPublicationUsesSingleUIBoundary verifies complete configuration is wrapped as one UI action.
func TestTrayPublicationUsesSingleUIBoundary(parseTest *testing.T) {
	parseState := newNativeTrayShortcutState()
	parseCalls := 0
	isInsideUI := false
	parseState.invokeUI = func(parseAction func()) {
		parseCalls++
		isInsideUI = true
		parseAction()
		isInsideUI = false
	}
	parseConfigured := false
	parseState.runTrayUI(func() {
		if !isInsideUI {
			parseTest.Fatal("tray configuration escaped UI boundary")
		}
		parseConfigured = true
	})
	if parseCalls != 1 || !parseConfigured || isInsideUI {
		parseTest.Fatalf("unexpected UI publication: calls=%d configured=%v inside=%v", parseCalls, parseConfigured, isInsideUI)
	}
}
