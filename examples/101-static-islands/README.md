# 101 Static Islands

This example is the current islands-style reference page for the repo.

It prerenders one content-heavy marketing shell and hydrates only two explicit browser roots:

- `#newsletter-island`
- `#quote-island`

The page keeps a visible budget panel for:

- total startup time across both island hydrations
- per-island hydration time
- latest pricing interaction time
- latest quote-card interaction time

Open:

- `/examples/101-static-islands/islands.html`

What this example demonstrates:

- selective activation as explicit multi-root ownership
- static regions that remain inert after first paint
- island-local hydration instead of one full-page browser root
- budget visibility that keeps startup, hydration, and interaction cost measurable in-page
