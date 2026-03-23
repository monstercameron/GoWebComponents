//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

const innerHTMLProperty = "__gwc_prop__:innerHTML"
const activeDemoAnchorProperty = "__gwc_active_demo_anchor__"
const demoAnchorScrollRequestProperty = "__gwc_demo_anchor_scroll_request__"
const apiReferenceSearchProperty = "__gwc_api_reference_search__"

func setRequestedDemoAnchor(anchorID string) {
	window := js.Global().Get("window")
	if window.IsUndefined() || window.IsNull() {
		return
	}
	window.Set(demoAnchorScrollRequestProperty, anchorID)
}

func clearRequestedDemoAnchor() {
	setRequestedDemoAnchor("")
}

func findScrollableAncestor(target js.Value) js.Value {
	if target.IsUndefined() || target.IsNull() {
		return js.Undefined()
	}
	current := target.Get("parentElement")
	for !current.IsUndefined() && !current.IsNull() {
		style := js.Global().Get("window").Call("getComputedStyle", current)
		overflowY := style.Get("overflowY").String()
		isScrollableOverflow := overflowY == "auto" || overflowY == "scroll"
		if isScrollableOverflow && current.Get("scrollHeight").Float() > current.Get("clientHeight").Float()+1 {
			return current
		}
		current = current.Get("parentElement")
	}
	return js.Undefined()
}

func isGroupedAPIItem(item docsItem) bool {
	return item.Type == kindAPI && item.Content.AnchorID != "" && item.Content.SourcePath != ""
}

func apiReferenceDocumentURL(item docsItem) string {
	if item.Content.SourcePath == "" {
		return ""
	}
	url := docsSourceURL(item.Content.SourcePath)
	if item.Content.AnchorID != "" {
		url += "#" + item.Content.AnchorID
	}
	return url
}

func scrollToDemoAnchor(anchorID string, ready bool, body string) {
	ui.UseEffect(func() func() {
		if !ready || body == "" || anchorID == "" {
			return nil
		}
		window := js.Global().Get("window")
		if window.IsUndefined() || window.IsNull() {
			return nil
		}
		requestedAnchor := window.Get(demoAnchorScrollRequestProperty)
		if requestedAnchor.IsUndefined() || requestedAnchor.IsNull() || requestedAnchor.String() != anchorID {
			return nil
		}
		window.Set(activeDemoAnchorProperty, anchorID)
		attemptScroll := func() {
			currentAnchor := window.Get(activeDemoAnchorProperty)
			if currentAnchor.IsUndefined() || currentAnchor.IsNull() || currentAnchor.String() != anchorID {
				return
			}
			document := js.Global().Get("document")
			if document.IsUndefined() || document.IsNull() {
				return
			}
			container := document.Call("querySelector", "#demo")
			selector := fmt.Sprintf("#demo #%s", anchorID)
			target := document.Call("querySelector", selector)
			if container.IsUndefined() || container.IsNull() || target.IsUndefined() || target.IsNull() {
				return
			}
			if target.Get("scrollIntoView").Type() == js.TypeFunction {
				target.Call("scrollIntoView", true)
			}
			targetRect := target.Call("getBoundingClientRect")
			if targetRect.Get("top").Float() >= 0 && targetRect.Get("top").Float() <= 48 {
				return
			}
			scrollContainer := findScrollableAncestor(target)
			if !scrollContainer.IsUndefined() && !scrollContainer.IsNull() {
				containerRect := scrollContainer.Call("getBoundingClientRect")
				nextTop := scrollContainer.Get("scrollTop").Float() + targetRect.Get("top").Float() - containerRect.Get("top").Float() - 24
				if scrollContainer.Get("scrollTo").Type() == js.TypeFunction {
					scrollContainer.Call("scrollTo", 0, nextTop)
					return
				}
				scrollContainer.Set("scrollTop", nextTop)
				return
			}
			if window.IsUndefined() || window.IsNull() {
				return
			}
			nextWindowTop := window.Get("scrollY").Float() + targetRect.Get("top").Float() - 24
			if window.Get("scrollTo").Type() == js.TypeFunction {
				window.Call("scrollTo", 0, nextWindowTop)
				return
			}
			documentElement := document.Get("documentElement")
			if !documentElement.IsUndefined() && !documentElement.IsNull() {
				documentElement.Set("scrollTop", nextWindowTop)
			}
		}

		attemptScroll()

		delays := []time.Duration{16 * time.Millisecond, 80 * time.Millisecond, 180 * time.Millisecond, 320 * time.Millisecond, 520 * time.Millisecond, 900 * time.Millisecond, 1400 * time.Millisecond, 2 * time.Second}
		timers := make([]interop.Timer, 0, len(delays))
		for _, delay := range delays {
			timer, err := interop.SetTimeout(delay, attemptScroll)
			if err != nil {
				continue
			}
			timers = append(timers, timer)
		}
		if len(timers) == 0 {
			return nil
		}
		return func() {
			for _, timer := range timers {
				_ = timer.Cancel()
			}
		}
	}, anchorID, ready, len(body))
}

func renderInjectedHTMLFragment(className, body string) ui.Node {
	return Tag("div", FromProps(Props{Class: className, Raw: map[string]interface{}{innerHTMLProperty: body}}))
}

