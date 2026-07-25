//go:build js && wasm

package app

import (
	"fmt"
	"strings"
	"time"

	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/i18n"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// parseConversationRelativeTime renders a conversation start timestamp as a
// locale-aware relative phrase ("3 days ago") via i18n.FormatRelativeTime,
// returning "" when the wire timestamp doesn't parse.
func parseConversationRelativeTime(parseIntl i18n.Runtime, parseStartedAt string) string {
	parseTrimmed := strings.TrimSpace(parseStartedAt)
	if parseTrimmed == "" {
		return ""
	}
	for _, parseLayout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if parseParsed, parseErr := time.Parse(parseLayout, parseTrimmed); parseErr == nil {
			return i18n.FormatRelativeTime(parseIntl.Locale(), parseParsed, time.Now())
		}
	}
	return ""
}

func parseSidebar(parseConvList []convSummary, parseActiveConvID int64, isStreaming bool, parseUserName, parseUserInitials string, isOpen bool, parseOnNew ui.Handler, parseConversationList conversationListController, parseOnEditName, parseOnToggle ui.Handler, isParseCanAccessAdmin bool, parseOnOpenAdmin ui.Handler) ui.Node {
	parseIntl := i18n.UseI18n()
	parseGroups, parseUnfiledThreads := parseBuildSidebarThreadGroups(parseConvList, parseConversationList.Organization)
	return Div(
		ClassStr(ClassNames(
			"sidebar flex-col shrink-0 bg-[#0b0a12] border-r border-white/[0.04] h-full overflow-hidden hidden md:flex",
			When(isOpen, "sidebar-open"),
			When(!isOpen, "sidebar-closed"),
		)),
		Div(ClassStr("flex items-center gap-2 px-3 pt-4 pb-2"),
			Img(
				Src(brandLogoURL),
				Attr("alt", appBrandName),
				ClassStr("h-14 w-auto shrink-0 object-contain flex-1 min-w-0"),
			),
			Button(
				ClassStr("p-1.5 rounded-xl text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out shrink-0"),
				OnClick(parseOnToggle),
				parseSidebarToggleIcon(!isOpen),
			),
		),
		// workspace / account context strip
		Div(
			ClassStr("px-3 pb-2"),
			A(
				Href(settingsRoutePath+"?"+settingsPanelQueryKey+"="+settingsSectionBilling),
				ClassStr("block rounded-xl border border-white/[0.05] bg-white/[0.03] px-3 py-2 transition-colors hover:bg-white/[0.05]"),
				Div(ClassStr("text-[10px] uppercase tracking-[0.18em] text-white/25"), Text("Workspace")),
				Div(ClassStr("mt-0.5 truncate text-xs font-medium text-white/55"), Textf("%s's workspace", parseUserName)),
				Div(ClassStr("mt-1 text-[10px] text-white/30"), Text("Starter · Settings & billing →")),
			),
		),
		Div(ClassStr("px-2 mt-2"),
			Button(
				ClassStr(ClassNames(
					"flex items-center gap-2 w-full px-3 py-2 rounded-2xl text-sm text-[#b4b8d0] border border-white/8 bg-[#12121c] hover:bg-[#181830] transition-colors",
					When(isStreaming, "opacity-50 cursor-not-allowed"),
				)),
				DisabledIf(isStreaming),
				OnClick(parseOnNew),
				Span(ClassStr("text-lg leading-none"), Text("+")),
				Text(parseIntl.T(chatI18nNamespace, "sidebar.newChat")),
			),
		),
		If(isParseCanAccessAdmin,
			Div(ClassStr("px-2 mt-2"),
				Button(
					ID("open-admin-dashboard-btn"),
					ClassStr("flex w-full items-center justify-center rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-white/50 transition-colors hover:bg-white/8 hover:text-white/70"),
					OnClick(parseOnOpenAdmin),
					Text("Admin dashboard"),
				),
			),
		),
		Div(
			ID(idConvList),
			ClassStr("chat-scrollbar chat-scrollbar--sidebar flex-1 overflow-y-auto px-2 pt-3 flex flex-col gap-0.5"),
			Div(ClassStr("mb-2 flex items-center justify-between px-2"),
				Span(ClassStr("text-[10px] font-semibold uppercase tracking-[0.16em] text-white/30"), Text(parseIntl.T(chatI18nNamespace, "sidebar.threads"))),
				Button(
					ClassStr("rounded-lg border border-white/8 bg-white/[0.04] px-2 py-1 text-[10px] font-medium text-white/45 transition-colors hover:bg-white/[0.08] hover:text-white/70"),
					FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "sidebar.newFolder")}}),
					OnClick(parseConversationList.CreateFolder),
					Text(parseIntl.T(chatI18nNamespace, "sidebar.newFolderShort")),
				),
			),
			If(len(parseConvList) == 0,
				Span(ClassStr("text-white/30 text-xs px-2"), Text(parseIntl.T(chatI18nNamespace, "sidebar.noConversations"))),
			),
			MapKeyed(parseGroups,
				func(parseGroup sidebarThreadGroup) any { return parseGroup.Folder.ID },
				func(parseGroup sidebarThreadGroup) ui.Node {
					return renderSidebarFolderGroup(parseIntl, parseGroup, parseActiveConvID, isStreaming, parseConversationList)
				},
			),
			renderSidebarUnfiledGroup(parseIntl, parseUnfiledThreads, parseActiveConvID, isStreaming, parseConversationList),
		),
		Div(ClassStr("mt-auto border-t border-white/[0.06]"),
			Button(
				ID("open-settings-btn"),
				ClassStr("w-full flex items-center gap-3 px-3 py-3 hover:bg-white/[0.06] transition-colors text-left"),
				OnClick(parseOnEditName),
				Div(ClassStr("h-8 w-8 rounded-full border border-white/12 bg-white/6 flex items-center justify-center shrink-0 text-xs font-semibold"),
					Text(parseUserInitials),
				),
				Div(ClassStr("flex flex-col items-start min-w-0 flex-1 overflow-hidden"),
					Span(ClassStr("text-sm text-white truncate w-full"), Text(parseUserName)),
					Span(ClassStr("text-xs text-white/40"), Text(parseIntl.T(chatI18nNamespace, "sidebar.editSettings"))),
				),
				Span(ClassStr("text-white/30 text-sm shrink-0"), Text("\u270e")),
			),
		),
	)
}

