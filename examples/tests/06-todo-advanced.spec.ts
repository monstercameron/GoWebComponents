import { test, expect } from '@playwright/test';

test.describe('06-Todo Advanced', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    await page.goto('/06-todo-advanced/todo-advanced.html');
    // The heading is rendered by the Go app, wait for it
    // Use a more generic locator if role isn't working, or check if it's actually an h1
    await expect(page.locator('h1')).toContainText('Advanced Todo App');
  });

  test('should add complex todo', async ({ page }) => {
    // Fill form
    await page.getByPlaceholder('What needs to be done?').fill('Complex Task');
    console.log(await page.content()); // Dump content to debug
    await page.getByPlaceholder('Work, Personal...').fill('Work');
    
    // Select priority
    await page.locator('select').selectOption('high');
    
    await page.getByRole('button', { name: 'Add Todo' }).click();
    
    // Verify item appears with details
    const item = page.locator('li').filter({ hasText: 'Complex Task' });
    await expect(item).toBeVisible();
    await expect(item).toContainText('High'); // Priority badge
    await expect(item).toContainText('Work'); // Category badge
  });

  test('should filter todos', async ({ page }) => {
    // Add active task
    await page.getByPlaceholder('What needs to be done?').fill('Active Task');
    await page.getByRole('button', { name: 'Add Todo' }).click();
    
    // Add another and complete it
    await page.getByPlaceholder('What needs to be done?').fill('Completed Task');
    await page.getByRole('button', { name: 'Add Todo' }).click();
    
    const completedItem = page.locator('li').filter({ hasText: 'Completed Task' });
    // The checkbox is the first input in the li
    await completedItem.locator('input[type="checkbox"]').check();
    
    // Filter by Active
    await page.getByRole('button', { name: 'Active' }).click();
    await expect(page.getByText('Active Task')).toBeVisible();
    await expect(page.getByText('Completed Task')).not.toBeVisible();
    
    // Filter by Completed
    await page.getByRole('button', { name: 'Completed' }).click();
    await expect(page.getByText('Active Task')).not.toBeVisible();
    await expect(page.getByText('Completed Task')).toBeVisible();
    
    // Filter by All
    await page.getByRole('button', { name: 'All' }).click();
    await expect(page.getByText('Active Task')).toBeVisible();
    await expect(page.getByText('Completed Task')).toBeVisible();
  });
});
