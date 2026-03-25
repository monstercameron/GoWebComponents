//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

type composerProps struct {
	Intl              i18n.Runtime
	Value             string
	Disabled          bool
	ThreadCostSummary threadCostSummary
	OnInput           ui.Handler
	OnKeyDown         ui.Handler
	OnSend            ui.Handler
}

func inputArea(props composerProps) ui.Node {
	sendable := strings.TrimSpace(props.Value) != "" && !props.Disabled
	return Div(
		Class("shrink-0 pb-0.5 px-4"),
		Div(Class("max-w-[72rem] mx-auto"),
			Div(
				ID(idChatInputWrap),
				Class("relative flex items-end gap-3 bg-[#2f2f2f] border border-white/10 rounded-3xl px-4 py-3 shadow-lg cursor-text"),
				Tag("textarea",
					ID(idChatInput),
					Class("flex-1 bg-transparent resize-none text-[1.3125rem] text-white placeholder:text-white/40 focus:outline-none leading-relaxed min-h-[1.5rem]"),
					Placeholder(props.Intl.T(chatI18nNamespace, "input.placeholder")),
					Value(props.Value),
					OnInput(props.OnInput),
					OnKeyDown(props.OnKeyDown),
				),
				Div(Class("flex items-center gap-2 shrink-0 self-end"),
					Button(
						ID(idSendBtn),
						Class(ClassNames(
							"w-8 h-8 flex items-center justify-center rounded-full transition-colors shrink-0",
							When(sendable, "bg-white text-black hover:bg-white/90"),
							When(!sendable, "bg-white/10 text-white/30 cursor-not-allowed"),
						)),
						DisabledIf(!sendable),
						OnClick(props.OnSend),
						Span(Class("text-sm leading-none select-none"), Text("\u2191")),
					),
					Tag("span",
						Class("text-white/25 text-base leading-none cursor-default select-none"),
						FromProps(Props{Raw: map[string]interface{}{"title": props.Intl.T(chatI18nNamespace, "input.disclaimer")}}),
						Text("\u24d8"),
					),
				),
			),
			If(props.ThreadCostSummary.HasAnyExactCosts && props.ThreadCostSummary.TotalCost > 0,
				P(Class("text-right text-base text-white/20 mt-1 pr-1"),
					Text(func() string {
						cost := formatCostUSD(props.ThreadCostSummary.TotalCost)
						if props.ThreadCostSummary.AllAssistantCostsExact {
							return props.Intl.T(chatI18nNamespace, "input.threadTotal", i18n.Arguments{"cost": cost})
						}
						return props.Intl.T(chatI18nNamespace, "input.threadTotalPartial", i18n.Arguments{"cost": cost})
					}()),
				),
			),
		),
	)
}
