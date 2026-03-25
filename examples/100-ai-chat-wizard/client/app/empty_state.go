//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func emptyState() ui.Node {
	intl := i18n.UseI18n()
	return Div(
		ID(idEmptyState),
		Class("thread-screen flex flex-col items-center gap-4 text-white/40 select-none"),
		Div(Class("h-16 w-16 rounded-full bg-gradient-to-br from-[#19c37d] to-[#0ea47e] flex items-center justify-center text-2xl"),
			Text(assistantBadgeText),
		),
		P(Class("text-xl font-semibold text-white/60 text-center"), Text(intl.T(chatI18nNamespace, "empty.heading"))),
		P(Class("text-lg text-center max-w-xl"), Text(intl.T(chatI18nNamespace, "empty.body"))),
	)
}
