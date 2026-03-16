import { expect, test } from '@playwright/test';

test.describe('79-Form Accessibility', () => {
  test('announces validation failures, focuses the first invalid field, and reports successful submit state', async ({ page }) => {
    await page.goto('/79-form-accessibility/form-accessibility.html');

    await page.getByRole('button', { name: 'Submit review request' }).click();

    await expect(page.getByLabel('Reviewer name')).toBeFocused();
    await expect(page.getByText('Name must be at least 2 characters.')).toBeVisible();
    await expect(page.locator('[aria-live="assertive"]').first()).toContainText('Please correct');
    await expect(page.locator('#form-announcement')).toContainText('Please correct');

    await page.getByLabel('Reviewer name').fill('Ada Lovelace');
    await page.getByLabel('Notification email').fill('ada@example.com');
    await page.getByLabel('Announcement channel').selectOption('slack');
    await page.getByRole('button', { name: 'Submit review request' }).click();

    await expect(page.locator('#form-announcement')).toContainText('Submitting the accessibility review request.');
    await expect(page.locator('#form-announcement')).toContainText('Accessibility review request sent.', { timeout: 5000 });
    await expect(page.locator('[aria-live="polite"]').first()).toContainText('Accessibility review request sent.');
  });
});