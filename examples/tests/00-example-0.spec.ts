import { expect, test } from '@playwright/test';

async function gotoExample0(page: Parameters<typeof test>[1] extends never ? never : any) {
  const candidates = [
    '/00-example-0/example-0.html',
    '/examples/00-example-0/example-0.html',
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

function captureRuntimeFailures(page: Parameters<typeof test>[1] extends never ? never : any) {
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];
  const requestFailures: string[] = [];

  page.on('pageerror', (error: Error) => {
    pageErrors.push(error.message);
  });

  page.on('console', (message: { type(): string; text(): string }) => {
    if (message.type() === 'error') {
      consoleErrors.push(message.text());
    }
  });

  page.on('requestfailed', (request: { method(): string; url(): string; failure(): { errorText?: string } | null }) => {
    requestFailures.push(`${request.method()} ${request.url()} :: ${request.failure()?.errorText ?? 'failed'}`);
  });

  return { pageErrors, consoleErrors, requestFailures };
}

function expectNoRuntimeFailures(failures: ReturnType<typeof captureRuntimeFailures>) {
  const runtimeFailures = [
    ...failures.pageErrors.map((message) => `pageerror: ${message}`),
    ...failures.consoleErrors.map((message) => `console: ${message}`),
    ...failures.requestFailures
      .filter((message) => !message.includes('GET http://127.0.0.1:8081/static/bin/example-0.wasm :: net::ERR_ABORTED'))
      .map((message) => `request: ${message}`),
  ];

  expect(runtimeFailures, 'example 0 emitted runtime failures').toEqual([]);
}

test.describe('00-example-0 catalog', () => {
  test('loads the catalog and applies filters without runtime failures', async ({ page }) => {
    const failures = captureRuntimeFailures(page);

    const { response, url } = await gotoExample0(page);
    expect(response, 'navigation should return a response for example 0').not.toBeNull();
    expect(response?.ok(), `navigation failed for ${url}`).toBeTruthy();

    const bootShell = page.locator('#boot-shell');
    await expect(bootShell).toBeVisible();
    await expect(bootShell.getByRole('heading', { name: 'Loading GoWebComponents client', exact: true })).toBeVisible();
    await expect(page.locator('#boot-status')).toContainText(/Requesting the WebAssembly binary|Preparing WebAssembly runtime|Downloading WebAssembly binary|Compiling and instantiating WebAssembly|Starting the GoWebComponents client/, { timeout: 5000 });
    await expect(page.locator('#boot-phase')).toContainText(/Requesting|Preparing|Downloading|Compiling|Starting/, { timeout: 5000 });

    await expect(page.getByRole('heading', { name: 'GoWebComponents docs, APIs, and live wasm examples', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(bootShell).toHaveAttribute('data-state', 'ready');
    await expect(page.getByText('12 results', { exact: false })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('button', { name: /Start With GoWebComponents/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /Atlas Commerce OS/ })).toBeVisible();

    await page.getByRole('button', { name: 'Example', exact: true }).click();
    await expect(page.getByText('4 results', { exact: false })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /Go Counter Demo/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /Atlas Commerce OS/ })).toBeVisible();

    await page.getByRole('combobox', { name: 'Status' }).selectOption('deprecated');
    await expect(page.getByText('1 results', { exact: false })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /Atlas Commerce OS/ })).toBeVisible();

    await page.getByRole('combobox', { name: 'Module' }).selectOption('commerce');
    await expect(page.getByText('1 results', { exact: false })).toBeVisible({ timeout: 5000 });

    await page.getByRole('textbox', { name: 'Search concepts, APIs, examples...' }).fill('atlas');
    await expect(page.getByText('1 results', { exact: false })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /Atlas Commerce OS/ })).toBeVisible();

    await page.getByRole('button', { name: /Atlas Commerce OS/ }).click();
    const detailPanel = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Atlas Commerce OS', exact: true }) });
    await expect(detailPanel.getByRole('heading', { name: 'Atlas Commerce OS', exact: true })).toBeVisible({ timeout: 5000 });
    await expect(detailPanel.getByText('The production-shaped reference app for SSR, hydration, routes, mutations, and reviewer-facing docs.', { exact: true })).toBeVisible();

    expectNoRuntimeFailures(failures);
  });

  test('smoke tests the demo surface inside #demo', async ({ page }) => {
    const failures = captureRuntimeFailures(page);

    const { response, url } = await gotoExample0(page);
    expect(response, 'navigation should return a response for example 0').not.toBeNull();
    expect(response?.ok(), `navigation failed for ${url}`).toBeTruthy();

    await expect(page.getByRole('heading', { name: 'GoWebComponents docs, APIs, and live wasm examples', exact: true })).toBeVisible({ timeout: 15000 });

    await page.getByRole('button', { name: 'Example', exact: true }).click();
    await expect(page.getByText('4 results', { exact: false })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /Go Counter Demo/ }).click();

    const demoRoot = page.locator('#demo');
    await expect(demoRoot).toBeVisible();
    await expect(demoRoot.getByText('Interactive example', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('Live widget', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('0', { exact: true })).toBeVisible();

    await demoRoot.getByRole('button', { name: 'Increment', exact: true }).click();
    await expect(demoRoot.getByText('1', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('State tone: Positive', { exact: true })).toBeVisible();

    await demoRoot.getByRole('button', { name: 'Decrement', exact: true }).click();
    await expect(demoRoot.getByText('0', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('State tone: Ready', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: 'All', exact: true }).click();
    await expect(page.getByText('12 results', { exact: false })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /ui.UseState and Local State/ }).click();
    await expect(demoRoot.getByText('API reference', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('Structured documentation', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('Signature', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: /Start With GoWebComponents/ }).click();
    await expect(demoRoot.getByText('Concept article', { exact: true })).toBeVisible();
    await expect(demoRoot.getByText('Markdown-style write-up', { exact: true })).toBeVisible();

    expectNoRuntimeFailures(failures);
  });
});