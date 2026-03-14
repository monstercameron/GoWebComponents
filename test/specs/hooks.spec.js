import { test, expect } from '@playwright/test';

function mainIncrementButton(page) {
  return page.getByRole('button', { name: 'Increment main' });
}

test.describe('GoWebComponents Hooks - UseState', () => {
  test('UseState initializes with correct value', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check initial count is 0
    const countText = await page.locator('[data-testid="count-display"]').textContent();
    expect(countText).toContain('Count: 0');
  });

  test('UseState updates on button click', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Initial value
    let countText = await page.locator('[data-testid="count-display"]').textContent();
    expect(countText).toContain('Count: 0');
    
    // Click increment button
    await mainIncrementButton(page).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');
    
    // Check updated value
    countText = await page.locator('[data-testid="count-display"]').textContent();
    expect(countText).toContain('Count: 1');
  });

  test('UseState supports multiple clicks', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = mainIncrementButton(page);
    
    // Click 10 times
    for (let i = 0; i < 10; i++) {
      await button.click();
      await page.waitForTimeout(50);
    }
    
    // Wait for all renders to complete (requestAnimationFrame batching)
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 10');
    
    // Check final value
    const countText = await page.locator('[data-testid="count-display"]').textContent();
    expect(countText).toContain('Count: 10');
  });

  test('UseState triggers re-render', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const paragraph = page.locator('[data-testid="count-display"]');
    
    // Get initial text
    const initialText = await paragraph.textContent();
    
    // Click to update state
    await mainIncrementButton(page).click();
    await expect(paragraph).toHaveText('Count: 1');
    
    // Text should have changed
    const updatedText = await paragraph.textContent();
    expect(updatedText).not.toBe(initialText);
  });
});

test.describe('GoWebComponents Hooks - UseEffect', () => {
  test('UseEffect runs on mount', async ({ page }) => {
    const logs = [];
    page.on('console', msg => logs.push(msg.text()));
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    await page.waitForTimeout(500);
    
    // Verify effect ran on mount
    expect(logs.some(log => log.includes('UseEffect ran'))).toBeTruthy();
  });

  test('UseEffect cleanup runs on unmount', async ({ page }) => {
    const logs = [];
    page.on('console', (msg) => logs.push(msg.text()));
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    const toggle = page.locator('#toggle-child-btn').first();
    const cleanup = page.locator('#cleanup-status').first();

    // Initially empty
    await expect(cleanup).toHaveText('');

    // Toggle to mount
    await toggle.click();
    await page.waitForSelector('#effect-child', { timeout: 30000 });
    await page.waitForTimeout(300);
    await expect(cleanup).toHaveText('mounted');
    await page.waitForTimeout(50);
    expect(logs.some(l => l.includes('EffectChild mounted'))).toBeTruthy();

    // Toggle again to unmount
    await toggle.click();
    await page.waitForTimeout(300);
    await expect(cleanup).toHaveText('cleaned');
    await page.waitForTimeout(50);
    expect(logs.some(l => l.includes('EffectChild cleaned up'))).toBeTruthy();
  });

  test('UseEffect re-runs when dependencies change', async ({ page }) => {
    const logs = [];
    page.on('console', msg => {
      if (msg.text().includes('UseEffect ran')) {
        logs.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    await page.waitForTimeout(200);
    
    // Should have run once on mount
    const initialCount = logs.filter(log => log.includes('UseEffect ran')).length;
    expect(initialCount).toBe(1);
    
    // Click to change count (dependency)
    await mainIncrementButton(page).click();
    await expect.poll(() => logs.filter(log => log.includes('UseEffect ran')).length).toBe(2);
    
    // Effect should have run again due to count change
    const afterClickCount = logs.filter(log => log.includes('UseEffect ran')).length;
    expect(afterClickCount).toBe(2);
  });
});

test.describe('GoWebComponents Hooks - UseMemo', () => {
  test('UseMemo caches computed values', async ({ page }) => {
    const logs = [];
    page.on('console', msg => {
      if (msg.text().includes('UseMemo computing')) {
        logs.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    await page.waitForTimeout(200);
    
    // Should compute on initial mount (framework performs initial render)
    const initialComputeCount = logs.filter(log => log.includes('UseMemo computing')).length;
    expect(initialComputeCount).toBeGreaterThan(0);
    
    // Click to trigger re-render
    await mainIncrementButton(page).click();
    await expect(page.locator('#doubled')).toHaveText('Doubled: 2');
    
    // Should compute again because count changed (dependency)
    const afterClickCount = logs.filter(log => log.includes('UseMemo computing')).length;
    expect(afterClickCount).toBeGreaterThan(initialComputeCount);
    
    // Verify doubled value is correct
    const doubledText = await page.locator('#doubled').textContent();
    expect(doubledText).toContain('Doubled: 2');
  });

  test('UseMemo recomputes when dependencies change', async ({ page }) => {
    const logs = [];
    page.on('console', msg => {
      if (msg.text().includes('UseMemo computing')) {
        logs.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    await page.waitForTimeout(200);
    
    const initialComputeCount = logs.length;
    
    // Multiple clicks with proper waiting
    for (let i = 0; i < 3; i++) {
      await mainIncrementButton(page).click();
    }

    await expect(page.locator('#doubled')).toHaveText('Doubled: 6');
    
    // Should have recomputed 3 more times (once per click)
    expect(logs.length).toBe(initialComputeCount + 3);
    
    // Final value should be correct
    const doubledText = await page.locator('#doubled').textContent();
    expect(doubledText).toContain('Doubled: 6');
  });
});

test.describe('GoWebComponents State - UseAtom', () => {
  test('UseAtom provides global state', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    const aText = await page.locator('#atom-value-a').textContent();
    const bText = await page.locator('#atom-value-b').textContent();
    expect(aText).toContain('AtomA: 0');
    expect(bText).toContain('AtomB: 0');
  });

  test('UseAtom updates sync across components', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    const incButton = page.locator('#atom-increment');
    await incButton.click();
    await page.waitForTimeout(200);

    const aText = await page.locator('#atom-value-a').textContent();
    const bText = await page.locator('#atom-value-b').textContent();
    expect(aText).toContain('AtomA: 1');
    expect(bText).toContain('AtomB: 1');
  });
});
