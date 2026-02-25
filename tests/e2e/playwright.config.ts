import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  retries: 0,
  timeout: 60000,
  use: {
    baseURL: 'http://localhost:8090',
    trace: 'on-first-retry',
    navigationTimeout: 15000,
    actionTimeout: 10000,
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: 'cd ../.. && make run',
    url: 'http://localhost:8090',
    reuseExistingServer: true,
    timeout: 30000,
  },
});
