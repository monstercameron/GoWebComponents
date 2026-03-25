import { expect, test } from '@playwright/test';

test('virtualized hydration lane preserves the prerendered initial window and post-hydration scrolling', async ({ browser }) => {
  const staticContext = await browser.newContext({ javaScriptEnabled: false });
  const staticPage = await staticContext.newPage();
  const staticResponse = await staticPage.goto('/103-virtualized-feed/virtualized-feed-hydration.html', { waitUntil: 'domcontentloaded' });
  expect(staticResponse?.ok(), 'static hydration page failed to load').toBeTruthy();

  await expect(staticPage.getByRole('heading', { name: 'virtualization.List hydration' })).toBeVisible();
  await expect(staticPage.locator('#hydration-rendered-range')).toHaveText('0-8');
  await expect(staticPage.locator('#virtualized-feed-hydration').locator('[key]').filter({ hasText: 'signal' })).toHaveCount(8);
  await staticContext.close();

  const context = await browser.newContext();
  const page = await context.newPage();
  const response = await page.goto('/103-virtualized-feed/virtualized-feed-hydration.html', { waitUntil: 'domcontentloaded' });
  expect(response?.ok(), 'hydrated page failed to load').toBeTruthy();

  await expect(page.getByRole('heading', { name: 'virtualization.List hydration' })).toBeVisible();
  await expect(page.locator('#hydration-visible-range')).toHaveText('0-6');
  await expect(page.locator('#hydration-rendered-range')).toHaveText('0-8');
  await expect(page.locator('#hydration-rendered-rows')).toHaveText('8');

  const list = page.locator('#virtualized-feed-hydration');
  await list.evaluate((element: HTMLElement) => {
    element.scrollTop = 972;
    element.dispatchEvent(new Event('scroll'));
  });
  await page.waitForTimeout(100);

  await expect(page.locator('#hydration-anchor-title')).toHaveText('Warn signal 019');
  await expect(page.locator('#hydration-visible-range')).toHaveText('18-24');
  await context.close();
});
