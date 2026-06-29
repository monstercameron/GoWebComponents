//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

// traceResult is the JSON payload for `gwc trace`.
type traceResult struct {
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	DurationMs int64  `json:"durationMs"`
	Reloaded   bool   `json:"reloaded"`
	Attached   bool   `json:"attached"`
}

// a11yNode is one role+name entry from the accessibility tree.
type a11yNode struct {
	Role string `json:"role"`
	Name string `json:"name"`
}

// a11yResult is the JSON payload for `gwc a11y`.
type a11yResult struct {
	Count    int        `json:"count"`
	Nodes    []a11yNode `json:"nodes"`
	Attached bool       `json:"attached"`
}

// runTraceCommand records a replayable Playwright trace.zip (screenshots + DOM
// snapshots + network) over a window — for reproducing failures.
var runTraceCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("trace", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch and trace (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint")
	parseOut := parseFs.String("out", "bin/trace.zip", "output trace.zip path")
	parseDuration := parseFs.Duration("duration", 3*time.Second, "how long to record")
	parseReload := parseFs.Bool("reload", false, "reload after starting to capture from a clean load")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("trace", false, nil, nil, fmt.Errorf("trace: -url or -cdp is required"))
	}
	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, 1280, 800)
	if parseErr != nil {
		return writeAgenticEnvelope("trace", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseTracing := parsePage.page.Context().Tracing()
	if parseErr := parseTracing.Start(playwright.TracingStartOptions{
		Screenshots: playwright.Bool(true),
		Snapshots:   playwright.Bool(true),
		Sources:     playwright.Bool(true),
	}); parseErr != nil {
		return writeAgenticEnvelope("trace", false, nil, nil, fmt.Errorf("start tracing: %w", parseErr))
	}
	parseReloaded := false
	if *parseReload {
		if _, parseErr := parsePage.page.Reload(); parseErr == nil {
			parseReloaded = true
		}
	}
	time.Sleep(*parseDuration)

	if parseDir := filepath.Dir(*parseOut); parseDir != "" {
		_ = os.MkdirAll(parseDir, 0o755)
	}
	if parseErr := parseTracing.Stop(*parseOut); parseErr != nil {
		return writeAgenticEnvelope("trace", false, nil, nil, fmt.Errorf("stop tracing: %w", parseErr))
	}
	var parseBytes int64
	if parseInfo, parseErr := os.Stat(*parseOut); parseErr == nil {
		parseBytes = parseInfo.Size()
	}
	parseResult := traceResult{
		Path:       *parseOut,
		Bytes:      parseBytes,
		DurationMs: parseDuration.Milliseconds(),
		Reloaded:   parseReloaded,
		Attached:   parsePage.attached,
	}
	return writeAgenticEnvelope("trace", true, parseResult, nil, nil)
}

// axStringField reads node[field]["value"] as a string from the raw AX tree.
func axStringField(parseNode map[string]any, parseField string) string {
	parseObj, parseOk := parseNode[parseField].(map[string]any)
	if !parseOk {
		return ""
	}
	if parseVal, parseOk := parseObj["value"].(string); parseOk {
		return parseVal
	}
	return ""
}

// runA11yCommand dumps the accessibility tree (roles + accessible names) via a
// raw CDP Accessibility.getFullAXTree session — how an agent reasons about UI
// semantics, which dom (one element's HTML) cannot express.
var runA11yCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("a11y", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch and read (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("a11y", false, nil, nil, fmt.Errorf("a11y: -url or -cdp is required"))
	}
	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, 1280, 800)
	if parseErr != nil {
		return writeAgenticEnvelope("a11y", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	parseSession, parseErr := parsePage.page.Context().NewCDPSession(parsePage.page)
	if parseErr != nil {
		return writeAgenticEnvelope("a11y", false, nil, nil, fmt.Errorf("cdp session: %w", parseErr))
	}
	if _, parseErr := parseSession.Send("Accessibility.enable", map[string]any{}); parseErr != nil {
		return writeAgenticEnvelope("a11y", false, nil, nil, fmt.Errorf("accessibility.enable: %w", parseErr))
	}
	parseRaw, parseErr := parseSession.Send("Accessibility.getFullAXTree", map[string]any{})
	if parseErr != nil {
		return writeAgenticEnvelope("a11y", false, nil, nil, fmt.Errorf("getFullAXTree: %w", parseErr))
	}
	parseMap, parseOk := parseRaw.(map[string]any)
	if !parseOk {
		return writeAgenticEnvelope("a11y", false, nil, nil, fmt.Errorf("unexpected AX tree shape"))
	}
	parseRawNodes, _ := parseMap["nodes"].([]any)
	parseNodes := make([]a11yNode, 0, len(parseRawNodes))
	for _, parseN := range parseRawNodes {
		parseNode, parseOk := parseN.(map[string]any)
		if !parseOk {
			continue
		}
		if parseIgnored, parseOk := parseNode["ignored"].(bool); parseOk && parseIgnored {
			continue
		}
		parseRole := axStringField(parseNode, "role")
		parseName := axStringField(parseNode, "name")
		if parseRole == "" && parseName == "" {
			continue
		}
		parseNodes = append(parseNodes, a11yNode{Role: parseRole, Name: parseName})
	}
	parseResult := a11yResult{Count: len(parseNodes), Nodes: parseNodes, Attached: parsePage.attached}
	return writeAgenticEnvelope("a11y", true, parseResult, nil, nil)
}
