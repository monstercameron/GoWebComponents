//go:build js && wasm

package app

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func renderEditableUserMemories(parseIntl i18n.Runtime, parseMemories []editableUserMemory, parseSettings profileSettingsController) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseMemories))
	for parseIndex, parseMemory := range parseMemories {
		parseIndexText := strconv.Itoa(parseIndex)
		parseManagedName := isManagedUserNameMemory(parseMemory)
		parseNodes = append(parseNodes, Div(
			ClassStr("group rounded-xl border border-white/10 bg-white/[0.03] overflow-hidden"),
			// Compact header: category badge + icon actions
			Div(ClassStr("flex items-center gap-2 border-b border-white/8 px-3 py-2"),
				Span(ClassStr("flex-1 text-[11px] font-semibold uppercase tracking-wide text-white/45 truncate"),
					Text(func() string {
						if parseManagedName {
							return parseIntl.T(chatI18nNamespace, "modal.displayName")
						}
						return parseIntl.T(chatI18nNamespace, "modal.memoryItem")
					}()),
				),
				If(!parseManagedName,
					Button(
						ClassStr("shrink-0 rounded-lg p-1.5 text-white/30 transition-colors hover:bg-red-500/20 hover:text-red-300"),
						FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "modal.memoryDelete")}}),
						Data(dataMemoryIndex, parseIndexText),
						OnClick(parseSettings.DeleteMemory),
						parseConversationDeleteIcon(),
					),
				),
			),
			// Compact fields
			Div(ClassStr("flex flex-col gap-2 p-3"),
				Input(
					ClassStr("w-full rounded-lg border border-white/15 bg-[#3a3a3a] px-3 py-2 text-sm text-white placeholder:text-white/35 focus:outline-none focus:border-white/30"),
					Placeholder(parseIntl.T(chatI18nNamespace, "modal.memoryCategoryPlaceholder")),
					Value(parseMemory.Category),
					DisabledIf(parseManagedName),
					Data(dataMemoryIndex, parseIndexText),
					Data(dataMemoryField, "category"),
					OnInput(parseSettings.HandleMemoryChange),
				),
				Input(
					ClassStr("w-full rounded-lg border border-white/15 bg-[#3a3a3a] px-3 py-2 text-sm text-white placeholder:text-white/35 focus:outline-none focus:border-white/30"),
					Placeholder(parseIntl.T(chatI18nNamespace, "modal.memorySummaryPlaceholder")),
					Value(parseMemory.Summary),
					DisabledIf(parseManagedName),
					Data(dataMemoryIndex, parseIndexText),
					Data(dataMemoryField, "summary"),
					OnInput(parseSettings.HandleMemoryChange),
				),
				Tag("textarea",
					ClassStr("w-full min-h-[3rem] rounded-lg border border-white/15 bg-[#3a3a3a] px-3 py-2 text-sm text-white placeholder:text-white/30 focus:outline-none focus:border-white/30 resize-y"),
					Placeholder(parseIntl.T(chatI18nNamespace, "modal.memoryDetailPlaceholder")),
					Value(parseMemory.Detail),
					DisabledIf(parseManagedName),
					Data(dataMemoryIndex, parseIndexText),
					Data(dataMemoryField, "detail"),
					OnInput(parseSettings.HandleMemoryChange),
				),
				Tag("textarea",
					ClassStr("w-full min-h-[2.5rem] rounded-lg border border-white/10 bg-[#3a3a3a]/60 px-3 py-2 text-xs text-white/55 placeholder:text-white/25 focus:outline-none focus:border-white/25 resize-y"),
					Placeholder(parseIntl.T(chatI18nNamespace, "modal.memoryReasonPlaceholder")),
					Value(parseMemory.RubricReason),
					DisabledIf(parseManagedName),
					Data(dataMemoryIndex, parseIndexText),
					Data(dataMemoryField, "rubric_reason"),
					OnInput(parseSettings.HandleMemoryChange),
				),
			),
		))
	}
	return parseNodes
}
