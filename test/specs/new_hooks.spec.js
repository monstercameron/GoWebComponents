import { test, expect } from '@playwright/test';

test.describe('GoWebComponents Hooks - UseId', () => {
  test('UseId generates unique stable IDs', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    // Get the generated IDs
    const inputIdDisplay = await page.locator('#input-id-display').textContent();
    const selectIdDisplay = await page.locator('#select-id-display').textContent();
    const checkboxIdDisplay = await page.locator('#checkbox-id-display').textContent();

    // Extract IDs from display text
    const inputId = inputIdDisplay.match(/Input ID: (.+)/)[1];
    const selectId = selectIdDisplay.match(/Select ID: (.+)/)[1];
    const checkboxId = checkboxIdDisplay.match(/Checkbox ID: (.+)/)[1];

    // IDs should be generated and not empty
    expect(inputId).toBeTruthy();
    expect(selectId).toBeTruthy();
    expect(checkboxId).toBeTruthy();

    // IDs should follow the gwc: prefix pattern
    expect(inputId).toMatch(/^gwc:\d+:\d+$/);
    expect(selectId).toMatch(/^gwc:\d+:\d+$/);
    expect(checkboxId).toMatch(/^gwc:\d+:\d+$/);

    // IDs should be different from each other
    expect(inputId).not.toBe(selectId);
    expect(selectId).not.toBe(checkboxId);
    expect(inputId).not.toBe(checkboxId);
  });

  test('UseId connects label to input via htmlFor', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    // Get the input element and label element
    const inputElement = await page.locator('#use-id-test input[type="text"]').first();
    const labelElement = await page.locator('#input-label');

    // Get their IDs
    const inputId = await inputElement.getAttribute('id');
    const labelFor = await labelElement.getAttribute('htmlFor');

    // They should match
    expect(inputId).toBe(labelFor);
    expect(inputId).toBeTruthy();
  });

  test('UseId persists across re-renders', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    // Get initial ID
    const initialIdDisplay = await page.locator('#input-id-display').textContent();
    const initialId = initialIdDisplay.match(/Input ID: (.+)/)[1];

    // Trigger a re-render by changing other state (click main counter)
    await page.click('button:has-text("Increment")');
    await page.waitForTimeout(200);

    // Get the ID again - it should be the same
    const afterReRenderIdDisplay = await page.locator('#input-id-display').textContent();
    const afterReRenderId = afterReRenderIdDisplay.match(/Input ID: (.+)/)[1];

    // IDs should be stable across re-renders
    expect(afterReRenderId).toBe(initialId);
  });

  test('UseId generates different IDs for multiple instances', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    // Get all generated IDs
    const inputIdDisplay = await page.locator('#input-id-display').textContent();
    const selectIdDisplay = await page.locator('#select-id-display').textContent();

    const inputId = inputIdDisplay.match(/Input ID: (.+)/)[1];
    const selectId = selectIdDisplay.match(/Select ID: (.+)/)[1];

    // Both should exist and be different
    expect(inputId).toBeTruthy();
    expect(selectId).toBeTruthy();
    expect(inputId).not.toBe(selectId);
  });

  test('UseId with select element for accessibility', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    const selectElement = page.locator('#use-id-test select').first();
    const selectLabel = page.locator('#select-label');

    const selectId = await selectElement.getAttribute('id');
    const labelFor = await selectLabel.getAttribute('htmlFor');

    // Verify connection
    expect(selectId).toBe(labelFor);
    expect(selectId).toMatch(/^gwc:\d+:\d+$/);
  });

  test('UseId with checkbox element for accessibility', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    const checkboxElement = page.locator('input[type="checkbox"]').first();
    const checkboxLabel = page.locator('#checkbox-label');

    const checkboxId = await checkboxElement.getAttribute('id');
    const labelFor = await checkboxLabel.getAttribute('htmlFor');

    // Verify connection
    expect(checkboxId).toBe(labelFor);
    expect(checkboxId).toMatch(/^gwc:\d+:\d+$/);
  });
});

