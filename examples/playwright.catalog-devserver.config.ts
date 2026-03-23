import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: ['all-examples-*.spec.ts'],
  outputDir: '../bin/test-results/examples-catalog',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8091',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'powershell -NoProfile -Command "$env:PORT=\'8091\'; node ../tools/dev-server/server.mjs"',
    url: 'http://127.0.0.1:8091/healthz',
    reuseExistingServer: false,
    timeout: 120 * 1000,
  },
});