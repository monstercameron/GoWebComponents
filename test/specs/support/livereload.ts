import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import type { Page } from '@playwright/test';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const embeddedClientScriptPath = path.join(__dirname, '..', '..', '..', 'tools', 'livereload', 'scripts', 'livereload-client.txt');
const embeddedClientScript = readFileSync(embeddedClientScriptPath, 'utf8');

export async function addLivereloadClient(page: Page) {
  await page.addInitScript({ content: embeddedClientScript });
}
