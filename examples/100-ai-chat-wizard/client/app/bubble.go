//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

const messageBubbleBodyClass = "border border-[#8effd8]/16 bg-[linear-gradient(165deg,rgba(13,25,43,0.9),rgba(8,15,28,0.94))] text-[1.3125rem] leading-relaxed text-[#e8f9ff] px-5 py-3 shadow-[0_16px_38px_rgba(3,10,22,0.42)] backdrop-blur-sm"
const assistantPlainTextMessageBubbleClass = messageBubbleBodyClass + " whitespace-pre-wrap max-w-full rounded-[1.35rem] rounded-bl-md"
const assistantRichTextMessageBubbleClass = messageBubbleBodyClass + " rounded-[1.35rem] rounded-bl-md"
const userPlainTextMessageBubbleClass = messageBubbleBodyClass + " whitespace-pre-wrap max-w-full rounded-[1.35rem] rounded-br-md border-[#6dd8ff]/32 bg-[linear-gradient(160deg,rgba(10,39,62,0.92),rgba(8,24,42,0.94))]"

type messageBubbleProps struct {
	Intl                    i18n.Runtime
	Message                 message
	Index                   int
	ActiveStream            bool
	IsEditing               bool
	EditValue               string
	StartEdit               ui.Handler
	CancelEdit              ui.Handler
	HandleEditChange        ui.Handler
	SubmitEdit              ui.Handler
	HandleEditKey           ui.Handler
	OnFork                  ui.Handler
	OnOpenCanvas            ui.Handler
	ToggleThoughtSection    ui.Handler
	ModelOptions            []modelOption
	DefaultModelID          string
	ThreadCostSummary       threadCostSummary
	UserInitials            string
	UseMarkdownFallback     bool
	ExpandedThoughtSections map[string]bool
	ThoughtCacheByMessage   map[int]renderWorkerThoughtCacheEntry
	CanvasCacheByMessage    map[int]renderWorkerCanvasCacheEntry
	TTSAudio                ttsAudioController
	OnSpeechUpgrade         func()
}

