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
	TTSAudio                ttsAudioController
	OnSpeechUpgrade         func()
	ShowScrollToBottom      bool
	ScrollToBottom          ui.Handler
}

func messageList(props messageListProps) ui.Node {
	if len(props.Messages) == 0 {
		return Div(
			Class("relative flex-1 min-h-0"),
			Div(
				ID(idMessageList),
				Class("chat-scrollbar chat-scrollbar--panel flex h-full flex-col items-center justify-center gap-4 overflow-y-auto"),
				emptyState(),
			),
		)
	}

	rows := make([]ui.Node, 0, len(props.Messages)+1)
	for idx, msg := range props.Messages {
		isEditing := props.EditIdx == idx && msg.Role == roleUser
		rows = append(rows, WithKey(
			messageBubble(messageBubbleProps{
				Intl:                    props.Intl,
				Message:                 msg,
				Index:                   idx,
				ActiveStream:            idx == len(props.Messages)-1 && props.IsStreaming,
				IsEditing:               isEditing,
				EditValue:               props.EditText,
				StartEdit:               props.StartEdit,
				CancelEdit:              props.CancelEdit,
				HandleEditChange:        props.HandleEditChange,
				SubmitEdit:              props.SubmitEdit,
				HandleEditKey:           props.HandleEditKey,
				OnFork:                  props.OnFork,
				OnOpenCanvas:            props.OnOpenCanvas,
				ToggleThoughtSection:    props.ToggleThoughtSection,
				ModelOptions:            props.ModelOptions,
				DefaultModelID:          props.DefaultModelID,
				ThreadCostSummary:       props.ThreadCostSummary,
				UserInitials:            props.UserInitials,
				UseMarkdownFallback:     props.UseMarkdownFallback,
				ExpandedThoughtSections: props.ExpandedThoughtSections,
				TTSAudio:                props.TTSAudio,
				OnSpeechUpgrade:         props.OnSpeechUpgrade,
			}),
			idx,
		))
	}
	rows = append(rows, Div(ID(idScrollAnchor)))

	return Div(
		Class("relative flex-1 min-h-0"),
		Div(
			ID(idMessageList),
			Class("chat-scrollbar chat-scrollbar--panel h-full overflow-y-auto"),
			Div(ID(idThreadScreen), Class("thread-screen max-w-[72rem] mx-auto px-4 py-4 flex flex-col gap-6"), rows),
		),
		If(props.ShowScrollToBottom,
			Button(
				ID(idScrollToBottomBtn),
				Class("fixed bottom-[6.25rem] left-1/2 z-30 flex h-12 w-12 -translate-x-1/2 items-center justify-center rounded-full border-2 border-white/50 bg-[#171717]/92 text-white shadow-[0_16px_36px_rgba(0,0,0,0.34)] backdrop-blur transition-all duration-200 ease-out hover:-translate-x-1/2 hover:-translate-y-1 hover:border-[#19c37d]/60 hover:bg-[#1d1d1d]/98 active:-translate-x-1/2 active:translate-y-0 active:scale-[0.97]"),
				FromProps(Props{Aria: map[string]string{"label": "Scroll to bottom"}}),
				OnClick(props.ScrollToBottom),
				Text("\u2193"),
			),
		),
	)
}
