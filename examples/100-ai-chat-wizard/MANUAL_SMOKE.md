# Manual Smoke For Example 100

This is an opt-in headed Playwright smoke pass for the AI chat wizard.

It is intentionally small and human-oriented:

- log in with the seeded demo account
- confirm the WASM app mounts
- inspect seeded conversation history
- open one conversation and verify replayed messages
- open settings and confirm the dialog renders

## Preconditions

From the repo root, make sure the example client can build.

```powershell
.\examples\100-ai-chat-wizard\scripts\build-client.ps1
```

## Run The Manual Smoke

From the repo root:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestChatWizard -v
```

## Seeded Credentials

- demo account: `demo@example.com / password123`
- admin dev account: `admin@example.com / password`

These credentials are email-based logins. If you are testing against a manually started server instead of the Playwright-managed one, make sure the server and `cmd/seed-test-db` share the same `CHAT_DB_PATH` as described in `README.md`.

## What To Watch For

- login succeeds without redirect loops
- the boot shell disappears and the chat UI becomes interactive
- both seeded sidebar conversations appear
- selecting `Golang Goroutines Explained` shows the saved user and assistant messages
- the settings dialog opens and shows the editable profile controls

## Notes

- Keep this smoke pass broad; deterministic assertions belong in the automated suite at `test/playwrightgo/examples/examples_suite_test.go`.
