# Public API Conventions

This document defines the naming and signature rules for every exported symbol in GoWebComponents. All public APIs must conform to these rules. Deviations require documented justification.

---

## Verb-first naming

Every exported function must start with a verb. The verb encodes the **contract** — callers know the lifetime, error behaviour, and cleanup requirements without reading docs.

### Verb tier table

| Verb | Signature contract | Cleanup |
|---|---|---|
| `Use*` | hook context required; no direct error return | none — framework managed |
| `Open*` | returns `(T, error)`; caller is responsible for cleanup | `.Cancel()` or `.Close()` |
| `New*` | returns `T`; purely in-memory; never errors | none |
| `Get*` | returns `(T, error)` or `(T, bool, error)`; read-only; no side effects | none |
| `Build*` | assembles a config/plan value; returns `(T, error)` | none |
| `Parse*` / `Decode*` | reads bytes/string into a typed value; returns `(T, error)` | none |
| `Marshal*` / `Encode*` | encodes a typed value to bytes/string; returns `([]byte, error)` | none |
| `Register*` | installs a listener or service; returns handle, subscription, or error | via returned handle |
| `Schedule*` | enqueues a future execution; returns `(T, error)` with cancel handle | `.Cancel()` |
| `Render*` | SSR functions return `(string, error)`; client DSL functions return `Node` | none |
| `Is*` | returns `bool`; no error return; pure predicate | none |
| `Load*` | returns `(T, bool, error)` -- value may be absent (bool=false); read+cache | none |
| `Inspect*` | returns `(T, error)`; reads/analyzes complex runtime state | none |
| `Observe*` | returns `(T, error)` with cleanup handle; installs a live observer | `.Cancel()` or `.Close()` |
| `Enable*` / `Disable*` | void -- toggles a boolean side-effect flag | none |
| `Wrap*` | returns `T`; adapts an existing value; never errors | none |
| `Apply*` | applies an external payload/snapshot to internal state; returns `error` | none |

**Rule: the verb determines the expected signature. Violating the contract (e.g., an `Open*` that never errors, or a `New*` that returns error) is not allowed.**

---

## DSL exemption

Functions in the `html` and `ui` packages that build element trees are exempt from verb-first naming. These are terse by design because they appear many times per component. The exemption applies to:

- HTML element builders: `Div`, `Span`, `Button`, `Input`, `A`, `P`, etc.
- Combinators: `If`, `IfElse`, `Unless`, `Map`, `MapKeyed`, `FlatMap`, `FilterMap`, `Join`, `Maybe`, `Switch`, `Case`, `Default`, `Fragment`, `WithKey`
- Prop builders: `Class`, `ID`, `For`, `Name`, `Value`, `Style`, `Data`, `Aria`, `OnClick`, `OnInput`, etc.
- UI tree builders (`ui` package): `Fragment`, `Portal`, `Text`, `Lazy`, `AsyncBoundary`, `Overlay`, `HotReloadBoundary`, `AccessibleOverlay`, `ReactiveRegion`

Everything else — including any function outside `html` and `ui` — must use verb-first naming.

---

## Options pattern

All non-hook functions that accept optional configuration must use variadic options:

```go
// correct
func OpenMutationQueue(options ...MutationQueueOptions) (MutationQueue, error)

// wrong — required struct prevents zero-config calls
func NewHistoryRouter(options RouterOptions) *Router
```

Hooks are exempt from this rule. Hooks may take inline parameters if the values are required and named (e.g., `UseAtom[T](id string, initialValue T)`).

---

## Context parameter

Any function that performs I/O, has a timeout, or supports cancellation must accept `ctx context.Context` as its **first parameter**, named exactly `ctx`.

Hooks that need context internally receive it implicitly through the framework. Loader/guard callbacks passed to hooks may accept `context.Context` as their first parameter.

---

## Generic type parameters

Use single-letter descriptive names:

- One type: `[T any]`
- Two types with distinct roles: `[S any, A any]` (state + action), `[T any, U any]` (input + output)
- Constrained types: `[T comparable]`, `[T FastComparable]`

Do not use `I`, `V`, `X` or other ambiguous names.

---

## Error return placement

- `(T, error)` — standard; value is zero when error is non-nil
- `(T, bool, error)` — when absence is distinct from error (e.g., key not found vs storage failure)
- `(T, bool)` — when the operation cannot fail but the value may be absent

Never return a raw `bool` as the sole error signal for a fallible operation.

---

## Boolean parameters

Boolean parameters must be:

1. At the **end** of the parameter list, or
2. In an options struct

Never place a `bool` parameter in the middle of a signature.

---

## Subscription and cleanup

Any long-lived browser resource that requires cleanup must return a type that implements `.Cancel()`:

```go
type Subscription struct { cancel func() }
func (s Subscription) Cancel()
```

Cleanup functions returned from `UseEffect` are the only case where `func()` cleanup is acceptable, and only because the framework calls them, not user code.

---

## Constructor disambiguation

When multiple constructors exist for the same type but create different variants, the variant must be in the name:

```go
NewHashRouter(options ...RouterOptions) *Router     // hash-fragment routing
NewHistoryRouter(options ...RouterOptions) *Router  // HTML5 History API routing
```

