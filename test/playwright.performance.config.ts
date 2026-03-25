import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { resolveArtifactOutputPath } from '../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..');
const serveCommand = [
  'go',
  'run',
  '../tools/gwc',
  'serve',
  '-root',
  './benchmark',
  '-host',
  '127.0.0.1',
  '-port',
  '8082',
].join(' ');

export default defineConfig({
  testDir: './specs',
  testMatch: ['performance_benchmark.spec.ts'],
  outputDir: resolveArtifactOutputPath(repoRoot, 'test-results', 'test-performance'),
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
    command: serveCommand,
    cwd: __dirname,
    url: 'http://127.0.0.1:8082',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
