import { test, expect } from '@playwright/test';

test.describe('07-Goroutines', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    await page.goto('/07-goroutines/goroutines.html');
    await expect(page.locator('h2')).toContainText('Goroutine Example');
  });

  test('should run background task', async ({ page }) => {
    const startButton = page.getByRole('button', { name: 'Start Task' });
    const progressBar = page.locator('.w-full.bg-black\\/30 > div');
    const statusText = page.getByText('Status:').first();

    await expect(statusText).toContainText('Ready');
    await startButton.click();

    await expect(statusText).toContainText(/Starting|Processing/);
    await expect(statusText).toContainText('Processing...');

    // Wait for completion (it takes about 5 seconds: 10 steps * 500ms)
    await expect(statusText).toContainText('Completed!', { timeout: 10000 });
    await expect(progressBar).toHaveAttribute('style', /width:\s*100%;?/);
  });

  test('should cancel background task', async ({ page }) => {
    const startButton = page.getByRole('button', { name: 'Start Task' });
    const cancelButton = page.getByRole('button', { name: 'Cancel Task' });
    const statusText = page.getByText('Status:').first();

    await startButton.click();
    await expect(statusText).toContainText('Processing...');
    
    await cancelButton.click();
    await expect(statusText).toContainText('Cancelled');
  });

  test('should run timer', async ({ page }) => {
    const startButton = page.getByRole('button', { name: 'Start Timer' });
    const timerDisplay = page.locator('.text-4xl.font-mono');

    await expect(timerDisplay).toHaveText('00:00');
    
    await startButton.click();
    
    const stopButton = page.getByRole('button', { name: 'Stop Timer' });
    await expect(stopButton).toBeVisible();
    
    // Wait for at least 2 seconds to pass
    await expect(timerDisplay).not.toHaveText('00:00', { timeout: 3000 });
    
    await stopButton.click();
    await expect(page.getByRole('button', { name: 'Start Timer' })).toBeVisible();
  });

  test('should reset timer', async ({ page }) => {
    const startButton = page.getByRole('button', { name: 'Start Timer' });
    const resetButton = page.getByRole('button', { name: 'Reset Timer' });
    const timerDisplay = page.locator('.text-4xl.font-mono');

    await startButton.click();
    await expect(timerDisplay).not.toHaveText('00:00', { timeout: 3000 });
    
    await resetButton.click();
    await expect(timerDisplay).toHaveText('00:00');
    await expect(page.getByRole('button', { name: 'Start Timer' })).toBeVisible();
  });
});
