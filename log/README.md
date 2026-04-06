# Log

Location: `log/`

This directory is a local log sink kept out of source control.

Current purpose:

- give local tooling or experiments a stable place to write logs without polluting the repo root
- keep log files separate from committed docs and source

Rules:

- contents are disposable
- do not commit generated log files
- prefer `bin/runtime/logs/` for launcher-managed runtime logs when the launcher owns the process lifecycle
