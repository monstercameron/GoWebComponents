# Framework Scope

GoWebComponents remains a UI framework, not a first-party app framework.

## Decision

- Core owns rendering, hooks, state, fetch, router primitives, SSR, hydration, devtools, and runtime diagnostics.
- App-framework concerns such as file-based routing, deployment adapters, opinionated build conventions, and starter scaffolds stay outside the core runtime surface.
- The recommended path for larger apps is a maintained starter or a sibling package that layers on project conventions without expanding the core runtime API by default.
- Server-owned interactive rendering is not a current product direction for core. If the idea is explored later, it should live as a separate experiment or sibling package instead of being implied by the normal browser-owned runtime.
- Fine-grained reactivity is now an active performance direction for core when it helps avoid unnecessary full-component or page-level rerenders in high-frequency UI paths.
- The framework still stays hooks-compatible and fiber-based overall, but the `state` layer and subscribed render regions may gain narrower update semantics where that meaningfully reduces broad rerender work.
- Adoption should be incremental: start with explicit primitives and clear mixed-model rules rather than replacing the existing component model wholesale.

## Why

- The repo already documents a broad UI framework surface with routing, shared state, SSR, and hydration.
- Pulling app-framework conventions into core would make the public surface larger, harder to stabilize, and more opinionated than the current architecture needs.
- Keeping the boundary explicit lets examples, onboarding, and starter guidance evolve without forcing every consumer to adopt a heavyweight application shell.
- The current state model already narrows updates to subscribed components, but some workflows still pay for larger rerender passes than necessary when only small regions change.
- A measured fine-grained layer can improve dashboards, large filtered lists, editors, and spreadsheet-style views without requiring the whole framework to abandon hooks or reconciliation.
- Making the direction explicit lets benchmarks, devtools, scheduling, and interoperability work converge on a single goal instead of treating reactivity as permanently out of scope.

## What Lives Where

- `core`: rendering, hooks, state, fetch, router primitives, SSR/hydration, devtools, diagnostics
- `docs`: onboarding, workflow, integration, and starter guidance
- `starter/sibling packages`: file-based routing, app templates, deployment adapters, and repo-specific conventions

## Follow-Up Work

- Keep the core API focused on reusable UI/runtime primitives.
- Add production-app guidance in the onboarding and server-integration docs instead of moving those conventions into the runtime.
- Prefer starter variants when a project wants a more opinionated app shell.
- Introduce fine-grained primitives only behind explicit APIs, benchmarks, and diagnostics so the narrower update model stays debuggable.
