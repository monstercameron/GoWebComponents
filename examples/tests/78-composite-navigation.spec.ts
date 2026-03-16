import { expect, test } from '@playwright/test';

test.describe('78-Composite Navigation', () => {
  test('supports roving tabindex, Home/End, and active-descendant listbox navigation', async ({ page }) => {
    await page.goto('/78-composite-navigation/composite-navigation.html');

    const overviewTab = page.getByRole('tab', { name: 'Overview' });
    await overviewTab.focus();
    await page.keyboard.press('ArrowRight');
    await expect(page.getByRole('tab', { name: 'API checklist' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByRole('heading', { name: 'API checklist' })).toBeVisible();

    await page.keyboard.press('End');
    await expect(page.getByRole('tab', { name: 'Release notes' })).toHaveAttribute('aria-selected', 'true');

    const listbox = page.locator('#owner-listbox');
    await listbox.focus();
    await page.keyboard.press('p');
    await expect(page.getByRole('option', { name: 'Platform' })).toHaveAttribute('aria-selected', 'true');
    await expect(listbox).toHaveAttribute('aria-activedescendant', 'owner-platform');

    await page.keyboard.press('End');
    await expect(page.getByRole('option', { name: 'Support desk' })).toHaveAttribute('aria-selected', 'true');
    await expect(listbox).toHaveAttribute('aria-activedescendant', 'owner-support');
  });
});