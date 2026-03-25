# Tailwind CSS CLI Cache

This directory stores the standalone Tailwind CSS CLI binary that is managed by:

```powershell
go run ./tools/gwc tailwind
```

The launcher caches downloaded release binaries under `third_party/tailwindcss/bin/<version>/`.
