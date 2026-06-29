# Capability Matrix

> Generated from `docs/capabilities/capabilities.go`. Do not edit by hand; run `CAPABILITIES_WRITE=1 go test ./docs/capabilities/` to regenerate.

| Capability | Package(s) | Example | Chapter |
| --- | --- | --- | --- |
| Components & hooks | `ui` | [counter](../../examples/public/counter/) | [04-ui-rendering-and-hooks.md](04-ui-rendering-and-hooks.md) |
| DOM refs & lifecycle | `ui`, `html` | [dom-ref](../../examples/public/dom-ref/) | [04-ui-rendering-and-hooks.md](04-ui-rendering-and-hooks.md) |
| HTML authoring | `html`, `html/shorthand` | [semantic-html](../../examples/public/semantic-html/) | [05-html-authoring.md](05-html-authoring.md) |
| Typed CSS | `css`, `css/u` | [typed-css](../../examples/public/typed-css/) | [05-html-authoring.md](05-html-authoring.md) |
| Typed theme tokens (gwc css gen) | `css/u` | [typed-css-tokens-demo](../../examples/public/typed-css-tokens-demo/) | [05-html-authoring.md](05-html-authoring.md) |
| Markup nodes (raw HTML & SVG) | `html` | [raw-html](../../examples/public/raw-html/) | [05-html-authoring.md](05-html-authoring.md) |
| Shared state & reactivity | `state` | [state-atoms](../../examples/public/state-atoms/) | [06-state-and-reactivity.md](06-state-and-reactivity.md) |
| Client-side SQLite & durable state | `db/sqlite`, `kvstate` | [sqlite-persistence](../../examples/public/sqlite-persistence/) | [06-state-and-reactivity.md](06-state-and-reactivity.md) |
| Data loading & mutations | `fetch` | [use-resource](../../examples/public/use-resource/) | [07-data-loading-and-mutations.md](07-data-loading-and-mutations.md) |
| Realtime data | `fetch` | [cross-tab-sync](../../examples/public/cross-tab-sync/) | [07-data-loading-and-mutations.md](07-data-loading-and-mutations.md) |
| Routing | `router` | [browser-router](../../examples/public/browser-router/) | [08-routing.md](08-routing.md) |
| SSR & hydration | `ui`, `head` | [hydration](../../examples/public/hydration/) | [09-ssr-and-hydration.md](09-ssr-and-hydration.md) |
| Static islands | `ui` | [static-islands](../../examples/public/static-islands/) | [09-ssr-and-hydration.md](09-ssr-and-hydration.md) |
| Browser interop & workers | `interop` | [browser-interop](../../examples/public/browser-interop/) | [10-browser-interop-and-workers.md](10-browser-interop-and-workers.md) |
| Forms & accessibility | `ui` | [form-accessibility](../../examples/public/form-accessibility/) | [11-forms-accessibility-and-i18n.md](11-forms-accessibility-and-i18n.md) |
| Shared client/server validation | `validate`, `ui` | [shared-form-validation](../../examples/public/shared-form-validation/) | [11-forms-accessibility-and-i18n.md](11-forms-accessibility-and-i18n.md) |
| Internationalization | `i18n` | [locale-switcher](../../examples/public/locale-switcher/) | [11-forms-accessibility-and-i18n.md](11-forms-accessibility-and-i18n.md) |
| Devtools & diagnostics | `devtools` | [devtools-panel](../../examples/public/devtools-panel/) | [12-devtools-testing-and-observability.md](12-devtools-testing-and-observability.md) |
| PWA & offline | `pwa` | [progressive-web-app-offline-cache](../../examples/public/progressive-web-app-offline-cache/) | [13-assets-deployment-and-pwa.md](13-assets-deployment-and-pwa.md) |
| Feature flags | `flags` | [feature-flags](../../examples/public/feature-flags/) | [06-state-and-reactivity.md](06-state-and-reactivity.md) |
| Fine-grained signals | `state` | [fine-grained-signal](../../examples/public/fine-grained-signal/) | [06-state-and-reactivity.md](06-state-and-reactivity.md) |
| Server functions (//gwc:server) | `serverfn` | — | [07-data-loading-and-mutations.md](07-data-loading-and-mutations.md) |
| Query cache & mutations | `query`, `ui` | [use-query](../../examples/public/use-query/) | [07-data-loading-and-mutations.md](07-data-loading-and-mutations.md) |
| Generative UI (agent-native) | `agentui` | [agentui-registry](../../examples/public/agentui-registry/) | [07-data-loading-and-mutations.md](07-data-loading-and-mutations.md) |
| Whole-stack one-binary deploy | `wholestack`, `serverfn` | — | [13-assets-deployment-and-pwa.md](13-assets-deployment-and-pwa.md) |
| Local-first CRDT sync | `localfirst` | [localfirst-crdt](../../examples/public/localfirst-crdt/) | [06-state-and-reactivity.md](06-state-and-reactivity.md) |
| Time-travel devtools | `timetravel` | [timetravel-devpanel](../../examples/public/timetravel-devpanel/) | [12-devtools-testing-and-observability.md](12-devtools-testing-and-observability.md) |
| Animations & FLIP | `anim` | [flip-keyed-list](../../examples/public/flip-keyed-list/) | [04-ui-rendering-and-hooks.md](04-ui-rendering-and-hooks.md) |
| Two-way binding | `html` | [bind-to](../../examples/public/bind-to/) | [05-html-authoring.md](05-html-authoring.md) |
| Named slots | `html/shorthand` | [named-slots](../../examples/public/named-slots/) | [05-html-authoring.md](05-html-authoring.md) |
| Typed search params | `router` | [typed-decode-query](../../examples/public/typed-decode-query/) | [08-routing.md](08-routing.md) |
| Typed i18n message accessors | `i18n` | — | [11-forms-accessibility-and-i18n.md](11-forms-accessibility-and-i18n.md) |
