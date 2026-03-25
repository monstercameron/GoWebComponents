import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const examplesRoot = path.resolve(__dirname, '../../../examples');

export default {
  testDir: path.join(examplesRoot, 'tests'),
  testMatch: ['startup-experiments.spec.ts'],
  outputDir: '../../../bin/test-results/examples-startup',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8084',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: 'powershell -NoProfile -Command "$env:GOOS=\'js\'; $env:GOARCH=\'wasm\'; go build -o .\\static\\bin\\ui-render.wasm .\\21-ui-render; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; go build -o .\\static\\bin\\browser-router.wasm .\\56-browser-router; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; Remove-Item Env:\\GOOS -ErrorAction SilentlyContinue; Remove-Item Env:\\GOARCH -ErrorAction SilentlyContinue; npx http-server . -p 8084"',
    cwd: examplesRoot,
    url: 'http://127.0.0.1:8084',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
    stdout: 'ignore',
    stderr: 'pipe',
  },
};
