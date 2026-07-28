// Package bootfallback owns every byte of JavaScript the Atlas document ships,
// and every byte of markup that JavaScript reveals when the client cannot start.
//
// # WHY A GO PACKAGE FOR THIS
//
// GoWebComponents' claim is that a browser application can be written in Go. The
// honest version of that claim has to name the exception, because there is
// exactly one and it is not zero:
//
//	<script src="/assets/script/wasm_exec.js"></script>   (575 lines, from $GOROOT/lib/wasm)
//	<script> ...the snippet in BootScript()... </script>  (~20 lines, generated here)
//
// That is the whole JavaScript surface of Atlas. Everything else — the fallback
// copy, the fallback markup, the inline styling of that markup, the <noscript>
// content, the decision of which failure message applies — is Go, in this file.
// The snippet's only jobs are the two things Go provably cannot do from inside a
// wasm module that has not started yet:
//
//  1. Instantiate the module. Go code cannot instantiate the Go runtime that
//     would run it. Something outside the module must call
//     WebAssembly.instantiate* and hand the import object over, and in a browser
//     the only language that can do that is JavaScript.
//  2. Report that step failing. If instantiation fails there is no Go heap, no
//     syscall/js bridge, and no reconciler; a failure surface written in Go
//     would need the very thing that just failed. (The client's own richer
//     surface, client/bootsurface.go, is plain JS for the same reason one layer
//     up: a Go panic calls exit(2), after which every js.FuncOf callback throws
//     "bad callback: Go program has already exited".)
//
// # WHY wasm_exec.js CANNOT BE REPLACED
//
// wasm_exec.js is not a convenience wrapper. It is the other half of the Go wasm
// runtime's ABI: it implements the ~40 host functions the compiler emits imports
// for (runtime.wasmWrite, runtime.scheduleTimeoutEvent, syscall/js.valueCall,
// syscall/js.valueNew, the whole JS value reference table), plus argv/environ
// setup and the memory-growth dance. It ships with the toolchain in
// $GOROOT/lib/wasm and is version-locked to the compiler that built the module.
// Hand-writing or trimming it means reimplementing an unstable internal ABI, and
// a mismatch surfaces as an "invalid import" at instantiate time or, worse, as
// memory corruption at runtime. Copy it, do not author it. (GOOS=wasip1 avoids
// it, but wasip1 has no DOM access, so it is not an option for a UI.)
//
// # THE FAILURE HISTORY THIS EXISTS TO FIX
//
// Atlas's boot snippet used to be one line ending in
// `.catch(err => console.error('Failed to hydrate Atlas WASM:', err))`, against a
// document whose body is `<div id="app"></div>`. Every failure — 404 on the
// binary, wrong Content-Type, corrupt build, WebAssembly switched off by policy,
// scripting disabled — produced the identical result: a blank page and one line
// in a console nobody had open. Atlas shipped four months in that state (the
// client called atlas.App() outside a render pass and the runtime panicked)
// partly because a broken boot and a slow boot were pixel-identical.
//
// So the rule this package encodes: no boot failure is allowed to be silent. Each
// one names what failed, whether it is the browser's fault or the deployment's,
// and what to do next.
package bootfallback

import (
	"fmt"
	"html"
	"strings"
)

// DOM contract. These are the assertion hooks for browser tests and the shared
// vocabulary between the generated JS and the generated markup. Copy is localized
// and gets rewritten; attribute values do not, so tests should assert on these
// rather than on sentences.
const (
	// HostID is the wrapper that holds every reason block. The snippet unhides
	// it; the server renders it already visible for the missing-binary case,
	// which needs no JavaScript at all because the server knows at render time.
	HostID = "atlas-boot-fallback"

	// ReasonIDPrefix + kind is the id of the block for one specific failure.
	// The snippet builds the id by concatenation, which is why the kinds below
	// are the single source of truth for both sides.
	ReasonIDPrefix = HostID + "-"

	// DetailID is the <pre> that receives the browser's own error text. It is a
	// separate node so the snippet can use textContent: the string can contain a
	// URL or a server message, i.e. content this page does not control, and an
	// error surface that uses innerHTML turns a blank page into an XSS sink.
	DetailID = HostID + "-detail"

	// KindAttr on the host records which failure was revealed, for tests and for
	// anyone reading the DOM after the fact.
	KindAttr = "data-atlas-boot-fallback"

	// NoScriptAttr marks the <noscript> payload.
	NoScriptAttr = "data-atlas-noscript"

	// StateAttr / StateFailed mirror client/bootsurface.go's boot-state contract
	// so "did this page fail?" is one selector regardless of which layer noticed.
	StateAttr   = "data-atlas-boot-state"
	StateFailed = "failed"
)

