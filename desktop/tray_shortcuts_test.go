package desktop

import (
	"context"
	"encoding/json"
	"testing"
)

type parseTrayShortcutFake struct {
	*parseNativeFake
	parseTrays     map[string]TrayRequest
	parseShortcuts map[string]ShortcutRequest
	parseCalls     int
}

// ConfigureTray records replacement and removal lifecycle operations.
func (parseFake *parseTrayShortcutFake) ConfigureTray(_ context.Context, parseRequest TrayRequest) error {
	parseFake.parseCalls++
	if parseRequest.Action == "remove" {
		delete(parseFake.parseTrays, parseRequest.ID)
	} else {
		parseFake.parseTrays[parseRequest.ID] = parseRequest
	}
	return nil
}

// ConfigureShortcut records replacement and removal lifecycle operations.
func (parseFake *parseTrayShortcutFake) ConfigureShortcut(_ context.Context, parseRequest ShortcutRequest) error {
	parseFake.parseCalls++
	if parseRequest.Action == "remove" {
		delete(parseFake.parseShortcuts, parseRequest.ID)
	} else {
		parseFake.parseShortcuts[parseRequest.ID] = parseRequest
	}
	return nil
}

// TestTrayShortcutValidationStopsBackend verifies bounds and typed actions before effects.
func TestTrayShortcutValidationStopsBackend(parseTest *testing.T) {
	parseFake := &parseTrayShortcutFake{parseNativeFake: &parseNativeFake{parseFeatures: []Feature{SystemTray, GlobalShortcuts}}, parseTrays: map[string]TrayRequest{}, parseShortcuts: map[string]ShortcutRequest{}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parseFake, parsePolicy)
	parseInvalid := []error{
		parseHost.ConfigureTray(context.Background(), TrayRequest{Action: "upsert", ID: "bad/id"}),
		parseHost.ConfigureTray(context.Background(), TrayRequest{Action: "upsert", ID: "tray", Icon: []byte("not-png")}),
		parseHost.ConfigureTray(context.Background(), TrayRequest{Action: "remove", ID: "tray", Tooltip: "extra"}),
		parseHost.ConfigureTray(context.Background(), TrayRequest{Action: "upsert", ID: "tray", Menu: &Menu{Items: []MenuItem{{ID: "open", Kind: MenuItemCommand, Label: "Open", Enabled: true, Shortcut: "Ctrl+O"}}}}),
		parseHost.ConfigureShortcut(context.Background(), ShortcutRequest{Action: "upsert", ID: "hotkey"}),
		parseHost.ConfigureShortcut(context.Background(), ShortcutRequest{Action: "remove", ID: "hotkey", Accelerator: "Ctrl+K"}),
	}
	for _, parseErr := range parseInvalid {
		if parseErr == nil {
			parseTest.Fatal("expected invalid request")
		}
	}
	if parseFake.parseCalls != 0 {
		parseTest.Fatal("invalid operations reached backend")
	}
}

// TestTrayEventValidationRejectsArbitraryCallbacks verifies the event namespace has a fixed schema.
func TestTrayEventValidationRejectsArbitraryCallbacks(parseTest *testing.T) {
	for _, parseEvent := range []TrayEvent{{ID: "main", Kind: "script"}, {ID: "bad/id", Kind: "click"}, {ID: "main", Kind: "menu"}} {
		if parseErr := validateTrayEvent(parseEvent); parseErr == nil {
			parseTest.Fatalf("accepted malformed tray event: %#v", parseEvent)
		}
	}
	if parseErr := validateTrayEvent(TrayEvent{ID: "main", Kind: "menu", ItemID: "open"}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
}

// TestTrayShortcutReplacementLifecycle verifies stable identifiers replace and remove resources.
func TestTrayShortcutReplacementLifecycle(parseTest *testing.T) {
	parseFake := &parseTrayShortcutFake{parseNativeFake: &parseNativeFake{parseFeatures: []Feature{SystemTray, GlobalShortcuts}}, parseTrays: map[string]TrayRequest{}, parseShortcuts: map[string]ShortcutRequest{}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parseFake, parsePolicy)
	parseContext := context.Background()
	for _, parseRequest := range []TrayRequest{{Action: "upsert", ID: "main", Tooltip: "one"}, {Action: "upsert", ID: "main", Tooltip: "two"}, {Action: "remove", ID: "main"}} {
		if parseErr := parseHost.ConfigureTray(parseContext, parseRequest); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
	}
	for _, parseRequest := range []ShortcutRequest{{Action: "upsert", ID: "search", Accelerator: "Ctrl+K"}, {Action: "upsert", ID: "search", Accelerator: "Ctrl+Shift+K"}, {Action: "remove", ID: "search"}} {
		if parseErr := parseHost.ConfigureShortcut(parseContext, parseRequest); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
	}
	if len(parseFake.parseTrays) != 0 || len(parseFake.parseShortcuts) != 0 || parseFake.parseCalls != 6 {
		parseTest.Fatalf("unexpected fake lifecycle state: trays=%d shortcuts=%d calls=%d", len(parseFake.parseTrays), len(parseFake.parseShortcuts), parseFake.parseCalls)
	}
}

// TestTrayShortcutFeatureGatesAndExecute verifies extension-aware advertisement and dispatch.
func TestTrayShortcutFeatureGatesAndExecute(parseTest *testing.T) {
	parseFake := &parseTrayShortcutFake{parseNativeFake: &parseNativeFake{parseFeatures: []Feature{SystemTray}}, parseTrays: map[string]TrayRequest{}, parseShortcuts: map[string]ShortcutRequest{}}
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseHost := NewNativeHost(parseFake, parsePolicy)
	parseMethods := parseHost.GetMethods()
	if len(parseMethods) != 1 || parseMethods[0] != TrayConfigureMethod {
		parseTest.Fatalf("unexpected methods: %#v", parseMethods)
	}
	parseArgs, _ := json.Marshal(TrayRequest{Action: "upsert", ID: "main", Tooltip: "GWC"})
	parseReply := parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: TrayConfigureMethod, Args: parseArgs})
	if parseReply.Code != "" || parseFake.parseCalls != 1 {
		parseTest.Fatalf("unexpected execute reply: %#v", parseReply)
	}
	if parseErr := parseHost.ConfigureShortcut(context.Background(), ShortcutRequest{Action: "upsert", ID: "key", Accelerator: "Ctrl+K"}); parseErr == nil {
		parseTest.Fatal("backend-disabled shortcut should be denied")
	}
}
