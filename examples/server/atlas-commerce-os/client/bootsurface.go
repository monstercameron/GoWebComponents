//go:build js && wasm
// +build js,wasm

package main

// Failure surfaces for the Atlas client.
//
// WHY THIS FILE EXISTS
//
// Every production defect Atlas has shipped failed by producing NOTHING:
//
//   1. 2026-04-08 → 2026-07-26: the client called atlas.App(payload) eagerly, which
//      ran useAtlasAtom outside a render pass. The runtime panicked with
//      "GoUseAtom called outside component context". The user saw a blank page for
//      four months. The panic report went to a console nobody had open.
//   2. The client wasm was built to bin/examples/ while the server serves
//      examples/static/. /healthz reported "wasmPresent":true, the browser got a
//      404, and the page shipped `<div id="app"></div>` plus a console.warn.
//   3. In a sibling v5 example a worker-level failure carried no request id, was
//      delivered against id 0, matched no waiter, and was dropped: a command that
//      never returned and never errored.
//
// docs/PRODUCTION_READINESS.md draws the conclusion out of the M10 measurement:
// "every one of them fails by producing nothing, and the test only became
// diagnosable after it captured console output, page errors, and failed requests."
//
// So this file's whole job is to convert silence into text on the screen. It is
// deliberately built out of nothing but document.createElement and inline styles,
// for two reasons that both come from real failure modes:
//
//   - It cannot render through the GWC component tree, because the tree is what
//     may have just died. An error surface that needs a working reconciler is not
//     an error surface.
//   - It cannot depend on the design system. shared/design emits its rules through
//     the css package, which on wasm injects them into a <style> element the moment
//     a class is folded — i.e. from inside the same Go program that may have just
//     died. A surface styled with design.Class(design.Surface()) would be a correctly
//     styled panel on a healthy page and an invisible unstyled div on precisely the
//     boot that needed it: silence again, one layer down. Inline styles are the only
//     ones guaranteed to be there.
//
//     (Historical note: this used to say the visual system lived in an inline
//     <style> in client/atlas-commerce-os.html and in neither stylesheet the server
//     linked. There are no stylesheets now — the server inlines the design layer as
//     <style data-gwc-css> and this client seeds from it — but the conclusion is
//     unchanged and the reason is stronger, since the injector is now in-process.)

import (
	"strings"
	"syscall/js"
)

const (
	// atlasBootStateAttr is written on <html> at each boot milestone. It is the
	// always-on counterpart to debugLog: debug logs are gated behind the server's
	// data-atlas-debug-logs flag and therefore vanish in production, while this
	// attribute costs one setAttribute and survives every build. It is also the
	// assertion hook a browser test should use instead of scraping copy, since
	// copy is localized and changes, and "boot state" does not.
	atlasBootStateAttr = "data-atlas-boot-state"

	atlasBootStateBooting  = "booting"
	atlasBootStateHydrated = "hydrated"
	atlasBootStateFailed   = "failed"

	// atlasBootSurfaceScriptID marks the injected surface so a double install (a
	// hot reload, a second main()) is a no-op rather than a duplicated banner.
	atlasBootSurfaceScriptID = "__ATLAS_BOOT_SURFACE__"
	atlasBootSurfaceStyleID  = "__ATLAS_BOOT_SURFACE_STYLE__"

	// atlasBootSkeletonAttr marks pre-hydration placeholder markup that the page
	// shipped in its HTML. The client owns tearing it down; see
	// clearAtlasBootPlaceholder.
	atlasBootSkeletonAttr = "data-atlas-boot-skeleton"

	atlasSurfaceModeFatal  = "fatal"
	atlasSurfaceModeNotice = "notice"
)

// atlasCapturedPanic is the subset of the runtime's panic report that is worth
// putting on screen. The full report (frames, component stack, remediation,
// docs anchor) stays in the console, which is the right place for it — it is long,
// and it is written for a developer.
type atlasCapturedPanic struct {
	Code    string
	Message string
	Next    string
	Docs    string
	Phase   string
	Subject string
}

