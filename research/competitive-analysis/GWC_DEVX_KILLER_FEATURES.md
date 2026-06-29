# GWC — DevX Killer Features (Volume II: "top of every engineer's list")

> Volume I ([teardown](./GWC_COMPETITIVE_TEARDOWN.md)) mapped the strategic gaps.
> Volume II goes **super saiyan**: the *granular, daily-felt* DevX features and
> killer-demo moments that make engineers **evangelize** a framework — and the
> concrete, buildable GWC version of each (with API sketches, wow-factor, and effort).
>
> **Then read [Volume III — The Frontier](./GWC_FRONTIER_CATEGORY_MOVES.md):** the
> category-defining moves (local-first sync, zero-npm security, agent-native UI) that
> JS frameworks structurally can't follow.
>
> Researched 2026-06 against current releases. The lens here is not "what's missing"
> in the abstract — it's **"what would make an engineer say *I'm switching to this*"**
> after 5 minutes, after 1000 hours, and when their AI assistant writes the code.

A framework gets to the top of the list by winning three moments:

1. **The first 5 minutes** — a killer demo that produces a visceral "wait, that's it?"
2. **The 1000th hour** — micro-ergonomics that never make you rage-quit.
3. **The 2026 moment** — *the AI assistant writes it correctly the first time.*

GWC's one-language, typed, statically-analyzable, batteries-included design lets it win
all three *better than anyone* — but almost none of it is built yet. Here's the build list.