// Failure kinds. Each is a genuinely different cause with a genuinely different
// remedy, which is why there are four blocks instead of one generic message.
const (
	// KindIdle is the host's attribute value before anything has failed.
	KindIdle = "idle"

	// KindUnsupported: no WebAssembly runtime in this browser. Not a fault —
	// an environment limit. Nothing the operator can fix.
	KindUnsupported = "wasm-unsupported"

	// KindLoaderMissing: wasm_exec.js did not load, so `Go` is undefined. A
	// deployment fault: static assets are not being served.
	KindLoaderMissing = "loader-missing"

	// KindBootFailed: fetch or compile of the module failed. Deployment fault
	// (missing, truncated, or mis-typed binary) or a dropped request.
	KindBootFailed = "boot-failed"

	// KindBinaryMissing: the server checked its own disk and the binary is not
	// there. Known before the response is written, so this one is rendered
	// visible with no script involved.
	KindBinaryMissing = "wasm-missing"
)

// Options carries the deployment facts the copy needs to be specific. Vague
// failure text ("Something went wrong") is the thing this package exists to
// prevent, and a message can only be specific if it can name the URL it wanted
// and the command that produces it.
type Options struct {
	// WASMURL is the browser-visible URL of the client module.
	WASMURL string
	// LoaderURL is the browser-visible URL of wasm_exec.js.
	LoaderURL string
	// BuildCommand is the command that produces WASMURL's file.
	BuildCommand string
}

// Inline styles, not classes.
//
// The Atlas visual system (.atlas-*, --atlas-*) lives in an inline <style> in
// client/atlas-commerce-os.html and is NOT in either stylesheet the server links
// (assets/css/tailwind.css, assets/css/example-shell.css). A fallback panel
// styled with .atlas-shell-card would look right on the standalone page and be an
// invisible unstyled div on the served one — silence again, one layer down. A
// failure surface may not depend on anything that can fail, and that includes a
// stylesheet.
//
// Note what is absent from panelStyle: any `display` declaration. The panels are
// toggled with the `hidden` attribute, and tailwind's reset ends with
// `[hidden] { display: none !important }`. An inline `display:block` here would
// lose to that !important rule in one direction and beat `hidden` in the other,
// so the styles stay layout-only and visibility stays one attribute.
const (
	hostStyle = "margin:2rem auto;max-width:52rem;padding:0 1.25rem;"

	panelFaultStyle = "padding:1.25rem 1.4rem;border:1px solid #b4534f;border-left-width:6px;" +
		"border-radius:0.75rem;background:#2b1113;color:#f6d9d9;" +
		"font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;" +
		"font-size:0.95rem;line-height:1.6;box-shadow:0 18px 40px rgba(0,0,0,0.35);text-align:left;"

	// A blocked environment is not a broken deployment, and the two should not
	// look the same: amber reads as "this cannot run here", red as "this is
	// broken". Operators triage by colour before they read.
	panelBlockedStyle = "padding:1.25rem 1.4rem;border:1px solid #8a6a2f;border-left-width:6px;" +
		"border-radius:0.75rem;background:#241a0d;color:#f1e2c6;" +
		"font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;" +
		"font-size:0.95rem;line-height:1.6;box-shadow:0 18px 40px rgba(0,0,0,0.35);text-align:left;"

	headlineStyle = "margin:0 0 0.6rem 0;font-size:1.15rem;font-weight:700;color:#ffece9;"
	bodyStyle     = "margin:0 0 0.75rem 0;"
	actionStyle   = "margin:0;font-size:0.88rem;opacity:0.85;"
	detailStyle   = "margin:0.85rem 0 0 0;padding:0.7rem 0.85rem;border-radius:0.5rem;background:#180a0b;" +
		"color:#ffc9c4;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:0.82rem;" +
		"line-height:1.5;white-space:pre-wrap;word-break:break-word;overflow-x:auto;"
)

// reason is one cause-specific block of the fallback.
type reason struct {
	kind      string
	blocked   bool
	headline  string
	body      string
	action    string
	hasDetail bool
}

