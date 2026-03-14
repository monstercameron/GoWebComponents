import { test, expect } from '@playwright/test';
import { gotoApp, textNumber } from './support/app.js';

test.describe('GoWebComponents State Stress', () => {
  test.beforeEach(async ({ page }) => {
    await gotoApp(page);
  });

  test('applies 5, 25, and 100 functional transforms exactly once per burst render', async ({ page }) => {
    const initialRenders = await textNumber(page, '#stress-renders');

    await page.locator('#stress-plus-5').click();
    await expect(page.locator('#stress-count')).toHaveText('Stress Count: 5');
    expect(await textNumber(page, '#stress-renders')).toBe(initialRenders + 1);

    await page.locator('#stress-plus-25').click();
    await expect(page.locator('#stress-count')).toHaveText('Stress Count: 30');
    expect(await textNumber(page, '#stress-renders')).toBe(initialRenders + 2);

    await page.locator('#stress-plus-100').click();
    await expect(page.locator('#stress-count')).toHaveText('Stress Count: 130');
    expect(await textNumber(page, '#stress-renders')).toBe(initialRenders + 3);

    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 0');
  });

  test('survives 100 updates across repeated events and preserves state through unrelated parent rerenders', async ({ page }) => {
    const initialRenders = await textNumber(page, '#stress-renders');

    for (let i = 0; i < 20; i++) {
      await page.locator('#stress-plus-5').click();
    }

    await expect(page.locator('#stress-count')).toHaveText('Stress Count: 100');
    expect(await textNumber(page, '#stress-renders')).toBe(initialRenders + 20);

    await page.getByRole('button', { name: 'Increment main' }).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');
    await expect(page.locator('#stress-count')).toHaveText('Stress Count: 100');
    expect(await textNumber(page, '#stress-renders')).toBe(initialRenders + 20);

    await page.locator('#stress-plus-25').click();
    await expect(page.locator('#stress-count')).toHaveText('Stress Count: 125');
    expect(await textNumber(page, '#stress-renders')).toBe(initialRenders + 21);
  });

  test('mixed local and shared burst updates stay synchronized at 50 and 100 increments', async ({ page }) => {
    await page.locator('#mixed-burst-50').click();
    await expect(page.locator('#mixed-local-count')).toHaveText('Local Count: 50');
    await expect(page.locator('#mixed-shared-count')).toHaveText('Shared Count: 50');
    await expect(page.locator('#mixed-shared-mirror')).toHaveText('Mirror Shared: 50');

    await page.locator('#mixed-burst-100').click();
    await expect(page.locator('#mixed-local-count')).toHaveText('Local Count: 150');
    await expect(page.locator('#mixed-shared-count')).toHaveText('Shared Count: 150');
    await expect(page.locator('#mixed-shared-mirror')).toHaveText('Mirror Shared: 150');

    await page.locator('#mixed-reset').click();
    await expect(page.locator('#mixed-local-count')).toHaveText('Local Count: 0');
    await expect(page.locator('#mixed-shared-count')).toHaveText('Shared Count: 0');
    await expect(page.locator('#mixed-shared-mirror')).toHaveText('Mirror Shared: 0');
  });

  test('todo state remains coherent after 25 adds and 10 completion toggles', async ({ page }) => {
    for (let i = 1; i <= 25; i++) {
      await page.locator('#todo-input').fill(`todo ${i}`);
      await page.locator('#todo-add').click();
    }

    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 25');
    await expect(page.locator('#todo-count')).toHaveText('Total: 25');
    await expect(page.locator('#todo-list')).toContainText('Active: 25');
    await expect(page.locator('#todo-list')).toContainText('Completed: 0');

    const checkboxes = page.locator('#todo-items input[type="checkbox"]');
    for (let i = 0; i < 10; i++) {
      await checkboxes.nth(i).check();
    }

    await expect(page.locator('#todo-list')).toContainText('Active: 15');
    await expect(page.locator('#todo-list')).toContainText('Completed: 10');

    await page.locator('#status-filter').selectOption('completed');
    await expect(page.locator('#todo-items')).toContainText('todo 1');
    await expect(page.locator('#todo-items')).toContainText('todo 10');
    await expect(page.locator('#todo-items')).not.toContainText('todo 25');
  });
});
