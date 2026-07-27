package a11y

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Item describes one option in headless composite widgets.
type Item struct {
	ID       string
	Label    string
	Value    string
	Disabled bool
	Selected bool
}

type MenuProps struct {
	ID       string
	Label    string
	Open     bool
	ActiveID string
	Items    []Item
}

type ComboboxProps struct {
	ID          string
	Label       string
	Value       string
	Expanded    bool
	ActiveID    string
	Placeholder string
	Items       []Item
}

type ListboxProps struct {
	ID       string
	Label    string
	ActiveID string
	Items    []Item
	// MultiSelect declares the listbox allows multiple selected options. When true the container
	// gets aria-multiselectable="true" — required by WAI-ARIA 1.2 §5.5 so assistive tech announces
	// the multi-select mode (per-option aria-selected alone is not enough).
	MultiSelect bool
}

type DatePickerProps struct {
	ID       string
	Label    string
	Month    time.Month
	Year     int
	Selected int
}

type TableColumn struct {
	ID     string
	Header string
}

type TableProps struct {
	ID      string
	Caption string
	Columns []TableColumn
	Rows    []map[string]string
}

// Menu renders a WAI-ARIA menu surface using the core focus/composite primitives.
func Menu(parseProps MenuProps) ui.Node {
	parseItems := make([]ui.Node, 0, len(parseProps.Items))
	for _, parseItem := range parseProps.Items {
		parseItems = append(parseItems, html.Tag("div", html.Props{
			ID:       parseItem.ID,
			Role:     "menuitem",
			Disabled: parseItem.Disabled,
			TabIndex: -1,
			Aria: map[string]string{
				"disabled": boolString(parseItem.Disabled),
			},
		}, html.Text(parseItem.Label)))
	}
	return html.Tag("div", html.Props{
		ID:   parseProps.ID,
		Role: "menu",
		Aria: map[string]string{
			"label":            parseProps.Label,
			"expanded":         boolString(parseProps.Open),
			"activedescendant": strings.TrimSpace(parseProps.ActiveID),
		},
	}, parseItems...)
}

// Listbox renders a headless listbox with option selection state.
func Listbox(parseProps ListboxProps) ui.Node {
	parseItems := make([]ui.Node, 0, len(parseProps.Items))
	for _, parseItem := range parseProps.Items {
		parseItems = append(parseItems, html.Tag("div", html.Props{
			ID:   parseItem.ID,
			Role: "option",
			Aria: map[string]string{
				"selected": boolString(parseItem.Selected),
				"disabled": boolString(parseItem.Disabled),
			},
		}, html.Text(parseItem.Label)))
	}
	parseAria := map[string]string{
		"label":            parseProps.Label,
		"activedescendant": strings.TrimSpace(parseProps.ActiveID),
	}
	if parseProps.MultiSelect {
		parseAria["multiselectable"] = "true"
	}
	return html.Tag("div", html.Props{
		ID:   parseProps.ID,
		Role: "listbox",
		// TabIndexZero, not 0. A plain 0 is Go's zero value for an int field, so
		// html.Props cannot tell "put this in the tab order" from "field unset" and
		// emits no attribute at all. This listbox therefore shipped as a
		// role="listbox" that no keyboard user could focus — the exact defect this
		// package exists to prevent, in this package.
		//
		// A composite widget MUST be focusable: the listbox owns arrow-key
		// navigation and aria-activedescendant, so if it cannot take focus, none of
		// that is reachable and the role is a promise the widget does not keep.
		TabIndex: html.TabIndexZero,
		Aria:     parseAria,
	}, parseItems...)
}

// Combobox renders the input/listbox shell expected by screen readers.
func Combobox(parseProps ComboboxProps) ui.Node {
	parseListID := parseProps.ID + "-listbox"
	return html.Tag("div", html.Props{ID: parseProps.ID + "-root"},
		html.Input(html.Props{
			ID:          parseProps.ID,
			Role:        "combobox",
			Value:       parseProps.Value,
			Placeholder: parseProps.Placeholder,
			Aria: map[string]string{
				"label":            parseProps.Label,
				"expanded":         boolString(parseProps.Expanded),
				"controls":         parseListID,
				"autocomplete":     "list",
				"activedescendant": strings.TrimSpace(parseProps.ActiveID),
			},
		}),
		Listbox(ListboxProps{ID: parseListID, Label: parseProps.Label + " options", ActiveID: parseProps.ActiveID, Items: parseProps.Items}),
	)
}

