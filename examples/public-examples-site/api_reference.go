//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const innerHTMLProperty = "__gwc_prop__:innerHTML"
const activeDemoAnchorProperty = "__gwc_active_demo_anchor__"
const demoAnchorScrollRequestProperty = "__gwc_demo_anchor_scroll_request__"
const apiReferenceSearchProperty = "__gwc_api_reference_search__"

func setRequestedDemoAnchor(parseAnchorID string) {
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return
	}
	parseWindow.Set(demoAnchorScrollRequestProperty, parseAnchorID)
}

func clearRequestedDemoAnchor() {
	setRequestedDemoAnchor("")
}

func findScrollableAncestor(parseTarget js.Value) js.Value {
	if parseTarget.IsUndefined() || parseTarget.IsNull() {
		return js.Undefined()
	}
	parseCurrent := parseTarget.Get("parentElement")
	for !parseCurrent.IsUndefined() && !parseCurrent.IsNull() {
		parseStyle := js.Global().Get("window").Call("getComputedStyle", parseCurrent)
		parseOverflowY := parseStyle.Get("overflowY").String()
		isScrollableOverflow := parseOverflowY == "auto" || parseOverflowY == "scroll"
		if isScrollableOverflow && parseCurrent.Get("scrollHeight").Float() > parseCurrent.Get("clientHeight").Float()+1 {
			return parseCurrent
		}
		parseCurrent = parseCurrent.Get("parentElement")
	}
	return js.Undefined()
}

func isGroupedAPIItem(parseItem docsItem) bool {
	return parseItem.Type == kindAPI && parseItem.Content.AnchorID != "" && parseItem.Content.SourcePath != ""
}

func apiReferenceDocumentURL(parseItem docsItem) string {
	if parseItem.Content.SourcePath == "" {
		return ""
	}
	parseUrl := docsSourceURL(parseItem.Content.SourcePath)
	if parseItem.Content.AnchorID != "" {
		parseUrl += "#" + parseItem.Content.AnchorID
	}
	return parseUrl
}

func scrollToDemoAnchor(parseAnchorID string, isReady bool, parseBody string) {
	ui.UseEffect(func() func() {
		if !isReady || parseBody == "" || parseAnchorID == "" {
			return nil
		}
		parseWindow := js.Global().Get("window")
		if parseWindow.IsUndefined() || parseWindow.IsNull() {
			return nil
		}
		parseRequestedAnchor := parseWindow.Get(demoAnchorScrollRequestProperty)
		if parseRequestedAnchor.IsUndefined() || parseRequestedAnchor.IsNull() || parseRequestedAnchor.String() != parseAnchorID {
			return nil
		}
		parseWindow.Set(activeDemoAnchorProperty, parseAnchorID)
		parseAttemptScroll := func() {
			parseCurrentAnchor := parseWindow.Get(activeDemoAnchorProperty)
			if parseCurrentAnchor.IsUndefined() || parseCurrentAnchor.IsNull() || parseCurrentAnchor.String() != parseAnchorID {
				return
			}
			parseDocument := js.Global().Get("document")
			if parseDocument.IsUndefined() || parseDocument.IsNull() {
				return
			}
			parseContainer := parseDocument.Call("querySelector", "#demo")
			parseSelector := fmt.Sprintf("#demo #%s", parseAnchorID)
			parseTarget := parseDocument.Call("querySelector", parseSelector)
			if parseContainer.IsUndefined() || parseContainer.IsNull() || parseTarget.IsUndefined() || parseTarget.IsNull() {
				return
			}
			if parseTarget.Get("scrollIntoView").Type() == js.TypeFunction {
				parseTarget.Call("scrollIntoView", true)
			}
			parseTargetRect := parseTarget.Call("getBoundingClientRect")
			if parseTargetRect.Get("top").Float() >= 0 && parseTargetRect.Get("top").Float() <= 48 {
				return
			}
			parseScrollContainer := findScrollableAncestor(parseTarget)
			if !parseScrollContainer.IsUndefined() && !parseScrollContainer.IsNull() {
				parseContainerRect := parseScrollContainer.Call("getBoundingClientRect")
				parseNextTop := parseScrollContainer.Get("scrollTop").Float() + parseTargetRect.Get("top").Float() - parseContainerRect.Get("top").Float() - 24
				if parseScrollContainer.Get("scrollTo").Type() == js.TypeFunction {
					parseScrollContainer.Call("scrollTo", 0, parseNextTop)
					return
				}
				parseScrollContainer.Set("scrollTop", parseNextTop)
				return
			}
			if parseWindow.IsUndefined() || parseWindow.IsNull() {
				return
			}
			parseNextWindowTop := parseWindow.Get("scrollY").Float() + parseTargetRect.Get("top").Float() - 24
			if parseWindow.Get("scrollTo").Type() == js.TypeFunction {
				parseWindow.Call("scrollTo", 0, parseNextWindowTop)
				return
			}
			parseDocumentElement := parseDocument.Get("documentElement")
			if !parseDocumentElement.IsUndefined() && !parseDocumentElement.IsNull() {
				parseDocumentElement.Set("scrollTop", parseNextWindowTop)
			}
		}

		parseAttemptScroll()

		parseDelays := []time.Duration{16 * time.Millisecond, 80 * time.Millisecond, 180 * time.Millisecond, 320 * time.Millisecond, 520 * time.Millisecond, 900 * time.Millisecond, 1400 * time.Millisecond, 2 * time.Second}
		parseTimers := make([]interop.Timer, 0, len(parseDelays))
		for _, parseDelay := range parseDelays {
			parseTimer, parseErr := interop.ScheduleTimeout(parseDelay, parseAttemptScroll)
			if parseErr != nil {
				continue
			}
			parseTimers = append(parseTimers, parseTimer)
		}
		if len(parseTimers) == 0 {
			return nil
		}
		return func() {
			for _, parseTimer2 := range parseTimers {
				_ = parseTimer2.Cancel()
			}
		}
	}, parseAnchorID, isReady, len(parseBody))
}

