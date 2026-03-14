import { test, expect, Page } from '@playwright/test';

const LIST_SIZE = 10;

const waitForRenderedList = async (page: Page) => {
  await page.waitForFunction(
    (expectedCount) =>
      document.querySelectorAll('.list-item').length === expectedCount &&
      document.querySelector('#item-count')?.textContent === `Count: ${expectedCount}`,
    LIST_SIZE,
  );
};

const waitForUpdatedList = async (page: Page) => {
  await page.waitForFunction(
    (expectedCount) =>
      document.querySelectorAll('.list-item').length === expectedCount &&
      Array.from(document.querySelectorAll('.list-item')).every((node) =>
        node.textContent?.includes('(Updated)'),
      ),
    LIST_SIZE,
  );
};

const measureRender = async (page: Page, url: string, label: string) => {
  await page.goto(url);
  await page.waitForSelector('#btn-render');

  let start = Date.now();
  await page.click('#btn-render');
  await waitForRenderedList(page);
  const renderTime = Date.now() - start;

  return { label, renderTime };
};

const measureUpdate = async (page: Page, url: string, label: string) => {
  await page.goto(url);
  await page.waitForSelector('#btn-render');
  await page.click('#btn-render');
  await waitForRenderedList(page);

  let start = Date.now();
  await page.click('#btn-update');
  await waitForUpdatedList(page);
  const updateTime = Date.now() - start;

  return { label, updateTime };
};

const measureClear = async (page: Page, url: string) => {
  await page.goto(url);
  await page.waitForSelector('#btn-render');
  await page.click('#btn-render');
  await waitForRenderedList(page);

  const start = Date.now();
  await page.click('#btn-clear');
  await page.waitForFunction(() => document.querySelector('#item-count')?.textContent === 'Count: 0');
  return Date.now() - start;
};

const measureDeepTree = async (page: Page, url: string) => {
  await page.goto(url);
  await page.waitForSelector('#btn-render');

  const start = Date.now();
  await page.click('#btn-deep');
  await page.waitForSelector('#deep-leaf');
  return Date.now() - start;
};

const measureHooks = async (page: Page, url: string) => {
  await page.goto(url);
  await page.waitForSelector('#btn-render');

  const start = Date.now();
  await page.click('#btn-hooks');
  await page.waitForSelector('#hooks-container');
  return Date.now() - start;
};

const runSecondaryBenchmark = async (page: Page, url: string, label: string) => {
  const clearTime = await measureClear(page, url);
  const deepTreeTime = await measureDeepTree(page, url);
  const hooksTime = await measureHooks(page, url);

  return { label, clearTime, deepTreeTime, hooksTime };
};

test.describe('Performance Comparison', () => {
  test.describe.configure({ mode: 'serial' });

  test('Compare GoWebComponents vs React render', async ({ page }) => {
    test.setTimeout(120_000);

    const goResults = await measureRender(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await measureRender(page, '/react/index.html', 'React');

    console.log('\nRender Comparison (Lower is better):');
    console.table([goResults, reactResults]);

    expect(goResults.renderTime).toBeGreaterThan(0);
    expect(reactResults.renderTime).toBeGreaterThan(0);
  });

  test('Compare GoWebComponents vs React update', async ({ page }) => {
    test.setTimeout(120_000);

    const goResults = await measureUpdate(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await measureUpdate(page, '/react/index.html', 'React');

    console.log('\nUpdate Comparison (Lower is better):');
    console.table([goResults, reactResults]);

    expect(goResults.updateTime).toBeGreaterThan(0);
    expect(reactResults.updateTime).toBeGreaterThan(0);
  });

  test('Compare GoWebComponents vs React secondary scenarios', async ({ page }) => {
    test.setTimeout(120_000);

    const goResults = await runSecondaryBenchmark(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await runSecondaryBenchmark(page, '/react/index.html', 'React');

    console.log('\nSecondary Comparison (Lower is better):');
    console.table([goResults, reactResults]);

    expect(goResults.clearTime).toBeGreaterThanOrEqual(0);
    expect(reactResults.clearTime).toBeGreaterThanOrEqual(0);
  });
});
