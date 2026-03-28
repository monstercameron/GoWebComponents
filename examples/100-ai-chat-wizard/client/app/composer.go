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
	OnInput            ui.Handler
	OnKeyDown          ui.Handler
	OnSend             ui.Handler
}

func parseInputArea(parseProps composerProps) ui.Node {
	isParseSendable := strings.TrimSpace(parseProps.Value) != "" && !parseProps.Disabled
	return Div(
		Class("shrink-0 pb-0.5 px-4"),
		Div(Class("max-w-[72rem] mx-auto"),
			Div(
				ID(idChatInputWrap),
				Class("relative flex items-end gap-3 bg-[#2f2f2f] border border-white/10 rounded-3xl px-4 py-3 shadow-lg cursor-text"),
				Tag("textarea",
					ID(idChatInput),
					Class("flex-1 bg-transparent resize-none text-[1.3125rem] text-white placeholder:text-white/40 focus:outline-none leading-relaxed min-h-[1.5rem]"),
					Placeholder(parseProps.Intl.T(chatI18nNamespace, "input.placeholder")),
					Value(parseProps.Value),
					OnInput(parseProps.OnInput),
					OnKeyDown(parseProps.OnKeyDown),
				),
				Div(Class("flex items-center gap-2 shrink-0 self-end"),
					Button(
						ID(idSendBtn),
						Class(ClassNames(
							"w-8 h-8 flex items-center justify-center rounded-full transition-colors shrink-0",
							When(isParseSendable, "bg-white text-black hover:bg-white/90"),
							When(!isParseSendable, "bg-white/10 text-white/30 cursor-not-allowed"),
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
