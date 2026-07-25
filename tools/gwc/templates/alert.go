package components

import (
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// AlertProps configures an Alert.
type AlertProps struct {
	// Variant styles the alert (info/success/warning/danger); it is reflected as a class
	// and does not change semantics.
	Variant string
	// Title is an optional bold heading.
	Title string
	// Message is the alert body.
	Message string
	// Assertive escalates the live-region politeness for urgent alerts.
	Assertive bool
}

// Alert is an accessible status message implementing the WAI-ARIA alert pattern: a
// role="alert" region (with an aria-live politeness the screen reader announces) carrying a
// short, restyleable message. Copied into your repo by `gwc add alert` — you own it.
func Alert(parseProps AlertProps) ui.Node {
	parseLive := "polite"
	if parseProps.Assertive {
		parseLive = "assertive"
	}
	parseClass := "gwc-alert"
	if parseProps.Variant != "" {
		parseClass += " gwc-alert-" + parseProps.Variant
	}

	parseChildren := make([]ui.Node, 0, 2)
	if parseProps.Title != "" {
		parseChildren = append(parseChildren, html.Tag("strong", html.Props{Class: "gwc-alert-title"}, html.Text(parseProps.Title)))
	}
	parseChildren = append(parseChildren, html.Tag("span", html.Props{Class: "gwc-alert-message"}, html.Text(parseProps.Message)))

	return html.Tag("div", html.Props{
		Class: parseClass,
		Role:  "alert",
		Aria:  map[string]string{"live": parseLive},
	}, parseChildren...)
}
