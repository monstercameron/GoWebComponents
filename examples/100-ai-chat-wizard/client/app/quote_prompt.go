//go:build js && wasm

package app

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

type quoteSelectionState struct {
	Visible bool
	Pending bool
	Text    string
	X       float64
	Y       float64
}

func quoteSelectionPrompt(state quoteSelectionState, onQuote, stopMouseUp ui.Handler) ui.Node {
	intl := i18n.UseI18n()
	if !state.Visible || state.Text == "" {
		return nil
	}

	style := map[string]string{
		"left": fmt.Sprintf("%.0fpx", state.X),
		"top":  fmt.Sprintf("%.0fpx", state.Y),
	}

	if state.Pending {
		return Div(
			ID(idQuoteSpinner),
			Class("quote-selection-ui quote-selection-spinner-card"),
			Style(style),
			OnMouseUp(stopMouseUp),
			Span(Class("quote-selection-spinner-dot")),
		)
	}

	return Button(
		ID(idQuotePrompt),
		Class("quote-selection-ui quote-selection-chip"),
		Style(style),
		OnMouseUp(stopMouseUp),
		OnClick(onQuote),
		Text(intl.T(chatI18nNamespace, "quote.prompt")),
	)
}
