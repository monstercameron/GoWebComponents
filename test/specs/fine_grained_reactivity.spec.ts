import { test, expect } from '@playwright/test';
import { gotoApp, textNumber } from './support/app';

test.describe('Fine-Grained Reactivity', () => {
  test.beforeEach(async ({ page }) => {
    await gotoApp(page);
  });

  test('keeps parent and static sibling stable while hot regions update independently', async ({ page }) => {
    await expect(page.locator('#fg-left-value')).toHaveText('1');
    await expect(page.locator('#fg-right-value')).toHaveText('8');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(1);
    expect(await textNumber(page, '#fg-static-renders')).toBe(1);

    await page.evaluate(() => {
      window.__fgRefs = {
        staticStrong: document.querySelector('#fg-static-strong'),
        leftValue: document.querySelector('#fg-left-value'),
        rightValue: document.querySelector('#fg-right-value'),
        selectorValue: document.querySelector('#fg-selector-value'),
      };
    });

    await page.locator('#fg-left-inc').click();
    await expect(page.locator('#fg-left-value')).toHaveText('2');
    await expect(page.locator('#fg-right-value')).toHaveText('8');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(1);
    expect(await textNumber(page, '#fg-static-renders')).toBe(1);

    const afterLeft = await page.evaluate(() => ({
      staticStable: window.__fgRefs.staticStrong === document.querySelector('#fg-static-strong'),
      leftStable: window.__fgRefs.leftValue === document.querySelector('#fg-left-value'),
      rightStable: window.__fgRefs.rightValue === document.querySelector('#fg-right-value'),
      selectorStable: window.__fgRefs.selectorValue === document.querySelector('#fg-selector-value'),
    }));
    expect(afterLeft.staticStable).toBe(true);
    expect(afterLeft.leftStable).toBe(true);
    expect(afterLeft.rightStable).toBe(true);
    expect(afterLeft.selectorStable).toBe(true);

    await page.locator('#fg-right-inc').click();
    await expect(page.locator('#fg-left-value')).toHaveText('2');
    await expect(page.locator('#fg-right-value')).toHaveText('9');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(1);
    expect(await textNumber(page, '#fg-static-renders')).toBe(1);

    const afterRight = await page.evaluate(() => ({
      staticStable: window.__fgRefs.staticStrong === document.querySelector('#fg-static-strong'),
      leftStable: window.__fgRefs.leftValue === document.querySelector('#fg-left-value'),
      rightStable: window.__fgRefs.rightValue === document.querySelector('#fg-right-value'),
    }));
    expect(afterRight.staticStable).toBe(true);
    expect(afterRight.leftStable).toBe(true);
    expect(afterRight.rightStable).toBe(true);

    await page.locator('#fg-parent-rerender').click();
    await expect(page.locator('#fg-parent-label')).toContainText('updated');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(2);
    expect(await textNumber(page, '#fg-static-renders')).toBe(1);
    await expect(page.locator('#fg-left-value')).toHaveText('2');
    await expect(page.locator('#fg-right-value')).toHaveText('9');

    const afterParent = await page.evaluate(() => ({
      staticStable: window.__fgRefs.staticStrong === document.querySelector('#fg-static-strong'),
      leftStable: window.__fgRefs.leftValue === document.querySelector('#fg-left-value'),
      rightStable: window.__fgRefs.rightValue === document.querySelector('#fg-right-value'),
    }));
    expect(afterParent.staticStable).toBe(true);
    expect(afterParent.leftStable).toBe(true);
    expect(afterParent.rightStable).toBe(true);
  });

  test('selector-backed fine-grained output avoids parent rerenders when projection is unchanged', async ({ page }) => {
    await expect(page.locator('#fg-selector-value')).toHaveText('odd');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(1);

    await page.evaluate(() => {
      window.__fgSelectorRef = document.querySelector('#fg-selector-value');
    });

    await page.locator('#fg-selector-same').click();
    await expect(page.locator('#fg-selector-value')).toHaveText('odd');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(1);

    const selectorStableAfterSame = await page.evaluate(() => {
      return window.__fgSelectorRef === document.querySelector('#fg-selector-value');
    });
    expect(selectorStableAfterSame).toBe(true);

    await page.locator('#fg-selector-change').click();
    await expect(page.locator('#fg-selector-value')).toHaveText('even');
    expect(await textNumber(page, '#fg-parent-renders')).toBe(1);

    const selectorStableAfterChange = await page.evaluate(() => {
      return window.__fgSelectorRef === document.querySelector('#fg-selector-value');
    });
    expect(selectorStableAfterChange).toBe(true);
  });
});
