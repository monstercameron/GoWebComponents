//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// parseAssistantAvatar renders the circular assistant badge used beside assistant bubbles.
func parseAssistantAvatar() ui.Node {
	return Div(
		ClassStr("mt-0.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-[#8e7bff]/22 bg-gradient-to-br from-[#8e7bff]/12 to-[#8e7bff]/5 text-[12px] font-semibold leading-none tracking-[0.06em] text-[#c9c2f0]"),
		Text(assistantBadgeText),
	)
}
