//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderSupportIDChip renders one compact copyable request-identifier chip suitable for customer support handoff.
// The chip is styled for legibility inside error banners and toasts and copies the raw identifier to the clipboard on click.
func renderSupportIDChip(parseRequestID string) ui.Node {
	parseRequestID = strings.TrimSpace(parseRequestID)
	if parseRequestID == "" {
		return nil
	}
	return Span(
		Class("mt-1.5 inline-flex cursor-pointer select-all items-center gap-1 rounded border border-white/10 bg-white/5 px-1.5 py-0.5 font-mono text-[0.6rem] tracking-wide text-[#b8c2d9] transition hover:border-white/20 hover:bg-white/8"),
		Attr("title", "Click to copy request ID"),
		OnClick(func() { parseCopyToClipboard(parseRequestID) }),
		Text("ID: "+parseRequestID),
	)
}
