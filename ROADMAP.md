# GoWebComponents Roadmap

This is the **public** direction for the framework. It is intentionally
theme-level, not a dated commitment: priorities shift with contributor capacity
and real-world feedback from [Discussions](https://github.com/monstercameron/GoWebComponents/discussions).
A feature request that aligns with a theme here is far more likely to be
accepted quickly (see the triage SLA in [CONTRIBUTING.md](CONTRIBUTING.md)).

How to read it: **Now** is actively being worked; **Next** is accepted and
queued; **Later** is directionally agreed but unscheduled; **Exploring** is open
for discussion before commitment.

## Now

- **DevX polish to "god-tier".** Closing the per-feature last-mile gaps tracked
  in `research/devx-maxxing/IMPROVEMENT_PLAN.md` — structured `gwc check --fix`
  remediations, the transitive server-leak analyzer, a blocking vuln gate, the
  bench-drift merge gate, and cross-platform prebuilt `gwc` binaries.
- **Accessibility by default.** Router-managed focus on navigation, the static
  `gwc-a11y` lint, and arrow-key roving in the catalog components.

## Next

- **Local-first / collaboration.** Op-based CRDTs beyond counters — the new
  RGA collaborative-text type (`localfirst.Text`) extended toward richer shared
  documents, plus presence/cursor ergonomics.
- **Offline-durable data.** `fetch.UseDurableMutation` (optimistic cache ↔
  durable queue) extended with automatic background replay/drain and conflict UX.
- **Progressive hydration.** Island budgets and resumption strategies hardened
  with measured startup budgets in CI.

## Later

- **Build experience.** A persistent build daemon keeping the Go build cache hot
  across saves, with published cold/warm timings.
- **Third-party JS interop.** A documented `ImportModule` + typed-bridge pattern
  with native-stub parity, so npm libraries integrate without leaving Go.
- **Streaming agent UI.** `agentui` streamed-and-rendered trees, and the
  component catalog exposed as a first-class MCP tool (the catalog tool shipped;
  streaming is the remainder).

## Exploring

- A JS-free / native renderer seam (the `DOMAdapter` already exists; the bridge
  cost is the open question — see the native-backend research notes).
- A broader `gwc add` registry beyond the core a11y catalog.
- Governance maturation (additional maintainers, a documented RFC process).

---

**Want to influence this?** Open an *Ideas* post in
[Discussions](https://github.com/monstercameron/GoWebComponents/discussions/categories/ideas).
Accepted ideas are linked from the relevant section above. The label taxonomy
and decision process are in [GOVERNANCE.md](GOVERNANCE.md).
