import { expect, test } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';

const resultsDir = '..\\bin';
const resultsPath = '..\\bin\\playwright-virtualization-benchmark.json';

type ScrollBenchmarkResult = {
  target: string;
  durationMs: number;
  scrollHeight: number;
  clientHeight: number;
};

const results: ScrollBenchmarkResult[] = [];

async function benchmarkScroll(page: import('@playwright/test').Page, selector: string) {
  return page.evaluate(async (targetSelector) => {
    const element = document.querySelector<HTMLElement>(targetSelector);
    if (!element) {
      throw new Error(`missing scroll container: ${targetSelector}`);
    }
    element.scrollTop = 0;
    await new Promise(requestAnimationFrame);

    const maxScroll = Math.max(0, element.scrollHeight - element.clientHeight);
    const stepCount = 24;
    const started = performance.now();
    for (let step = 1; step <= stepCount; step++) {
      element.scrollTop = (maxScroll * step) / stepCount;
      await new Promise(requestAnimationFrame);
    }
    const durationMs = performance.now() - started;
    return {
      durationMs,
      scrollHeight: element.scrollHeight,
      clientHeight: element.clientHeight,
    };
  }, selector);
}

test.describe.configure({ mode: 'serial' });

test.afterAll(async () => {
  await mkdir(resultsDir, { recursive: true });
  await writeFile(
    resultsPath,
    JSON.stringify(
      {
        generatedAt: new Date().toISOString(),
        targetClass: 'virtualized-scroll-surfaces',
        results,
      },
      null,
      2,
    ),
    'utf8',
  );
});

test('records browser scroll timings for virtualized and full-render feed surfaces', async ({ page }) => {
  const response = await page.goto('/103-virtualized-feed/virtualized-feed.html', { waitUntil: 'domcontentloaded' });
  expect(response?.ok(), 'virtualized feed example failed to load').toBeTruthy();

  await expect(page.getByRole('heading', { name: 'virtualization.List' })).toBeVisible();
  await expect(page.locator('#virtualized-benchmark-feed')).toBeVisible();
  await expect(page.locator('#full-benchmark-feed')).toBeVisible();

  const virtualized = await benchmarkScroll(page, '#virtualized-benchmark-feed');
  const full = await benchmarkScroll(page, '#full-benchmark-feed');

  results.push({ target: 'virtualized', ...virtualized });
  results.push({ target: 'full', ...full });

  expect(virtualized.scrollHeight).toBeGreaterThan(virtualized.clientHeight);
  expect(full.scrollHeight).toBeGreaterThan(full.clientHeight);
});
