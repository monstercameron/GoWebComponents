package a11y

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// DialogButton is one action button in a dialog.
type DialogButton struct {
	ID      string
	Label   string
	OnClick func()
	// Primary marks the default/confirm action.
	Primary bool
}

// AlertDialogProps configures a headless WAI-ARIA alert dialog — the themeable,
// e2e-drivable replacement for window.confirm/alert (G18).
type AlertDialogProps struct {
	ID        string
	Title     string
	Message   string
	Buttons   []DialogButton
	OnDismiss func()
}

// AlertDialog renders a headless alertdialog: role="alertdialog", aria-modal,
// aria-labelledby/aria-describedby wired to the title/message, and action
// buttons. Drive open/close from component state and pair with ui.AccessibleOverlay
// (G10) for focus trap + Esc.
func AlertDialog(parseProps AlertDialogProps) ui.Node {
	parseTitleID := parseProps.ID + "-title"
	parseDescID := parseProps.ID + "-desc"

	parseButtons := make([]ui.Node, 0, len(parseProps.Buttons))
	for _, parseBtn := range parseProps.Buttons {
		// html.OnClick wraps a plain func via toHandler — NOT a hook — so this is
		// safe inside the loop (no rules-of-hooks violation; verified by hookcheck).
		parseOptions := []html.PropOption{
			html.ID(parseBtn.ID),
			html.Type("button"),
			html.OnClick(parseBtn.OnClick),
		}
		if parseBtn.Primary {
			parseOptions = append(parseOptions, html.Aria("keyshortcuts", "Enter"))
		}
		parseButtons = append(parseButtons, html.Button(html.PropsOf(parseOptions...), html.Text(parseBtn.Label)))
	}

	return html.Tag("div", html.Props{
		ID:   parseProps.ID,
		Role: "alertdialog",
		Aria: map[string]string{
			"modal":       "true",
			"labelledby":  strings.TrimSpace(parseTitleID),
			"describedby": strings.TrimSpace(parseDescID),
		},
	},
		html.Tag("h2", html.Props{ID: parseTitleID}, html.Text(parseProps.Title)),
		html.Tag("p", html.Props{ID: parseDescID}, html.Text(parseProps.Message)),
		html.Tag("div", html.Props{Role: "group"}, parseButtons...),
	)
}
