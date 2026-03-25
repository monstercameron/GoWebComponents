/**
 * Playwright config for the 100-ai-chat-wizard end-to-end tests.
 *
 * The config starts the chat wizard Go server on port 8099 (separate from the
 * default dev-server on 8081 and the user's live server on 8095) so that tests
 * always run against a fresh, deterministic SQLite database seeded by
 * globalSetup before the server boots.
 *
 * Run with:
 *   npx playwright test --config=playwright.chat-wizard.config.ts
 */
import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';

// Absolute path to the test-specific SQLite database.
// We write it inside the example's testdata directory so it stays out of the
// developer's live DB.
const TEST_DB_PATH = path.resolve(
  __dirname,
  '100-ai-chat-wizard/testdata/test_chat.db',
);

// Repo root — the server must run from here so its static-dir resolution finds
// examples/100-ai-chat-wizard/client and examples/static.
const REPO_ROOT = path.resolve(__dirname, '..');

// Seed command: removes stale DB files, ensures the directory exists, seeds
// fresh test data, then starts the server in one sequential shell command.
// This guarantees seeding completes before the HTTP listener opens.
const seedAndServe = [
  'powershell -NoProfile -Command "',
  // Remove stale WAL/SHM artefacts from previous runs
  `Remove-Item -ErrorAction SilentlyContinue '${TEST_DB_PATH}','${TEST_DB_PATH}-wal','${TEST_DB_PATH}-shm';`,
  // Ensure the directory exists
  `New-Item -ItemType Directory -Force '${path.dirname(TEST_DB_PATH)}' | Out-Null;`,
  // Seed the database
  `$env:CHAT_DB_PATH='${TEST_DB_PATH}';`,
  `go run ./examples/100-ai-chat-wizard/cmd/seed-test-db;`,
  `if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE };`,
  // Launch the server (inherits CHAT_DB_PATH and LISTEN_ADDR from env)
  `go run ./examples/100-ai-chat-wizard/cmd/server`,
  '"',
].join(' ');

export default defineConfig({
  testDir: './tests',
  testMatch: ['100-ai-chat-wizard.spec.ts'],
  outputDir: '../bin/test-results/examples-chat-wizard',

  // Run serially — the server holds in-memory session state and a single-
  // writer SQLite DB; parallel tests would race on the same DB file.
  fullyParallel: false,
  workers: 1,

  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: 'list',

  use: {
    baseURL: 'http://127.0.0.1:8099',
    trace: 'on-first-retry',
    // Record video on failure so failures are easy to diagnose locally.
    video: 'retain-on-failure',
    // The WASM binary is ~24 MB; allow up to 90 s per test so the browser has
    // time to download, compile, and instantiate it.
    actionTimeout: 90_000,
  },

  // 90 s per test — covers cold WASM compilation on the first test.
  timeout: 90_000,

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    // The command seeds the DB then starts the server in one sequential
    // PowerShell session, ensuring the DB exists before the server reads it.
    command: seedAndServe,
    cwd: REPO_ROOT,
    url: 'http://127.0.0.1:8099/healthz',
    // Use a dedicated test port so we never interfere with the dev server on
    // 8095, and always get our own server + test database.
    reuseExistingServer: false,
    // Allow time for two go run compilations (seed + server).
    timeout: 90_000,
    env: {
      LISTEN_ADDR: '127.0.0.1:8099',
      CHAT_DB_PATH: TEST_DB_PATH,
      // No OPENAI_API_KEY — chat Send RPC will return Unavailable, which is
      // expected; all other RPCs (List/Load/Delete) work fine without a key.
    },
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
