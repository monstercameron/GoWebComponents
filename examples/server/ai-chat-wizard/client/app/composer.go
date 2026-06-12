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
		parseCueClass = "border-[#8e7bff]/40 bg-[#241f4a]/40 text-[#e2dcff]"
	case "success":
		parseCueClass = "border-[#5eead4]/30 bg-[#123a32]/35 text-[#d8fff1]"
	default:
		parseCueClass = "border-[#8e7bff]/20 bg-[#1a1a2c]/50 text-[#d6d5e8]"
	}
	return Div(
		Class("shrink-0 px-4 pb-3"),
		Div(Class("max-w-[46rem] mx-auto"),
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
				Class("relative flex items-end gap-3 rounded-[1.75rem] border border-white/[0.1] bg-[#17161f] px-5 py-3.5 shadow-[0_16px_48px_rgba(0,0,0,0.55),inset_0_1px_0_rgba(255,255,255,0.06)] cursor-text transition-[border-color,box-shadow] duration-200 focus-within:border-[#8e7bff]/50 focus-within:shadow-[0_0_0_3px_rgba(142,123,255,0.14),0_16px_48px_rgba(0,0,0,0.55)]"),
				Tag("textarea",
					ID(idChatInput),
					Class("min-h-[1.5rem] flex-1 resize-none bg-transparent text-[1.0625rem] leading-[1.7] text-[#ededf4] placeholder:text-[#7a7d99] focus:outline-none"),
					Placeholder(parseProps.Journey.parsePlaceholder),
					Value(parseProps.Value),
					OnInput(parseProps.OnInput),
					OnKeyDown(parseProps.OnKeyDown),
				),
				Div(Class("flex items-center gap-2 shrink-0 self-end"),
					Button(
						ID(idSendBtn),
						Class(ClassNames(
							"h-9 w-9 flex items-center justify-center rounded-full transition-all duration-200 shrink-0",
							When(isParseSendable, "bg-gradient-to-br from-[#8e7bff] to-[#6d5ce6] text-white shadow-[0_2px_12px_rgba(142,123,255,0.45)] hover:from-[#a99bff] hover:to-[#8e7bff] hover:shadow-[0_2px_16px_rgba(142,123,255,0.6)]"),
							When(!isParseSendable, "bg-[#1d1e2c] text-[#5c5e75] cursor-not-allowed"),
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
