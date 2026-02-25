import { test, expect } from '@playwright/test';
import { uniqueName, createProject } from './helpers';

test.describe('Address Management', () => {
  let projectId: string;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    const projectName = uniqueName('Address Test Project');
    projectId = await createProject(page, projectName);
    await page.close();
  });

  test('navigate to bill-to address list', async ({ page }) => {
    await page.goto(`/projects/${projectId}/addresses/bill-to`, { waitUntil: 'domcontentloaded' });
    const body = await page.locator('body').textContent();
    expect(body?.toLowerCase()).toContain('bill');
  });

  test('navigate to ship-to address list', async ({ page }) => {
    await page.goto(`/projects/${projectId}/addresses/ship-to`, { waitUntil: 'domcontentloaded' });
    const body = await page.locator('body').textContent();
    expect(body?.toLowerCase()).toContain('ship');
  });

  test('create bill-to address form renders', async ({ page }) => {
    await page.goto(`/projects/${projectId}/addresses/bill-to/new`, { waitUntil: 'domcontentloaded' });

    // Key form fields should be present
    await expect(page.locator('input[name="company_name"]')).toBeVisible();
    await expect(page.locator('input[name="address_line_1"]')).toBeVisible();
    await expect(page.locator('input[name="city"]')).toBeVisible();
  });

  test('create a bill-to address', async ({ page }) => {
    const companyName = uniqueName('Bill Corp');
    await page.goto(`/projects/${projectId}/addresses/bill-to/new`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input[name="company_name"]');

    await page.fill('input[name="company_name"]', companyName);
    await page.fill('input[name="contact_person"]', 'Jane Doe');
    await page.fill('input[name="address_line_1"]', '123 Billing Street');
    await page.fill('input[name="city"]', 'Mumbai');
    await page.fill('input[name="pin_code"]', '400001');
    await page.fill('input[name="phone"]', '9876543210');
    // GSTIN is required for bill_to type
    await page.fill('input[name="gstin"]', '27AAPFU0939F1ZV');

    // Fill state - detect if select or input
    const stateEl = page.locator('[name="state"]').first();
    if (await stateEl.count() > 0) {
      const tagName = await stateEl.evaluate(el => el.tagName.toLowerCase());
      if (tagName === 'select') {
        // Select first non-empty option
        const options = await stateEl.locator('option').all();
        for (const opt of options) {
          const val = await opt.getAttribute('value');
          if (val && val !== '') {
            await stateEl.selectOption(val);
            break;
          }
        }
      } else {
        await stateEl.fill('Maharashtra');
      }
    }

    // Fill country
    const countryEl = page.locator('[name="country"]').first();
    if (await countryEl.count() > 0) {
      const tagName = await countryEl.evaluate(el => el.tagName.toLowerCase());
      if (tagName === 'select') {
        const options = await countryEl.locator('option').all();
        for (const opt of options) {
          const val = await opt.getAttribute('value');
          if (val && val !== '') {
            await countryEl.selectOption(val);
            break;
          }
        }
      } else {
        await countryEl.fill('India');
      }
    }

    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');

    // Wait for potential redirect
    await page.waitForTimeout(1000);

    // Verify address appears in list
    await page.goto(`/projects/${projectId}/addresses/bill-to`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).toContainText(companyName);
  });

  test('create a ship-to address', async ({ page }) => {
    const companyName = uniqueName('Ship Corp');
    await page.goto(`/projects/${projectId}/addresses/ship-to/new`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input[name="company_name"]');

    await page.fill('input[name="company_name"]', companyName);
    await page.fill('input[name="contact_person"]', 'John Doe');
    await page.fill('input[name="address_line_1"]', '456 Shipping Lane');
    await page.fill('input[name="city"]', 'Delhi');
    await page.fill('input[name="pin_code"]', '110001');
    await page.fill('input[name="phone"]', '9876543211');

    // Fill state
    const stateEl = page.locator('[name="state"]').first();
    if (await stateEl.count() > 0) {
      const tagName = await stateEl.evaluate(el => el.tagName.toLowerCase());
      if (tagName === 'select') {
        const options = await stateEl.locator('option').all();
        for (const opt of options) {
          const val = await opt.getAttribute('value');
          if (val && val !== '') {
            await stateEl.selectOption(val);
            break;
          }
        }
      } else {
        await stateEl.fill('Delhi');
      }
    }

    // Fill country
    const countryEl = page.locator('[name="country"]').first();
    if (await countryEl.count() > 0) {
      const tagName = await countryEl.evaluate(el => el.tagName.toLowerCase());
      if (tagName === 'select') {
        const options = await countryEl.locator('option').all();
        for (const opt of options) {
          const val = await opt.getAttribute('value');
          if (val && val !== '') {
            await countryEl.selectOption(val);
            break;
          }
        }
      } else {
        await countryEl.fill('Delhi');
      }
    }

    await page.click('button[type="submit"]');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    await page.goto(`/projects/${projectId}/addresses/ship-to`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).toContainText(companyName);
  });

  test('all address types are accessible', async ({ page }) => {
    const types = ['bill-from', 'ship-from', 'bill-to', 'ship-to', 'install-at'];
    for (const type of types) {
      const response = await page.goto(`/projects/${projectId}/addresses/${type}`, { waitUntil: 'domcontentloaded' });
      expect(response?.status()).toBe(200);
    }
  });

  test('address template download works', async ({ page, request }) => {
    // Route uses {type} param which expects underscore format (ship_to, not ship-to)
    const response = await request.get(`/projects/${projectId}/addresses/ship_to/template`);
    expect(response.status()).toBe(200);
    const contentType = response.headers()['content-type'] || '';
    // Should return an Excel file
    expect(contentType).toContain('spreadsheet');
  });
});
