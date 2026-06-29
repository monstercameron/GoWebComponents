# Global Events

A child component owns a **document-level keydown listener** and increments a counter that lives in
the parent — so the count survives the child's unmount/remount and proves the listener is cleaned up.

## What it shows
- `ui.UseGlobalKey(fn)` — a managed `document` keydown listener whose underlying `js.Func` lifetime is
  scoped to the effect (registered on mount, released on unmount).
- Lifting the counter into the parent so toggling the child off and on cannot reset it.
- Leak-safety: if the listener weren't released on unmount, a single keypress after remount would
  double-count. It doesn't.

## Run
```
gwc dev examples/public/global-events
```
Press a key to increment; toggle the listener child off and on and confirm a keypress still counts
exactly once.