// reasonsFor returns the blocks to render.
//
// When the binary is missing the server already knows the single cause, so it
// renders that one block visible and skips the script entirely — the
// zero-JavaScript path through this whole file. When the binary is present the
// cause cannot be known server-side, so all three script-detectable blocks ship
// hidden and the snippet reveals exactly one. Shipping all three costs ~2 KB of
// gzipped markup and buys a specific message for every failure without a second
// round trip on a page that just proved it cannot make round trips reliably.
func reasonsFor(parseOptions Options, isBinaryPresent bool) []reason {
	if !isBinaryPresent {
		return []reason{{
			kind:     KindBinaryMissing,
			headline: "Atlas has no client program to load",
			body: "The server is running, but " + parseOptions.WASMURL + " is not on disk. " +
				"Atlas renders its entire interface from that file, so this page is the document " +
				"shell and nothing else: no catalog, no warehouse availability, no internal workspace. " +
				"Nothing here is browsable yet.",
			action: "Build the client from the repository root, then reload this page: " + parseOptions.BuildCommand,
		}}
	}
	return []reason{
		{
			kind:     KindUnsupported,
			blocked:  true,
			headline: "This browser cannot run Atlas",
			body: "Atlas ships its interface as a WebAssembly program, and this browser exposes no " +
				"WebAssembly runtime. There is no reduced version of the page to fall back to: " +
				"the catalog, warehouse, and workspace screens are all rendered by that program.",
			action: "Open this URL in a current version of Chrome, Edge, Firefox, or Safari. If you are " +
				"already using one, WebAssembly is being switched off for this page — usually by an " +
				"enterprise policy, a hardened browser profile, or a content blocker.",
		},
		{
			kind:     KindLoaderMissing,
			headline: "Atlas is missing its loader",
			body: "The page loaded but " + parseOptions.LoaderURL + " did not, so the browser has no way " +
				"to start the client program. That file is the Go runtime's own JavaScript bridge and it " +
				"ships with the toolchain, so this is a serving fault rather than a build one.",
			action: "Reload once, in case the request was dropped. If this stays, the server is not serving " +
				"its static assets and the operator needs to fix that before the page can work.",
		},
		{
			kind:     KindBootFailed,
			headline: "Atlas could not load its client program",
			body: "The browser reached this page but could not download or compile " + parseOptions.WASMURL + ". " +
				"Until it can, this page has no content: every screen is rendered by that program.",
			action: "Reload to retry. If it keeps failing, the served file is missing, truncated, or served " +
				"with the wrong content type, and the exact browser error is above.",
			hasDetail: true,
		},
	}
}

// Markup renders the fallback host, its reason blocks, and the <noscript>
// payload. It is emitted AFTER <div id="app"></div> and never inside it: markup
// inside the mount point is diffed against the client tree during hydration and
// reported as a mismatch (see client/bootsurface.go's note on the boot
// placeholder). Anything the page ships for the pre-hydration or failed-boot case
// has to be a sibling of the mount, not a child.
func Markup(parseOptions Options, isBinaryPresent bool) string {
	parseBuilder := &strings.Builder{}

	parseBuilder.WriteString(noScriptMarkup(parseOptions))

	// The host is hidden only when a script is expected to reveal it. With the
	// binary missing there is no script, so hiding it would reproduce the blank
	// page this package exists to eliminate.
	parseHidden := " hidden"
	parseKind := KindIdle
	if !isBinaryPresent {
		parseHidden = ""
		parseKind = KindBinaryMissing
	}
	_, _ = fmt.Fprintf(parseBuilder, `<div id=%q %s=%q role="alert" style=%q%s>`,
		HostID, KindAttr, parseKind, hostStyle, parseHidden)

	for _, parseReason := range reasonsFor(parseOptions, isBinaryPresent) {
		// With the binary missing there is exactly one block and it is the
		// message; with the binary present every block waits for the snippet.
		parseReasonHidden := " hidden"
		if !isBinaryPresent {
			parseReasonHidden = ""
		}
		parseBuilder.WriteString(renderReason(parseReason, parseReasonHidden))
	}

	parseBuilder.WriteString(`</div>`)
	return parseBuilder.String()
}

