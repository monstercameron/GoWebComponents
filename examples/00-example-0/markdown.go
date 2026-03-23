//go:build js && wasm
// +build js,wasm

package main

import (
	gwchtml "github.com/monstercameron/GoWebComponents/html"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

func markdownRenderOptions(sourcePath string) gwchtml.MarkdownRenderOptions {
	return gwchtml.MarkdownRenderOptions{
		SourcePath:     sourcePath,
		CodeBlockLabel: labelExampleMarkdown,
		LinkTarget:     "_blank",
		LinkRel:        "noreferrer",
		ResolveHref: func(sourcePath, destination string) string {
			resolved := gwchtml.ResolveMarkdownHref(sourcePath, destination)
			return markdownDocumentHref(resolved)
		},
		Classes: gwchtml.MarkdownClasses{
			Heading1:           "text-white font-semibold tracking-tight text-3xl",
			Heading2:           "text-white font-semibold tracking-tight mt-8 text-2xl",
			Heading3:           "text-white font-semibold tracking-tight mt-6 text-xl",
			Heading4:           "text-white font-semibold tracking-tight mt-5 text-lg",
			Heading5:           "text-white font-semibold tracking-tight mt-4 text-base",
			Heading6:           "text-white font-semibold tracking-tight mt-4 text-sm uppercase tracking-[0.18em] text-slate-300",
			Paragraph:          "text-sm leading-7 text-slate-300",
			Blockquote:         "border-l-2 border-cyan-300/35 bg-cyan-400/5 pl-4 text-slate-200",
			List:               "space-y-2 pl-5 text-sm leading-7 text-slate-300",
			ListItem:           "text-sm leading-7 text-slate-300",
			CodeBlockContainer: "rounded-[20px] border border-white/10 bg-[#06101d] p-4",
			CodeBlockLabel:     "text-xs uppercase tracking-[0.18em] text-slate-500",
			CodeBlockPre:       "mt-3 overflow-x-auto text-sm leading-6 text-cyan-100",
			InlineCode:         "rounded bg-white/10 px-1.5 py-0.5 text-cyan-100",
			Strong:             "font-semibold text-white",
			Emphasis:           "italic text-slate-200",
			Link:               "text-cyan-200 underline decoration-cyan-300/40 underline-offset-4",
			HorizontalRule:     "border-white/10",
		},
	}
}

func markdownDocumentHref(sourcePath string) (href string) {
	if sourcePath == "" {
		return ""
	}
	defer func() {
		if recover() != nil {
			href = sourcePath
		}
	}()
	return docsSourceURL(sourcePath)
}

func renderMarkdownState(panelProps contentPanelProps) ui.Node {
	switch {
	case panelProps.MarkdownLoading && !panelProps.MarkdownReady:
		return Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-slate-300"), Text(messageDocLoading))
	case panelProps.MarkdownError != "" && !panelProps.MarkdownReady:
		return Div(Class("rounded-[20px] border border-rose-400/20 bg-rose-400/10 p-4"),
			Div(Class("text-sm font-medium text-rose-100"), Text("Document request failed")),
			P(Class("mt-2 text-sm leading-7 text-rose-50/90"), Text(panelProps.MarkdownError)),
			Button(Type("button"), OnClick(panelProps.OnRetryMarkdown), Class("mt-4 cursor-pointer rounded-2xl border border-rose-300/30 bg-rose-400/15 px-4 py-2 text-sm font-medium text-rose-100 transition hover:bg-rose-400/20"), Text(buttonRetryDocument)),
		)
	case panelProps.Item.Content.SourcePath == "":
		return Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-slate-300"), Text(messageDocUnavailable))
	case panelProps.MarkdownReady:
		markdownNodes := gwchtml.RenderMarkdown(panelProps.MarkdownBody, markdownRenderOptions(panelProps.Item.Content.SourcePath))
		if len(markdownNodes) == 0 {
			return Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-slate-300"), Text(messageDocEmpty))
		}
		return Div(Class("space-y-4"),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelRenderedMarkdown)),
			Div(Class("space-y-4"), markdownNodes),
		)
	default:
		return Div(Class("rounded-[20px] border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-slate-300"), Text(messageDocLoading))
	}
}
