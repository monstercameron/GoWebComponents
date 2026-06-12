package servercomponents

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
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
}
