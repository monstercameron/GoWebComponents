# v6 example migration

## Acceptance standard

All application-owned visible elements and interactive controls must be GWC
nodes, including rich-text content, dialogs and benchmark dashboards. Preserve
existing behavior, accessibility, and visual identity. Use public APIs rather
than internal DOM properties. Add regression tests for framework edge cases.

Host documents may provide metadata, scripts and mount containers before Wasm
starts. Intentional third-party benchmark subjects remain comparison controls;
their surrounding dashboard is in scope. Generated catalog mirrors must be
refreshed from canonical sources, not independently patched. These boundaries
do not exempt ordinary application UI or arbitrary direct DOM manipulation.

The initial migration work retained /v5. The maintainer subsequently approved
the /v6 semantic-import migration for v6.0.0 publication. Further example
migration is paused at that release checkpoint; unchecked items below remain
unfinished and are not release acceptance claims.

## Sequential work queue

- [x] Replace chat Markdown internal HTML injection with sanitized GWC nodes;
  validate formatting, hostile markup and content replacement.
- [x] Migrate plain code-block copy controls from chat bootstrap to GWC components.
- [ ] Migrate remaining chat Mermaid/math rich-content ownership.
- [ ] Audit public catalog, SSR examples, static pages and desktop lab for direct
  DOM UI; migrate each identified surface and add an enforceable ownership guard.
- [x] Migrate load-harness dashboard controls and result rendering to GWC.
- [ ] Migrate render-benchmark dashboard while retaining comparison subjects.
- [ ] Resolve existing chat and Atlas browser failures without weakening assertions.
- [ ] Refresh generated mirrors/assets through their build commands.
- [ ] Run all example native, Wasm and browser suites plus affected framework
  regressions; visually inspect representative flows and record exact results.

## Initial audit

Confirmed legacy paths include chat bubble internal innerHTML props, chat
bootstrap JavaScript-created rich-text controls, benchmark-runner innerHTML,
load-harness static controls, and legacy static website DOM construction.
Existing uncommitted application repairs are preserved and must be verified.

## Checkpoint 1 — rich-text ownership

Changed chat bubble rendering and removed its internal HTML property constant;
added a native/Wasm-testable markup renderer and hostile-markup regression.
Extended the public raw-html example and its real-browser test with three
replacement/restore cycles, removed-node checks, unsafe-link checks and adjacent
SVG namespace checks. Native and Wasm focused tests, chat Wasm compilation and
the Chromium regression passed. The first browser attempt used a `section`
outside the documented default sanitizer allowlist; corrected the fixture to
an allowed `div`, without expanding the security policy.

The root-module `go test ./examples/... -count=1 -timeout=10m` baseline exited 0
with 58 tested packages. This does not include nested desktop modules or the
build-tagged browser suite. Log: `bin/test-results/v6-examples-native-migration.log`.
Residual risk: bootstrap rich-text observers still mutate GWC-owned descendants;
their replacement is the next chat migration item. Other example migrations and
full visual/browser acceptance remain open.

## Checkpoint 2 — code-block controls

The plain-code copy button now belongs to a GWC component rather than a
MutationObserver. It uses a cancellable task, reports denied/unavailable clipboard
access, and resets on source replacement. Position plus content identity prevents
identical sibling blocks from colliding and cancels obsolete clipboard work.
Native SSR, Wasm render-fixture tests (including denied clipboard, replacement
and duplicate siblings), and focused server-bootstrap contracts pass. The chat
Wasm client builds. Mermaid and math observers remain pending, not exempt.

The authenticated chat browser journey passed with freshly built assets after
isolating provider credentials (see checkpoint 3).

## Checkpoint 3 — harness dashboard and exposed defects

The load-harness host document now contains only its mount and bootstrap scripts.
GWC owns the run button, status, output and measured subject wrapper. JavaScript
returns report data without writing the DOM; measurement algorithms and budgets
are unchanged. Three adapter tests and the real Chromium dashboard test pass,
including pending/disabled state, promise rejection, retry, text escaping and
preservation of the measured input. Browser control tests use a controlled
measurement promise; they do not claim a passing performance budget.

The browser test exposed lost JavaScript Error.message details in interop Await:
JSON serialization produced `{}` for non-enumerable messages. The framework now
reads the message data descriptor first, with a regression on the real rejection
path. The full Wasm interop package passes; the existing blank-message fallback
test remains unchanged and passing.

The chat browser fixture requested fallback stub providers but inherited real
credentials, which take precedence. Stubbed test launches now explicitly clear
provider credentials and set the canonical stub option, preventing external paid
calls. The authenticated happy-path journey subsequently passed in 23.091s.
Historical failures remain in the diagnostic logs; do not count them as passes.

## Checkpoint 4 — deferred input plus immediate navigation

Extended the real-browser dashboard regression to filter, clear, and immediately
advance the row window without a compensating delay. This exposed an unsafe
dirty-owner coalescing shortcut: once traversal has started, suppressing schedule
accounting can omit the required follow-up render. State, granular, and ordinary
fiber paths now coalesce early only before traversal begins. Six unit subcases
cover scheduled versus in-flight writes across the three paths.

Validation: unchanged dashboard regression passed three consecutive isolated
Wasm builds; native runtime passed a three-repeat run and a 20-repeat stress run;
full Wasm runtime and interop packages passed; native race runtime and interop
packages passed. One initial broad native runtime attempt failed before retained
diagnostic runs, but subsequent attempts did not reproduce it; its exact failing
test was not retained in the truncated output. Do not describe that attempt as
passing. Temporary diagnostic prints have been removed.

Browser-use inspection verified the actual dashboard layout, filtering, keyboard
clear and immediate advance to rows 0040–0079. The browser tool's empty-string
fill operation left text unchanged in manual inspection; real keyboard input and
the independent Playwright fill/advance regression verified the application.

The chat authentication matrix passed in 26.766s. These are scoped results, not
completion of this migration queue or a passing full example/browser suite.

## Checkpoint 5 — Atlas reproducibility and lane navigation

Added an optional ATLAS_DB_PATH override and a configuration regression; browser
launches now use a test-temporary database instead of the developer's working
database. Refreshed the Atlas Wasm client and verified recovery/not-found routes.
The buyer/operator journeys exposed a stale heading assertion: the existing GWC
view calls its lane roster "On hand by hub". Updated the expected heading and
added a table-scoped assertion that frame-desk lane links actually exist.

Validation: focused configuration tests pass, recovery browser tests pass, and
all selected Atlas buyer/operator journeys pass (46.257s). This does not certify
the entire browser suite. Next: replace the server-interactive POC's manually
constructed markup and browser DOM replacement with GWC-owned views.
