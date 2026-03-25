# Framework Comparisons

This document explains how GoWebComponents compares to several mature UI frameworks that adopters are likely to evaluate alongside it.

Use it when deciding whether the framework's current tradeoffs match your team's constraints, and when you need one honest place that separates intentional design choices from gaps that are still being closed.

## At A Glance

- GoWebComponents compares best when the evaluation rewards first-party ownership of routing, forms, SSR or hydration, browser interop, and a Go-first programming model.
- It compares worst when the evaluation rewards ecosystem depth, hiring pool, starter maturity, and convention-heavy application scaffolding.
- React remains the closest mental-model comparison for components and hooks.
- Solid and Qwik remain the clearest comparisons for fine-grained-first or resumability-centered architectures.
- Blazor is the strongest comparison for teams that want a non-JavaScript primary language plus a heavier official app-platform story.
- Vue and Svelte remain stronger mainstream defaults for teams prioritizing polished convention stacks and broader ecosystem maturity today.

## Quick Comparison Chooser

Compare against React first when:

- the team wants the closest component-and-hook mental model
- the real question is Go-first authoring versus the JavaScript ecosystem leader

Compare against Blazor first when:

- the team wants a non-JavaScript primary language
- enterprise workflow, procurement comfort, and official full-stack conventions matter heavily

Compare against Solid or Qwik first when:

- the evaluation centers on fine-grained reactivity, startup behavior, or resumability
- the team is deciding how much of the framework identity should come from runtime versus compiler or optimizer assumptions

Compare against Vue or Svelte first when:

- the team values starter maturity, convention-heavy workflows, and mainstream DX polish
- the real tradeoff is Go-first explicitness versus a more standardized application platform

Rule of thumb: if the deciding factor is first-party breadth inside one Go-centric repo, GoWebComponents will look stronger here than it will on market maturity alone.

## How To Read This Page

This is not a marketing matrix.

The purpose is to answer three questions clearly:

- where GoWebComponents is intentionally different
- where it aims at a similar problem space but is not yet equally complete
- which gaps are active backlog or policy work rather than hidden surprises

## Feature Scorecard

This scorecard is deliberately blunt.

It normalizes every framework and every capability to the same three-point rubric so the page answers position, not just presence.

The GoWebComponents column below is based on a fresh repo scan across the current public packages and docs, including `ui`, `html`, `state`, `fetch`, `router`, `interop`, `pwa`, `i18n`, `head`, `devtools`, `hotreload`, and the reference map.

The competitor columns were refreshed against official documentation for the normal official story teams typically buy into, not against arbitrary third-party plugin combinations.

Direct comparison set:

- React
- Solid
- Blazor
- Vue
- Svelte
- Qwik

React, Solid, and Blazor are included here as direct first-class comparison columns, not as side notes.

Normalized legend:

- `H3`: High support, mature, central, or first-class in the framework's normal story
- `M2`: Medium support, supported in a normal way but not the framework's defining strength
- `L1`: Low support, absent, ecosystem-led, more manual, more experimental, or materially behind the category leaders

Important reading rule:

- this is an integrated product-surface score, not a popularity score
- React is intentionally penalized where the official answer is "use the broader ecosystem or a surrounding framework"
- GoWebComponents is intentionally rewarded where the same repo owns and documents more of the stack directly

