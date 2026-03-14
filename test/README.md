# Browser Test Suite

Location: `test/`

This directory contains the Playwright-based browser test layers for GoWebComponents.

## What Is Covered

### Component contracts

`specs/component_contract.spec.js`

Focus:

- local state isolation
- rerender stability
- effect cleanup behavior
- `UseId` wiring
- stateful component behavior under normal interactions

### Integration flows

`specs/integration_flows.spec.js`

Focus:

- multi-component state coordination
- shared atom flows
- todo app interactions
- fetch interactions against the mock API
- form behavior in the browser

### Deep state stress

`specs/state_stress.spec.js`

Focus:

- repeated `UseState` updates
- `5+`, `25+`, `100+` update bursts
- mixed local and shared state updates
- state continuity through rerenders

### Existing browser suites

The directory also keeps the older broader Playwright specs that cover router behavior, basic hooks behavior, and other end-to-end flows.

## Setup

```powershell
cd test
npm install
npm run install:browsers
```

## Run All Browser Tests

```powershell
npm test
```

## Run Focused Suites

```powershell
npm run test:components
npm run test:integration
npm run test:state
```

## Other Useful Commands

```powershell
npm run test:headed
npm run test:debug
npm run test:ui
```

## How The Test App Works

- `testapp/` contains the Go wasm app used by these tests
- the test scripts rebuild `testapp/main.wasm` before the relevant suites
- `server.js` provides the local test server and mock API endpoints used by fetch/integration flows

## Notes

- Older docs referenced building unrelated example apps before running the test suite. That is stale. The test package builds its own `testapp/` target.
- Playwright residue such as `test-results/` and reports are ignored and should not be committed.