func renderSidebarFolderGroup(parseIntl i18n.Runtime, parseGroup sidebarThreadGroup, parseActiveConvID int64, isStreaming bool, parseConversationList conversationListController) ui.Node {
	parseFolderID := parseGroup.Folder.ID
	return Div(
		ClassStr("mt-2 flex flex-col gap-0.5 rounded-xl border border-white/[0.04] bg-white/[0.02] p-1"),
		Attr("draggable", "true"),
		Data(dataSidebarDragKind, sidebarDragKindFolder),
		Data(dataSidebarFolderID, parseFolderID),
		OnDragStart(parseConversationList.DragStart),
		OnDragOver(parseConversationList.DragOver),
		OnDrop(parseConversationList.DropFolder),
		OnDragEnd(parseConversationList.DragEnd),
		Div(ClassStr("flex items-center gap-1 px-1 py-1"),
			Span(ClassStr("cursor-grab text-white/25"), Text("::")),
			Input(
				Type("text"),
				Value(parseGroup.Folder.Name),
				Data(dataSidebarFolderID, parseFolderID),
				ClassStr("min-w-0 flex-1 rounded-lg border border-transparent bg-transparent px-1.5 py-1 text-[11px] font-semibold uppercase tracking-[0.12em] text-white/38 outline-none transition-colors hover:border-white/8 hover:bg-white/[0.03] focus:border-[#8e7bff]/35 focus:bg-white/[0.05] focus:text-white/70"),
				FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "sidebar.renameFolder")}}),
				OnChange(parseConversationList.ChangeFolderName),
			),
			Span(ClassStr("rounded-full bg-white/[0.04] px-1.5 py-0.5 text-[10px] text-white/25"), Textf("%d", len(parseGroup.Threads))),
		),
		If(len(parseGroup.Threads) == 0,
			Div(
				ClassStr("rounded-lg border border-dashed border-white/[0.08] px-2 py-2 text-[11px] text-white/25"),
				Data(dataSidebarFolderID, parseFolderID),
				OnDragOver(parseConversationList.DragOver),
				OnDrop(parseConversationList.DropFolder),
				Text(parseIntl.T(chatI18nNamespace, "sidebar.dropThreadsHere")),
			),
		),
		MapKeyed(parseGroup.Threads,
			func(parseSummary convSummary) any { return parseSummary.ID },
			func(parseSummary convSummary) ui.Node {
				return renderSidebarThreadRow(parseIntl, parseSummary, parseActiveConvID, isStreaming, parseFolderID, parseConversationList)
			},
		),
	)
}

