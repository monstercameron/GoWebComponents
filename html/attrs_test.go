package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html/internal/attrtable"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderAttrMarkup renders one element carrying parseProps and returns its markup.
func renderAttrMarkup(parseT *testing.T, parseTag string, parseProps Props) string {
	parseT.Helper()
	parseMarkup, parseErr := ui.RenderToString(Tag(parseTag, parseProps))
	if parseErr != nil {
		parseT.Fatalf("render %s: %v", parseTag, parseErr)
	}
	return parseMarkup
}

func TestGeneratedAttrBuildersRenderTheirAttribute(parseT *testing.T) {
	parseCases := getGeneratedAttrCases()
	if len(parseCases) != len(attrtable.Entries) {
		parseT.Fatalf("generated cases = %d, table entries = %d", len(parseCases), len(attrtable.Entries))
	}
	for _, parseEntry := range attrtable.Entries {
		parseOption, isPresent := parseCases[parseEntry.Name]
		if !isPresent {
			parseT.Fatalf("missing generated builder %s", parseEntry.Name)
		}
		parseMarkup := renderAttrMarkup(parseT, "div", PropsOf(parseOption))
		parseWant := " " + parseEntry.Attr + `="sample"`
		switch parseEntry.Kind {
		case attrtable.KindEnumBool, attrtable.KindAriaBool:
			parseWant = " " + parseEntry.Attr + `="true"`
		case attrtable.KindFlag:
			parseWant = " " + parseEntry.Attr
		}
		if !strings.Contains(parseMarkup, parseWant) {
			parseT.Errorf("%s: want %q in %s", parseEntry.Name, parseWant, parseMarkup)
		}
	}
}

func TestGeneratedBooleanBuildersHonourFalse(parseT *testing.T) {
	if parseMarkup := renderAttrMarkup(parseT, "button", PropsOf(FormNoValidate(false))); strings.Contains(parseMarkup, "formnovalidate") {
		parseT.Errorf("FormNoValidate(false) must omit the attribute: %s", parseMarkup)
	}
	if parseMarkup := renderAttrMarkup(parseT, "button", PropsOf(FormNoValidate(true), FormNoValidate(false))); strings.Contains(parseMarkup, "formnovalidate") {
		parseT.Errorf("a later FormNoValidate(false) must win: %s", parseMarkup)
	}
	if parseMarkup := renderAttrMarkup(parseT, "div", PropsOf(Draggable(false), AriaHidden(false))); !strings.Contains(parseMarkup, `draggable="false"`) || !strings.Contains(parseMarkup, `aria-hidden="false"`) {
		parseT.Errorf("false enum booleans must render false: %s", parseMarkup)
	}
}

func TestGeneratedConditionalPropOptions(parseT *testing.T) {
	parseCases := []struct {
		name      string
		option    PropOption
		attr      string
		isPresent bool
	}{
		{"CheckedIf true", CheckedIf(true), " checked", true},
		{"CheckedIf false", CheckedIf(false), " checked", false},
		{"RequiredIf true", RequiredIf(true), " required", true},
		{"RequiredIf false", RequiredIf(false), " required", false},
		{"IfProp true", IfProp(true, Title("on"), Hidden()), `title="on"`, true},
		{"IfProp false", IfProp(false, Title("on"), Hidden()), `title="on"`, false},
		{"UnlessProp true", UnlessProp(true, Title("off")), `title="off"`, false},
		{"UnlessProp false", UnlessProp(false, Title("off")), `title="off"`, true},
		{"IfProp empty", IfProp(true), "title", false},
	}
	for _, parseCase := range parseCases {
		parseMarkup := renderAttrMarkup(parseT, "input", PropsOf(parseCase.option))
		if got := strings.Contains(parseMarkup, parseCase.attr); got != parseCase.isPresent {
			parseT.Errorf("%s: contains %q = %v, want %v (%s)", parseCase.name, parseCase.attr, got, parseCase.isPresent, parseMarkup)
		}
	}
	parseMarkup := renderAttrMarkup(parseT, "div", PropsOf(IfProp(true, Title("a"), Hidden())))
	if !strings.Contains(parseMarkup, "hidden") || !strings.Contains(parseMarkup, `title="a"`) {
		parseT.Errorf("IfProp must apply every option: %s", parseMarkup)
	}
}
