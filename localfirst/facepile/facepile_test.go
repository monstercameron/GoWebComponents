package facepile_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/localfirst"
	"github.com/monstercameron/GoWebComponents/v6/localfirst/facepile"
)

// TestFacepileRendersLivePeers renders the presence facepile headlessly and asserts every
// live peer appears — the multiplayer "who's here" surface over PresenceSet.
func TestFacepileRendersLivePeers(parseT *testing.T) {
	parsePresence := localfirst.NewPresenceSet(5)
	parsePresence.Update(localfirst.Presence{ClientID: "ada", State: `{"color":"red"}`})
	parsePresence.Update(localfirst.Presence{ClientID: "bob", State: `{"color":"blue"}`})

	parseText := render(parseT, facepile.Facepile(parsePresence))
	if !strings.Contains(parseText, "ada") || !strings.Contains(parseText, "bob") {
		parseT.Fatalf("expected both peers in the facepile, got %q", parseText)
	}
}

// TestFacepileReflectsExpiry proves the facepile shows only live peers — an expired peer
// (no heartbeat) drops off.
func TestFacepileReflectsExpiry(parseT *testing.T) {
	parsePresence := localfirst.NewPresenceSet(1)
	parsePresence.Update(localfirst.Presence{ClientID: "ghost"})
	parsePresence.Tick()
	parsePresence.Tick() // ghost exceeds ttl

	parseText := render(parseT, facepile.Facepile(parsePresence))
	if strings.Contains(parseText, "ghost") {
		parseT.Fatalf("an expired peer should not appear, got %q", parseText)
	}
}

// TestFacepileNilPresence proves a nil presence set renders panic-free.
func TestFacepileNilPresence(parseT *testing.T) {
	_ = render(parseT, facepile.Facepile(nil))
}

func render(parseT *testing.T, parseNode *runtime.Element) string {
	parseT.Helper()
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, parseNode); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	return collectText(parseAdapter, parseRoot)
}

func collectText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectText(parseAdapter, parseChild)
	}
	return parseText
}
