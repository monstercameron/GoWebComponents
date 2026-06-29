//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// msg-bubble-assistant / msg-bubble-user are stable behavioral markers for
// tests and devtools; keep them when restyling the presentation classes.
//
// Assistant replies render open-canvas — no card, no border — so the answer
// reads like a document; only the user's turns are pills.
const assistantMessageBodyClass = "msg-bubble-assistant max-w-full text-[1.0625rem] leading-[1.75] text-[#e8e7f2]"
const assistantPlainTextMessageBubbleClass = assistantMessageBodyClass + " whitespace-pre-wrap"
const assistantRichTextMessageBubbleClass = assistantMessageBodyClass
const userPlainTextMessageBubbleClass = "msg-bubble-user whitespace-pre-wrap max-w-[85%] rounded-[1.3rem] rounded-br-[0.4rem] border border-[#8e7bff]/25 bg-gradient-to-br from-[#272252] to-[#1b1836] px-4 py-2.5 text-[1rem] leading-[1.65] text-[#f1eeff] shadow-[0_6px_20px_rgba(0,0,0,0.3),inset_0_1px_0_rgba(255,255,255,0.05)]"

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
		parseBubbleClass := "max-w-full min-w-0 rounded-[1.6rem] border border-[#8e7bff]/12 bg-[#14151d]/85 px-4 py-3 text-[1rem] leading-[1.65] text-[#c9c8e0]/60 italic shadow-[0_4px_16px_rgba(0,0,0,0.25)] backdrop-blur-md"
		return Div(
			ClassStr(parseBubbleClass),
			Div(ClassStr("mb-1 text-[0.68rem] uppercase tracking-[0.28em] text-[#a99bff]/80 not-italic"), Text(parseProps.Intl.T(chatI18nNamespace, "message.thinking"))),
			Div(ClassStr("flex flex-col gap-2"),
				// MapIndexed + StyleVar drive the entry stagger through one
				// --stagger-i custom property per card (animation-delay:
				// calc(var(--stagger-i) * 45ms) in styles.go), so the stagger
				// is unbounded instead of capped by nth-child rules.
				MapIndexed(parseSections, func(parseSectionIdx int, parseSection thoughtSection) ui.Node {
					parseExpanded := parseProps.ExpandedThoughtSections[parseSection.Key]
					isParseShowBody := parseExpanded && parseSection.Body != ""
					return Div(
						ClassStr("thought-section-card rounded-2xl border border-[#8e7bff]/12 bg-black/14 overflow-hidden"),
						StyleVar("--stagger-i", strconv.Itoa(parseSectionIdx)),
						Button(
							ClassStr(ClassNames(
								"thought-section-heading thought-section-heading-enter w-full flex items-center gap-3 px-3 py-2 text-left transition-colors not-italic",
								When(parseExpanded, "bg-[#8e7bff]/10 text-[#edebfb]"),
								When(!parseExpanded, "text-[#d8d6ec]/80 hover:bg-white/6"),
								When(isStreaming, "thought-section-heading-streaming"),
							)),
							Data(dataThoughtSection, parseSection.Key),
							OnClick(parseProps.ToggleThoughtSection),
							Span(ClassStr("text-[0.72rem] font-mono text-[#b9aeff]/74"), Text(func() string {
								if parseExpanded {
									return "[-]"
								}
								return "[+]"
							}())),
							Span(ClassStr("thought-section-title flex-1 min-w-0 font-semibold"), Text(parseThoughtHeadingLabel(parseProps.Intl, parseSection.Heading))),
							If(isStreaming,
								Span(ClassStr("thought-section-live text-[0.68rem] uppercase tracking-[0.2em] text-[#b9aeff]/62"), Text(parseProps.Intl.T(chatI18nNamespace, "message.live"))),
							),
						),
						If(isParseShowBody,
							Div(ClassStr("border-t border-white/6 px-3 py-3 whitespace-pre-wrap [text-shadow:0_0_18px_rgba(154,247,208,0.12)]"), Text(parseSection.Body)),
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
			ClassStr("flex items-center gap-3 py-1"),
			Div(ClassStr("flex-1 h-px bg-white/10")),
			Span(ClassStr("text-xs text-white/25 shrink-0 select-none"), Text(parseProps.Intl.T(chatI18nNamespace, "message.switchedTo", i18n.Arguments{"model": parseLabel}))),
			Div(ClassStr("flex-1 h-px bg-white/10")),
		)
	}

	if parseM.Role == roleUser {
		if parseProps.IsEditing {
			return Div(
				ClassStr("flex justify-end"),
				Div(ClassStr("w-full max-w-full flex flex-col gap-2"),
					Tag("textarea",
						ClassStr("w-full rounded-2xl border border-white/8 bg-[#15161f] px-4 py-3 text-[1.0625rem] leading-[1.7] text-[#ededf4] resize-none focus:outline-none"),
						Rows(4),
						Value(parseProps.EditValue),
						OnInput(parseProps.HandleEditChange),
						OnKeyDown(parseProps.HandleEditKey),
					),
					Div(ClassStr("flex justify-end gap-2"),
						Button(
							ClassStr("rounded-lg border border-white/12 bg-white/6 px-3 py-1.5 text-xs text-white/72 transition-colors hover:bg-white/12"),
							OnClick(parseProps.CancelEdit),
							Text(parseProps.Intl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							ClassStr("rounded-lg border border-[#b9aeff]/55 bg-[#8e7bff] px-3 py-1.5 text-xs font-medium text-[#0a0a14] transition-colors hover:bg-[#a99bff]"),
							OnClick(parseProps.SubmitEdit),
							Text(parseProps.Intl.T(chatI18nNamespace, "message.saveResend")),
						),
					),
				),
			)
		}

		return Div(
			ClassStr("group flex w-full justify-end items-end gap-3 msg-bubble"),
			Div(ClassStr("flex flex-col gap-1 items-end max-w-full min-w-0"),
				Div(
					ClassStr(userPlainTextMessageBubbleClass),
					Text(parseM.Content),
				),
				Div(ClassStr("flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"),
					Button(
						ClassStr("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						OnClick(func() { parseCopyToClipboard(parseM.Content) }),
						Text("\u29c9 "+parseProps.Intl.T(chatI18nNamespace, "message.copy")),
					),
					Button(
						ClassStr("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						Data(dataIdx, parseIdxText),
						OnClick(parseProps.StartEdit),
						Text("\u270e "+parseProps.Intl.T(chatI18nNamespace, "message.edit")),
					),
					Button(
						ClassStr("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						Data(dataIdx, parseIdxText),
						OnClick(parseProps.OnFork),
						Text("\u2387 "+parseProps.Intl.T(chatI18nNamespace, "message.fork")),
					),
				),
			),
			Div(
				ClassStr("h-10 w-10 rounded-full border border-white/12 bg-gradient-to-br from-white/8 to-white/4 flex items-center justify-center shrink-0 text-xs font-semibold select-none"),
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
			ClassStr("flex items-start gap-3"),
			parseAssistantAvatar(),
			Div(ClassStr("thinking-dots"), Repeat(3, Span())),
		)
	}

	if parseM.Pending || parseProps.ActiveStream {
		return Div(
			ID(idStreamingBubble),
			ClassStr("flex items-start gap-3"),
			parseAssistantAvatar(),
			Div(ClassStr("flex flex-col gap-2 max-w-full min-w-0"),
				If(isParseShowThought,
					parseThoughtBubble(parseM.Thought, parseM.ThoughtPending),
				),
				If(hasContent,
					Div(
						ClassStr(assistantPlainTextMessageBubbleClass+" cursor"),
						Text(parseM.Content),
					),
				),
			),
		)
	}

	if parseProps.UseMarkdownFallback {
		parseRendered := renderMarkdownSync(parseM.Content)
		return Div(
			ClassStr("group flex items-start gap-3 msg-bubble"),
			parseAssistantAvatar(),
			Div(ClassStr("flex flex-col gap-2 max-w-full min-w-0"),
				If(isParseShowThought,
					parseThoughtBubble(parseM.Thought, false),
				),
				Div(ClassStr(assistantRichTextMessageBubbleClass),
					Tag("div", FromProps(Props{
						Class: "prose text-[1.0625rem] leading-[1.75] text-[#e8e7f2] min-w-0",
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
			ClassStr("flex items-start gap-3"),
			parseAssistantAvatar(),
			Div(ClassStr("flex flex-col gap-2 max-w-full min-w-0"),
				If(isParseShowThought,
					parseThoughtBubble(parseM.Thought, false),
				),
				Div(
					ClassStr(assistantPlainTextMessageBubbleClass),
					Text(parseM.Content),
				),
			),
		)
	}

	return Div(
		ClassStr("group flex items-start gap-3 msg-bubble"),
		parseAssistantAvatar(),
		Div(ClassStr("flex flex-col gap-2 max-w-full min-w-0"),
			If(isParseShowThought,
				parseThoughtBubble(parseM.Thought, false),
			),
			Div(ClassStr(assistantRichTextMessageBubbleClass),
				Tag("div", FromProps(Props{
					Class: "prose text-[1.0625rem] leading-[1.75] text-[#e8e7f2] min-w-0",
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
