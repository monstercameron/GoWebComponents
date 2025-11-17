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
    const button = page.locator('button').first();
    const classes = await button.getAttribute('class');
    expect(classes).toBeTruthy();
  });

  test('components can be reused multiple times', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check that multiple Counter components are rendered
    const counters = await page.locator('.counter-instance').count();
    expect(counters).toBeGreaterThanOrEqual(3);
    
    // Verify each counter has independent state
    const counterA = page.locator('[data-counter-id="A"]');
    const counterB = page.locator('[data-counter-id="B"]');
    
    // Click first counter
    await counterA.locator('.counter-btn').click();
    await page.waitForTimeout(100);
    
    // Verify only first counter updated
    await expect(counterA.locator('.counter-value')).toHaveText('Counter A: 1');
    await expect(counterB.locator('.counter-value')).toHaveText('Counter B: 0');
  });
});

test.describe('GoWebComponents - Event Handling', () => {
  test('onclick events trigger handlers', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button').first();
    const counter = page.locator('#app p').first();
    const initialCount = await counter.textContent();
    
    await button.click();
    await page.waitForTimeout(100);
    
    const updatedCount = await counter.textContent();
    expect(updatedCount).not.toBe(initialCount);
  });

  test('onchange events work on inputs', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const input = page.locator('#test-input');
    const display = page.locator('#input-value');
    
    // Initial state should be empty
    await expect(display).toHaveText('Input: ');
    
    // Type into input and trigger change
    await input.fill('test value');
    await input.blur(); // Trigger onchange
    
    // Verify state updated
    await expect(display).toHaveText('Input: test value');
  });

  test('onsubmit events work on forms', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const form = page.locator('#test-form');
    const formInput = page.locator('#form-input');
    const submitDisplay = page.locator('#submit-value');
    
    // Initial state should be empty
    await expect(submitDisplay).toHaveText('Submitted: ');
    
    // Fill input and submit form
    await formInput.fill('form data');
    await form.locator('button[type="submit"]').click();
    
    // Verify submission worked
    await expect(submitDisplay).toHaveText('Submitted: form data');
  });

  test('event.preventDefault works', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const form = page.locator('#test-form');
    const formInput = page.locator('#form-input');
    const submitDisplay = page.locator('#submit-value');
    
    // Fill input
    await formInput.fill('prevented');
    
    // Submit form - if preventDefault works, page won't reload
    await form.locator('button[type="submit"]').click();
    
    // If preventDefault didn't work, the page would reload and this would fail
    await expect(submitDisplay).toHaveText('Submitted: prevented');
    
    // Verify we're still on the same page (no navigation occurred)
    expect(page.url()).toContain('/');
  });
});

test.describe('GoWebComponents - DOM Attributes', () => {
  test('class attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button').first();
    const classes = await button.getAttribute('class');
    expect(classes).toContain('bg-blue-500');
  });

  test('id attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check that id attribute works
    const heading = page.locator('#main-heading');
    await expect(heading).toBeVisible();
    const headingText = await heading.textContent();
    expect(headingText).toContain('GoWebComponents Test');
  });

  test('data attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check data-testid attribute
    const paragraph = page.locator('[data-testid="count-display"]');
    await expect(paragraph).toBeVisible();
    const text = await paragraph.textContent();
    expect(text).toContain('Count:');
  });

  test('style attributes are applied', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check inline style
    const doubled = page.locator('#doubled');
    const style = await doubled.getAttribute('style');
    expect(style).toContain('font-weight');
  });
});

test.describe('GoWebComponents - Reactivity', () => {
  test('state changes trigger re-renders', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button').first();
    const paragraph = page.locator('p').first();
    
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
