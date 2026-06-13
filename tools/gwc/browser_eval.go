//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"strings"
)

// domResult is the JSON payload returned by `gwc dom`.
type domResult struct {
	Selector  string            `json:"selector"`
	Text      string            `json:"text"`
	HTML      string            `json:"html"`
	Attrs     map[string]string `json:"attrs,omitempty"`
	Attached  bool              `json:"attached"`
}

// evalResult is the JSON payload returned by `gwc eval`.
type evalResult struct {
	Expression string `json:"expression"`
	Value      any    `json:"value"`
	Attached   bool   `json:"attached"`
}

// domReadAttrs is the small allowlist of attributes `gwc dom` reports.
var domReadAttrs = []string{"id", "class", "href", "src", "alt", "title", "role", "aria-label", "type", "name", "value", "placeholder"}

// runDomCommand reads the REAL rendered DOM for a selector (text, outerHTML, a
// few attributes) — beyond what the GWC fiber-tree snapshot can see. Read-only.
var runDomCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("dom", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch and read (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint (the engineer's open gwc browser window)")
	parseSelector := parseFs.String("selector", "", "CSS selector to read (required)")
	parseWidth := parseFs.Int("width", 1280, "viewport width (launch mode)")
	parseHeight := parseFs.Int("height", 800, "viewport height (launch mode)")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseSelector) == "" {
		return writeAgenticEnvelope("dom", false, nil, nil, fmt.Errorf("dom: -selector is required"))
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("dom", false, nil, nil, fmt.Errorf("dom: -url or -cdp is required"))
	}
	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, *parseWidth, *parseHeight)
	if parseErr != nil {
		return writeAgenticEnvelope("dom", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseLoc := parsePage.page.Locator(*parseSelector).First()
	parseText, parseErr := parseLoc.InnerText()
	if parseErr != nil {
		return writeAgenticEnvelope("dom", false, nil, nil, fmt.Errorf("read %s: %w", *parseSelector, parseErr))
	}
	parseHTML, _ := parseLoc.InnerHTML()
	parseAttrs := map[string]string{}
	for _, parseName := range domReadAttrs {
		if parseVal, parseErr := parseLoc.GetAttribute(parseName); parseErr == nil && parseVal != "" {
			parseAttrs[parseName] = parseVal
		}
	}
	parseResult := domResult{
		Selector: *parseSelector,
		Text:     parseText,
		HTML:     parseHTML,
		Attrs:    parseAttrs,
		Attached: parsePage.attached,
	}
	return writeAgenticEnvelope("dom", true, parseResult, nil, nil)
}

// runEvalCommand evaluates a JavaScript expression in the page and returns the
// result. Dual-use (JS can mutate the page); gated by the same loopback+token
// surface as the rest of the proxy and excluded from release builds. Prefer
// `gwc dom` for plain reads.
var runEvalCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("eval", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch and evaluate against (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint (the engineer's open gwc browser window)")
	parseExpr := parseFs.String("expr", "", "JavaScript expression to evaluate (required)")
	parseWidth := parseFs.Int("width", 1280, "viewport width (launch mode)")
	parseHeight := parseFs.Int("height", 800, "viewport height (launch mode)")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseExpr) == "" {
		return writeAgenticEnvelope("eval", false, nil, nil, fmt.Errorf("eval: -expr is required"))
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("eval", false, nil, nil, fmt.Errorf("eval: -url or -cdp is required"))
	}
	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, *parseWidth, *parseHeight)
	if parseErr != nil {
		return writeAgenticEnvelope("eval", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseValue, parseErr := parsePage.page.Evaluate(*parseExpr)
	if parseErr != nil {
		return writeAgenticEnvelope("eval", false, nil, nil, fmt.Errorf("evaluate: %w", parseErr))
	}
	parseResult := evalResult{Expression: *parseExpr, Value: parseValue, Attached: parsePage.attached}
	return writeAgenticEnvelope("eval", true, parseResult, nil, nil)
}
