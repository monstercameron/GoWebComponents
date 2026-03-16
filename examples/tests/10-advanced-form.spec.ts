import { test, expect } from '@playwright/test';

test.describe('10-Advanced Form', () => {
  test('should validate and submit registration form', async ({ page }) => {
    await page.goto('/10-advanced-form/advanced-form.html');

    await expect(page.locator('input[name="username"]')).toBeVisible({ timeout: 15000 });

    await page.locator('input[name="username"]').fill('ab');
    await page.locator('input[name="email"]').fill('invalid');
    await page.locator('input[name="password"]').fill('123');
    await page.locator('input[name="confirm_password"]').fill('456');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Username must be at least 3 characters', { exact: true })).toBeVisible();
    await expect(page.getByText('Please enter a valid email', { exact: true })).toBeVisible();
    await expect(page.getByText('Password must be at least 6 characters', { exact: true })).toBeVisible();
    await expect(page.getByText('Passwords do not match', { exact: true })).toBeVisible();

    await page.locator('input[name="username"]').fill('admin');
    await page.locator('input[name="email"]').fill('cam@blocked.test');
    await page.locator('input[name="password"]').fill('secret123');
    await page.locator('input[name="confirm_password"]').fill('secret123');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Running async validation', { exact: true })).toBeVisible();
    await expect(page.getByText('Username is reserved', { exact: true })).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Registrations from blocked.test are disabled', { exact: true })).toBeVisible({ timeout: 3000 });

    await page.locator('input[name="username"]').fill('retryuser');
    await page.locator('input[name="email"]').fill('cam@retry.test');
    await page.locator('input[name="password"]').fill('secret123');
    await page.locator('input[name="confirm_password"]').fill('secret123');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Temporary signup outage. Please retry.', { exact: true })).toBeVisible({ timeout: 3000 });
    await expect(page.getByRole('button', { name: 'Retry Sign Up' })).toBeVisible();

    await page.locator('input[name="username"]').fill('monstercam');
    await page.locator('input[name="email"]').fill('cam@example.com');
    await page.locator('input[name="password"]').fill('secret123');
    await page.locator('input[name="confirm_password"]').fill('secret123');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Registration Successful!', { exact: true })).toBeVisible();
    await expect(page.getByText('Welcome aboard, monstercam', { exact: true })).toBeVisible();
    await expect(page).toHaveURL(/advanced-form\.html#\/success$/);

    await page.getByRole('button', { name: 'Register Another Account' }).click();
    await expect(page.locator('input[name="username"]')).toBeVisible();
    await expect(page).toHaveURL(/advanced-form\.html#\/$/);
  });
});