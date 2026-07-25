//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// parseCase holds a single formatting test case for the fixture matrix.
type parseCase struct {
	parseLocale string
	parseValue  float64
	parseOpts   interop.IntlNumberOptions
	// parseJSOpts is the JSON-serialisable form of parseOpts for data-opts.
	parseJSOpts map[string]interface{}
}

// groupingPtr returns a *bool for the UseGrouping option.
func groupingPtr(parseValue bool) *bool { return &parseValue }

// parseCases is the fixed matrix of NumberFormat scenarios exercised by the
// fixture. Each entry exercises a distinct locale/style/digit combination.
//
// parseJSOpts mirrors EXACTLY the JS options bag the GWC bridge sends, so the
// cross-check is apples-to-apples. The bridge now only writes useGrouping when
// the caller sets it (UseGrouping is a *bool): an unset value omits the key so
// the browser default (grouping on) applies, and an explicit value is forwarded.
var parseCases = []parseCase{
	{
		parseLocale: "en-US",
		parseValue:  1234567.89,
		parseOpts:   interop.IntlNumberOptions{},
		parseJSOpts: map[string]interface{}{}, // default grouping (thousands separators) on
	},
	{
		parseLocale: "de-DE",
		parseValue:  1234567.89,
		parseOpts:   interop.IntlNumberOptions{},
		parseJSOpts: map[string]interface{}{},
	},
	{
		parseLocale: "en-US",
		parseValue:  0.4567,
		parseOpts:   interop.IntlNumberOptions{Style: "percent"},
		parseJSOpts: map[string]interface{}{"style": "percent"},
	},
	{
		parseLocale: "ja-JP",
		parseValue:  1234567,
		parseOpts:   interop.IntlNumberOptions{},
		parseJSOpts: map[string]interface{}{},
	},
	{
		parseLocale: "fr-FR",
		parseValue:  1234.5,
		parseOpts:   interop.IntlNumberOptions{MinimumFractionDigits: 2, MaximumFractionDigits: 2},
		parseJSOpts: map[string]interface{}{"minimumFractionDigits": 2, "maximumFractionDigits": 2},
	},
	{
		// Explicit UseGrouping=false must suppress separators (override path).
		parseLocale: "en-US",
		parseValue:  1234567.89,
		parseOpts:   interop.IntlNumberOptions{UseGrouping: groupingPtr(false)},
		parseJSOpts: map[string]interface{}{"useGrouping": false},
	},
}

func main() {
	parseDoc := js.Global().Get("document")
	parseBody := parseDoc.Get("body")

	parseIntlAvailStr := "false"
	if interop.IntlAvailable() {
		parseIntlAvailStr = "true"
	}

	for parseIdx, parseC := range parseCases {
		// Format the number via the GWC interop bridge.
		parseGWCResult, parseErr := interop.IntlFormatNumber(parseC.parseLocale, parseC.parseValue, parseC.parseOpts)
		if parseErr != nil {
			parseGWCResult = fmt.Sprintf("ERROR:%v", parseErr)
		}

		// Serialise the JS options bag to JSON for the data-opts attribute.
		parseOptsJSON, parseJSONErr := json.Marshal(parseC.parseJSOpts)
		if parseJSONErr != nil {
			parseOptsJSON = []byte("{}")
		}

		// Build the div element.
		parseDiv := parseDoc.Call("createElement", "div")
		parseDiv.Call("setAttribute", "class", "intl-case")
		parseDiv.Call("setAttribute", "data-idx", strconv.Itoa(parseIdx))
		parseDiv.Call("setAttribute", "data-locale", parseC.parseLocale)
		parseDiv.Call("setAttribute", "data-value", strconv.FormatFloat(parseC.parseValue, 'f', -1, 64))
		parseDiv.Call("setAttribute", "data-opts", string(parseOptsJSON))
		parseDiv.Call("setAttribute", "data-gwc", parseGWCResult)
		parseDiv.Call("setAttribute", "data-intl-available", parseIntlAvailStr)

		parseBody.Call("appendChild", parseDiv)
	}

	// Append the sentinel so the Playwright test knows all cases are rendered.
	parseDone := parseDoc.Call("createElement", "div")
	parseDone.Call("setAttribute", "id", "intl-done")
	parseDone.Set("textContent", "ready")
	parseBody.Call("appendChild", parseDone)

	// Block forever — standard pattern for a wasm main that drives the page.
	select {}
}
