# Ready Signal

Fires a deterministic **"app is mounted"** signal after the first commit — both to Go code and to the
host page — so external scripts can wait for a real first frame instead of guessing.

## What it shows
- `ui.OnReady(fn)` — a Go callback that runs once, right after the first commit to the DOM.
- The `gwc:ready` DOM event dispatched at the same point, which any host-page script (or test harness)
  can listen for.

## Run
```
gwc dev examples/public/ready-signal
```
The page logs the Go `OnReady` callback firing and receives the dispatched `gwc:ready` event after the
first render.
