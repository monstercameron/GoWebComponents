import { expect, test } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

type ExampleEntry = {
  dir: string;
  htmlFile: string;
  url: string;
};

const examplesRoot = path.resolve(__dirname, '..');
const skipDirs = new Set([
  '18-ssr-server-routing',
]);

function discoverExamples(): ExampleEntry[] {
  return fs
    .readdirSync(examplesRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && /^\d{2}-/.test(entry.name))
    .filter((entry) => !skipDirs.has(entry.name))
    .map((entry) => {
      const htmlFile = fs
        .readdirSync(path.join(examplesRoot, entry.name))
        .filter((name) => name.endsWith('.html'))
        .sort()[0];

      if (!htmlFile) {
        throw new Error(`No HTML entrypoint found for ${entry.name}`);
      }

      return {
        dir: entry.name,
        htmlFile,
        url: `/${entry.name}/${htmlFile}`,
      };
    })
    .sort((left, right) => left.dir.localeCompare(right.dir));
}

const examples = discoverExamples();

test.describe('Catalog smoke coverage', () => {
  for (const example of examples) {
    test(`${example.dir} renders without runtime errors`, async ({ page }) => {
      test.slow();

      const pageErrors: string[] = [];
      const consoleErrors: string[] = [];
      const requestFailures: string[] = [];

      page.on('pageerror', (error) => {
        pageErrors.push(error.message);
      });

      page.on('console', (message) => {
        if (message.type() === 'error') {
          consoleErrors.push(message.text());
        }
      });

      page.on('requestfailed', (request) => {
        requestFailures.push(`${request.method()} ${request.url()} :: ${request.failure()?.errorText ?? 'failed'}`);
      });

      const response = await page.goto(example.url, { waitUntil: 'domcontentloaded' });
      expect(response, 'navigation should return a response').not.toBeNull();
      expect(response?.ok(), `navigation failed for ${example.url}`).toBeTruthy();

      await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
      await expect(page.locator('body')).toBeVisible();

      const bodyText = (await page.locator('body').innerText()).trim();
      expect(bodyText.length, `body text should not be empty for ${example.dir}`).toBeGreaterThan(0);

      const runtimeFailures = [
        ...pageErrors.map((message) => `pageerror: ${message}`),
        ...consoleErrors.map((message) => `console: ${message}`),
        ...requestFailures.map((message) => `request: ${message}`),
      ];

      expect(runtimeFailures, `${example.dir} emitted runtime failures`).toEqual([]);
    });
  }
});