//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

type messageListProps struct {
	Intl                    i18n.Runtime
	Messages                []message
	IsStreaming             bool
	UseMarkdownFallback     bool
	EditIdx                 int
	EditText                string
	StartEdit               ui.Handler
	CancelEdit              ui.Handler
	HandleEditChange        ui.Handler
	SubmitEdit              ui.Handler
	HandleEditKey           ui.Handler
	OnFork                  ui.Handler
	OnOpenCanvas            ui.Handler
	ToggleThoughtSection    ui.Handler
	ModelOptions            []modelOption
	DefaultModelID          string
	ThreadCostSummary       threadCostSummary
	UserInitials            string
	ExpandedThoughtSections map[string]bool
	ThoughtCacheByMessage   map[int]renderWorkerThoughtCacheEntry
	CanvasCacheByMessage    map[int]renderWorkerCanvasCacheEntry
	TTSAudio                ttsAudioController
	OnSpeechUpgrade         func()
	ShowScrollToBottom      bool
	ScrollToBottom          ui.Handler
	ApplyStarterPrompt      ui.Handler
	Journey                 chatJourneyState
	HandleSelectionMouse    ui.Handler
}

func parseMessageList(parseProps messageListProps) ui.Node {
	if len(parseProps.Messages) == 0 {
		return Div(
			Class("relative flex-1 min-h-0"),
			Div(
				ID(idMessageList),
				Class("chat-scrollbar chat-scrollbar--panel flex h-full flex-col items-center justify-center gap-4 overflow-y-auto"),
				parseEmptyState(parseProps.Journey, parseProps.ApplyStarterPrompt),
			),
		)
	}

	parseRows := MapKeyedIndexed(parseProps.Messages,
		func(parseIdx int, _ message) any { return parseIdx },
		func(parseIdx int, parseMsg message) ui.Node {
			isEditing := parseProps.EditIdx == parseIdx && parseMsg.Role == roleUser
			return parseMessageBubble(messageBubbleProps{
				Intl:                    parseProps.Intl,
				Message:                 parseMsg,
				Index:                   parseIdx,
				ActiveStream:            parseIdx == len(parseProps.Messages)-1 && parseProps.IsStreaming,
				IsEditing:               isEditing,
				EditValue:               parseProps.EditText,
				StartEdit:               parseProps.StartEdit,
				CancelEdit:              parseProps.CancelEdit,
				HandleEditChange:        parseProps.HandleEditChange,
				SubmitEdit:              parseProps.SubmitEdit,
				HandleEditKey:           parseProps.HandleEditKey,
				OnFork:                  parseProps.OnFork,
				OnOpenCanvas:            parseProps.OnOpenCanvas,
				ToggleThoughtSection:    parseProps.ToggleThoughtSection,
				ModelOptions:            parseProps.ModelOptions,
				DefaultModelID:          parseProps.DefaultModelID,
				ThreadCostSummary:       parseProps.ThreadCostSummary,
				UserInitials:            parseProps.UserInitials,
				UseMarkdownFallback:     parseProps.UseMarkdownFallback,
				ExpandedThoughtSections: parseProps.ExpandedThoughtSections,
				ThoughtCacheByMessage:   parseProps.ThoughtCacheByMessage,
				CanvasCacheByMessage:    parseProps.CanvasCacheByMessage,
				TTSAudio:                parseProps.TTSAudio,
				OnSpeechUpgrade:         parseProps.OnSpeechUpgrade,
			})
		})
	parseRows = append(parseRows, Div(ID(idScrollAnchor)))

	return Div(
		Class("relative flex-1 min-h-0"),
		Div(
			ID(idMessageList),
			Class("chat-scrollbar chat-scrollbar--panel h-full overflow-y-auto"),
			OnMouseUp(parseProps.HandleSelectionMouse),
			Div(ID(idThreadScreen), Class("thread-screen mx-auto w-full max-w-[46rem] px-4 pt-8 pb-6 flex flex-col gap-8"), parseRows),
		),
		// Show (not If) keeps the button mounted and toggles the hidden
		// attribute: visibility flips constantly while scrolling, and staying
		// mounted avoids re-creating the node (and its listener) every flip.
		Show(parseProps.ShowScrollToBottom,
			Button(
				ID(idScrollToBottomBtn),
				Class("absolute bottom-6 left-1/2 z-30 flex h-10 w-10 -translate-x-1/2 items-center justify-center rounded-full border border-white/[0.12] bg-[#16151f]/95 text-sm text-white/70 shadow-[0_8px_24px_rgba(0,0,0,0.45)] backdrop-blur-md transition-all duration-200 ease-out hover:-translate-x-1/2 hover:-translate-y-0.5 hover:border-[#8e7bff]/45 hover:text-white active:-translate-x-1/2 active:translate-y-0 active:scale-95"),
				FromProps(Props{Aria: map[string]string{"label": "Scroll to bottom"}}),
				OnClick(parseProps.ScrollToBottom),
				Text("\u2193"),
			),
		),
	)
}
