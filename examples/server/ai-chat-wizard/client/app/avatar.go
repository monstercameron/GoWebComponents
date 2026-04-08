//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// parseAssistantAvatar renders the circular assistant badge used beside assistant bubbles.
func parseAssistantAvatar() ui.Node {
	return Div(
		Class("mt-0.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-[#00d9ff]/22 bg-gradient-to-br from-[#00d9ff]/12 to-[#00d9ff]/5 text-[12px] font-semibold leading-none tracking-[0.06em] text-[#b8e8f8]"),
		Text(assistantBadgeText),
	)
}
