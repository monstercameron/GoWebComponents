import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const examplesRoot = path.resolve(__dirname, '../../../examples');

export default {
  testDir: path.join(examplesRoot, 'tests'),
  testMatch: ['86-atlas-commerce-os-startup.spec.ts'],
  outputDir: '../../../bin/test-results/examples-atlas-startup',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8096',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: 'powershell -NoProfile -Command "Push-Location .\\static; npm run build:css; $cssExitCode=$LASTEXITCODE; Pop-Location; if ($cssExitCode -ne 0) { exit $cssExitCode }; New-Item -ItemType Directory -Path .\\..\\bin\\examples -Force | Out-Null; $env:GOOS=\'js\'; $env:GOARCH=\'wasm\'; go build -o .\\..\\bin\\examples\\atlas-commerce-os.wasm .\\86-atlas-commerce-os\\client; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; Remove-Item Env:\\GOOS -ErrorAction SilentlyContinue; Remove-Item Env:\\GOARCH -ErrorAction SilentlyContinue; go run .\\86-atlas-commerce-os\\server"',
    cwd: examplesRoot,
    url: 'http://127.0.0.1:8096/healthz',
    reuseExistingServer: true,
    timeout: 120 * 1000,
  },
};
