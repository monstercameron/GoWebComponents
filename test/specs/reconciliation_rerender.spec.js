import { test, expect } from '@playwright/test';

test.describe('Reconciliation Re-render Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    // Wait for WASM to load
    await page.waitForTimeout(2000);
  });

  test('should not duplicate components on re-render', async ({ page }) => {
    // Get the initial Statistics section
    const statsHeadings = page.locator('h3').filter({ hasText: 'Statistics' });
    const initialCount = await statsHeadings.count();
    
    expect(initialCount).toBe(1, 'Should have exactly 1 Statistics heading on initial render');
    
    // Interact with the app to trigger a re-render (add a todo)
    const textInput = page.locator('input[placeholder="What needs to be done?"]');
    await textInput.fill('Test todo');
    
    const addButton = page.locator('button:has-text("Add Todo")');
    await addButton.click();
    
    // Wait for re-render
    await page.waitForTimeout(500);
    
    // Check Statistics section again - should still be 1, not 2
    const statsHeadingsAfter = page.locator('h3').filter({ hasText: 'Statistics' });
    const countAfter = await statsHeadingsAfter.count();
    
    expect(countAfter).toBe(1, `Should still have exactly 1 Statistics heading after re-render, but got ${countAfter}`);
  });

  test('should not duplicate "No todos" message on re-render', async ({ page }) => {
    // Get initial count of "No todos match" messages
    const noTodosMessages = page.locator('p').filter({ hasText: 'No todos match the current filters' });
    const initialCount = await noTodosMessages.count();
    
    expect(initialCount).toBe(1, 'Should have exactly 1 "No todos" message on initial render');
    
    // Trigger filter change to cause re-render
    const filterSelect = page.locator('select').first();
    await filterSelect.selectOption('active');
    
    // Wait for re-render
    await page.waitForTimeout(500);
    
    // Check message count again
    const noTodosMessagesAfter = page.locator('p').filter({ hasText: 'No todos match the current filters' });
    const countAfter = await noTodosMessagesAfter.count();
    
    expect(countAfter).toBe(1, `Should still have exactly 1 "No todos" message after filter change, but got ${countAfter}`);
  });

  test('should maintain correct DOM structure through multiple re-renders', async ({ page }) => {
    // Count form inputs initially
    const formInputs = page.locator('form input[type="text"], form select, form input[type="date"]');
    const initialInputCount = await formInputs.count();
    
    // Add several todos to trigger multiple re-renders
    for (let i = 0; i < 3; i++) {
      const textInput = page.locator('input[placeholder="What needs to be done?"]');
      await textInput.fill(`Todo ${i + 1}`);
      
      const addButton = page.locator('button:has-text("Add Todo")');
      await addButton.click();
      
      await page.waitForTimeout(300);
    }
    
    // Check that form inputs count hasn't increased
    const formInputsAfter = page.locator('form input[type="text"], form select, form input[type="date"]');
    const finalInputCount = await formInputsAfter.count();
    
    expect(finalInputCount).toBe(initialInputCount, 
      `Form input count should remain stable: expected ${initialInputCount}, got ${finalInputCount}`);
  });

  test('should have consistent child count after filter changes', async ({ page }) => {
    // Add a todo first
    const textInput = page.locator('input[placeholder="What needs to be done?"]');
    await textInput.fill('Test todo');
    
    const addButton = page.locator('button:has-text("Add Todo")');
    await addButton.click();
    await page.waitForTimeout(500);
    
    // Count todos in list
    const todoItems = page.locator('ul li');
    const initialTodoCount = await todoItems.count();
    
    // Change status filter
    const statusSelect = page.locator('select').first(); // Status filter
    await statusSelect.selectOption('active');
    await page.waitForTimeout(500);
    
    // Count todos again
    const todoItemsAfter = page.locator('ul li');
    const todoCountAfter = await todoItemsAfter.count();
    
    expect(todoCountAfter).toBe(initialTodoCount, 
      `Todo list should show same count after filter: expected ${initialTodoCount}, got ${todoCountAfter}`);
  });
});
