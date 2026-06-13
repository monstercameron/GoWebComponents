//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

// mockResult is the JSON payload for `gwc mock`.
type mockResult struct {
	Route       string `json:"route"`
	Mode        string `json:"mode"`
	Status      int    `json:"status,omitempty"`
	Intercepted int    `json:"intercepted"`
	Reloaded    bool   `json:"reloaded"`
	Attached    bool   `json:"attached"`
}

// runMockCommand intercepts requests matching a glob and either fulfills them
// with a stubbed status/body or aborts them — so an agent can exercise error
// paths. The read side is `gwc network`; this is the write side.
var runMockCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("mock", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to launch (unless -cdp is set)")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint")
	parseRoute := parseFs.String("route", "", "URL glob to intercept, e.g. **/api/** (required)")
	parseStatus := parseFs.Int("status", 0, "stub response status (use with -body); 0 = not set")
	parseBody := parseFs.String("body", "", "stub response body")
	parseContentType := parseFs.String("content-type", "text/plain", "stub response content type")
	parseAbort := parseFs.Bool("abort", false, "abort (fail) matching requests instead of stubbing")
	parseDuration := parseFs.Duration("duration", 3*time.Second, "how long to keep the route active")
	parseReload := parseFs.Bool("reload", true, "reload after installing the route to exercise it")
	parseFs.Bool("json", false, "emit the JSON envelope")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseRoute) == "" {
		return writeAgenticEnvelope("mock", false, nil, nil, fmt.Errorf("mock: -route is required"))
	}
	if !*parseAbort && *parseStatus == 0 {
		return writeAgenticEnvelope("mock", false, nil, nil, fmt.Errorf("mock: provide -status (+optional -body) to stub, or -abort to fail"))
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("mock", false, nil, nil, fmt.Errorf("mock: -url or -cdp is required"))
	}
	parsePage, parseErr := openProxyPage(*parseCDP, *parseURL, 1280, 800)
	if parseErr != nil {
		return writeAgenticEnvelope("mock", false, nil, nil, parseErr)
	}
	defer parsePage.close()

	var parseMu sync.Mutex
	parseCount := 0
	parseHandler := func(parseRouteObj playwright.Route) {
		parseMu.Lock()
		parseCount++
		parseMu.Unlock()
		if *parseAbort {
			_ = parseRouteObj.Abort()
			return
		}
		_ = parseRouteObj.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(*parseStatus),
			Body:        *parseBody,
			ContentType: playwright.String(*parseContentType),
		})
	}
	if parseErr := parsePage.page.Route(*parseRoute, parseHandler); parseErr != nil {
		return writeAgenticEnvelope("mock", false, nil, nil, fmt.Errorf("install route: %w", parseErr))
	}

	parseReloaded := false
	if *parseReload {
		if _, parseErr := parsePage.page.Reload(); parseErr == nil {
			parseReloaded = true
		}
	}
	time.Sleep(*parseDuration)
	_ = parsePage.page.Unroute(*parseRoute)

	parseMode := "fulfill"
	if *parseAbort {
		parseMode = "abort"
	}
	parseMu.Lock()
	parseFinal := parseCount
	parseMu.Unlock()
	parseResult := mockResult{
		Route:       *parseRoute,
		Mode:        parseMode,
		Status:      *parseStatus,
		Intercepted: parseFinal,
		Reloaded:    parseReloaded,
		Attached:    parsePage.attached,
	}
	return writeAgenticEnvelope("mock", true, parseResult, nil, nil)
}
