# Manual Smoke For Example 100

This is an opt-in headed Playwright smoke pass for the AI chat wizard.

It is intentionally small and human-oriented:

- log in with the seeded demo account
- confirm the WASM app mounts
- inspect seeded conversation history
- open one conversation and verify replayed messages
- open settings and confirm the dialog renders

## Preconditions

From the repo root, make sure the example client can build and the example test dependencies are installed.

```powershell
.\examples\100-ai-chat-wizard\scripts\build-client.ps1
npm --prefix examples install
```

## Run The Manual Smoke

From the `examples/` directory:

```powershell
$env:PLAYWRIGHT_MANUAL_SMOKE = '1'
npx playwright test tests/100-ai-chat-wizard.manual-smoke.spec.ts --config=playwright.chat-wizard.config.ts --headed
```

The existing chat-wizard Playwright config seeds the test database automatically and boots the server on `127.0.0.1:8099`.

## Seeded Credentials

- email: `demo@example.com`
- password: `password123`

## What To Watch For

- login succeeds without redirect loops
- the boot shell disappears and the chat UI becomes interactive
- both seeded sidebar conversations appear
- selecting `Golang Goroutines Explained` shows the saved user and assistant messages
- the settings dialog opens and shows the editable profile controls

## Notes

- This suite is skipped unless `PLAYWRIGHT_MANUAL_SMOKE` is set.
- Keep this smoke pass broad; deterministic assertions belong in the automated spec at `examples/tests/100-ai-chat-wizard.spec.ts`.
