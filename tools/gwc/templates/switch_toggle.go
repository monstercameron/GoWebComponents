package components

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// SwitchProps configures a Switch.
type SwitchProps struct {
	// Label is the accessible name of the switch.
	Label string
	// On is the current state (controlled by the parent).
	On bool
	// OnToggle is called with the requested next state when the user activates the switch.
	OnToggle func(parseNext bool)
	// Disabled greys out and blocks interaction.
	Disabled bool
}

// Switch is an accessible on/off control implementing the WAI-ARIA switch pattern: a
// role="switch" button carrying aria-checked, toggled by click (and, natively, the
// space/enter a button already handles). It is controlled — render with On and handle
// OnToggle. Copied into your repo by `gwc add switch`; restyle via the gwc-switch classes.
func Switch(parseProps SwitchProps) ui.Node {
	parseToggle := ui.UseEvent(func() {
		if !parseProps.Disabled && parseProps.OnToggle != nil {
			parseProps.OnToggle(!parseProps.On)
		}
	})

	parseClass := "gwc-switch"
	if parseProps.On {
		parseClass += " gwc-switch-on"
	}

	return html.Button(html.Props{
		Class:    parseClass,
		Type:     "button",
		Role:     "switch",
		Disabled: parseProps.Disabled,
		OnClick:  parseToggle,
		Aria: map[string]string{
			"checked": switchBoolAttr(parseProps.On),
			"label":   parseProps.Label,
		},
	}, html.Tag("span", html.Props{Class: "gwc-switch-thumb"}))
}

// switchBoolAttr renders a Go bool as an ARIA "true"/"false" string.
func switchBoolAttr(parseValue bool) string {
	if parseValue {
		return "true"
	}
	return "false"
}
