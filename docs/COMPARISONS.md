# Framework Comparisons

This document explains how GoWebComponents compares to several mature UI frameworks that adopters are likely to evaluate alongside it.

Use it when deciding whether the framework's current tradeoffs match your team's constraints, and when you need one honest place that separates intentional design choices from gaps that are still being closed.

## How To Read This Page

This is not a marketing matrix.

The purpose is to answer three questions clearly:

- where GoWebComponents is intentionally different
- where it aims at a similar problem space but is not yet equally complete
- which gaps are active backlog or policy work rather than hidden surprises

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