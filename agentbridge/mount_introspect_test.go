package agentbridge

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// TestListMountComponents pins that RegisterMountComponent makes a component
// discoverable through ListMountComponents (and thus through describe), sorted.
func TestListMountComponents(t *testing.T) {
	RegisterMountComponent("ZWidget", func(map[string]any) *runtime.Element { return nil })
	RegisterMountComponent("APanel", func(map[string]any) *runtime.Element { return nil })

	parseNames := ListMountComponents()
	parseHasZ, parseHasA, parseSorted := false, false, true
	parsePrev := ""
	for _, parseName := range parseNames {
		if parseName == "ZWidget" {
			parseHasZ = true
		}
		if parseName == "APanel" {
			parseHasA = true
		}
		if parsePrev != "" && parseName < parsePrev {
			parseSorted = false
		}
		parsePrev = parseName
	}
	if !parseHasZ || !parseHasA {
		t.Fatalf("registered components not listed: %v", parseNames)
	}
	if !parseSorted {
		t.Fatalf("mount components not sorted: %v", parseNames)
	}
}

func TestRegisterMountComponentIgnoresInvalidRegistrations(t *testing.T) {
	writeMountMu.Lock()
	parseBefore := len(writeMountComponents)
	writeMountMu.Unlock()

	RegisterMountComponent("   ", func(map[string]any) *runtime.Element { return runtime.Div(nil) })
	RegisterMountComponent("agentbridge.invalid.NilFactory", nil)

	writeMountMu.Lock()
	parseAfter := len(writeMountComponents)
	writeMountMu.Unlock()
	if parseAfter != parseBefore {
		t.Fatalf("invalid registrations changed component count: before=%d after=%d", parseBefore, parseAfter)
	}
}

func TestWriteRemountSuccessDuplicateAndRollbackContracts(t *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "agent-remount")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	RegisterMountComponent("agentbridge.remount.Panel", func(parseProps map[string]any) *runtime.Element {
		parseText, _ := parseProps["text"].(string)
		return runtime.Div(map[string]any{"id": "remounted-panel"}, parseText)
	})
	RegisterMountComponent("agentbridge.remount.Nil", func(map[string]any) *runtime.Element { return nil })

	if parseErr := writeRemount("remount-1", "agentbridge.remount.Panel", "#agent-remount", map[string]any{"text": "hello"}); parseErr != nil {
		t.Fatalf("writeRemount success returned error: %v", parseErr)
	}
	writeMountMu.Lock()
	_, parseMounted := writeMountedRoots["remount-1"]
	writeMountMu.Unlock()
	if !parseMounted {
		t.Fatal("successful remount did not reserve mount id")
	}
	if len(parseAdapter.GetChildren(parseContainer)) == 0 {
		t.Fatal("successful remount did not render into container")
	}

	parseDupErr := writeRemount("remount-1", "agentbridge.remount.Panel", "#agent-remount", map[string]any{})
	if parseDupErr == nil || parseDupErr.Code != ErrorCodeBadPayload || !strings.Contains(parseDupErr.Message, "already mounted") {
		t.Fatalf("duplicate remount error = %#v, want already mounted", parseDupErr)
	}

	parseMissingErr := writeRemount("missing-component", "agentbridge.remount.Missing", "#agent-remount", map[string]any{})
	if parseMissingErr == nil || parseMissingErr.Code != ErrorCodeBadPayload || !strings.Contains(parseMissingErr.Message, "is not registered") {
		t.Fatalf("missing remount component error = %#v, want not registered", parseMissingErr)
	}

	parseNilErr := writeRemount("nil-remount", "agentbridge.remount.Nil", "#agent-remount", map[string]any{})
	if parseNilErr == nil || parseNilErr.Code != ErrorCodeBadPayload || !strings.Contains(parseNilErr.Message, "rendered nil") {
		t.Fatalf("nil remount error = %#v, want rendered nil", parseNilErr)
	}
	writeMountMu.Lock()
	_, parseNilLeaked := writeMountedRoots["nil-remount"]
	writeMountMu.Unlock()
	if parseNilLeaked {
		t.Fatal("remount reservation leaked after nil factory")
	}
}
