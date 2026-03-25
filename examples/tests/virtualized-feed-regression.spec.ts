import { expect, test } from '@playwright/test';

test('virtualized row-local state does not bleed to unrelated items after scroll reuse', async ({ page }) => {
  const response = await page.goto('/103-virtualized-feed/virtualized-feed.html', { waitUntil: 'domcontentloaded' });
  expect(response?.ok(), 'virtualized feed example failed to load').toBeTruthy();

  await expect(page.getByRole('heading', { name: 'virtualization.List' })).toBeVisible();

  const pitfall = page.locator('#virtualized-feed-pitfall');
  await expect(pitfall).toBeVisible();

  const firstRow = pitfall.locator('label').first();
  await expect(firstRow).toContainText('Warn signal 001');
  const firstCheckbox = firstRow.locator('input[type="checkbox"]');
  await firstCheckbox.check();
  await expect(firstCheckbox).toBeChecked();

  await pitfall.evaluate((element: HTMLElement) => {
    element.scrollTop = 2400;
    element.dispatchEvent(new Event('scroll'));
  });
  await page.waitForTimeout(100);

  await expect(pitfall.locator('label').first()).not.toContainText('Warn signal 001');
  await expect(pitfall.locator('input[type="checkbox"]:checked')).toHaveCount(0);

  await pitfall.evaluate((element: HTMLElement) => {
    element.scrollTop = 0;
    element.dispatchEvent(new Event('scroll'));
  });
  await page.waitForTimeout(100);

  await expect(pitfall.locator('label').first()).toContainText('Warn signal 001');
  await expect(pitfall.locator('label').first().locator('input[type="checkbox"]')).not.toBeChecked();
});
