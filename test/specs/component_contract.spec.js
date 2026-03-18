import { test, expect } from '@playwright/test';
import { gotoApp, textNumber } from './support/app.js';

test.describe('GoWebComponents Component Contracts', () => {
  test.beforeEach(async ({ page }) => {
    await gotoApp(page);
  });

  test('reusable counter instances keep isolated state through sibling and root rerenders', async ({ page }) => {
    const counterA = page.locator('[data-counter-id="A"]');
    const counterB = page.locator('[data-counter-id="B"]');
    const counterC = page.locator('[data-counter-id="C"]');

    await counterA.locator('.counter-btn').click();
    await expect(counterA.locator('.counter-value')).toHaveText('Counter A: 1');
    await expect(counterB.locator('.counter-value')).toHaveText('Counter B: 0');
    await expect(counterC.locator('.counter-value')).toHaveText('Counter C: 0');

    await page.getByRole('button', { name: 'Increment main' }).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');

    await expect(counterA.locator('.counter-value')).toHaveText('Counter A: 1');
    await expect(counterB.locator('.counter-value')).toHaveText('Counter B: 0');
    await expect(counterC.locator('.counter-value')).toHaveText('Counter C: 0');

    await counterB.locator('.counter-btn').click();
    await expect(counterA.locator('.counter-value')).toHaveText('Counter A: 1');
    await expect(counterB.locator('.counter-value')).toHaveText('Counter B: 1');
    await expect(counterC.locator('.counter-value')).toHaveText('Counter C: 0');
  });

  test('bad props do not crash neighboring components or break the app tree', async ({ page }) => {
    await expect(page.locator('#bad-props')).toHaveText('BadProps');

    await page.locator('#test-input').fill('stable app');
    await expect(page.locator('#input-value')).toHaveText('Input: stable app');

    await page.getByRole('button', { name: 'Increment main' }).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');
  });

  test('useId values stay stable and labels remain connected across rerenders', async ({ page }) => {
    const input = page.locator('#use-id-test input[type="text"]').first();
    const label = page.locator('#input-label');

    const initialId = await input.getAttribute('id');
    const initialFor = await label.getAttribute('for');
    expect(initialId).toBeTruthy();
    expect(initialId).toBe(initialFor);

    await page.getByRole('button', { name: 'Increment main' }).click();
    await expect(page.locator('[data-testid="count-display"]')).toHaveText('Count: 1');

    const rerenderId = await input.getAttribute('id');
    const rerenderFor = await label.getAttribute('for');
    expect(rerenderId).toBe(initialId);
    expect(rerenderFor).toBe(initialId);
  });

  test('effect child mount and unmount updates cleanup status contract', async ({ page }) => {
    const toggle = page.locator('#toggle-child-btn');

    await expect(page.locator('#cleanup-status')).toHaveText('');

    await toggle.click();
    await expect(page.locator('#effect-child')).toBeVisible();
    await expect(page.locator('#cleanup-status')).toHaveText('mounted');

    await toggle.click();
    await expect(page.locator('#effect-child')).toHaveCount(0);
    await expect(page.locator('#cleanup-status')).toHaveText('cleaned');
  });

  test('reactivity demo rerenders only the affected component and batches logical updates', async ({ page }) => {
    const aRendersBefore = await textNumber(page, '#react-a-renders');
    const bRendersBefore = await textNumber(page, '#react-b-renders');
    const batchRendersBefore = await textNumber(page, '#react-batch-renders');

    await page.locator('#react-a-inc').click();
    await expect(page.locator('#react-a-value')).toHaveText('A Value: 1');
    expect(await textNumber(page, '#react-a-renders')).toBe(aRendersBefore + 1);
    expect(await textNumber(page, '#react-b-renders')).toBe(bRendersBefore);

    await page.locator('#react-batch-btn').click();
    await expect(page.locator('#react-batch-value')).toHaveText('Batch Value: 3');
    expect(await textNumber(page, '#react-batch-renders')).toBe(batchRendersBefore + 1);
  });
});
