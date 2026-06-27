# GWC Competitive Analysis — the trilogy

A three-volume strategy to take GoWebComponents to the top of every engineer's list,
researched June 2026 against current stable releases, using the 23-app
[framework-bench](../framework-bench/) as the comparison corpus.

| Vol | Document | Question it answers | Outcome |
|---|---|---|---|
| **I** | [GWC_COMPETITIVE_TEARDOWN.md](./GWC_COMPETITIVE_TEARDOWN.md) | *Where is GWC behind the field?* | **Parity** — 13 ranked dimensions (signals, server fns, routing, data, headless components, DX loop, …) with steal-this + impact×effort |
| **II** | [GWC_DEVX_KILLER_FEATURES.md](./GWC_DEVX_KILLER_FEATURES.md) | *What makes engineers love it day-to-day?* | **Delight** — killer-demo + micro-DX features with API sketches: `//gwc:server` (tRPC w/ zero schema tax), AI-native DevX, `ui.Defer`, optimistic/actions, `gwc workbench`, Elm-grade errors |
| **III** | [GWC_FRONTIER_CATEGORY_MOVES.md](./GWC_FRONTIER_CATEGORY_MOVES.md) | *What category can GWC own that no one else can enter?* | **Moat** — local-first sync engine, zero-npm security product, agent-native/generative UI, one-binary whole-stack, wasm-platform leap |

## The one-sentence strategy

Ship Vol I/II to be **credible**, then bet on the three frontier categories from Vol III
that are exploding in 2026, that GWC is already ~80% built for, and that the entire JS
ecosystem is **structurally barred from entering**:

1. **Local-first sync engine** (F1) — "the year of the sync engine"; GWC has the parts.
2. **Zero-npm as a security product** (F2) — true, free, and catastrophically timely.
3. **Agent-native / generative-UI runtime** (F3) — typed components = safe agent UI.

Keystone prerequisite both build on: **`//gwc:server`** (Vol II A1). Build that first.

> **Don't be a better React. Be the only framework where the compiler checks the whole
> stack, the app is a local-first binary with no npm, and the agents that build it can't
> break it.**
