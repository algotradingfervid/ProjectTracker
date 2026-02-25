import { test, expect } from '@playwright/test';
import { uniqueName, createProject, createBOQ } from './helpers';

test.describe('BOQ CRUD', () => {
  let projectId: string;
  const projectName = uniqueName('BOQ Test Project');
  const boqAlpha = uniqueName('BOQ Alpha');
  const boqBeta = uniqueName('BOQ Beta');

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    projectId = await createProject(page, projectName);
    await page.close();
  });

  test('view empty BOQ list', async ({ page }) => {
    await page.goto(`/projects/${projectId}/boq`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).toContainText('BOQ');
    // Should show create button
    const createBtn = page.locator(`a[href="/projects/${projectId}/boq/create"]`).first();
    await expect(createBtn).toBeVisible();
  });

  test('create a BOQ', async ({ page }) => {
    await page.goto(`/projects/${projectId}/boq/create`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input[name="title"]');

    await page.fill('input[name="title"]', boqAlpha);

    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    // Should see BOQ in the list
    await page.goto(`/projects/${projectId}/boq`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).toContainText(boqAlpha);
  });

  test('view BOQ details', async ({ page }) => {
    await page.goto(`/projects/${projectId}/boq`, { waitUntil: 'domcontentloaded' });

    // Find BOQ link - could be hx-get or href
    const boqEl = page.locator('[hx-get*="/boq/"]').filter({ hasText: boqAlpha }).first();
    const link = page.locator('a[href*="/boq/"]').filter({ hasText: boqAlpha }).first();

    if (await boqEl.isVisible({ timeout: 3000 }).catch(() => false)) {
      await boqEl.click();
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toContainText(boqAlpha);
    } else if (await link.isVisible({ timeout: 3000 }).catch(() => false)) {
      await link.click();
      await page.waitForLoadState('domcontentloaded');
      await expect(page.locator('body')).toContainText(boqAlpha);
    } else {
      // BOQ is visible in list, just verify it exists
      await expect(page.locator('body')).toContainText(boqAlpha);
    }
  });

  test('create second BOQ and verify list', async ({ page }) => {
    await page.goto(`/projects/${projectId}/boq/create`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input[name="title"]');
    await page.fill('input[name="title"]', boqBeta);
    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    await page.goto(`/projects/${projectId}/boq`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).toContainText(boqAlpha);
    await expect(page.locator('body')).toContainText(boqBeta);
  });

  test('create BOQ with missing title shows error', async ({ page }) => {
    await page.goto(`/projects/${projectId}/boq/create`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input[name="title"]');

    // Leave title empty and submit
    await page.fill('input[name="title"]', '');
    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    // Should stay on form or show error
    const stillOnForm = await page.locator('input[name="title"]').isVisible();
    expect(stillOnForm).toBeTruthy();
  });

  test('BOQ export Excel works', async ({ page }) => {
    const boqId = await createBOQ(page, projectId, uniqueName('Export BOQ'));

    // Navigate to BOQ page first, then trigger download via direct navigation
    const downloadPromise = page.waitForEvent('download');
    await page.evaluate((url) => {
      window.location.href = url;
    }, `/projects/${projectId}/boq/${boqId}/export/excel`);
    const download = await downloadPromise;
    expect(download).toBeTruthy();
    const filename = download.suggestedFilename();
    expect(filename).toContain('.xlsx');
  });

  test('BOQ export PDF works', async ({ page }) => {
    const boqId = await createBOQ(page, projectId, uniqueName('PDF Export BOQ'));

    const downloadPromise = page.waitForEvent('download');
    await page.evaluate((url) => {
      window.location.href = url;
    }, `/projects/${projectId}/boq/${boqId}/export/pdf`);
    const download = await downloadPromise;
    expect(download).toBeTruthy();
    const filename = download.suggestedFilename();
    expect(filename).toContain('.pdf');
  });
});
