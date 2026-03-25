import { defineConfig, devices } from '@playwright/test';

const port = '8093';

const buildAndServe = [
  'pwsh -NoProfile -Command "',
  '$repo = Resolve-Path .;',
  "$bin = Join-Path $repo '../bin/examples';",
  'New-Item -ItemType Directory -Force -Path $bin | Out-Null;',
  "$env:GOOS='js';",
  "$env:GOARCH='wasm';",
  "go build -o (Join-Path $bin 'hydrate.wasm') ./71-hydrate;",
  'if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE };',
  "go build -o (Join-Path $bin 'ssr-bootstrap.wasm') ./73-ssr-bootstrap;",
  'if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE };',
  "go build -o (Join-Path $bin 'static-islands.wasm') ./101-static-islands;",
  'if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE };',
  'Remove-Item Env:GOOS -ErrorAction SilentlyContinue;',
  'Remove-Item Env:GOARCH -ErrorAction SilentlyContinue;',
  `$env:PORT='${port}';`,
  'node ../tools/dev-server/server.mjs',
  '"',
].join(' ');

export default defineConfig({
  testDir: './tests',
  testMatch: ['browser-compatibility.spec.ts'],
  outputDir: '../bin/test-results/examples-browser-compat',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  webServer: {
    command: buildAndServe,
    cwd: __dirname,
    url: `http://127.0.0.1:${port}/healthz`,
    reuseExistingServer: false,
    timeout: 180 * 1000,
  },
});
