import { test, expect } from '@playwright/test';

test.describe('05-Todo Basic', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/05-todo-basic/todo-basic.html');
    // Wait for app to load
    await expect(page.getByText('Todo List', { exact: true })).toBeVisible();
  });

  test('should add and remove todos', async ({ page }) => {
    const todoText = 'Buy milk';
    
    // Add todo
    await page.getByPlaceholder('Enter a new todo...').fill(todoText);
    await page.getByRole('button', { name: 'Add' }).click();
    
    // Verify todo added
    await expect(page.getByText(todoText)).toBeVisible();
    await expect(page.getByText('Tasks: 1')).toBeVisible();
    
    // Remove todo
    await page.getByRole('button', { name: 'Remove' }).click();
    
    // Verify todo removed
    await expect(page.getByText(todoText)).not.toBeVisible();
    await expect(page.getByText('Tasks: 0')).toBeVisible();
  });

  test('should clear all todos', async ({ page }) => {
    // Add multiple todos
    const todos = ['Task 1', 'Task 2', 'Task 3'];
    
    for (const todo of todos) {
      await page.getByPlaceholder('Enter a new todo...').fill(todo);
      await page.getByRole('button', { name: 'Add' }).click();
    }
    
    await expect(page.getByText('Tasks: 3')).toBeVisible();
    
    // Clear all
    await page.getByRole('button', { name: 'Clear All' }).click();
    
    // Verify all removed
    await expect(page.locator('ul[id^="todo-list-"] li')).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Clear All' })).toBeVisible();
    for (const todo of todos) {
      await expect(page.getByText(todo)).not.toBeVisible();
    }
  });
});