func renderReason(parseReason reason, parseHidden string) string {
	parseStyle := panelFaultStyle
	if parseReason.blocked {
		parseStyle = panelBlockedStyle
	}
	parseBuilder := &strings.Builder{}
	_, _ = fmt.Fprintf(parseBuilder, `<div id=%q style=%q%s>`,
		ReasonIDPrefix+parseReason.kind, parseStyle, parseHidden)
	_, _ = fmt.Fprintf(parseBuilder, `<p style=%q>%s</p>`, headlineStyle, html.EscapeString(parseReason.headline))
	_, _ = fmt.Fprintf(parseBuilder, `<p style=%q>%s</p>`, bodyStyle, html.EscapeString(parseReason.body))
	_, _ = fmt.Fprintf(parseBuilder, `<p style=%q>%s</p>`, actionStyle, html.EscapeString(parseReason.action))
	if parseReason.hasDetail {
		// Ships empty and hidden. The snippet fills it with the browser's own
		// error text, which is the one piece of the message the server cannot
		// know in advance.
		_, _ = fmt.Fprintf(parseBuilder, `<pre id=%q style=%q hidden></pre>`, DetailID, detailStyle)
	}
	parseBuilder.WriteString(`</div>`)
	return parseBuilder.String()
}

// noScriptMarkup is the only part of this file that needs no cooperation from
// anything: <noscript> is inert HTML the browser reveals on its own when
// scripting is off. Without it a scripting-disabled client gets
// `<div id="app"></div>` and a blank window.
//
// WHEN SSR MARKUP LANDS, CHANGE THIS COPY. Today the server renders head
// metadata and the __ATLAS_BOOTSTRAP__ payload but no page tree, so "this notice
// is the whole page" is literally true. Once the page tree is server-rendered
// into #app, this block must stop claiming the page is empty and start describing
// what is degraded instead — the content is readable, but navigation stays
// full-page, and anything interactive (filtering, quoting, moderation) does not
// respond. Copy that overstates the damage is as misleading as copy that
// understates it, and this is the one sentence in the package that a future SSR
// change silently falsifies.
func noScriptMarkup(parseOptions Options) string {
	parseBody := "Scripting is switched off for this page. Atlas renders its interface from a " +
		"WebAssembly program, and starting that program takes a few lines of JavaScript, so with " +
		"scripting off there is nothing to render: this notice is the whole page. No part of the " +
		"catalog or the workspace is readable in this state."
	parseAction := "Allow scripting for this origin and reload. Atlas needs it for the initial load " +
		"only — the interface itself is compiled Go, not JavaScript."
	parseBuilder := &strings.Builder{}
	_, _ = fmt.Fprintf(parseBuilder, `<noscript><div %s="true" style=%q><div style=%q>`,
		NoScriptAttr, hostStyle, panelBlockedStyle)
	_, _ = fmt.Fprintf(parseBuilder, `<p style=%q>Atlas needs JavaScript to start</p>`, headlineStyle)
	_, _ = fmt.Fprintf(parseBuilder, `<p style=%q>%s</p>`, bodyStyle, html.EscapeString(parseBody))
	_, _ = fmt.Fprintf(parseBuilder, `<p style=%q>%s</p>`, actionStyle, html.EscapeString(parseAction))
	parseBuilder.WriteString(`</div></div></noscript>`)
	return parseBuilder.String()
}

