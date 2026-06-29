# Compiler-Assisted Features

This page records the current project direction for compiler-assisted features.

Use it when deciding whether GoWebComponents should add compile-time ergonomics, generated helpers, or build-time optimization beyond the normal Go toolchain.

## At A Glance

The project stance is intentionally conservative:

- plain Go plus ordinary `go build` remains the default authoring and shipping path
- compiler work is opt-in, capability-specific, and experimental
- compiler output must stay inspectable, attributable, and reversible
- compiler experiments must lower into the existing runtime model instead of inventing a second default framework model

This document exists to keep experimentation possible without letting the supported product story drift away from Go-first authoring.

## Decision

- Compiler-assisted features are an experimental direction, not a first-class product direction for core.
- The default authoring and shipping model remains plain Go source compiled with the ordinary Go toolchain.
- Compiler work may continue in examples, companion tooling, or opt-in experiments when it improves a narrow workflow without becoming a required step for normal application development.
- Any future compiler-backed feature must justify itself as a specific capability, not as a generic "compiler story."

## Evaluation Lens

When reviewing a proposed compiler-assisted feature, ask these questions first:

- does it solve a specific, narrow problem better than explicit Go code already does?
- does it preserve the existing hook, hydration, scheduling, and runtime ownership model?
- can a user inspect the generated result without proprietary tooling or hidden compiler state?
- can the same workflow still be explained and supported in plain Go when it is part of the main product story?

If the answer to any of those is no, the feature is not ready to move closer to the supported surface.

## What Counts As In Scope

Compiler-assisted work may be explored when it fits one of these buckets:

- syntax sugar or generated helpers that lower repetitive Go UI code into ordinary inspectable Go output
- build-time dead-code elimination or asset shaping that does not change the runtime ownership model
- targeted SSR or release-build optimization that keeps `go build` as the canonical front door
- experimental browser-hosted tooling such as the browser compiler example

## What Does Not Follow From This

- core does not require a template compiler to write components
- core does not require a source transform to use hooks, state, routing, SSR, or hydration
- the browser compiler example is not treated as the primary app-development workflow
- the existence of compiler experiments does not imply a second mandatory mental model beside ordinary Go component authoring

## Why

- The current framework surface is already centered on plain Go components, hooks, SSR, hydration, and runtime primitives.
- The repo already treats the browser compiler as an experimental example outside the core runtime path.
- Ordinary `go build` remains the most explainable, compatible, and debuggable build entrypoint for adopters.
- Treating compiler work as opt-in experimentation avoids forcing every app into generated code paths before the value is proven.

## Immediate Project Stance

- Keep the main product story hydrate-first, hooks-compatible, and Go-toolchain-native.
- Treat compiler work as opt-in and capability-specific.
- Prefer documentation that names the exact experiment under consideration instead of using "compiler" as a catch-all label.

## Safe Experimental Shape

The safest experiments are the ones that stay obviously additive:

- helper generation that reduces repetitive UI code while preserving readable output
- template or syntax lowering that still compiles to attributable Go-owned runtime behavior
- build-time optimization that reshapes artifacts without changing application correctness rules

The unsafe experiments are the ones that silently redefine how components, hooks, or hydration behave depending on a hidden compiler mode.

## Compile-Time Reactivity And Hooks

The current project answer is conservative:

- compile-time reactivity is not a default core direction
- the shipped mental model remains `UseState`, `UseEffect`, explicit state sources, and normal runtime reconciliation
- any compile-time reactivity experiment must compile down to explicit runtime primitives instead of replacing hook semantics with an unrelated authoring model

## Compatibility Rule

Compile-time reactivity is only considered compatible when all of the following stay true:

- ordinary handwritten hook-based components remain first-class and fully supported
- generated output is still inspectable Go or a clearly attributable generated artifact
- hook lifecycle, effect timing, cleanup ordering, and transition semantics do not become compiler-only behavior
- compile-time dependency extraction is limited to explicit regions or generated helpers that can coexist with normal component rerenders
- developers can mix generated and non-generated components without needing two different runtime ownership models