| Category | Capability | GoWebComponents | React | Solid | Blazor | Vue | Svelte | Qwik |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Language | Primary application language can stay non-JS | H3 | L1 | L1 | H3 | L1 | L1 | L1 |
| Authoring | Function-component mental model | H3 | H3 | H3 | L1 | M2 | M2 | M2 |
| Authoring | Hook-style local state and effects as the main model | H3 | H3 | H3 | L1 | L1 | L1 | L1 |
| Authoring | Typed HTML builders instead of templates/JSX | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Authoring | Template- or JSX-first authoring | L1 | H3 | H3 | H3 | H3 | H3 | H3 |
| Authoring | Typed context/provider API in the first-party surface | H3 | H3 | H3 | M2 | M2 | M2 | H3 |
| Runtime | Client-side rendering | H3 | H3 | H3 | H3 | H3 | H3 | H3 |
| Runtime | Component-local async boundary / suspense primitive | H3 | H3 | H3 | L1 | M2 | M2 | H3 |
| Runtime | Transition / deferred rendering primitives | H3 | H3 | H3 | L1 | L1 | L1 | M2 |
| Routing | Router in the normal first-party story | H3 | L1 | M2 | H3 | H3 | H3 | H3 |
| Routing | Route loaders / route-owned data APIs | H3 | L1 | M2 | M2 | H3 | H3 | H3 |
| Routing | Redirects, guards, and route metadata in the official story | H3 | L1 | M2 | H3 | H3 | H3 | H3 |
| SSR | Request-time SSR story | H3 | L1 | H3 | H3 | H3 | H3 | H3 |
| SSR | Hydration | H3 | H3 | H3 | H3 | H3 | H3 | L1 |
| SSR | Streaming SSR direction | L1 | H3 | M2 | M2 | H3 | H3 | H3 |
| SSR | Resumability instead of traditional hydration | L1 | L1 | L1 | L1 | L1 | L1 | H3 |
| State | Shared state in the first-party package surface | H3 | L1 | M2 | H3 | M2 | L1 | M2 |
| State | Fine-grained reactivity as the default model | L1 | L1 | H3 | L1 | L1 | H3 | H3 |
| State | Fine-grained reactivity as an optional direction | H3 | L1 | H3 | L1 | M2 | H3 | H3 |
| State | Snapshot export/import and serialized state transfer helpers | H3 | L1 | L1 | M2 | L1 | L1 | H3 |
| Forms | Progressive forms and explicit form helpers | H3 | L1 | M2 | H3 | M2 | M2 | M2 |
| Forms | Server-side form actions in the official story | M2 | M2 | L1 | H3 | L1 | H3 | H3 |
| UI primitives | Portals / teleports | H3 | H3 | H3 | L1 | H3 | L1 | L1 |
| UI primitives | Overlay stack / focus-managed modal primitives | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| UI primitives | Virtualization / large-list primitives | H3 | L1 | L1 | H3 | L1 | L1 | L1 |
| Accessibility | First-party focus, live-region, and composite-widget helpers | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Interop | Built-in browser interop surfaces | H3 | L1 | L1 | M2 | L1 | L1 | L1 |
| Interop | Web worker integration story | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Interop | Cross-tab / multi-window coordination helpers | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Platform | Head / SEO metadata in the official story | H3 | L1 | M2 | H3 | H3 | H3 | H3 |
| Platform | Internationalization in the official story | H3 | L1 | L1 | H3 | M2 | L1 | L1 |
| Tooling | Hot reload in the documented core workflow | M2 | H3 | H3 | H3 | H3 | H3 | H3 |
| Tooling | In-app devtools package owned by the same repo/ecosystem | M2 | M2 | L1 | H3 | H3 | L1 | L1 |
| Tooling | First-party test utilities in the official story | H3 | M2 | M2 | H3 | M2 | M2 | L1 |
| Platform | PWA / offline integration guidance | M2 | L1 | M2 | M2 | H3 | H3 | M2 |
| Platform | Durable offline write / mutation replay story | M2 | L1 | L1 | L1 | L1 | L1 | L1 |
| Platform | Prerender / static site export in the official story | H3 | L1 | M2 | M2 | H3 | H3 | H3 |
| Observability | Scoped structured logger with level-based API | H3 | L1 | L1 | M2 | L1 | L1 | L1 |
| Observability | Browser-event console attachment (clicks, changes, submits, navigation, visibility, resize, lifecycle) | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Observability | Sensitive-field redaction in interaction logs | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Observability | Structured diagnostic reports with stack-frame classification (app / framework / platform) | H3 | L1 | L1 | M2 | L1 | L1 | L1 |
| Observability | HTTP error response from structured diagnostic report | H3 | L1 | L1 | H3 | L1 | L1 | L1 |
| Observability | SSR render-timing and correlation-ID instrumentation | H3 | L1 | M2 | H3 | M2 | M2 | H3 |
| Observability | Bootstrap payload size metrics (JSON and CBOR paths) | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Observability | Hydration timing, mismatch count, fallback count, and discarded-node count | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Observability | In-process SSR observation subscription (`ObserveSSR`) | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Observability | Devtools runtime diagnostics panel (severity, classification, code, docs, remediation) | H3 | M2 | L1 | H3 | H3 | L1 | L1 |
| Observability | Devtools runtime profiling counters (renders, commits, effects, hot branches, per-fiber timing) | H3 | M2 | L1 | H3 | M2 | L1 | L1 |
| Observability | Devtools log capture with correlation IDs and structured fields | H3 | L1 | L1 | M2 | L1 | L1 | L1 |
| Observability | Devtools snapshot export, comparison, and fingerprinting | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Observability | Multi-client peer inspection (peer state, traffic, failures) | H3 | L1 | L1 | L1 | L1 | L1 | L1 |
| Extensibility | Plugin / companion package extension contract | H3 | L1 | M2 | H3 | H3 | M2 | M2 |
| Compiler | Compiler-first optimization pipeline as default | L1 | L1 | L1 | L1 | L1 | H3 | H3 |
| Docs | Plain-language server integration docs in the main project | H3 | L1 | M2 | H3 | H3 | H3 | H3 |
| Adoption | First-party starter / app-generator maturity | L1 | H3 | M2 | H3 | H3 | H3 | H3 |
| Adoption | Ecosystem depth and third-party package volume | L1 | H3 | M2 | H3 | H3 | H3 | M2 |
| Adoption | Enterprise support / procurement comfort | L1 | H3 | L1 | H3 | M2 | M2 | L1 |
| Adoption | Community reach / hiring pool | L1 | H3 | L1 | H3 | H3 | M2 | L1 |
| Tooling | End-to-end DX coherence in the official story | M2 | M2 | M2 | H3 | H3 | H3 | M2 |
| Adoption | Learning resources / training depth | L1 | H3 | L1 | H3 | H3 | H3 | M2 |

