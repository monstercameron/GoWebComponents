//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

func assistantAvatar() ui.Node {
	return Div(
		Class("h-8 w-8 rounded-full bg-gradient-to-br from-[#19c37d] to-[#0ea47e] flex items-center justify-center shrink-0 mt-0.5 text-sm"),
		Text(assistantBadgeText),
	)
}
