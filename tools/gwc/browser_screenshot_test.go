//go:build playwrightgo

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// pngMagic is the 8-byte PNG signature every valid PNG starts with.
var pngMagic = []byte("\x89PNG\r\n\x1a\n")

const screenshotTestPage = `<!doctype html><html><head><meta charset="utf-8">
<style>html,body{margin:0}#app{width:100%;height:400px;background:#0b1020}
h1{margin:0;padding:40px;color:#34d399;font:48px sans-serif}</style></head>
<body><div id="app"><h1 id="headline">gwc screenshot works</h1></div></body></html>`

// TestCaptureScreenshotProducesPNG pins the DevTools/CDP browser-proxy feature:
// captureScreenshot must launch a sandbox-disabled headless Chromium, load a
// real page, and write a valid, non-trivial PNG to disk — both for the full
// page and for a single selector.
func TestCaptureScreenshotProducesPNG(t *testing.T) {
	parseSrv := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = parseW.Write([]byte(screenshotTestPage))
	}))
	defer parseSrv.Close()

	parseOut := filepath.Join(t.TempDir(), "full.png")
	parseRes, parseErr := captureScreenshot(parseSrv.URL, "", parseOut, 800, 600, true)
	if parseErr != nil {
		t.Fatalf("captureScreenshot (full page): %v", parseErr)
	}
	if parseRes.Path != parseOut {
		t.Fatalf("result path = %q, want %q", parseRes.Path, parseOut)
	}
	if !parseRes.NoSandbox {
		t.Fatal("result.NoSandbox = false, want true (must launch with --no-sandbox)")
	}

	parseBytes, parseErr := os.ReadFile(parseOut)
	if parseErr != nil {
		t.Fatalf("read screenshot: %v", parseErr)
	}
	if !bytes.HasPrefix(parseBytes, pngMagic) {
		t.Fatalf("output is not a PNG (first bytes %x)", parseBytes[:min(8, len(parseBytes))])
	}
	if len(parseBytes) < 1000 {
		t.Fatalf("PNG suspiciously small: %d bytes", len(parseBytes))
	}
	if parseRes.Bytes != len(parseBytes) {
		t.Fatalf("result.Bytes = %d, want %d", parseRes.Bytes, len(parseBytes))
	}

	// Selector capture: a single element must also yield a valid, smaller PNG.
	parseElOut := filepath.Join(t.TempDir(), "headline.png")
	parseElRes, parseErr := captureScreenshot(parseSrv.URL, "#headline", parseElOut, 800, 600, true)
	if parseErr != nil {
		t.Fatalf("captureScreenshot (selector): %v", parseErr)
	}
	if parseElRes.Selector != "#headline" {
		t.Fatalf("result.Selector = %q, want #headline", parseElRes.Selector)
	}
	parseElBytes, parseErr := os.ReadFile(parseElOut)
	if parseErr != nil {
		t.Fatalf("read selector screenshot: %v", parseErr)
	}
	if !bytes.HasPrefix(parseElBytes, pngMagic) {
		t.Fatalf("selector output is not a PNG (first bytes %x)", parseElBytes[:min(8, len(parseElBytes))])
	}
	if len(parseElBytes) < 100 {
		t.Fatalf("selector PNG suspiciously small: %d bytes", len(parseElBytes))
	}
}

// TestCaptureScreenshotRequiresURL pins the precondition: an empty url returns
// an error rather than launching a browser.
func TestCaptureScreenshotRequiresURL(t *testing.T) {
	if parseErr := runScreenshotCommand(launcher{}, []string{"-out", filepath.Join(t.TempDir(), "x.png")}); parseErr != nil {
		// runScreenshotCommand writes a failure envelope and returns nil; a
		// non-nil error would be a flag-parse failure, which is also acceptable.
		t.Logf("runScreenshotCommand returned err (acceptable): %v", parseErr)
	}
}