func parseMessageBubble(parseProps messageBubbleProps) ui.Node {
	parseM := parseProps.Message
	parseIdx := parseProps.Index
	parseIdxText := strconv.Itoa(parseIdx)
	parseResolvedModelID := strings.TrimSpace(parseM.ModelID)
	if parseResolvedModelID == "" {
		parseResolvedModelID = parseProps.DefaultModelID
	}
	parseCanvasArtifacts := []canvasArtifact(nil)
	parseTtsKey := parseIdxText + ":" + parseResolvedModelID + ":" + parseM.Content
	parseTtsStatus := parseProps.TTSAudio.ParseStatus(parseTtsKey, parseResolvedModelID)
	parseThoughtBubble := func(parseThoughtText string, isStreaming bool) ui.Node {
		parseSections := parseResolveThoughtSectionsForMessage(parseProps.ThoughtCacheByMessage, parseIdx, parseThoughtText)
		if len(parseSections) == 0 {
			return nil
		}
		parseBubbleClass := "max-w-full min-w-0 rounded-[1.6rem] border border-[#00d9ff]/20 bg-[linear-gradient(180deg,rgba(14,32,40,0.86),rgba(10,22,31,0.76))] px-4 py-3 text-[1.125rem] leading-relaxed text-[#d7f6ff]/56 italic shadow-[0_12px_40px_rgba(0,0,0,0.22)] backdrop-blur-sm"
		return Div(
			Class(parseBubbleClass),
			Div(Class("mb-1 text-[0.68rem] uppercase tracking-[0.28em] text-[#7ee9ff]/70 not-italic"), Text(parseProps.Intl.T(chatI18nNamespace, "message.thinking"))),
			Div(Class("flex flex-col gap-2"),
				Map(parseSections, func(parseSection thoughtSection) ui.Node {
					parseExpanded := parseProps.ExpandedThoughtSections[parseSection.Key]
					isParseShowBody := parseExpanded && parseSection.Body != ""
					return Div(
						Class("thought-section-card rounded-2xl border border-[#00d9ff]/12 bg-black/14 overflow-hidden"),
						Button(
							Class(ClassNames(
								"thought-section-heading thought-section-heading-enter w-full flex items-center gap-3 px-3 py-2 text-left transition-colors not-italic",
								When(parseExpanded, "bg-[#00d9ff]/10 text-[#e7fbff]"),
								When(!parseExpanded, "text-[#d7f6ff]/80 hover:bg-white/6"),
								When(isStreaming, "thought-section-heading-streaming"),
							)),
							Data(dataThoughtSection, parseSection.Key),
							OnClick(parseProps.ToggleThoughtSection),
							Span(Class("text-[0.72rem] font-mono text-[#7ee9ff]/74"), Text(func() string {
								if parseExpanded {
									return "[-]"
								}
								return "[+]"
							}())),
							Span(Class("thought-section-title flex-1 min-w-0 font-semibold"), Text(parseThoughtHeadingLabel(parseProps.Intl, parseSection.Heading))),
							If(isStreaming,
								Span(Class("thought-section-live text-[0.68rem] uppercase tracking-[0.2em] text-[#7ee9ff]/62"), Text(parseProps.Intl.T(chatI18nNamespace, "message.live"))),
							),
						),
						If(isParseShowBody,
							Div(Class("border-t border-white/6 px-3 py-3 whitespace-pre-wrap [text-shadow:0_0_18px_rgba(154,247,208,0.12)]"), Text(parseSection.Body)),
						),
					)
				}),
			),
		)
	}

	if parseM.Role == roleSwitch {
		parseLabel := parseM.Content
		if parseOption, parseOk := parseModelOptionByID(parseM.Content, parseProps.ModelOptions); parseOk {
			parseLabel = parseOption.Label
		}
		return Div(
			Class("flex items-center gap-3 py-1"),
			Div(Class("flex-1 h-px bg-white/10")),
			Span(Class("text-xs text-white/25 shrink-0 select-none"), Text(parseProps.Intl.T(chatI18nNamespace, "message.switchedTo", i18n.Arguments{"model": parseLabel}))),
			Div(Class("flex-1 h-px bg-white/10")),
		)
	}

	if parseM.Role == roleUser {
		if parseProps.IsEditing {
			return Div(
				Class("flex justify-end"),
				Div(Class("w-full max-w-full flex flex-col gap-2"),
					Tag("textarea",
						Class("w-full rounded-2xl border border-[#8fffd8]/20 bg-[linear-gradient(160deg,rgba(14,27,45,0.94),rgba(8,15,27,0.96))] px-4 py-3 text-[1.3125rem] leading-relaxed text-[#e8f9ff] resize-none focus:outline-none"),
						Rows(4),
						Value(parseProps.EditValue),
						OnInput(parseProps.HandleEditChange),
						OnKeyDown(parseProps.HandleEditKey),
					),
					Div(Class("flex justify-end gap-2"),
						Button(
							Class("rounded-lg border border-white/12 bg-white/6 px-3 py-1.5 text-xs text-white/72 transition-colors hover:bg-white/12"),
							OnClick(parseProps.CancelEdit),
							Text(parseProps.Intl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							Class("rounded-lg border border-[#7fe9ff]/55 bg-[#00d9ff] px-3 py-1.5 text-xs font-medium text-[#05111d] transition-colors hover:bg-[#33e3ff]"),
							OnClick(parseProps.SubmitEdit),
							Text(parseProps.Intl.T(chatI18nNamespace, "message.saveResend")),
						),
					),
				),
			)
		}

		return Div(
			Class("group flex w-full justify-end items-end gap-3 msg-bubble"),
			Div(Class("flex flex-col gap-1 items-end max-w-full min-w-0"),
				Div(
					Class(userPlainTextMessageBubbleClass),
					Text(parseM.Content),
				),
				Div(Class("flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"),
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						OnClick(func() { parseCopyToClipboard(parseM.Content) }),
						Text("\u29c9 "+parseProps.Intl.T(chatI18nNamespace, "message.copy")),
					),
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						Data(dataIdx, parseIdxText),
						OnClick(parseProps.StartEdit),
						Text("\u270e "+parseProps.Intl.T(chatI18nNamespace, "message.edit")),
					),
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						Data(dataIdx, parseIdxText),
						OnClick(parseProps.OnFork),
						Text("\u2387 "+parseProps.Intl.T(chatI18nNamespace, "message.fork")),
					),
				),
			),
			Div(
				Class("h-8 w-8 rounded-full border border-[#00d9ff]/40 bg-gradient-to-br from-[#00d9ff]/28 to-[#3b82f6]/22 flex items-center justify-center shrink-0 text-xs font-semibold select-none"),
				Text(parseProps.UserInitials),
			),
		)
	}

	parseCanvasArtifacts = parseResolveCanvasArtifactsForMessage(parseProps.CanvasCacheByMessage, parseIdx, parseM.Content)
	hasThought := strings.TrimSpace(parseM.Thought) != ""
	isParseShowThought := hasThought || parseM.ThoughtPending
	hasContent := strings.TrimSpace(parseM.Content) != ""

	if parseM.Pending && parseM.Content == "" && !hasThought {
		return Div(
			ID(idStreamingBubble),
			Class("flex items-start gap-3"),
			parseAssistantAvatar(),
			Div(Class("thinking-dots"),
				Span(),
				Span(),
				Span(),
			),
		)
	}

	if parseM.Pending || parseProps.ActiveStream {
		return Div(
			ID(idStreamingBubble),
			Class("flex items-start gap-3"),
			parseAssistantAvatar(),
			Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
				If(isParseShowThought,
					parseThoughtBubble(parseM.Thought, parseM.ThoughtPending),
				),
				If(hasContent,
					Div(
						Class(assistantPlainTextMessageBubbleClass+" cursor"),
						Text(parseM.Content),
					),
				),
			),
		)
	}

	if parseProps.UseMarkdownFallback {
		parseRendered := renderMarkdownSync(parseM.Content)
		return Div(
			Class("group flex items-start gap-3 msg-bubble"),
			parseAssistantAvatar(),
			Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
				If(isParseShowThought,
					parseThoughtBubble(parseM.Thought, false),
				),
				Div(Class(assistantRichTextMessageBubbleClass),
					Tag("div", FromProps(Props{
						Class: "prose text-[1.3125rem] text-white/90 min-w-0",
						Raw:   map[string]interface{}{innerHTMLProp: parseRendered},
					})),
				),
				parseAssistantMessageMetaRow(assistantMessageMetaProps{
					Intl:              parseProps.Intl,
					Message:           parseM,
					Index:             parseIdx,
					CanvasArtifacts:   parseCanvasArtifacts,
					ModelOptions:      parseProps.ModelOptions,
					ThreadCostSummary: parseProps.ThreadCostSummary,
					OnFork:            parseProps.OnFork,
					OnOpenCanvas:      parseProps.OnOpenCanvas,
					TTSStatus:         parseTtsStatus,
					OnTTSToggle:       func() { parseProps.TTSAudio.ParseToggle(parseTtsKey, parseM.Content, parseResolvedModelID) },
					OnTTSStop:         func() { parseProps.TTSAudio.ParseStop(parseTtsKey) },
					OnSpeechUpgrade:   parseProps.OnSpeechUpgrade,
				}),
			),
		)
	}

	parseRendered2, parseOk2 := cachedRenderedMarkdown(parseM.Content)
	if !parseOk2 {
		return Div(
			Class("flex items-start gap-3"),
			parseAssistantAvatar(),
			Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
				If(isParseShowThought,
					parseThoughtBubble(parseM.Thought, false),
				),
				Div(
					Class(assistantPlainTextMessageBubbleClass),
					Text(parseM.Content),
				),
			),
		)
	}

	return Div(
		Class("group flex items-start gap-3 msg-bubble"),
		parseAssistantAvatar(),
		Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
			If(isParseShowThought,
				parseThoughtBubble(parseM.Thought, false),
			),
			Div(Class(assistantRichTextMessageBubbleClass),
				Tag("div", FromProps(Props{
					Class: "prose text-[1.3125rem] text-white/90 min-w-0",
					Raw:   map[string]interface{}{innerHTMLProp: parseRendered2},
				})),
			),
			parseAssistantMessageMetaRow(assistantMessageMetaProps{
				Intl:              parseProps.Intl,
				Message:           parseM,
				Index:             parseIdx,
				CanvasArtifacts:   parseCanvasArtifacts,
				ModelOptions:      parseProps.ModelOptions,
				ThreadCostSummary: parseProps.ThreadCostSummary,
				OnFork:            parseProps.OnFork,
				OnOpenCanvas:      parseProps.OnOpenCanvas,
				TTSStatus:         parseTtsStatus,
				OnTTSToggle:       func() { parseProps.TTSAudio.ParseToggle(parseTtsKey, parseM.Content, parseResolvedModelID) },
				OnTTSStop:         func() { parseProps.TTSAudio.ParseStop(parseTtsKey) },
				OnSpeechUpgrade:   parseProps.OnSpeechUpgrade,
			}),
		),
	)
}

