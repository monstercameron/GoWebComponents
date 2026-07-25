//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func parseConversationDeleteIcon() ui.Node {
	return Div(
		ClassStr("relative h-3.5 w-3.5 shrink-0"),
		Span(ClassStr("absolute left-[2px] top-1/2 h-px w-[10px] -translate-y-1/2 rotate-45 rounded-full bg-current")),
		Span(ClassStr("absolute left-[2px] top-1/2 h-px w-[10px] -translate-y-1/2 -rotate-45 rounded-full bg-current")),
	)
}
