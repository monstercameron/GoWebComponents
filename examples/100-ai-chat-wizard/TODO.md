# Example 100 Todo

- [x] Prevent newly created threads from being cleared when the first assistant response finishes.
	The root-route reset guard now waits for a resolved public conversation ID before treating `/` as an explicit "new chat" navigation.
- [x] Add diagnostics for unresolved thread-route state after a fresh reply.
	The client now logs one warning when route-sync intentionally defers a root reset because a new conversation has an ID but no public route yet, and it logs when a reply completes before the conversation list can resolve that route.
- [x] Flatten the provider, model, and intelligence controls so they use width more efficiently.
	The control bar now keeps labels and selects on the same row, gives the model picker more horizontal room, and compresses the mobile layout into a tighter two-row grid instead of a tall three-row stack.
- [x] Add hover and press animations to the toolbar selects.
	The provider, model, and intelligence selects now animate on hover, focus, and press, and the native option rows receive styled hover or selected states where the browser honors option styling.
- [x] Add a floating down-arrow when more thread content is available below.
	The thread panel now shows a clickable jump-to-bottom button only when the message list has meaningful scrollable space below the viewport, and clicking it smoothly returns the user to the latest messages.
- [x] Rebrand the example-100 product surface away from the framework name.
	The visible app name is now `RelayDesk`, and the server-rendered page title now presents the app as an AI chat workspace instead of a framework lab page.
- [x] Keep provider/model/intelligence selection stable across new chats and reconnects.
	The model picker now repairs blank or invalid persisted selections, persists the recovered model back to the server, and defaults fallback recovery to the first catalog model with `medium` thinking enabled so users do not land in an empty provider/model state.

## Checkpoints

### 2026-03-25 02:31 America/New_York

- completed todo: Define an AI provider companion package pattern for LLM-backed GWC applications.
- files changed: `docs/ECOSYSTEM.md`, `docs/TODO.md`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The documented companion-package boundary matches the live catalog RPC path and the current wasm client surface.
- residual risk: The ecosystem guidance now captures the package boundary, but only RelayDesk currently validates the pattern, so promotion beyond `Experimental` would still require a second production-shaped app.
- next suggested todo: None in the current example-100 provider-switching slice.

### 2026-03-25 02:28 America/New_York

- completed todo: Define the SQL-backed model catalog pattern for runtime provider and model discovery.
- files changed: `examples/100-ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`
- result: Passed. The documented SQL-backed catalog path matches the live startup and RPC behavior exercised by the server tests.
- residual risk: The README now defines the recommended bootstrap payload and freshness policy, but RelayDesk still relies on authenticated RPC revalidation rather than shipping the initial catalog through SSR bootstrap today.
- next suggested todo: Define an AI provider companion package pattern for LLM-backed GWC applications.

### 2026-03-25 02:14 America/New_York

- completed todo: Promote RelayDesk as the reference implementation for runtime AI provider switching.
- files changed: `examples/100-ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `npx playwright test --config=playwright.chat-wizard.config.ts --grep "provider and model selection sync across open tabs"`
- result: Passed. The example README now explicitly positions RelayDesk as the runtime provider-switching reference app and points maintainers to the focused browser regression that proves the shipped flow.
- residual risk: The reference example now documents the shipped switching path clearly, but capability-aware filtering beyond provider membership is still not implemented in the UI.
- next suggested todo: Add capability-aware model filtering and picker messaging so RelayDesk can demonstrate why a provider or model disappears when a workflow requires a missing capability.

### 2026-03-25 01:18 America/New_York

- completed todo: Keep provider/model/intelligence selection stable across new chats and reconnects.
- files changed: `examples/100-ai-chat-wizard/client/app/helpers.go`, `examples/100-ai-chat-wizard/client/app/helpers_wasm_test.go`, `examples/100-ai-chat-wizard/client/app/model_preferences.go`, `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestGetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|TestRPCFallbacksWhenStoreOrProvidersAreUnavailable"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go build ./examples/100-ai-chat-wizard/client/...`
- result: Passed. New-chat and reconnect flows now recover and persist provider/model state instead of leaving the picker blank.
- residual risk: This is covered by unit tests and compile checks, but there is still no browser-level end-to-end regression that exercises the full picker flow through a real page reload.
- next suggested todo: Add a Playwright regression that selects a non-default provider/model, reloads, starts a new chat, and verifies the same provider/model remains selected with a non-blank intelligence mode.

### 2026-03-24 22:12 America/New_York

