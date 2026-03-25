//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func canvasPreviewPane(preview canvasArtifact) ui.Node {
	intl := i18n.UseI18n()
	return Div(
		ID(idCanvasPreviewPane),
		Class("flex flex-col shrink-0 border-t border-white/8 bg-[#171717]/72 backdrop-blur-md xl:w-[min(42vw,34rem)] xl:border-l xl:border-t-0 min-h-[22rem] xl:min-h-0"),
		Div(Class("flex items-center justify-between gap-3 px-4 py-3 border-b border-white/8"),
			Div(Class("flex flex-col gap-1"),
				Span(Class("text-[0.68rem] uppercase tracking-[0.22em] text-[#9af7d0]/58"), Text(intl.T(chatI18nNamespace, "canvas.badge"))),
				Span(Class("text-sm font-semibold text-white/90"), Text(intl.T(chatI18nNamespace, "canvas.title"))),
			),
			Span(Class("text-[0.75rem] text-white/40"), Text(intl.T(chatI18nNamespace, "canvas.caption"))),
		),
		Div(Class("flex-1 min-h-0 p-3"),
			Tag("iframe",
				ID(idCanvasPreviewFrame),
				Class("h-full min-h-[18rem] w-full rounded-[1.35rem] border border-black/8 bg-white shadow-[0_24px_70px_rgba(0,0,0,0.24)]"),
				FromProps(Props{Raw: map[string]interface{}{
					"sandbox": "allow-scripts",
					"srcdoc":  preview.Document,
					"title":   intl.T(chatI18nNamespace, "canvas.title"),
				}}),
			),
		),
	)
}
