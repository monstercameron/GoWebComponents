package a11y

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
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
