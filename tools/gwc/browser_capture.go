//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// captureCap bounds retained entries so a chatty page cannot exhaust memory;
// overflow is counted and reported, never silently dropped.
const captureCap = 500

// consoleEntry is one captured browser console message or uncaught error.
type consoleEntry struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// consoleResult is the JSON payload returned by `gwc console`.
type consoleResult struct {
	URL        string         `json:"url,omitempty"`
	Attached   bool           `json:"attached"`
	DurationMs int64          `json:"durationMs"`
	Reloaded   bool           `json:"reloaded"`
	Count      int            `json:"count"`
	Dropped    int            `json:"dropped"`
	Errors     int            `json:"errors"`
	Entries    []consoleEntry `json:"entries"`
}

// networkEntry is one captured request/response/failure.
type networkEntry struct {
	Method   string `json:"method"`
	URL      string `json:"url"`
	Type     string `json:"resourceType,omitempty"`
	Status   int    `json:"status,omitempty"`
	Failed   bool   `json:"failed"`
	Failure  string `json:"failure,omitempty"`
}

// networkResult is the JSON payload returned by `gwc network`.
type networkResult struct {
	URL        string         `json:"url,omitempty"`
	Attached   bool           `json:"attached"`
	DurationMs int64          `json:"durationMs"`
	Reloaded   bool           `json:"reloaded"`
	Count      int            `json:"count"`
	Dropped    int            `json:"dropped"`
	Failures   int            `json:"failures"`
	Entries    []networkEntry `json:"entries"`
}

// captureFlags are shared by console and network.
type captureFlags struct {
	fs       *flag.FlagSet
	url      *string
	cdp      *string
	duration *time.Duration
	reload   *bool
	width    *int
	height   *int
}

func newCaptureFlags(parseName string) *captureFlags {
	parseFs := flag.NewFlagSet(parseName, flag.ContinueOnError)
	parseC := &captureFlags{
		fs:       parseFs,
		url:      parseFs.String("url", "", "URL to launch and observe (unless -cdp is set)"),
		cdp:      parseFs.String("cdp", "", "attach to a running browser's CDP endpoint (the engineer's open gwc browser window)"),
		duration: parseFs.Duration("duration", 3*time.Second, "how long to capture events"),
		reload:   parseFs.Bool("reload", false, "reload the page after attaching to capture boot-time events"),
		width:    parseFs.Int("width", 1280, "viewport width (launch mode)"),
		height:   parseFs.Int("height", 800, "viewport height (launch mode)"),
	}
	parseFs.Bool("json", false, "emit the JSON envelope")
	return parseC
}

func (parseC *captureFlags) resolve(parseName string, parseArgs []string) (*proxyPage, error) {
	if parseErr := parseC.fs.Parse(parseArgs); parseErr != nil {
		return nil, parseErr
	}
	if strings.TrimSpace(*parseC.url) == "" && strings.TrimSpace(*parseC.cdp) == "" {
		return nil, fmt.Errorf("%s: -url or -cdp is required", parseName)
	}
	return openProxyPage(*parseC.cdp, *parseC.url, *parseC.width, *parseC.height)
}

