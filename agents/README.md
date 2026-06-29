# Agents

Location: `agents/`

This folder stores repo-local agent and skill instructions used by coding assistants. It is not runtime code and it is not part of the public Go module surface.

## File Layout

- `design.skills`: frontend design guidance for landing pages, components, and visual UI work in this repo

## Rules Of Thumb

- Keep machine-facing assistant instructions here, not in package READMEs.
- Add one file per focused capability or workflow instead of one large mixed instruction file.
- When a skill changes package structure or workflow expectations, update the closest human-facing README as well.