// parseResolveCanvasArtifactsForMessage resolves one message's canvas artifact list from worker cache or synchronous fallback.
func parseResolveCanvasArtifactsForMessage(parseCanvasCacheByMessage map[int]renderWorkerCanvasCacheEntry, parseMessageIndex int, parseContentText string) []canvasArtifact {
	if parseCacheEntry, hasParseCacheEntry := parseCanvasCacheByMessage[parseMessageIndex]; hasParseCacheEntry && parseCacheEntry.GetContentText == parseContentText {
		return parseCacheEntry.GetArtifact
	}
	return canvasArtifactsFromMarkdown(parseMessageIndex, parseContentText)
}

// parseResolveThoughtSectionsForMessage resolves one message's thought sections from worker cache or synchronous fallback.
func parseResolveThoughtSectionsForMessage(parseThoughtCacheByMessage map[int]renderWorkerThoughtCacheEntry, parseMessageIndex int, parseThoughtText string) []thoughtSection {
	if parseCacheEntry, hasParseCacheEntry := parseThoughtCacheByMessage[parseMessageIndex]; hasParseCacheEntry && parseCacheEntry.GetThoughtText == parseThoughtText {
		return parseCacheEntry.GetSection
	}
	return parseThoughtSections(parseMessageIndex, parseThoughtText)
}
