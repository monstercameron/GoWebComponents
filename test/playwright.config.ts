import { defineConfig } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { resolveArtifactOutputPath, resolveWorkspaceBuildPath } from '../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..');
const fixturePath = path.relative(__dirname, path.join(__dirname, 'fixtures', 'user-123.json')).split(path.sep).join('/');
const testAppWasmPath = path.relative(__dirname, resolveWorkspaceBuildPath(repoRoot, 'test', 'testapp', 'main.wasm')).split(path.sep).join('/');

const configuredWorkers = process.env.PLAYWRIGHT_WORKERS
  ? Number(process.env.PLAYWRIGHT_WORKERS)
  : undefined;
const configuredPort = process.env.PORT || '8083';
const baseURL = `http://127.0.0.1:${configuredPort}`;
const serveCommand = [
  'go',
  'run',
  '../tools/gwc',
  'serve',
  '-root',
  './testapp',
  '-host',
  '127.0.0.1',
  '-port',
  configuredPort,
  '-wasm-route',
  '/main.wasm',
  '-wasm-file',
  testAppWasmPath,
  '-fixture-json',
  `/api/user/123=${fixturePath}`,
].map((value) => value.includes(' ') ? `"${value}"` : value).join(' ');

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
    command: serveCommand,
    cwd: __dirname,
    url: `${baseURL}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
