//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"strings"

	playwright "github.com/mxschmitt/playwright-go"
)

// inputResult is the JSON payload returned by the input verbs.
type inputResult struct {
	Action   string  `json:"action"`
	Selector string  `json:"selector,omitempty"`
	Text     string  `json:"text,omitempty"`
	Key      string  `json:"key,omitempty"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
	Attached bool    `json:"attached"`
}

// inputFlags holds the flags common to every input verb.
type inputFlags struct {
	fs       *flag.FlagSet
	url      *string
	cdp      *string
	selector *string
	width    *int
	height   *int
}

func newInputFlags(parseName string) *inputFlags {
	parseFs := flag.NewFlagSet(parseName, flag.ContinueOnError)
	return &inputFlags{
		fs:       parseFs,
		url:      parseFs.String("url", "", "URL to launch and act on (unless -cdp is set)"),
		cdp:      parseFs.String("cdp", "", "attach to a running browser's CDP endpoint (the engineer's open gwc browser window)"),
		selector: parseFs.String("selector", "", "target CSS selector"),
		width:    parseFs.Int("width", 1280, "viewport width (launch mode)"),
		height:   parseFs.Int("height", 800, "viewport height (launch mode)"),
	}
}

// resolveInputPage parses flags and opens the proxy page for an input verb.
func (parseF *inputFlags) resolve(parseName string, parseArgs []string) (*proxyPage, error) {
	if parseErr := parseF.fs.Parse(parseArgs); parseErr != nil {
		return nil, parseErr
	}
	if strings.TrimSpace(*parseF.url) == "" && strings.TrimSpace(*parseF.cdp) == "" {
		return nil, fmt.Errorf("%s: -url or -cdp is required", parseName)
	}
	return openProxyPage(*parseF.cdp, *parseF.url, *parseF.width, *parseF.height)
}

// runClickCommand clicks a selector (or x/y coordinates) on the live DOM.
var runClickCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("click")
	parseX := parseFlags.fs.Float64("x", -1, "click x coordinate (when no -selector)")
	parseY := parseFlags.fs.Float64("y", -1, "click y coordinate (when no -selector)")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("click", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("click", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel != "" {
		if parseErr := parsePage.page.Locator(parseSel).Click(); parseErr != nil {
			return writeAgenticEnvelope("click", false, nil, nil, fmt.Errorf("click %s: %w", parseSel, parseErr))
		}
	} else if *parseX >= 0 && *parseY >= 0 {
		if parseErr := parsePage.page.Mouse().Click(*parseX, *parseY); parseErr != nil {
			return writeAgenticEnvelope("click", false, nil, nil, fmt.Errorf("click at %v,%v: %w", *parseX, *parseY, parseErr))
		}
	} else {
		return writeAgenticEnvelope("click", false, nil, nil, fmt.Errorf("click: provide -selector or -x/-y"))
	}
	return writeAgenticEnvelope("click", true, inputResult{Action: "click", Selector: parseSel, X: maxZero(*parseX), Y: maxZero(*parseY), Attached: parsePage.attached}, nil, nil)
}

// runTypeCommand fills (or appends to) a text input on the live DOM.
var runTypeCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("type")
	parseText := parseFlags.fs.String("text", "", "text to enter into the selector")
	parseAppend := parseFlags.fs.Bool("append", false, "type keystroke-by-keystroke and keep existing value instead of replacing")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("type", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("type", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel == "" {
		return writeAgenticEnvelope("type", false, nil, nil, fmt.Errorf("type: -selector is required"))
	}
	parseLoc := parsePage.page.Locator(parseSel)
	if *parseAppend {
		if parseErr := parseLoc.PressSequentially(*parseText); parseErr != nil {
			return writeAgenticEnvelope("type", false, nil, nil, fmt.Errorf("type into %s: %w", parseSel, parseErr))
		}
	} else {
		if parseErr := parseLoc.Fill(*parseText); parseErr != nil {
			return writeAgenticEnvelope("type", false, nil, nil, fmt.Errorf("fill %s: %w", parseSel, parseErr))
		}
	}
	return writeAgenticEnvelope("type", true, inputResult{Action: "type", Selector: parseSel, Text: *parseText, Attached: parsePage.attached}, nil, nil)
}

// runPressCommand sends a key/chord to the focused element or a selector.
var runPressCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("press")
	parseKey := parseFlags.fs.String("key", "", "key or chord to press, e.g. Enter, Tab, Control+A")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("press", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("press", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	if strings.TrimSpace(*parseKey) == "" {
		return writeAgenticEnvelope("press", false, nil, nil, fmt.Errorf("press: -key is required"))
	}
	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel != "" {
		if parseErr := parsePage.page.Locator(parseSel).Press(*parseKey); parseErr != nil {
			return writeAgenticEnvelope("press", false, nil, nil, fmt.Errorf("press %s on %s: %w", *parseKey, parseSel, parseErr))
		}
	} else {
		if parseErr := parsePage.page.Keyboard().Press(*parseKey); parseErr != nil {
			return writeAgenticEnvelope("press", false, nil, nil, fmt.Errorf("press %s: %w", *parseKey, parseErr))
		}
	}
	return writeAgenticEnvelope("press", true, inputResult{Action: "press", Selector: parseSel, Key: *parseKey, Attached: parsePage.attached}, nil, nil)
}

// runHoverCommand hovers a selector on the live DOM.
var runHoverCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("hover")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("hover", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("hover", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel == "" {
		return writeAgenticEnvelope("hover", false, nil, nil, fmt.Errorf("hover: -selector is required"))
	}
	if parseErr := parsePage.page.Locator(parseSel).Hover(); parseErr != nil {
		return writeAgenticEnvelope("hover", false, nil, nil, fmt.Errorf("hover %s: %w", parseSel, parseErr))
	}
	return writeAgenticEnvelope("hover", true, inputResult{Action: "hover", Selector: parseSel, Attached: parsePage.attached}, nil, nil)
}

// runScrollCommand scrolls a selector into view, or the window by x/y deltas.
var runScrollCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("scroll")
	parseDX := parseFlags.fs.Float64("x", 0, "horizontal scroll delta (when no -selector)")
	parseDY := parseFlags.fs.Float64("y", 0, "vertical scroll delta (when no -selector)")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("scroll", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("scroll", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel != "" {
		if parseErr := parsePage.page.Locator(parseSel).ScrollIntoViewIfNeeded(); parseErr != nil {
			return writeAgenticEnvelope("scroll", false, nil, nil, fmt.Errorf("scroll to %s: %w", parseSel, parseErr))
		}
	} else {
		if _, parseErr := parsePage.page.Evaluate(fmt.Sprintf("() => window.scrollBy(%v, %v)", *parseDX, *parseDY)); parseErr != nil {
			return writeAgenticEnvelope("scroll", false, nil, nil, fmt.Errorf("scroll by %v,%v: %w", *parseDX, *parseDY, parseErr))
		}
	}
	return writeAgenticEnvelope("scroll", true, inputResult{Action: "scroll", Selector: parseSel, X: *parseDX, Y: *parseDY, Attached: parsePage.attached}, nil, nil)
}

// maxZero clamps a sentinel -1 coordinate to 0 for the result payload.
func maxZero(parseV float64) float64 {
	if parseV < 0 {
		return 0
	}
	return parseV
}

// runSelectCommand selects option(s) in a <select> by value or label.
var runSelectCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("select")
	parseValue := parseFlags.fs.String("value", "", "option value to select")
	parseLabel := parseFlags.fs.String("label", "", "option visible label to select (when no -value)")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("select", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("select", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel == "" {
		return writeAgenticEnvelope("select", false, nil, nil, fmt.Errorf("select: -selector is required"))
	}
	if strings.TrimSpace(*parseValue) == "" && strings.TrimSpace(*parseLabel) == "" {
		return writeAgenticEnvelope("select", false, nil, nil, fmt.Errorf("select: -value or -label is required"))
	}
	parseValues := selectOptionByValueOrLabel(*parseValue, *parseLabel)
	if _, parseErr := parsePage.page.Locator(parseSel).SelectOption(parseValues); parseErr != nil {
		return writeAgenticEnvelope("select", false, nil, nil, fmt.Errorf("select %s: %w", parseSel, parseErr))
	}
	parseChosen := *parseValue
	if parseChosen == "" {
		parseChosen = *parseLabel
	}
	return writeAgenticEnvelope("select", true, inputResult{Action: "select", Selector: parseSel, Text: parseChosen, Attached: parsePage.attached}, nil, nil)
}

// runUploadCommand sets file(s) on a file input.
var runUploadCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newInputFlags("upload")
	var parseFiles multiFlag
	parseFlags.fs.Var(&parseFiles, "file", "file path to upload (repeatable)")
	parseFlags.fs.Bool("json", false, "emit the JSON envelope")
	parsePage, parseErr := parseFlags.resolve("upload", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("upload", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSel := strings.TrimSpace(*parseFlags.selector)
	if parseSel == "" {
		return writeAgenticEnvelope("upload", false, nil, nil, fmt.Errorf("upload: -selector is required"))
	}
	if len(parseFiles) == 0 {
		return writeAgenticEnvelope("upload", false, nil, nil, fmt.Errorf("upload: at least one -file is required"))
	}
	if parseErr := parsePage.page.Locator(parseSel).SetInputFiles([]string(parseFiles)); parseErr != nil {
		return writeAgenticEnvelope("upload", false, nil, nil, fmt.Errorf("upload to %s: %w", parseSel, parseErr))
	}
	return writeAgenticEnvelope("upload", true, inputResult{Action: "upload", Selector: parseSel, Text: strings.Join(parseFiles, ","), Attached: parsePage.attached}, nil, nil)
}

// runDragCommand drags one selector onto another.
var runDragCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("drag", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint")
	parseFrom := parseFs.String("from", "", "source CSS selector (required)")
	parseTo := parseFs.String("to", "", "target CSS selector (required)")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseFrom) == "" || strings.TrimSpace(*parseTo) == "" {
		return writeAgenticEnvelope("drag", false, nil, nil, fmt.Errorf("drag: -from and -to are required"))
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("drag", false, nil, nil, fmt.Errorf("drag: -url or -cdp is required"))
	}
	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, 1280, 800)
	if parseErr != nil {
		return writeAgenticEnvelope("drag", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	if parseErr := parsePage.page.DragAndDrop(*parseFrom, *parseTo); parseErr != nil {
		return writeAgenticEnvelope("drag", false, nil, nil, fmt.Errorf("drag %s -> %s: %w", *parseFrom, *parseTo, parseErr))
	}
	return writeAgenticEnvelope("drag", true, inputResult{Action: "drag", Selector: *parseFrom + " -> " + *parseTo, Attached: parsePage.attached}, nil, nil)
}

// selectOptionByValueOrLabel builds SelectOptionValues from a value or label.
func selectOptionByValueOrLabel(parseValue string, parseLabel string) playwright.SelectOptionValues {
	if strings.TrimSpace(parseValue) != "" {
		parseV := []string{parseValue}
		return playwright.SelectOptionValues{Values: &parseV}
	}
	parseL := []string{parseLabel}
	return playwright.SelectOptionValues{Labels: &parseL}
}

// multiFlag collects a repeatable string flag.
type multiFlag []string

func (parseM *multiFlag) String() string { return strings.Join(*parseM, ",") }
func (parseM *multiFlag) Set(parseV string) error {
	*parseM = append(*parseM, parseV)
	return nil
}
