/**
 * Playwright globalSetup for the chat-wizard spec.
 *
 * Runs before the webServer is started.  It:
 *   1. Removes any stale test database from a previous run.
 *   2. Creates the testdata/ directory if needed.
 *   3. Runs the Go seed-test-db command to populate a fresh SQLite database
 *      with two known test conversations.
 */
import { execSync } from 'node:child_process';
import { existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';

// __dirname here is examples/tests/support/
const REPO_ROOT  = path.resolve(__dirname, '../../..');
const TEST_DB_PATH = path.resolve(__dirname, '../../100-ai-chat-wizard/testdata/test_chat.db');
const TESTDATA_DIR = path.dirname(TEST_DB_PATH);

export default async function globalSetup(): Promise<void> {
  // ── Clean up any leftover DB from a previous run ──────────────────────────
  for (const ext of ['', '-wal', '-shm']) {
    const f = TEST_DB_PATH + ext;
    if (existsSync(f)) rmSync(f);
  }

  // ── Ensure the directory exists ───────────────────────────────────────────
  mkdirSync(TESTDATA_DIR, { recursive: true });

  // ── Seed the database ─────────────────────────────────────────────────────
  console.log('[chat-wizard-setup] seeding test database:', TEST_DB_PATH);
  execSync('go run ./examples/100-ai-chat-wizard/cmd/seed-test-db', {
    cwd: REPO_ROOT,
    env: { ...process.env, CHAT_DB_PATH: TEST_DB_PATH },
    stdio: 'inherit',
  });
  console.log('[chat-wizard-setup] seed complete');
}
