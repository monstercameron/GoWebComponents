package agentbridge

import (
	"testing"

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
