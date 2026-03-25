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
}

func messageList(props messageListProps) ui.Node {
	if len(props.Messages) == 0 {
		return Div(
			ID(idMessageList),
			Class("chat-scrollbar chat-scrollbar--panel flex-1 overflow-y-auto flex flex-col items-center justify-center gap-4"),
			emptyState(),
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
			}),
			idx,
		))
	}
	rows = append(rows, Div(ID(idScrollAnchor)))

	return Div(
		ID(idMessageList),
		Class("chat-scrollbar chat-scrollbar--panel flex-1 overflow-y-auto"),
		Div(ID(idThreadScreen), Class("thread-screen max-w-[72rem] mx-auto px-4 py-8 flex flex-col gap-6"), rows),
	)
}
