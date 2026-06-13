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
