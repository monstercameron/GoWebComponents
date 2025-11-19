import { test, expect } from '@playwright/test';

test.describe('02-Text Input', () => {
  test('should update text on input', async ({ page }) => {
    await page.goto('/02-text-input/text-input.html');
    
    // Wait for load
    await expect(page.getByText('Text Input Example')).toBeVisible();
    
    const input = page.getByPlaceholder('Enter text here...');
    
    // Initial state check
    await expect(page.getByText('Live Preview')).toBeVisible();
    await expect(page.getByText('...', { exact: true })).toBeVisible(); // Shows ... when empty
    await expect(page.getByText('0', { exact: true })).toBeVisible(); // Character count 0
    
    // Type text
    await input.fill('Hello World');
    
    // Verify display updates
    await expect(page.getByText('Hello World', { exact: true })).toBeVisible();
    await expect(page.getByText('11', { exact: true })).toBeVisible(); // Character count 11
    
    // Clear text using the button
    await page.getByRole('button', { name: 'Clear Text' }).click();
    
    // Verify cleared
    await expect(input).toHaveValue('');
    await expect(page.getByText('...', { exact: true })).toBeVisible();
    await expect(page.getByText('0', { exact: true })).toBeVisible();
  });
});
