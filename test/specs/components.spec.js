import { test, expect } from '@playwright/test';

test.describe('GoWebComponents - Component Composition', () => {
  test('components render nested children', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check that nested elements exist (use .first() to avoid strict mode violation)
    const container = page.locator('#app > div').first();
    await expect(container).toBeVisible();
    
    const heading = page.locator('h1');
    await expect(heading).toBeVisible();
  });

  test('components accept and render props', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check that props are applied (class attributes)
    const button = page.locator('button');
    const classes = await button.getAttribute('class');
    expect(classes).toBeTruthy();
  });

  test.skip('components can be reused multiple times', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test multiple instances of same component
  });
});

test.describe('GoWebComponents - Event Handling', () => {
  test('onclick events trigger handlers', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button');
    const initialCount = await page.locator('p').textContent();
    
    await button.click();
    await page.waitForTimeout(100);
    
    const updatedCount = await page.locator('p').textContent();
    expect(updatedCount).not.toBe(initialCount);
  });

  test.skip('onchange events work on inputs', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test input change events
  });

  test.skip('onsubmit events work on forms', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test form submission
  });

  test.skip('event.preventDefault works', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test preventDefault functionality
  });
});

test.describe('GoWebComponents - DOM Attributes', () => {
  test('class attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button');
    const classes = await button.getAttribute('class');
    expect(classes).toContain('bg-blue-500');
  });

  test.skip('id attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test id attributes
  });

  test.skip('data attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test data-* attributes
  });

  test.skip('style attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test inline styles
  });
});

test.describe('GoWebComponents - Reactivity', () => {
  test('state changes trigger re-renders', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button');
    const paragraph = page.locator('p');
    
    // Get initial state
    const initialText = await paragraph.textContent();
    expect(initialText).toContain('Count: 0');
    
    // Trigger state change
    await button.click();
    await page.waitForTimeout(100);
    
    // Verify re-render occurred
    const updatedText = await paragraph.textContent();
    expect(updatedText).toContain('Count: 1');
    expect(updatedText).not.toBe(initialText);
  });

  test.skip('only affected components re-render', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test selective re-rendering
  });

  test.skip('batched updates work correctly', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test update batching
  });
});
