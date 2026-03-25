import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { resolveArtifactOutputPath } from '../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..');

const configuredWorkers = process.env.PLAYWRIGHT_WORKERS
  ? Number(process.env.PLAYWRIGHT_WORKERS)
  : undefined;
const configuredPort = process.env.PORT || '8083';
const baseURL = `http://127.0.0.1:${configuredPort}`;

export default defineConfig({
  testDir: './specs',
  outputDir: resolveArtifactOutputPath(repoRoot, 'test-results', 'test'),
  testIgnore: [
    '**/12-portfolio-site*.spec.ts', // Portfolio site tests require examples server
    '**/performance_benchmark.spec.ts', // Benchmark suite runs under dedicated config/server
  ],
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: configuredWorkers ?? (process.env.CI ? 1 : 12),
  reporter: 'list',
  
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  // Run local dev server before tests
  webServer: {
    command: `powershell -NoProfile -Command "$env:PORT='${configuredPort}'; node server.js"`,
    url: `${baseURL}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
