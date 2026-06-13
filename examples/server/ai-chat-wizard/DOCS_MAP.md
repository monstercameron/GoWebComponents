# Example 100 Docs Map

Use this map when onboarding or debugging example 100.

| Need | Read this file first | Purpose |
|---|---|---|
| Setup + local run | `README.md` | Entrypoint for prerequisites, build commands, server lifecycle, and quick verification. |
| Full systems narrative | `docs/HOW_EXAMPLE_100_WORKS.md` | Chapter-order overview of public delivery, authenticated runtime, chat, admin, data, observability, and extension paths. |
| Public shell architecture | `docs/PUBLIC_ROUTE_DELIVERY.md` | Server shell delivery, boot shell, i18n bootstrap, hydration, and auth-entry transitions. |
| Authenticated shell architecture | `docs/AUTHENTICATED_SHELL.md` | Route ownership, shell composition, preferences, model selection, cross-tab sync, worker usage, and gRPC lifecycle. |
| Chat request lifecycle | `docs/CHAT_REQUEST_LIFECYCLE.md` | Draft/send/server/provider/stream/persist/replay flow plus notable failure branches. |
| Admin/dashboard architecture | `docs/ADMIN_DASHBOARD_SUBSYSTEM.md` | Role scope, shared slice pattern, typed admin RPCs, drill-downs, mutations, and diagnostics. |
| Data architecture | `docs/DATA_LAYER.md` | Schema ownership, query loading, store boundaries, migrations, seed data, and policy placement. |
| Failure and diagnostics model | `docs/OBSERVABILITY_FAILURE_HANDLING.md` | Logs, audit events, support IDs, customer-safe errors, operator diagnostics, and outage behavior. |
| Extension guidance | `docs/EXTENSION_SEAMS.md` | How to add providers, dashboard slices, routes, worker tasks, settings sections, and RPCs. |
| Systems terms | `docs/SYSTEMS_GLOSSARY.md` | Local vocabulary for shell, slice, bootstrap, provider snapshot, control plane, scoped admin view, and public thread routes. |
| Product and runtime flows | `FLOWS.md` | Visit-to-first-chat, admin journey, operational workflows, and bug-fix flow definitions. |
| Manual release gate | `MANUAL_SMOKE.md` | Human smoke checklist + journey-to-test matrix. |
| Bug filing format | `docs/BUG_REPORT_TEMPLATES.md` | Route/first-chat/admin bug templates with required evidence fields. |
| Accessibility release gate | `docs/ACCESSIBILITY_CHECKLIST.md` | Keyboard, landmarks, forms, live regions, motion, contrast, and screen-reader smoke coverage. |
| Performance proof | `docs/PERFORMANCE_PROOF.md` | Local runtime claims, cold-start measurements, first-chat trace, dashboard/sidebar/worker proof, and regression thresholds. |
| Performance and benches | `docs/PERFORMANCE.md` | Benchmark commands, SLA sweep interpretation, and server-side performance notes. |
| Bridge churn hardening | `docs/BRIDGE_CHURN_HARDENING.md` | Bridge state model, RPC policy, best-effort queue, retry/idempotency rules, UX, observability, regressions, and multi-bridge decision. |
| Schema and wiring state | `SCHEMA_TABLES.md` | Table inventory, wiring classification, and integration status. |
| Design direction | `DESIGN.md` | Intended product/design direction and IA notes. |
| Operator lifecycle steps | `OPERATOR_RUNBOOK.md` | Build/seed/start/verify sequence for operators and maintainers. |
| Work log + checkpoints | `CHANGELOG.md` | Timestamped completion checkpoints and validation history. |
| Active backlog | `TODO.md` | Current prioritized work items by agent/topic. |
