//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// conditionResult is the JSON payload for expect/wait.
type conditionResult struct {
	Holds      bool   `json:"holds"`
	Condition  string `json:"condition"`
	Selector   string `json:"selector,omitempty"`
	Detail     string `json:"detail,omitempty"`
	WaitedMs   int64  `json:"waitedMs"`
	TimeoutMs  int64  `json:"timeoutMs"`
	Attached   bool   `json:"attached"`
}

// proxyCondition describes a DOM/page assertion shared by expect and wait.
type proxyCondition struct {
	selector string
	state    string // visible | attached | hidden | detached
	text     string
	count    int // -1 when unset
	eval     string
}

// truthyJS reports whether a JS-evaluated value is truthy by JS-ish rules.
func truthyJS(parseV any) bool {
	switch parseT := parseV.(type) {
	case nil:
		return false
	case bool:
		return parseT
	case float64:
		return parseT != 0
	case int:
		return parseT != 0
	case int64:
		return parseT != 0
	case string:
		return parseT != ""
	case []any:
		return len(parseT) > 0
	case map[string]any:
		return len(parseT) > 0
	default:
		return true
	}
}

// evalProxyCondition blocks until the condition holds or the timeout elapses,
// returning whether it holds plus a human detail. Selector presence/visibility
// uses Locator.WaitFor; text/count/eval poll until the deadline.
func evalProxyCondition(parsePage playwright.Page, parseCond proxyCondition, parseTimeout time.Duration) (bool, string, error) {
	parseDeadline := time.Now().Add(parseTimeout)

	// eval has highest precedence: poll the expression for truthiness.
	if strings.TrimSpace(parseCond.eval) != "" {
		for {
			parseVal, parseErr := parsePage.Evaluate(parseCond.eval)
			if parseErr == nil && truthyJS(parseVal) {
				return true, fmt.Sprintf("eval truthy: %v", parseVal), nil
			}
			if time.Now().After(parseDeadline) {
				return false, "eval not truthy before timeout", nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// count: selector match count equals N.
	if parseCond.count >= 0 {
		if strings.TrimSpace(parseCond.selector) == "" {
			return false, "", fmt.Errorf("-count requires -selector")
		}
		for {
			parseN, parseErr := parsePage.Locator(parseCond.selector).Count()
			if parseErr == nil && parseN == parseCond.count {
				return true, fmt.Sprintf("selector count == %d", parseCond.count), nil
			}
			if time.Now().After(parseDeadline) {
				parseGot, _ := parsePage.Locator(parseCond.selector).Count()
				return false, fmt.Sprintf("selector count == %d before timeout (last %d)", parseCond.count, parseGot), nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// text: page (or selector) contains substring.
	if strings.TrimSpace(parseCond.text) != "" {
		for {
			var parseHay string
			var parseErr error
			if strings.TrimSpace(parseCond.selector) != "" {
				parseHay, parseErr = parsePage.Locator(parseCond.selector).First().InnerText()
			} else {
				parseHay, parseErr = parsePage.InnerText("body")
			}
			if parseErr == nil && strings.Contains(parseHay, parseCond.text) {
				return true, "text found", nil
			}
			if time.Now().After(parseDeadline) {
				return false, fmt.Sprintf("text %q not found before timeout", parseCond.text), nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// selector presence/visibility via WaitFor.
	if strings.TrimSpace(parseCond.selector) != "" {
		parseState := playwright.WaitForSelectorStateVisible
		switch parseCond.state {
		case "attached":
			parseState = playwright.WaitForSelectorStateAttached
		case "hidden":
			parseState = playwright.WaitForSelectorStateHidden
		case "detached":
			parseState = playwright.WaitForSelectorStateDetached
		case "", "visible":
			parseState = playwright.WaitForSelectorStateVisible
		default:
			return false, "", fmt.Errorf("unknown -state %q", parseCond.state)
		}
		parseErr := parsePage.Locator(parseCond.selector).WaitFor(playwright.LocatorWaitForOptions{
			State:   parseState,
			Timeout: playwright.Float(float64(parseTimeout.Milliseconds())),
		})
		if parseErr == nil {
			return true, fmt.Sprintf("selector %s", string(*parseState)), nil
		}
		return false, fmt.Sprintf("selector %s not %s before timeout", parseCond.selector, string(*parseState)), nil
	}

	return false, "", fmt.Errorf("no condition given: provide -selector, -text, -count, or -eval")
}

// describeCondition renders a short label for the result payload.
func describeCondition(parseCond proxyCondition) string {
	switch {
	case parseCond.eval != "":
		return "eval:" + parseCond.eval
	case parseCond.count >= 0:
		return fmt.Sprintf("count==%d", parseCond.count)
	case parseCond.text != "":
		return "text:" + parseCond.text
	case parseCond.selector != "":
		parseState := parseCond.state
		if parseState == "" {
			parseState = "visible"
		}
		return "selector " + parseState
	default:
		return "(none)"
	}
}

// runConditionVerb is the shared body for expect and wait.
func runConditionVerb(parseName string, parseDefaultTimeout time.Duration, parseDefaultState string, parseArgs []string) error {
	parseFs := flag.NewFlagSet(parseName, flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint")
	parseSelector := parseFs.String("selector", "", "CSS selector to check")
	parseVisible := parseFs.Bool("visible", false, "require the selector to be visible")
	parseHidden := parseFs.Bool("hidden", false, "require the selector to be hidden")
	parseState := parseFs.String("state", parseDefaultState, "selector state: visible|attached|hidden|detached")
	parseText := parseFs.String("text", "", "require this text in the selector (or page body)")
	parseCount := parseFs.Int("count", -1, "require the selector match count to equal N")
	parseEval := parseFs.String("eval", "", "require this JS expression to be truthy")
	parseTimeout := parseFs.Duration("timeout", parseDefaultTimeout, "max time to wait for the condition")
	parseFs.Int("width", 1280, "viewport width (launch mode)")
	parseFs.Int("height", 800, "viewport height (launch mode)")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope(parseName, false, nil, nil, fmt.Errorf("%s: -url or -cdp is required", parseName))
	}
	parseEffState := *parseState
	if *parseVisible {
		parseEffState = "visible"
	}
	if *parseHidden {
		parseEffState = "hidden"
	}

	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, 1280, 800)
	if parseErr != nil {
		return writeAgenticEnvelope(parseName, false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseCond := proxyCondition{
		selector: strings.TrimSpace(*parseSelector),
		state:    parseEffState,
		text:     *parseText,
		count:    *parseCount,
		eval:     *parseEval,
	}
	parseStart := time.Now()
	parseHolds, parseDetail, parseErr := evalProxyCondition(parsePage.page, parseCond, *parseTimeout)
	parseWaited := time.Since(parseStart)
	if parseErr != nil {
		return writeAgenticEnvelope(parseName, false, nil, nil, parseErr)
	}
	parseResult := conditionResult{
		Holds:     parseHolds,
		Condition: describeCondition(parseCond),
		Selector:  parseCond.selector,
		Detail:    parseDetail,
		WaitedMs:  parseWaited.Milliseconds(),
		TimeoutMs: parseTimeout.Milliseconds(),
		Attached:  parsePage.attached,
	}
	// ok mirrors whether the condition held, so an agent can gate directly.
	return writeAgenticEnvelope(parseName, parseHolds, parseResult, nil, nil)
}

// runExpectCommand asserts a DOM/page condition holds now (within a short
// timeout) and returns ok = whether it holds — a gateable verify primitive.
var runExpectCommand = func(parseL launcher, parseArgs []string) error {
	return runConditionVerb("expect", 5*time.Second, "visible", parseArgs)
}

// runWaitCommand blocks until a DOM condition holds (longer default timeout) —
// the browser counterpart to bridge wait-for.
var runWaitCommand = func(parseL launcher, parseArgs []string) error {
	return runConditionVerb("wait", 30*time.Second, "visible", parseArgs)
}