func renderInjectedHTMLFragment(parseClassName, parseBody string) ui.Node {
	return Tag("div", FromProps(Props{Class: parseClassName, Raw: map[string]interface{}{innerHTMLProperty: parseBody}}))
}

func filterAPIReferenceSections(parseQuery string, isReady bool, parseBody string) {
	ui.UseEffect(func() func() {
		if !isReady || parseBody == "" {
			return nil
		}
		parseWindow := js.Global().Get("window")
		if parseWindow.IsUndefined() || parseWindow.IsNull() {
			return nil
		}
		parseNormalizedQuery := strings.TrimSpace(strings.ToLower(parseQuery))
		parseWindow.Set(apiReferenceSearchProperty, parseNormalizedQuery)
		applyFilter := func() {
			parseCurrentQuery := parseWindow.Get(apiReferenceSearchProperty)
			if parseCurrentQuery.IsUndefined() || parseCurrentQuery.IsNull() || parseCurrentQuery.String() != parseNormalizedQuery {
				return
			}
			parseDocument := js.Global().Get("document")
			if parseDocument.IsUndefined() || parseDocument.IsNull() {
				return
			}
			parseFragment := parseDocument.Call("querySelector", "#demo .api-reference-fragment")
			if parseFragment.IsUndefined() || parseFragment.IsNull() {
				return
			}
			parseApiItems := parseFragment.Call("querySelectorAll", ".api-item")
			parseItemCount := parseApiItems.Get("length").Int()
			for parseIndex := 0; parseIndex < parseItemCount; parseIndex++ {
				parseItem := parseApiItems.Index(parseIndex)
				isParseMatches := parseNormalizedQuery == "" || strings.Contains(strings.ToLower(parseItem.Get("textContent").String()), parseNormalizedQuery)
				if isParseMatches {
					parseItem.Get("style").Set("display", "")
				} else {
					parseItem.Get("style").Set("display", "none")
				}
			}

			parseArticles := parseFragment.Call("querySelectorAll", "article")
			parseArticleCount := parseArticles.Get("length").Int()
			for parseIndex2 := 0; parseIndex2 < parseArticleCount; parseIndex2++ {
				parseArticle := parseArticles.Index(parseIndex2)
				parseItemNodes := parseArticle.Call("querySelectorAll", ".api-item")
				parseItemNodeCount := parseItemNodes.Get("length").Int()
				if parseItemNodeCount == 0 {
					parseArticle.Get("style").Set("display", "")
					continue
				}
				hasVisibleItem := false
				for parseItemIndex := 0; parseItemIndex < parseItemNodeCount; parseItemIndex++ {
					if parseItemNodes.Index(parseItemIndex).Get("style").Get("display").String() != "none" {
						hasVisibleItem = true
						break
					}
				}
				if hasVisibleItem {
					parseArticle.Get("style").Set("display", "")
				} else {
					parseArticle.Get("style").Set("display", "none")
				}
			}

			parseSections := parseFragment.Call("querySelectorAll", "section")
			parseSectionCount := parseSections.Get("length").Int()
			for parseIndex3 := 0; parseIndex3 < parseSectionCount; parseIndex3++ {
				parseSection := parseSections.Index(parseIndex3)
				if parseNormalizedQuery == "" {
					parseSection.Get("style").Set("display", "")
					continue
				}
				parseMatchesHeading := strings.Contains(strings.ToLower(parseSection.Get("textContent").String()), parseNormalizedQuery)
				parseVisibleItems := parseSection.Call("querySelectorAll", ".api-item")
				parseVisibleCount := parseVisibleItems.Get("length").Int()
				hasVisibleItem := false
				for parseItemIndex2 := 0; parseItemIndex2 < parseVisibleCount; parseItemIndex2++ {
					if parseVisibleItems.Index(parseItemIndex2).Get("style").Get("display").String() != "none" {
						hasVisibleItem = true
						break
					}
				}
				if parseMatchesHeading || hasVisibleItem {
					parseSection.Get("style").Set("display", "")
				} else {
					parseSection.Get("style").Set("display", "none")
				}
			}
		}
		parseDelays := []time.Duration{16 * time.Millisecond, 80 * time.Millisecond, 180 * time.Millisecond}
		parseTimers := make([]interop.Timer, 0, len(parseDelays))
		for _, parseDelay := range parseDelays {
			parseTimer, parseErr := interop.ScheduleTimeout(parseDelay, applyFilter)
			if parseErr != nil {
				continue
			}
			parseTimers = append(parseTimers, parseTimer)
		}
		return func() {
			for _, parseTimer2 := range parseTimers {
				_ = parseTimer2.Cancel()
			}
		}
	}, parseQuery, isReady, len(parseBody))
}