var (
	isBootSurfaceInstalled bool
	capturedPanic          atlasCapturedPanic
	hasCapturedPanic       bool

	// js.Func values must stay reachable from Go for as long as JS can call them:
	// a collected Func throws "syscall/js: Value.Call: ..." on invoke. These are
	// package-level for lifetime, not for convenience.
	runtimePanicListener js.Func
	goLivenessProbe      js.Func
	hydrationAuditTimer  js.Func
)

// atlasBootSurfaceJS is the failure surface itself, in plain JS.
//
// WHY PLAIN JS RATHER THAN GO:
//
// A Go panic that is not contained calls exit(2). After that the Go heap is gone
// and every js.FuncOf callback throws "bad callback: Go program has already
// exited" (see wasm_exec.js). An error surface written in Go therefore cannot
// report the one failure the user is most likely to hit — the app dying — because
// it dies with it. These functions are ordinary JS closures, so they outlive the
// Go runtime and can still repaint the page after it is gone.
//
// It is injected as a <script> element with textContent rather than eval()d: an
// inline script needs script-src 'unsafe-inline' (which this server already relies
// on for its bootstrap payload), while eval additionally needs 'unsafe-eval'.
// Choosing the narrower requirement keeps this working if a CSP is ever added.
//
// No backticks anywhere below: this is a Go raw string literal.
const atlasBootSurfaceJS = `
(function () {
  if (window.__atlasSurfaceFailure) { return; }

  var STATE_ATTR = 'data-atlas-boot-state';
  var MAX_NOTICES = 3;
  var noticeCount = 0;
  var livenessTimer = null;
  var stoppedReported = false;

  var PANEL = 'margin:0 auto;max-width:52rem;padding:1.25rem 1.4rem;border:1px solid #b4534f;' +
    'border-left-width:6px;border-radius:0.75rem;background:#2b1113;color:#fdeaea;' +
    'font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;font-size:0.95rem;line-height:1.6;' +
    'box-shadow:0 18px 40px rgba(0,0,0,0.35);text-align:left;';
  var HEAD = 'margin:0 0 0.5rem 0;font-size:1.15rem;font-weight:700;color:#ffd9d6;';
  var BODY = 'margin:0 0 0.75rem 0;color:#f6d9d9;';
  var CODE = 'margin:0 0 0.75rem 0;padding:0.7rem 0.85rem;border-radius:0.5rem;background:#180a0b;' +
    'color:#ffc9c4;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:0.82rem;' +
    'line-height:1.5;white-space:pre-wrap;word-break:break-word;overflow-x:auto;';
  var HINT = 'margin:0;color:#e0b3b1;font-size:0.85rem;';
  var BTN = 'margin-top:0.9rem;padding:0.45rem 0.9rem;border:1px solid #d78b87;border-radius:999px;' +
    'background:transparent;color:#ffd9d6;font:inherit;font-size:0.85rem;cursor:pointer;';

  function docRoot() { return document.documentElement; }

  function bootState() {
    var el = docRoot();
    return el ? (el.getAttribute(STATE_ATTR) || '') : '';
  }

  function setBootState(value) {
    var el = docRoot();
    if (el) { el.setAttribute(STATE_ATTR, value); }
  }

  // textContent everywhere, never innerHTML. Failure text is not trusted input:
  // it can carry a request URL, a query string, or a server message, i.e. content
  // an attacker may influence. An error surface that injects markup turns a blank
  // page into an XSS sink.
  function node(tag, style, text) {
    var el = document.createElement(tag);
    if (style) { el.setAttribute('style', style); }
    if (text) { el.textContent = text; }
    return el;
  }

  function reloadButton() {
    var btn = node('button', BTN, 'Reload the page');
    btn.setAttribute('type', 'button');
    // A dead Go runtime cannot re-render, so reloading is the only real recovery
    // and the surface should offer it rather than describe it.
    btn.addEventListener('click', function () { window.location.reload(); });
    return btn;
  }

  function buildPanel(record, roleValue) {
    var panel = node('div', PANEL, null);
    panel.setAttribute('role', roleValue);
    panel.setAttribute('data-atlas-error-surface', record.mode || 'fatal');
    panel.setAttribute('data-atlas-error-kind', record.kind || 'unknown');
    panel.appendChild(node('p', HEAD, record.headline || 'Atlas failed to start'));
    if (record.detail) { panel.appendChild(node('p', BODY, record.detail)); }
    if (record.technical) { panel.appendChild(node('pre', CODE, record.technical)); }
    panel.appendChild(node('p', HINT, record.hint ||
      'Open the browser console for the full report: the GWC runtime names the file, ' +
      'the rule that was violated, the fix, and a docs anchor.'));
    panel.appendChild(reloadButton());
    return panel;
  }

  function dropNotices() {
    // A fatal panel supersedes every notice: they are almost always the same
    // incident told less completely, and two panels describing one failure reads
    // as two failures.
    var open = document.querySelectorAll('[data-atlas-runtime-notice]');
    for (var i = open.length - 1; i >= 0; i--) {
      var el = open[i];
      if (el.parentNode) { el.parentNode.removeChild(el); }
    }
  }

  function showFatal(record) {
    // First failure wins. Later errors during a failed boot are almost always
    // consequences of the first one, and overwriting the cause with a symptom is
    // how a diagnosable failure becomes a confusing one.
    if (bootState() === 'failed') { return false; }
    var host = document.getElementById('app') || document.body;
    if (!host) { return false; }
    dropNotices();
    var shell = node('div', 'padding:2.5rem 1.25rem;', null);
    shell.setAttribute('data-atlas-boot-failure', 'true');
    shell.appendChild(buildPanel(record, 'alert'));
    // Replacing the mount contents is deliberate: whatever is in there did not
    // finish rendering, so leaving it up presents a half-built UI as a working one.
    host.textContent = '';
    host.appendChild(shell);
    setBootState('failed');
    return true;
  }

  function showNotice(record) {
    if (noticeCount >= MAX_NOTICES) { return false; }
    if (bootState() === 'failed') { return false; }
    if (!document.body) { return false; }
    noticeCount += 1;
    // Prepended in NORMAL FLOW rather than position:fixed. A fixed bar was tried
    // first and covered the fatal panel underneath it — a banner that hides the
    // more important message. Pushing the page down costs one layout shift, on a
    // page that has already failed, and never hides anything.
    var bar = node('div', 'padding:0.75rem;background:rgba(6,4,5,0.94);', null);
    bar.setAttribute('data-atlas-runtime-notice', 'true');
    bar.appendChild(buildPanel(record, 'status'));
    document.body.insertBefore(bar, document.body.firstChild);
    return true;
  }

  window.__atlasSurfaceFailure = function (record) {
    if (!record) { return false; }
    // Kept for tests and for a human poking at the console after the fact: the
    // rendered panel is the primary channel, these are the machine-readable ones.
    // __atlasBootFailure holds the first FATAL record — a notice must not claim
    // that slot, or a "kept going" warning ends up filed as the boot failure while
    // the real one is on screen. __atlasLastFailure holds the most recent of either.
    if (!window.__atlasBootFailure && record.mode !== 'notice') { window.__atlasBootFailure = record; }
    window.__atlasLastFailure = record;
    if (record.mode === 'notice') { return showNotice(record); }
    return showFatal(record);
  };

  // An unhandled rejection is the shape a failed fetch/instantiate takes. The
  // server's boot snippet ends in .catch(err => console.error(...)), so a wasm
  // that 404s or fails to instantiate is already reported — to the console only,
  // which is exactly the gap this closes.
  window.addEventListener('unhandledrejection', function (event) {
    var reason = event ? event.reason : null;
    var text = reason && reason.message ? reason.message : String(reason);
    // Ignore navigation/transition aborts. Clicking a second link before the first
    // route settles rejects the in-flight transition with an AbortError, and
    // "AbortError: Transition was skipped" is observable on ordinary fast clicking
    // through the Atlas nav. Reporting it would train the user to ignore this
    // banner, which costs more than the one real failure it might one day catch.
    if ((reason && reason.name === 'AbortError') || text.indexOf('AbortError') !== -1) {
      return;
    }
    window.__atlasSurfaceFailure({
      mode: bootState() === 'hydrated' ? 'notice' : 'fatal',
      kind: 'unhandled-rejection',
      headline: bootState() === 'hydrated'
        ? 'Atlas hit an unhandled error while running'
        : 'Atlas could not finish loading',
      detail: 'A promise rejected with nobody listening. Before hydration this is ' +
        'usually the client wasm failing to download or instantiate.',
      technical: text
    });
  });

  window.addEventListener('error', function (event) {
    var text = event && event.message ? event.message : 'unknown page error';
    // wasm_exec.js throws exactly this out of every callback once the Go program
    // has exited, so an ordinary click on a dead page lands here. That is the
    // moment a "frozen" page can finally explain itself.
    if (text.indexOf('Go program has already exited') !== -1) {
      reportStopped(text);
      return;
    }
    if (bootState() === 'hydrated' || bootState() === 'failed') { return; }
    window.__atlasSurfaceFailure({
      mode: 'fatal',
      kind: 'page-error',
      headline: 'Atlas could not finish loading',
      detail: 'A script error interrupted the boot sequence.',
      technical: text
    });
  });

  function reportStopped(why) {
    // Two independent detectors find the same death — the liveness interval and
    // the window 'error' listener catching wasm_exec's "already exited" throw —
    // and without this flag both reported it, giving the user two banners for one
    // event. Redundant DETECTION is wanted; redundant REPORTING is not.
    if (stoppedReported) { return; }
    stoppedReported = true;
    if (livenessTimer !== null) { window.clearInterval(livenessTimer); livenessTimer = null; }
    window.__atlasSurfaceFailure({
      // If the app had already rendered, its content is stale but still readable,
      // so say so in a banner instead of destroying what the user was looking at.
      // If it never rendered, the blank page IS the problem and gets the full panel.
      mode: bootState() === 'hydrated' ? 'notice' : 'fatal',
      kind: 'runtime-exit',
      headline: 'Atlas stopped running',
      detail: 'The Go runtime exited, so nothing on this page will respond any more. ' +
        'A Go panic exits with code 2; wasm_exec.js reports that as a single ' +
        'console.warn("exit code:", 2) and the page otherwise just looks frozen.',
      technical: why,
      hint: 'The panic trace is in the console, above this message.'
    });
  }

  // Liveness probe. __atlasGoAlive is a Go callback, so calling it after the Go
  // program exits throws rather than returning — the throw IS the signal. The
  // interval is long (2.5s) because this only has to beat a human noticing that
  // clicking does nothing, and it clears itself the moment it fires.
  livenessTimer = window.setInterval(function () {
    var probe = window.__atlasGoAlive;
    if (typeof probe !== 'function') {
      // Not installed yet (or never installed, if Go died before this line). The
      // blank-page case is covered by the hydration audit, so stay quiet here.
      return;
    }
    try {
      if (probe() !== true) { reportStopped('__atlasGoAlive did not answer'); }
    } catch (err) {
      reportStopped(err && err.message ? err.message : String(err));
    }
  }, 2500);

  window.__atlasBootSurfaceReady = true;
}());
`

