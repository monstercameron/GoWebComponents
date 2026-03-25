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

func assistantMessageMetaRow(props assistantMessageMetaProps) ui.Node {
	indexText := strconv.Itoa(props.Index)
	messageCost, hasExactCost := props.ThreadCostSummary.AssistantMessageCosts[props.Index]
	hasPerformanceStats := props.Message.Tokens > 0

	return Div(Class("mt-1 flex flex-col gap-1"),
		Div(Class("flex items-center gap-2"),
			If(hasExactCost || hasPerformanceStats,
				Div(Class("flex flex-wrap items-center gap-1.5 px-2 py-1 text-xs text-white/45 select-none"),
					If(hasExactCost,
						Span(Text(strings.TrimSpace(modelLabelForID(messageCost.ModelID, props.ModelOptions)+" "+formatCostUSD(messageCost.Cost)))),
					),
					If(hasExactCost && hasPerformanceStats,
						Span(Class("text-white/15"), Text(".")),
					),
					If(hasPerformanceStats,
						Span(Textf("TTFT %.2fs", props.Message.TTFT)),
					),
					If(hasPerformanceStats,
						Span(Class("text-white/15"), Text(".")),
					),
					If(hasPerformanceStats,
						Span(Textf("%.1f t/s", props.Message.TKPS)),
					),
					If(hasPerformanceStats,
						Span(Class("text-white/15"), Text(".")),
					),
					If(hasPerformanceStats,
						Span(Textf("%d tok", props.Message.Tokens)),
					),
				),
			),
			If(len(props.CanvasArtifacts) > 0,
				Div(Class("flex flex-wrap items-center gap-1"),
					Map(props.CanvasArtifacts, func(artifact canvasArtifact) ui.Node {
						return Button(
							Class("flex items-center gap-1 rounded-lg border border-[#19c37d]/22 px-2 py-1 text-xs text-[#b8ffe0] hover:bg-[#19c37d]/12 transition-colors"),
							Data(dataCanvasID, artifact.ID),
							OnClick(props.OnOpenCanvas),
							Text("Preview: "+artifact.Label),
						)
					}),
				),
			),
			Div(Class("ml-auto flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"),
				Button(
					Class(ClassNames(
						"flex items-center gap-1 px-2 py-1 text-xs rounded-lg transition-colors",
						When(props.TTSStatus.IsLoading, "bg-white/10 text-white/50"),
						When(props.TTSStatus.IsPlaying, "bg-[#19c37d]/18 text-[#b8ffe0] hover:bg-[#19c37d]/24"),
						When(!props.TTSStatus.IsLoading && !props.TTSStatus.IsPlaying && props.TTSStatus.Supported, "text-white/40 hover:text-white/70 hover:bg-white/10"),
						When(!props.TTSStatus.IsLoading && !props.TTSStatus.IsPlaying && !props.TTSStatus.Supported, "text-[#ffd7a3]/75 hover:text-[#ffe6c6] hover:bg-[#f59e0b]/10"),
					)),
					OnClick(func() {
						if !props.TTSStatus.Supported {
							if props.OnSpeechUpgrade != nil {
								props.OnSpeechUpgrade()
							}
							return
						}
						if props.OnTTSToggle != nil {
							props.OnTTSToggle()
						}
					}),
					Text(func() string {
						switch {
						case props.TTSStatus.IsLoading:
							return props.Intl.T(chatI18nNamespace, "assistant.loading")
						case props.TTSStatus.IsPlaying:
							return props.Intl.T(chatI18nNamespace, "assistant.pause")
						default:
							return props.Intl.T(chatI18nNamespace, "assistant.play")
						}
					}()),
				),
				If(!props.TTSStatus.Supported,
					Span(Class("px-2 py-1 text-xs text-white/25 select-none"), Text(props.Intl.T(chatI18nNamespace, "assistant.speechUnavailable"))),
				),
				If(props.TTSStatus.CanStop,
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						OnClick(props.OnTTSStop),
						Text(props.Intl.T(chatI18nNamespace, "assistant.stop")),
					),
				),
				Button(
					Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
					OnClick(func() { copyToClipboard(props.Message.Content) }),
					Text(props.Intl.T(chatI18nNamespace, "message.copy")),
				),
				Button(
					Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
					Data(dataIdx, indexText),
					OnClick(props.OnFork),
					Text(props.Intl.T(chatI18nNamespace, "message.fork")),
				),
			),
		),
		If(strings.TrimSpace(props.TTSStatus.Error) != "",
			Div(Class("px-2 text-xs text-[#ffb0b0]/80"), Text(props.TTSStatus.Error)),
		),
	)
}
