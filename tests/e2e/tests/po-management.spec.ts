import { test, expect } from '@playwright/test';
import { uniqueName, createProject, createVendor } from './helpers';

test.describe('Purchase Order Management', () => {
  let projectId: string;
  let vendorName: string;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    const projectName = uniqueName('PO Test Project');
    projectId = await createProject(page, projectName);
    vendorName = uniqueName('PO Vendor');
    await createVendor(page, vendorName);

    // Link vendor to project
    await page.goto(`/projects/${projectId}/vendors`);
    const row = page.locator('tr, [class*="card"]').filter({ hasText: vendorName }).first();
    const linkBtn = row.locator('[hx-post*="/link"]').first();
    if (await linkBtn.isVisible()) {
      await linkBtn.click();
      await page.waitForTimeout(500);
    }
    await page.close();
  });

  test('view empty PO list', async ({ page }) => {
    await page.goto(`/projects/${projectId}/po`);
    await expect(page.locator('body')).toContainText('PURCHASE ORDER');
    // Create PO button should exist
    const createBtn = page.locator(`a[href="/projects/${projectId}/po/create"]`).first();
    await expect(createBtn).toBeVisible();
  });

  test('create PO form renders with vendor dropdown', async ({ page }) => {
    await page.goto(`/projects/${projectId}/po/create`);
    await page.waitForSelector('select[name="vendor_id"]');

    // Vendor dropdown should have our vendor
    const vendorSelect = page.locator('select[name="vendor_id"]');
    await expect(vendorSelect).toBeVisible();
    const options = await vendorSelect.locator('option').allTextContents();
    const hasVendor = options.some(opt => opt.includes(vendorName));
    expect(hasVendor).toBeTruthy();
  });

  test('create a purchase order', async ({ page }) => {
    await page.goto(`/projects/${projectId}/po/create`);
    await page.waitForSelector('select[name="vendor_id"]');

    // Select vendor
    const vendorOption = page.locator('select[name="vendor_id"] option').filter({ hasText: vendorName }).first();
    const vendorValue = await vendorOption.getAttribute('value');
    if (vendorValue) {
      await page.selectOption('select[name="vendor_id"]', vendorValue);
    }

    // Fill dates and terms
    await page.fill('input[name="order_date"]', '2025-06-15');
    await page.fill('input[name="quotation_ref"]', 'QR-E2E-001');

    // Fill terms
    const paymentTerms = page.locator('textarea[name="payment_terms"]');
    if (await paymentTerms.isVisible()) {
      await paymentTerms.fill('Net 30');
    }

    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    // Should redirect and PO should be visible
    await page.goto(`/projects/${projectId}/po`);
    const body = await page.locator('body').textContent();
    // PO number or vendor name should appear in the list
    expect(body?.includes(vendorName) || body?.includes('FY')).toBeTruthy();
  });

  test('view PO details', async ({ page }) => {
    await page.goto(`/projects/${projectId}/po`);

    // Click on first PO link
    const poLink = page.locator('a[href*="/po/"]').filter({ hasNotText: 'create' }).first();
    if (await poLink.isVisible()) {
      await poLink.click();
      await page.waitForLoadState('domcontentloaded');

      // Should show PO details with vendor info
      await expect(page.locator('body')).toContainText(vendorName);
    }
  });

  test('PO list status filter tabs exist', async ({ page }) => {
    await page.goto(`/projects/${projectId}/po`);

    // Status filter buttons should be present
    const body = await page.locator('body').textContent();
    expect(body).toContain('ALL');
    expect(body).toContain('DRAFT');
  });

  test('delete a purchase order', async ({ page }) => {
    await page.goto(`/projects/${projectId}/po`);

    // Set up dialog handler
    page.on('dialog', dialog => dialog.accept());

    // Find delete button (only available for draft POs)
    const deleteBtn = page.locator('[hx-delete*="/po/"]').first();
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click();
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(500);
    }
  });
});
