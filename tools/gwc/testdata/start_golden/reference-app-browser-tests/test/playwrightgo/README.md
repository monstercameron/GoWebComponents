# Browser smoke tests for golden-reference-app

Use this folder for Playwright-Go smoke tests that validate starter boot and basic user interactions.

Run from the generated project root:

```powershell
go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v
```
