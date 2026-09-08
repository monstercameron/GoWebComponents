//go:build js && wasm

package app

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderMessageCodeBlock owns the copy button and cancels pending work on unmount.
func renderMessageCodeBlock(parseProps messageCodeProps) ui.Node {
	parseCopy := ui.UseTask(func(parseContext context.Context) (bool, error) {
		parseClipboard, parseErr := interop.GetClipboard()
		if parseErr != nil {
			return false, parseErr
		}
		parseErr = parseClipboard.WriteText(parseContext, parseProps.parseSource)
		return parseErr == nil, parseErr
	})
	parseClick := ui.UseEvent(func() { parseCopy.Start() })
	parseStatus := parseCopy.Get()
	parseLabel := "Copy"
	if parseStatus.Running {
		parseLabel = "Copying…"
	} else if parseStatus.Error != nil {
		parseLabel = "Copy failed — retry"
	} else if parseStatus.Ready {
		parseLabel = "Copied!"
	}
	return html.Div(html.Props{Class: "prose-pre-wrap", Raw: map[string]any{"data-gwc-code": "true"}},
		parseProps.parseElement,
		html.Button(html.Props{Type: "button", Class: "prose-copy-btn", Disabled: parseStatus.Running, OnClick: parseClick}, html.Text(parseLabel)),
	)
}
