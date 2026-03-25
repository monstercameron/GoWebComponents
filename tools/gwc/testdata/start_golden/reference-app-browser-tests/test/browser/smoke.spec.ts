import { expect, test } from '@playwright/test';

test('starter shell renders', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: /golden-reference-app/i })).toBeVisible();
});
