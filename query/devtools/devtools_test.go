package devtools_test

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/query"
	"github.com/monstercameron/GoWebComponents/v6/query/devtools"
)

// TestCachePanelRendersKeysAndStatus renders the query devtools panel headlessly and
// asserts each cached key appears with its status — the data layer observed in-app.
func TestCachePanelRendersKeysAndStatus(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("user/1", "Ada")
	parseCache.Set("user/2", "Bob")

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, devtools.CachePanel(parseCache)); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}

	parseText := collectText(parseAdapter, parseRoot)
	for _, parseWant := range []string{"Query cache", "user/1", "user/2", "fresh"} {
		if !strings.Contains(parseText, parseWant) {
			parseT.Fatalf("cache panel DOM missing %q; got %q", parseWant, parseText)
		}
	}
}

// TestInspectReportsFreshness proves the introspection API the panel reads classifies
// entries (fresh vs empty), independent of rendering.
func TestInspectReportsFreshness(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("k", "v")

	parseInfos := parseCache.Inspect()
	if len(parseInfos) != 1 || parseInfos[0].Key != "k" {
		parseT.Fatalf("expected one entry for k, got %+v", parseInfos)
	}
	if !parseInfos[0].HasData || parseInfos[0].Stale || parseInfos[0].Fetching {
		parseT.Fatalf("a freshly set entry should be has-data, not stale, not fetching: %+v", parseInfos[0])
	}
}

// TestCachePanelNilCache proves a nil cache renders a panic-free placeholder.
func TestCachePanelNilCache(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, devtools.CachePanel(nil)); parseErr != nil {
		parseT.Fatalf("nil cache panel should render: %v", parseErr)
	}
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
