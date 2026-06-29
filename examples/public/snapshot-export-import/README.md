# Snapshot Export / Import

Captures a subset of reactive atoms into a portable **snapshot**, lets you mutate them, then restores
the exact in-memory values — the ingress/egress shape behind cross-app and cross-device state sync.

## What it shows
- `state.UseAtom` for reactive values and capturing a subset into a `state.Snapshot`.
- `state.MarshalSnapshotJSON` to serialize the snapshot for transport or storage (egress).
- Restoring a snapshot back into the live atoms (ingress), so the UI returns to the captured values.

## Run
```
gwc dev examples/public/snapshot-export-import
```
Capture the atoms, change them, then restore — the values snap back to the captured snapshot.
