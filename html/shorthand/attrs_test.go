package shorthand

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html/internal/attrtable"
	_ "github.com/monstercameron/GoWebComponents/v6/html/markdown"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestShorthandExposesEveryGeneratedAttrBuilder(parseT *testing.T) {
	parseCases := getGeneratedAttrCases()
	if len(parseCases) != len(attrtable.Entries) {
		parseT.Fatalf("shorthand cases = %d, table entries = %d", len(parseCases), len(attrtable.Entries))
	}
	for _, parseEntry := range attrtable.Entries {
		parseOption, isPresent := parseCases[parseEntry.Name]
		if !isPresent {
			parseT.Fatalf("shorthand is missing %s", parseEntry.Name)
		}
		parseMarkup, parseErr := ui.RenderToString(Div(PropsOf(parseOption)))
		if parseErr != nil {
			parseT.Fatalf("%s: %v", parseEntry.Name, parseErr)
		}
		parseWant := " " + parseEntry.Attr
		if parseEntry.Kind != attrtable.KindFlag {
			parseWant += `="sample"`
			if parseEntry.Kind == attrtable.KindEnumBool || parseEntry.Kind == attrtable.KindAriaBool {
				parseWant = " " + parseEntry.Attr + `="true"`
			}
		}
		if !strings.Contains(parseMarkup, parseWant) {
			parseT.Errorf("%s: want %q in %s", parseEntry.Name, parseWant, parseMarkup)
		}
	}
}

func TestShorthandConditionalAndMarkdownOptions(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(Input(PropsOf(CheckedIf(false), RequiredIf(true), IfProp(true, Title("t")))))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if strings.Contains(parseMarkup, "checked") || !strings.Contains(parseMarkup, "required") || !strings.Contains(parseMarkup, `title="t"`) {
		parseT.Errorf("unexpected markup: %s", parseMarkup)
	}
	parseNodes := Markdown("[x](ftp://x.example)", WithURLSchemes())
	parseOut, parseErr := ui.RenderToString(Div(parseNodes))
	if parseErr != nil || strings.Contains(parseOut, "ftp://") {
		parseT.Errorf("WithURLSchemes() must reject ftp: %v %s", parseErr, parseOut)
	}
}