// atlasBootSurfaceCSS carries only what inline style attributes cannot express:
// keyframes and a reduced-motion opt-out. The failure panels deliberately do NOT
// use it — they are styled inline so they still render if this injection fails.
// The route skeleton does, and degrades to a static block without it.
const atlasBootSurfaceCSS = `
@keyframes atlas-boot-pulse { 0%,100% { opacity: 0.45; } 50% { opacity: 0.9; } }
[data-atlas-loading] .atlas-boot-bar,
[data-atlas-boot-skeleton] .atlas-boot-bar { animation: atlas-boot-pulse 1.4s ease-in-out infinite; }
@media (prefers-reduced-motion: reduce) {
  [data-atlas-loading] .atlas-boot-bar,
  [data-atlas-boot-skeleton] .atlas-boot-bar { animation: none; opacity: 0.6; }
}
`

// installAtlasBootSurface injects the failure surface. It must run FIRST in main,
// before any work that can fail, because a surface installed after the failure it
// was meant to report is decoration.
func installAtlasBootSurface() {
	if isBootSurfaceInstalled {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseHead := parseDocument.Get("head")
	parseParent := parseHead
	if !parseParent.Truthy() {
		parseParent = parseDocument.Get("body")
	}
	if !parseParent.Truthy() {
		return
	}
	if !parseDocument.Call("getElementById", atlasBootSurfaceStyleID).Truthy() {
		parseStyle := parseDocument.Call("createElement", "style")
		parseStyle.Set("id", atlasBootSurfaceStyleID)
		parseStyle.Set("textContent", atlasBootSurfaceCSS)
		parseParent.Call("appendChild", parseStyle)
	}
	if !parseDocument.Call("getElementById", atlasBootSurfaceScriptID).Truthy() {
		parseScript := parseDocument.Call("createElement", "script")
		parseScript.Set("id", atlasBootSurfaceScriptID)
		// Appending a <script> with textContent executes it synchronously, so
		// window.__atlasSurfaceFailure is callable on the next Go statement.
		parseScript.Set("textContent", atlasBootSurfaceJS)
		parseParent.Call("appendChild", parseScript)
	}
	isBootSurfaceInstalled = true
}

// markAtlasBootState records a boot milestone on <html>.
//
// This exists because the three interesting boot milestones (hydrate.start,
// hydrate.done, router.mount) are debugLog calls, and debugLog returns early
// unless the server set data-atlas-debug-logs="1" — which it does only when
// LogsEnabled is on. In production those three lines do not exist, so the one
// question you always want answered ("did it get past hydration?") had no answer
// at all. One attribute answers it in every build, at no console cost.
func markAtlasBootState(parseState string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return
	}
	if strings.EqualFold(strings.TrimSpace(parseRoot.Call("getAttribute", atlasBootStateAttr).String()), atlasBootStateFailed) {
		// Never downgrade a failed boot back to a hopeful state: the later
		// milestone may still fire (a contained panic does not stop main), and
		// overwriting it would erase the only durable record of the failure.
		return
	}
	parseRoot.Call("setAttribute", atlasBootStateAttr, parseState)
}

