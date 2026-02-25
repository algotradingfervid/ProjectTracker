import { test, expect } from '@playwright/test';

test.describe('Project CRUD', () => {
  const projectName = `Test Project ${Date.now()}`;
  const updatedName = `Updated ${projectName}`;

  test('create a new project', async ({ page }) => {
    await page.goto('/projects');

    // Click create button
    await page.click('a[href="/projects/create"]');
    await page.waitForSelector('input[name="name"]');

    // Fill form
    await page.fill('input[name="name"]', projectName);
    await page.fill('input[name="client_name"]', 'Test Client');
    await page.fill('input[name="reference_number"]', 'REF-E2E-001');

    // Submit
    await page.click('button[type="submit"]');

    // Should redirect to projects list
    await page.waitForURL('/projects');
    await expect(page.locator('body')).toContainText(projectName);
  });

  test('edit the project', async ({ page }) => {
    await page.goto('/projects');

    // Project cards use hx-get="/projects/{id}/edit" and contain the name in h3
    const card = page.locator(`[hx-get*="/projects/"][hx-get$="/edit"]`).filter({ hasText: projectName }).first();
    await expect(card).toBeVisible();

    // Click the card to navigate to the edit form
    await card.click();
    await page.waitForSelector('input[name="name"]');

    // Update name
    await page.fill('input[name="name"]', updatedName);
    await page.click('button[type="submit"]');

    // Should redirect back and show updated name
    await page.waitForURL('/projects');
    await expect(page.locator('body')).toContainText(updatedName);
  });

  test('view project details', async ({ page }) => {
    await page.goto('/projects');

    // Click the project card to view/edit it
    const card = page.locator(`[hx-get*="/projects/"][hx-get$="/edit"]`).filter({ hasText: updatedName }).first();
    await expect(card).toBeVisible();
    await card.click();

    // Should show project form with the name
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('body')).toContainText(updatedName);
  });

  test('delete a project', async ({ page }) => {
    // Create a project to delete
    const deleteName = `Delete Me ${Date.now()}`;
    await page.goto('/projects/create');
    await page.waitForSelector('input[name="name"]');
    await page.fill('input[name="name"]', deleteName);
    await page.click('button[type="submit"]');
    await page.waitForURL('/projects');
    await expect(page.locator('body')).toContainText(deleteName);

    // Navigate to the project edit page (card click)
    const card = page.locator(`[hx-get*="/projects/"][hx-get$="/edit"]`).filter({ hasText: deleteName }).first();
    await card.click();
    await page.waitForLoadState('domcontentloaded');

    // Set up dialog handler for confirm
    page.on('dialog', dialog => dialog.accept());

    // Find and click delete button
    const deleteBtn = page.locator('[hx-delete*="/projects/"]').first();
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click();
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(500);

      // Verify project is gone
      await page.goto('/projects');
      const body = await page.locator('body').textContent();
      expect(body).not.toContain(deleteName);
    }
  });

  test('project settings page loads', async ({ page }) => {
    await page.goto('/projects');

    // Find a project card and extract ID
    const card = page.locator(`[hx-get*="/projects/"][hx-get$="/edit"]`).first();
    const hxGet = await card.getAttribute('hx-get');
    const match = hxGet?.match(/\/projects\/([a-z0-9_]+)\/edit/);
    if (match) {
      const projectId = match[1];
      const response = await page.goto(`/projects/${projectId}/settings`);
      expect(response?.status()).toBe(200);
    }
  });
});
