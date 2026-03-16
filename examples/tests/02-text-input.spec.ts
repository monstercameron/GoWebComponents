import { test, expect } from '@playwright/test';

test.describe('02-Text Input', () => {
  test('should update text on input', async ({ page }) => {
    await page.goto('/02-text-input/text-input.html');
    
    // Wait for load
    await expect(page.getByText('Text Input Example')).toBeVisible();
    
    const input = page.getByPlaceholder('Enter text here...');
    const countBadges = page.locator('span.font-mono.font-bold');
    
    // Initial state check
    await expect(page.getByText('Live Preview')).toBeVisible();
    await expect(page.getByText('Debounced preview', { exact: true })).toBeVisible();
    await expect(page.getByText('Debounced value settled', { exact: true })).toBeVisible();
    await expect(page.getByText('Count is synced', { exact: true })).toBeVisible();
    await expect(countBadges.nth(0)).toHaveText('0');
    await expect(countBadges.nth(1)).toHaveText('0');
    
    // Type text
    await input.click();
    await input.type('Hello World', { delay: 20 });
    
    // Verify display updates
    await expect(page.getByText('Hello World', { exact: true })).toHaveCount(2);
    await expect(countBadges.nth(0)).toHaveText('11');
    await expect(page.getByText('Waiting for debounce window', { exact: true })).toBeVisible();
    await expect(page.getByText('Debounced value settled', { exact: true })).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Count is synced', { exact: true })).toBeVisible({ timeout: 3000 });
    await expect(countBadges.nth(1)).toHaveText('11', { timeout: 3000 });
    
    // Clear text using the button
    await page.getByRole('button', { name: 'Clear Text' }).click();
    
    // Verify cleared
    await expect(input).toHaveValue('');
    await expect(page.getByText('Debounced value settled', { exact: true })).toBeVisible();
    await expect(page.getByText('Count is synced', { exact: true })).toBeVisible();
    await expect(countBadges.nth(0)).toHaveText('0');
    await expect(countBadges.nth(1)).toHaveText('0');
  });
});