func filterAPIReferenceSections(query string, ready bool, body string) {
	ui.UseEffect(func() func() {
		if !ready || body == "" {
			return nil
		}
		window := js.Global().Get("window")
		if window.IsUndefined() || window.IsNull() {
			return nil
		}
		normalizedQuery := strings.TrimSpace(strings.ToLower(query))
		window.Set(apiReferenceSearchProperty, normalizedQuery)
		applyFilter := func() {
			currentQuery := window.Get(apiReferenceSearchProperty)
			if currentQuery.IsUndefined() || currentQuery.IsNull() || currentQuery.String() != normalizedQuery {
				return
			}
			document := js.Global().Get("document")
			if document.IsUndefined() || document.IsNull() {
				return
			}
			fragment := document.Call("querySelector", "#demo .api-reference-fragment")
			if fragment.IsUndefined() || fragment.IsNull() {
				return
			}
			apiItems := fragment.Call("querySelectorAll", ".api-item")
			itemCount := apiItems.Get("length").Int()
			for index := 0; index < itemCount; index++ {
				item := apiItems.Index(index)
				matches := normalizedQuery == "" || strings.Contains(strings.ToLower(item.Get("textContent").String()), normalizedQuery)
				if matches {
					item.Get("style").Set("display", "")
				} else {
					item.Get("style").Set("display", "none")
				}
			}

			articles := fragment.Call("querySelectorAll", "article")
			articleCount := articles.Get("length").Int()
			for index := 0; index < articleCount; index++ {
				article := articles.Index(index)
				itemNodes := article.Call("querySelectorAll", ".api-item")
				itemNodeCount := itemNodes.Get("length").Int()
				if itemNodeCount == 0 {
					article.Get("style").Set("display", "")
					continue
				}
				hasVisibleItem := false
				for itemIndex := 0; itemIndex < itemNodeCount; itemIndex++ {
					if itemNodes.Index(itemIndex).Get("style").Get("display").String() != "none" {
						hasVisibleItem = true
						break
					}
				}
				if hasVisibleItem {
					article.Get("style").Set("display", "")
				} else {
					article.Get("style").Set("display", "none")
				}
			}

			sections := fragment.Call("querySelectorAll", "section")
			sectionCount := sections.Get("length").Int()
			for index := 0; index < sectionCount; index++ {
				section := sections.Index(index)
				if normalizedQuery == "" {
					section.Get("style").Set("display", "")
					continue
				}
				matchesHeading := strings.Contains(strings.ToLower(section.Get("textContent").String()), normalizedQuery)
				visibleItems := section.Call("querySelectorAll", ".api-item")
				visibleCount := visibleItems.Get("length").Int()
				hasVisibleItem := false
				for itemIndex := 0; itemIndex < visibleCount; itemIndex++ {
					if visibleItems.Index(itemIndex).Get("style").Get("display").String() != "none" {
						hasVisibleItem = true
						break
					}
				}
				if matchesHeading || hasVisibleItem {
					section.Get("style").Set("display", "")
				} else {
					section.Get("style").Set("display", "none")
				}
			}
		}
		delays := []time.Duration{16 * time.Millisecond, 80 * time.Millisecond, 180 * time.Millisecond}
		timers := make([]interop.Timer, 0, len(delays))
		for _, delay := range delays {
			timer, err := interop.SetTimeout(delay, applyFilter)
			if err != nil {
				continue
			}
			timers = append(timers, timer)
		}
		return func() {
			for _, timer := range timers {
				_ = timer.Cancel()
			}
		}
	}, query, ready, len(body))
}

func renderGroupedAPIReference(panelProps contentPanelProps) ui.Node {
	apiSearchQuery := ui.UseState("")
	updateAPISearchQuery := ui.UseEvent(func(event ui.InputEvent) {
		apiSearchQuery.Set(event.GetValue())
	})
	scrollToDemoAnchor(panelProps.Item.Content.AnchorID, panelProps.MarkdownReady, panelProps.MarkdownBody)
	filterAPIReferenceSections(apiSearchQuery.Get(), panelProps.MarkdownReady, panelProps.MarkdownBody)

	return Div(Class("min-w-0 flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-3 shadow-inner shadow-black/20"),
		Div(Class("border-b border-white/10 pb-2"),
			Div(Class("text-sm font-medium text-white"), Text(labelAPIReference)),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text("Catalog-owned grouped API reference document")),
		),
		Div(Class("mt-3 rounded-[20px] border border-white/10 bg-white/[0.04] p-3"),
			Div(Class("text-xs uppercase tracking-[0.18em] text-slate-500"), Text(labelReferenceSearch)),
			Input(
				Value(apiSearchQuery.Get()),
				OnInput(updateAPISearchQuery),
				Placeholder(messageReferenceSearch),
				Class("mt-2 w-full rounded-xl border border-white/10 bg-slate-950/40 px-3 py-2 text-sm text-white outline-none placeholder:text-slate-500 transition focus:border-cyan-300/40 focus:bg-slate-950/60"),
			),
		),
		Div(Class("mt-3 rounded-[22px] border border-violet-400/20 bg-violet-400/10 p-3"),
			Div(Class("text-xs uppercase tracking-[0.18em] text-violet-200"), Text("Reference scope")),
			P(Class("mt-2 text-sm leading-7 text-violet-50"), Text("This API group is defined in the catalog and resolves to a shared HTML reference fragment. Selecting a card fetches the fragment into this surface and then scrolls to the matching section anchor.")),
			Div(Class("mt-2 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm leading-6 text-cyan-100 break-all"), Text(apiReferenceDocumentURL(panelProps.Item))),
		),
		Div(Class("mt-3 min-h-[72vh] rounded-[22px] border border-white/10 bg-[#08111f] p-2"),
			renderInjectedHTMLFragment("api-reference-fragment", panelProps.MarkdownBody),
		),
	)
}
