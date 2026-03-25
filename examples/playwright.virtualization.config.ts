import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: ['virtualized-feed-benchmark.spec.ts', 'virtualized-feed-regression.spec.ts', 'virtualized-feed-restoration.spec.ts', 'virtualized-feed-hydration.spec.ts'],
  outputDir: '../bin/test-results/examples-virtualization',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8087',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'powershell -NoProfile -Command "Push-Location .\\static; npm run build:css; $cssExitCode=$LASTEXITCODE; Pop-Location; if ($cssExitCode -ne 0) { exit $cssExitCode }; New-Item -ItemType Directory -Path .\\static\\bin -Force | Out-Null; $env:GOOS=\'js\'; $env:GOARCH=\'wasm\'; go build -o .\\static\\bin\\virtualized-feed.wasm .\\103-virtualized-feed; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; Remove-Item Env:\\GOOS -ErrorAction SilentlyContinue; Remove-Item Env:\\GOARCH -ErrorAction SilentlyContinue; npx http-server . -p 8087"',
    url: 'http://127.0.0.1:8087',
    reuseExistingServer: false,
    timeout: 120 * 1000,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
