// Package erroroverlay renders an Elm-grade, in-page error overlay as a GoWebComponents
// component (FB4). It is the visual surface for a development error/diagnostic: a clear
// title, the message, an optional actionable hint ("here's likely how to fix it"), and an
// optional stack — rendered as a dismissible modal so a failure is legible in the page
// instead of only in the console. The structured diagnostics data already lives in the
// runtime/diagnostics packages; this is the dogfooded view over it.
package erroroverlay

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Props configures an ErrorOverlay.
type Props struct {
	// Title is the short heading (e.g. "Render error"). Defaults to "Error".
	Title string
	// Message is the primary error text.
	Message string
	// Hint is an optional actionable suggestion — the Elm-grade "try this" line.
	Hint string
	// Stack is optional call-stack / trace frames.
	Stack []string
	// OnDismiss, when set, wires the close control.
	OnDismiss func()
}

// FromError builds overlay props from a Go error (Title defaults applied by ErrorOverlay).
func FromError(parseErr error) Props {
	if parseErr == nil {
		return Props{}
	}
	return Props{Message: parseErr.Error()}
}

// ErrorOverlay renders the error as a dismissible, accessible modal overlay. Restyle via the
// gwc-error-overlay classes.
func ErrorOverlay(parseProps Props) ui.Node {
	parseTitle := parseProps.Title
	if parseTitle == "" {
		parseTitle = "Error"
	}

	parseChildren := []ui.Node{
		html.Tag("header", html.Props{Class: "gwc-error-overlay-header"},
			html.Tag("h2", html.Props{Class: "gwc-error-overlay-title"}, html.Text(parseTitle)),
			html.Button(html.Props{
				Class:   "gwc-error-overlay-dismiss",
				Type:    "button",
				OnClick: ui.UseEvent(func() { dismiss(parseProps) }),
				Aria:    map[string]string{"label": "Dismiss error"},
			}, html.Text("×")),
		),
		html.Tag("pre", html.Props{Class: "gwc-error-overlay-message"}, html.Text(parseProps.Message)),
	}

	if parseProps.Hint != "" {
		parseChildren = append(parseChildren, html.Tag("p", html.Props{Class: "gwc-error-overlay-hint"},
			html.Tag("strong", html.Props{}, html.Text("Try: ")),
			html.Text(parseProps.Hint),
		))
	}

	if len(parseProps.Stack) > 0 {
		parseFrames := make([]ui.Node, 0, len(parseProps.Stack))
		for _, parseFrame := range parseProps.Stack {
			parseFrames = append(parseFrames, html.Tag("li", html.Props{Class: "gwc-error-overlay-frame"}, html.Text(parseFrame)))
		}
		parseChildren = append(parseChildren, html.Tag("ol", html.Props{Class: "gwc-error-overlay-stack"}, parseFrames...))
	}

	return html.Div(html.Props{
		Class: "gwc-error-overlay",
		Role:  "alertdialog",
		Aria:  map[string]string{"modal": "true", "label": parseTitle},
	}, parseChildren...)
}

// dismiss invokes the dismiss callback when present.
func dismiss(parseProps Props) {
	if parseProps.OnDismiss != nil {
		parseProps.OnDismiss()
	}
}
