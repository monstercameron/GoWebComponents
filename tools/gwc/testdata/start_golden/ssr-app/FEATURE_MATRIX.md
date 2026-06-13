# Starter Feature Matrix

Preset: `ssr-app` (SSR App)

This file is generated from the selected scaffold capabilities.

- available `router`: Route shell and navigation affordances are scaffolded in the starter layout.
- selected `ssr`: Server rendering and hydration expectations are documented in the generated matrix.
- available `forms`: Starter form interactions are scaffolded so teams can extend typed form state deliberately.
- available `fetch`: Async resource ownership is called out for data-loading and mutation setup.
- available `state`: Shared state ownership is planned as a first-class concern in this scaffold.
- available `devtools`: Devtools adoption is surfaced as part of the starter capability model.
- available `hot-reload`: State-preserving local reload is included as the recommended inner-loop path.
- available `browser-tests`: Playwright-Go smoke-test scaffolding is generated under test/playwrightgo.
- selected `hydration`: Client boot and hydration ownership are expected from the first app shell.
- available `release-profile`: Release-minded defaults are encoded in launcher metadata and docs.
- available `dev-profile`: Fast local iteration is pre-wired through gwc dev defaults.
- available `browser-mount`: Direct browser mount behavior is kept explicit in the main entrypoint.
- available `ui`: Component composition is scaffolded through the public ui package.
- available `html`: Typed HTML builders are used as the default authoring path.

Selected capability order:

- `ssr`: SSR
- `hydration`: Hydration

Generated starter output is intentionally disposable; treat this scaffold as a starting point you can edit or replace freely.