## Score Totals

Maximum score is `180` per framework: `168` technical points across 56 capability rows plus `12` market points across 4 market rows.

| Framework | Technical total | Market total | Overall total | Read |
| --- | --- | --- | --- | --- |
| GoWebComponents | 149 / 168 | 5 / 12 | 154 / 180 | Broad first-party surface, weak market maturity |
| Blazor | 114 / 168 | 12 / 12 | 126 / 180 | Strongest enterprise-integrated competitor |
| Vue | 104 / 168 | 11 / 12 | 115 / 180 | Most balanced mainstream official stack |
| Qwik | 105 / 168 | 6 / 12 | 111 / 180 | Strong technical novelty, lighter market weight |
| Svelte | 101 / 168 | 10 / 12 | 111 / 180 | Strong compiler-led product with good DX |
| Solid | 97 / 168 | 5 / 12 | 102 / 180 | Excellent reactive core, smaller surrounding stack |
| React | 87 / 168 | 11 / 12 | 98 / 180 | Market leader, but much of the stack is intentionally delegated |

## Matrix Notes

- `GoWebComponents` is scored against what this repo currently documents and ships publicly, not against speculative future direction.
- The GoWebComponents score is high because the repo now clearly owns routing, loaders, forms, browser interop, accessibility helpers, worker seams, cross-tab coordination, state transfer helpers, diagnostics, virtualization primitives, static export, test utilities, and structured logging in first-party packages and docs.
- `Head / SEO` is now scored H3 for GoWebComponents. The `head` package ships social metadata, Open Graph, Twitter card, alternate links, and router-integrated title management as first-party surface.
- `Plugin / companion package extension contract` is now scored H3 for GoWebComponents. The launcher ships a capability-based JSON I/O plugin contract, pre/post command hooks, policy packs, and trust reporting as first-party documented behavior.
- `Virtualization / large-list primitives` is scored H3 for GoWebComponents and Blazor. React, Vue, Svelte, Solid, and Qwik leave this area to ecosystem libraries.
- `React` lands lower than its market position would suggest because this rubric measures integrated first-party ownership. In real adoption decisions React still wins heavily on ecosystem scale, staffing, and surrounding framework choices.
- `Vue`, `Svelte`, and `Qwik` are scored against their normal framework-plus-official-stack story because that is how teams typically evaluate them in practice.
- `Solid` scores high where its reactive model is the point, but lower where the broader product stack is lighter or less standardized.
- `Blazor` scores extremely well on enterprise and integrated full-stack concerns because Microsoft owns a large amount of the end-to-end story directly.
- `L1` does not always mean "impossible". It usually means the capability is delegated, less mature, less central, or materially weaker than the leaders in that row.
- The four market rows are intentionally narrower than the technical rows. They are there to stop the scorecard from pretending API breadth is the whole market story.

