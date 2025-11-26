import { test, expect } from '@playwright/test';

test('browser compiler terminal output', async ({ page }) => {
    // Capture console logs
    page.on('console', msg => console.log(`[Browser] ${msg.text()}`));

    // Navigate to the example page
    await page.goto('/13-browser-compiler/');
  
    // Wait for the terminal to be ready
    await page.waitForSelector('#terminal-input');

    // Type 'go run' in the terminal
    await page.fill('#terminal-input', 'go run');
    await page.press('#terminal-input', 'Enter');

    // Wait for the output to appear in the terminal
    // The output should contain "Hello from Browser Compiler!"
    // We might need to wait a bit for compilation
    await expect(page.locator('#console-output')).toContainText('Hello from Browser Compiler!', { timeout: 60000 });
    
    // Also verify "This code was compiled in your browser."
    await expect(page.locator('#console-output')).toContainText('This code was compiled in your browser.');
});
