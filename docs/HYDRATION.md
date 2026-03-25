# Hydration

This page records the current shipped hydration contract for `ui.Hydrate(...)` and `router.HydrateMount(...)`.

Use it when you need to know what the runtime will reuse, when it gives up on reuse, and which hydration behaviors are already implemented versus still open backlog work.

## At A Glance

- `ui.Hydrate(...)` is the browser resume path for SSR-rendered DOM.
- `ui.HydrationOptions` can load bootstrap inline, from a bootstrap-reference script, or from an explicit `BootstrapRef`, and also controls strict mismatch behavior plus observability.
- `router.HydrateMount(...)` uses the same underlying DOM-reuse path while layering router startup and route data reuse on top.
- Hydration is reuse-first but not reuse-at-all-costs: structural mismatches fall back per subtree, while text and attribute mismatches warn and let the client commit own the final DOM.

## Quick Resume Path

Use this path by default:

- render HTML on the server with `ui.RenderToString(...)`
- emit bootstrap state only for data the client actually needs during resume
- call `ui.Hydrate(...)` in the browser with `HydrationOptions` when bootstrap, strict mode, or observability matter
- use strict hydration in tests or development when you want mismatches to fail fast instead of quietly degrading to warnings and subtree fallback

## Example Shape

```go
payload, err := ui.ReadBootstrapScript("")
if err != nil {
	return err
}

_, err = ui.Hydrate(App(), "#app", ui.HydrationOptions{
	Bootstrap: payload,
	Strict:    true,
	Observability: ui.SSRObservabilityOptions{
		CorrelationID: "docs-hydration",
	},
})
return err
```

The same public options shape also supports `ScriptID`, `ReferenceScriptID`, and `BootstrapRef` when bootstrap payload discovery should come from the DOM or an external bootstrap reference instead of a pre-read payload.

## Current Scope

- Hydration is a browser resume path over HTML that already exists in the DOM.
- `ui.Hydrate(...)` restores bootstrap data first, then asks the runtime to reuse matching DOM where possible.
- `router.HydrateMount(...)` layers router startup on top of that same resume path.
- The current implementation targets normal host trees, text nodes, route shells, and supported provider or component descendants.

## Fiber To DOM Binding

- The runtime starts each hydration pass with a boundary rooted at the target container's first child.
- Each hydrated fiber asks the boundary for the next non-ignorable DOM candidate.
- Matching host fibers bind directly to that existing DOM node instead of creating a fresh node.
- Matching text fibers bind to the existing text node and stay reusable for later updates.
- Provider, fragment, and function-component fibers do not claim DOM nodes themselves. Their descendants continue matching inside the same parent boundary.

This binding path is implemented in `internal/runtime/hydration.go` through `claimHydrationNode(...)`, `nextHydrationCandidate(...)`, and the per-fiber hydration boundary fields wired during reconciliation.

## Matching Rules

- Whitespace-only text nodes between siblings are ignored during candidate selection.
- Host elements reuse DOM only when the next candidate is an element node with the same tag name.
- Text nodes reuse DOM only when the next candidate is a text node.
- Fragments do not require a dedicated wrapper node. Their children continue matching against the shared boundary.
- Missing candidates, wrong node kinds, or wrong tag names abort reuse for the current subtree boundary.
- Extra trailing DOM nodes under an otherwise successful boundary are removed with a warning once matching completes.
- Comparable SSR-visible attributes are checked after reuse. Mismatches warn through diagnostics, and the client commit still owns the final DOM state.
- Text mismatches also warn through diagnostics, then the reused text node is updated to the client value during commit.

## Effect And Subscription Timing

- Atom subscriptions are queued while hydration is in progress and only attach after the hydrated commit finishes.
- Hydration-time updates triggered before commit are deferred and scheduled after the commit gate lifts.
- Effects run after the committed tree becomes current, not during the DOM-matching walk.
- Event handlers are rebound during the hydrated commit on reused DOM nodes, before effects run.
- The runtime does not replay browser events that happened before hydration finished. Early user input is browser-owned state, not a queued framework event stream.

This is the current shipped answer to "defer effects and subscriptions until hydration completes." The implementation lives in `internal/runtime/hydration.go` and the effect ordering note is reflected in `internal/runtime/reconciler.go`.

## Uncontrolled Form State