// bootScriptTemplate is the entire hand-written JavaScript surface of Atlas.
// Every line is here because Go cannot do that line from inside a module that is
// not running yet. Read it as a list of things the language boundary forces:
//
//	reveal(kind, detail)
//	    The failure path. It does not BUILD anything: the markup, the copy, and
//	    the styling were all rendered by Markup() above, in Go. This function
//	    only flips `hidden` and writes one attribute, which is the smallest
//	    possible JS half of "server-rendered fallback, client-side reveal".
//	    textContent, never innerHTML — `detail` is a browser error string that can
//	    embed a URL or a server message.
//
//	typeof WebAssembly !== 'object' || typeof WebAssembly.instantiate !== 'function'
//	    Feature detection has to happen before `new Go()`, which touches
//	    WebAssembly.Memory and would throw a ReferenceError into nowhere. Both
//	    halves are checked because "WebAssembly exists" and "WebAssembly is
//	    usable" are different states: a CSP with no 'wasm-unsafe-eval', an
//	    enterprise policy, or a stripped-down embedded webview can leave the
//	    object present and instantiation unavailable.
//
//	typeof Go !== 'function'
//	    wasm_exec.js defines the global `Go`. If that request 404s, this snippet
//	    still runs and `new Go()` throws a ReferenceError that nothing catches —
//	    the blank page again, one layer earlier. One typeof check turns the
//	    repository's most likely deployment mistake into a sentence on screen.
//
//	instantiateFromBuffer()
//	    The compatibility path AND the diagnostic path, which is why it exists
//	    even though instantiateStreaming is available everywhere that matters.
//
//	    Compatibility: instantiateStreaming requires the response to arrive as
//	    Content-Type: application/wasm. The spec makes the browser REJECT rather
//	    than sniff, so any server that does not know the .wasm extension — a
//	    plain static host, an SPA catch-all, a CDN with a default type, an old
//	    IIS install — fails with "Incorrect response MIME type. Expected
//	    'application/wasm'" even though the bytes are perfect. This is the single
//	    most common first-deploy break for Go wasm. WebAssembly.instantiate on an
//	    ArrayBuffer has no MIME requirement at all, so it succeeds where
//	    streaming refused. Go's own $GOROOT/lib/wasm/wasm_exec.html does the same
//	    dance for the same reason.
//
//	    Diagnostic: the fallback re-fetches instead of reusing the first
//	    Response, because instantiateStreaming consumes the body — reading it
//	    again throws "body stream already read" and buries the real cause. The
//	    re-fetch is normally served from the HTTP cache, only ever happens on the
//	    failure path, and it is what makes `if (!response.ok)` reachable. That
//	    matters: on a 404, streaming reports "expected magic word 00 61 73 6d"
//	    (the browser tried to compile an HTML error page), whereas this path
//	    reports "GET /assets/bin/atlas-commerce-os.wasm returned HTTP 404". The
//	    first sentence sends a developer hunting a corrupt build; the second
//	    names the actual problem. Atlas has already lost time to exactly that
//	    confusion, when the client was built to bin/examples/ while the server
//	    served examples/static/.
//
//	go.run(result.instance)
//	    Hands control to Go. Everything after this point in Atlas is Go: the
//	    router, the reconciler, the DOM writes, and — via client/bootsurface.go —
//	    a much richer failure surface than this one, which takes over as soon as
//	    the runtime is alive.
//
// Deliberately NOT here, because each belongs to a layer that can express it
// better: retries and backoff (a reload is the honest recovery for a boot that
// failed, and the fallback says so); telemetry (no endpoint on a page that just
// failed to fetch from this origin); a loading indicator (the standalone shell's
// #atlas-boot-placeholder does that, and the client tears it down); panic and
// runtime-exit reporting (client/bootsurface.go installs listeners for those in
// Go, and it can, because by then the runtime is up).
//
// No comments in the emitted JS: the rationale lives here, where it is
// maintainable, instead of shipping on every page view. No backticks either —
// this is a Go raw string literal.
const bootScriptTemplate = `<script>
(function(){
  function reveal(kind,detail){
    var host=document.getElementById('%[1]s');
    if(!host){return;}
    var block=document.getElementById('%[2]s'+kind);
    if(block){block.hidden=false;}
    var slot=document.getElementById('%[3]s');
    if(slot&&detail){slot.textContent=String(detail);slot.hidden=false;}
    host.setAttribute('%[4]s',kind);
    host.hidden=false;
    document.documentElement.setAttribute('%[5]s','%[6]s');
  }
  if(typeof WebAssembly!=='object'||typeof WebAssembly.instantiate!=='function'){reveal('%[7]s','');return;}
  if(typeof Go!=='function'){reveal('%[8]s','');return;}
  var url='%[10]s';
  var go=new Go();
  function instantiateFromBuffer(){
    return fetch(url).then(function(response){
      if(!response.ok){throw new Error('GET '+url+' returned HTTP '+response.status+' '+response.statusText);}
      return response.arrayBuffer();
    }).then(function(bytes){return WebAssembly.instantiate(bytes,go.importObject);});
  }
  var booted=typeof WebAssembly.instantiateStreaming==='function'
    ? WebAssembly.instantiateStreaming(fetch(url),go.importObject).catch(instantiateFromBuffer)
    : instantiateFromBuffer();
  booted.then(function(result){go.run(result.instance);}).catch(function(error){reveal('%[9]s',error);});
}());
</script>`

// BootScript returns the inline boot snippet. It is only emitted when the module
// is actually on disk; with the module missing, Markup() has already rendered a
// visible explanation and there is nothing for a script to add.
//
// The ids and kinds are substituted from the constants above rather than written
// twice, so a rename cannot leave the JS reaching for an element the Go stopped
// emitting — a drift that would fail silently, at exactly the moment the page is
// already broken.
func BootScript(parseOptions Options) string {
	return fmt.Sprintf(bootScriptTemplate,
		HostID,            // 1
		ReasonIDPrefix,    // 2
		DetailID,          // 3
		KindAttr,          // 4
		StateAttr,         // 5
		StateFailed,       // 6
		KindUnsupported,   // 7
		KindLoaderMissing, // 8
		KindBootFailed,    // 9
		parseOptions.WASMURL,
	)
}
