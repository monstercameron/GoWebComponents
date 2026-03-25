import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { resolveWorkspaceBuildPath } from '../../../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '../../..');
const examplesRoot = path.resolve(repoRoot, 'examples');
const serverBinaryPath = resolveWorkspaceBuildPath(repoRoot, 'examples', '18-ssr-server-routing', 'playwright-server.exe');

export default {
  testDir: path.join(examplesRoot, 'tests'),
  testMatch: ['18-ssr-server-routing.spec.ts'],
  outputDir: '../../../bin/test-results/examples-ssr-server',
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
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: `powershell -NoProfile -Command "$env:GOOS='js'; $env:GOARCH='wasm'; go build -o .\\static\\bin\\ssr-server-routing.wasm .\\18-ssr-server-routing; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; New-Item -ItemType Directory -Force -Path '${path.dirname(serverBinaryPath)}' | Out-Null; $env:GOOS='windows'; $env:GOARCH='amd64'; $env:PORT='8082'; go build -o '${serverBinaryPath}' .\\18-ssr-server-routing\\server_main.go .\\18-ssr-server-routing\\shared.go .\\18-ssr-server-routing\\constants.go; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; & '${serverBinaryPath}'"`,
    cwd: examplesRoot,
    url: 'http://127.0.0.1:8082/healthz',
    reuseExistingServer: true,
    timeout: 120 * 1000,
  },
};
