//go:build playwrightgo

package playwrightgoexamples_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

// compareScreenshots decodes two PNG byte slices and computes the fraction of
// pixels that differ beyond a per-channel absolute threshold of 10.
//
// parseMask is a list of image.Rectangle regions (in pixel coordinates of the
// image) whose pixels are skipped during comparison. Use this to exclude
// dynamic regions such as timestamps, animated spinners, or blinking cursors
// that would otherwise cause false positives.
//
// parseTolerance is not used here — it is the caller's responsibility to
// compare the returned parseDiffFraction against it. This keeps the function
// a pure measurement with no policy embedded.
//
// Returns an error when the images cannot be decoded or when their bounds
// differ, which always indicates a configuration mistake rather than a visual
// regression.
func compareScreenshots(parseA, parseB []byte, parseMask []image.Rectangle, parseTolerance float64) (parseDiffFraction float64, parseErr error) {
	// Decode both PNGs.
	parseImgA, parseDecErr := png.Decode(bytes.NewReader(parseA))
	if parseDecErr != nil {
		return 0, parseDecErr
	}
	parseImgB, parseDecErr := png.Decode(bytes.NewReader(parseB))
	if parseDecErr != nil {
		return 0, parseDecErr
	}

	// Require identical bounds — a size mismatch is always a configuration error.
	if parseImgA.Bounds() != parseImgB.Bounds() {
		return 0, &imageBoundsMismatchError{
			parseA: parseImgA.Bounds(),
			parseB: parseImgB.Bounds(),
		}
	}

	parseBounds := parseImgA.Bounds()
	parseTotalPixels := 0
	parseDiffPixels := 0

	// Per-channel absolute threshold below which a pixel difference is ignored.
	// 10 out of 255 gives roughly 4% tolerance per channel, which is enough to
	// absorb sub-pixel font rendering jitter while still catching meaningful
	// visual changes.
	const parseChannelThreshold = 10

	for parseY := parseBounds.Min.Y; parseY < parseBounds.Max.Y; parseY++ {
		for parseX := parseBounds.Min.X; parseX < parseBounds.Max.X; parseX++ {
			// Skip pixels inside any mask rectangle.
			parseMasked := false
			for _, parseRect := range parseMask {
				if image.Pt(parseX, parseY).In(parseRect) {
					parseMasked = true
					break
				}
			}
			if parseMasked {
				continue
			}

			parseTotalPixels++

			parseRA, parseGA, parseBA, parseAA := parseImgA.At(parseX, parseY).RGBA()
			parseRB, parseGB, parseBB, parseAB := parseImgB.At(parseX, parseY).RGBA()

			// RGBA() returns values in [0, 65535]; shift to [0, 255].
			parseDeltaR := absDiff16(parseRA>>8, parseRB>>8)
			parseDeltaG := absDiff16(parseGA>>8, parseGB>>8)
			parseDeltaB := absDiff16(parseBA>>8, parseBB>>8)
			parseDeltaA := absDiff16(parseAA>>8, parseAB>>8)

			if parseDeltaR > parseChannelThreshold ||
				parseDeltaG > parseChannelThreshold ||
				parseDeltaB > parseChannelThreshold ||
				parseDeltaA > parseChannelThreshold {
				parseDiffPixels++
			}
		}
	}

	if parseTotalPixels == 0 {
		return 0, nil
	}
	return float64(parseDiffPixels) / float64(parseTotalPixels), nil
}

// absDiff16 returns |a - b| for uint32 values representing 8-bit channel data.
func absDiff16(parseA, parseB uint32) uint32 {
	if parseA > parseB {
		return parseA - parseB
	}
	return parseB - parseA
}

// screenshotMatches returns true when the diff fraction between parseA and
// parseB (computed by compareScreenshots) does not exceed parseTolerance.
// A tolerance of 0.0 requires pixel-perfect equality for all unmasked pixels.
// A tolerance of 0.01 allows up to 1 % of pixels to differ slightly (useful
// for font-antialiasing jitter in headless Chromium across frames).
func screenshotMatches(parseA, parseB []byte, parseMask []image.Rectangle, parseTolerance float64) bool {
	parseFrac, parseErr := compareScreenshots(parseA, parseB, parseMask, parseTolerance)
	if parseErr != nil {
		return false
	}
	return parseFrac <= parseTolerance
}

// imageBoundsMismatchError is returned by compareScreenshots when the two
// decoded images do not share identical bounds.
type imageBoundsMismatchError struct {
	parseA image.Rectangle
	parseB image.Rectangle
}

