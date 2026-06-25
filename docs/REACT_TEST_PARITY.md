# React test-suite parity for GoWebComponents

Goal: port every **applicable** test from React's suite (`facebook/react`) to
Go/GWC — adapting for architectural differences, keeping the *spirit* of each
test — and make GWC pass them. This file is the living coverage map.

React's suite has **354** test files. Most are not applicable to a Go/WASM
React-like framework (devtools, react-native, server-dom-webpack/turbopack,
refresh, eslint plugin, JSX transform, ES6 classes, profiler-devtools). The
applicable surface is the core element/reconciler/hooks/SSR behavior.

Legend: ✅ done · 🔶 partial · ⬜ todo · 🚫 N/A (no GWC analog / architectural)

## How tests are ported

GWC's full client runtime is testable natively via the mock DOM adapter (React's
noop-renderer pattern): `runtime.NewRuntime(Config{DOMAdapter: mockdom...})` +
`RenderInto`. With no scheduler configured, state updates flush synchronously, so
re-renders/effects are observable inline. SSR behavior is tested via
`ui.RenderToString`. See `ui/reconciler_native_test.go` and siblings.

## react-dom — SSR / server integration

| React test file | Spirit | Status | GWC test |
|---|---|---|---|
| ReactDOMServerIntegrationUntrustedURL | javascript: URL sanitization | ✅ | ui/url_sanitization_test.go (+ found v3.4.7) |
| ReactDOMServerIntegrationAttributes | attr boolean/nil/numeric/escaping | ✅ | ui/attribute_serialization_test.go |
| ReactDOMServerIntegrationElements | void elements, text escaping | ✅ | ui/void_element_test.go, ui/text_escaping_test.go |
| ReactDOMServerIntegrationBasic | basic SSR render, nested components, nil | ✅ | ui/ssr_components_test.go |
| ReactDOMServerIntegrationFragment | fragment SSR | ✅ | ui/ssr_components_test.go (component->fragment hoist) |
| ReactDOMServerIntegrationSpecialTypes | numbers/bools/null children | 🚫 | Go static typing (ui.Node children) |
| ReactDOMServerIntegrationHooks | hooks under SSR | 🔶 | ui/ssr_hooks_boundary_test.go — ARCHITECTURAL GAP: ui.RenderToString is hook-less; hook/context components render only via the reconciler (RenderInto), which IS tested. String-SSR of hook components errors. Candidate for a dedicated "SSR hooks" feature (transient fiber tree + context propagation + hook server-mode). |
| ReactDOMServerIntegrationRefs | refs inert during SSR | ✅ | ui/ssr_ref_and_context_test.go |
| ReactDOMServerIntegrationNewContext | context under SSR / multi-consumer | ✅ | ui/ssr_ref_and_context_test.go, ui/context_native_test.go |
| ReactDOMServerIntegrationTextarea/Input/Checkbox | controlled inputs SSR | ✅ | ui/ssr_controlled_inputs_test.go (textarea value->content: found+fixed v3.4.9) |
| ReactDOMServerIntegrationSelect | select value -> selected option | ✅ | ui/ssr_select_test.go (found+fixed v3.4.10; match by value/text, optgroup) |
| ReactDOMServerIntegrationRefs | refs under SSR | ⬜ | |
| ReactDOMServerIntegrationNewContext | context under SSR | ⬜ | |
| ReactDOMFizzServer* | streaming SSR | 🔶 | partial (internal/runtime/ssr_stream) |
| ReactDOMServer*Hydration / SelectiveHydration | hydration | 🔶 | partial (existing hydration tests) |

## react (core)

| React test file | Spirit | Status | GWC test |
|---|---|---|---|
| ReactChildren-test | flatten/filter/toArray spirit | ✅ | ui/children_native_test.go (map/only/forEach N/A — GWC has normalize, not the Children API) |
| ReactCreateElement-test | element assembly: type/props/children/key | ✅ | ui/create_element_native_test.go |
| ReactElementClone-test | cloneElement | 🚫 | no GWC cloneElement (immutable element model) |
| ReactCreateRef-test / forwardRef | refs (DOM + value) | ✅ | ui/refs_native_test.go, ui/useref_native_test.go |
| SSR primitive children (number/bool/null) | render primitives as children | 🚫 | Go static typing: html.* children are ui.Node; primitives go via ui.Text |
| onlyChild-test | Children.only | ⬜ | |
| ReactContextValidator / NewContext | context | ✅ | ui/context_native_test.go |
| ReactStartTransition | transitions | ⬜ | (GWC has StartTransition) |
| ReactJSX* / ES6Class / PureComponent / Version / Profiler-devtools | — | 🚫 | no Go analog |

## react-reconciler (spirit-applicable)

| React test file | Spirit | Status | GWC test |
|---|---|---|---|
| ReactHooksWithNoopRenderer | hooks via noop renderer | 🔶 | ui/hooks_state_native_test.go (state/multi/order), ui/hooks_update_native_test.go (functional update, stable setter, set-after-unmount); useReducer is a ui-level construct on useState |
| ReactEffectOrdering | effect run/cleanup order | ⬜ | ui/effects_native_test.go (basic) |
| ReactFragment / ReactTopLevelFragment | fragments | ⬜ | |
| ReactFiberRefs | ref attach/detach, detach-before-attach on remount | ✅ | ui/refs_native_test.go |
| ReactEffectOrdering | deletion cleanup parent->child | ✅ | ui/effect_ordering_native_test.go |
| ReactFragment / ReactTopLevelFragment | fragments hoist; keyed state preserved | ✅ | ui/fragment_native_test.go |
| ReactIncrementalSideEffects | mount/unmount side effects | 🔶 | ui/effects_native_test.go (partial) |
| ReactNewContext / ReactContextPropagation | context propagation | ✅ | ui/context_native_test.go |
| ReactMemo | memoization / bailout | ⬜ | |
| ReactIncrementalUpdates | batched/sequenced updates | ⬜ | |
| ErrorBoundaryReconciliation | error boundary recovery | ✅ | internal/runtime/error_boundary_test.go |
| keyed reconciliation (various) | keyed move/insert/remove/duplicate | ✅ | ui/reconciler_native_test.go, reconciler_duplicate_key_test.go (found v3.4.8) |
| ReactClass* / ReactAct* / ReactExpiration / ReactLazy / Scheduler | — | 🚫 | concurrent/class/act — no analog |

## N/A packages (no GWC analog)

react-devtools-*, react-native-renderer, react-server-dom-{webpack,turbopack,fb},
react-refresh, react-debug-tools, react-is, react-cache, eslint-plugin-react-hooks,
use-sync-external-store, use-subscription, react-test-renderer (RTR-specific),
scheduler internals.

## Bugs found by this effort

- **v3.4.7** — javascript:/vbscript: URL XSS in the element render API (UntrustedURL port)
- **v3.4.8** — duplicate-key reconciler node leak/corruption (adversarial keyed-reconciliation probing)
