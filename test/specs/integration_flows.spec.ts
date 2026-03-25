import { test, expect } from '@playwright/test';
import { gotoApp } from './support/app';

test.describe('GoWebComponents Integration Flows', () => {
  test.beforeEach(async ({ page }) => {
    await gotoApp(page);
  });

  test('completed-filter workflow stays coherent across theme toggles and fetch activity', async ({ page }) => {
    await page.locator('#todo-input').fill('first todo');
    await page.locator('#todo-add').click();
    await page.locator('#todo-input').fill('second todo');
    await page.locator('#todo-add').click();
    await page.locator('#todo-input').fill('third todo');
    await page.locator('#todo-add').click();

    await page.locator('#todo-items input[type="checkbox"]').nth(1).check();
    await expect(page.locator('#todo-list')).toContainText('Active: 2');
    await expect(page.locator('#todo-list')).toContainText('Completed: 1');

    await page.locator('#status-filter').selectOption('completed');
    await expect(page.locator('#todo-items')).toContainText('second todo');
    await expect(page.locator('#todo-items')).not.toContainText('first todo');
    await expect(page.locator('#todo-items')).not.toContainText('third todo');

    await page.getByRole('button', { name: 'Toggle Theme' }).click();
    await expect(page.locator('body')).toHaveAttribute('data-theme', 'dark');
    await expect(page.locator('#todo-items')).toContainText('second todo');

    await page.locator('#fetch-button').click();
    await expect(page.locator('#fetch-data')).toContainText('Ada Lovelace');
    await expect(page.locator('#todo-items')).toContainText('second todo');
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 3');

    await page.locator('#todo-items button[data-id="2"]').click();
    await expect(page.locator('#todo-items')).toContainText('No todos match the current filters');
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 2');
    await expect(page.locator('#todo-count')).toHaveText('Total: 2');

    await page.locator('#status-filter').selectOption('all');
    await expect(page.locator('#todo-items')).toContainText('first todo');
    await expect(page.locator('#todo-items')).toContainText('third todo');
    await expect(page.locator('#todo-items')).not.toContainText('second todo');
  });

  test('todo flow keeps list, stats, and shared header atom in sync', async ({ page }) => {
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 0');
    await expect(page.locator('#todo-count')).toHaveText('Total: 0');

    await page.locator('#todo-input').fill('buy milk');
    await page.locator('#todo-add').click();
    await expect(page.locator('#todo-text-1')).toHaveText('buy milk');

    await page.locator('#todo-input').fill('write tests');
    await page.locator('#todo-add').click();
    await expect(page.locator('#todo-text-2')).toHaveText('write tests');

    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 2');
    await expect(page.locator('#todo-count')).toHaveText('Total: 2');
    await expect(page.locator('#todo-list')).toContainText('Active: 2');
    await expect(page.locator('#todo-list')).toContainText('Completed: 0');

    await page.locator('#todo-items input[type="checkbox"]').first().check();
    await expect(page.locator('#todo-list')).toContainText('Active: 1');
    await expect(page.locator('#todo-list')).toContainText('Completed: 1');

    await page.locator('#status-filter').selectOption('completed');
    await expect(page.locator('#todo-items')).toContainText('buy milk');
    await expect(page.locator('#todo-items')).not.toContainText('write tests');

    await page.locator('#todo-filter').fill('write');
    await expect(page.locator('#todo-items')).toContainText('No todos match the current filters');

    await page.locator('#status-filter').selectOption('all');
    await page.locator('#todo-filter').fill('');
    await expect(page.locator('#todo-items')).toContainText('write tests');

    await page.locator('#todo-items button[data-id="1"]').click();
    await expect(page.locator('#todo-text-1')).toHaveCount(0);
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 1');
    await expect(page.locator('#todo-count')).toHaveText('Total: 1');
  });

  test('theme toggle and todo header share atom state without disturbing local form state', async ({ page }) => {
    await page.locator('#test-input').fill('local state');
    await expect(page.locator('#input-value')).toHaveText('Input: local state');

    await page.getByRole('button', { name: 'Toggle Theme' }).click();
    await expect(page.locator('body')).toHaveAttribute('data-theme', 'dark');

    await page.locator('#todo-input').fill('theme-safe todo');
    await page.locator('#todo-add').click();
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 1');
    await expect(page.locator('#input-value')).toHaveText('Input: local state');

    await page.getByRole('button', { name: 'Toggle Theme' }).click();
    await expect(page.locator('body')).toHaveAttribute('data-theme', 'light');
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 1');
    await expect(page.locator('#todo-text-1')).toHaveText('theme-safe todo');
  });

  test('manual fetch integrates with the running app without resetting sibling hook state', async ({ page }) => {
    const inputIdBefore = await page.locator('#use-id-test input[type="text"]').first().getAttribute('id');

    await page.locator('#fetch-button').click();
    await expect(page.locator('#fetch-data')).toContainText('"id":123');
    await expect(page.locator('#fetch-data')).toContainText('Ada Lovelace');
    await expect(page.locator('#fetch-state-display')).toContainText('Loading=false');

    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 0');
    await expect(page.locator('#input-value')).toHaveText('Input: ');

    const inputIdAfter = await page.locator('#use-id-test input[type="text"]').first().getAttribute('id');
    expect(inputIdAfter).toBe(inputIdBefore);
  });

  test('form submit preventDefault path keeps the page mounted and local state coherent', async ({ page }) => {
    await page.locator('#form-input').fill('integration payload');
    await page.locator('#test-form button[type="submit"]').click();

    await expect(page.locator('#submit-value')).toHaveText('Submitted: integration payload');
    await expect(page.locator('#main-heading')).toHaveText('GoWebComponents Test');
    await expect(page).toHaveURL(/\/$/);

    await page.getByRole('button', { name: 'Increment main' }).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');
    await expect(page.locator('#submit-value')).toHaveText('Submitted: integration payload');
  });

  test('effect cleanup and sibling state survive broader app interactions', async ({ page }) => {
    await page.locator('#todo-input').fill('cleanup safe todo');
    await page.locator('#todo-add').click();
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 1');

    await page.locator('#toggle-child-btn').click();
    await expect(page.locator('#effect-child')).toBeVisible();
    await expect(page.locator('#cleanup-status')).toHaveText('mounted');

    await page.locator('#fetch-button').click();
    await expect(page.locator('#fetch-data')).toContainText('runtime-test-user');

    await page.locator('#toggle-child-btn').click();
    await expect(page.locator('#effect-child')).toHaveCount(0);
    await expect(page.locator('#cleanup-status')).toHaveText('cleaned');

    await expect(page.locator('#todo-text-1')).toHaveText('cleanup safe todo');
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 1');
    await expect(page.locator('#fetch-data')).toContainText('Ada Lovelace');

    await page.getByRole('button', { name: 'Increment main' }).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');
    await expect(page.locator('#cleanup-status')).toHaveText('cleaned');
  });

  test('mixed shared-state bursts remain synchronized after todo, form, and theme interactions', async ({ page }) => {
    await page.locator('#mixed-burst-50').click();
    await expect(page.locator('#mixed-local-count')).toHaveText('Local Count: 50');
    await expect(page.locator('#mixed-shared-count')).toHaveText('Shared Count: 50');
    await expect(page.locator('#mixed-shared-mirror')).toHaveText('Mirror Shared: 50');

    await page.locator('#todo-input').fill('burst todo');
    await page.locator('#todo-add').click();
    await page.locator('#form-input').fill('burst payload');
    await page.locator('#test-form button[type="submit"]').click();
    await page.getByRole('button', { name: 'Toggle Theme' }).click();

    await expect(page.locator('body')).toHaveAttribute('data-theme', 'dark');
    await expect(page.locator('#submit-value')).toHaveText('Submitted: burst payload');
    await expect(page.locator('#todo-header-count')).toHaveText('Total Todos: 1');
    await expect(page.locator('#todo-text-1')).toHaveText('burst todo');

    await expect(page.locator('#mixed-local-count')).toHaveText('Local Count: 50');
    await expect(page.locator('#mixed-shared-count')).toHaveText('Shared Count: 50');
    await expect(page.locator('#mixed-shared-mirror')).toHaveText('Mirror Shared: 50');

    await page.locator('#mixed-burst-100').click();
    await expect(page.locator('#mixed-local-count')).toHaveText('Local Count: 150');
    await expect(page.locator('#mixed-shared-count')).toHaveText('Shared Count: 150');
    await expect(page.locator('#mixed-shared-mirror')).toHaveText('Mirror Shared: 150');
  });
});