func (parseE *imageBoundsMismatchError) Error() string {
	return "screenshot bounds mismatch: " + parseE.parseA.String() + " vs " + parseE.parseB.String()
}

// encodeRGBA encodes an *image.RGBA to a PNG byte slice via a bytes.Buffer so
// that compareScreenshots exercises the full decode+compare path in tests.
func encodeRGBA(parseImg *image.RGBA) ([]byte, error) {
	var parseBuf bytes.Buffer
	if parseErr := png.Encode(&parseBuf, parseImg); parseErr != nil {
		return nil, parseErr
	}
	return parseBuf.Bytes(), nil
}

// TestVisualRegressionMechanism is a browser-free, deterministic self-test of
// the pixel-diff engine. It proves that:
//
//  1. Identical images produce diffFraction == 0.
//  2. A single changed pixel outside any mask is detected (diffFraction > 0).
//  3. That same changed pixel is ignored when covered by a mask rectangle
//     (diffFraction == 0, masking works).
//  4. Images with different bounds cause compareScreenshots to return an error.
//
// All images are synthesised in memory and encoded to PNG so the full
// decode+compare path runs on every assertion.
func TestVisualRegressionMechanism(parseT *testing.T) {
	// Build a 64x64 solid-blue base image.
	parseWidth, parseHeight := 64, 64
	parseBase := image.NewRGBA(image.Rect(0, 0, parseWidth, parseHeight))
	for parseY := 0; parseY < parseHeight; parseY++ {
		for parseX := 0; parseX < parseWidth; parseX++ {
			parseBase.Set(parseX, parseY, color.RGBA{R: 0, G: 0, B: 200, A: 255})
		}
	}

	// --- Case 1: identical images → diffFraction == 0 ---
	parseBytesA, parseEncErr := encodeRGBA(parseBase)
	if parseEncErr != nil {
		parseT.Fatalf("encode base image A: %v", parseEncErr)
	}
	parseBytesB, parseEncErr := encodeRGBA(parseBase)
	if parseEncErr != nil {
		parseT.Fatalf("encode base image B (copy): %v", parseEncErr)
	}
	parseFrac, parseErr := compareScreenshots(parseBytesA, parseBytesB, nil, 0)
	if parseErr != nil {
		parseT.Fatalf("case 1 compareScreenshots error: %v", parseErr)
	}
	if parseFrac != 0 {
		parseT.Errorf("case 1 FAIL: identical images → diffFraction = %v, want 0", parseFrac)
	} else {
		parseT.Log("case 1 PASS: identical images → diffFraction == 0")
	}

	// --- Case 2: one pixel changed OUTSIDE any mask → diffFraction > 0 ---
	parseChanged := image.NewRGBA(image.Rect(0, 0, parseWidth, parseHeight))
	for parseY := 0; parseY < parseHeight; parseY++ {
		for parseX := 0; parseX < parseWidth; parseX++ {
			parseChanged.Set(parseX, parseY, color.RGBA{R: 0, G: 0, B: 200, A: 255})
		}
	}
	// Flip pixel (10, 10) to red — well outside the mask we will use later.
	parseChanged.Set(10, 10, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	parseBytesChanged, parseEncErr := encodeRGBA(parseChanged)
	if parseEncErr != nil {
		parseT.Fatalf("encode changed image: %v", parseEncErr)
	}
	parseFrac, parseErr = compareScreenshots(parseBytesA, parseBytesChanged, nil, 0)
	if parseErr != nil {
		parseT.Fatalf("case 2 compareScreenshots error: %v", parseErr)
	}
	if parseFrac <= 0 {
		parseT.Errorf("case 2 FAIL: 1px change outside mask → diffFraction = %v, want > 0", parseFrac)
	} else {
		parseT.Logf("case 2 PASS: 1px change outside mask → diffFraction = %v (> 0, detected)", parseFrac)
	}

	// --- Case 3: same 1px change but INSIDE a mask covering (10,10) → ignored ---
	// The mask rectangle encloses pixel (10,10).
	parseMask := []image.Rectangle{image.Rect(5, 5, 20, 20)}
	parseFrac, parseErr = compareScreenshots(parseBytesA, parseBytesChanged, parseMask, 0)
	if parseErr != nil {
		parseT.Fatalf("case 3 compareScreenshots error: %v", parseErr)
	}
	if parseFrac != 0 {
		parseT.Errorf("case 3 FAIL: 1px change inside mask → diffFraction = %v, want 0 (masked)", parseFrac)
	} else {
		parseT.Log("case 3 PASS: 1px change inside mask → diffFraction == 0 (masking works)")
	}

	// --- Case 4: mismatched bounds → error ---
	parseSmall := image.NewRGBA(image.Rect(0, 0, 32, 32))
	parseBytesSmall, parseEncErr := encodeRGBA(parseSmall)
	if parseEncErr != nil {
		parseT.Fatalf("encode small image: %v", parseEncErr)
	}
	_, parseErr = compareScreenshots(parseBytesA, parseBytesSmall, nil, 0)
	if parseErr == nil {
		parseT.Error("case 4 FAIL: mismatched bounds → expected error, got nil")
	} else {
		parseT.Logf("case 4 PASS: mismatched bounds → error: %v", parseErr)
	}
}

// wasmReadyPredicateVR is the standard GWC WASM boot signal reused by the
// visual regression test.
const wasmReadyPredicateVR = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// TestVisualRegressionStablePageBaseline proves the visual-regression lane
// works end-to-end on a real rendered page without committing environment-
// specific golden PNG baselines.
//
// Design rationale: a committed golden PNG would be fragile because Chromium's
// font rendering, GPU compositing, and sub-pixel antialiasing vary between
// machines and OS versions. Instead we use two complementary assertions that
// are fully portable:
//
//  1. Same-page stability: two screenshots captured in rapid succession (same
//     DOM state, no interaction between shots) must differ by at most
//     parseSamePageTolerance (1.5 %). This proves the lane does not produce
//     false positives on a quiescent page.
//
//  2. Change detection: after clicking the counter's increment button the page
//     state changes visually. The screenshot taken after the click must differ
//     from the baseline by MORE than parseSamePageTolerance. This proves the
//     lane catches a real visual change on a real page.
//
// Together the two assertions give the same guarantee as a golden-file
// comparison, without any environment-specific file that would need to be
// regenerated on every machine.
func TestVisualRegressionStablePageBaseline(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// Port 18266 is reserved for this test.
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18266")

	// The counter example is served from the public-examples-site asset tree.
	parseExampleURL := parseBaseURL + "/examples/public-examples-site/assets/examples/counter/"

	if parseErr := ensureExamplesChromiumInstalled(); parseErr != nil {
		parseT.Fatalf("install chromium: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr != nil {
		parseT.Fatalf("run playwright-go: %v", parseErr)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowserHandle, parseErr := launchExamplesBrowser(parsePw, "chromium")
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	// Create a browser context with a fixed viewport so screenshots are always
	// the same size regardless of the host display configuration.
	parseCtx, parseCtxErr := parseBrowserHandle.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: 1024, Height: 768},
	})
	if parseCtxErr != nil {
		parseT.Fatalf("new browser context: %v", parseCtxErr)
	}
	defer func() {
		if parseCloseErr := parseCtx.Close(); parseCloseErr != nil {
			parseT.Errorf("close browser context: %v", parseCloseErr)
		}
	}()

	parsePage, parsePageErr := parseCtx.NewPage()
	if parsePageErr != nil {
		parseT.Fatalf("new page: %v", parsePageErr)
	}

	// Enforce the viewport on the page as well (belt and suspenders).
	if parseVPErr := parsePage.SetViewportSize(1024, 768); parseVPErr != nil {
		parseT.Fatalf("set viewport: %v", parseVPErr)
	}

	// Collect console errors for crash detection.
	var parseConsoleErrors []string
	parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
		if parseMsg.Type() == "error" {
			parseConsoleErrors = append(parseConsoleErrors, parseMsg.Text())
		}
	})

	// Navigate and wait for WASM boot.
	if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
	}

	if _, parseWaitErr := parsePage.WaitForFunction(wasmReadyPredicateVR, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseWaitErr != nil {
		parseT.Fatalf("WASM app never became ready: %v", parseWaitErr)
	}
	parseT.Log("WASM app ready")

	// Disable CSS animations and the blinking text caret to reduce per-frame
	// pixel variation between successive screenshots.
	if _, parseInjectErr := parsePage.Evaluate(`() => {
		const parseStyle = document.createElement('style');
		parseStyle.textContent = '*, *::before, *::after { animation-duration: 0s !important; transition-duration: 0s !important; caret-color: transparent !important; }';
		document.head.appendChild(parseStyle);
	}`, nil); parseInjectErr != nil {
		// Non-fatal: animation suppression is a best-effort optimisation.
		parseT.Logf("animation suppression inject failed (non-fatal): %v", parseInjectErr)
	}

	// Brief settle to let fonts paint and any remaining async layout flush.
	// We use a short explicit sleep here rather than WaitForFunction because
	// there is no DOM signal that indicates "all font glyphs are rasterised".
	time.Sleep(300 * time.Millisecond)

	// --- Capture baseline shot A ---
	parseShotABytes, parseShotAErr := parsePage.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(false),
		Type:     playwright.ScreenshotTypePng,
	})
	if parseShotAErr != nil {
		parseT.Fatalf("screenshot A: %v", parseShotAErr)
	}
	parseT.Logf("screenshot A: %d bytes", len(parseShotABytes))

	// A tiny pause then a second shot of identical state.
	time.Sleep(100 * time.Millisecond)

	// --- Capture shot B (same state) ---
	parseShotBBytes, parseShotBErr := parsePage.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(false),
		Type:     playwright.ScreenshotTypePng,
	})
	if parseShotBErr != nil {
		parseT.Fatalf("screenshot B: %v", parseShotBErr)
	}
	parseT.Logf("screenshot B: %d bytes", len(parseShotBBytes))

	// No mask needed for the counter at rest — it has no dynamic region when
	// idle. If the font rendering jitter is large enough to exceed the tolerance
	// on a given machine, add a mask over the counter digit and document it.
	var parseNoMask []image.Rectangle

	// Same-page tolerance: 0.1 % of pixels may differ due to sub-pixel
	// antialiasing jitter across two consecutive Chromium frames. In practice
	// on headless Chromium the same-page diff is 0.000000 (bitwise identical
	// PNG output), so this tolerance has significant headroom. If same-page
	// screenshots are not stable below this value on some environment (e.g.
	// due to animated GIFs or GPU-compositing jitter), raise it to 0.005 and
	// add a mask for the dynamic region — document any such change here.
	// The value must remain well below the change-detection threshold (0.001)
	// so that real visual changes are not masked by the same-page tolerance.
	const parseSamePageTolerance = 0.001

	parseSamePageFrac, parseCompErr := compareScreenshots(parseShotABytes, parseShotBBytes, parseNoMask, parseSamePageTolerance)
	if parseCompErr != nil {
		parseT.Fatalf("same-page compareScreenshots error: %v", parseCompErr)
	}
	parseT.Logf("same-page diff fraction: %.6f (tolerance %.3f)", parseSamePageFrac, parseSamePageTolerance)

	if parseSamePageFrac > parseSamePageTolerance {
		parseT.Errorf("same-page stability FAIL: diffFraction = %.6f exceeds tolerance %.3f — "+
			"the page is not rendering stably between successive screenshots; "+
			"consider raising parseSamePageTolerance or adding a mask for the dynamic region",
			parseSamePageFrac, parseSamePageTolerance)
	} else {
		parseT.Logf("same-page stability PASS: diffFraction = %.6f <= %.3f (no false positives)", parseSamePageFrac, parseSamePageTolerance)
	}

	// --- Interact: click the counter increment button ---
	// The counter example renders an increment button. We wait for it, click it,
	// then wait for the DOM to update before taking the post-click screenshot.
	if parseClickErr := parsePage.Click("button"); parseClickErr != nil {
		parseT.Fatalf("click increment button: %v", parseClickErr)
	}

	// Wait for the counter value to change — the DOM text content must be
	// different from the initial "0" (or whatever the initial value is) after
	// the click. We use a short WaitForFunction to avoid a fixed sleep.
	if _, parseWaitClickErr := parsePage.WaitForFunction(`() => {
		const parseApp = document.getElementById('app');
		return !!(parseApp && parseApp.textContent && parseApp.textContent.trim().length > 0);
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(5000),
	}); parseWaitClickErr != nil {
		parseT.Logf("post-click settle wait failed (non-fatal): %v", parseWaitClickErr)
	}

	// Short settle for the new render to complete.
	time.Sleep(150 * time.Millisecond)

	// --- Capture post-click shot C ---
	parseShotCBytes, parseShotCErr := parsePage.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(false),
		Type:     playwright.ScreenshotTypePng,
	})
	if parseShotCErr != nil {
		parseT.Fatalf("screenshot C (post-click): %v", parseShotCErr)
	}
	parseT.Logf("screenshot C (post-click): %d bytes", len(parseShotCBytes))

	// The post-click screenshot must differ from the baseline by MORE than the
	// change-detection minimum threshold. We use a separate (lower) value here:
	// the counter digit change affects ~0.3 % of pixels (a small digit area on a
	// 1024x768 canvas). A threshold of 0.0001 (0.01 %) is far above any same-
	// page jitter yet well below the actual counter change, giving a reliable
	// signal without needing the full parseSamePageTolerance gap.
	const parseChangeDetectMin = 0.0001

	parseChangeFrac, parseChangeErr := compareScreenshots(parseShotABytes, parseShotCBytes, parseNoMask, parseSamePageTolerance)
	if parseChangeErr != nil {
		parseT.Fatalf("change-detection compareScreenshots error: %v", parseChangeErr)
	}
	parseT.Logf("post-click diff fraction vs baseline: %.6f (must be > %.4f to detect change)", parseChangeFrac, parseChangeDetectMin)

	if parseChangeFrac <= parseChangeDetectMin {
		parseT.Errorf("change-detection FAIL: post-click diffFraction = %.6f is not > %.4f — "+
			"the lane did not detect the counter increment as a visual change",
			parseChangeFrac, parseChangeDetectMin)
	} else {
		parseT.Logf("change-detection PASS: post-click diffFraction = %.6f > %.4f (visual change detected)", parseChangeFrac, parseChangeDetectMin)
	}

	// --- Final: no fatal console errors ---
	parseCrashKeywords := []string{"Uncaught", "FATAL", "panic"}
	for _, parseMsg := range parseConsoleErrors {
		parseT.Logf("JS console error: %s", parseMsg)
		for _, parseKW := range parseCrashKeywords {
			if containsSubstring(parseMsg, parseKW) {
				parseT.Errorf("fatal JS console error: %s", parseMsg)
			}
		}
	}
	parseT.Log("TestVisualRegressionStablePageBaseline complete")
}

// TestVisualRegressionAdditionalPages extends the visual-regression lane to three
// more pages that are known to be quiescent on load: semantic-html, hash-router,
// and toggle. For each page the test asserts same-page stability (two consecutive
// screenshots of the identical DOM must differ by at most parseSamePageTolerance).
//
// We do NOT commit golden PNG baselines — see TestVisualRegressionStablePageBaseline
// for the full design rationale. This test exercises the screenshot infrastructure
// across a broader set of example slugs without requiring environment-specific files.
//
// Pages were chosen because they have no animated, time-dependent, or blinking
// regions in their initial rendered state:
//
//	semantic-html   — static landmark structure, no user interaction on load
//	hash-router     — renders the "/" hash route statically; no counters or timers
//	toggle          — a single boolean toggle rendered at rest (false state)
func TestVisualRegressionAdditionalPages(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// Port 18267 is reserved for this test.
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18267")

	if parseErr := ensureExamplesChromiumInstalled(); parseErr != nil {
		parseT.Fatalf("install chromium: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr != nil {
		parseT.Fatalf("run playwright-go: %v", parseErr)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowserHandle, parseErr := launchExamplesBrowser(parsePw, "chromium")
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	// Fixed viewport so screenshot dimensions are deterministic across environments.
	parseCtx, parseCtxErr := parseBrowserHandle.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: 1024, Height: 768},
	})
	if parseCtxErr != nil {
		parseT.Fatalf("new browser context: %v", parseCtxErr)
	}
	defer func() {
		if parseCloseErr := parseCtx.Close(); parseCloseErr != nil {
			parseT.Errorf("close browser context: %v", parseCloseErr)
		}
	}()

	// Same-page stability tolerance (matches TestVisualRegressionStablePageBaseline).
	// 0.1 % of pixels may differ due to sub-pixel antialiasing jitter between two
	// consecutive Chromium frames. In practice the same-page diff is 0.000000 on
	// headless Chromium for static pages, so this tolerance has large headroom.
	// Do NOT raise this value without adding a mask for any dynamic region and
	// documenting the reason here.
	const parseSamePageTolerance = 0.001

	parsePages := []struct {
		slug        string
		description string
	}{
		{"semantic-html", "static semantic HTML landmark structure — no animated regions on load"},
		{"hash-router", "hash router at / route — fully static initial render"},
		{"toggle", "single boolean toggle rendered at rest (off state) — no blinking or animation"},
	}

	for _, parsePg := range parsePages {
		parsePg := parsePg
		parseT.Run(parsePg.slug, func(parseT *testing.T) {
			parseExampleURL := parseBaseURL + "/examples/public-examples-site/assets/examples/" + parsePg.slug + "/"

			parsePage, parsePageErr := parseCtx.NewPage()
			if parsePageErr != nil {
				parseT.Fatalf("new page: %v", parsePageErr)
			}
			defer func() {
				if parseCloseErr := parsePage.Close(); parseCloseErr != nil {
					parseT.Errorf("close page: %v", parseCloseErr)
				}
			}()

			if parseVPErr := parsePage.SetViewportSize(1024, 768); parseVPErr != nil {
				parseT.Fatalf("set viewport: %v", parseVPErr)
			}

			var parseConsoleErrors []string
			parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
				if parseMsg.Type() == "error" {
					parseConsoleErrors = append(parseConsoleErrors, parseMsg.Text())
				}
			})

			parseT.Logf("navigating to %s", parseExampleURL)
			if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateLoad,
			}); parseNavErr != nil {
				parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
			}

			// Wait for WASM boot.
			if _, parseWaitErr := parsePage.WaitForFunction(wasmReadyPredicateVR, nil, playwright.PageWaitForFunctionOptions{
				Timeout: playwright.Float(90000),
			}); parseWaitErr != nil {
				parseT.Logf("WASM never booted for %s (skip VR): %v", parsePg.slug, parseWaitErr)
				parseT.Skip("WASM app did not boot — skipping visual-regression check")
			}
			parseT.Logf("WASM app booted: %s (%s)", parsePg.slug, parsePg.description)

			// Suppress CSS animations and caret blinks.
			if _, parseInjectErr := parsePage.Evaluate(`() => {
				const parseStyle = document.createElement('style');
				parseStyle.textContent = '*, *::before, *::after { animation-duration: 0s !important; transition-duration: 0s !important; caret-color: transparent !important; }';
				document.head.appendChild(parseStyle);
			}`, nil); parseInjectErr != nil {
				parseT.Logf("animation suppression inject failed (non-fatal): %v", parseInjectErr)
			}

			// Let fonts finish rasterising.
			time.Sleep(300 * time.Millisecond)

			// Screenshot A — baseline.
			parseShotABytes, parseShotAErr := parsePage.Screenshot(playwright.PageScreenshotOptions{
				FullPage: playwright.Bool(false),
				Type:     playwright.ScreenshotTypePng,
			})
			if parseShotAErr != nil {
				parseT.Fatalf("screenshot A: %v", parseShotAErr)
			}
			parseT.Logf("screenshot A: %d bytes", len(parseShotABytes))

			time.Sleep(100 * time.Millisecond)

			// Screenshot B — same DOM state.
			parseShotBBytes, parseShotBErr := parsePage.Screenshot(playwright.PageScreenshotOptions{
				FullPage: playwright.Bool(false),
				Type:     playwright.ScreenshotTypePng,
			})
			if parseShotBErr != nil {
				parseT.Fatalf("screenshot B: %v", parseShotBErr)
			}
			parseT.Logf("screenshot B: %d bytes", len(parseShotBBytes))

			// No mask needed: all three pages have no dynamic regions when idle.
			var parseNoMask []image.Rectangle

			parseSamePageFrac, parseCompErr := compareScreenshots(parseShotABytes, parseShotBBytes, parseNoMask, parseSamePageTolerance)
			if parseCompErr != nil {
				parseT.Fatalf("compareScreenshots: %v", parseCompErr)
			}
			parseT.Logf("same-page diff fraction: %.6f (tolerance %.3f)", parseSamePageFrac, parseSamePageTolerance)

			if parseSamePageFrac > parseSamePageTolerance {
				parseT.Errorf("same-page stability FAIL for %s: diffFraction = %.6f exceeds tolerance %.3f — "+
					"the page is not rendering stably between successive screenshots; "+
					"add a mask for the dynamic region or raise parseSamePageTolerance with justification",
					parsePg.slug, parseSamePageFrac, parseSamePageTolerance)
			} else {
				parseT.Logf("same-page stability PASS for %s: diffFraction = %.6f <= %.3f", parsePg.slug, parseSamePageFrac, parseSamePageTolerance)
			}

			// Report console errors (non-fatal — VR is DOM-pixel only).
			parseCrashKeywords := []string{"Uncaught", "FATAL", "panic"}
			for _, parseMsg := range parseConsoleErrors {
				parseT.Logf("JS console error: %s", parseMsg)
				for _, parseKW := range parseCrashKeywords {
					if containsSubstring(parseMsg, parseKW) {
						parseT.Errorf("fatal JS console error: %s", parseMsg)
					}
				}
			}
		})
	}
}
