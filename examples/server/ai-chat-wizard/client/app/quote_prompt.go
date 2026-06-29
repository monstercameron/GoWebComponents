//go:build js && wasm

package app

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type quoteSelectionState struct {
	Visible bool
	Pending bool
	Text    string
	X       float64
	Y       float64
}

func parseQuoteSelectionPrompt(parseState quoteSelectionState, parseOnQuote, parseStopMouseUp ui.Handler) ui.Node {
	parseIntl := i18n.UseI18n()
	if !parseState.Visible || parseState.Text == "" {
		return nil
	}

	parseStyle := map[string]string{
		"left": fmt.Sprintf("%.0fpx", parseState.X),
		"top":  fmt.Sprintf("%.0fpx", parseState.Y),
	}

	if parseState.Pending {
		return Div(
			ID(idQuoteSpinner),
			ClassStr("quote-selection-ui quote-selection-spinner-card"),
			Style(parseStyle),
			OnMouseUp(parseStopMouseUp),
			Span(ClassStr("quote-selection-spinner-dot")),
		)
	}

	return Button(
		ID(idQuotePrompt),
		ClassStr("quote-selection-ui quote-selection-chip"),
		Style(parseStyle),
		OnMouseUp(parseStopMouseUp),
		OnClick(parseOnQuote),
		Text(parseIntl.T(chatI18nNamespace, "quote.prompt")),
	)
}