Legend: **🤯 wow** (drives evangelism) · **🔧 effort** (S/M/L/XL) · **⚡ asymmetric**
(GWC can do this *better* than the competitor because it's Go-on-both-sides).

---

## Part A — The "first 5 minutes" killer demos

These are the features you put in the README GIF. Each must produce an audible reaction.

### A1 ⚡🤯🤯🤯 — `//gwc:server`: tRPC with **zero schema tax**

**The competitor.** tRPC is one of the most-loved DX tools of the era: end-to-end type
safety, **no codegen**, "calling a procedure is like calling a local function." One
migration reported **89% fewer bugs**. But tRPC *still* makes you write routers,
procedures, and `.input(z.object({...}))` schemas — the "schema tax."

**The GWC version (strictly better).** Same language both sides → the request/response
types are *the same Go struct*. No router, no procedure builder, no input schema, no
codegen, no type drift — because there is no boundary to bridge.

```go
// search.go  — one file, shared by client + server
type SearchReq struct{ Q string; Limit int }
type Hit struct{ Title, URL string }

//gwc:server
func Search(ctx ui.Ctx, req SearchReq) ([]Hit, error) {
    return db.Query[Hit](ctx, "SELECT title,url FROM docs WHERE title LIKE ? LIMIT ?", req.Q, req.Limit)
}

// In a component — it's just a function call. Fully typed. Runs the POST for you.
hits, err := Search(ctx, SearchReq{Q: q.Get(), Limit: 10})
```

The build tags `Search` as server-only, strips its body from the wasm bundle, generates
the client stub + server route, and shares the struct verbatim. **This is the single
most evangelism-worthy demo GWC could ship** — it's tRPC's whole value prop minus the
ceremony that even tRPC can't shed. 🔧 M.

### A2 ⚡🤯🤯 — Typed SQL that runs **offline and online** (Drizzle-in-the-browser)

**The competitor.** Drizzle's beloved trait: **types inferred from schema, no codegen
step**, "type freshness" — types are always current. Prisma even rewrote its engine to
*wasm* in 2026 to chase this.

**The GWC version (nobody has this).** GWC already ships **browser SQLite** (`db/sqlite`,
pure-Go wasm, IndexedDB-durable, encrypted at rest). Add a Drizzle-style **typed query
builder where the same typed query runs in the browser (offline) *or* on the server** —
one query, one type, two runtimes:

```go
// One typed query. Offline in wasm against local SQLite, or server-side. Same code.
rows, _ := db.Select[Invoice](ctx).
    Where(db.Eq("status", "open")).
    OrderBy(db.Desc("due")).
    Limit(20).Run()   // rows is []Invoice, inferred, no codegen
```

"Write a typed SQL query once; it works offline in the browser *and* on the server" is a
demo no JS framework can give — they have no in-browser typed DB story. Pair with the
durable `kvstate` + cross-tab sync you already ship. 🔧 M-L.

### A3 ⚡🤯 — `ui.Defer`: deferrable views that **lazy-load wasm** (turn the size weakness into a feature)

**The competitor.** Angular's **`@defer`** blocks lazy-load on triggers
(viewport / idle / interaction / timer) with `@placeholder`, `@loading`, `@error`
sub-blocks — beloved for shrinking initial bundles + improving LCP.

**The GWC version (solves GWC's #1 objection).** GWC's heaviest critique is wasm size.
`@defer` maps *perfectly*: defer-load **wasm chunks** per view, so the initial payload
carries only above-the-fold interactivity.

```go
ui.Defer(ui.DeferOpts{
    On:          ui.OnViewport,            // or OnIdle / OnInteraction / OnTimer
    Prefetch:    ui.OnIdle,
    Placeholder: skeletonCard(),
    Loading:     spinner(),
    Error:       func(e error) ui.Node { return errCard(e) },
    Content:     func() ui.Node { return ui.CreateElement(HeavyChart, props) },
})
```

This reframes "Go wasm is big" into "GWC streams interactivity on demand." Combine with
per-route wasm splitting (Vol I D4). 🔧 L.

### A4 🤯 — Optimistic UI + Actions as one-liners

**The competitor.** React 19's `useOptimistic` + `useActionState` + `useFormStatus` +
form Actions made optimistic UI and form pending/error states a *one-liner* — a headline
DX win of 2024-2026.

**The GWC version.** Pair with `//gwc:server` (A1) so a form submits to a Go server
function with progressive enhancement (works without wasm, better with it):

```go
opt := ui.UseOptimistic(todos, func(cur []Todo, draft Todo) []Todo { return append(cur, draft) })
action := ui.UseAction(AddTodo)          // AddTodo is a //gwc:server fn
// action.Pending / action.Error are first-class; opt.Value renders instantly, rolls back on error
```

🔧 M (depends on A1).

---

## Part B — The "1000th hour" micro-ergonomics

The touches that decide whether engineers *stay*. Individually small; collectively the
difference between "fine" and "I love this."

| # | Steal from | The micro-feature | GWC sketch | 🔧 |
|---|---|---|---|---|
| B1 | **Svelte `$inspect`** | Log every change of a reactive value, with before/after, in dev only | `ui.Inspect(count)` → console-traces each set with source loc | S |
| B2 | **Svelte snippets / Vue slots / React children-fn** | First-class **named slots** for passing template fragments into components (not just one `Child`) | `ui.Slot`/`props.Slots["header"]`; typed snippet params | M |
| B3 | **Svelte `bind:` / `$bindable`** | Two-way binding sugar for form inputs (kill the `value=`+`onInput=parse` boilerplate) | `h.Bind(state)` on `Input` wires value+onInput+parse | S |
| B4 | **Solid `createResource` + Suspense** | `{#await}`-style async rendering with pending/error built in | `ui.Await(resource, Pending:…, Error:…, Then:…)` (you have `AsyncBoundary` — make it terse) | S-M |
| B5 | **Vue `<script setup>` / auto-import** | Kill import boilerplate for the common hooks/tags via a dot-import or `gwc`-managed prelude | already partly there (shorthand dot-import); extend to `ui`/`state` preludes | S |
| B6 | **TanStack derived/structural sharing** | Derived values that don't re-fire when the *projection* is unchanged | strengthen `UseSelector` equality + memo ergonomics | S |
| B7 | **Solid/Svelte control-flow** | Terse keyed list + show/hide that read like the DOM | you have `Map`/`MapKeyed`/`If` — add `Show`, `Switch/Match`, `Index` | S |
| B8 | **React `useId` / a11y wiring** | Auto-wire `label`↔`input`, `aria-describedby`, focus rings | a11y helpers that auto-generate + link ids | S |
| B9 | **Vite env + glob imports** | `gwc`-managed typed env + asset globbing | `gwc`-generated `env.go` + `assets.go` | S |

Ship 8 of these and the daily-driver experience stops leaking engineers.

---

## Part C — Bug-class eliminators (the "I never debug this again" features)

Features that *delete an entire category of bug*. Engineers remember these forever.

- **C1 ⚡ — Compile-time-eliminated server/client leaks.** `//gwc:server` strips server
  code (and secrets) from the wasm bundle at build time; the `gwc check` analyzer flags
  any server-only import reaching client code. Deletes the "I shipped my DB creds to the
  browser" bug class that plagues RSC's fragile `"use client"`/`"use server"` rules
  (whose inconsistency the research explicitly calls out).
- **C2 — Rules-of-hooks + a11y + exhaustive-deps, statically.** Extend the `gwc check`
  analyzer (the hook-context analyzer already exists) into a full ESLint-grade suite:
  rules-of-hooks, missing `key`, unlabelled inputs, invalid ARIA, missing effect deps,
  unreachable routes. Go's type checker + this analyzer ≈ JSX+TS+ESLint, *without npm*.
- **C3 — Exhaustive route/param/search typing.** Typed routes + typed search params
  (Vol I D4) delete the "undefined param" and "stringly-typed query" bug classes
  TanStack Router is famous for killing.
- **C4 — Snapshot-driven time-travel.** You already have snapshot export/import; wire it
  into devtools as **undo/replay** — repro any bug by scrubbing state history. Cheap
  because the infra exists; almost nobody ships it.

---

## Part D — The meta-DevX (where 2026 is actually decided)

### D1 ⚡🤯🤯🤯 — **Be the framework AI writes correctly** (the 2026 sleeper, GWC's biggest unclaimed moat)

**The shift.** **85% of developers use AI coding tools in 2026.** Onboarding now happens
*inside* Cursor/Copilot/Claude Code. Frameworks are increasingly chosen by *what the AI
produces correct code for*. The doc-side standards have crystallized: **llms.txt /
llms-full.txt / skill.md**, **markdown content negotiation** (≈90% fewer tokens than
HTML), and **auto-hosted MCP servers** so agents query *current* docs mid-task.

**Why GWC is uniquely positioned (and almost nobody else is).** AI writes *more correct*
code when the framework is **typed, explicit, statically analyzable, and has a built-in
correctness checker** — which is exactly GWC's character:
- **Typed Go + no template magic** → less ambiguity for the model than JSX+CSS-in-JS+10
  config files.
- **`gwc check` is an AI guardrail.** It already catches the *exact* mistakes models make
  (we built it to catch a hooks-outside-render bug). Position it as the **lint loop the
  agent runs after every edit** — the framework that *self-corrects AI output*.
- **One language** → the agent reasons about client, server, and DB in one mental model,
  not three runtimes.

**The build list (cheap, enormous leverage):**
1. Auto-generate **`llms.txt` + `llms-full.txt` + `skill.md`** from the reference manual
   + capability matrix (you already have machine-generated docs — `capabilities.go`).
2. Serve **markdown** of every doc page (you author in markdown already).
3. Ship a **`gwc mcp` docs+API server** (you already expose `gwc` as an MCP server for
   tooling — extend it to docs/symbol search/analyzer-as-a-tool).
4. Make **`gwc check --fix`** the canonical "agent post-edit" hook, and publish a
   `CLAUDE.md`/`AGENTS.md` ruleset for the framework.
5. Market the angle explicitly: **"the framework your AI gets right the first time."**

🔧 S-M, 🤯 enormous. This is plausibly the **single highest-ROI "top of every list"
move for 2026**, and GWC's design is *accidentally perfect* for it.

### D2 — Elm-grade error messages

**The competitor.** Elm's famously friendly, actionable compiler errors *made the
language beloved* — and directly inspired Rust's. This is a known evangelism driver.

**The GWC version.** You already have an actionable-panic system (`ACTIONABLE_ERRORS.md`,
the unified panic contract) and the check analyzer. Push *every* framework error to
Elm quality: plain-English cause, source frame, **"did you mean"** suggestions, a copy-
pasteable fix, and a docs anchor. Make the in-page **error overlay** (Vite-grade) the
default dev experience. 🔧 M, 🤯 high — punches way above its weight on perception.

### D3 — `gwc workbench`: built-in Storybook + stories-as-tests

**The competitor.** **Storybook 9 + Vitest browser mode**: develop components in
isolation; **stories double as component tests** in a real browser; lightning HMR.
Component-driven development is a workflow engineers love and currently must bolt on.

**The GWC version.** You already have SSR rendering + `testkit` + browser test lanes.
Ship `gwc workbench`: isolated component dev + a stories format whose stories *are* the
wasm/browser tests. One built-in tool replaces an external dependency. 🔧 M.

### D4 — Zero-config full-stack scaffold + deploy adapters

**The competitor.** SvelteKit/Nuxt: `create` → SSR + routing + data + deploy adapter,
convention over config; one-command deploy to any host.

**The GWC version.** `gwc new` → SSR + client + router + DB + `//gwc:server` wired, plus
**deploy adapters** (`gwc deploy vercel|fly|cloudrun|static`). You already have `start`,
`release`, `deploy` commands — make them turnkey + convention-first. 🔧 M.

### D5 — Migration/codemod tooling

**The competitor.** Frameworks ship codemods for every major version; adoption needs an
on-ramp. GWC already has `gwc import` (HTML/JSX→GWC).
**Steal-this:** extend toward **React/JSX→GWC codemods** so teams can port incrementally.
🔧 L, but a real adoption lever.

---

## Part E — Design patterns engineers now expect (table stakes)

Ship these as documented, first-class patterns or engineers feel the framework is "old":

| Pattern | Steal from | GWC status / move |
|---|---|---|
| **Compound components** | Radix/Ark | Formalize via slots (B2) + context; ship Tabs/Accordion as the reference |
| **Headless hooks** (logic without markup) | React Aria, `use*` libs | Ship `UseDisclosure`, `UseCombobox`, `UseDrag` etc. as logic-only hooks |
| **Suspense-for-data / resource** | Solid/React | `ui.Await` (B4) + `fetch.UseResource` already align |
| **Optimistic mutation** | React 19 / TanStack | A4 + the durable offline queue (you have it — differentiator) |
| **Finite state machines** | XState / Ark UI | Ship a tiny typed FSM (Go) for complex interaction logic; Ark proves machines beat ad-hoc state |
| **Server actions / progressive enhancement** | Remix/RSC | A1 + A4 forms that work without wasm |
| **Streaming + out-of-order Suspense** | React/Qwik | you have streaming SSR — make the API ergonomic |
| **Partial/island hydration + resumability** | Astro/Qwik | you have islands (Vol I D2) — push to default + resume |

---

## Part F — "Top of the list" scorecard: which gap wins which engineer

Mapping the build list to *who adopts because of it* — so prioritization tracks personas.

| If GWC ships… | …the engineer who switches is | Wow |
|---|---|---|
| **`//gwc:server` (A1)** | every full-stack TS dev sick of tRPC/REST/GraphQL boilerplate | 🤯🤯🤯 |
| **AI-native DevX (D1)** | the 85% who code with Cursor/Copilot/Claude and want correct output | 🤯🤯🤯 |
| **Typed offline+online SQL (A2)** | local-first / offline / field-app builders (no JS peer exists) | 🤯🤯 |
| **`ui.Defer` wasm-lazy (A3)** | anyone who dismissed GWC over bundle size | 🤯🤯 |
| **Signals-by-default (Vol I D1)** | perf-sensitive / Solid/Svelte refugees | 🤯🤯 |
| **`gwc add` component registry (Vol I D6)** | teams that need Dialog/Combobox *today* | 🤯🤯 |
| **Stable query layer (Vol I D5)** | every React-Query user | 🤯 |
| **Optimistic/actions (A4)** | product engineers shipping forms | 🤯 |
| **Elm-grade errors (D2)** | everyone, subconsciously (perception) | 🤯 |
| **Workbench (D3)** | design-system / component-library teams | 🤯 |

---

## G — If you do only five things (the super-saiyan shortlist)

Ranked by **(asymmetric advantage × evangelism × achievability)**:

1. **`//gwc:server` — tRPC with zero schema tax (A1).** The demo that wins the room;
   GWC can do it *better than the inventors* because it's one language. Foundation for
   A2/A4/forms.
2. **AI-native DevX (D1).** Cheapest huge win in 2026; GWC's typed+analyzable design is
   accidentally perfect, and `gwc check` is already an AI guardrail. *Market it.*
3. **`gwc add` headless component registry (Vol I D6).** Removes the #1 adoption blocker
   with shadcn's proven distribution model.
4. **Signals-by-default + Stable query layer (Vol I D1/D5).** Pays the architecture debt
   the whole market already paid.
5. **`ui.Defer` + resumability (A3 / Vol I D2).** Converts the bundle-size objection from
   a dealbreaker into a feature.

Do #1 and #2 first. They are where GWC stops being "a Go React" and becomes **the thing
an engineer can't get anywhere else** — which is the only durable way to the top of the list.

---

## Sources

tRPC: [trpc.io](https://trpc.io/) · [painless typesafety](https://colinhacks.com/essays/painless-typesafety) · [tRPC v11 no schema tax](https://sadiqueali.medium.com/trpc-v11-end-to-end-type-safety-without-the-schema-tax-bfe848567efc).
Drizzle/typed SQL: [Drizzle vs Prisma 2026 (Bytebase)](https://www.bytebase.com/blog/drizzle-vs-prisma/) · [type freshness](https://makerkit.dev/blog/tutorials/drizzle-vs-prisma).
@defer / React 19: [Angular deferrable views](https://angular.dev/guide/templates/defer) · [React 19](https://react.dev/blog/2024/12/05/react-19) · [useOptimistic/useFormStatus](https://dev.to/a1guy/react-19-deep-dive-forms-actions-with-useformstate-useformstatus-and-useoptimistic-4kdg).
Svelte micro-DX: [bind:/$bindable](https://svelte.dev/docs/svelte/bind) · [Svelte 5 migration (snippets/$inspect)](https://svelte.dev/docs/svelte/v5-migration-guide).
Validation: [Standard Schema](https://standardschema.dev/json-schema) · [Zod/Valibot/ArkType 2026](https://dev.to/gabrielanhaia/zod-4-vs-valibot-vs-arktype-a-type-system-teardown-4lha).
AI-native DevX: [llms.txt for AI agents (Fern)](https://buildwithfern.com/post/optimizing-api-docs-ai-agents-llms-txt-guide) · [Mintlify AI docs](https://www.mintlify.com/library/best-ai-documentation-tools) · [serving markdown to AI assistants](https://www.deployhq.com/blog/making-your-documentation-ai-friendly-serving-markdown-to-ai-coding-assistants).
Workbench/testing: [Storybook + Vitest](https://storybook.js.org/blog/component-test-with-storybook-and-vitest/).
Errors: [Rust learned errors from Elm (HN)](https://news.ycombinator.com/item?id=39923127).
