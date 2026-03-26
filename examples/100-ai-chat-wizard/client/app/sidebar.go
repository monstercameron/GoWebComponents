//go:build js && wasm

package app

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parseSidebar(parseConvList []convSummary, parseActiveConvID int64, isStreaming bool, parseUserName, parseUserInitials string, isOpen bool, parseOnNew, parseOnLoadConv, parseOnDeleteConv, parseOnEditName, parseOnToggle ui.Handler) ui.Node {
	parseIntl := i18n.UseI18n()
	return Div(
		Class(ClassNames(
			"sidebar flex-col shrink-0 bg-[#171717] border-r border-white/5 h-full overflow-hidden hidden md:flex",
			When(isOpen, "sidebar-open"),
			When(!isOpen, "sidebar-closed"),
		)),
		Div(Class("flex items-center gap-2 px-3 pt-4 pb-2"),
			Div(Class("h-8 w-8 rounded-full bg-gradient-to-br from-[#19c37d] to-[#0ea47e] flex items-center justify-center shrink-0"),
				Text(assistantBadgeText),
			),
			Div(Class("flex flex-col min-w-0 flex-1"),
				Span(Class("font-semibold text-sm tracking-tight truncate"), Text(appBrandName)),
				Span(Class("text-[10px] text-white/30 uppercase tracking-[0.18em]"), Text(appVersion)),
			),
			Button(
				Class("p-1.5 rounded-lg text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out shrink-0"),
				OnClick(parseOnToggle),
				parseSidebarToggleIcon(!isOpen),
			),
		),
		Div(Class("px-2 mt-2"),
			Button(
				Class(ClassNames(
					"flex items-center gap-2 w-full px-3 py-2 rounded-lg text-sm text-white/80 hover:bg-white/10 transition-colors",
					When(isStreaming, "opacity-50 cursor-not-allowed"),
				)),
				DisabledIf(isStreaming),
				OnClick(parseOnNew),
				Span(Class("text-lg leading-none"), Text("+")),
				Text(parseIntl.T(chatI18nNamespace, "sidebar.newChat")),
			),
		),
		Div(
			ID(idConvList),
			Class("chat-scrollbar chat-scrollbar--sidebar flex-1 overflow-y-auto px-2 pt-3 flex flex-col gap-0.5"),
			IfElse(
				len(parseConvList) == 0,
				Span(Class("text-white/30 text-xs px-2"), Text(parseIntl.T(chatI18nNamespace, "sidebar.noConversations"))),
				Div(Class("flex flex-col gap-0.5"),
					Map(parseConvList, func(parseSummary convSummary) ui.Node {
						isActive := parseSummary.ID == parseActiveConvID
						parseIdStr := fmt.Sprintf("%d", parseSummary.ID)
						parsePreview := parseSummary.Preview
						if len(parsePreview) > sidebarPreviewLen {
							parsePreview = parsePreview[:sidebarPreviewLen] + "..."
						}
						if parsePreview == "" {
							parsePreview = parseIntl.T(chatI18nNamespace, "sidebar.emptyConversation")
						}
						return Div(
							Class(ClassNames(
								"conv-row group flex items-center rounded-lg text-sm transition-colors",
								When(isActive, "bg-white/[0.13] text-white font-medium"),
								When(!isActive, "text-white/50 hover:bg-white/[0.07] hover:text-white/85"),
								When(isStreaming, "pointer-events-none opacity-60"),
							)),
							Data(dataConvID, parseIdStr),
							Data(dataConvRow, "1"),
							Button(
								Class("flex-1 text-left px-3 py-2 truncate min-w-0"),
								Data(dataConvID, parseIdStr),
								OnClick(parseOnLoadConv),
								Text(parsePreview),
							),
							Button(
								Class("shrink-0 p-2 mr-2 rounded opacity-0 group-hover:opacity-100 text-white/35 hover:text-red-400 hover:bg-white/10 transition-all duration-200 ease-out"),
								FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "sidebar.deleteConversation")}}),
								Data(dataConvID, parseIdStr),
								OnClick(parseOnDeleteConv),
								parseConversationDeleteIcon(),
							),
						)
					}),
				),
			),
		),
		Div(Class("mt-auto border-t border-white/5"),
			Button(
				Class("w-full flex items-center gap-3 px-3 py-3 hover:bg-white/10 transition-colors text-left"),
				OnClick(parseOnEditName),
				Div(Class("h-8 w-8 rounded-full bg-gradient-to-br from-[#6366f1] to-[#4f46e5] flex items-center justify-center shrink-0 text-xs font-medium"),
					Text(parseUserInitials),
				),
				Div(Class("flex flex-col items-start min-w-0 flex-1 overflow-hidden"),
					Span(Class("text-sm text-white truncate w-full"), Text(parseUserName)),
					Span(Class("text-xs text-white/40"), Text(parseIntl.T(chatI18nNamespace, "sidebar.editSettings"))),
				),
				Span(Class("text-white/30 text-sm shrink-0"), Text("\u270e")),
			),
		),
	)
}
