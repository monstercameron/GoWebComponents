package devtools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBrowserExtensionManifestForChromeAndFirefox(parseT *testing.T) {
	parseChrome := BrowserExtensionManifestFor("chrome")
	if parseChrome.ManifestVersion != 3 {
		parseT.Fatalf("manifest version = %d, want 3", parseChrome.ManifestVersion)
	}
	if parseChrome.DevtoolsPage != "devtools.html" {
		parseT.Fatalf("devtools page = %q", parseChrome.DevtoolsPage)
	}
	if len(parseChrome.BrowserSpecific) != 0 {
		parseT.Fatalf("chrome manifest unexpectedly has browser-specific settings: %#v", parseChrome.BrowserSpecific)
	}

	parseFirefox := BrowserExtensionManifestFor("firefox")
	if parseFirefox.BrowserSpecific["gecko"] == nil {
		parseT.Fatalf("firefox manifest missing gecko settings: %#v", parseFirefox.BrowserSpecific)
	}
}

func TestExportExtensionPanelPayloadJSONKeepsCoreInspectionSections(parseT *testing.T) {
	parsePayload, parseErr := ExportExtensionPanelPayloadJSON(Snapshot{
		Tree: &Node{Name: "App", Path: "0"},
		Stats: Stats{
			TotalFibers:     3,
			ComponentFibers: 1,
		},
		Profiling: Profiling{
			RenderCalls: 2,
		},
		Extensions: []ExtensionSection{{Name: "Atoms", Summary: map[string]string{"count": "1"}}},
		Logs:       []Log{{Domain: "runtime", Message: "mounted"}},
	})
	if parseErr != nil {
		parseT.Fatalf("ExportExtensionPanelPayloadJSON returned error: %v", parseErr)
	}

	var parseDecoded ExtensionPanelPayload
	if parseErr := json.Unmarshal(parsePayload, &parseDecoded); parseErr != nil {
		parseT.Fatalf("payload is not valid JSON: %v", parseErr)
	}
	if parseDecoded.SchemaVersion != "gwc.devtools.extension.v1" {
		parseT.Fatalf("schema = %q", parseDecoded.SchemaVersion)
	}
	if parseDecoded.Tree == nil || parseDecoded.Tree.Name != "App" {
		parseT.Fatalf("tree was not preserved: %#v", parseDecoded.Tree)
	}
	if !strings.Contains(string(parsePayload), `"RenderCalls":2`) {
		parseT.Fatalf("payload missing profiling section: %s", string(parsePayload))
	}
}