A plain `New<Type>` is acceptable only when there is exactly one constructor.

---

## Naming reference: violations corrected in v1

The following renames were applied to bring the codebase into compliance:

### `interop` package

| Old | New | Reason |
|---|---|---|
| `GlobalThis()` | `GetGlobalThis()` | noun-only → `Get*` read |
| `LocalStorage()` | `GetLocalStorage()` | noun-only → `Get*` read |
| `SessionStorage()` | `GetSessionStorage()` | noun-only → `Get*` read |
| `SharedWindowEnv()` | `GetWindowEnv()` | noun-only → `Get*` read |
| `WindowLocation()` | `GetWindowLocation()` | noun-only → `Get*` read |
| `WindowHistory()` | `GetWindowHistory()` | noun-only → `Get*` read |
| `NavigatorClipboard()` | `GetClipboard()` | noun-only → `Get*` read |
| `WindowEvents()` | `GetWindowEvents()` | noun-only → `Get*` read |
| `DocumentEvents()` | `GetDocumentEvents()` | noun-only → `Get*` read |
| `CurrentDocument()` | `GetDocument()` | noun-only → `Get*` read |
| `MatchMedia(query)` | `GetMediaQuery(query)` | noun-only → `Get*` read |
| `SetTimeout(delay, fn)` | `ScheduleTimeout(delay, fn)` | `Set*` is for properties; this schedules |
| `SetInterval(interval, fn)` | `ScheduleInterval(interval, fn)` | same |
| `NewWorker(ctx, opts)` | `OpenWorker(ctx, opts)` | worker holds external resources → `Open*` |
| `NewGoWASMWorker(ctx, opts)` | `OpenGoWASMWorker(ctx, opts)` | same |
| `CurrentWorkerScope()` | `GetWorkerScope()` | noun-only → `Get*` read |
| `WindowOpenerChannel(opts)` | `OpenWindowOpenerChannel(opts)` | acquires channel resource → `Open*` |

### `router` package

| Old | New | Reason |
|---|---|---|
| `NewRouter(options RouterOptions)` | `NewHistoryRouter(options ...RouterOptions)` | disambiguate variant; make variadic |
| `(r *Router) RevalidateCurrentRoute()` | `(r *Router) Revalidate()` | "CurrentRoute" is redundant on router instance |
| `RevalidateCurrentRoute()` | `Revalidate()` | same, package-level alias |
| `(r *Router) IsRouteLoading()` | `(r *Router) IsLoading()` | "Route" is redundant on router instance |
| `Outlet()` | `GetOutlet()` | noun-only → `Get*` read |

### `state` package

| Old | New | Reason |
|---|---|---|
| `ExportSnapshot()` | `GetSnapshot()` | it reads current state → `Get*` |
| `ImportSnapshot(s)` | `ApplySnapshot(s)` | it applies state → descriptive action verb |

### `head` package

| Old | New | Reason |
|---|---|---|
| `Render(document)` | `RenderToString(document)` | `Render` clashes with `ui.Render` (renders to DOM); SSR path returns string |

### `hotreload` package

| Old | New | Reason |
|---|---|---|
| `ExportSnapshot()` | `GetSnapshot()` | it reads current snapshot → `Get*` |
| `ImportSnapshot(payload)` | `ApplySnapshot(payload)` | it restores from snapshot → descriptive action verb |

---

## Signature reference: violations corrected in v1

The following functions had their **return types** corrected to match the verb tier table. Renaming alone was not sufficient — the signatures also violated the contract.

### `interop` package

| Old signature | New signature | Rule violated |
|---|---|---|
| `GetWindowEnv() WindowEnv` | `GetWindowEnv() (WindowEnv, error)` | `Get*` must return `(T, error)` |
| `SharedWindowEnv() WindowEnv` *(compat)* | `SharedWindowEnv() (WindowEnv, error)` | follows `GetWindowEnv` |

**Before:**
```go
env := interop.GetWindowEnv()
selector := env.String("__gwcMountSelector", "#app")
```

**After:**
```go
env, err := interop.GetWindowEnv()
if err != nil {
    // handle unavailable window object
}
selector := env.String("__gwcMountSelector", "#app")
```

When the error is non-fatal (e.g., reading optional config), discard it explicitly:
```go
env, _ := interop.GetWindowEnv()
selector := env.String("__gwcMountSelector", "#app")
```

---

### `state` package

| Old signature | New signature | Rule violated |
|---|---|---|
| `GetSnapshot() Snapshot` | `GetSnapshot() (Snapshot, error)` | `Get*` must return `(T, error)` |
| `ExportSnapshot() Snapshot` *(compat)* | `ExportSnapshot() (Snapshot, error)` | follows `GetSnapshot` |

**Before:**
```go
snapshot := state.GetSnapshot().Select("theme", "user")
```

**After:**
```go
snap, err := state.GetSnapshot()
if err != nil {
    // handle
}
snapshot := snap.Select("theme", "user")
```

When the error is non-fatal, discard it explicitly:
```go
snap, _ := state.GetSnapshot()
snapshot := snap.Select("theme", "user")
```
