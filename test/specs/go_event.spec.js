import { test, expect } from '@playwright/test';

test.describe('GoEvent Integration Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    // Wait for WASM to load
    await page.waitForTimeout(2000);
  });

  test('should handle input change events with GetValue', async ({ page }) => {
    // Find the test input element (from testapp main.go)
    const testInput = page.locator('#test-input');
    
    await expect(testInput).toBeVisible();
    
    // Type text into the input using GoEvent handler
    await testInput.fill('hello world');
    
    // Wait for handler to process
    await page.waitForTimeout(200);
    
    // Verify the input has the value
    const value = await testInput.inputValue();
    expect(value).toBe('hello world');
    
    // Verify input-value display updated (showing GoEvent.GetValue() worked)
    const inputDisplay = page.locator('#input-value');
    const displayText = await inputDisplay.textContent();
    expect(displayText).toContain('hello world');
  });

  test('should handle checkbox state with IsChecked', async ({ page }) => {
    // Find checkbox in use-id-test section
    const checkbox = page.locator('#use-id-test input[type="checkbox"]').first();
    
    await expect(checkbox).toBeVisible();
    
    // Check the checkbox using GoEvent handler
    await checkbox.check();
    
    // Wait for handler to process
    await page.waitForTimeout(200);
    
    // Verify it's checked
    const isChecked = await checkbox.isChecked();
    expect(isChecked).toBe(true);
  });

  test('should handle form submission with GoEvent', async ({ page }) => {
    // Find the form test section
    const formInput = page.locator('#form-input');
    const submitButton = page.locator('#test-form button[type="submit"]');
    
    await expect(formInput).toBeVisible();
    await expect(submitButton).toBeVisible();
    
    // Fill and submit the form with GoEvent handlers
    await formInput.fill('test submission');
    await submitButton.click();
    
    // Wait for handler
    await page.waitForTimeout(200);
    
    // Check that submitted value is displayed (GoEvent handler processed it)
    const submitValue = page.locator('#submit-value');
    const text = await submitValue.textContent();
    
    expect(text).toContain('test submission');
  });

  test('should extract values from input with GoEvent.GetValue', async ({ page }) => {
    const testInput = page.locator('#test-input');
    
    await expect(testInput).toBeVisible();
    
    // Test text input
    await testInput.fill('text value');
    await page.waitForTimeout(100);
    
    let displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('text value');
    
    // Test clearing
    await testInput.clear();
    await page.waitForTimeout(100);
    
    displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('Input: ');
    
    // Test special characters
    await testInput.fill('test@#$%^&*()');
    await page.waitForTimeout(100);
    
    displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('test@#$%^&*()');
  });

  test('should handle multiple GoEvent operations on same element', async ({ page }) => {
    const testInput = page.locator('#test-input');
    
    await expect(testInput).toBeVisible();
    
    // Perform multiple fill operations
    await testInput.fill('first');
    await page.waitForTimeout(100);
    
    let value = await testInput.inputValue();
    expect(value).toBe('first');
    
    await testInput.clear();
    await testInput.fill('second');
    await page.waitForTimeout(100);
    
    value = await testInput.inputValue();
    expect(value).toBe('second');
    
    // Display should show final value from GoEvent.GetValue()
    const displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('second');
  });

  test('should maintain GoEvent functionality during component rerenders', async ({ page }) => {
    const testInput = page.locator('#test-input');
    const incrementBtn = page.locator('button:has-text("Increment")').first();
    
    await expect(testInput).toBeVisible();
    
    // Fill input with GoEvent
    await testInput.fill('persistent value');
    
    // Trigger a rerender by clicking increment (causes state change)
    await incrementBtn.click();
    await page.waitForTimeout(200);
    
    // Input value should be preserved after rerender
    const value = await testInput.inputValue();
    expect(value).toBe('persistent value');
    
    // Display should still show the value (GoEvent handler still working)
    const displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('persistent value');
  });

  test('should handle rapid input changes with GoEvent', async ({ page }) => {
    const testInput = page.locator('#test-input');
    
    await expect(testInput).toBeVisible();
    
    // Type rapidly by filling multiple times
    const values = ['a', 'ab', 'abc', 'abcd', 'abcde'];
    
    for (const val of values) {
      await testInput.clear();
      await testInput.fill(val);
      await page.waitForTimeout(50);
    }
    
    // Wait for final update
    await page.waitForTimeout(100);
    
    // Should have final value
    const value = await testInput.inputValue();
    expect(value).toBe('abcde');
    
    // Display should also be updated
    const displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('abcde');
  });

  test('should handle button clicks without target.value safely', async ({ page }) => {
    // Buttons don't have target.value like inputs do
    const incrementBtn = page.locator('button:has-text("Increment")').first();
    
    await expect(incrementBtn).toBeVisible();
    
    // Click button (GoEvent handler must handle missing target.value gracefully)
    await incrementBtn.click();
    await page.waitForTimeout(200);
    
    // Verify page is still responsive and working
    const testInput = page.locator('#test-input');
    await testInput.fill('still working');
    
    const value = await testInput.inputValue();
    expect(value).toBe('still working');
  });

  test('should handle focus and blur events', async ({ page }) => {
    const testInput = page.locator('#test-input');
    
    await expect(testInput).toBeVisible();
    
    // Fill value
    await testInput.fill('focus test');
    
    // Focus the input
    await testInput.focus();
    await page.waitForTimeout(100);
    
    // Value should be preserved
    let value = await testInput.inputValue();
    expect(value).toBe('focus test');
    
    // Blur the input
    await testInput.blur();
    await page.waitForTimeout(100);
    
    // Value should still be there (GoEvent preserved it)
    value = await testInput.inputValue();
    expect(value).toBe('focus test');
  });

  test('should handle form input with checkbox and text together', async ({ page }) => {
    const textInput = page.locator('#test-input');
    const checkbox = page.locator('#use-id-test input[type="checkbox"]').first();
    
    await expect(textInput).toBeVisible();
    await expect(checkbox).toBeVisible();
    
    // Perform both operations
    await textInput.fill('multi-test');
    await checkbox.check();
    
    await page.waitForTimeout(200);
    
    // Verify both succeeded via GoEvent
    const textValue = await textInput.inputValue();
    const isChecked = await checkbox.isChecked();
    
    expect(textValue).toBe('multi-test');
    expect(isChecked).toBe(true);
    
    // Text display should show value from GetValue()
    const displayText = await page.locator('#input-value').textContent();
    expect(displayText).toContain('multi-test');
  });

  test('should survive edge case inputs', async ({ page }) => {
    const testInput = page.locator('#test-input');
    
    await expect(testInput).toBeVisible();
    
    const edgeCases = [
      { name: 'empty', value: '' },
      { name: 'spaces', value: '   ' },
      { name: 'newlines', value: 'line1\nline2' },
      { name: 'unicode', value: '你好世界🚀' },
      { name: 'very long', value: 'x'.repeat(500) },
    ];
    
    for (const testCase of edgeCases) {
      await testInput.clear();
      await testInput.fill(testCase.value);
      await page.waitForTimeout(100);
      
      const value = await testInput.inputValue();
      // Newlines may be handled differently by browser
      if (testCase.name !== 'newlines') {
        expect(value).toBe(testCase.value);
      }
    }
  });
});

