//go:build js && wasm

package app

import (
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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
func parseUseQuoteSelection(parseApp ui.Reducer[appState, appAction]) quoteSelectionController {
	parseQuoteSelection := ui.UseState(quoteSelectionState{})
	parseQuoteSelectionVersion := ui.UseRef(0)
	parseQuoteSelectionTimer := ui.UseRef(interop.Timer{})

	parseCancelSelectionTimer := func() {
		parseTimer := parseQuoteSelectionTimer.Get()
		if parseErr := parseTimer.Cancel(); parseErr == nil {
			parseQuoteSelectionTimer.Set(interop.Timer{})
		}
	}

	parseCommitQuoteSelectionReady := func(parseVersion int, parseReadyState quoteSelectionState) {
		if parseQuoteSelectionVersion.Get() != parseVersion {
			return
		}
		parseReadyState.Pending = false
		parseQuoteSelection.Set(parseReadyState)
	}

	parseQueueQuoteSelection := func(parseSelection2 quoteSelectionAnchor) {
		parseNextVersion := parseQuoteSelectionVersion.Get() + 1
		parseQuoteSelectionVersion.Set(parseNextVersion)
		parsePendingState := quoteSelectionState{
			Visible: true,
			Pending: true,
			Text:    parseSelection2.Text,
			X:       parseSelection2.X,
			Y:       parseSelection2.Y,
		}
		parseQuoteSelection.Set(parsePendingState)
		parseCancelSelectionTimer()
		parseTimer2, parseErr2 := interop.ScheduleTimeout(500*time.Millisecond, func() {
			parseQuoteSelectionTimer.Set(interop.Timer{})
			parseCommitQuoteSelectionReady(parseNextVersion, parsePendingState)
		})
		if parseErr2 != nil {
			go func(parseVersion2 int, parseReadyState2 quoteSelectionState) {
				time.Sleep(500 * time.Millisecond)
				parseCommitQuoteSelectionReady(parseVersion2, parseReadyState2)
			}(parseNextVersion, parsePendingState)
			return
		}
		parseQuoteSelectionTimer.Set(parseTimer2)
	}

	parseDismissQuoteSelection := func() {
		parseQuoteSelectionVersion.Set(parseQuoteSelectionVersion.Get() + 1)
		parseCancelSelectionTimer()
		if parseQuoteSelection.Get().Visible {
			parseQuoteSelection.Set(quoteSelectionState{})
		}
	}

	handleSelectionMouse := ui.UseEvent(func(parseE ui.Event) {
		parseTarget := parseE.JSValue().Get("target")
		if parseSelectionTargetMatches(parseTarget, "#"+idChatInputWrap) || parseSelectionTargetMatches(parseTarget, "#"+idQuotePrompt) {
			return
		}
		parseSelection, parseOk := parseReadQuoteSelection(
			parseE.JSValue().Get("clientX").Float(),
			parseE.JSValue().Get("clientY").Float(),
		)
		if !parseOk {
			parseDismissQuoteSelection()
			return
		}
		parseQueueQuoteSelection(parseSelection)
	})

	parseStopPromptMouseUp := ui.UseEvent(func(parseE2 ui.Event) { parseE2.StopPropagation() })

	parseQuoteSelectedText := ui.UseEvent(func(parseE3 ui.Event) {
		parseE3.StopPropagation()
		parseCurrentQuoteSelection := parseQuoteSelection.Get()
		if strings.TrimSpace(parseCurrentQuoteSelection.Text) == "" {
			parseDismissQuoteSelection()
			return
		}
		parseApp.Dispatch(appAction{
			Type:      appActionSetInputText,
			InputText: formatQuotedInput(parseApp.Get().InputText, parseCurrentQuoteSelection.Text),
		})
		clearQuoteSelection()
		parseDismissQuoteSelection()
		parseScheduleFocusChatInput(focusDelay)
	})

	ui.UseEffect(func() func() {
		return func() {
			parseCancelSelectionTimer()
		}
	}, true)

	return quoteSelectionController{
		State:                parseQuoteSelection.Get(),
		HandleSelectionMouse: handleSelectionMouse,
		QuoteSelectedText:    parseQuoteSelectedText,
		StopPromptMouseUp:    parseStopPromptMouseUp,
	}
}