- On reused hydrated nodes, the initial resume pass does not overwrite live `value`, `checked`, `selected`, or `autofocus` state.
- This preserves browser-owned draft state more safely when a user typed, toggled, or focused a control before hydration finished.
- Later normal updates after hydration can still write those properties when application state changes.
- Current focus and selection preservation comes from DOM reuse plus not re-running those initial property writes; there is no separate cursor or selection restoration subsystem yet.

This is the current shipped answer to "preserve uncontrolled form state where safe." It is intentionally conservative: preserve the live browser state on resume, then let explicit post-hydration updates take ownership again.

## Complex Subtrees

- Portals render their children inline during SSR, then hydrate those children against the current boundary before committing them into the resolved portal target.
- `ui.AsyncBoundary(...)` hydrates whichever branch is currently rendered by the initial client pass. If the initial branch is fallback content, that fallback hydrates first and later async resolution is a normal update.
- `ui.Lazy(...)` starts its loader from an effect, so lazy work begins after hydration commit instead of racing the DOM matcher.
- Event-heavy components hydrate in place like any other host subtree; handler properties are rebound during commit, not during the DOM scan.
- Current non-goals for this document remain more advanced event replay, automatic partial-hydration scheduling across many roots, and explicit per-subtree hydration prioritization.

## Fallback Behavior

- Hydration fallback is subtree-scoped, not whole-app by default.
- When a boundary cannot safely continue, the runtime removes the unmatched server DOM for that boundary and client-renders that subtree.
- The rest of the page can continue hydrating if its own boundaries are still valid.
- Fallbacks are reported through runtime diagnostics as hydration warnings.
- Hydration fallback is separate from `ui.ErrorBoundary`. DOM continuity failures degrade through subtree client render, not error-boundary fallback UI.

## Mismatch Diagnostics

Current shipped mismatch reporting already covers:

- text mismatches
- tag or structure mismatches that trigger subtree fallback
- critical SSR-visible attribute mismatches
- discarded trailing DOM nodes under an otherwise matched boundary

Those diagnostics now include fiber-path and component-stack context through the runtime diagnostic surface so developers can localize the failing subtree faster.

In normal mode, the recovery behavior is deterministic:

- text and attribute mismatches warn and continue with the client commit owning the final DOM
- structural mismatches abort reuse for the current hydration boundary and fall back to client rendering for that subtree
- unexpected trailing nodes are removed with a warning once the matched boundary finishes

## Strict Hydration Mode

- `ui.HydrationOptions{Strict: true}` turns hydration mismatches into fail-fast errors instead of warning and continuing.
- Strict mode is intended for tests and development runs where a fallback or silent DOM rewrite would hide a real SSR contract bug.
- In strict mode, text mismatches, attribute mismatches, structural mismatches, and unexpected trailing nodes all stop the hydration pass at the first mismatch.
- Strict mode uses the same diagnostic surface, but mismatch entries are emitted as errors instead of warnings before the panic.

## Production Mismatch Behavior

Current production-oriented behavior is:

- valid matching markup should reuse DOM and hydrate normally
- text and attribute mismatches degrade to warnings plus client-owned final DOM updates
- structural mismatches degrade to subtree replacement, not whole-app restart by default
- hydration mismatch recovery is not rendered through `ui.ErrorBoundary` fallback UI
- strict hydration is opt-in for development and test workflows, not the default production mode

This means production behavior today favors continuity and explicit diagnostics over aborting the whole page when one subtree cannot be safely reused.

## Coverage That Exists Today

- `internal/runtime/hydration_test.go` covers simple DOM reuse, tag mismatch fallback, text mismatch recovery, trailing-node cleanup, post-hydration component updates, deferred atom subscriptions, and effect-triggered updates after hydration.
- `internal/runtime/hydration_test.go` also covers mismatch diagnostics with path and component-stack context, plus strict-mode failure behavior for text and structural mismatches.
- `internal/runtime/hydration_test.go` also covers event-handler attachment ordering and live input-value preservation across hydration.
- `router/router_test.go` includes nested route-loader cache reuse coverage for `HydrateMount(...)`.
- `internal/runtime/hydration_benchmark_test.go` includes baseline reuse, mismatch fallback, medium-tree hydration cost, and a first-update-after-hydration benchmark.
- `ui/ui.go` and `internal/runtime/runtime.go` both document the public hydration path as DOM reuse with subtree fallback.

## Still Open

These are still not defined by this document yet:

- route-aware reuse coverage deeper than the current nested loader-cache resume test
- browser-measured interaction timing beyond the current runtime microbenchmarks
- automatic partial hydration orchestration beyond explicit island roots
- explicit hydration-priority policies
