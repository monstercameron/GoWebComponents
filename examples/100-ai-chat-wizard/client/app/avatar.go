//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// parseAssistantAvatar renders the circular assistant badge used beside assistant bubbles.
func parseAssistantAvatar() ui.Node {
	return Div(
		Class("mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-[#00d9ff]/45 bg-gradient-to-br from-[#00d9ff]/24 to-[#00d9ff]/8 text-[11px] font-semibold leading-none tracking-[0.06em] text-[#d8f8ff]"),
		Text(assistantBadgeText),
	)
}
