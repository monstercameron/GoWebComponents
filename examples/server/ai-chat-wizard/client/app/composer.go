//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

type composerProps struct {
	Intl               i18n.Runtime
	Value              string
	Disabled           bool
	ThreadCostSummary  threadCostSummary
	AccountCostSummary accountCostSummary
	Journey            chatJourneyState
	CueTone            string
	CueTitle           string
	CueBody            string
	OnInput            ui.Handler
	OnKeyDown          ui.Handler
	OnSend             ui.Handler
}

func parseInputArea(parseProps composerProps) ui.Node {
	isParseSendable := strings.TrimSpace(parseProps.Value) != "" && !parseProps.Disabled
	parseCueClass := ""
	switch strings.TrimSpace(parseProps.CueTone) {
	case "warning":
		parseCueClass = "border-[#f6b84b]/35 bg-[#f6b84b]/10 text-[#ffe8b8]"
	case "upgrade":
		parseCueClass = "border-[#87d8ff]/38 bg-[#0f3a58]/34 text-[#d7f0ff]"
	case "success":
		parseCueClass = "border-[#8effd8]/34 bg-[#12382e]/34 text-[#d8fff1]"
	default:
		parseCueClass = "border-[#8effd8]/24 bg-[#0f2339]/40 text-[#d4e7fb]"
	}
	return Div(
		Class("shrink-0 pb-0.5 px-4"),
		Div(Class("max-w-[72rem] mx-auto"),
			If(strings.TrimSpace(parseProps.CueTitle) != "",
				Div(
					Class("mb-2 rounded-2xl border px-4 py-3"),
					Class(parseCueClass),
					Div(Class("text-[0.67rem] font-semibold uppercase tracking-[0.16em]"), Text(parseProps.CueTitle)),
					If(strings.TrimSpace(parseProps.CueBody) != "",
						Div(Class("mt-1 text-sm leading-5"), Text(parseProps.CueBody)),
					),
				),
			),
			Div(
				ID(idChatInputWrap),
				Class("relative flex items-end gap-3 rounded-[1.55rem] border border-white/8 bg-[#0c1929] px-4 py-3 shadow-[0_4px_16px_rgba(2,8,18,0.22)] backdrop-blur-sm cursor-text"),
				Tag("textarea",
					ID(idChatInput),
					Class("min-h-[1.5rem] flex-1 resize-none bg-transparent text-[1.3125rem] leading-relaxed text-[#e8f9ff] placeholder:text-[#9cb2c9] focus:outline-none"),
					Placeholder(parseProps.Journey.parsePlaceholder),
					Value(parseProps.Value),
					OnInput(parseProps.OnInput),
					OnKeyDown(parseProps.OnKeyDown),
				),
				Div(Class("flex items-center gap-2 shrink-0 self-end"),
					Button(
						ID(idSendBtn),
						Class(ClassNames(
							"h-9 w-9 flex items-center justify-center rounded-full transition-colors shrink-0",
							When(isParseSendable, "bg-[#00d9ff] text-[#05111d] hover:bg-[#33e3ff]"),
							When(!isParseSendable, "bg-[#14314a] text-[#7e99b5] cursor-not-allowed"),
						)),
						DisabledIf(!isParseSendable),
						OnClick(parseProps.OnSend),
						Span(Class("text-sm leading-none select-none"), Text("\u2191")),
					),
					Tag("span",
						Class("text-white/25 text-base leading-none cursor-default select-none"),
						FromProps(Props{Raw: map[string]interface{}{"title": parseProps.Intl.T(chatI18nNamespace, "input.disclaimer")}}),
						Text("\u24d8"),
					),
				),
			),
			renderComposerCostParallelRegion(parseProps),
		),
	)
}
