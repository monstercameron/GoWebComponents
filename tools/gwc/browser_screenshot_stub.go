//go:build !playwrightgo

package main

import "fmt"

// runScreenshotCommand is the default-build stub for `gwc screenshot`. The real
// implementation drives Chromium through playwright-go and is compiled only with
// the playwrightgo build tag, keeping the browser toolchain out of the lean
// default binary. Building with -tags playwrightgo enables the real capture.
var runScreenshotCommand = func(parseL launcher, parseArgs []string) error {
	return writeAgenticEnvelope("screenshot", false, nil, nil,
		fmt.Errorf("gwc screenshot requires building with -tags playwrightgo (it launches Chromium via playwright-go for CDP/DevTools screenshots): go run -tags playwrightgo ./tools/gwc screenshot -url <url> -out <file.png>"))
}

// runBrowserCommand is the default-build stub for `gwc browser`. The real
// implementation opens a headed (visible) Chromium window via playwright-go and
// is compiled only with the playwrightgo build tag.
var runBrowserCommand = func(parseL launcher, parseArgs []string) error {
	return writeAgenticEnvelope("browser", false, nil, nil,
		fmt.Errorf("gwc browser requires building with -tags playwrightgo (it opens a headed Chromium window the engineer can watch during copilot dev): go run -tags playwrightgo ./tools/gwc browser -url <url>"))
}

// playwrightProxyStub returns the standard "build with -tags playwrightgo" error
// for a browser-proxy verb that needs the Chromium toolchain.
func playwrightProxyStub(parseCommand string) error {
	return writeAgenticEnvelope(parseCommand, false, nil, nil,
		fmt.Errorf("gwc %s requires building with -tags playwrightgo (it drives Chromium via playwright-go over CDP): go run -tags playwrightgo ./tools/gwc %s ...", parseCommand, parseCommand))
}

// Default-build stubs for the CDP browser-proxy verbs (input, capture, read).
var runClickCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("click") }
var runTypeCommand = func(parseL launcher, parseArgs []string) error  { return playwrightProxyStub("type") }
var runPressCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("press") }
var runHoverCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("hover") }
var runScrollCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("scroll") }
var runConsoleCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("console") }
var runNetworkCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("network") }
var runDomCommand = func(parseL launcher, parseArgs []string) error  { return playwrightProxyStub("dom") }
var runEvalCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("eval") }
var runExpectCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("expect") }
var runWaitCommand = func(parseL launcher, parseArgs []string) error   { return playwrightProxyStub("wait") }
var runTraceCommand = func(parseL launcher, parseArgs []string) error  { return playwrightProxyStub("trace") }
var runA11yCommand = func(parseL launcher, parseArgs []string) error   { return playwrightProxyStub("a11y") }
var runSelectCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("select") }
var runUploadCommand = func(parseL launcher, parseArgs []string) error { return playwrightProxyStub("upload") }
var runDragCommand = func(parseL launcher, parseArgs []string) error   { return playwrightProxyStub("drag") }
var runMockCommand = func(parseL launcher, parseArgs []string) error   { return playwrightProxyStub("mock") }