// DatePicker renders a headless grid for one month.
func DatePicker(parseProps DatePickerProps) ui.Node {
	parseYear := parseProps.Year
	if parseYear == 0 {
		parseYear = time.Now().Year()
	}
	parseMonth := parseProps.Month
	if parseMonth == 0 {
		parseMonth = time.Now().Month()
	}
	parseDays := daysInMonth(parseYear, parseMonth)
	parseCells := make([]ui.Node, 0, parseDays)
	for parseDay := 1; parseDay <= parseDays; parseDay++ {
		parseCells = append(parseCells, html.Tag("div", html.Props{
			ID:   fmt.Sprintf("%s-day-%02d", parseProps.ID, parseDay),
			Role: "gridcell",
			Aria: map[string]string{
				"selected": boolString(parseDay == parseProps.Selected),
				"label":    fmt.Sprintf("%s %d, %d", parseMonth.String(), parseDay, parseYear),
			},
		}, html.Text(parseDay)))
	}
	return html.Tag("div", html.Props{
		ID:   parseProps.ID,
		Role: "grid",
		Aria: map[string]string{
			"label": parseProps.Label,
		},
	}, parseCells...)
}

// Table renders a semantic data table with stable headers and captions.
func Table(parseProps TableProps) ui.Node {
	parseHeaders := make([]ui.Node, 0, len(parseProps.Columns))
	for _, parseColumn := range parseProps.Columns {
		parseHeaders = append(parseHeaders, html.Th(html.Props{ID: parseColumn.ID, Raw: map[string]any{"scope": "col"}}, html.Text(parseColumn.Header)))
	}
	parseRows := make([]ui.Node, 0, len(parseProps.Rows))
	for parseRowIndex, parseRow := range parseProps.Rows {
		parseCells := make([]ui.Node, 0, len(parseProps.Columns))
		for _, parseColumn := range parseProps.Columns {
			parseCells = append(parseCells, html.Td(html.Props{Raw: map[string]any{"headers": parseColumn.ID}}, html.Text(parseRow[parseColumn.ID])))
		}
		parseRows = append(parseRows, html.Tag("tr", html.Props{Key: fmt.Sprintf("row-%d", parseRowIndex)}, parseCells...))
	}
	return html.Table(html.Props{ID: parseProps.ID},
		html.Tag("caption", html.Props{}, html.Text(parseProps.Caption)),
		html.Thead(html.Props{}, html.Tag("tr", html.Props{}, parseHeaders...)),
		html.Tag("tbody", html.Props{}, parseRows...),
	)
}

// RadioGroupProps configures a headless WAI-ARIA radio group. Mark the chosen
// option with Item.Selected; Item.Disabled removes it from the tab order.
type RadioGroupProps struct {
	ID          string
	Label       string
	Orientation string // "horizontal" (default) or "vertical"
	Items       []Item
}

// RadioGroup renders a headless radio group following the WAI-ARIA radiogroup
// pattern: role="radiogroup" wrapping role="radio" options with aria-checked /
// aria-disabled, and a single roving tab stop (tabindex 0 on the checked radio,
// or the first enabled one when none is selected; all others tabindex -1). Wire
// arrow-key navigation with ui.UseCompositeNavigation and reflect the active
// index back through Item.Selected — so Segmented/Swatch/Toggle controls get
// correct radiogroup semantics without each reimplementing them.
func RadioGroup(parseProps RadioGroupProps) ui.Node {
	parseTabStop := radioGroupTabStop(parseProps.Items)
	parseRadios := make([]ui.Node, 0, len(parseProps.Items))
	for parseIndex, parseItem := range parseProps.Items {
		parseTabIndex := "-1"
		if parseIndex == parseTabStop {
			parseTabIndex = "0"
		}
		parseRadios = append(parseRadios, html.Tag("div", html.Props{
			ID:   parseItem.ID,
			Role: "radio",
			// tabindex is emitted through Raw because Props.TabIndex==0 is
			// intentionally omitted by the serializer — which would erase the
			// roving tab stop. As a string it is always rendered (0 and -1).
			Raw: map[string]any{"tabindex": parseTabIndex},
			Aria: map[string]string{
				"checked":  boolString(parseItem.Selected),
				"disabled": boolString(parseItem.Disabled),
			},
		}, html.Text(parseItem.Label)))
	}
	parseOrientation := strings.ToLower(strings.TrimSpace(parseProps.Orientation))
	if parseOrientation != "vertical" {
		parseOrientation = "horizontal"
	}
	return html.Tag("div", html.Props{
		ID:   parseProps.ID,
		Role: "radiogroup",
		Aria: map[string]string{
			"label":       parseProps.Label,
			"orientation": parseOrientation,
		},
	}, parseRadios...)
}

// radioGroupTabStop returns the index of the single tabbable radio: the selected
// enabled option, else the first enabled option, else -1.
func radioGroupTabStop(parseItems []Item) int {
	parseFirstEnabled := -1
	for parseIndex, parseItem := range parseItems {
		if parseItem.Disabled {
			continue
		}
		if parseFirstEnabled < 0 {
			parseFirstEnabled = parseIndex
		}
		if parseItem.Selected {
			return parseIndex
		}
	}
	return parseFirstEnabled
}

func boolString(parseValue bool) string {
	if parseValue {
		return "true"
	}
	return "false"
}

func daysInMonth(parseYear int, parseMonth time.Month) int {
	return time.Date(parseYear, parseMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