## Practical Reading

If you read the scorecard row by row, the honest summary is:

- GoWebComponents now has a larger first-party product surface than the old matrix showed, especially around routing, browser interop, platform ownership, and diagnostics.
- The framework is strongest when the evaluation criteria reward explicit first-party ownership of the stack rather than reliance on external conventions.
- It is intentionally different on language choice, builder-based UI authoring, and preference for runtime primitives over compiler-first transforms.
- It remains behind the mainstream leaders on enterprise comfort, staffing market, starter maturity, and ecosystem depth.
- Qwik and Solid still own clearer positions around resumability and fine-grained-first reactivity.
- Blazor, Vue, and Svelte remain stronger default answers for teams that want a more standardized or convention-heavy application platform immediately.

### Market Read

- GoWebComponents stands out most where teams want one Go-first surface for rendering, routing, state, SSR or hydration, forms, browser interop, and diagnostics without handing core architecture to a JavaScript app framework.
- The scorecard is favorable to GoWebComponents because it rewards first-party product breadth. That is real, but it is not the same thing as market dominance.
- Blazor is the closest "integrated platform" competitor for enterprise teams that value non-JS authoring and first-party full-stack conventions.
- Vue and Svelte are the most balanced mainstream alternatives if the team values polished developer experience and official-stack coherence more than Go-first architecture.
- React remains the safest hiring and ecosystem bet even though this particular rubric scores it lower on integrated surface area.
- Qwik is technically differentiated enough that teams optimizing hardest for startup and resumability should still evaluate it separately instead of treating it as a variant of the others.

### Quick Reads For React, Solid, And Blazor

- React: closest mental-model match for components and hooks, still the strongest market choice on ecosystem and hiring, but much less self-contained as a single official product surface.
- Solid: strongest current comparison point for fine-grained reactivity, with GoWebComponents still intentionally keeping that model optional rather than making it the whole framework.
- Blazor: strongest comparison point for non-JavaScript primary-language UI and a fuller enterprise app-framework story, while GoWebComponents stays more explicit, Go-first, and less convention-heavy.

## Project Position In One Paragraph

GoWebComponents is a Go plus WebAssembly UI framework with React-style function components, hooks, typed HTML builders, routing, shared state, forms, SSR, hydration, browser interop, accessibility primitives, companion-package-oriented extensibility, first-party virtualization, static export and prerender, test utilities, structured logging, and a surprisingly broad first-party browser platform surface.

It is not yet a full convention-heavy app framework with file-based routing, generated starters, deployment adapters, mature commercial backing, or a large hiring market. The project direction stays explicit and Go-first: ordinary Go source, explicit runtime primitives, documented public packages, and companion packages for areas like head management or plugins rather than hidden transforms or convention-heavy scaffolding.

## Compared To React

### Where it is similar

