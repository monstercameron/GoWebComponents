# Framework Scope

GoWebComponents remains a UI framework, not a first-party app framework.

## At A Glance

- Core owns reusable UI/runtime primitives: rendering, hooks, state, fetch, router primitives, SSR or hydration, devtools, and diagnostics.
- Core does not aim to become a convention-heavy application platform with file-based routing, deployment adapters, or starter-owned build rules.
- Opinionated app shells should live in starters, sibling packages, or documented workflows rather than expanding the core runtime API by default.
- Fine-grained reactivity is in scope as a performance direction when it improves narrow hot-path updates without replacing the existing component model.
- Server-owned interactive rendering is not part of the current core product direction.

## Quick Scope Test

Keep a capability in core when:

- it is a reusable runtime or UI primitive needed across many app shapes
- it improves correctness, debuggability, or performance without imposing one app structure
- it fits the existing browser-owned runtime model and public package surface

Keep a capability out of core when:

- it mainly encodes project conventions, repo layout, or deployment workflow preferences
- it assumes one preferred starter, hosting model, or file layout
- it is better expressed as documentation, a starter, or a sibling package layered on top of the runtime

Rule of thumb: if a feature mostly tells teams how to structure an app, deploy an app, or scaffold an app, it should usually stay outside core.

## Decision

- Core owns rendering, hooks, state, fetch, router primitives, SSR, hydration, devtools, and runtime diagnostics.
- App-framework concerns such as file-based routing, deployment adapters, opinionated build conventions, and starter scaffolds stay outside the core runtime surface.
- The recommended path for larger apps is a maintained starter or a sibling package that layers on project conventions without expanding the core runtime API by default.
- Server-owned interactive rendering is not a current product direction for core. If the idea is explored later, it should live as a separate experiment or sibling package instead of being implied by the normal browser-owned runtime.
- Fine-grained reactivity is now an active performance direction for core when it helps avoid unnecessary full-component or page-level rerenders in high-frequency UI paths.
- The framework still stays hooks-compatible and fiber-based overall, but the `state` layer and subscribed render regions may gain narrower update semantics where that meaningfully reduces broad rerender work.
- Adoption should be incremental: start with explicit primitives and clear mixed-model rules rather than replacing the existing component model wholesale.
- Fine-grained updates remain opt-in through explicit selectors and subscribed regions. Core should not quietly turn them into the default authoring expectation for ordinary application code.

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

## In Scope Versus Out Of Scope

Clearly in scope for core:

- browser-owned rendering and reconciliation
- local and shared state primitives
- route primitives, loader primitives, and navigation control
- SSR, hydration, diagnostics, and developer tooling tied to the runtime
- measured performance work such as explicit fine-grained subscribed regions

Clearly out of scope for core by default:

- file-based routing conventions
- deployment adapters for specific hosts or platforms
- opinionated starter scaffolds and generated project layouts
- convention-heavy build pipelines that assume one app shape
- server-owned interactive rendering as a normal framework mode

## Follow-Up Work

- Keep the core API focused on reusable UI/runtime primitives.
- Add production-app guidance in the onboarding and server-integration docs instead of moving those conventions into the runtime.
- Prefer starter variants when a project wants a more opinionated app shell.
- Introduce fine-grained primitives only behind explicit APIs, benchmarks, and diagnostics so the narrower update model stays debuggable.

## Review Checklist

Before moving a new capability into core, verify all of the following:

- the feature is a reusable primitive rather than a project convention
- the feature does not force one starter, deployment target, or repo structure
- the public API remains explainable without importing a full app-framework philosophy
- diagnostics and docs can make the new surface understandable without hidden rules
- a starter or sibling package would not solve the need more cleanly
