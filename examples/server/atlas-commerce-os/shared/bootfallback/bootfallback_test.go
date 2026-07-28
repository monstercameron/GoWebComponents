package bootfallback

import (
	"strings"
	"testing"
)

func testOptions() Options {
	return Options{
		WASMURL:      "/assets/bin/atlas-commerce-os.wasm",
		LoaderURL:    "/assets/script/wasm_exec.js",
		BuildCommand: "GOOS=js GOARCH=wasm go build -o examples/static/bin/atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client",
	}
}

// TestMarkupWithBinaryPresentShipsEveryReasonHidden pins the contract the boot
// snippet depends on: with the module on disk, all three script-detectable
// reasons are in the DOM and none of them is visible.
func TestMarkupWithBinaryPresentShipsEveryReasonHidden(parseT *testing.T) {
	parseMarkup := Markup(testOptions(), true)

	if !strings.Contains(parseMarkup, `id="`+HostID+`" `+KindAttr+`="`+KindIdle+`"`) {
		parseT.Fatalf("expected idle host, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `style="`+hostStyle+`" hidden>`) {
		parseT.Fatalf("expected hidden host when the module is present, got %q", parseMarkup)
	}
	for _, parseKind := range []string{KindUnsupported, KindLoaderMissing, KindBootFailed} {
		parseOpen := `<div id="` + ReasonIDPrefix + parseKind + `"`
		parseIndex := strings.Index(parseMarkup, parseOpen)
		if parseIndex < 0 {
			parseT.Fatalf("expected reason block for %q, got %q", parseKind, parseMarkup)
		}
		parseTagEnd := strings.Index(parseMarkup[parseIndex:], ">")
		if parseTagEnd < 0 || !strings.Contains(parseMarkup[parseIndex:parseIndex+parseTagEnd], " hidden") {
			parseT.Fatalf("reason block %q must ship hidden, got %q", parseKind, parseMarkup[parseIndex:])
		}
	}
	if strings.Contains(parseMarkup, KindBinaryMissing) {
		parseT.Fatalf("missing-binary copy must not ship when the module is present, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `<pre id="`+DetailID+`"`) || !strings.Contains(parseMarkup, `hidden></pre>`) {
		parseT.Fatalf("expected an empty hidden detail slot, got %q", parseMarkup)
	}
}

// TestMarkupWithBinaryMissingIsVisibleWithoutScript is the zero-JavaScript path:
// the server knows the module is absent, so the explanation must already be on
// screen rather than waiting for a script to reveal it.
func TestMarkupWithBinaryMissingIsVisibleWithoutScript(parseT *testing.T) {
	parseOptions := testOptions()
	parseMarkup := Markup(parseOptions, false)

	parseHostOpen := `<div id="` + HostID + `" ` + KindAttr + `="` + KindBinaryMissing + `" role="alert" style="` + hostStyle + `">`
	if !strings.Contains(parseMarkup, parseHostOpen) {
		parseT.Fatalf("expected a visible missing-binary host, got %q", parseMarkup)
	}
	parseReasonOpen := `<div id="` + ReasonIDPrefix + KindBinaryMissing + `"`
	parseIndex := strings.Index(parseMarkup, parseReasonOpen)
	if parseIndex < 0 {
		parseT.Fatalf("expected missing-binary reason block, got %q", parseMarkup)
	}
	parseTagEnd := strings.Index(parseMarkup[parseIndex:], ">")
	if strings.Contains(parseMarkup[parseIndex:parseIndex+parseTagEnd], " hidden") {
		parseT.Fatal("missing-binary reason must be visible: no script will reveal it")
	}
	// The copy has to name the file it wanted and the command that makes it,
	// because "build the client" without the command is the vague message this
	// package exists to replace.
	for _, parseWant := range []string{parseOptions.WASMURL, parseOptions.BuildCommand} {
		if !strings.Contains(parseMarkup, parseWant) {
			parseT.Fatalf("expected missing-binary copy to name %q, got %q", parseWant, parseMarkup)
		}
	}
	// It must not claim the shell is browsable: the server renders an empty
	// #app, so any copy implying readable content is a lie.
	if !strings.Contains(parseMarkup, "Nothing here is browsable yet.") {
		parseT.Fatalf("expected the copy to state the shell holds no content, got %q", parseMarkup)
	}
}

// TestNoScriptShipsOnBothPaths guards the one fallback that needs no cooperation
// from any runtime.
func TestNoScriptShipsOnBothPaths(parseT *testing.T) {
	for _, isPresent := range []bool{true, false} {
		parseMarkup := Markup(testOptions(), isPresent)
		if !strings.Contains(parseMarkup, `<noscript><div `+NoScriptAttr+`="true"`) {
			parseT.Fatalf("binaryPresent=%v: expected noscript payload, got %q", isPresent, parseMarkup)
		}
		if !strings.Contains(parseMarkup, "Atlas needs JavaScript to start") {
			parseT.Fatalf("binaryPresent=%v: expected noscript headline, got %q", isPresent, parseMarkup)
		}
		if !strings.Contains(parseMarkup, "this notice is the whole page") {
			parseT.Fatalf("binaryPresent=%v: noscript copy must not imply the page has content, got %q", isPresent, parseMarkup)
		}
		if !strings.HasPrefix(parseMarkup, "<noscript>") {
			parseT.Fatalf("binaryPresent=%v: noscript must precede the host, got %q", isPresent, parseMarkup)
		}
	}
}

// TestBootScriptReachesTheMarkupItReveals is the drift guard. Every id and
// attribute the snippet touches has to exist in the markup; a mismatch would
// fail silently at the exact moment the page is already broken.
func TestBootScriptReachesTheMarkupItReveals(parseT *testing.T) {
	parseOptions := testOptions()
	parseScript := BootScript(parseOptions)
	parseMarkup := Markup(parseOptions, true)

	for _, parseWant := range []string{
		"getElementById('" + HostID + "')",
		"getElementById('" + ReasonIDPrefix + "'+kind)",
		"getElementById('" + DetailID + "')",
		"setAttribute('" + KindAttr + "',kind)",
		"setAttribute('" + StateAttr + "','" + StateFailed + "')",
		"reveal('" + KindUnsupported + "','')",
		"reveal('" + KindLoaderMissing + "','')",
		"reveal('" + KindBootFailed + "',error)",
		"var url='" + parseOptions.WASMURL + "'",
	} {
		if !strings.Contains(parseScript, parseWant) {
			parseT.Fatalf("boot script missing %q, got %q", parseWant, parseScript)
		}
	}
	for _, parseKind := range []string{KindUnsupported, KindLoaderMissing, KindBootFailed} {
		if !strings.Contains(parseMarkup, `id="`+ReasonIDPrefix+parseKind+`"`) {
			parseT.Fatalf("boot script reveals %q but the markup has no such block", parseKind)
		}
	}
}

// TestBootScriptKeepsBothInstantiationPaths guards the two behaviours that are
// the whole reason this snippet is longer than one line: the MIME-safe
// ArrayBuffer fallback, and the HTTP status check that turns a 404 into a
// sentence naming the URL instead of "expected magic word 00 61 73 6d".
func TestBootScriptKeepsBothInstantiationPaths(parseT *testing.T) {
	parseScript := BootScript(testOptions())

	for _, parseWant := range []string{
		"typeof WebAssembly!=='object'",
		"typeof WebAssembly.instantiate!=='function'",
		"typeof Go!=='function'",
		"typeof WebAssembly.instantiateStreaming==='function'",
		"WebAssembly.instantiateStreaming(fetch(url),go.importObject).catch(instantiateFromBuffer)",
		"if(!response.ok)",
		"response.arrayBuffer()",
		"WebAssembly.instantiate(bytes,go.importObject)",
		"go.run(result.instance)",
	} {
		if !strings.Contains(parseScript, parseWant) {
			parseT.Fatalf("boot script missing %q, got %q", parseWant, parseScript)
		}
	}
	// The fallback must re-fetch rather than reuse the streamed Response, whose
	// body instantiateStreaming has already consumed.
	if strings.Count(parseScript, "fetch(url)") != 2 {
		parseT.Fatalf("expected exactly two fetches (streaming plus a fresh one for the buffer path), got %q", parseScript)
	}
	// innerHTML in a surface that renders browser error text is an XSS sink.
	if strings.Contains(parseScript, "innerHTML") {
		parseT.Fatalf("boot script must not use innerHTML, got %q", parseScript)
	}
}

// TestBootScriptStaysMinimal is a deliberate size ratchet. The snippet is a
// bootstrap; the moment it grows a loader framework, retries, or telemetry, that
// logic belongs in Go instead. 29 non-blank lines including the <script> tags is
// the current cost of "instantiate the module and explain it when that fails";
// the cap is one line of slack above it, so growth is a decision rather than a
// drift.
const bootScriptLineCap = 30

func TestBootScriptStaysMinimal(parseT *testing.T) {
	parseScript := BootScript(testOptions())
	parseLines := 0
	for _, parseLine := range strings.Split(parseScript, "\n") {
		if strings.TrimSpace(parseLine) != "" {
			parseLines++
		}
	}
	if parseLines > bootScriptLineCap {
		parseT.Fatalf("boot snippet grew to %d lines (cap %d); JavaScript beyond bootstrapping and reporting belongs in Go", parseLines, bootScriptLineCap)
	}
	for _, parseForbidden := range []string{"setTimeout", "setInterval", "XMLHttpRequest", "navigator.sendBeacon", "localStorage"} {
		if strings.Contains(parseScript, parseForbidden) {
			parseT.Fatalf("boot snippet uses %q; that behaviour belongs in Go once the runtime is up", parseForbidden)
		}
	}
}

// TestMarkupEscapesInterpolatedDeploymentFacts keeps the fallback from becoming
// an injection point if a URL or build command ever carries markup characters.
func TestMarkupEscapesInterpolatedDeploymentFacts(parseT *testing.T) {
	parseMarkup := Markup(Options{
		WASMURL:      "/assets/bin/<script>alert(1)</script>.wasm",
		LoaderURL:    "/assets/script/wasm_exec.js",
		BuildCommand: `go build -o "out" && echo <done>`,
	}, false)

	if strings.Contains(parseMarkup, "<script>alert(1)</script>") {
		parseT.Fatalf("expected escaped URL in fallback copy, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		parseT.Fatalf("expected HTML-escaped URL, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "&lt;done&gt;") {
		parseT.Fatalf("expected HTML-escaped build command, got %q", parseMarkup)
	}
}
