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
				Class("h-14 w-auto shrink-0 object-contain flex-1 min-w-0"),
			),
			Button(
				Class("p-1.5 rounded-xl text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out shrink-0"),
				OnClick(parseOnToggle),
				parseSidebarToggleIcon(!isOpen),
			),
		),
		// workspace / account context strip
		Div(
			Class("px-3 pb-2"),
			A(
				Href(settingsRoutePath+"?"+settingsPanelQueryKey+"="+settingsSectionBilling),
				Class("block rounded-xl border border-white/[0.05] bg-white/[0.03] px-3 py-2 transition-colors hover:bg-white/[0.05]"),
				Div(Class("text-[10px] uppercase tracking-[0.18em] text-white/25"), Text("Workspace")),
				Div(Class("mt-0.5 truncate text-xs font-medium text-white/55"), Text(parseUserName+"'s workspace")),
				Div(Class("mt-1 text-[10px] text-white/30"), Text("Starter · Settings & billing →")),
			),
		),
		Div(Class("px-2 mt-2"),
			Button(
				Class(ClassNames(
					"flex items-center gap-2 w-full px-3 py-2 rounded-2xl text-sm text-[#b8c2d9] border border-white/8 bg-[#101a27] hover:bg-[#132235] transition-colors",
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
					ID("open-admin-dashboard-btn"),
					Class("flex w-full items-center justify-center rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-white/50 transition-colors hover:bg-white/8 hover:text-white/70"),
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
								When(isActive, "border-white/20 bg-white/8 text-[#f2fbff] font-medium"),
								When(!isActive, "border-transparent text-white/60 hover:border-white/10 hover:bg-[#132235] hover:text-[#f2fbff]"),
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
				ID("open-settings-btn"),
				Class("w-full flex items-center gap-3 px-3 py-3 hover:bg-white/[0.06] transition-colors text-left"),
				OnClick(parseOnEditName),
				Div(Class("h-8 w-8 rounded-full border border-white/12 bg-white/6 flex items-center justify-center shrink-0 text-xs font-semibold"),
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
