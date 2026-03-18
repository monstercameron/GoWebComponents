import { expect, test } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

type ExampleEntry = {
  dir: string;
  htmlFile: string;
};

type InteractionPlan = {
  maxFields?: number;
  maxActions?: number;
};

const examplesRoot = path.resolve(__dirname, '..');
const dangerousActionPattern = /(error|throw|panic|crash|fail|destroy|explode)/i;
const interactionPlans: Record<string, InteractionPlan> = {
  '14-omi': {
    maxFields: 3,
    maxActions: 0,
  },
};

function discoverExamples(): ExampleEntry[] {
  return fs
    .readdirSync(examplesRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && /^\d{2}-/.test(entry.name))
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
      };
    })
    .sort((left, right) => left.dir.localeCompare(right.dir));
}

async function gotoExample(
  page: Parameters<typeof test>[1] extends never ? never : any,
  example: ExampleEntry,
) {
  const candidates = [
    `/${example.dir}/${example.htmlFile}`,
    `/examples/${example.dir}/${example.htmlFile}`,
  ];

  let lastResponse: Awaited<ReturnType<typeof page.goto>> | null = null;
  for (const candidate of candidates) {
    const response = await page.goto(candidate, { waitUntil: 'domcontentloaded' });
    lastResponse = response;
    if (response?.ok()) {
      return { response, url: candidate };
    }
  }

  return { response: lastResponse, url: candidates[0] };
}

async function fillField(locator: ReturnType<typeof test['extend']> extends never ? never : any) {
  const tagName = await locator.evaluate((node: Element) => node.tagName.toLowerCase());
  if (tagName === 'textarea') {
    await locator.fill('Playwright interaction sample');
    return;
  }

  if (tagName === 'select') {
    const optionCount = await locator.locator('option').count();
    if (optionCount > 1) {
      await locator.selectOption({ index: 1 });
    }
    return;
  }

  const type = ((await locator.getAttribute('type')) || 'text').toLowerCase();
  switch (type) {
    case 'checkbox':
    case 'radio':
      await locator.click();
      return;
    case 'number':
    case 'range':
      await locator.fill('3');
      return;
    case 'date':
      await locator.fill('2026-03-16');
      return;
    case 'email':
      await locator.fill('playwright@example.com');
      return;
    case 'url':
      await locator.fill('https://example.com');
      return;
    case 'tel':
      await locator.fill('5550101');
      return;
    case 'color':
      await locator.fill('#22c55e');
      return;
    default:
      await locator.fill('Playwright');
  }
}

async function runGenericInteractions(page: Parameters<typeof test>[1] extends never ? never : any, exampleDir: string) {
  const plan = interactionPlans[exampleDir] || {};
  const inputs = page.locator('input:not([type="hidden"]):not([type="file"]), textarea, select');
  const inputCount = Math.min(await inputs.count(), plan.maxFields ?? 3);
  for (let index = 0; index < inputCount; index += 1) {
    const locator = inputs.nth(index);
    if (!(await locator.isVisible().catch(() => false))) {
      continue;
    }
    if (!(await locator.isEnabled().catch(() => false))) {
      continue;
    }

    await fillField(locator);
    await page.waitForTimeout(150);
  }

  const maxActions = plan.maxActions ?? 3;
  if (maxActions === 0) {
    return;
  }

  const actions = page.locator('button, [role="button"], a[href]');
  const actionCount = await actions.count();
  let clicked = 0;
  for (let index = 0; index < actionCount && clicked < maxActions; index += 1) {
    const locator = actions.nth(index);
    if (!(await locator.isVisible().catch(() => false))) {
      continue;
    }
    if (!(await locator.isEnabled().catch(() => false))) {
      continue;
    }

    const text = ((await locator.innerText().catch(() => '')) || '').trim();
    if (dangerousActionPattern.test(text)) {
      continue;
    }

    const href = (await locator.getAttribute('href').catch(() => null)) || '';
    if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('mailto:')) {
      continue;
    }

    await locator.click({ timeout: 5000 }).catch(() => undefined);
    clicked += 1;
    await page.waitForTimeout(200);
  }
}

const examples = discoverExamples();

test.describe('All example entrypoints on the dev server', () => {
  for (const example of examples) {
    test(`${example.dir} loads and accepts browser interaction`, async ({ page }) => {
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

      const { response, url } = await gotoExample(page, example);
      expect(response, `navigation should return a response for ${example.dir}`).not.toBeNull();
      expect(response?.ok(), `navigation failed for ${example.dir}`).toBeTruthy();

      await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
      await expect(page.locator('body')).toBeVisible();
      await runGenericInteractions(page, example.dir);
      await page.waitForTimeout(250);

      const bodyText = (await page.locator('body').innerText()).trim();
      expect(bodyText.length, `body text should not be empty for ${url}`).toBeGreaterThan(0);

      const runtimeFailures = [
        ...pageErrors.map((message) => `pageerror: ${message}`),
        ...consoleErrors.map((message) => `console: ${message}`),
        ...requestFailures.map((message) => `request: ${message}`),
      ];

      expect(runtimeFailures, `${example.dir} emitted runtime failures`).toEqual([]);
    });
  }
});