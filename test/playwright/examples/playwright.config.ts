import path from 'node:path';
import { fileURLToPath } from 'node:url';

const configuredWorkers = process.env.PLAYWRIGHT_WORKERS
  ? Number(process.env.PLAYWRIGHT_WORKERS)
  : undefined;
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const examplesRoot = path.resolve(__dirname, '../../../examples');

export default {
  testDir: path.join(examplesRoot, 'tests'),
  testIgnore: ['18-ssr-server-routing.spec.ts', 'all-examples-links.spec.ts'],
  outputDir: '../../../bin/test-results/examples',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: configuredWorkers ?? (process.env.CI ? 1 : undefined),
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8081',
    trace: 'on-first-retry',
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],

  /* Run your local dev server before starting the tests */
  webServer: {
    command: `node --input-type=module -e "process.env.PORT='8081'; await import('../tools/dev-server/server.mjs');"`,
    cwd: examplesRoot,
    url: 'http://127.0.0.1:8081',
    reuseExistingServer: !process.env.CI,
    stdout: 'ignore',
    stderr: 'pipe',
  },
};
