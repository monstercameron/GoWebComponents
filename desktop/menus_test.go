package desktop

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

type parseRuntimeMenuFake struct {
	*parseNativeFake
	parseMenu  Menu
	parseCalls int
}

// TestSubscribeMenuSelectionsRejectsMalformedEvents verifies native event data is validated after decoding.
func TestSubscribeMenuSelectionsRejectsMalformedEvents(parseTest *testing.T) {
	parseEvents := make(chan error, 1)
	parseTransport := &desktopFakeTransport{
		parseCapabilities: Capabilities{Protocol: ProtocolVersion, Features: []Feature{RuntimeMenus}, Methods: []string{MenuReplaceMethod, ContextMenuInstallMethod, ContextMenuShowMethod, ContextMenuRemoveMethod}, Topics: []string{MenuSelectedTopic}},
		parseNext: func(string) (Reply, error) {
			return Reply{Done: true, Data: json.RawMessage(`{"id":"bad\u0000id"}`)}, nil
		},
	}
	parseStop, parseErr := NewClient(parseTransport).SubscribeMenuSelections(context.Background(), func(_ MenuSelection, parseErr error) {
		parseEvents <- parseErr
	})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer parseStop()
	select {
	case parseErr = <-parseEvents:
		if !interop.IsCode(parseErr, interop.CodeDecode) {
			parseTest.Fatalf("malformed selection error = %v", parseErr)
		}
	case <-time.After(time.Second):
		parseTest.Fatal("timed out waiting for malformed menu event")
	}
}

// ReplaceMenu records an isolated copy of the requested menu.
func (parseFake *parseRuntimeMenuFake) ReplaceMenu(_ context.Context, parseMenu Menu) error {
	parseFake.parseCalls++
	parseFake.parseMenu = parseMenu
	return nil
}

// getValidMenu returns a representative bounded menu tree.
func getValidMenu() Menu {
	return Menu{Items: []MenuItem{{ID: "file", Kind: MenuItemSubmenu, Label: "File", Enabled: true, Items: []MenuItem{
		{ID: "open", Kind: MenuItemCommand, Label: "Open", Enabled: true, Shortcut: "Ctrl+O", Icon: []byte{1, 2, 3}},
		{Kind: MenuItemSeparator},
		{ID: "autosave", Kind: MenuItemCheckbox, Label: "Autosave", Enabled: true, Checked: true},
	}}}}
}

// TestRuntimeMenuHostRequiresOptionalBackend verifies advertised-only legacy menu hosts remain non-RPC.
func TestRuntimeMenuHostRequiresOptionalBackend(parseTest *testing.T) {
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseLegacy := NewNativeHost(&parseNativeFake{parseFeatures: []Feature{NativeMenus, RuntimeMenus}}, parsePolicy)
	if hasName(parseLegacy.GetMethods(), MenuReplaceMethod) {
		parseTest.Fatal("host advertised runtime menu RPC without optional backend")
	}
	if parseErr := parseLegacy.ReplaceMenu(context.Background(), getValidMenu()); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseTest.Fatalf("legacy runtime menu error = %v", parseErr)
	}

	parseBackend := &parseRuntimeMenuFake{parseNativeFake: &parseNativeFake{parseFeatures: []Feature{RuntimeMenus}}}
	parseHost := NewNativeHost(parseBackend, parsePolicy)
	if !hasName(parseHost.GetMethods(), MenuReplaceMethod) {
		parseTest.Fatal("optional runtime menu backend did not advertise method")
	}
}

// TestRuntimeMenuValidationAndIsolation verifies invalid trees stop before effects and inputs are copied.
func TestRuntimeMenuValidationAndIsolation(parseTest *testing.T) {
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseBackend := &parseRuntimeMenuFake{parseNativeFake: &parseNativeFake{parseFeatures: []Feature{RuntimeMenus}}}
	parseHost := NewNativeHost(parseBackend, parsePolicy)
	parseDuplicate := Menu{Items: []MenuItem{{ID: "same", Kind: MenuItemCommand, Label: "A"}, {ID: "same", Kind: MenuItemCommand, Label: "B"}}}
	if parseErr := parseHost.ReplaceMenu(context.Background(), parseDuplicate); !interop.IsCode(parseErr, interop.CodeInvalid) || parseBackend.parseCalls != 0 {
		parseTest.Fatalf("invalid menu reached backend: calls=%d error=%v", parseBackend.parseCalls, parseErr)
	}
	parseMenu := getValidMenu()
	if parseErr := parseHost.ReplaceMenu(context.Background(), parseMenu); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseMenu.Items[0].Items[0].Label = "mutated"
	parseMenu.Items[0].Items[0].Icon[0] = 9
	if parseBackend.parseMenu.Items[0].Items[0].Label != "Open" {
		parseTest.Fatal("backend received caller-owned nested slice")
	}
	if parseBackend.parseMenu.Items[0].Items[0].Icon[0] != 1 {
		parseTest.Fatal("backend received caller-owned icon bytes")
	}
}

// TestRuntimeMenuExecute verifies the versioned native dispatcher uses the typed menu path.
func TestRuntimeMenuExecute(parseTest *testing.T) {
	parsePolicy, _ := ParseFeaturePolicy("all")
	parseBackend := &parseRuntimeMenuFake{parseNativeFake: &parseNativeFake{parseFeatures: []Feature{RuntimeMenus}}}
	parseHost := NewNativeHost(parseBackend, parsePolicy)
	parseArgs, _ := json.Marshal(getValidMenu())
	parseReply := parseHost.Execute(context.Background(), NativeRequest{Version: NativeContractVersion, Method: MenuReplaceMethod, Args: parseArgs})
	if parseReply.Code != "" || parseBackend.parseCalls != 1 {
		parseTest.Fatalf("execute reply=%#v calls=%d", parseReply, parseBackend.parseCalls)
	}
}
