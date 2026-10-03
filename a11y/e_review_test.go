package a11y

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestEmptyActiveDescendantIsOmitted verifies an unset ActiveID does not emit an
// empty aria-activedescendant IDREF, and a set one is emitted.
func TestEmptyActiveDescendantIsOmitted(parseT *testing.T) {
	parseItems := []Item{{ID: "a", Label: "A"}}
	parseNodes := map[string]ui.Node{
		"menu":     Menu(MenuProps{ID: "m", Label: "M", Items: parseItems}),
		"listbox":  Listbox(ListboxProps{ID: "l", Label: "L", Items: parseItems}),
		"combobox": Combobox(ComboboxProps{ID: "c", Label: "C", Items: parseItems}),
	}
	for parseName, parseNode := range parseNodes {
		parseOut, parseErr := ui.RenderToString(parseNode)
		if parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if strings.Contains(parseOut, `aria-activedescendant=""`) {
			parseT.Errorf("%s emits an empty aria-activedescendant: %s", parseName, parseOut)
		}
	}
	parseOut, _ := ui.RenderToString(Listbox(ListboxProps{ID: "l", Label: "L", ActiveID: " a ", Items: parseItems}))
	if !strings.Contains(parseOut, `aria-activedescendant="a"`) {
		parseT.Errorf("set ActiveID must still be emitted: %s", parseOut)
	}
}

// TestComboboxPopupListboxIsNotATabStop verifies the combobox keeps focus on its
// input: the popup listbox must not add a second tab stop.
func TestComboboxPopupListboxIsNotATabStop(parseT *testing.T) {
	parseOut, parseErr := ui.RenderToString(Combobox(ComboboxProps{ID: "c", Label: "C", Items: []Item{{ID: "a", Label: "A"}}}))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if strings.Contains(strings.ToLower(parseOut), `tabindex="0"`) {
		parseT.Fatalf("combobox popup must not be tabindex=0: %s", parseOut)
	}
}
