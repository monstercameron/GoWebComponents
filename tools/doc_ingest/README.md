# Docs Ingest Tooling

Location: `tools/doc_ingest/`

This folder contains the implementation assets for the local docs-ingestion experiment. The prose plan stays in `docs/`; this folder holds the executable pieces.

## File Layout

- `schema.sql`: sqlite schema for the local docs-ingestion database
- `seed.ps1`: local seeding script that reads repo-root docs sources and writes generated outputs under `bin/doc_ingest/`

## Usage

Run from repo root:

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\doc_ingest\seed.ps1
```

## Layout Rules

- Keep source docs and planning notes in `docs/`.
- Keep generated sqlite databases and temp seed files under `bin/doc_ingest/`.
- Add future ingestion helpers here rather than back under `docs/`.
