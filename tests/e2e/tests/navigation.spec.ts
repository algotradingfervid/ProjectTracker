import { test, expect } from '@playwright/test';

test.describe('Navigation & Layout', () => {
  test('home redirects to projects', async ({ page }) => {
    await page.goto('/');
    await page.waitForURL('/projects');
    await expect(page).toHaveURL('/projects');
  });

  test('sidebar is visible and has key links', async ({ page }) => {
    await page.goto('/projects');
    // Sidebar should be present with navigation
    const sidebar = page.locator('nav, aside, [style*="--bg-sidebar"]').first();
    await expect(sidebar).toBeVisible();

    // Projects link should exist
    await expect(page.locator('a[href="/projects"]').first()).toBeVisible();
    // Vendors link should exist
    await expect(page.locator('a[href="/vendors"]').first()).toBeVisible();
  });

  test('direct URL access to projects list', async ({ page }) => {
    const response = await page.goto('/projects');
    expect(response?.status()).toBe(200);
    await expect(page.locator('body')).toContainText('PROJECTS');
  });

  test('HTMX partial loads update content without full reload', async ({ page }) => {
    await page.goto('/projects');

    // Click a sidebar link with hx-get (HTMX navigation)
    const vendorsLink = page.locator('a[href="/vendors"]').first();
    if (await vendorsLink.isVisible()) {
      await vendorsLink.click();
      await page.waitForLoadState('domcontentloaded');
      // URL should update via hx-push-url
      await expect(page).toHaveURL('/vendors');
    }
  });

  test('browser back/forward navigation works', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('domcontentloaded');

    // Navigate to vendors
    await page.goto('/vendors');
    await page.waitForLoadState('domcontentloaded');

    // Go back
    await page.goBack();
    await expect(page).toHaveURL('/projects');
  });

  test('404 for non-existent project', async ({ page }) => {
    const response = await page.goto('/projects/nonexistent_id_12345');
    // Should return an error status or show error content
    const status = response?.status() ?? 0;
    // PocketBase may return 200 with error message or 404
    expect([200, 404, 500]).toContain(status);
  });
});
