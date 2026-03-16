import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: ['18-ssr-server-routing.spec.ts'],
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8082',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'powershell -NoProfile -Command "$env:GOOS=\'windows\'; $env:GOARCH=\'amd64\'; $env:PORT=\'8082\'; go build -o .\\18-ssr-server-routing\\playwright-server.exe .\\18-ssr-server-routing; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; .\\18-ssr-server-routing\\playwright-server.exe"',
    url: 'http://127.0.0.1:8082/healthz',
    reuseExistingServer: false,
    timeout: 120 * 1000,
  },
});