// isAtlasBootFailed reports whether a fatal surface is already on screen.
func isAtlasBootFailed() bool {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(parseRoot.Call("getAttribute", atlasBootStateAttr).String()), atlasBootStateFailed)
}

// surfaceAtlasFailure puts a failure on the page and in the console.
//
// Both, always, deliberately: the DOM is where a user or a QA engineer will see
// it, the console is where the actionable detail lives, and the two channels fail
// independently (a broken mount point kills the first, a closed devtools window
// hides the second).
func surfaceAtlasFailure(parseMode string, parseKind string, parseHeadline string, parseDetail string, parseTechnical string) {
	errorLog("boot."+parseKind, parseHeadline+": "+parseTechnical, map[string]any{
		"kind":   parseKind,
		"mode":   parseMode,
		"detail": parseDetail,
	})
	parseRecord := map[string]any{
		"mode":      parseMode,
		"kind":      parseKind,
		"headline":  parseHeadline,
		"detail":    parseDetail,
		"technical": parseTechnical,
	}
	// A contained runtime panic reports itself through the gwc:runtime-panic event
	// rather than through the value we recovered, so the remediation text usually
	// lives there and not in parseTechnical. Fold it in when we have it — but only
	// once: callers that already know about the panic pass its text through
	// parseTechnical, and appending it again printed the same report twice.
	if hasCapturedPanic {
		isPanicTextAlreadyShown := capturedPanic.Message != "" && strings.Contains(parseTechnical, capturedPanic.Message)
		if !isPanicTextAlreadyShown {
			parseRecord["technical"] = strings.TrimSpace(strings.Join([]string{parseTechnical, capturedPanic.Code, capturedPanic.Message}, "\n"))
		}
		if parseNext := strings.TrimSpace(capturedPanic.Next); parseNext != "" {
			parseHint := "Next step: " + parseNext
			if parseDocs := strings.TrimSpace(capturedPanic.Docs); parseDocs != "" {
				parseHint += "  Docs: " + parseDocs
			}
			parseRecord["hint"] = parseHint + "  The full report is in the browser console."
		}
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() {
		if parseFn := parseWindow.Get("__atlasSurfaceFailure"); parseFn.Truthy() {
			parseFn.Invoke(js.ValueOf(parseRecord))
			return
		}
	}
	surfaceAtlasFailureLastResort(parseHeadline, parseDetail, parseTechnical)
}

// surfaceAtlasFailureLastResort writes the failure with nothing but createElement
// and textContent, for the case where the injected surface itself never installed
// (no <head>, no <body>, a CSP that blocked the inline script). It is ugly on
// purpose; an ugly message beats a blank page.
func surfaceAtlasFailureLastResort(parseHeadline string, parseDetail string, parseTechnical string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseHost := parseDocument.Call("getElementById", "app")
	if !parseHost.Truthy() {
		parseHost = parseDocument.Get("body")
	}
	if !parseHost.Truthy() {
		return
	}
	parseNode := parseDocument.Call("createElement", "pre")
	parseNode.Call("setAttribute", "data-atlas-boot-failure", "last-resort")
	parseNode.Call("setAttribute", "style", "padding:1rem;white-space:pre-wrap;color:#b91c1c;background:#fff5f5;")
	parseNode.Set("textContent", strings.TrimSpace(parseHeadline+"\n\n"+parseDetail+"\n\n"+parseTechnical))
	parseHost.Call("appendChild", parseNode)
	markAtlasBootState(atlasBootStateFailed)
}

// installRuntimePanicCapture subscribes to the runtime's panic channel.
//
// WHY A LISTENER AND NOT recover():
//
// ui.ensureInitialized configures the runtime with HideRawPanicOutput: true, so
// crash containment is the DEFAULT in wasm. internal/runtime's
// finalizeUnhandledPanicContext then formats the report, emits it, and RETURNS —
// it does not re-panic. Concretely, ui.Hydrate into a selector that matches
// nothing prints a perfect diagnostic and returns a nil error, and a recover() in
// main never runs. Containment is the right default (a re-thrown panic in wasm
// kills the page), but it means the panic is invisible to Go control flow.
//
// internal/runtime/panic_report_console_wasm.go also dispatches every report as a
// "gwc:runtime-panic" CustomEvent, added for "dev-only tooling that wants to avoid
// scraping console output". That is this. It is the only in-page channel that sees
// a contained panic, so it is what the error surface is built on.
func installRuntimePanicCapture() {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
		return
	}
	runtimePanicListener = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseDetail := parseArgs[0].Get("detail")
		if !parseDetail.Truthy() {
			return nil
		}
		capturedPanic = atlasCapturedPanic{
			Code:    jsStringField(parseDetail, "code"),
			Message: jsStringField(parseDetail, "message"),
			Next:    jsStringField(parseDetail, "next"),
			Docs:    jsStringField(parseDetail, "docs"),
			Phase:   jsStringField(parseDetail, "phase"),
			Subject: jsStringField(parseDetail, "subject"),
		}
		hasCapturedPanic = true
		errorLog("runtime.panic", capturedPanic.Code+" "+capturedPanic.Message, map[string]any{
			"phase":   capturedPanic.Phase,
			"subject": capturedPanic.Subject,
			"next":    capturedPanic.Next,
			"docs":    capturedPanic.Docs,
		})
		// During boot, the caller decides what to show once it knows whether the
		// page rendered anyway; a contained panic in one subtree does not
		// necessarily mean a blank page. After boot, show a banner and leave the
		// rendered UI alone — the alternative is wiping a mostly-working app
		// because one widget threw.
		if !isAtlasBootFailed() && strings.EqualFold(bootStateValue(), atlasBootStateHydrated) {
			surfaceAtlasFailure(atlasSurfaceModeNotice, "runtime-panic",
				"Atlas hit a runtime panic and kept going",
				"The runtime contained the panic instead of killing the page, so part of this screen may be stale or missing.",
				capturedPanic.Code+" "+capturedPanic.Message)
		}
		return nil
	})
	parseWindow.Call("addEventListener", "gwc:runtime-panic", runtimePanicListener)
}