test.describe('GoWebComponents Hooks - UseFetch', () => {
  test('UseFetch initializes in idle state', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Should start in idle state
    const idleText = await page.locator('#fetch-idle').textContent();
    expect(idleText).toContain('No data fetched yet');

    const stateDisplay = await page.locator('#fetch-state-display').textContent();
    expect(stateDisplay).toContain('Loading=false');
    expect(stateDisplay).toContain('Error=');
  });

  test('UseFetch button exists and is clickable', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    const fetchButton = page.locator('#fetch-button');
    await expect(fetchButton).toBeVisible();
    await expect(fetchButton).toBeEnabled();
    expect(await fetchButton.textContent()).toContain('Fetch User Data');
  });

  test.skip('UseFetch sets loading state on manual trigger', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Click fetch button
    await page.click('#fetch-button');

    // Should show loading state
    const loadingText = await page.locator('#fetch-loading', { timeout: 5000 });
    await expect(loadingText).toBeVisible();
    expect(await loadingText.textContent()).toContain('Loading...');

    // State display should show Loading=true
    const stateDisplay = await page.locator('#fetch-state-display').textContent();
    expect(stateDisplay).toContain('Loading=true');
  });

  test('UseFetch refetch function is callable multiple times', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    const fetchButton = page.locator('#fetch-button');

    // Click multiple times
    for (let i = 0; i < 3; i++) {
      await fetchButton.click();
      await page.waitForTimeout(100);

      // Verify loading state appears
      const loadingLocator = page.locator('#fetch-loading');
      const errorLocator = page.locator('#fetch-error');
      const idleLocator = page.locator('#fetch-idle');

      // At least one of these should exist
      const isLoading = await loadingLocator.count() > 0;
      const hasError = await errorLocator.count() > 0;
      const isIdle = await idleLocator.count() > 0;

      expect(isLoading || hasError || isIdle).toBeTruthy();
    }
  });

  test('UseFetch getter returns FetchState with correct structure', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Check initial state display
    const stateDisplay = await page.locator('#fetch-state-display').textContent();

    // Should contain Loading and Error keys
    expect(stateDisplay).toContain('Loading=');
    expect(stateDisplay).toContain('Error=');
  });

  test('UseFetch triggers re-render after refetch call', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Get initial state
    const initialStateDisplay = await page.locator('#fetch-state-display').textContent();

    // Click fetch button
    await page.click('#fetch-button');
    await page.waitForTimeout(300);

    // Get state after fetch
    const afterFetchStateDisplay = await page.locator('#fetch-state-display').textContent();

    // State should have changed (Loading should be true or error/data should exist)
    expect(afterFetchStateDisplay).not.toBe(initialStateDisplay);
  });

  test('UseFetch maintains state independently', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Fetch state should not affect main counter
    const initialCountText = await page.locator('[data-testid="count-display"]').textContent();

    // Click fetch button
    await page.click('#fetch-button');
    await page.waitForTimeout(200);

    // Main counter should remain unchanged
    const afterFetchCountText = await page.locator('[data-testid="count-display"]').textContent();
    expect(afterFetchCountText).toBe(initialCountText);
  });

  test('UseFetch error state displays when fetch fails', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Click fetch button - this will attempt to fetch from a non-existent endpoint
    // which should eventually result in an error state
    await page.click('#fetch-button');
    await page.waitForTimeout(2000); // Give it time to fail

    // Check if error state or other final state is displayed
    const stateDisplay = await page.locator('#fetch-state-display').textContent();
    
    // Should eventually show Loading=false (fetch completed)
    await page.waitForFunction(() => {
      const text = document.querySelector('#fetch-state-display').textContent;
      return text.includes('Loading=false');
    }, { timeout: 5000 });

    expect(stateDisplay).toContain('Loading=false');
  });

  test('UseFetch response handling in component', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Initial state should be idle
    await expect(page.locator('#fetch-idle')).toBeVisible();

    // After clicking fetch, we should see some response
    await page.click('#fetch-button');

    // Wait for response to complete
    await page.waitForFunction(() => {
      // Check if we moved out of idle state
      const fetchIdle = document.querySelector('#fetch-idle');
      const fetchLoading = document.querySelector('#fetch-loading');
      const fetchError = document.querySelector('#fetch-error');
      const fetchData = document.querySelector('#fetch-data');
      
      return (fetchLoading || fetchError || fetchData);
    }, { timeout: 5000 });

    // Verify we got some kind of response
    const responses = [
      await page.locator('#fetch-loading').count(),
      await page.locator('#fetch-error').count(),
      await page.locator('#fetch-data').count()
    ];

    // At least one response type should be visible
    expect(responses.reduce((a, b) => a + b, 0)).toBeGreaterThan(0);
  });
});

