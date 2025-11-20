import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './specs',
  testIgnore: [
    '**/12-portfolio-site*.spec.ts', // Portfolio site tests require examples server
  ],
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : 12,
  reporter: 'list',
  
  use: {
    baseURL: 'http://127.0.0.1:8081',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  // Run local dev server before tests
  webServer: {
    command: 'node server.js',
    url: 'http://localhost:8081',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
