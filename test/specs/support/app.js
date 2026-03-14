import { expect } from '@playwright/test';

export async function gotoApp(page) {
  await page.goto('/');
  await page.waitForSelector('#main-heading', { timeout: 30000 });
  await expect(page.locator('#main-heading')).toHaveText('GoWebComponents Test');
}

export async function textNumber(page, selector) {
  const text = await page.locator(selector).textContent();
  const match = text && text.match(/-?\d+/);
  if (!match) {
    throw new Error(`No numeric value found in ${selector}: ${text}`);
  }
  return parseInt(match[0], 10);
}

export async function clickAndWait(page, selector, waitFor) {
  await page.locator(selector).click();
  await waitFor();
}
