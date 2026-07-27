package a11y

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestHeadlessMenuAndComboboxExposeARIAContracts(parseT *testing.T) {
	parseMenuHTML, parseErr := ui.RenderToString(Menu(MenuProps{
		ID:       "actions",
		Label:    "Actions",
		Open:     true,
		ActiveID: "archive",
		Items: []Item{
			{ID: "archive", Label: "Archive"},
			{ID: "delete", Label: "Delete", Disabled: true},
		},
	}))
	if parseErr != nil {
		parseT.Fatalf("render menu: %v", parseErr)
	}
	for _, parseWant := range []string{`role="menu"`, `aria-expanded="true"`, `role="menuitem"`, `aria-disabled="true"`} {
		if !strings.Contains(parseMenuHTML, parseWant) {
			parseT.Fatalf("menu html missing %s: %s", parseWant, parseMenuHTML)
		}
	}

	parseComboHTML, parseErr := ui.RenderToString(Combobox(ComboboxProps{
		ID:       "assignee",
		Label:    "Assignee",
		Expanded: true,
		ActiveID: "person-a",
		Items:    []Item{{ID: "person-a", Label: "Ari", Selected: true}},
	}))
	if parseErr != nil {
		parseT.Fatalf("render combobox: %v", parseErr)
	}
	for _, parseWant := range []string{`role="combobox"`, `aria-controls="assignee-listbox"`, `role="listbox"`, `role="option"`} {
		if !strings.Contains(parseComboHTML, parseWant) {
			parseT.Fatalf("combobox html missing %s: %s", parseWant, parseComboHTML)
		}
	}
}

func TestDatePickerAndTableExposeGridAndHeaderContracts(parseT *testing.T) {
	parseCalendarHTML, parseErr := ui.RenderToString(DatePicker(DatePickerProps{
		ID:       "due",
		Label:    "Due date",
		Month:    time.February,
		Year:     2024,
		Selected: 29,
	}))
	if parseErr != nil {
		parseT.Fatalf("render date picker: %v", parseErr)
	}
	if !strings.Contains(parseCalendarHTML, `role="grid"`) || !strings.Contains(parseCalendarHTML, `aria-label="February 29, 2024"`) || !strings.Contains(parseCalendarHTML, `aria-selected="true"`) {
		parseT.Fatalf("date picker html missing grid semantics: %s", parseCalendarHTML)
	}

	parseTableHTML, parseErr := ui.RenderToString(Table(TableProps{
		ID:      "orders",
		Caption: "Orders",
		Columns: []TableColumn{
			{ID: "id", Header: "ID"},
			{ID: "status", Header: "Status"},
		},
		Rows: []map[string]string{{"id": "A-1", "status": "Queued"}},
	}))
	if parseErr != nil {
		parseT.Fatalf("render table: %v", parseErr)
	}
	for _, parseWant := range []string{`<caption>Orders</caption>`, `scope="col"`, `headers="status"`, `Queued`} {
		if !strings.Contains(parseTableHTML, parseWant) {
			parseT.Fatalf("table html missing %s: %s", parseWant, parseTableHTML)
		}
	}
}

// TestListboxMultiSelectEmitsAriaMultiselectable proves a multi-select listbox emits the
// container-level aria-multiselectable="true" (WAI-ARIA 1.2 §5.5), and a single-select one omits it.
// A composite widget that owns arrow-key navigation and aria-activedescendant
// must be focusable, or none of that is reachable and role="listbox" is a promise
// the widget does not keep.
//
// This regressed silently for a long time: the source said TabIndex: 0, which
// looks correct and is not — 0 is Go's zero value for an int field, so
// html.Props could not tell "put this in the tab order" from "field unset" and
// emitted no attribute at all. The fix is html.TabIndexZero. This test asserts
// the emitted attribute rather than the source intent, because the source intent
// was already right while the output was wrong.
func TestListboxIsKeyboardFocusable(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(Listbox(ListboxProps{
		ID: "hubs", Label: "Hubs", Items: []Item{{ID: "a", Label: "A"}},
	}))
	if parseErr != nil {
		parseT.Fatalf("render listbox: %v", parseErr)
	}
	// Case-insensitive: the typed path currently serializes the key as tabIndex,
	// which is valid because HTML attribute names are case-insensitive. This test
	// pins focusability, not the casing — see normalizeSSRAttrName.
	parseLower := strings.ToLower(parseMarkup)
	if !strings.Contains(parseLower, `tabindex="0"`) {
		parseT.Fatalf("listbox must be focusable via tabindex=\"0\", got %q", parseMarkup)
	}
}

func TestListboxMultiSelectEmitsAriaMultiselectable(parseT *testing.T) {
	parseMulti, parseErr := ui.RenderToString(Listbox(ListboxProps{
		ID: "tags", Label: "Tags", MultiSelect: true,
		Items: []Item{{ID: "a", Label: "A", Selected: true}, {ID: "b", Label: "B", Selected: true}},
	}))
	if parseErr != nil {
		parseT.Fatalf("render multi: %v", parseErr)
	}
	if !strings.Contains(parseMulti, `aria-multiselectable="true"`) {
		parseT.Fatalf("multi-select listbox must emit aria-multiselectable, got %q", parseMulti)
	}

	parseSingle, parseErr2 := ui.RenderToString(Listbox(ListboxProps{
		ID: "one", Label: "One", Items: []Item{{ID: "a", Label: "A"}},
	}))
	if parseErr2 != nil {
		parseT.Fatalf("render single: %v", parseErr2)
	}
	if strings.Contains(parseSingle, "aria-multiselectable") {
		parseT.Fatalf("single-select listbox must NOT emit aria-multiselectable, got %q", parseSingle)
	}
}