// bootStateValue reads the current boot milestone from <html>.
func bootStateValue() string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseRoot.Call("getAttribute", atlasBootStateAttr).String())
}

// jsStringField reads one string property without throwing on absent fields.
//
// js.Value.String() on an undefined property returns the literal "<undefined>",
// a NON-EMPTY string. That exact behaviour is defect 4 in the M10 write-up: every
// successful worker reply was read as a rejection because "<undefined>" is truthy
// to a len() check. Normalising it here keeps that bug out of the error surface,
// which would otherwise print "<undefined>" at people.
func jsStringField(parseValue js.Value, parseName string) string {
	if !parseValue.Truthy() {
		return ""
	}
	parseField := parseValue.Get(parseName)
	if parseField.Type() != js.TypeString {
		return ""
	}
	parseText := strings.TrimSpace(parseField.String())
	if parseText == "<undefined>" || parseText == "undefined" || parseText == "null" {
		return ""
	}
	return parseText
}

// installGoLivenessProbe publishes a callback the JS watchdog can poke.
//
// The value it returns is irrelevant; what matters is whether calling it throws.
// wasm_exec.js raises "bad callback: Go program has already exited" for any
// js.FuncOf invoked after exit, which turns "is the Go runtime alive?" into a
// question JS can ask without cooperation from a runtime that may be gone.
func installGoLivenessProbe() {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	goLivenessProbe = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return true
	})
	parseWindow.Set("__atlasGoAlive", goLivenessProbe)
}

