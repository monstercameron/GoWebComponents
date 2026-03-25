# State Architecture

This guide defines the current recommended state split for non-trivial GoWebComponents applications.

Use it when an app is growing past a few local hooks and needs a consistent answer for:

- what stays local to one component
- what should move into a reducer
- what should be shared through context
- what should become an atom
- where selectors and derived values fit
- when snapshot export or persistence is appropriate

## The Short Version

Use the smallest ownership model that matches the problem:

- `ui.UseState`: one component owns the value
- `ui.UseReducer`: one component subtree owns a local workflow with named transitions
- `ui.CreateContext` and `ui.UseContext`: one subtree needs a shared dependency without prop threading
- `state.UseAtom`: multiple unrelated consumers need the same shared source of truth
- `state.UseDerived` or `state.Select`: a shared value should be projected or computed from other shared state
- `state.ExportSnapshot` and related helpers: you need export, restore, or persistence for intentionally resumable client state

Do not start with atoms for everything. Start local, then widen ownership only when the app shape forces it.

## 1. Local Hook State

Keep state local with `ui.UseState` when:

- one component owns it
- the value does not need to survive outside that component subtree
- updates are simple and independent

Good examples:

- an input draft
- a selected tab
- a disclosure toggle
- a local loading spinner

Bad examples:

- user preferences needed across unrelated routes
- auth state read by multiple screens
- data that several distant components mutate

## 2. Local Workflow State

Use `ui.UseReducer` when one feature owns several fields that must transition together.

This is the right tool when:

- one user action updates several fields
- the feature has named transitions or workflow rules
- the renderer is starting to accumulate scattered `Set(...)` calls

Recommended shape:

- keep the reducer local to the feature subtree
- wrap it in an app-specific hook
- expose semantic handlers instead of raw `Dispatch(...)` throughout the renderer

See [SCALING_LOCAL_STATE_WITH_USE_REDUCER.md](SCALING_LOCAL_STATE_WITH_USE_REDUCER.md) for the reducer-specific guidance.

## 3. Subtree-Scoped Shared State

Use context when one route, layout, or feature subtree needs a shared dependency but the state does not belong to the entire app.

Good context candidates:

- current locale formatter
- current auth/session hint for one routed shell
- route-local service handles
- feature-local configuration shared across descendants

Avoid using context as a global dumping ground. If unrelated parts of the app must coordinate on one shared value, atoms are usually clearer.

## 4. App-Wide Shared State

Use `state.UseAtom` when multiple components outside one local subtree need the same source of truth.

Good atom candidates:

- current user summary
- selected tenant or workspace id
- UI preferences such as theme or density
- shared client-side coordination state between sibling branches

Keep atoms small and intentional.

Good atom content:

- identifiers
- small preference structs
- small coordination payloads
- shared UI-facing state that many components read

Avoid putting these into atoms by default:

- large fetched collections that already belong in a cache
- server-only policy decisions
- every form field in the app
- arbitrary runtime objects that are hard to serialize or restore

## 5. Derived State And Selectors

Do not duplicate state when a value can be computed from existing shared state.

Use:

- `state.UseDerived` for shared read-only derived atoms keyed by explicit source atom IDs
- `state.Select` for read-only projections of one atom or derived source

Preferred pattern:

- keep the canonical value in one atom
- derive labels, filtered slices, counts, or view-specific projections from that source

This avoids duplicated “primary plus copied secondary” state that can drift out of sync.

## 6. Async Data Versus Shared State

Do not use atoms as a replacement for async resource ownership.

Preferred split:

- `fetch.UseResource[T](...)` or the shared fetch cache owns server-backed data loading
- atoms own small shared UI coordination around that data
- reducers own feature-local workflow state

Example:

- fetched customer record: fetch resource or shared cache
- currently selected customer tab: local `UseState`
- edit workflow status: local `UseReducer`
- app-wide preferred customer view mode: atom

## 7. Snapshots And Persistence

Snapshot helpers are for intentional export and restore of client-owned state.

Use them when you need:

- save and restore of a known set of atom values
- browser storage restore for resumable client preferences
- hot-reload or bootstrap-friendly state transfer

Do not treat snapshots as a universal persistence layer.

Practical rules:

- persist only JSON-shaped values unless you own the codec story
- keep secrets and server-only policy out of persisted snapshots
- prefer selecting only the atoms you actually want to restore

Per-user preference persistence should follow a stricter pattern than generic
"save some atoms" snapshots.

Recommended shape:

- keep user-facing preferences such as theme, density, active model, or
  notification settings in one small app-owned preference atom or snapshot slice
- persist that slice under a key derived from the authenticated identity, such
  as `preferences:<userID>` or an equivalent server-backed record
- restore the preference slice before the first protected workspace render so
  the initial paint does not flash with the wrong defaults and then correct
  itself a moment later
- if identity is not known yet, hold the protected workspace in a pending shell
  until both auth and preference state are resolved instead of rendering
  anonymous defaults speculatively

Preference persistence must also stay isolated across users sharing one browser
profile.

Required rule:

- on sign-out, user switch, or tenant switch, clear the active preference atom
  and stop reading from the previous user's storage key before the next
  workspace render begins
- do not fall back to a generic last-used preference snapshot when the next
  authenticated identity differs from the one that wrote the existing data
- if the application migrates or repartitions preference keys, perform that work
  against the current authenticated identity only

The browser profile may be shared, but the restored preference state must still
behave as if each authenticated user had a separate namespace.

This is the preferred job for `state.ExportSnapshot().Select(...)`,
`state.SavePersistentSnapshot(...)`, or an equivalent application-owned typed
codec layered on top of those helpers.

Related APIs:

- `state.ExportSnapshot()`
- `state.ImportSnapshot(...)`
- `state.SaveSnapshot(...)`
- `state.LoadSnapshot(...)`
- `state.SavePersistentSnapshot(...)`
- `state.LoadPersistentSnapshot(...)`

## 8. Recommended Ownership Split

For a typical medium-size app, the state layers should usually look like this:

- local UI toggles, input drafts, temporary selections: `ui.UseState`
- feature workflow and local transition rules: `ui.UseReducer`
- route-shell or layout-scoped dependencies: context
- cross-route shared preferences and app coordination: atoms
- projected shared values: selectors or derived atoms
- server-backed records and collections: fetch resources or shared cache
- resumable client-owned values: selected snapshots and persistence helpers

## 9. Common Failure Modes

Watch for these mistakes:

- using atoms for every local field
- copying derived values into second atoms instead of deriving them
- storing fetched datasets in atoms when the fetch/cache layer should own them
- using context as a hidden app-wide registry
- persisting values that are not safe or stable to restore

## 10. Example Starting Points

Use these examples when you want a concrete starting point:

- local state: `examples/75-use-state`
- reducer scaling: `examples/28-scaling-local-state-with-use-reducer`
- shared atoms: `examples/37-use-atom`
- derived and selector patterns: `examples/39-use-derived`
- snapshot export and import: `examples/40-snapshot-export-import`
- snapshot persistence: `examples/41-snapshot-storage`
- async shared data: `examples/43-use-resource`, `examples/44-use-cached-resource`

## Review Checklist

- is each value owned at the smallest believable scope
- is shared state actually shared, not just globally convenient
- are projected values derived instead of duplicated
- is async server data owned by fetch or cache instead of random atoms
- are persistence and snapshot choices intentional and limited to safe values
