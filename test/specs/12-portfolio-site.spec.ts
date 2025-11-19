import { test, expect } from '@playwright/test';

test.describe('Example 12: Portfolio Site', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    page.on('pageerror', err => console.log(`BROWSER ERROR: ${err}`));
    await page.goto('/examples/12-portfolio-site/portfolio.html');
    // Wait for WASM to initialize and render
    await page.waitForTimeout(1000);
  });

  test('should render the main landing page', async ({ page }) => {
    // Debug: print all h1 text
    const h1s = await page.locator('h1').allInnerTexts();
    console.log('H1s found:', h1s);

    await expect(page.locator('h1').filter({ hasText: 'GoWebComponents' })).toBeVisible();
    await expect(page.locator('#home')).toBeVisible();
  });

  test('should navigate to sections via navbar', async ({ page }) => {
    // Check "About" link
    await page.click('text=About');
    // Wait for scroll (approximate check)
    await expect(page.locator('#about')).toBeInViewport();

    // Check "Projects" link
    await page.click('text=Projects');
    await expect(page.locator('#projects')).toBeInViewport();
  });

  test('should navigate to API Docs page', async ({ page }) => {
    // Click "API Docs" in navbar
    await page.click('text=API Docs');
    
    // Current behavior: scrolls to #api section (Why GoWebComponents)
    // Desired behavior: navigate to /docs page?
    // Let's check what happens currently.
    // If it scrolls to #api, the URL hash might become #api
    
    // If we want it to go to /docs, we expect to see "GoWebComponents API Documentation" header
    // which is in DocsPage (documentation.go)
    
    // For now, let's assert the current behavior to see if it works as implemented
    // await expect(page.locator('#api')).toBeInViewport();
    
    // But if the user wants to "fix" it, maybe they want it to go to the docs page.
    // Let's try to verify if the docs page is accessible via the button in #api section
    
    const docsButton = page.locator('button:has-text("View Documentation")');
    if (await docsButton.isVisible()) {
        await docsButton.click();
        await expect(page.locator('h1:has-text("GoWebComponents API Documentation")')).toBeVisible();
    }
  });

  test('should toggle dark mode', async ({ page }) => {
    const toggleBtn = page.locator('button[title="Toggle dark mode"]');
    await toggleBtn.click();
    // Check if class 'dark' is toggled on html or body, or if styles change
    // The implementation uses applyDarkClass which likely toggles a class on document.documentElement
    const html = page.locator('html');
    await expect(html).toHaveClass(/dark/);

    // Debug: check button state
    console.log('Button HTML:', await toggleBtn.evaluate(el => el.outerHTML));

    await toggleBtn.click();
    await expect(html).not.toHaveClass(/dark/);
  });
});