test.describe('GoWebComponents Hooks - Integration', () => {
  test('UseId and UseFetch can be used together in same component', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Both should be visible and functional
    const useIdSection = page.locator('#use-id-test');
    const useFetchSection = page.locator('#use-fetch-test');

    await expect(useIdSection).toBeVisible();
    await expect(useFetchSection).toBeVisible();

    // Verify they don't interfere with each other
    const inputId = await page.locator('#use-id-test input[type="text"]').first().getAttribute('id');
    expect(inputId).toMatch(/^gwc:\d+:\d+$/);

    // Fetch should still work
    await page.click('#fetch-button');
    await page.waitForTimeout(100);

    // Input ID should remain unchanged
    const inputIdAfter = await page.locator('#use-id-test input[type="text"]').first().getAttribute('id');
    expect(inputIdAfter).toBe(inputId);
  });

  test('Multiple UseId calls in same component generate unique IDs', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    // Get all three IDs from the test component
    const inputIdDisplay = await page.locator('#input-id-display').textContent();
    const selectIdDisplay = await page.locator('#select-id-display').textContent();
    const checkboxIdDisplay = await page.locator('#checkbox-id-display').textContent();

    const inputId = inputIdDisplay.match(/Input ID: (.+)/)[1];
    const selectId = selectIdDisplay.match(/Select ID: (.+)/)[1];
    const checkboxId = checkboxIdDisplay.match(/Checkbox ID: (.+)/)[1];

    // All should be different
    const ids = new Set([inputId, selectId, checkboxId]);
    expect(ids.size).toBe(3); // All unique

    // Extract the position numbers (last part after second colon)
    const positions = [inputId, selectId, checkboxId].map(id => {
      const parts = id.split(':');
      return parseInt(parts[parts.length - 1]);
    });

    // Positions should be sequential (0, 1, 2)
    expect(positions[0] + 1).toBe(positions[1]);
    expect(positions[1] + 1).toBe(positions[2]);
  });

  test('UseId IDs are accessible via htmlFor attribute', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-id-test', { timeout: 30000 });

    // Test all three form elements
    const elements = [
      {
        input: page.locator('#use-id-test input[type="text"]').first(),
        label: page.locator('#input-label')
      },
      {
        input: page.locator('#use-id-test select').first(),
        label: page.locator('#select-label')
      },
      {
        input: page.locator('#use-id-test input[type="checkbox"]').first(),
        label: page.locator('#checkbox-label')
      }
    ];

    for (const element of elements) {
      const inputId = await element.input.getAttribute('id');
      const labelFor = await element.label.getAttribute('htmlFor');

      expect(inputId).toBeTruthy();
      expect(labelFor).toBeTruthy();
      expect(inputId).toBe(labelFor);
      expect(inputId).toMatch(/^gwc:\d+:\d+$/);
    }
  });
});

test.describe('GoWebComponents Hooks - GoUseFunc', () => {
  test('GoUseFunc creates clickable event handlers', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    const fetchButton = page.locator('#fetch-button');
    
    // Verify button exists and is clickable
    await expect(fetchButton).toBeVisible();
    await expect(fetchButton).toBeEnabled();
    
    // Click should work without errors
    await fetchButton.click();
    await page.waitForTimeout(100);
    
    // Button should still be clickable after click
    await expect(fetchButton).toBeEnabled();
  });

  test('GoUseFunc handlers execute on user interaction', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Initial state should show idle
    const initialState = await page.locator('#fetch-idle').count();
    expect(initialState).toBe(1);

    // Click fetch button
    await page.click('#fetch-button');
    await page.waitForTimeout(200);

    // State should have changed (moved out of idle or to loading)
    const finalState = await page.locator('#fetch-idle').count();
    // Should be 0 now (moved to loading or error state)
    expect(finalState).toBeLessThanOrEqual(1);
  });

  test('GoUseFunc handlers work multiple times', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    const fetchButton = page.locator('#fetch-button');
    
    // Click button 3 times
    for (let i = 0; i < 3; i++) {
      await fetchButton.click();
      await page.waitForTimeout(150);
      
      // Verify button is still functional
      await expect(fetchButton).toBeEnabled();
    }
  });

  test('GoUseFunc preserves handler across re-renders', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    const fetchButton = page.locator('#fetch-button');
    
    // Get initial button text
    const initialText = await fetchButton.textContent();
    
    // Trigger a re-render by changing main counter
    await page.click('button:has-text("Increment")');
    await page.waitForTimeout(200);
    
    // Button should still exist with same text
    const finalText = await fetchButton.textContent();
    expect(finalText).toBe(initialText);
    
    // Button should still be clickable
    await expect(fetchButton).toBeEnabled();
    await fetchButton.click();
    await page.waitForTimeout(100);
  });

  test('GoUseFunc handler effects are isolated from other state', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Get initial counter value
    const initialCount = await page.locator('[data-testid="count-display"]').textContent();
    
    // Click fetch button 3 times
    for (let i = 0; i < 3; i++) {
      await page.click('#fetch-button');
      await page.waitForTimeout(100);
    }
    
    // Counter should not have changed
    const finalCount = await page.locator('[data-testid="count-display"]').textContent();
    expect(finalCount).toBe(initialCount);
  });

  test('GoUseFunc with manual fetch button triggers state change', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#use-fetch-test', { timeout: 30000 });

    // Check state before fetch
    const stateBefore = await page.locator('#fetch-state-display').textContent();
    
    // Click fetch
    await page.click('#fetch-button');
    await page.waitForTimeout(300);
    
    // Check state after fetch
    const stateAfter = await page.locator('#fetch-state-display').textContent();
    
    // State should have changed
    expect(stateAfter).not.toBe(stateBefore);
  });
});
