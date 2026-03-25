//go:build js && wasm

package app

import (
	"fmt"
	"strings"
	"syscall/js"
)

type quoteSelectionAnchor struct {
	Text string
	X    float64
	Y    float64
}

func readQuoteSelection(fallbackX, fallbackY float64) (quoteSelectionAnchor, bool) {
	global := js.Global()
	if global.IsUndefined() || global.IsNull() {
		return quoteSelectionAnchor{}, false
	}
	selection := global.Call("getSelection")
	if selection.IsUndefined() || selection.IsNull() {
		return quoteSelectionAnchor{}, false
	}
	if selection.Get("rangeCount").Int() == 0 || selection.Get("isCollapsed").Bool() {
		return quoteSelectionAnchor{}, false
	}
	text := normalizeQuoteSelectionText(selection.Call("toString").String())
	if text == "" {
		return quoteSelectionAnchor{}, false
	}
	x := fallbackX
	y := fallbackY
	rng := selection.Call("getRangeAt", 0)
	if !rng.IsUndefined() && !rng.IsNull() {
		rect := rng.Call("getBoundingClientRect")
		width := rect.Get("width").Float()
		height := rect.Get("height").Float()
		left := rect.Get("left").Float()
		top := rect.Get("top").Float()
		if width > 0 {
			x = left + (width / 2)
		}
		if height > 0 {
			y = top - 10
		}
	}
	return quoteSelectionAnchor{
		Text: text,
		X:    clampQuotePromptX(global, x),
		Y:    clampQuotePromptY(y),
	}, true
}

func clearQuoteSelection() {
	global := js.Global()
	if global.IsUndefined() || global.IsNull() {
		return
	}
	selection := global.Call("getSelection")
	if selection.IsUndefined() || selection.IsNull() {
		return
	}
	selection.Call("removeAllRanges")
}

func selectionTargetMatches(target js.Value, selector string) bool {
	if selector == "" || target.IsUndefined() || target.IsNull() {
		return false
	}
	closest := target.Get("closest")
	if closest.IsUndefined() || closest.IsNull() || closest.Type() != js.TypeFunction {
		return false
	}
	match := target.Call("closest", selector)
	return !match.IsUndefined() && !match.IsNull()
}

func normalizeQuoteSelectionText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	normalized := make([]string, 0, len(lines))
	lastBlank := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if lastBlank {
				continue
			}
			normalized = append(normalized, "")
			lastBlank = true
			continue
		}
		normalized = append(normalized, strings.TrimRight(line, " \t"))
		lastBlank = false
	}
	return strings.TrimSpace(strings.Join(normalized, "\n"))
}

func formatQuotedInput(existing, selected string) string {
	selected = normalizeQuoteSelectionText(selected)
	if selected == "" {
		return existing
	}
	lines := strings.Split(selected, "\n")
	quotedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			quotedLines = append(quotedLines, ">")
			continue
		}
		quotedLines = append(quotedLines, "> "+line)
	}
	quotedText := strings.Join(quotedLines, "\n")
	existing = strings.TrimRight(existing, " \n\t")
	if existing == "" {
		return quotedText + "\n\n"
	}
	return fmt.Sprintf("%s\n\n%s\n\n", existing, quotedText)
}

func clampQuotePromptX(global js.Value, x float64) float64 {
	width := global.Get("innerWidth").Float()
	if width <= 0 {
		width = 1280
	}
	if x < 96 {
		return 96
	}
	maxX := width - 96
	if x > maxX {
		return maxX
	}
	return x
}

func clampQuotePromptY(y float64) float64 {
	if y < 72 {
		return 72
	}
	return y
}