// runConsoleCommand captures browser console messages + uncaught errors.
var runConsoleCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newCaptureFlags("console")
	parsePage, parseErr := parseFlags.resolve("console", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("console", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	var parseMu sync.Mutex
	parseEntries := []consoleEntry{}
	parseDropped := 0
	parseErrors := 0
	parseAdd := func(parseE consoleEntry) {
		parseMu.Lock()
		defer parseMu.Unlock()
		if parseE.Type == "error" {
			parseErrors++
		}
		if len(parseEntries) >= captureCap {
			parseDropped++
			return
		}
		parseEntries = append(parseEntries, parseE)
	}
	parsePage.page.OnConsole(func(parseMsg playwright.ConsoleMessage) {
		parseAdd(consoleEntry{Type: parseMsg.Type(), Text: parseMsg.Text()})
	})
	parsePage.page.OnPageError(func(parseErr error) {
		parseAdd(consoleEntry{Type: "error", Text: parseErr.Error()})
	})

	parseReloaded := false
	if *parseFlags.reload {
		if _, parseErr := parsePage.page.Reload(); parseErr == nil {
			parseReloaded = true
		}
	}
	time.Sleep(*parseFlags.duration)

	parseMu.Lock()
	defer parseMu.Unlock()
	parseResult := consoleResult{
		URL:        strings.TrimSpace(*parseFlags.url),
		Attached:   parsePage.attached,
		DurationMs: parseFlags.duration.Milliseconds(),
		Reloaded:   parseReloaded,
		Count:      len(parseEntries),
		Dropped:    parseDropped,
		Errors:     parseErrors,
		Entries:    parseEntries,
	}
	return writeAgenticEnvelope("console", true, parseResult, nil, nil)
}

// runNetworkCommand captures requests/responses/failures from the live browser.
var runNetworkCommand = func(parseL launcher, parseArgs []string) error {
	parseFlags := newCaptureFlags("network")
	parsePage, parseErr := parseFlags.resolve("network", parseArgs)
	if parseErr != nil {
		return writeAgenticEnvelope("network", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	var parseMu sync.Mutex
	parseByURL := map[string]*networkEntry{}
	parseOrder := []string{}
	parseDropped := 0
	parseUpsert := func(parseKey string, parseFn func(*networkEntry)) {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseEntry, parseOk := parseByURL[parseKey]
		if !parseOk {
			if len(parseOrder) >= captureCap {
				parseDropped++
				return
			}
			parseEntry = &networkEntry{}
			parseByURL[parseKey] = parseEntry
			parseOrder = append(parseOrder, parseKey)
		}
		parseFn(parseEntry)
	}
	parsePage.page.OnRequest(func(parseReq playwright.Request) {
		parseUpsert(parseReq.URL(), func(parseE *networkEntry) {
			parseE.Method = parseReq.Method()
			parseE.URL = parseReq.URL()
			parseE.Type = parseReq.ResourceType()
		})
	})
	parsePage.page.OnResponse(func(parseResp playwright.Response) {
		parseUpsert(parseResp.URL(), func(parseE *networkEntry) {
			parseE.Status = parseResp.Status()
			if !parseResp.Ok() {
				parseE.Failed = true
				if parseE.Failure == "" {
					parseE.Failure = fmt.Sprintf("HTTP %d", parseResp.Status())
				}
			}
		})
	})
	parsePage.page.OnRequestFailed(func(parseReq playwright.Request) {
		parseUpsert(parseReq.URL(), func(parseE *networkEntry) {
			parseE.Method = parseReq.Method()
			parseE.URL = parseReq.URL()
			parseE.Failed = true
			if parseFail := parseReq.Failure(); parseFail != nil {
				parseE.Failure = parseFail.Error()
			} else if parseE.Failure == "" {
				parseE.Failure = "request failed"
			}
		})
	})

	parseReloaded := false
	if *parseFlags.reload {
		if _, parseErr := parsePage.page.Reload(); parseErr == nil {
			parseReloaded = true
		}
	}
	time.Sleep(*parseFlags.duration)

	parseMu.Lock()
	defer parseMu.Unlock()
	parseEntries := make([]networkEntry, 0, len(parseOrder))
	parseFailures := 0
	for _, parseKey := range parseOrder {
		parseE := parseByURL[parseKey]
		if parseE.Failed {
			parseFailures++
		}
		parseEntries = append(parseEntries, *parseE)
	}
	parseResult := networkResult{
		URL:        strings.TrimSpace(*parseFlags.url),
		Attached:   parsePage.attached,
		DurationMs: parseFlags.duration.Milliseconds(),
		Reloaded:   parseReloaded,
		Count:      len(parseEntries),
		Dropped:    parseDropped,
		Failures:   parseFailures,
		Entries:    parseEntries,
	}
	return writeAgenticEnvelope("network", true, parseResult, nil, nil)
}