- function components and hooks are the main authoring model
- local state, effects, reducers, refs, context, and error boundaries follow familiar mental models
- routing, shared state, SSR, hydration, and diagnostics are layered on the same runtime surface rather than split into unrelated paradigms

### Where it is intentionally different

- application code is written in Go, not JavaScript or TypeScript
- typed HTML builders replace JSX as the default authoring surface
- the framework treats Go plus the normal Go toolchain as the default path instead of requiring a Node-first build stack for core app code
- extension pressure is expected to flow through companion packages and explicit hooks, not a large app-framework surface in core

### Where it is not yet as complete

- React's ecosystem depth, third-party library volume, and starter maturity are substantially ahead
- measured browser benchmark numbers in this repo still show React ahead on the current mechanical render-update suite
- React's surrounding toolchain for code splitting, starter apps, and production integrations is more mature today

Current measured browser comparison from [README.md](../README.md):

- core render: GoWebComponents `59 ms`, React `42 ms`
- core update: GoWebComponents `67 ms`, React `33 ms`
- hooks: GoWebComponents `62 ms`, React `33 ms`

### Gaps actively being closed

- stronger adoption and ecosystem documentation through [ADOPTION.md](ADOPTION.md) and [ECOSYSTEM.md](ECOSYSTEM.md)
- explicit companion-package story for integrations such as `head` and the experimental `plugin` host
- continued performance work, including fine-grained reactivity and runtime benchmarks documented in [FINE_GRAINED_REACTIVITY.md](FINE_GRAINED_REACTIVITY.md) and [PERFORMANCE.md](PERFORMANCE.md)

## Compared To Vue

### Where it overlaps

- both target component-driven UI, routing, shared state, and SSR-aware applications
- both aim to make common app patterns approachable rather than only exposing low-level DOM operations

### Where it is intentionally different

- GoWebComponents stays Go-first and builder-based instead of template-first and SFC-first
- the project does not try to replicate the Vue app-framework stack in core
- composition is expressed through normal Go functions and typed props instead of template directives and compile-time template semantics

### Where it is not yet as complete

- Vue has stronger first-party app scaffolding, richer ecosystem conventions, and a more mature official app-framework story
- GoWebComponents does not yet offer a Vue-style polished starter and convention stack

### Gaps actively being closed

- onboarding and workflow guidance are being made more explicit so teams have a documented path even before starter packages exist
- team-scale and adoption-maturity backlog items remain open in [TODO.md](TODO.md)

## Compared To Svelte

### Where it overlaps

- both care about keeping authored UI code direct and readable
- both support SSR and hydration as part of the product story

### Where it is intentionally different

- GoWebComponents does not make a template compiler the default product story
- the default contract is ordinary Go source plus runtime primitives, not a second authored language or syntax layer
- compile-assisted features are explicitly optional and experimental, as documented in [COMPILER_ASSISTED_FEATURES.md](COMPILER_ASSISTED_FEATURES.md)

### Where it is not yet as complete

- Svelte's compiler-driven ergonomics and surrounding ecosystem are more mature for teams that want compile-first authoring
- GoWebComponents does not yet provide the same level of compile-time ergonomics, generated routing, or scaffolded app flow

### Gaps actively being closed

- compiler-assisted work remains experimental and capability-specific rather than abandoned
- the repo is explicitly documenting what compiler experiments are allowed to become and what remains out of bounds

## Compared To Solid

### Where it overlaps

- both value reducing unnecessary broad rerenders
- both are interested in explicit finer-grained update paths for high-frequency UI work

### Where it is intentionally different

- GoWebComponents remains hooks-compatible and fiber-based overall instead of switching to a signal-first default programming model
- finer-grained updates are being introduced as explicit regions and state-driven subscribed paths, not as a wholesale replacement for the main component model
- the project does not want two unrelated correctness models for ordinary components versus compiler- or signal-owned components

### Where it is not yet as complete

- Solid's fine-grained model is more mature and central to its identity than GoWebComponents' current shipped surface
- GoWebComponents still relies primarily on normal component rerender semantics outside the documented fine-grained boundaries

