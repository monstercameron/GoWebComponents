import { expect, test, type Locator, type Page } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';

const resultsDir = '..\\tmp';
const resultsPath = '..\\tmp\\playwright-startup-static.json';

type ExperimentResult = {
  id: string;
  url: string;
  readyMs: number;
  networkIdleMs: number | null;
  interactionMs: number;
  navigation: Record<string, number | null> | null;
  wasmResources: Array<Record<string, number | string>>;
};

const results: ExperimentResult[] = [];

function statValueForLabel(page: Page, label: string): Locator {
  return page.locator('small', { hasText: label }).locator('xpath=following-sibling::p[1]');
}

async function collectBrowserTimings(page: Page) {
  return page.evaluate(() => {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined;
    const wasmResources = performance
      .getEntriesByType('resource')
      .filter((entry): entry is PerformanceResourceTiming => entry instanceof PerformanceResourceTiming)
      .filter(entry => entry.name.endsWith('.wasm'))
      .map(entry => ({
        name: entry.name,
        transferSize: entry.transferSize,
        encodedBodySize: entry.encodedBodySize,
        decodedBodySize: entry.decodedBodySize,
        responseEnd: entry.responseEnd,
        duration: entry.duration,
      }));

    return {
      navigation: navigation
        ? {
            domContentLoadedEventEnd: navigation.domContentLoadedEventEnd,
            loadEventEnd: navigation.loadEventEnd,
            responseEnd: navigation.responseEnd,
            domInteractive: navigation.domInteractive,
          }
        : null,
      wasmResources,
    };
  });
}

test.describe.configure({ mode: 'serial' });

test.afterAll(async () => {
  await mkdir(resultsDir, { recursive: true });
  await writeFile(
    resultsPath,
    JSON.stringify(
      {
        generatedAt: new Date().toISOString(),
        targetClass: 'static-examples',
        results,
      },
      null,
      2,
    ),
    'utf8',
  );
});

const experiments = [
  {
    id: 'ui-render',
    url: '/21-ui-render/ui-render.html',
    readyLocator: (page: Page) => page.getByRole('button', { name: 'Increment mounted state' }),
    interaction: async (page: Page) => {
      const stat = statValueForLabel(page, 'Clicks');
      await expect(stat).toHaveText('0');
      await page.getByRole('button', { name: 'Increment mounted state' }).click();
      await expect(stat).toHaveText('1');
    },
  },
  {
    id: 'browser-router',
    url: '/56-browser-router/browser-router.html',
    readyLocator: (page: Page) => page.getByRole('button', { name: 'Pricing' }),
    interaction: async (page: Page) => {
      await page.getByRole('button', { name: 'Pricing' }).click();
      await expect(page.getByRole('heading', { name: 'Pricing route' })).toBeVisible();
      await expect(statValueForLabel(page, 'Current path')).toContainText('/pricing');
    },
  },
];

for (const experiment of experiments) {
  test(`records startup timings for ${experiment.id}`, async ({ page }) => {
    const start = Date.now();
    const response = await page.goto(experiment.url, { waitUntil: 'domcontentloaded' });
    expect(response, 'navigation should return a response').not.toBeNull();
    expect(response?.ok(), `navigation failed for ${experiment.id}`).toBeTruthy();

    const readyLocator = experiment.readyLocator(page);
    await expect(readyLocator).toBeVisible();
    const readyMs = Date.now() - start;

    let networkIdleMs: number | null = null;
    try {
      await page.waitForLoadState('networkidle', { timeout: 15000 });
      networkIdleMs = Date.now() - start;
    } catch {
      networkIdleMs = null;
    }

    const interactionStart = Date.now();
    await experiment.interaction(page);
    const interactionMs = Date.now() - interactionStart;

    const timings = await collectBrowserTimings(page);
    results.push({
      id: experiment.id,
      url: experiment.url,
      readyMs,
      networkIdleMs,
      interactionMs,
      navigation: timings.navigation,
      wasmResources: timings.wasmResources,
    });
  });
}