// scheduleHydrationOutputAudit is the guard that catches the four-month blank page.
//
// THE INCIDENT: atlas.App(payload) was called eagerly, useAtlasAtom ran with no
// owning fiber, the runtime panicked, containment swallowed it, ui.Hydrate
// returned a nil error, and #app stayed empty forever. Every synchronous check
// passed. The only observable fact was "the mount point has no children", and
// nobody was looking.
//
// So look. Asynchronously, because a commit is NOT synchronous with the Hydrate
// call: the scheduler commits in a task, and the first route render additionally
// waits on a loader fetch. A synchronous emptiness check would fire on every
// healthy boot. parseDelayMS has to outlast a slow first paint (measured: ~1.3s
// from navigation on localhost with a 19 MB wasm) without outlasting a user's
// patience.
func scheduleHydrationOutputAudit(parseSelector string, parseDelayMS int) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() || !parseWindow.Get("setTimeout").Truthy() {
		return
	}
	hydrationAuditTimer = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		auditHydrationOutput(parseSelector)
		return nil
	})
	parseWindow.Call("setTimeout", hydrationAuditTimer, parseDelayMS)
}

// auditHydrationOutput reports a mount point that never received any output.
func auditHydrationOutput(parseSelector string) {
	if isAtlasBootFailed() {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseHost := parseDocument.Call("querySelector", parseSelector)
	if !parseHost.Truthy() {
		// Already reported synchronously by the pre-hydration mount check; if the
		// element vanished later, say so rather than guessing.
		surfaceAtlasFailure(atlasSurfaceModeFatal, "mount-missing",
			"Atlas lost its mount point",
			"The element matching "+parseSelector+" is no longer in the document, so nothing can render into it.",
			"querySelector("+parseSelector+") returned null after hydration")
		return
	}
	if parseHost.Get("childElementCount").Int() > 0 {
		return
	}
	parseTechnical := parseSelector + " still has 0 child elements"
	parseDetail := "Hydration reported no error, but nothing was rendered. When the runtime contains a panic it prints a report and returns normally, so a nil error does not mean a rendered page."
	if hasCapturedPanic {
		parseDetail = "A contained runtime panic stopped this render. The runtime kept the page alive and returned a nil error, which is why nothing else reported a problem."
	}
	surfaceAtlasFailure(atlasSurfaceModeFatal, "empty-mount",
		"Atlas hydrated without rendering anything",
		parseDetail, parseTechnical)
}

// isServerRenderedDocument reports whether this page came from the Atlas server.
//
// It exists to keep one guard from crying wolf. "No SSR bootstrap payload" is a
// real failure on a served page — the client silently falls back to empty state
// and renders a plausible-looking wrong page. On client/atlas-commerce-os.html it
// is the DESIGNED mode: that page is a static shell with no server behind it, and
// flagging it painted a red banner over a demo that was working exactly as
// intended. A guard that fires on correct behaviour gets muted, and then it is not
// a guard.
//
// data-atlas-surface is the discriminator because the server writes it on <html>
// for every rendered page (renderPageStatusWithPayload) and the static shell does
// not have it.
func isServerRenderedDocument() bool {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return false
	}
	// hasAttribute, NOT getAttribute(...).String() != "". js.Value.String() renders
	// a null as the literal "<null>", so the obvious emptiness test would report
	// every page as server-rendered. Same class of bug as the "<undefined>" read
	// that made every successful worker reply look like a rejection in the M10
	// write-up: syscall/js stringifies absent values into non-empty strings.
	return parseRoot.Call("hasAttribute", "data-atlas-surface").Bool()
}

