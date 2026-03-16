const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30000,
  expect: {
    timeout: 15000,
  },
  use: {
    baseURL: 'http://127.0.0.1:8081',
    headless: true,
  },
  webServer: {
    command: 'node tools/context_test_server.js',
    url: 'http://127.0.0.1:8081/static/index.html',
    reuseExistingServer: true,
    timeout: 120000,
  },
  reporter: [['list']],
});