//go:build js && wasm

package app

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/ui"
)

const pricingTopSectionID = "pricing-top"

// parseNormalizePricingFragmentID returns the supported pricing-page fragment identifier.
func parseNormalizePricingFragmentID(parseHash string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseHash))
	parseNormalized = strings.TrimPrefix(parseNormalized, "#")
	switch parseNormalized {
	case "plans", "compare", "faq", "contact":
		return parseNormalized
	default:
		return ""
	}
}

// parseResolvePricingFragmentScroll returns the target pricing section ID or whether the page should reset to top.
func parseResolvePricingFragmentScroll(parseCurrentPath, parseHash string) (string, bool) {
	if strings.TrimSpace(parseCurrentPath) != marketingPricingRoute {
		return "", false
	}
	parseNormalized := parseNormalizePricingFragmentID(parseHash)
	if parseNormalized != "" {
		return parseNormalized, false
	}
	parseTrimmedHash := strings.TrimSpace(parseHash)
	return "", parseTrimmedHash == "" || parseTrimmedHash == "#"
}

// parseCurrentLocationHash returns the current browser location hash.
func parseCurrentLocationHash() string {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return ""
	}
	parseLocation := parseWindow.Get("location")
	if !parseLocation.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseLocation.Get("hash").String())
}

// parseScrollPricingFragment applies pricing-page fragment scrolling for initial load and history navigation.
func parseScrollPricingFragment(parseCurrentPath, parseHash string) {
	parseTargetID, shouldParseScrollTop := parseResolvePricingFragmentScroll(parseCurrentPath, parseHash)
	if shouldParseScrollTop {
		parseTargetID = pricingTopSectionID
	}
	if parseTargetID == "" {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() || parseDocument.Get("getElementById").Type() != js.TypeFunction {
		return
	}
	parseTarget := parseDocument.Call("getElementById", parseTargetID)
	if !parseTarget.Truthy() || parseTarget.Get("scrollIntoView").Type() != js.TypeFunction {
		return
	}
	parseTarget.Call("scrollIntoView", map[string]interface{}{
		"behavior": "auto",
		"block":    "start",
	})
}

// parseSchedulePricingFragmentScroll applies pricing-page fragment scrolling after the next paint when possible.
func parseSchedulePricingFragmentScroll(parseCurrentPath, parseHash string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		parseScrollPricingFragment(parseCurrentPath, parseHash)
		return
	}
	parseScheduleTimeout := func(parseDelayMS int) {
		if parseWindow.Get("setTimeout").Type() != js.TypeFunction {
			parseScrollPricingFragment(parseCurrentPath, parseHash)
			return
		}
		var parseCallback js.Func
		parseCallback = js.FuncOf(func(js.Value, []js.Value) interface{} {
			parseScrollPricingFragment(parseCurrentPath, parseHash)
			parseCallback.Release()
			return nil
		})
		parseWindow.Call("setTimeout", parseCallback, parseDelayMS)
	}
	if parseWindow.Get("requestAnimationFrame").Type() != js.TypeFunction {
		parseScheduleTimeout(0)
		parseScheduleTimeout(120)
		return
	}
	var parseFrame js.Func
	parseFrame = js.FuncOf(func(js.Value, []js.Value) interface{} {
		parseScheduleTimeout(0)
		parseScheduleTimeout(120)
		parseFrame.Release()
		return nil
	})
	parseWindow.Call("requestAnimationFrame", parseFrame)
}

// parseUsePricingFragmentScroll keeps pricing-page hash anchors working on initial load and browser history navigation.
func parseUsePricingFragmentScroll(parseCurrentPath string) {
	ui.UseEffect(func() func() {
		if strings.TrimSpace(parseCurrentPath) != marketingPricingRoute {
			return nil
		}
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() {
			return nil
		}

		// Run once after the pricing DOM mounts, then keep fragment-driven scroll in sync with history changes.
		parseApplyScroll := func() {
			parseSchedulePricingFragmentScroll(parseCurrentPath, parseCurrentLocationHash())
		}
		parseApplyScroll()

		parseHashListener := js.FuncOf(func(js.Value, []js.Value) interface{} {
			parseApplyScroll()
			return nil
		})
		parsePopListener := js.FuncOf(func(js.Value, []js.Value) interface{} {
			parseApplyScroll()
			return nil
		})
		parseWindow.Call("addEventListener", "hashchange", parseHashListener)
		parseWindow.Call("addEventListener", "popstate", parsePopListener)
		return func() {
			parseWindow.Call("removeEventListener", "hashchange", parseHashListener)
			parseWindow.Call("removeEventListener", "popstate", parsePopListener)
			parseHashListener.Release()
			parsePopListener.Release()
		}
	}, parseCurrentPath)
}