func renderGroupedAPIReference(parsePanelProps contentPanelProps) ui.Node {
	return ui.Component(renderGroupedAPIReferenceComponent, parsePanelProps)
}

func renderGroupedAPIReferenceComponent(parsePanelProps contentPanelProps) ui.Node {
	parseApiSearchQuery := ui.UseState("")
	parseUpdateAPISearchQuery := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseApiSearchQuery.Set(parseEvent.GetValue())
	})
	scrollToDemoAnchor(parsePanelProps.Item.Content.AnchorID, parsePanelProps.MarkdownReady, parsePanelProps.MarkdownBody)
	filterAPIReferenceSections(parseApiSearchQuery.Get(), parsePanelProps.MarkdownReady, parsePanelProps.MarkdownBody)

	return Div(ClassStr("min-w-0 flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-3 shadow-inner shadow-black/20"),
		Div(ClassStr("border-b border-white/10 pb-2"),
			Div(ClassStr("text-sm font-medium text-white"), Text(labelAPIReference)),
			Div(ClassStr("text-xs uppercase tracking-[0.18em] text-slate-500"), Text("Catalog-owned grouped API reference document")),
		),
		Div(ClassStr("mt-3 rounded-[20px] border border-white/10 bg-white/[0.04] p-3"),
			Div(ClassStr("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelReferenceSearch)),
			Input(
				Value(parseApiSearchQuery.Get()),
				OnInput(parseUpdateAPISearchQuery),
				Placeholder(messageReferenceSearch),
				ClassStr("mt-2 w-full rounded-xl border border-white/10 bg-slate-950/40 px-3 py-2 text-sm text-white outline-none placeholder:text-slate-500 transition focus:border-cyan-300/40 focus:bg-slate-950/60"),
			),
		),
		Div(ClassStr("mt-3 rounded-[22px] border border-violet-400/20 bg-violet-400/10 p-3"),
			Div(ClassStr("text-xs uppercase tracking-[0.18em] text-violet-200"), Text("Reference scope")),
			P(ClassStr("mt-2 text-sm leading-7 text-violet-50"), Text("This API group is defined in the catalog and resolves to a shared HTML reference fragment. Selecting a card fetches the fragment into this surface and then scrolls to the matching section anchor.")),
			Div(ClassStr("mt-2 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm leading-6 text-cyan-100 break-all"), Text(apiReferenceDocumentURL(parsePanelProps.Item))),
		),
		Div(ClassStr("mt-3 min-h-[72vh] rounded-[22px] border border-white/10 bg-[#08111f] p-2"),
			renderInjectedHTMLFragment("api-reference-fragment", parsePanelProps.MarkdownBody),
		),
	)
}
