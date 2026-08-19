---
name: playwright-cli
description: Automate browser interactions, test web pages and work with Playwright tests.
allowed-tools: Bash(node:*) Bash(npm:*) Bash(npx:*)
---

# Browser Automation with Playwright

## Important: Two approaches

**⚠️ `playwright-cli` blocks the terminal** — it keeps the session alive by design, which freezes the agent's terminal. **Do NOT use `playwright-cli` for autonomous tasks.**

**✅ Use Node.js scripts with the Playwright API** — fully autonomous, non-blocking, returns control immediately.

## Quick start (Node.js API)

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://example.com');
  console.log('Title:', await page.title());
  
  await page.screenshot({ path: 'screenshot.png' });
  await browser.close();
})();
```

Run with: `node script.js`

## Setup

Playwright is installed locally in `~\node_modules\playwright`. If needed:

```bash
cd ~ && npm install playwright
```

## Core patterns

### Navigate and extract content

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://example.com');
  
  // Get page info
  const title = await page.title();
  const content = await page.textContent('body');
  const html = await page.content();
  
  // Get all links
  const links = await page.$$eval('a', els => els.map(e => ({ text: e.textContent, href: e.href })));
  
  console.log(JSON.stringify({ title, links }, null, 2));
  await browser.close();
})();
```

### Click, type, fill

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://example.com/form');
  
  // Fill inputs
  await page.fill('input[name="email"]', 'user@example.com');
  await page.fill('input[name="password"]', 'password123');
  
  // Click button
  await page.click('button[type="submit"]');
  
  // Wait for navigation
  await page.waitForURL('**/dashboard');
  
  // Take screenshot
  await page.screenshot({ path: 'after-login.png' });
  
  await browser.close();
})();
```

### Screenshot and PDF

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://example.com');
  
  // Full page screenshot
  await page.screenshot({ path: 'full-page.png', fullPage: true });
  
  // Element screenshot
  await page.screenshot({ path: 'element.png', clip: { x: 0, y: 0, width: 500, height: 300 } });
  
  // PDF
  await page.pdf({ path: 'page.pdf', format: 'A4' });
  
  await browser.close();
})();
```

### Wait for elements

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://example.com');
  
  // Wait for element to appear
  await page.waitForSelector('.dynamic-content', { timeout: 10000 });
  
  // Wait for network idle
  await page.waitForLoadState('networkidle');
  
  // Wait for specific response
  await page.waitForResponse(resp => resp.url().includes('/api/data'));
  
  await browser.close();
})();
```

### Handle dialogs

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  // Auto-accept dialogs
  page.on('dialog', dialog => dialog.accept());
  
  await page.goto('https://example.com');
  await page.click('#trigger-dialog');
  
  await browser.close();
})();
```

### Network interception

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  // Mock API response
  await page.route('**/api/data', route => {
    route.fulfill({
      status: 200,
      body: JSON.stringify({ mocked: true })
    });
  });
  
  // Log all requests
  page.on('request', req => console.log('Request:', req.url()));
  page.on('response', resp => console.log('Response:', resp.url(), resp.status()));
  
  await page.goto('https://example.com');
  
  await browser.close();
})();
```

### Multiple tabs

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  
  // Open multiple pages
  const page1 = await context.newPage();
  const page2 = await context.newPage();
  
  await page1.goto('https://example.com');
  await page2.goto('https://google.com');
  
  // List all pages
  const pages = context.pages();
  console.log('Open pages:', pages.map(p => p.url()));
  
  // Switch to first page
  await pages[0].bringToFront();
  
  await browser.close();
})();
```

### Storage state (cookies, localStorage)

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();
  
  // Login
  await page.goto('https://example.com/login');
  await page.fill('#email', 'user@example.com');
  await page.fill('#password', 'pass');
  await page.click('#submit');
  await page.waitForURL('**/dashboard');
  
  // Save storage state
  await context.storageState({ path: 'auth.json' });
  
  await browser.close();
  
  // Later: restore state
  const browser2 = await chromium.launch({ headless: true });
  const context2 = await browser2.newContext({ storageState: 'auth.json' });
  const page2 = await context2.newPage();
  
  // Already logged in
  await page2.goto('https://example.com/dashboard');
  
  await browser2.close();
})();
```

### Mobile emulation

```javascript
const { chromium, devices } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const iPhone = devices['iPhone 13'];
  const context = await browser.newContext({ ...iPhone });
  const page = await context.newPage();
  
  await page.goto('https://example.com');
  await page.screenshot({ path: 'mobile.png' });
  
  await browser.close();
})();
```

## Complete examples

### Form submission test

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://demo.playwright.dev/todomvc');
  
  // Add todos
  await page.fill('.new-todo', 'Buy groceries');
  await page.press('.new-todo', 'Enter');
  await page.fill('.new-todo', 'Water flowers');
  await page.press('.new-todo', 'Enter');
  
  // Check first todo
  await page.click('.toggle');
  
  // Screenshot result
  await page.screenshot({ path: 'todos.png', fullPage: true });
  
  // Get all todos
  const todos = await page.$$eval('.todo-list li', els => els.map(e => e.textContent));
  console.log('Todos:', todos);
  
  await browser.close();
})();
```

### Scrape multiple pages

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  const results = [];
  
  for (let i = 1; i <= 3; i++) {
    await page.goto(`https://example.com/page/${i}`);
    const title = await page.title();
    const content = await page.textContent('main');
    results.push({ page: i, title, content: content.substring(0, 200) });
  }
  
  console.log(JSON.stringify(results, null, 2));
  await browser.close();
})();
```

### Test with assertions

```javascript
const { chromium } = require('playwright');
const assert = require('assert');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  await page.goto('https://example.com');
  
  // Assertions
  const title = await page.title();
  assert.strictEqual(title, 'Example Domain', 'Title should match');
  
  const heading = await page.textContent('h1');
  assert.strictEqual(heading, 'Example Domain', 'Heading should match');
  
  const link = await page.$('a');
  assert.ok(link, 'Should have a link');
  
  console.log('All tests passed!');
  await browser.close();
})();
```

## Troubleshooting

### Browser not launching

```bash
# Install browser binaries
npx playwright install chromium
```

### Headed mode (for debugging)

```javascript
const browser = await chromium.launch({ headless: false });
```

### Slow on Windows

```javascript
const browser = await chromium.launch({ 
  headless: true,
  args: ['--disable-gpu', '--no-sandbox']
});
```

### Timeout issues

```javascript
const browser = await chromium.launch({ headless: true });
const page = await browser.newPage();
page.setDefaultTimeout(30000); // 30 seconds
```

## CLI reference (manual use only)

If you need to use `playwright-cli` manually (not recommended for autonomous tasks):

```bash
playwright-cli open https://example.com
playwright-cli snapshot
playwright-cli click e15
playwright-cli type "hello"
playwright-cli screenshot
playwright-cli close
```

**Note:** CLI commands block the terminal. Use Node.js API for autonomous agent work.
