import { test, expect, Page } from '@playwright/test';

const runBenchmark = async (page: Page, url: string, label: string) => {
  await page.goto(url);
  await page.waitForSelector('#btn-render');

  // Render
  let start = Date.now();
  await page.click('#btn-render');
  await page.waitForFunction(() => document.querySelector('#item-count')?.textContent === 'Count: 1000');
  const renderTime = Date.now() - start;

  // Update
  start = Date.now();
  await page.click('#btn-update');
  await page.waitForFunction(() => document.querySelector('.list-item')?.textContent?.includes('(Updated)'));
  const updateTime = Date.now() - start;

  // Clear
  start = Date.now();
  await page.click('#btn-clear');
  await page.waitForFunction(() => document.querySelector('#item-count')?.textContent === 'Count: 0');
  const clearTime = Date.now() - start;

  // Deep Tree
  start = Date.now();
  await page.click('#btn-deep');
  await page.waitForSelector('#deep-leaf');
  const deepTreeTime = Date.now() - start;

  // Many Hooks
  start = Date.now();
  await page.click('#btn-hooks');
  await page.waitForSelector('#hooks-container');
  const hooksTime = Date.now() - start;

  // Compute Primes
  start = Date.now();
  await page.click('#btn-compute');
  await page.waitForFunction(() => document.querySelector('#compute-result')?.textContent?.includes('Found'));
  const computeTime = Date.now() - start;

  return { label, renderTime, updateTime, clearTime, deepTreeTime, hooksTime, computeTime };
};

test.describe('Performance Comparison', () => {
  test('Compare GoWebComponents vs React', async ({ page }) => {
    const goResults = await runBenchmark(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await runBenchmark(page, '/react/index.html', 'React');

    console.log('\nPerformance Comparison (Lower is better):');
    console.table([goResults, reactResults]);

    // Basic assertions to ensure tests ran correctly
    expect(goResults.renderTime).toBeGreaterThan(0);
    expect(reactResults.renderTime).toBeGreaterThan(0);
  });
});
