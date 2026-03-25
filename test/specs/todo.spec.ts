import { test, expect } from '@playwright/test';

test.describe('GoWebComponents - Todo App', () => {
  test.beforeEach(async ({ page }) => {
    // capture console logs for debugging
    page.on('console', (msg) => console.log('PAGE LOG:', msg.text()));
  });
  test('can add and remove todos and header count updates', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    // Ensure header shows total 0 initially
    await page.waitForSelector('#todo-app-heading');
    let headerCount = await page.locator('#todo-header-count').textContent();
    expect(headerCount).toContain('Total Todos: 0');

    // Add a todo
    await page.fill('#todo-input', 'buy milk');
    await page.click('#todo-add');
    
    // Wait for item to appear
    await page.waitForSelector('#todo-text-1', { timeout: 5000 });
    const todoText = await page.locator('#todo-text-1').textContent();
    expect(todoText).toContain('buy milk');

    // Verify count updated (be lenient with whitespace/multiple renders)
    await page.waitForFunction(() => {
      const el = document.querySelector('#todo-count');
      return el && el.textContent && el.textContent.includes('Total: 1');
    }, { timeout: 5000 });

    // Delete the todo by clicking the delete button
    const deleteBtn = page.locator('#todo-items button').first();
    await deleteBtn.click();
    
    // Wait for the item to be removed from DOM
    await page.waitForFunction(() => {
      const el = document.querySelector('#todo-text-1');
      return !el; // Wait for element to not exist
    }, { timeout: 5000 });
    
    // Verify count is back to 0
    const countAfter = await page.locator('#todo-count').textContent();
    expect(countAfter).toContain('Total: 0');
    
    // Verify header updated
    headerCount = await page.locator('#todo-header-count').textContent();
    expect(headerCount).toContain('Total Todos: 0');
  });

  test('filter uses memoization and shows filtered list', async ({ page }) => {
    const logs = [];
    page.on('console', (msg) => {
      if (msg.text().includes('UseMemo computing: filtered')) logs.push(msg.text());
    });

    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    // Add two todos
    await page.fill('#todo-input', 'milk');
    await page.click('#todo-add');
    await page.waitForTimeout(100);
    await page.fill('#todo-input', 'eggs');
    await page.click('#todo-add');
    await page.waitForTimeout(200);

    // initial log should have recorded some memo compute
    expect(logs.length).toBeGreaterThanOrEqual(1);

    // Filter to 'milk'
    await page.fill('#todo-filter', 'milk');
    await page.dispatchEvent('#todo-filter', 'change');
    await page.waitForTimeout(150);

    await expect(page.locator('#todo-items')).toContainText('milk');
    await expect(page.locator('#todo-items')).not.toContainText('eggs');
  });

  test('toggle completion updates UI', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    // Add a todo
    await page.fill('#todo-input', 'walk dog');
    await page.click('#todo-add');
    await page.waitForTimeout(150);

    // Toggle checkbox
    await page.click('#todo-items input[type=checkbox]');
    await page.waitForTimeout(100);

    // Check checkbox is checked (the rendered input should reflect completion)
    const isChecked = await page.locator('#todo-items input[type=checkbox]').isChecked();
    expect(isChecked).toBeTruthy();
  });
});
