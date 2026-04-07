# Bin

Location: `bin/`

This directory holds ignored local runtime outputs and generated artifacts.

Typical contents:

- managed example server executables and state files under `bin/runtime/examples-servers/`
- launcher-owned runtime logs under `bin/runtime/logs/`
- temporary local outputs from build, verify, and benchmark flows

Rules:

- do not commit files from here unless there is an explicit release reason
- it is usually safe to delete local contents when no launcher-managed process is running
- if a managed example server is live, stop it before deleting its executable or state file

Canonical workflow docs:

- [../tools/README.md](../tools/README.md)
- [../docs/REFERENCE_MANUAL/02-gwc-workflows.md](../docs/REFERENCE_MANUAL/02-gwc-workflows.md)
