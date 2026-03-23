import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: ['86-atlas-commerce-os-ssr.spec.ts'],
  outputDir: '../bin/test-results/examples-atlas-ssr',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8096',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'powershell -NoProfile -Command "Push-Location .\\static; npm run build:css; $cssExitCode=$LASTEXITCODE; Pop-Location; if ($cssExitCode -ne 0) { exit $cssExitCode }; New-Item -ItemType Directory -Path .\\..\\bin\\examples -Force | Out-Null; $env:GOOS=\'js\'; $env:GOARCH=\'wasm\'; go build -o .\\..\\bin\\examples\\atlas-commerce-os.wasm .\\86-atlas-commerce-os\\client; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; Remove-Item Env:\\GOOS -ErrorAction SilentlyContinue; Remove-Item Env:\\GOARCH -ErrorAction SilentlyContinue; go run .\\86-atlas-commerce-os\\server"',
    url: 'http://127.0.0.1:8096/healthz',
    reuseExistingServer: true,
    timeout: 120 * 1000,
  },
});