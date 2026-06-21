# 06 State And Reactivity

Use this chapter when you are deciding who should own mutable state, derived values, or resumable client state.

It is the right chapter for:

- choosing between `ui.UseState`, `ui.UseReducer`, context, atoms, and snapshots
- deciding whether a value is local, subtree-scoped, app-shared, or server-owned
- separating `UseComputed`, `UseDerived`, and selector-style projections cleanly
- understanding when `ui.ReactiveRegion(...)` is justified on top of shared state

Use another chapter instead when:

- you need hook mechanics and component composition first: go to [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
- you need async reads, cache ownership, optimistic UI, or offline replay: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need route-owned data, params, or loader revalidation: go to [08 Routing](08-routing.md)
- you need SSR bootstrap ownership or hydration restore in depth: go to [09 SSR And Hydration](09-ssr-and-hydration.md)

## Overview

The state model in GoWebComponents is intentionally layered.

Start with the smallest owner that matches the real problem:

- `ui.UseState`: one component owns the value
- `ui.UseReducer`: one feature subtree owns several named transitions
- `ui.CreateContext` and `ui.UseContext`: one subtree needs shared access without prop threading
- `state.UseAtom`: unrelated components need the same shared source of truth
- `state.UseComputed`: the current component wants a typed render-time derived value
- `state.UseDerived`: several components need the same read-only derived shared value
- `state.UseSelector`: one consumer needs a narrower shared projection from a larger atom or derived source
- snapshot helpers: the app intentionally exports, restores, or persists selected client-owned atoms

Keep one ownership rule in mind for the rest of the manual:

- server-backed records belong in route loaders, `fetch.UseResource[T](...)`, or `fetch.UseCachedResource[T](...)`
- client-owned UI coordination belongs in local hooks, reducers, context, atoms, and snapshots

## Stability Note

Core ownership tools are `Stable`:

- `ui.UseState`
- `ui.UseReducer`
- `ui.CreateContext`
- `ui.UseContext`
- `state.UseAtom`
- `state.UseComputed`
- `state.UseDerived`
- `state.GetSnapshot`, `state.ExportSnapshot`, `state.ApplySnapshot`, `state.ImportSnapshot`
- `state.SaveSnapshot`, `state.LoadSnapshot`, `state.RestoreSnapshot`
- `state.SavePersistentSnapshot`, `state.LoadPersistentSnapshot`, `state.RestorePersistentSnapshot`

Important advanced surfaces:

- `state.UseSelector(...)` is a public shared-projection helper and should be treated as an explicit optimization-oriented tool, not the first state primitive you reach for
- `ui.ReactiveRegion(...)` stays an opt-in hot-path optimization layered on top of shared state, not the default reactivity model

## Minimal Example

Start with local ownership first. If one component owns the value, `ui.UseState` plus `state.UseComputed` is enough.

```go
package main

import (
	"fmt"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// renderInvitePlanner keeps both the writable source and the derived summary local to one component.
func renderInvitePlanner() ui.Node {
	storeSeatCount := ui.UseState(3)
	getSeatSummary := state.UseComputed(func() string {
		if storeSeatCount.Get() == 1 {
			return "1 editor seat"
		}
		return fmt.Sprintf("%d editor seats", storeSeatCount.Get())
	}, storeSeatCount.Get())

	handleUserAddSeat := ui.UseEvent(func() {
		storeSeatCount.Update(func(getPreviousSeatCount int) int {
			return getPreviousSeatCount + 1
		})
	})

	return h.Main(
		h.Class("mx-auto max-w-xl space-y-4 p-6"),
		h.H1("Ownership starts local"),
		h.P(getSeatSummary.Get()),
		h.Button(h.Type("button"), h.OnClick(handleUserAddSeat), "Add seat"),
	)
}

// main mounts the local-state example into the browser DOM.
func main() {
	ui.Render(ui.CreateElement(renderInvitePlanner, nil), "#app")
	utils.WaitForever()
}
```

Why this is the right baseline:

- the component owns the writable value
- the derived label stays local with `UseComputed`
- there is no shared-state or persistence machinery before the app actually needs it

## Production-Shaped Example

When a feature grows, widen ownership in steps: reducer for local workflow, context for subtree sharing, and one small atom for app-wide preference reuse.

```go
package workspace

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

var workspaceFilterContext = ui.CreateContext(workspaceFilterState{})

type workspaceFilterState struct {
	Query        string
	ShowAssigned bool
}

type workspaceFilterAction struct {
	Kind  string
	Value string
}

// buildWorkspaceFilterState applies one named workflow transition for the workspace filter model.
func buildWorkspaceFilterState(getState workspaceFilterState, getAction workspaceFilterAction) workspaceFilterState {
	switch getAction.Kind {
	case "set-query":
		getState.Query = getAction.Value
	case "toggle-assigned":
		getState.ShowAssigned = !getState.ShowAssigned
	}
	return getState
}

// getWorkspaceDensityAtom returns one small shared preference atom for unrelated workspace branches.
func getWorkspaceDensityAtom() state.Atom[string] {
	return state.UseAtom("workspace-density", "comfortable")
}

// renderWorkspaceShell widens ownership deliberately instead of pushing every value into one global store.
func renderWorkspaceShell() ui.Node {
	storeFilter := ui.UseReducer(buildWorkspaceFilterState, workspaceFilterState{})
	storeDensity := getWorkspaceDensityAtom()

	handleUserQuery := ui.UseEvent(func(getEvent ui.InputEvent) {
		storeFilter.Dispatch(workspaceFilterAction{
			Kind:  "set-query",
			Value: getEvent.GetValue(),
		})
	})
	handleUserAssignedToggle := ui.UseEvent(func() {
		storeFilter.Dispatch(workspaceFilterAction{Kind: "toggle-assigned"})
	})
	handleUserDensity := ui.UseEvent(func() {
		if storeDensity.Get() == "comfortable" {
			storeDensity.Set("compact")
			return
		}
		storeDensity.Set("comfortable")
	})

	getProvidedWorkspace := ui.CreateElement(workspaceFilterContext.Provider, ui.ContextProviderProps[workspaceFilterState]{
		Value: storeFilter.Get(),
		Child: ui.Fragment(
			ui.CreateElement(renderWorkspaceToolbar, workspaceToolbarProps{
				HandleUserQuery:          handleUserQuery,
				HandleUserAssignedToggle: handleUserAssignedToggle,
				HandleUserDensity:        handleUserDensity,
			}),
			ui.CreateElement(renderWorkspaceSummary, nil),
		),
	})

	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		getProvidedWorkspace,
	)
}

type workspaceToolbarProps struct {
	HandleUserQuery          ui.Handler
	HandleUserAssignedToggle ui.Handler
	HandleUserDensity        ui.Handler
}

// renderWorkspaceToolbar consumes subtree-scoped filter state and one app-shared preference atom.
func renderWorkspaceToolbar(getProps workspaceToolbarProps) ui.Node {
	getFilter := ui.UseContext(workspaceFilterContext)
	storeDensity := getWorkspaceDensityAtom()

	return h.Div(
		h.Class("space-y-3"),
		h.Input(
			h.Type("text"),
			h.Value(getFilter.Query),
			h.OnInput(getProps.HandleUserQuery),
			h.Placeholder("Search workspace"),
		),
		h.Label(
			h.Class("flex items-center gap-3"),
			h.Input(
				h.Type("checkbox"),
				h.Checked(getFilter.ShowAssigned),
				h.OnChange(getProps.HandleUserAssignedToggle),
			),
			h.Span("Show assigned only"),
		),
		h.Button(
			h.Type("button"),
			h.OnClick(getProps.HandleUserDensity),
			h.Textf("Density: %s", storeDensity.Get()),
		),
	)
}

// renderWorkspaceSummary reads the same subtree-scoped filter state without prop threading.
func renderWorkspaceSummary() ui.Node {
	getFilter := ui.UseContext(workspaceFilterContext)

	return h.Div(
		h.Class("rounded-xl bg-slate-50 p-4 text-sm text-slate-700"),
		h.Textf("Query=%q, assignedOnly=%t", getFilter.Query, getFilter.ShowAssigned),
	)
}
```

Why this is the right middle layer:

- reducer state stays local to one feature subtree
- context shares that local model without turning it into app-global state
- the atom stays small and intentional because it represents a true cross-branch preference

## Scale-Up Example

In a larger codebase, separate app-owned shared state, persistence, and hot-path projections into a dedicated package instead of recreating atom IDs ad hoc in feature code.

```go
package appstate

import (
	"context"

	"github.com/monstercameron/GoWebComponents/state"
)

type workspacePrefs struct {
	Density string
	QueueMode string
}

const workspacePrefsID = "workspace-prefs"

// getWorkspacePrefsAtom returns the canonical shared preference source for the authenticated workspace.
func getWorkspacePrefsAtom() state.Atom[workspacePrefs] {
	return state.UseAtom(workspacePrefsID, workspacePrefs{
		Density: "comfortable",
		QueueMode: "mine",
	})
}

// getWorkspaceDensitySource projects one narrow shared value from the wider preference atom.
func getWorkspaceDensitySource() state.Derived[string] {
	getPrefs := getWorkspacePrefsAtom()
	return state.UseSelector("workspace-density", getPrefs, func(getValue workspacePrefs) string {
		return getValue.Density
	})
}

// storeWorkspacePrefsSnapshot persists only the selected preference slice so restore stays intentional.
func storeWorkspacePrefsSnapshot(getCtx context.Context, getUserID string) error {
	getSnapshot, getErr := state.GetSnapshot()
	if getErr != nil {
		return getErr
	}
	return state.SavePersistentSnapshot(getCtx, "prefs:"+getUserID, getSnapshot.Select(workspacePrefsID))
}

// applyWorkspacePrefsSnapshot restores the current user's preference slice before protected UI renders.
func applyWorkspacePrefsSnapshot(getCtx context.Context, getUserID string) (bool, error) {
	return state.RestorePersistentSnapshot(getCtx, "prefs:"+getUserID)
}
```

```go
package dashboard

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

type queueModel struct {
	HotCount int
	Title    string
}

// getQueueModelAtom returns the canonical shared dashboard counter source.
func getQueueModelAtom() state.Atom[queueModel] {
	return state.UseAtom("dashboard-queue-model", queueModel{HotCount: 0, Title: "Unassigned"})
}

// renderHotQueueBadge narrows one hot shared value before isolating the subscribed display region.
func renderHotQueueBadge() ui.Node {
	getHotCount := state.UseSelector("dashboard-queue-hot-count", getQueueModelAtom(), func(getModel queueModel) int {
		return getModel.HotCount
	})

	return h.Span(
		h.Class("inline-flex rounded-full bg-cyan-50 px-3 py-1 text-sm font-semibold text-cyan-900"),
		ui.ReactiveRegion(func() ui.Node {
			return h.Textf("%d waiting", getHotCount.Get())
		}, getHotCount),
	)
}
```

Why this scales:

- one package owns atom IDs and persistence keys
- selectors narrow wider shared models before hot consumers subscribe
- `ReactiveRegion(...)` only appears on measured hot leaves instead of spreading across ordinary feature code

## Choosing The Smallest Owner

Use this escalation order:

1. start with `ui.UseState`
2. widen to `ui.UseReducer` when one action changes several fields together
3. add context when one subtree needs the same local model without prop threading
4. move to atoms only when unrelated branches truly need the same source of truth
5. add selectors or reactive regions only when the shared source or rerender cost is measurably too broad

That order keeps the mental model predictable.

## Computed, Derived, And Selector Rules

Use `state.UseComputed(...)` when:

- the current component already owns the source values
- you want a typed memoized derived value in render
- the value does not need to be shared across unrelated components

Use `state.UseDerived(...)` when:

- several consumers need the same read-only shared derived value
- recomputation should follow explicit source atom IDs
- the derived value deserves its own stable shared identity

Use `state.UseSelector(...)` when:

- the real problem is that a shared source is too broad
- one consumer only needs a smaller projected shared value
- you are preparing a hot path for `ui.ReactiveRegion(...)` or similar narrow subscribed work

Practical rule:

- `UseComputed` is local
- `UseDerived` is shared and read-only
- `UseSelector` is a narrower shared projection layered on top of an existing shared source

## Snapshot And Persistence Rules

Use snapshot helpers intentionally:

- `GetSnapshot` or `ExportSnapshot`: clone the current runtime atoms
- `ApplySnapshot` or `ImportSnapshot`: merge a snapshot back into the runtime
- `Snapshot.Select(...)`: filter to the exact atom keys you actually want to move or persist
- `SaveSnapshot`, `LoadSnapshot`, `RestoreSnapshot`: JSON browser-storage helpers
- `SavePersistentSnapshot`, `LoadPersistentSnapshot`, `RestorePersistentSnapshot`: durable IndexedDB-first persistence helpers

Keep these boundaries clear:

- same-process snapshots preserve exact live Go values for restore
- JSON persistence is only stable for JSON-compatible values unless the app owns a typed codec
- JSON snapshots are written as `gwc.state.snapshot` v1 envelopes; legacy raw snapshot maps still load, while future snapshot versions are rejected
- persisted snapshots should contain client-owned preferences, drafts, or resumable UI slices, not secrets or generic server data

## Workspace Persistence Rules

When the app restores a real user workspace, use one restore order on purpose:

1. request-owned auth and route bootstrap from the server
2. route-owned first-read data and cache seeds
3. client-owned preferences, drafts, and panel state

Keep workspace persistence bounded:

- scope keys by user, tenant, or workspace identity
- purge or rotate persisted keys on sign-out, tenant switch, or privilege narrowing
- keep retention and expiry rules app-owned instead of assuming browser storage is a durable source of truth
- persist reconstructible UI state, not secrets or authoritative server records

## Client-Side SQLite And Durable State

For state that must outlive a reload — drafts, caches, offline records — the framework ships a
**client-side SQLite** database and a **durable reactive-state** layer on top of it.

### `db/sqlite` — SQLite in the browser

`db/sqlite` runs a pure-Go SQLite engine compiled to wasm (no cgo, no server). A database is opened with
`Open` and exposes `Exec`/`Query`/`QueryRow`/`Tx`/`Flush`/`Close`.

```go
db, err := sqlite.Open(ctx, sqlite.Options{
    Name:        "workspace",
    Persistence: sqlite.IndexedDB, // snapshot the image to IndexedDB on Flush, rehydrate on Open
})
```

- **Persistence backends.** `Memory` (lost on reload), `IndexedDB` (the durable v1 backend — the image is
  snapshotted on `Flush`/`Close` and rehydrated on `Open`), and `OPFS` (reserved; falls back to the
  IndexedDB path in v1).
- **Encryption at rest.** Set `Options.Encryptor` to seal the database image before it touches the store.
  `NewPassphraseEncryptor(passphrase, iterations)` derives an AES-256-GCM key with PBKDF2-HMAC-SHA256;
  the key is never stored, a wrong passphrase or tampered image is surfaced (not silently discarded), and
  the salt is minted once per database. nil stores the image unencrypted. See the package threat model in
  `db/sqlite/encryption.go`.

See [sqlite-persistence](../../examples/public/sqlite-persistence/) for a counter that survives reloads.

### `kvstate` — durable reactive state

`kvstate` binds reactive state to a pluggable `PersistenceBackend` (SQLite by default), so atoms persist
and rehydrate without bespoke wiring:

- **Codecs.** JSON or CBOR encoding of stored values.
- **Write strategies.** `Immediate`, `Debounced`, or `OnUnload` — trade write frequency against latency.
- **Conflict resolution.** `LastWriteWins` or `Versioned`, applied on import and cross-tab merge.
- **Cross-tab sync.** A `BroadcastChannel` watch keeps every open tab consistent (see
  [cross-tab-sync](../../examples/public/cross-tab-sync/)).
- **Ingress / egress.** `Export` reads all records out and `Import` writes them back through the conflict
  resolver — the surface for backing up, restoring, or syncing state across apps, domains, and devices
  over any transport you choose. There is nothing domain-specific here: a sync is just an egress on one
  side and an ingress on the other.

## API Family Reference

Use this table before widening ownership.

| API family | Representative APIs | Stability | Use it when | Prefer something else when |
| --- | --- | --- | --- | --- |
| Local state | `ui.UseState` | `Stable` | one component owns a small writable value | several fields must transition together |
| Local workflow | `ui.UseReducer` | `Stable` | one feature subtree owns named transitions | the value is simple enough for one or two independent `UseState` calls |
| Subtree sharing | `ui.CreateContext`, `ui.UseContext` | `Stable` | one subtree needs shared access without prop threading | unrelated branches need the same source of truth |
| Shared writable state | `state.UseAtom` | `Stable` | multiple unrelated consumers need one shared source | the value is local, route-owned, or really async server data |
| Local derived state | `state.UseComputed` | `Stable` | the current component wants a typed memoized derived value | several components need to share the derived result |
| Shared derived state | `state.UseDerived` | `Stable` | a shared read-only value should recompute from explicit source atom IDs | the derivation is only local to one component |
| Shared projected state | `state.UseSelector` | advanced public projection helper | one consumer needs a narrower projection from a wider shared source | the shared source is already small enough or the owner still needs full rerender |
| Snapshot export and restore | `GetSnapshot`, `ApplySnapshot`, `ExportSnapshot`, `ImportSnapshot`, `Snapshot.Select` | `Stable` | you need exact same-process export, restore, or filtered snapshot transfer | the value should stay route-owned or server-owned |
| Browser snapshot persistence | `SaveSnapshot`, `LoadSnapshot`, `RestoreSnapshot`, `SavePersistentSnapshot`, `LoadPersistentSnapshot`, `RestorePersistentSnapshot` | `Stable` | the app intentionally persists resumable client-owned state | the state is unsafe, too large, or not serialization-safe |
| Fine-grained hot path | `ui.ReactiveRegion` with `state.UseSelector`, atoms, or derived sources | explicit optimization surface | one anchored hot leaf should update without rerunning its owner | the normal component rerender is still correct and cheap enough |

## Design Notes And Boundaries

Keep these rules in mind when choosing state ownership:

- start local and widen ownership only when the app shape forces it
- atoms are for shared source of truth, not a generic async cache
- context is for subtree scope, not a hidden app-global registry
- derived values should usually be computed instead of duplicated into second writable stores
- selectors and reactive regions are explicit performance tools, not the default authoring path
- snapshots are for intentional export, restore, or persistence of client-owned state slices
- route loaders and fetch resources still own server-backed read models

## Common Failure Modes

- moving every value into atoms because it feels globally convenient
- copying derived labels, counts, or filters into second writable atoms instead of deriving them
- using context as an unbounded dependency bucket for the whole app
- storing fetched datasets in atoms instead of letting the fetch or router layer own them
- persisting snapshots without filtering them to the safe, intended atom slice
- adding selectors and `ReactiveRegion(...)` before measuring whether the hot path is actually hot
- assuming JSON storage round-trips arbitrary Go values exactly

## Validation

Use the smallest examples that prove the ownership layer you are adopting.

Local state, reducers, and context:

```powershell
go run ./tools/gwc dev -app .\examples\public\use-state\main.go
go run ./tools/gwc dev -app .\examples\public\scaling-local-state-with-use-reducer\main.go
go run ./tools/gwc dev -app .\examples\public\context-api\main.go
```

Shared atoms and derived values:

```powershell
go run ./tools/gwc dev -app .\examples\public\use-atom\main.go
go run ./tools/gwc dev -app .\examples\public\use-computed\main.go
go run ./tools/gwc dev -app .\examples\public\use-derived\main.go
```

Snapshot export and persistence:

```powershell
go run ./tools/gwc dev -app .\examples\public\snapshot-export-import\main.go
go run ./tools/gwc dev -app .\examples\public\snapshot-storage\main.go
```

Fine-grained reactivity design boundary:

```powershell
go run ./tools/gwc dev -app .\examples\public\state-atoms\main.go
```

## Topic Pagination
Topic 6 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [05 HTML Authoring](05-html-authoring.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
