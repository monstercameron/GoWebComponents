import { expect, test } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';

const resultsDir = '..\\tmp';
const resultsPath = '..\\tmp\\playwright-startup-atlas.json';

test.describe.configure({ mode: 'serial' });

test('records startup and first client navigation timings for atlas commerce os', async ({ page }) => {
  const start = Date.now();
  const response = await page.goto('/shop', { waitUntil: 'domcontentloaded' });
  expect(response, 'navigation should return a response').not.toBeNull();
  expect(response?.ok(), 'atlas startup navigation failed').toBeTruthy();

  await page.waitForLoadState('networkidle', { timeout: 20000 });
  await expect(page).toHaveTitle(/Atlas Shop/);
  await page.waitForFunction(() => (document.querySelector('#app')?.textContent ?? '').trim().length > 0, undefined, { timeout: 30000 });
  await expect(page.getByText('Frame Bench', { exact: true }).first()).toBeVisible({ timeout: 15000 });
  const readyMs = Date.now() - start;

  let networkIdleMs: number | null = null;
  networkIdleMs = Date.now() - start;

  const interactionStart = Date.now();
  await page.getByText('Frame Bench', { exact: true }).first().click();
  await expect(page).toHaveURL(/\/shop\/frame-bench$/);
  await expect(page).toHaveTitle(/Atlas Frame Bench/);
  await expect(page.getByRole('heading', { name: 'Atlas Frame Bench', exact: true })).toBeVisible();
  const interactionMs = Date.now() - interactionStart;

  const timings = await page.evaluate(() => {
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

  await mkdir(resultsDir, { recursive: true });
  await writeFile(
    resultsPath,
    JSON.stringify(
      {
        generatedAt: new Date().toISOString(),
        targetClass: 'atlas-ssr',
        result: {
          id: 'atlas-commerce-os',
          url: '/shop',
          readyMs,
          networkIdleMs,
          interactionMs,
          navigation: timings.navigation,
          wasmResources: timings.wasmResources,
        },
      },
      null,
      2,
    ),
    'utf8',
  );
});