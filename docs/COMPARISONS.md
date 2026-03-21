# Framework Comparisons

This document explains how GoWebComponents compares to several mature UI frameworks that adopters are likely to evaluate alongside it.

Use it when deciding whether the framework's current tradeoffs match your team's constraints, and when you need one honest place that separates intentional design choices from gaps that are still being closed.

## How To Read This Page

This is not a marketing matrix.

The purpose is to answer three questions clearly:

- where GoWebComponents is intentionally different
- where it aims at a similar problem space but is not yet equally complete
- which gaps are active backlog or policy work rather than hidden surprises

## Feature Matrix

This matrix is deliberately blunt.

It compares the major public capabilities that GoWebComponents currently documents against the frameworks this repo already treats as the closest comparison set.

Direct comparison set:

- React
- Solid
- Blazor
- Vue
- Svelte
- Qwik

React, Solid, and Blazor are included here as direct first-class comparison columns, not as side notes.

Status legend:

- `Strong`: mature, central, or first-class in the framework's normal story
- `Present`: supported in a normal way, but not necessarily the framework's defining strength
- `Partial`: possible, but narrower, more manual, more experimental, or more dependent on surrounding choices
- `Weak/External`: mainly delegated to other tools, app frameworks, or community packages rather than the base framework itself
- `No`: not a normal first-class capability in the framework's default product story

| Category | Capability | GoWebComponents | React | Solid | Blazor | Vue | Svelte | Qwik |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Language | Primary application language can stay non-JS | Strong | No | No | Strong | No | No | No |
| Authoring | Function-component mental model | Strong | Strong | Strong | Partial | Present | Present | Present |
| Authoring | Hook-style local state and effects as the main model | Strong | Strong | Strong | Partial | Partial | Partial | Partial |
| Authoring | Typed HTML builders instead of templates/JSX | Strong | No | No | Partial | No | No | No |
| Authoring | Template- or JSX-first authoring | No | Strong | Strong | Strong | Strong | Strong | Strong |
| Runtime | Client-side rendering | Strong | Strong | Strong | Strong | Strong | Strong | Strong |
| Routing | Router in the normal first-party story | Present | Weak/External | Present | Strong | Strong | Present | Strong |
| Routing | Route loaders / route-owned data APIs | Present | Weak/External | Present | Present | Strong | Present | Strong |
| SSR | Request-time SSR story | Present | Weak/External | Strong | Strong | Strong | Strong | Strong |
| SSR | Hydration | Strong | Strong | Strong | Strong | Strong | Strong | Partial |
| SSR | Streaming SSR direction | Partial | Strong | Present | Present | Strong | Strong | Strong |
| SSR | Resumability instead of traditional hydration | No | No | No | No | No | No | Strong |
| State | Shared state in the first-party package surface | Present | Weak/External | Present | Strong | Present | Weak/External | Present |
| State | Fine-grained reactivity as the default model | Partial | No | Strong | No | Partial | Strong | Strong |
| State | Fine-grained reactivity as an optional direction | Strong | Partial | Strong | No | Present | Strong | Strong |
| Forms | Progressive forms and explicit form helpers | Present | Weak/External | Present | Strong | Present | Present | Present |
| UI primitives | Overlays / portals / focus-managed modal primitives | Present | Partial | Present | Strong | Present | Partial | Partial |
| Interop | Built-in browser interop surfaces | Strong | Weak/External | Weak/External | Present | Weak/External | Weak/External | Weak/External |
| Interop | Web worker integration story | Present | Weak/External | Weak/External | Present | Weak/External | Weak/External | Weak/External |
| Interop | Cross-tab / multi-window coordination helpers | Present | Weak/External | Weak/External | Weak/External | Weak/External | Weak/External | Weak/External |
| Tooling | Hot reload in the documented core workflow | Present | Strong | Strong | Strong | Strong | Strong | Strong |
| Tooling | In-app devtools package owned by the same repo | Present | Weak/External | Weak/External | Strong | Present | Weak/External | Weak/External |
| Platform | PWA / offline integration guidance | Present | Weak/External | Present | Present | Strong | Strong | Present |
| Extensibility | Plugin / companion package extension contract | Present | Weak/External | Present | Strong | Strong | Present | Present |
| Compiler | Compiler-first optimization pipeline as default | No | Partial | Partial | Partial | Partial | Strong | Strong |
| Docs | Plain-language server integration docs in the main project | Strong | Weak/External | Present | Strong | Strong | Strong | Strong |
| Adoption | First-party starter / app-generator maturity | Weak/External | Strong | Present | Strong | Strong | Strong | Strong |
| Adoption | Ecosystem depth and third-party package volume | Weak/External | Strong | Present | Strong | Strong | Strong | Present |

## Matrix Notes

- `GoWebComponents` is scored against what this repo currently documents and ships publicly, not against speculative future direction.
- `React` is marked `Weak/External` in areas like routing, loaders, shared state, and forms because those are usually solved with the broader React ecosystem or an app framework such as Next or Remix rather than React core alone.
- `Vue`, `Svelte`, and `Qwik` are scored against their normal framework-plus-official-stack story because that is how teams typically evaluate them in practice.
- `Solid` is scored high on fine-grained reactivity because that is central to its identity, even though surrounding full-stack conventions are less dominant than React or Vue ecosystems.
- `Blazor` is scored high where its full-stack and server or hosting story is stronger than GoWebComponents today, even though its authoring model is intentionally different.
- `GoWebComponents` is intentionally unusual in three cells: Go as the application language, typed HTML builders as a first-class surface, and explicit browser interop documented in the same project rather than delegated almost entirely to community conventions.

## Practical Reading

If you read the matrix row by row, the honest summary is:

- GoWebComponents is competitive today on component model, hooks, routing, SSR or hydration-aware architecture, browser interop, and explicit documentation depth.
- It is intentionally different on language choice, builder-based UI authoring, and the preference for runtime primitives over compiler-first magic.
- It is behind the largest opponent frameworks on starter maturity, ecosystem depth, deployment scaffolding, and polished convention stacks.
- It is not trying to win on resumability-first design or template-compiler ergonomics; those belong more naturally to Qwik and Svelte.
- Its fine-grained reactivity direction is now real, but still not as central or mature as Solid's default model.

### Quick Reads For React, Solid, And Blazor

- React: closest mental-model match for components and hooks, but still well ahead on ecosystem depth, starters, and surrounding production tooling.
- Solid: strongest current comparison point for fine-grained reactivity, with GoWebComponents still intentionally keeping that model optional rather than making it the whole framework.
- Blazor: strongest comparison point for non-JavaScript primary-language UI and a fuller enterprise app-framework story, while GoWebComponents stays more explicit, Go-first, and less convention-heavy.

## Project Position In One Paragraph

GoWebComponents is a Go plus WebAssembly UI framework with React-style function components, hooks, typed HTML builders, routing, shared state, SSR, hydration, and companion-package-oriented extensibility.

It is not a full app framework with file-based routing, generated starters, deployment adapters, or a mandatory compiler step. The project direction stays explicit and Go-first: ordinary Go source, explicit runtime primitives, and documented public packages rather than hidden transforms or convention-heavy scaffolding.

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