//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parseEmptyState() ui.Node {
	parseIntl := i18n.UseI18n()
	return Div(
		ID(idEmptyState),
		Class("thread-screen flex flex-col items-center gap-4 text-white/40 select-none"),
		Img(
			Src(brandChatIconURL),
			Attr("alt", appBrandName),
			Class("h-16 w-16 rounded-full object-cover"),
		),
		P(Class("text-xl font-semibold text-white/60 text-center"), Text(parseIntl.T(chatI18nNamespace, "empty.heading"))),
		P(Class("text-lg text-center max-w-xl"), Text(parseIntl.T(chatI18nNamespace, "empty.body"))),
	)
}
