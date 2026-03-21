import type { Browser, BrowserContext, Page } from '@playwright/test';

const examplesBaseURL = 'http://127.0.0.1:8081';

export function exampleURL(path: string): string {
  return `${examplesBaseURL}/${path.replace(/^\/+/, '')}`;
}

export async function openExamplePair(browser: Browser, path: string): Promise<{ context: BrowserContext; pageA: Page; pageB: Page }> {
  const context = await browser.newContext();
  const pageA = await context.newPage();
  const pageB = await context.newPage();
  await Promise.all([
    pageA.goto(exampleURL(path)),
    pageB.goto(exampleURL(path)),
  ]);
  return { context, pageA, pageB };
}

export async function ensureServiceWorkerReady(page: Page, serviceWorkerURL: string, scope: string): Promise<void> {
  await page.evaluate(async ({ serviceWorkerURL: url, serviceWorkerScope }) => {
    await navigator.serviceWorker.register(url, { scope: serviceWorkerScope });
    await navigator.serviceWorker.ready;
  }, { serviceWorkerURL, serviceWorkerScope: scope });

  await page.reload({ waitUntil: 'networkidle' });
  await page.waitForFunction(() => !!navigator.serviceWorker?.controller, undefined, { timeout: 15000 });
}
