package services

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TestAPIKeyBindingsRespectIndependentPolicies keeps fullscreen escape available without menus.
func TestAPIKeyBindingsRespectIndependentPolicies(parseTest *testing.T) {
	for _, parseCase := range []struct {
		parsePolicy string
		hasF8       bool
		hasEscape   bool
	}{{"all", true, true}, {"none", false, false}, {"native-menus", true, false}, {"window-controls", false, true}, {"clipboard", false, false}} {
		parsePolicy, parseErr := desktop.ParseFeaturePolicy(parseCase.parsePolicy)
		if parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		parseBindings := APIKeyBindings(&APIService{Policy: parsePolicy})
		if (parseBindings["F8"] != nil) != parseCase.hasF8 || (parseBindings["Escape"] != nil) != parseCase.hasEscape {
			parseTest.Fatalf("%s bindings do not match independent feature policy", parseCase.parsePolicy)
		}
	}
	if len(APIKeyBindings(nil)) != 0 {
		parseTest.Fatal("nil service granted native shortcuts")
	}
}

// TestAPIContextMenuStructure verifies native roles, initial selection and disabled state.
// Activation remains a real Windows input test, not a direct callback invocation.
func TestAPIContextMenuStructure(parseTest *testing.T) {
	parseMenu := application.NewMenu()
	defer parseMenu.Destroy()
	buildAPIContextItems(parseMenu, &APIService{})
	if parseItem := parseMenu.FindByLabel("Record context selection"); parseItem == nil || !parseItem.Enabled() {
		parseTest.Fatal("missing context command")
	}
	if parseItem := parseMenu.FindByLabel("Checked option"); parseItem == nil || !parseItem.IsCheckbox() || parseItem.Checked() {
		parseTest.Fatal("checkbox must begin unchecked")
	}
	for _, parseLabel := range []string{"Radio Alpha", "Radio Beta"} {
		parseItem := parseMenu.FindByLabel(parseLabel)
		if parseItem == nil || !parseItem.IsRadio() || parseItem.Checked() != (parseLabel == "Radio Alpha") {
			parseTest.Fatalf("incorrect radio state: %s", parseLabel)
		}
	}
	if parseItem := parseMenu.FindByLabel("More actions"); parseItem == nil || !parseItem.IsSubmenu() || parseItem.GetSubmenu().FindByLabel("Record submenu selection") == nil {
		parseTest.Fatal("missing nested command")
	}
	if parseItem := parseMenu.FindByLabel("Disabled action (must not run)"); parseItem == nil || parseItem.Enabled() {
		parseTest.Fatal("disabled action enabled")
	}
}