### Gaps actively being closed

- the fine-grained direction is now explicit in [FINE_GRAINED_REACTIVITY.md](FINE_GRAINED_REACTIVITY.md)
- benchmarks, diagnostics, and subscribed-region rules are being developed before any larger expansion of that model

## Compared To Blazor

### Where it overlaps

- both let teams build UI from a non-JavaScript primary language
- both care about SSR, routing, forms, and larger app workflow questions rather than only tiny client widgets

### Where it is intentionally different

- GoWebComponents is Go plus WebAssembly rather than .NET and Razor components
- the default authoring surface is explicit Go code and typed HTML builders, not Razor syntax
- server-owned interactive rendering is not a current core product direction; browser-owned runtime remains the default

### Where it is not yet as complete

- Blazor's broader app-framework and enterprise scaffolding story is further along
- GoWebComponents does not yet provide the same level of first-party full-stack conventions, templates, or hosting integrations

### Gaps actively being closed

- SSR, hydration, deployment, observability, and secure forms are being documented more explicitly so long-lived app paths do not depend on tribal knowledge

## Compared To Qwik

### Where it overlaps

- both care about startup cost, SSR, hydration or resume behavior, and the shape of production delivery rather than only client-rendered demos

### Where it is intentionally different

- GoWebComponents does not currently claim a Qwik-style resumability model
- hydration in this repo is still hydration: matching DOM is reused, but the runtime does not claim to serialize and resume the full interaction graph the way Qwik's model aims to
- the default product story is explicit runtime behavior and ordinary Go code, not a resumability-first optimizer pipeline

### Where it is not yet as complete

- Qwik's identity is more tightly centered on resumability and optimizer-driven startup tradeoffs
- GoWebComponents does not yet offer a comparable resumability model or surrounding compile pipeline

### Gaps actively being closed

- hydration, bootstrap transfer, and SSR boundaries are being specified more precisely in [HYDRATION.md](HYDRATION.md), [STATE_TRANSFER.md](STATE_TRANSFER.md), and [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- startup, build, and release tradeoffs continue to be measured in [BUILD_EXPERIMENTS.md](BUILD_EXPERIMENTS.md) and [WASM_RELEASES.md](WASM_RELEASES.md)

## What This Means For Adopters

GoWebComponents is the strongest fit today when all of these are true:

- your team wants Go as the main application language
- explicit runtime primitives are preferable to mandatory compiler magic
- the current routing, state, SSR, hydration, and diagnostics story is a better fit than waiting for a larger app-framework stack
- you can tolerate a younger ecosystem in exchange for a smaller, more explicit public surface

It is a weaker fit today when all of these are true:

- you need the largest existing ecosystem immediately
- you want a polished first-party app generator and convention stack on day one
- you need production-proven resumability, signal-first semantics, or compiler-first ergonomics as the default model
- you do not want to live near an experimental edge for newer SSR, router-data, or companion-package surfaces

## Current Bottom Line

GoWebComponents is intentionally closest to React in component and hook mental model, intentionally closer to explicit runtime primitives than to Svelte or Qwik's compiler-centered models, and intentionally less app-framework-heavy than Vue or Blazor ecosystems.

The repo now has a much clearer answer for adoption path, ecosystem boundaries, companion packages, SSR, state, routing, and deployment. The remaining honest gaps are starter maturity, comparison depth across more production case studies, and broader ecosystem scale.

## Related Docs

- [ADOPTION.md](ADOPTION.md)
- [API_POLICY.md](API_POLICY.md)
- [ECOSYSTEM.md](ECOSYSTEM.md)
- [FRAMEWORK_SCOPE.md](FRAMEWORK_SCOPE.md)
- [FINE_GRAINED_REACTIVITY.md](FINE_GRAINED_REACTIVITY.md)
- [PERFORMANCE.md](PERFORMANCE.md)
- [HYDRATION.md](HYDRATION.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [README.md](../README.md)