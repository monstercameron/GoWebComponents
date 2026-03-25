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
}

func messageBubble(props messageBubbleProps) ui.Node {
	m := props.Message
	idx := props.Index
	idxText := strconv.Itoa(idx)
	resolvedModelID := strings.TrimSpace(m.ModelID)
	if resolvedModelID == "" {
		resolvedModelID = props.DefaultModelID
	}
	canvasArtifacts := canvasArtifactsFromMarkdown(idx, m.Content)
	ttsKey := idxText + ":" + resolvedModelID + ":" + m.Content
	ttsStatus := props.TTSAudio.Status(ttsKey, resolvedModelID)
	thoughtBubble := func(thoughtText string, streaming bool) ui.Node {
		sections := parseThoughtSections(idx, thoughtText)
		if len(sections) == 0 {
			return nil
		}
		bubbleClass := "max-w-full min-w-0 rounded-[1.6rem] border border-[#9af7d0]/12 bg-[linear-gradient(180deg,rgba(20,38,33,0.82),rgba(13,24,22,0.72))] px-4 py-3 text-[1.125rem] leading-relaxed text-[#d8fff1]/44 italic shadow-[0_12px_40px_rgba(0,0,0,0.22)] backdrop-blur-sm"
		return Div(
			Class(bubbleClass),
			Div(Class("mb-1 text-[0.68rem] uppercase tracking-[0.28em] text-[#9af7d0]/55 not-italic"), Text(props.Intl.T(chatI18nNamespace, "message.thinking"))),
			Div(Class("flex flex-col gap-2"),
				Map(sections, func(section thoughtSection) ui.Node {
					expanded := props.ExpandedThoughtSections[section.Key]
					showBody := expanded && section.Body != ""
					return Div(
						Class("rounded-2xl border border-white/6 bg-black/10 overflow-hidden"),
						Button(
							Class(ClassNames(
								"w-full flex items-center gap-3 px-3 py-2 text-left transition-colors not-italic",
								When(expanded, "bg-white/8 text-[#e7fff5]"),
								When(!expanded, "text-[#d8fff1]/78 hover:bg-white/6"),
							)),
							Data(dataThoughtSection, section.Key),
							OnClick(props.ToggleThoughtSection),
							Span(Class("text-[0.72rem] font-mono text-[#9af7d0]/72"), Text(func() string {
								if expanded {
									return "[-]"
								}
								return "[+]"
							}())),
							Span(Class("flex-1 min-w-0 font-semibold"), Text(thoughtHeadingLabel(props.Intl, section.Heading))),
							If(streaming,
								Span(Class("text-[0.68rem] uppercase tracking-[0.2em] text-[#9af7d0]/55"), Text(props.Intl.T(chatI18nNamespace, "message.live"))),
							),
						),
						If(showBody,
							Div(Class("border-t border-white/6 px-3 py-3 whitespace-pre-wrap [text-shadow:0_0_18px_rgba(154,247,208,0.12)]"), Text(section.Body)),
						),
					)
				}),
			),
		)
	}

	if m.Role == roleSwitch {
		label := m.Content
		if option, ok := modelOptionByID(m.Content, props.ModelOptions); ok {
			label = option.Label
		}
		return Div(
			Class("flex items-center gap-3 py-1"),
			Div(Class("flex-1 h-px bg-white/10")),
			Span(Class("text-xs text-white/25 shrink-0 select-none"), Text(props.Intl.T(chatI18nNamespace, "message.switchedTo", i18n.Arguments{"model": label}))),
			Div(Class("flex-1 h-px bg-white/10")),
		)
	}

	if m.Role == roleUser {
		if props.IsEditing {
			return Div(
				Class("flex justify-end"),
				Div(Class("w-full max-w-full flex flex-col gap-2"),
					Tag("textarea",
						Class("w-full bg-[#3a3a3a] text-white text-[1.3125rem] rounded-2xl px-4 py-3 resize-none focus:outline-none border border-white/20 leading-relaxed"),
						Rows(4),
						Value(props.EditValue),
						OnInput(props.HandleEditChange),
						OnKeyDown(props.HandleEditKey),
					),
					Div(Class("flex justify-end gap-2"),
						Button(
							Class("px-3 py-1.5 text-xs rounded-lg bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
							OnClick(props.CancelEdit),
							Text(props.Intl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							Class("px-3 py-1.5 text-xs rounded-lg bg-white text-black hover:bg-white/90 transition-colors"),
							OnClick(props.SubmitEdit),
							Text(props.Intl.T(chatI18nNamespace, "message.saveResend")),
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
					Text(m.Content),
				),
				Div(Class("flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"),
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						OnClick(func() { copyToClipboard(m.Content) }),
						Text("\u29c9 "+props.Intl.T(chatI18nNamespace, "message.copy")),
					),
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						Data(dataIdx, idxText),
						OnClick(props.StartEdit),
						Text("\u270e "+props.Intl.T(chatI18nNamespace, "message.edit")),
					),
					Button(
						Class("flex items-center gap-1 px-2 py-1 text-xs text-white/40 hover:text-white/70 hover:bg-white/10 rounded-lg transition-colors"),
						Data(dataIdx, idxText),
						OnClick(props.OnFork),
						Text("\u2387 "+props.Intl.T(chatI18nNamespace, "message.fork")),
					),
				),
			),
			Div(
				Class("h-8 w-8 rounded-full bg-gradient-to-br from-[#6366f1] to-[#4f46e5] flex items-center justify-center shrink-0 text-xs font-medium select-none"),
				Text(props.UserInitials),
			),
		)
	}

	hasThought := strings.TrimSpace(m.Thought) != ""
	showThought := hasThought || m.ThoughtPending
	hasContent := strings.TrimSpace(m.Content) != ""

	if m.Pending && m.Content == "" && !hasThought {
		return Div(
			ID(idStreamingBubble),
			Class("flex items-start gap-3"),
			assistantAvatar(),
			Div(Class("thinking-dots"),
				Span(),
				Span(),
				Span(),
			),
		)
	}

	if m.Pending || props.ActiveStream {
		return Div(
			ID(idStreamingBubble),
			Class("flex items-start gap-3"),
			assistantAvatar(),
			Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
				If(showThought,
					thoughtBubble(m.Thought, m.ThoughtPending),
				),
				If(hasContent,
					Div(
						Class(assistantPlainTextMessageBubbleClass+" cursor"),
						Text(m.Content),
					),
				),
			),
		)
	}

	if props.UseMarkdownFallback {
		rendered := renderMarkdownSync(m.Content)
		return Div(
			Class("group flex items-start gap-3 msg-bubble"),
			assistantAvatar(),
			Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
				If(showThought,
					thoughtBubble(m.Thought, false),
				),
				Div(Class(assistantRichTextMessageBubbleClass),
					Tag("div", FromProps(Props{
						Class: "prose text-[1.3125rem] text-white/90 min-w-0",
						Raw:   map[string]interface{}{innerHTMLProp: rendered},
					})),
				),
				assistantMessageMetaRow(assistantMessageMetaProps{
					Intl:              props.Intl,
					Message:           m,
					Index:             idx,
					CanvasArtifacts:   canvasArtifacts,
					ModelOptions:      props.ModelOptions,
					ThreadCostSummary: props.ThreadCostSummary,
					OnFork:            props.OnFork,
					OnOpenCanvas:      props.OnOpenCanvas,
					TTSStatus:         ttsStatus,
					OnTTSToggle:       func() { props.TTSAudio.Toggle(ttsKey, m.Content, resolvedModelID) },
					OnTTSStop:         func() { props.TTSAudio.Stop(ttsKey) },
				}),
			),
		)
	}

	rendered, ok := cachedRenderedMarkdown(m.Content)
	if !ok {
		return Div(
			Class("flex items-start gap-3"),
			assistantAvatar(),
			Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
				If(showThought,
					thoughtBubble(m.Thought, false),
				),
				Div(
					Class(assistantPlainTextMessageBubbleClass),
					Text(m.Content),
				),
			),
		)
	}

	return Div(
		Class("group flex items-start gap-3 msg-bubble"),
		assistantAvatar(),
		Div(Class("flex flex-col gap-2 max-w-full min-w-0"),
			If(showThought,
				thoughtBubble(m.Thought, false),
			),
			Div(Class(assistantRichTextMessageBubbleClass),
				Tag("div", FromProps(Props{
					Class: "prose text-[1.3125rem] text-white/90 min-w-0",
					Raw:   map[string]interface{}{innerHTMLProp: rendered},
				})),
			),
			assistantMessageMetaRow(assistantMessageMetaProps{
				Intl:              props.Intl,
				Message:           m,
				Index:             idx,
				CanvasArtifacts:   canvasArtifacts,
				ModelOptions:      props.ModelOptions,
				ThreadCostSummary: props.ThreadCostSummary,
				OnFork:            props.OnFork,
				OnOpenCanvas:      props.OnOpenCanvas,
				TTSStatus:         ttsStatus,
				OnTTSToggle:       func() { props.TTSAudio.Toggle(ttsKey, m.Content, resolvedModelID) },
				OnTTSStop:         func() { props.TTSAudio.Stop(ttsKey) },
			}),
		),
	)
}
