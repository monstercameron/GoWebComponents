# Workspace Persistence

Use this page when a long-lived workspace needs per-user restore of threads,
drafts, canvas state, or scroll position across WASM restart and tab restore.

## Current Decision

Long-lived workspace state should persist per authenticated user, not per bare
browser profile.

Recommended split:

- server-backed storage is the source of truth for durable conversation history
  and any state that must survive device changes or support review
- IndexedDB-first browser storage is the preferred local layer for resumable
  drafts, canvas session state, scroll markers, and recent workspace mirrors
- every durable key should include the authenticated identity and the relevant
  workspace, thread, or canvas id

## Data Classes And Ownership

Use this ownership split:

- conversation history: server-backed when the product needs multi-device
  continuity, auditing, or team-visible history; browser storage may keep only a
  recent mirror for faster resume
- draft messages: browser-durable and per user plus thread so unfinished work
  can survive tab close or restart
- active canvas state: browser-durable per user plus route or document id, with
  app-owned autosave cadence and reset rules
- scroll position: browser-durable per user plus thread or panel id so long
  transcripts can reopen near the last read point

Do not persist secrets, raw auth material, or server-only policy alongside this
workspace state.

## Restore Order

Recommended restore order:

1. resolve auth and determine the active user namespace
2. restore the recent workspace mirror, drafts, canvas state, and scroll
   markers for that user only
3. render the protected workspace shell
4. reconcile browser-restored state with authoritative server history or route
   data as those loads complete

The important rule is identity-first restore. Do not reopen a previous user's
drafts or thread list while auth is still unresolved.

## Retention And Expiry

Browser-resident workspace history should stay bounded.

Recommended defaults:

- keep at most the 50 most recent conversation or workspace entries per user in
  browser-durable mirrors
- expire browser-resident conversation mirrors, drafts, canvas snapshots, and
  scroll markers after 30 days without local activity unless the product owns a
  stricter policy
- allow apps to choose shorter windows for sensitive workspaces or larger limits
  only when the domain can justify the storage and privacy cost

Server-backed history may use a different retention policy, but the browser
layer should still stay bounded and reviewable.

## Sign-Out And User-Switch Purge

Sign-out or user switch must clear the active browser workspace namespace.

That purge should remove:

- browser-resident conversation mirrors
- unsent draft messages
- active canvas or editor session snapshots
- scroll position markers
- any other user-scoped workspace resume metadata

If the product keeps conversation history on the server, sign-out should clear
only the browser copy, not delete authoritative server records.

## Recommended Storage Shape

The preferred browser shape is one user-partitioned durable store with stable
key prefixes such as:

- `workspace:<userID>:threads`
- `workspace:<userID>:draft:<threadID>`
- `workspace:<userID>:canvas:<documentID>`
- `workspace:<userID>:scroll:<threadID>`

Use `state.SavePersistentSnapshot(...)`, `interop.OpenPersistentStore(...)`, or
an application-owned typed codec on top of those APIs when the workspace needs a
clear durable contract instead of ad hoc `localStorage` writes.