// atlasMountPointExists reports whether the hydration target is in the document.
func atlasMountPointExists(parseSelector string) bool {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	return parseDocument.Call("querySelector", parseSelector).Truthy()
}

// clearAtlasBootPlaceholder removes pre-hydration placeholder markup.
//
// The placeholder deliberately lives OUTSIDE the mount point (see
// client/atlas-commerce-os.html). Skeleton markup inside #app would be compared
// against the client tree during hydration and reported as a mismatch, so the
// page keeps it as a sibling and the client removes it once real output exists.
// Absent on the server-rendered shell, where this is a no-op — see the note in
// main() about that gap.
func clearAtlasBootPlaceholder() {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseNodes := parseDocument.Call("querySelectorAll", "["+atlasBootSkeletonAttr+"]")
	if !parseNodes.Truthy() {
		return
	}
	for parseIndex := parseNodes.Get("length").Int() - 1; parseIndex >= 0; parseIndex-- {
		parseNode := parseNodes.Index(parseIndex)
		if !parseNode.Truthy() {
			continue
		}
		if parseParent := parseNode.Get("parentNode"); parseParent.Truthy() {
			parseParent.Call("removeChild", parseNode)
		}
	}
}

// errorLog emits a diagnostic that survives a production build.
//
// debugLog is gated on data-atlas-debug-logs, which the server sets from
// cfg.LogsEnabled — off by default. Every failure path in this client used it:
// bootstrap.read.failed, route.fetch.network.error, route.load.error,
// bootstrap.cache.restore.failed. In production none of them printed anything, so
// the shipped app had no error reporting at all while appearing to have thirty
// call sites of it. Diagnostics that are silenced in production are the same
// problem as no diagnostics, wearing a different costume.
//
// The message argument is passed to the console unsanitized and untruncated,
// because a 160-character-clipped error text is not an error text. The details map
// still goes through sanitizeDebugDetails so the same key redaction (token, csrf,
// cookie, password, secret, email) applies to structured fields.
func errorLog(parseEvent string, parseMessage string, parseDetails map[string]any) {
	parseConsole := js.Global().Get("console")
	if !parseConsole.Truthy() {
		return
	}
	if parseDetails == nil {
		parseDetails = map[string]any{}
	}
	parseDetails["event"] = parseEvent
	parseSanitized := sanitizeDebugDetails(parseDetails)
	parseMethod := "error"
	if !parseConsole.Get(parseMethod).Truthy() {
		parseMethod = "log"
		if !parseConsole.Get(parseMethod).Truthy() {
			return
		}
	}
	parseConsole.Call(parseMethod, "[atlas-wasm]", parseEvent, strings.TrimSpace(parseMessage), js.ValueOf(parseSanitized))
	// One durable slot for a browser test or a support session to read back, so
	// proving "the client reported this" does not require having had devtools
	// open at the right moment.
	if parseWindow := js.Global().Get("window"); parseWindow.Truthy() {
		parseWindow.Set("__atlasLastError", js.ValueOf(map[string]any{
			"event":   parseEvent,
			"message": strings.TrimSpace(parseMessage),
			"details": parseSanitized,
		}))
	}
}
