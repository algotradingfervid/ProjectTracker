import { Page, expect } from '@playwright/test';

/** Generate a unique name using timestamp to avoid collisions between test runs. */
export function uniqueName(prefix: string): string {
  return `${prefix} ${Date.now()}`;
}

/** Navigate via direct URL, waiting for main content to load. */
export async function navigateTo(page: Page, path: string) {
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}

/**
 * Create a project and return its ID extracted from the project card's hx-get attribute.
 * Project cards use: hx-get="/projects/{id}/edit"
 */
export async function createProject(page: Page, name: string): Promise<string> {
  await page.goto('/projects/create');
  await page.waitForSelector('input[name="name"]');
  await page.fill('input[name="name"]', name);
  await page.fill('input[name="client_name"]', 'E2E Client');
  await page.fill('input[name="reference_number"]', `REF-${Date.now()}`);
  await page.click('button[type="submit"]');
  await page.waitForURL('/projects');

  // Project cards have hx-get="/projects/{id}/edit" and contain the project name in h3
  const card = page.locator(`[hx-get*="/projects/"][hx-get$="/edit"]`).filter({ hasText: name }).first();
  await expect(card).toBeVisible({ timeout: 5000 });
  const hxGet = await card.getAttribute('hx-get');
  if (!hxGet) throw new Error(`Could not find project card for "${name}"`);
  const match = hxGet.match(/\/projects\/([a-z0-9_]+)\/edit/);
  if (!match) throw new Error(`Could not extract project ID from "${hxGet}"`);
  return match[1];
}

/** Create a BOQ within a project and return its ID. */
export async function createBOQ(page: Page, projectId: string, title: string): Promise<string> {
  await page.goto(`/projects/${projectId}/boq/create`, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('input[name="title"]');
  await page.fill('input[name="title"]', title);
  await page.click('button[type="submit"]');
  await page.waitForLoadState('domcontentloaded');

  // Find the BOQ in the list - BOQ rows have hx-get="/projects/{projectId}/boq/{id}"
  await page.goto(`/projects/${projectId}/boq`, { waitUntil: 'domcontentloaded' });

  // Look for elements that link to a specific BOQ (hx-get or href containing /boq/)
  const boqEl = page.locator(`[hx-get*="/boq/"]`).filter({ hasText: title }).first();
  await expect(boqEl).toBeVisible({ timeout: 5000 });
  const attr = await boqEl.getAttribute('hx-get') || await boqEl.getAttribute('href') || '';
  const match = attr.match(/\/boq\/([a-z0-9_]+)/);
  if (!match) throw new Error(`Could not extract BOQ ID from "${attr}"`);
  return match[1];
}

/** Create a vendor and return to the vendor list. */
export async function createVendor(page: Page, name: string) {
  await page.goto('/vendors/create');
  await page.waitForSelector('input[name="name"]');
  await page.fill('input[name="name"]', name);
  await page.fill('input[name="city"]', 'Mumbai');
  await page.fill('input[name="contact_name"]', 'Test Contact');
  await page.fill('input[name="phone"]', '9876543210');
  await page.click('button[type="submit"]');
  await page.waitForLoadState('domcontentloaded');
}
