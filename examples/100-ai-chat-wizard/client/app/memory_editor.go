//go:build js && wasm

package app

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func renderEditableUserMemories(parseIntl i18n.Runtime, parseMemories []editableUserMemory, parseSettings profileSettingsController) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseMemories))
	for parseIndex, parseMemory := range parseMemories {
		parseIndexText := strconv.Itoa(parseIndex)
		parseManagedName := parseManagedUserNameMemory(parseMemory)
		parseNodes = append(parseNodes, Div(
			Class("flex flex-col gap-2 rounded-xl border border-white/10 bg-white/[0.03] p-3"),
			Div(Class("flex items-center justify-between gap-3"),
				Span(Class("text-xs font-medium uppercase tracking-wide text-white/45"), Text(func() string {
					if parseManagedName {
						return parseIntl.T(chatI18nNamespace, "modal.displayName")
					}
					return parseIntl.T(chatI18nNamespace, "modal.memoryItem")
				}())),
				If(!parseManagedName,
					Button(
						Class("px-2.5 py-1 text-xs rounded-lg bg-red-500/15 text-red-200 hover:bg-red-500/25 transition-colors"),
						Data(dataMemoryIndex, parseIndexText),
						OnClick(parseSettings.DeleteMemory),
						Text(parseIntl.T(chatI18nNamespace, "modal.memoryDelete")),
					),
				),
			),
			Input(
				Class("w-full bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40"),
				Placeholder(parseIntl.T(chatI18nNamespace, "modal.memoryCategoryPlaceholder")),
				Value(parseMemory.Category),
				DisabledIf(parseManagedName),
				Data(dataMemoryIndex, parseIndexText),
				Data(dataMemoryField, "category"),
				OnInput(parseSettings.HandleMemoryChange),
			),
			Input(
				Class("w-full bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40"),
				Placeholder(parseIntl.T(chatI18nNamespace, "modal.memorySummaryPlaceholder")),
				Value(parseMemory.Summary),
				DisabledIf(parseManagedName),
				Data(dataMemoryIndex, parseIndexText),
				Data(dataMemoryField, "summary"),
				OnInput(parseSettings.HandleMemoryChange),
			),
			Tag("textarea",
				Class("w-full min-h-[5rem] bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40 resize-y"),
				Placeholder(parseIntl.T(chatI18nNamespace, "modal.memoryDetailPlaceholder")),
				Value(parseMemory.Detail),
				DisabledIf(parseManagedName),
				Data(dataMemoryIndex, parseIndexText),
				Data(dataMemoryField, "detail"),
				OnInput(parseSettings.HandleMemoryChange),
			),
			Tag("textarea",
				Class("w-full min-h-[4rem] bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40 resize-y"),
				Placeholder(parseIntl.T(chatI18nNamespace, "modal.memoryReasonPlaceholder")),
				Value(parseMemory.RubricReason),
				DisabledIf(parseManagedName),
				Data(dataMemoryIndex, parseIndexText),
				Data(dataMemoryField, "rubric_reason"),
				OnInput(parseSettings.HandleMemoryChange),
			),
		))
	}
	return parseNodes
}
