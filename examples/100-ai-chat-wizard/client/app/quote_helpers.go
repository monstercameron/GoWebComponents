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

func parseReadQuoteSelection(parseFallbackX, parseFallbackY float64) (quoteSelectionAnchor, bool) {
	parseGlobal := js.Global()
	if parseGlobal.IsUndefined() || parseGlobal.IsNull() {
		return quoteSelectionAnchor{}, false
	}
	parseSelection := parseGlobal.Call("getSelection")
	if parseSelection.IsUndefined() || parseSelection.IsNull() {
		return quoteSelectionAnchor{}, false
	}
	if parseSelection.Get("rangeCount").Int() == 0 || parseSelection.Get("isCollapsed").Bool() {
		return quoteSelectionAnchor{}, false
	}
	parseText := parseNormalizeQuoteSelectionText(parseSelection.Call("toString").ParseString())
	if parseText == "" {
		return quoteSelectionAnchor{}, false
	}
	parseX := parseFallbackX
	parseY := parseFallbackY
	parseRng := parseSelection.Call("getRangeAt", 0)
	if !parseRng.IsUndefined() && !parseRng.IsNull() {
		parseRect := parseRng.Call("getBoundingClientRect")
		parseWidth := parseRect.Get("width").Float()
		parseHeight := parseRect.Get("height").Float()
		parseLeft := parseRect.Get("left").Float()
		parseTop := parseRect.Get("top").Float()
		if parseWidth > 0 {
			parseX = parseLeft + (parseWidth / 2)
		}
		if parseHeight > 0 {
			parseY = parseTop - 10
		}
	}
	return quoteSelectionAnchor{
		Text: parseText,
		X:    parseClampQuotePromptX(parseGlobal, parseX),
		Y:    parseClampQuotePromptY(parseY),
	}, true
}

func clearQuoteSelection() {
	parseGlobal := js.Global()
	if parseGlobal.IsUndefined() || parseGlobal.IsNull() {
		return
	}
	parseSelection := parseGlobal.Call("getSelection")
	if parseSelection.IsUndefined() || parseSelection.IsNull() {
		return
	}
	parseSelection.Call("removeAllRanges")
}

func parseSelectionTargetMatches(parseTarget js.Value, parseSelector string) bool {
	if parseSelector == "" || parseTarget.IsUndefined() || parseTarget.IsNull() {
		return false
	}
	parseClosest := parseTarget.Get("closest")
	if parseClosest.IsUndefined() || parseClosest.IsNull() || parseClosest.Type() != js.TypeFunction {
		return false
	}
	parseMatch := parseTarget.Call("closest", parseSelector)
	return !parseMatch.IsUndefined() && !parseMatch.IsNull()
}

func parseNormalizeQuoteSelectionText(parseText string) string {
	parseText = strings.ReplaceAll(parseText, "\r\n", "\n")
	parseLines := strings.Split(parseText, "\n")
	parseNormalized := make([]string, 0, len(parseLines))
	isParseLastBlank := false
	for _, parseLine := range parseLines {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			if isParseLastBlank {
				continue
			}
			parseNormalized = append(parseNormalized, "")
			isParseLastBlank = true
			continue
		}
		parseNormalized = append(parseNormalized, strings.TrimRight(parseLine, " \t"))
		isParseLastBlank = false
	}
	return strings.TrimSpace(strings.Join(parseNormalized, "\n"))
}

func formatQuotedInput(parseExisting, parseSelected string) string {
	parseSelected = parseNormalizeQuoteSelectionText(parseSelected)
	if parseSelected == "" {
		return parseExisting
	}
	parseLines := strings.Split(parseSelected, "\n")
	parseQuotedLines := make([]string, 0, len(parseLines))
	for _, parseLine := range parseLines {
		if strings.TrimSpace(parseLine) == "" {
			parseQuotedLines = append(parseQuotedLines, ">")
			continue
		}
		parseQuotedLines = append(parseQuotedLines, "> "+parseLine)
	}
	parseQuotedText := strings.Join(parseQuotedLines, "\n")
	parseExisting = strings.TrimRight(parseExisting, " \n\t")
	if parseExisting == "" {
		return parseQuotedText + "\n\n"
	}
	return fmt.Sprintf("%s\n\n%s\n\n", parseExisting, parseQuotedText)
}

func parseClampQuotePromptX(parseGlobal js.Value, parseX float64) float64 {
	parseWidth := parseGlobal.Get("innerWidth").Float()
	if parseWidth <= 0 {
		parseWidth = 1280
	}
	if parseX < 96 {
		return 96
	}
	parseMaxX := parseWidth - 96
	if parseX > parseMaxX {
		return parseMaxX
	}
	return parseX
}

func parseClampQuotePromptY(parseY float64) float64 {
	if parseY < 72 {
		return 72
	}
	return parseY
}
