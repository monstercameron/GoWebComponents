//go:build js && wasm

package app

import (
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

// quoteSelectionController owns text-selection detection and quote insertion.
//
// Keeping this separate removes the popup timing and selection bookkeeping
// from App(), which only needs the current prompt state and handlers.
type quoteSelectionController struct {
	State                quoteSelectionState
	HandleSelectionMouse ui.Handler
	QuoteSelectedText    ui.Handler
	StopPromptMouseUp    ui.Handler
}

// useQuoteSelection manages the delayed quote prompt shown when the user
// selects text on the page and chooses to insert it into the composer.
func useQuoteSelection(app ui.Reducer[appState, appAction]) quoteSelectionController {
	quoteSelection := ui.UseState(quoteSelectionState{})
	quoteSelectionVersion := ui.UseRef(0)

	queueQuoteSelection := func(selection quoteSelectionAnchor) {
		nextVersion := quoteSelectionVersion.Get() + 1
		quoteSelectionVersion.Set(nextVersion)
		pendingState := quoteSelectionState{
			Visible: true,
			Pending: true,
			Text:    selection.Text,
			X:       selection.X,
			Y:       selection.Y,
		}
		quoteSelection.Set(pendingState)
		go func(version int, readyState quoteSelectionState) {
			time.Sleep(500 * time.Millisecond)
			if quoteSelectionVersion.Get() != version {
				return
			}
			readyState.Pending = false
			quoteSelection.Set(readyState)
		}(nextVersion, pendingState)
	}

	dismissQuoteSelection := func() {
		quoteSelectionVersion.Set(quoteSelectionVersion.Get() + 1)
		if quoteSelection.Get().Visible {
			quoteSelection.Set(quoteSelectionState{})
		}
	}

	handleSelectionMouse := ui.UseEvent(func(e ui.Event) {
		target := e.JSValue().Get("target")
		if selectionTargetMatches(target, "#"+idChatInputWrap) || selectionTargetMatches(target, "#"+idQuotePrompt) {
			return
		}
		selection, ok := readQuoteSelection(
			e.JSValue().Get("clientX").Float(),
			e.JSValue().Get("clientY").Float(),
		)
		if !ok {
			dismissQuoteSelection()
			return
		}
		queueQuoteSelection(selection)
	})

	stopPromptMouseUp := ui.UseEvent(func(e ui.Event) { e.StopPropagation() })

	quoteSelectedText := ui.UseEvent(func(e ui.Event) {
		e.StopPropagation()
		currentQuoteSelection := quoteSelection.Get()
		if strings.TrimSpace(currentQuoteSelection.Text) == "" {
			dismissQuoteSelection()
			return
		}
		app.Dispatch(appAction{
			Type:      appActionSetInputText,
			InputText: formatQuotedInput(app.Get().InputText, currentQuoteSelection.Text),
		})
		clearQuoteSelection()
		dismissQuoteSelection()
		go func() {
			time.Sleep(focusDelay)
			focusChatInput()
		}()
	})

	return quoteSelectionController{
		State:                quoteSelection.Get(),
		HandleSelectionMouse: handleSelectionMouse,
		QuoteSelectedText:    quoteSelectedText,
		StopPromptMouseUp:    stopPromptMouseUp,
	}
}