## What Is Out Of Bounds

- a second default authoring model that behaves like Svelte or Solid while ordinary hooks keep different semantics
- hidden dependency extraction that changes `UseEffect` or `UseState` meaning only for compiler-enabled files
- compiler-generated update paths that bypass the runtime's documented scheduling, hydration, or cleanup behavior
- any design that forces adopters to learn whether a component is hook-owned or compiler-owned before they can reason about correctness

## Practical Consequence

If compile-time reactivity is explored, the safe first shape is helper generation or template lowering into the existing mixed-model runtime boundaries:

- explicit subscribed regions
- explicit state reads or selectors
- normal hook-owned structure, effects, routing, and async behavior

That keeps compiler work additive. It does not authorize a split framework where hook components and compiler-reactive components follow different correctness rules.

## Source-Language Boundaries

The default source language boundary is plain Go.

- core APIs are authored and documented as Go packages consumed from handwritten Go code
- any compiler-assisted feature must preserve a plain-Go path for the same capability whenever that capability is part of the main product story
- generated helpers may target Go output when they reduce repetitive boilerplate without hiding the runtime contract

## Allowed Experimental Inputs

The project may experiment with these inputs, but they stay opt-in:

- Go source transforms that emit ordinary Go helpers or generated files
- HTML-like or template-authored experiments that lower into attributable generated Go or equivalent inspectable artifacts
- browser-hosted tooling experiments, including the existing browser compiler example
- release-time or SSR-time optimization passes that operate on build artifacts rather than on a new end-user source language

## Default And Non-Default Paths

The default path remains:

- write components in Go
- build with the ordinary Go toolchain
- debug against readable runtime behavior and inspectable generated output when generation is involved

Non-default paths may exist for experiments, but they must not redefine the default documentation or onboarding story until they prove stable value.

## Boundary Rules

- no required HTML-like template language for normal app development
- no assumption that browser-hosted compilation becomes the standard production workflow
- no compiler feature that only works when users abandon the documented Go package surface
- no generated output format that makes debugging impossible without proprietary tooling or hidden compiler state

In short: experiments may accept multiple input forms, but the supported product boundary remains Go-first and toolchain-native.

## Migration And Fallback Plan

If compiler-generated output ships for any workflow, migration must stay incremental and reversible.

## Required Migration Rules

- generated code paths must be opt-in rather than silently enabled for all builds
- users must be able to inspect generated artifacts directly in the workspace or build output
- the same feature must keep a documented non-generated fallback whenever it is part of the supported product story
- disabling generation must fall back to ordinary Go source and documented runtime primitives without changing application correctness rules
- errors and diagnostics must point to either the original authored input or the generated output with enough metadata to bridge the two

## Debugging Rules

- generated files must be attributable to a specific source input and tool version
- generation should prefer stable file naming or metadata so diffs and bug reports stay understandable
- production failures must remain debuggable without requiring a browser-hosted compiler UI or a hidden remote service
- migration docs should explain how to capture generated output in CI or local builds when teams need reproducible debugging

## Opt-Out Rules

- every compiler-assisted feature must have a documented disable switch or a supported non-generated usage path
- experiments must state whether opt-out happens per file, per package, per build profile, or per app
- onboarding and upgrade docs must not assume compiler output exists unless the user explicitly chose that path

## Rollout Guidance

The safe rollout order is:

- prove value in an experimental example or companion tool
- emit inspectable generated output
- document mixed-mode use with ordinary Go components
- add explicit opt-out and debugging guidance
- only then consider whether the workflow belongs in the supported product surface

This keeps compiler experiments easy to adopt, easy to inspect, and easy to abandon if the tradeoff does not hold up.

## Rollout Checklist

Before promoting any compiler-assisted feature beyond experiment status, verify all of the following:

- the generated path and non-generated path can coexist cleanly
- generated artifacts are attributable to source input and tool version
- diagnostics can bridge between authored input and generated output
- opt-out behavior is documented and practical
- migration and fallback guidance exists before broader rollout

If the feature cannot be inspected, disabled, or debugged locally, it is not ready for supported product messaging.