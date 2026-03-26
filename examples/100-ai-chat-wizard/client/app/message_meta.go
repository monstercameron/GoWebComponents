//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

type assistantMessageMetaProps struct {
	Intl              i18n.Runtime
	Message           message
	Index             int
	CanvasArtifacts   []canvasArtifact
	ModelOptions      []modelOption
	ThreadCostSummary threadCostSummary
	OnFork            ui.Handler
	OnOpenCanvas      ui.Handler
	TTSStatus         ttsClipStatus
	OnTTSToggle       func()
	OnTTSStop         func()
	OnSpeechUpgrade   func()
}

func parseAssistantMessageMetaRow(parseProps assistantMessageMetaProps) ui.Node {
	parseIndexText := strconv.Itoa(parseProps.Index)
	parseMessageCost, hasExactCost := parseProps.ThreadCostSummary.AssistantMessageCosts[parseProps.Index]
	hasPerformanceStats := parseProps.Message.Tokens > 0

	return Div(Class("mt-1 flex flex-col gap-1"),
		Div(Class("flex items-center gap-2"),
			If(hasExactCost || hasPerformanceStats,
				Div(Class("flex flex-wrap items-center gap-1.5 px-2 py-1 text-xs text-white/45 select-none"),
					If(hasExactCost,
						Span(Text(strings.TrimSpace(parseModelLabelForID(parseMessageCost.ModelID, parseProps.ModelOptions)+" "+formatCostUSD(parseMessageCost.Cost)))),
					),
					If(hasExactCost && hasPerformanceStats,
						Span(Class("text-white/15"), Text(".")),
					),
					If(hasPerformanceStats,
						Span(Textf("TTFT %.2fs", parseProps.Message.TTFT)),
					),
					If(hasPerformanceStats,
						Span(Class("text-white/15"), Text(".")),
					),
					If(hasPerformanceStats,
						Span(Textf("%.1f t/s", parseProps.Message.TKPS)),
					),
					If(hasPerformanceStats,
						Span(Class("text-white/15"), Text(".")),
					),
					If(hasPerformanceStats,
						Span(Textf("%d tok", parseProps.Message.Tokens)),
					),
				),
			),
			If(len(parseProps.CanvasArtifacts) > 0,
				Div(Class("flex flex-wrap items-center gap-1"),
					Map(parseProps.CanvasArtifacts, func(parseArtifact canvasArtifact) ui.Node {
						return Button(
							Class("flex items-center gap-1 rounded-lg border border-[#19c37d]/22 px-2 py-1 text-xs text-[#b8ffe0] hover:bg-[#19c37d]/12 transition-colors"),
							Data(dataCanvasID, parseArtifact.ID),
							OnClick(parseProps.OnOpenCanvas),
							Text("Preview: "+parseArtifact.Label),
						)
					}),
				),
			),
			Div(Class("ml-auto flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"),
				Button(
					Class(ClassNames(
						"flex items-center gap-1 px-2 py-1 text-xs rounded-lg transition-colors",
						When(parseProps.TTSStatus.IsLoading, "bg-white/10 text-white/50"),
						When(parseProps.TTSStatus.IsPlaying, "bg-[#19c37d]/18 text-[#b8ffe0] hover:bg-[#19c37d]/24"),
						When(!parseProps.TTSStatus.IsLoading && !parseProps.TTSStatus.IsPlaying && parseProps.TTSStatus.Supported, "text-white/40 hover:text-white/70 hover:bg-white/10"),
						When(!parseProps.TTSStatus.IsLoading && !parseProps.TTSStatus.IsPlaying && !parseProps.TTSStatus.Supported, "text-[#ffd7a3]/75 hover:text-[#ffe6c6] hover:bg-[#f59e0b]/10"),
					)),
					OnClick(func() {
						if !parseProps.TTSStatus.Supported {
							if parseProps.OnSpeechUpgrade != nil {
								parseProps.OnSpeechUpgrade()
							}
							return
						}
						if parseProps.OnTTSToggle != nil {
							parseProps.OnTTSToggle()
						}
					}),
					Text(func() string {
						switch {
						case parseProps.TTSStatus.IsLoading:
							return parseProps.Intl.T(chatI18nNamespace, "assistant.loading")
						case parseProps.TTSStatus.IsPlaying:
							return parseProps.Intl.T(chatI18nNamespace, "assistant.pause")
						default:
							return parseProps.Intl.T(chatI18nNamespace, "assistant.play")
						}
					}()),
				),
				If(!parseProps.TTSStatus.Supported,
					Span(Class("px-2 py-1 text-xs text-white/25 select-none"), Text(parseProps.Intl.T(chatI18nNamespace, "assistant.speechUnavailable"))),
				),
				If(parseProps.TTSStatus.CanStop,
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						OnClick(parseProps.OnTTSStop),
						Text(parseProps.Intl.T(chatI18nNamespace, "assistant.stop")),
					),
				),
				Button(
					Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
					OnClick(func() { parseCopyToClipboard(parseProps.Message.Content) }),
					Text(parseProps.Intl.T(chatI18nNamespace, "message.copy")),
				),
				Button(
					Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
					Data(dataIdx, parseIndexText),
					OnClick(parseProps.OnFork),
					Text(parseProps.Intl.T(chatI18nNamespace, "message.fork")),
				),
			),
		),
		If(strings.TrimSpace(parseProps.TTSStatus.Error) != "",
			Div(Class("px-2 text-xs text-[#ffb0b0]/80"), Text(parseProps.TTSStatus.Error)),
		),
	)
}
