# Test Suite

**Location:** `/test`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/
├── render/
├── router/
├── fetch/
├── internal/
├── examples/
├── test/             ← YOU ARE HERE
│   ├── specs/
│   ├── testapp/
│   ├── static/
│   └── ...
└── tools/
```

## Overview

End-to-end tests for GoWebComponents WASM using Playwright. Tests run in real browsers to verify all framework features.

## Setup

1. Install dependencies:
```powershell
cd test
npm install
```

2. Install Playwright browsers:
```powershell
npm run install:browsers
```

3. Build the WASM binary:
```powershell
cd ..
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o examples/static/bin/main.wasm examples/main.go
```

4. Copy wasm_exec.js (if not already present):
```powershell
Copy-Item "$env:GOROOT\misc\wasm\wasm_exec.js" -Destination "examples\static\script\wasm_exec.js"
```

## Running Tests

Run all tests:
```powershell
npm test
```

Run tests with UI:
```powershell
npm run test:ui
```

Run tests in headed mode (see browser):
```powershell
npm run test:headed
```

Debug tests:
```powershell
npm run test:debug
```

## Test Structure

- `specs/basic.spec.js` - Basic WASM loading and DOM rendering tests
- `specs/hooks.spec.js` - Tests for React-like hooks (UseState, UseEffect, etc.)
- `playwright.config.js` - Playwright configuration
- `package.json` - Node dependencies

## Writing Tests

Tests are written using Playwright's test framework:

```javascript
import { test, expect } from '@playwright/test';

test('my test', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#app')).toBeVisible();
});
```

## CI/CD

Tests can be run in CI with:
```bash
npm test
```

Set `CI=true` environment variable for CI-specific behavior.
