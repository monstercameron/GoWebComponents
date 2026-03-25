import { expect, test } from '@playwright/test';

test('virtualized list restores the anchored row across prepend, resize, and reload', async ({ page }) => {
  const response = await page.goto('/103-virtualized-feed/virtualized-feed.html', { waitUntil: 'domcontentloaded' });
  expect(response?.ok(), 'virtualized feed example failed to load').toBeTruthy();

  await expect(page.getByRole('heading', { name: 'virtualization.List' })).toBeVisible();

  const list = page.locator('#virtualized-feed-restoration');
  await expect(list).toBeVisible();

  await list.evaluate((element: HTMLElement) => {
    element.scrollTop = 972;
    element.dispatchEvent(new Event('scroll'));
  });
  await page.waitForTimeout(100);

  const anchorTitle = page.locator('#restoration-anchor-title');
  const initialAnchor = await anchorTitle.textContent();
  expect(initialAnchor).toBeTruthy();

  await page.locator('#restoration-prepend-button').click();
  await expect(anchorTitle).toHaveText(initialAnchor!);

  await page.locator('#restoration-height-button').click();
  await expect(anchorTitle).toHaveText(initialAnchor!);

  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'virtualization.List' })).toBeVisible();
  await expect(list).toBeVisible();
  await expect(anchorTitle).toHaveText(initialAnchor!);

  const scrollTop = await list.evaluate((element: HTMLElement) => element.scrollTop);
  expect(scrollTop).toBeGreaterThan(0);
});
