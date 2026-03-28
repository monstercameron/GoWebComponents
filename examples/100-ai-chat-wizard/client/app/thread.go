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
}

func parseMessageList(parseProps messageListProps) ui.Node {
	if len(parseProps.Messages) == 0 {
		return Div(
			Class("relative flex-1 min-h-0"),
			Div(
				ID(idMessageList),
				Class("chat-scrollbar chat-scrollbar--panel flex h-full flex-col items-center justify-center gap-4 overflow-y-auto"),
				parseEmptyState(),
			),
		)
	}

	parseRows := make([]ui.Node, 0, len(parseProps.Messages)+1)
	for parseIdx, parseMsg := range parseProps.Messages {
		isEditing := parseProps.EditIdx == parseIdx && parseMsg.Role == roleUser
		parseRows = append(parseRows, WithKey(
			parseMessageBubble(messageBubbleProps{
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
			}),
			parseIdx,
		))
	}
	parseRows = append(parseRows, Div(ID(idScrollAnchor)))

	return Div(
		Class("relative flex-1 min-h-0"),
		Div(
			ID(idMessageList),
			Class("chat-scrollbar chat-scrollbar--panel h-full overflow-y-auto"),
			Div(ID(idThreadScreen), Class("thread-screen max-w-[72rem] mx-auto px-4 py-4 flex flex-col gap-6"), parseRows),
		),
		If(parseProps.ShowScrollToBottom,
			Button(
				ID(idScrollToBottomBtn),
				Class("absolute bottom-4 left-1/2 z-30 flex h-12 w-12 -translate-x-1/2 items-center justify-center rounded-full border-2 border-white/50 bg-[#171717]/92 text-white shadow-[0_16px_36px_rgba(0,0,0,0.34)] backdrop-blur transition-all duration-200 ease-out hover:-translate-x-1/2 hover:-translate-y-1 hover:border-[#19c37d]/60 hover:bg-[#1d1d1d]/98 active:-translate-x-1/2 active:translate-y-0 active:scale-[0.97]"),
				FromProps(Props{Aria: map[string]string{"label": "Scroll to bottom"}}),
				OnClick(parseProps.ScrollToBottom),
				Text("\u2193"),
			),
		),
	)
}
