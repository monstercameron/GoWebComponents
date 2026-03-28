//go:build js && wasm

package app

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parseSidebar(parseConvList []convSummary, parseActiveConvID int64, isStreaming bool, parseUserName, parseUserInitials string, isOpen bool, parseOnNew, parseOnLoadConv, parseOnDeleteConv, parseOnEditName, parseOnToggle ui.Handler, isParseCanAccessAdmin bool, parseOnOpenAdmin ui.Handler) ui.Node {
	parseIntl := i18n.UseI18n()
	return Div(
		Class(ClassNames(
			"sidebar flex-col shrink-0 bg-[#0a1018] border-r border-[#8fffd8]/12 h-full overflow-hidden hidden md:flex",
			When(isOpen, "sidebar-open"),
			When(!isOpen, "sidebar-closed"),
		)),
		Div(Class("flex items-center gap-2 px-3 pt-4 pb-2"),
			Img(
				Src(brandLogoURL),
				Attr("alt", appBrandName),
				Class("h-9 w-auto shrink-0 object-contain flex-1 min-w-0"),
			),
			Button(
				Class("p-1.5 rounded-xl text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out shrink-0"),
				OnClick(parseOnToggle),
				parseSidebarToggleIcon(!isOpen),
			),
		),
		Div(Class("px-2 mt-2"),
			Button(
				Class(ClassNames(
					"flex items-center gap-2 w-full px-3 py-2 rounded-2xl text-sm text-[#dff5ff] border border-[#8fffd8]/16 bg-[#101a27] hover:bg-[#132235] transition-colors",
					When(isStreaming, "opacity-50 cursor-not-allowed"),
				)),
				DisabledIf(isStreaming),
				OnClick(parseOnNew),
				Span(Class("text-lg leading-none"), Text("+")),
				Text(parseIntl.T(chatI18nNamespace, "sidebar.newChat")),
			),
		),
		If(isParseCanAccessAdmin,
			Div(Class("px-2 mt-2"),
				Button(
					Class("flex w-full items-center justify-center rounded-2xl border border-[#8effd8]/30 bg-[#0f2339]/72 px-3 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-[#d6fff0] transition-colors hover:bg-[#143352]"),
					OnClick(parseOnOpenAdmin),
					Text("Admin dashboard"),
				),
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
								"conv-row group flex items-center rounded-2xl border text-sm transition-colors",
								When(isActive, "border-[#00d9ff]/40 bg-[#00d9ff]/12 text-[#f2fbff] font-medium"),
								When(!isActive, "border-transparent text-white/60 hover:border-[#8fffd8]/16 hover:bg-[#132235] hover:text-[#f2fbff]"),
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
								Class("shrink-0 p-2 mr-2 rounded-xl opacity-0 group-hover:opacity-100 text-white/35 hover:text-red-400 hover:bg-white/10 transition-all duration-200 ease-out"),
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
		Div(Class("mt-auto border-t border-white/[0.06]"),
			Button(
				Class("w-full flex items-center gap-3 px-3 py-3 hover:bg-white/[0.06] transition-colors text-left"),
				OnClick(parseOnEditName),
				Div(Class("h-8 w-8 rounded-full border border-[#00d9ff]/40 bg-gradient-to-br from-[#00d9ff]/28 to-[#3b82f6]/22 flex items-center justify-center shrink-0 text-xs font-semibold"),
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