- completed todo: Prevent newly created threads from being cleared when the first assistant response finishes.
- files changed: `examples/100-ai-chat-wizard/client/app/app.go`, `examples/100-ai-chat-wizard/client/app/route_sync.go`, `examples/100-ai-chat-wizard/client/app/route_sync_test.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/client/app -run TestShouldResetDraftForRootRoute`; `GOOS=js GOARCH=wasm go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The regression guard behaves correctly and the js/wasm app package still compiles.
- residual risk: This flow still lacks a browser-level end-to-end regression that drives a real send on a fresh draft thread.
- next suggested todo: Add an example-100 integration regression that seeds a fake provider response and verifies the browser stays on `/thread/:publicID` after the first reply.

### 2026-03-24 22:14 America/New_York

- completed todo: Add diagnostics for unresolved thread-route state after a fresh reply.
- files changed: `examples/100-ai-chat-wizard/client/app/app.go`, `examples/100-ai-chat-wizard/client/app/stream.go`, `examples/100-ai-chat-wizard/client/app/route_sync.go`, `examples/100-ai-chat-wizard/client/app/route_sync_test.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/client/app -run 'TestShould(ResetDraftForRootRoute|WarnPendingRootRoute)$'`; `GOOS=js GOARCH=wasm go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The warning guard condition is locked in and the js/wasm app package still compiles.
- residual risk: The warnings improve diagnosis but they do not replace a browser-level regression that exercises a real first-message send path.
- next suggested todo: Add an end-to-end regression with a fake provider that verifies the app stays in the new thread after the first streamed reply and asserts the warning does not appear in the healthy path.

### 2026-03-24 22:19 America/New_York

- completed todo: Flatten the provider, model, and intelligence controls so they use width more efficiently.
- files changed: `examples/100-ai-chat-wizard/client/app/panel.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The example-100 client app still compiles for js/wasm after the control-bar layout change.
- residual risk: This is a visual adjustment only; there is still no browser-level layout regression covering narrow widths and the inline control row.
- next suggested todo: Add a Playwright visual/layout smoke for the compact control bar at desktop and mobile widths.

### 2026-03-24 22:24 America/New_York

- completed todo: Add hover and press animations to the toolbar selects.
- files changed: `examples/100-ai-chat-wizard/client/app/panel.go`, `examples/100-ai-chat-wizard/client/app/styles.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The client app still compiles for js/wasm after the animated select styling pass.
- residual risk: Native option hover and press styling remain browser-dependent, so the select element motion is reliable but option-row animation fidelity will vary by platform.
- next suggested todo: Add a browser smoke that exercises the animated control bar in Chromium and confirms hover, focus, and press states remain readable.

### 2026-03-24 22:31 America/New_York

- completed todo: Add a floating down-arrow when more thread content is available below.
- files changed: `examples/100-ai-chat-wizard/client/app/app.go`, `examples/100-ai-chat-wizard/client/app/app_shell.go`, `examples/100-ai-chat-wizard/client/app/constants.go`, `examples/100-ai-chat-wizard/client/app/helpers.go`, `examples/100-ai-chat-wizard/client/app/panel.go`, `examples/100-ai-chat-wizard/client/app/scroll_memory.go`, `examples/100-ai-chat-wizard/client/app/scroll_visibility.go`, `examples/100-ai-chat-wizard/client/app/scroll_visibility_test.go`, `examples/100-ai-chat-wizard/client/app/thread.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/client/app -run TestHasScrollSpaceBelow`; `GOOS=js GOARCH=wasm go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The visibility helper is locked in and the js/wasm client app still compiles with the floating jump-to-bottom control.
- residual risk: Browser-level placement and overlap behavior still need a visual smoke pass, especially with split canvas mode and long threads.
- next suggested todo: Add a Playwright scroll smoke that verifies the button appears when the user scrolls up and jumps back to the latest message when clicked.

### 2026-03-24 22:37 America/New_York

- completed todo: Rebrand the example-100 product surface away from the framework name.
- files changed: `examples/100-ai-chat-wizard/client/app/constants.go`, `examples/100-ai-chat-wizard/server/app/bootstrap.go`, `examples/tests/100-ai-chat-wizard.spec.ts`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The visible app brand and server-rendered page title now use the new product name, and the client app still compiles for js/wasm.
- residual risk: This updates the primary visible branding, but supporting copy such as the empty-state headline still reads like an experiment rather than a polished product surface.
- next suggested todo: Refresh the empty-state and onboarding copy so the rest of the home screen matches the new product branding.
