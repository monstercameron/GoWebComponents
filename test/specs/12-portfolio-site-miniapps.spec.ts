import { test, expect } from '@playwright/test';

test.describe('Example 12: Portfolio Site - Mini Apps', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    page.on('pageerror', err => console.log(`BROWSER ERROR: ${err}`));
    await page.goto('/examples/12-portfolio-site/portfolio.html');
    // Wait for WASM to initialize and render
    await page.waitForTimeout(1000);
  });

  test('should flip card and show source code for Click Counter', async ({ page }) => {
    // Scroll to examples section
    const examplesSection = page.locator('#examples');
    await examplesSection.scrollIntoViewIfNeeded();

    // Find the Click Counter card
    // The card has a header with "Click Counter"
    const card = page.locator('.group', { hasText: 'Click Counter' }).first();
    await expect(card).toBeVisible();

    // Find the "View Code" button on the front side
    const viewCodeBtn = card.locator('button', { hasText: 'View Code' });
    await expect(viewCodeBtn).toBeVisible();

    // Click it
    await viewCodeBtn.click();
    
    // Wait for animation
    await page.waitForTimeout(1000);

    // Take a screenshot for verification
    await page.screenshot({ path: 'test-results/flip-verification.png' });

    // DEBUG: Dump the DOM of the card
    const cardHTML = await card.evaluate(el => el.outerHTML);
    console.log('Card HTML:', cardHTML);

    // Check if the flip container has the correct style
    // The container is the direct child of the .group div
    const flipContainer = card.locator('> div').first();
    await expect(flipContainer).toHaveAttribute('style', /rotateY\(180deg\)/);

    // Check if the back side is visible (it should be the one with rotateY(180deg) and translateZ(1px))
    // We look for the code content
    const codeBlock = card.locator('pre, div.font-mono'); // Adjust selector based on HighlightGoCode implementation
    
    // The HighlightGoCode function creates a div with class "text-xs font-mono..."
    // Let's look for some unique text from the source code
    const sourceText = 'func MiniClickCounter';
    await expect(card).toContainText(sourceText);
    
    // Verify the back face style to ensure it's not mirrored visually (proxy check)
    // The back face div should have transform: rotateY(180deg) translateZ(2px)
    const backFace = card.locator('div[style*="rotateY(180deg)"][style*="translateZ(2px)"]');
    await expect(backFace).toBeVisible();
  });

  test('should flip card and show source code for Random Number', async ({ page }) => {
    const examplesSection = page.locator('#examples');
    await examplesSection.scrollIntoViewIfNeeded();

    const card = page.locator('.group', { hasText: 'Random Number' }).first();
    const viewCodeBtn = card.locator('button', { hasText: 'View Code' });
    await viewCodeBtn.click();
    await page.waitForTimeout(800);

    const sourceText = 'func MiniRandomizer';
    await expect(card).toContainText(sourceText);
  });
});
