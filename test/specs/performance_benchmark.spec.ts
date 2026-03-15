import { test, expect, Page } from '@playwright/test';

const CORE_LIST_SIZE = 40;
const CONTENT_CARD_COUNT = 12;
const MEASUREMENT_LOOPS = 5;

const average = (values: number[]) =>
  Math.round(values.reduce((sum, value) => sum + value, 0) / values.length);

const measureLoops = async (measure: () => Promise<number>) => {
  const values: number[] = [];
  for (let i = 0; i < MEASUREMENT_LOOPS; i++) {
    values.push(await measure());
  }
  return average(values);
};

const waitForRenderedList = async (page: Page) => {
  await page.waitForFunction(
    (expectedCount) =>
      document.querySelectorAll('.core-list-item').length === expectedCount &&
      document.querySelector('#core-count')?.textContent === `Core Count: ${expectedCount}`,
    CORE_LIST_SIZE,
  );
};

const waitForUpdatedList = async (page: Page) => {
  await page.waitForFunction(
    (expectedCount) =>
      document.querySelectorAll('.core-list-item').length === expectedCount &&
      Array.from(document.querySelectorAll('.core-list-item')).every((node) =>
        node.textContent?.includes('(Updated)'),
      ),
    CORE_LIST_SIZE,
  );
};

const waitForRenderedContent = async (page: Page) => {
  await page.waitForFunction(
    (expectedCount) =>
      document.querySelectorAll('.content-card').length === expectedCount &&
      document.querySelector('#content-count')?.textContent === `Content Count: ${expectedCount}`,
    CONTENT_CARD_COUNT,
  );
};

const waitForUpdatedContent = async (page: Page) => {
  await page.waitForFunction(
    (expectedCount) =>
      document.querySelectorAll('.content-card').length === expectedCount &&
      Array.from(document.querySelectorAll('.content-title')).every((node) =>
        node.textContent?.includes('(Updated)'),
      ) &&
      Array.from(document.querySelectorAll('.content-status')).every((node) => node.textContent === 'live'),
    CONTENT_CARD_COUNT,
  );
};

const measureRender = async (page: Page, url: string, label: string) => {
  const renderTime = await measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-render');

    const start = Date.now();
    await page.click('#btn-render');
    await waitForRenderedList(page);
    return Date.now() - start;
  });

  return { label, renderTime };
};

const measureUpdate = async (page: Page, url: string, label: string) => {
  const updateTime = await measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-render');
    await page.click('#btn-render');
    await waitForRenderedList(page);

    const start = Date.now();
    await page.click('#btn-update');
    await waitForUpdatedList(page);
    return Date.now() - start;
  });

  return { label, updateTime };
};

const measureContentRender = async (page: Page, url: string, label: string) => {
  const renderTime = await measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-content-render');

    const start = Date.now();
    await page.click('#btn-content-render');
    await waitForRenderedContent(page);
    return Date.now() - start;
  });

  return { label, renderTime };
};

const measureContentUpdate = async (page: Page, url: string, label: string) => {
  const updateTime = await measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-content-render');
    await page.click('#btn-content-render');
    await waitForRenderedContent(page);

    const start = Date.now();
    await page.click('#btn-content-update');
    await waitForUpdatedContent(page);
    return Date.now() - start;
  });

  return { label, updateTime };
};

const measureClear = async (page: Page, url: string) => {
  return measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-render');
    await page.click('#btn-render');
    await waitForRenderedList(page);

    const start = Date.now();
    await page.click('#btn-clear');
    await page.waitForFunction(() => document.querySelector('#core-count')?.textContent === 'Core Count: 0');
    return Date.now() - start;
  });
};

const measureDeepTree = async (page: Page, url: string) => {
  return measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-render');

    const start = Date.now();
    await page.click('#btn-deep');
    await page.waitForSelector('#deep-leaf');
    return Date.now() - start;
  });
};

const measureHooks = async (page: Page, url: string) => {
  return measureLoops(async () => {
    await page.goto(url);
    await page.waitForSelector('#btn-render');

    const start = Date.now();
    await page.click('#btn-hooks');
    await page.waitForSelector('#hooks-container');
    return Date.now() - start;
  });
};

const runSecondaryBenchmark = async (page: Page, url: string, label: string) => {
  const clearTime = await measureClear(page, url);
  const deepTreeTime = await measureDeepTree(page, url);
  const hooksTime = await measureHooks(page, url);

  return { label, clearTime, deepTreeTime, hooksTime };
};

const runContentBenchmark = async (page: Page, url: string, label: string) => {
  const renderResults = await measureContentRender(page, url, label);
  const updateResults = await measureContentUpdate(page, url, label);
  return { label, renderTime: renderResults.renderTime, updateTime: updateResults.updateTime };
};

test.describe('Performance Comparison', () => {
  test.describe.configure({ mode: 'serial' });

  test('Compare GoWebComponents vs React render', async ({ page }) => {
    test.setTimeout(120_000);

    const goResults = await measureRender(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await measureRender(page, '/react/index.html', 'React');

    console.log(`\nCore Render Comparison (Average of ${MEASUREMENT_LOOPS} loops, lower is better):`);
    console.table([goResults, reactResults]);

    expect(goResults.renderTime).toBeGreaterThan(0);
    expect(reactResults.renderTime).toBeGreaterThan(0);
  });

  test('Compare GoWebComponents vs React update', async ({ page }) => {
    test.setTimeout(120_000);

    const goResults = await measureUpdate(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await measureUpdate(page, '/react/index.html', 'React');

    console.log(`\nCore Update Comparison (Average of ${MEASUREMENT_LOOPS} loops, lower is better):`);
    console.table([goResults, reactResults]);

    expect(goResults.updateTime).toBeGreaterThan(0);
    expect(reactResults.updateTime).toBeGreaterThan(0);
  });

  test('Compare GoWebComponents vs React content scenarios', async ({ page }) => {
    test.setTimeout(180_000);

    const goResults = await runContentBenchmark(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await runContentBenchmark(page, '/react/index.html', 'React');

    console.log(`\nContent Comparison (Average of ${MEASUREMENT_LOOPS} loops, lower is better):`);
    console.table([goResults, reactResults]);

    expect(goResults.renderTime).toBeGreaterThan(0);
    expect(goResults.updateTime).toBeGreaterThan(0);
    expect(reactResults.renderTime).toBeGreaterThan(0);
    expect(reactResults.updateTime).toBeGreaterThan(0);
  });

  test('Compare GoWebComponents vs React secondary scenarios', async ({ page }) => {
    test.setTimeout(180_000);

    const goResults = await runSecondaryBenchmark(page, '/benchmark.html', 'GoWebComponents');
    const reactResults = await runSecondaryBenchmark(page, '/react/index.html', 'React');

    console.log(`\nSecondary Comparison (Average of ${MEASUREMENT_LOOPS} loops, lower is better):`);
    console.table([goResults, reactResults]);

    expect(goResults.clearTime).toBeGreaterThanOrEqual(0);
    expect(reactResults.clearTime).toBeGreaterThanOrEqual(0);
  });
});