func renderSidebarUnfiledGroup(parseIntl i18n.Runtime, parseThreads []convSummary, parseActiveConvID int64, isStreaming bool, parseConversationList conversationListController) ui.Node {
	return Div(
		ClassStr("mt-2 flex flex-col gap-0.5"),
		Div(
			ClassStr("flex items-center justify-between rounded-lg border border-dashed border-white/[0.05] px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.14em] text-white/28"),
			OnDragOver(parseConversationList.DragOver),
			OnDrop(parseConversationList.DropUnfiled),
			Span(Text(parseIntl.T(chatI18nNamespace, "sidebar.unfiled"))),
			Span(ClassStr("rounded-full bg-white/[0.04] px-1.5 py-0.5"), Textf("%d", len(parseThreads))),
		),
		MapKeyed(parseThreads,
			func(parseSummary convSummary) any { return parseSummary.ID },
			func(parseSummary convSummary) ui.Node {
				return renderSidebarThreadRow(parseIntl, parseSummary, parseActiveConvID, isStreaming, "", parseConversationList)
			},
		),
	)
}

func renderSidebarThreadRow(parseIntl i18n.Runtime, parseSummary convSummary, parseActiveConvID int64, isStreaming bool, parseFolderID string, parseConversationList conversationListController) ui.Node {
	isActive := parseSummary.ID == parseActiveConvID
	parseIdStr := fmt.Sprintf("%d", parseSummary.ID)
	parsePreview := parseSummary.Preview
	if len(parsePreview) > sidebarPreviewLen {
		parsePreview = parsePreview[:sidebarPreviewLen] + "..."
	}
	if parsePreview == "" {
		parsePreview = parseIntl.T(chatI18nNamespace, "sidebar.emptyConversation")
	}
	parseRelativeTime := parseConversationRelativeTime(parseIntl, parseSummary.StartedAt)
	if parseConversationList.RenameTargetID == parseSummary.ID {
		return Div(
			ClassStr("conv-row flex items-center gap-1 rounded-2xl border border-[#8e7bff]/30 bg-[#8e7bff]/[0.08] px-2 py-1.5 text-sm"),
			Data(dataConvID, parseIdStr),
			Data(dataSidebarFolderID, parseFolderID),
			Input(
				Type("text"),
				Value(parseConversationList.RenameDraft),
				ClassStr("min-w-0 flex-1 rounded-xl border border-white/10 bg-[#0b0a12] px-2 py-1.5 text-xs text-white outline-none focus:border-[#8e7bff]/45"),
				FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "sidebar.renameConversation")}}),
				OnInput(parseConversationList.ChangeRenameDraft),
				OnKeyDown(parseConversationList.HandleRenameKey),
				OnBlur(parseConversationList.SaveRename),
			),
			Button(
				ClassStr("rounded-lg px-2 py-1 text-[11px] text-white/45 hover:bg-white/10 hover:text-white/75"),
				OnClick(parseConversationList.CancelRename),
				Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
			),
		)
	}
	return Div(
		ClassStr(ClassNames(
			"conv-row group flex items-center rounded-2xl border text-sm transition-colors",
			When(isActive, "border-[#8e7bff]/25 bg-[#8e7bff]/[0.08] text-[#f1eeff] font-medium"),
			When(!isActive, "border-transparent text-white/60 hover:border-white/10 hover:bg-[#181830] hover:text-[#f1eeff]"),
			When(isStreaming, "pointer-events-none opacity-60"),
		)),
		Attr("draggable", "true"),
		AttrIf(isActive, "aria-current", "page"),
		Data(dataConvID, parseIdStr),
		Data(dataConvRow, "1"),
		Data(dataSidebarDragKind, sidebarDragKindThread),
		Data(dataSidebarFolderID, parseFolderID),
		OnDragStart(parseConversationList.DragStart),
		OnDragOver(parseConversationList.DragOver),
		OnDrop(parseConversationList.DropThread),
		OnDragEnd(parseConversationList.DragEnd),
		Button(
			ClassStr("flex-1 text-left px-3 py-2 min-w-0"),
			Data(dataConvID, parseIdStr),
			OnClick(parseConversationList.Load),
			Div(ClassStr("truncate"), Text(parsePreview)),
			If(parseRelativeTime != "",
				Div(ClassStr("mt-0.5 truncate text-[10px] text-white/25 transition-colors group-hover:text-white/40"), Text(parseRelativeTime)),
			),
		),
		Button(
			ClassStr("shrink-0 p-2 rounded-xl opacity-0 group-hover:opacity-100 text-white/35 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out"),
			FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "sidebar.renameConversation")}}),
			Data(dataConvID, parseIdStr),
			OnClick(parseConversationList.StartRename),
			Text("\u270e"),
		),
		Button(
			ClassStr("shrink-0 p-2 mr-1 rounded-xl opacity-0 group-hover:opacity-100 text-white/35 hover:text-red-400 hover:bg-white/10 transition-all duration-200 ease-out"),
			FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "sidebar.deleteConversation")}}),
			Data(dataConvID, parseIdStr),
			OnClick(parseConversationList.RequestDelete),
			parseConversationDeleteIcon(),
		),
	)
}
