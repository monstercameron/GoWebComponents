//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

const messageBubbleBodyClass = "bg-[#1a1a1a] border border-white/2 text-[1.3125rem] leading-relaxed text-white/90 px-5 py-3"
const assistantPlainTextMessageBubbleClass = messageBubbleBodyClass + " whitespace-pre-wrap max-w-full rounded-3xl rounded-bl-md"
const assistantRichTextMessageBubbleClass = messageBubbleBodyClass + " rounded-3xl rounded-bl-md"
const userPlainTextMessageBubbleClass = messageBubbleBodyClass + " whitespace-pre-wrap max-w-full rounded-3xl rounded-br-md"

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
	parseCanvasArtifacts := canvasArtifactsFromMarkdown(parseIdx, parseM.Content)
	parseTtsKey := parseIdxText + ":" + parseResolvedModelID + ":" + parseM.Content
	parseTtsStatus := parseProps.TTSAudio.ParseStatus(parseTtsKey, parseResolvedModelID)
	parseThoughtBubble := func(parseThoughtText string, isStreaming bool) ui.Node {
		parseSections := parseThoughtSections(parseIdx, parseThoughtText)
		if len(parseSections) == 0 {
			return nil
		}
		parseBubbleClass := "max-w-full min-w-0 rounded-[1.6rem] border border-[#9af7d0]/12 bg-[linear-gradient(180deg,rgba(20,38,33,0.82),rgba(13,24,22,0.72))] px-4 py-3 text-[1.125rem] leading-relaxed text-[#d8fff1]/44 italic shadow-[0_12px_40px_rgba(0,0,0,0.22)] backdrop-blur-sm"
		return Div(
			Class(parseBubbleClass),
			Div(Class("mb-1 text-[0.68rem] uppercase tracking-[0.28em] text-[#9af7d0]/55 not-italic"), Text(parseProps.Intl.T(chatI18nNamespace, "message.thinking"))),
			Div(Class("flex flex-col gap-2"),
				Map(parseSections, func(parseSection thoughtSection) ui.Node {
					parseExpanded := parseProps.ExpandedThoughtSections[parseSection.Key]
					isParseShowBody := parseExpanded && parseSection.Body != ""
					return Div(
						Class("thought-section-card rounded-2xl border border-white/6 bg-black/10 overflow-hidden"),
						Button(
							Class(ClassNames(
								"thought-section-heading thought-section-heading-enter w-full flex items-center gap-3 px-3 py-2 text-left transition-colors not-italic",
								When(parseExpanded, "bg-white/8 text-[#e7fff5]"),
								When(!parseExpanded, "text-[#d8fff1]/78 hover:bg-white/6"),
								When(isStreaming, "thought-section-heading-streaming"),
							)),
							Data(dataThoughtSection, parseSection.Key),
							OnClick(parseProps.ToggleThoughtSection),
							Span(Class("text-[0.72rem] font-mono text-[#9af7d0]/72"), Text(func() string {
								if parseExpanded {
									return "[-]"
								}
								return "[+]"
							}())),
							Span(Class("thought-section-title flex-1 min-w-0 font-semibold"), Text(parseThoughtHeadingLabel(parseProps.Intl, parseSection.Heading))),
							If(isStreaming,
								Span(Class("thought-section-live text-[0.68rem] uppercase tracking-[0.2em] text-[#9af7d0]/55"), Text(parseProps.Intl.T(chatI18nNamespace, "message.live"))),
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
						Class("w-full bg-[#3a3a3a] text-white text-[1.3125rem] rounded-2xl px-4 py-3 resize-none focus:outline-none border border-white/20 leading-relaxed"),
						Rows(4),
						Value(parseProps.EditValue),
						OnInput(parseProps.HandleEditChange),
						OnKeyDown(parseProps.HandleEditKey),
					),
					Div(Class("flex justify-end gap-2"),
						Button(
							Class("px-3 py-1.5 text-xs rounded-lg bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
							OnClick(parseProps.CancelEdit),
							Text(parseProps.Intl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							Class("px-3 py-1.5 text-xs rounded-lg bg-white text-black hover:bg-white/90 transition-colors"),
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
				Class("h-8 w-8 rounded-full bg-gradient-to-br from-[#6366f1] to-[#4f46e5] flex items-center justify-center shrink-0 text-xs font-medium select-none"),
				Text(parseProps.UserInitials),
			),
		)
	}

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
