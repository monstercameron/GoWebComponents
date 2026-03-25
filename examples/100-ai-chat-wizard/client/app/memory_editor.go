//go:build js && wasm

package app

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func renderEditableUserMemories(intl i18n.Runtime, memories []editableUserMemory, settings profileSettingsController) []ui.Node {
	nodes := make([]ui.Node, 0, len(memories))
	for index, memory := range memories {
		indexText := strconv.Itoa(index)
		managedName := isManagedUserNameMemory(memory)
		nodes = append(nodes, Div(
			Class("flex flex-col gap-2 rounded-xl border border-white/10 bg-white/[0.03] p-3"),
			Div(Class("flex items-center justify-between gap-3"),
				Span(Class("text-xs font-medium uppercase tracking-wide text-white/45"), Text(func() string {
					if managedName {
						return intl.T(chatI18nNamespace, "modal.displayName")
					}
					return intl.T(chatI18nNamespace, "modal.memoryItem")
				}())),
				If(!managedName,
					Button(
						Class("px-2.5 py-1 text-xs rounded-lg bg-red-500/15 text-red-200 hover:bg-red-500/25 transition-colors"),
						Data(dataMemoryIndex, indexText),
						OnClick(settings.DeleteMemory),
						Text(intl.T(chatI18nNamespace, "modal.memoryDelete")),
					),
				),
			),
			Input(
				Class("w-full bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40"),
				Placeholder(intl.T(chatI18nNamespace, "modal.memoryCategoryPlaceholder")),
				Value(memory.Category),
				DisabledIf(managedName),
				Data(dataMemoryIndex, indexText),
				Data(dataMemoryField, "category"),
				OnInput(settings.HandleMemoryChange),
			),
			Input(
				Class("w-full bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40"),
				Placeholder(intl.T(chatI18nNamespace, "modal.memorySummaryPlaceholder")),
				Value(memory.Summary),
				DisabledIf(managedName),
				Data(dataMemoryIndex, indexText),
				Data(dataMemoryField, "summary"),
				OnInput(settings.HandleMemoryChange),
			),
			Tag("textarea",
				Class("w-full min-h-[5rem] bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40 resize-y"),
				Placeholder(intl.T(chatI18nNamespace, "modal.memoryDetailPlaceholder")),
				Value(memory.Detail),
				DisabledIf(managedName),
				Data(dataMemoryIndex, indexText),
				Data(dataMemoryField, "detail"),
				OnInput(settings.HandleMemoryChange),
			),
			Tag("textarea",
				Class("w-full min-h-[4rem] bg-[#3a3a3a] text-white text-sm rounded-xl px-4 py-3 focus:outline-none border border-white/20 placeholder:text-white/40 resize-y"),
				Placeholder(intl.T(chatI18nNamespace, "modal.memoryReasonPlaceholder")),
				Value(memory.RubricReason),
				DisabledIf(managedName),
				Data(dataMemoryIndex, indexText),
				Data(dataMemoryField, "rubric_reason"),
				OnInput(settings.HandleMemoryChange),
			),
		))
	}
	return nodes
}
