package servercomponents

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestServerOnlyRendersServerNodeAndCollectsManifest(parseT *testing.T) {
	parseManifest := Manifest{}
	parseProps := Props{
		ID:    "pricing",
		Name:  "PricingTable",
		Props: map[string]string{"plan": "pro"},
		ClientSlots: []ClientReference{{
			ID:     "buy",
			Name:   "BuyButton",
			Props:  map[string]string{"sku": "pro"},
			Export: "BuyButton",
		}},
		Render: func() ui.Node {
			return html.Tag("section", html.Props{ID: "pricing"}, html.Text("Server priced"))
		},
	}
	Collect(&parseManifest, parseProps)
	parseHTML, parseErr := ui.RenderToString(ServerOnly(parseProps))
	if parseErr != nil {
		parseT.Fatalf("render server component: %v", parseErr)
	}
	if !strings.Contains(parseHTML, "Server priced") {
		parseT.Fatalf("server component did not render server node: %s", parseHTML)
	}
	if len(parseManifest.Components) != 1 || parseManifest.Components[0].ClientSlots[0].Name != "BuyButton" {
		parseT.Fatalf("manifest missing descriptor: %#v", parseManifest)
	}
	parseProps.Props["plan"] = "mutated"
	if parseManifest.Components[0].Props["plan"] != "pro" {
		parseT.Fatalf("manifest props were not cloned: %#v", parseManifest.Components[0].Props)
	}
	parseProps.ClientSlots[0].Props["sku"] = "mutated"
	if parseManifest.Components[0].ClientSlots[0].Props["sku"] != "pro" {
		parseT.Fatalf("client slot props were not cloned: %#v", parseManifest.Components[0].ClientSlots[0].Props)
	}
}

func TestCollectHandlesNilManifestAndEmptyValues(parseT *testing.T) {
	Collect(nil, Props{ID: "ignored"})

	parseManifest := Manifest{}
	Collect(&parseManifest, Props{ID: "empty"})
	if len(parseManifest.Components) != 1 {
		parseT.Fatalf("expected one descriptor, got %#v", parseManifest.Components)
	}
	parseDescriptor := parseManifest.Components[0]
	if parseDescriptor.Props != nil || parseDescriptor.ClientSlots != nil {
		parseT.Fatalf("empty descriptor should omit maps/slices, got %#v", parseDescriptor)
	}
}

func TestServerOnlyNativeFallsBackToPlaceholderWithoutRender(parseT *testing.T) {
	parseNode := ServerOnly(Props{
		ID:          "client-only",
		Placeholder: html.Tag("span", html.Props{ID: "fallback"}, html.Text("Loading")),
	})
	parseHTML, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("render placeholder: %v", parseErr)
	}
	if !strings.Contains(parseHTML, `id="fallback"`) || !strings.Contains(parseHTML, "Loading") {
		parseT.Fatalf("placeholder did not render: %s", parseHTML)
	}
}
