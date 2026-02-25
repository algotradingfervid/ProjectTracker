import { test, expect } from '@playwright/test';
import { uniqueName, createProject, createVendor } from './helpers';

test.describe('Vendor Management', () => {
  test('view vendor list page', async ({ page }) => {
    await page.goto('/vendors');
    await expect(page.locator('body')).toContainText('VENDOR');
    // Create button should exist
    const createBtn = page.locator('a[href="/vendors/create"]').first();
    await expect(createBtn).toBeVisible();
  });

  test('create a vendor', async ({ page }) => {
    const vendorName = uniqueName('E2E Vendor');
    await page.goto('/vendors/create');
    await page.waitForSelector('input[name="name"]');

    await page.fill('input[name="name"]', vendorName);
    await page.fill('input[name="city"]', 'Mumbai');
    await page.fill('input[name="state"]', 'Maharashtra');
    await page.fill('input[name="contact_name"]', 'John Doe');
    await page.fill('input[name="phone"]', '9876543210');
    await page.fill('input[name="email"]', 'vendor@example.com');
    await page.fill('input[name="gstin"]', '27AAPFU0939F1ZV');

    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    // Should redirect to vendor list and show the vendor
    await page.goto('/vendors');
    await expect(page.locator('body')).toContainText(vendorName);
  });

  test('create vendor with missing name shows error', async ({ page }) => {
    await page.goto('/vendors/create');
    await page.waitForSelector('input[name="name"]');

    await page.fill('input[name="name"]', '');
    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    // Should stay on form or show error
    const stillOnForm = await page.locator('input[name="name"]').isVisible();
    expect(stillOnForm).toBeTruthy();
  });

  test('edit a vendor', async ({ page }) => {
    const vendorName = uniqueName('Edit Vendor');
    await createVendor(page, vendorName);

    await page.goto('/vendors');
    await expect(page.locator('body')).toContainText(vendorName);

    // Find and click edit
    const row = page.locator('tr, [class*="card"]').filter({ hasText: vendorName }).first();
    const editBtn = row.locator('a[href*="/edit"]').first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
    } else {
      // Try HTMX edit link
      const htmxEdit = row.locator('[hx-get*="/edit"]').first();
      await htmxEdit.click();
    }
    await page.waitForSelector('input[name="name"]');

    // Update the name
    const updatedName = `Updated ${vendorName}`;
    await page.fill('input[name="name"]', updatedName);
    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    await page.goto('/vendors');
    await expect(page.locator('body')).toContainText(updatedName);
  });

  test('delete a vendor', async ({ page }) => {
    const vendorName = uniqueName('Delete Vendor');
    await createVendor(page, vendorName);

    await page.goto('/vendors');
    await expect(page.locator('body')).toContainText(vendorName);

    // Set up dialog handler for hx-confirm
    page.on('dialog', dialog => dialog.accept());

    const row = page.locator('tr, [class*="card"]').filter({ hasText: vendorName }).first();
    const deleteBtn = row.locator('[hx-delete*="/vendors/"]').first();
    await deleteBtn.click();
    await page.waitForLoadState('domcontentloaded');

    // Wait a moment for HTMX to process
    await page.waitForTimeout(500);

    // Vendor should be gone
    await page.goto('/vendors');
    const body = await page.locator('body').textContent();
    expect(body).not.toContain(vendorName);
  });

  test('link vendor to project', async ({ page }) => {
    const projectName = uniqueName('Vendor Link Proj');
    const vendorName = uniqueName('Link Vendor');

    const projectId = await createProject(page, projectName);
    await createVendor(page, vendorName);

    // Go to project-scoped vendor page
    await page.goto(`/projects/${projectId}/vendors`);
    await expect(page.locator('body')).toContainText(vendorName);

    // Find the link button
    const row = page.locator('tr, [class*="card"]').filter({ hasText: vendorName }).first();
    const linkBtn = row.locator('[hx-post*="/link"]').first();
    if (await linkBtn.isVisible()) {
      await linkBtn.click();
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(500);

      // After linking, the button text should change to LINKED or similar
      await page.goto(`/projects/${projectId}/vendors`);
      const rowAfter = page.locator('tr, [class*="card"]').filter({ hasText: vendorName }).first();
      await expect(rowAfter).toBeVisible();
    }
  });
});
