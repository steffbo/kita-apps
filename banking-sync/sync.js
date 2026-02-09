const { chromium } = require('playwright');
const { Blob } = require('buffer');
const fs = require('fs');
const path = require('path');

// Configuration from environment
const CONFIG = {
  bankUrl:
    process.env.BANK_URL ||
    'https://www.sozialbank-onlinebanking.de/services_auth/auth-frontend/?v=d2037d6fa58a8828878a28a81fe07257&client_id=fkp&redirect_uri=https:%2F%2Fwww.sozialbank-onlinebanking.de%2Fservices_cloud%2Fportal%2Fportal-oauth%2Flogin',
  username: process.env.BANK_USERNAME,
  password: process.env.BANK_PASSWORD,
  apiUrl: process.env.API_URL || 'http://localhost:8081/api/fees/v1',
  apiToken: process.env.CRON_API_TOKEN,
  headless: process.env.HEADLESS !== 'false',
  downloadDir: process.env.DOWNLOAD_DIR || path.resolve(__dirname, 'output'),
  userDataDir: process.env.USER_DATA_DIR || path.resolve(__dirname, 'profile'),
  usePersistentContext: process.env.USE_PERSISTENT_CONTEXT === 'true',
  twoFaTimeoutMs: Number(process.env.TWO_FA_TIMEOUT_SECONDS || 600) * 1000,
  loginTimeoutMs: Number(process.env.LOGIN_TIMEOUT_SECONDS || 30) * 1000,
  loginOutcomeTimeoutMs: Number(process.env.LOGIN_OUTCOME_TIMEOUT_SECONDS || 45) * 1000,
  waitProgressIntervalMs: Number(process.env.WAIT_PROGRESS_INTERVAL_SECONDS || 10) * 1000,
  debugDir: process.env.DEBUG_DIR || path.join(process.env.DOWNLOAD_DIR || path.resolve(__dirname, 'output'), 'debug'),
  userAgent:
    process.env.USER_AGENT ||
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
  // Global timeout for entire sync operation (default 15 minutes)
  globalTimeoutMs: Number(process.env.GLOBAL_TIMEOUT_SECONDS || 900) * 1000,
};

// Global state for cancellation
let abortController = null;
let browserContext = null;
let browserInstance = null;

function getAbortController() {
  return abortController;
}

function cancelSync() {
  if (abortController) {
    abortController.abort();
  }
  // Force close browser if exists
  if (browserContext || browserInstance) {
    closeBrowserContext(browserContext, browserInstance).catch(() => undefined);
    browserContext = null;
    browserInstance = null;
  }
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function createDebugFileName(label, extension) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const safeLabel = String(label || 'debug').replace(/[^a-zA-Z0-9_-]/g, '_');
  return `${timestamp}_${safeLabel}.${extension}`;
}

function createLogger(onLog) {
  return message => {
    console.log(message);
    if (onLog) onLog(message);
  };
}

function getRootUrl(root) {
  try {
    if (typeof root.url === 'function') return root.url();
  } catch {
    return 'unknown';
  }
  return 'unknown';
}

function getRoots(page) {
  const frames = page.frames();
  return frames.length ? [page, ...frames] : [page];
}

function isPageClosed(page) {
  try {
    return page.isClosed();
  } catch {
    return true;
  }
}

async function findFirstCurrentlyVisible(page, builders) {
  const roots = getRoots(page);
  for (const root of roots) {
    for (let i = 0; i < builders.length; i++) {
      try {
        const locator = builders[i](root).first();
        if (await locator.isVisible()) {
          return { locator, selectorIndex: i, root };
        }
      } catch {
        // try next selector
      }
    }
  }
  return null;
}

async function findFirstVisible(page, builders, label, timeoutMs = 10000, log = null) {
  const roots = getRoots(page);
  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    if (isPageClosed(page)) {
      throw new Error(`Page closed while waiting for ${label}`);
    }

    for (const root of roots) {
      for (let i = 0; i < builders.length; i++) {
        let locator;
        try {
          locator = builders[i](root);
        } catch {
          continue;
        }

        const remainingMs = Math.max(1, deadline - Date.now());
        const attemptTimeoutMs = Math.min(1200, remainingMs);

        try {
          const first = locator.first();
          await first.waitFor({ state: 'visible', timeout: attemptTimeoutMs });
          if (log) {
            const rootInfo = roots.length > 1 ? ` (frame: ${getRootUrl(root)})` : '';
            log(`  Found ${label} using selector #${i + 1}${rootInfo}`);
          }
          return first;
        } catch {
          // try next selector
        }
      }
    }

    await page.waitForLoadState('domcontentloaded', { timeout: 300 }).catch(() => undefined);
  }

  const frameInfo = roots.map(getRootUrl).join(', ');
  throw new Error(`Could not find visible element for ${label}. Frames: ${frameInfo}`);
}

async function clickIfVisible(page, builders) {
  try {
    const element = await findFirstVisible(page, builders, 'element', 2000);
    await element.click().catch(() => undefined);
    return true;
  } catch {
    return false;
  }
}

async function dismissCookieBanner(page) {
  await clickIfVisible(page, [
    root => root.getByRole('button', { name: /Alle akzeptieren|Akzeptieren/i }),
    root => root.locator('button:has-text("Alle akzeptieren")'),
  ]);
}

async function fillCredentials(page, log) {
  await clickIfVisible(page, [
    root => root.getByRole('tab', { name: /Zugangsdaten/i }),
    root => root.locator('button:has-text("Mit Zugangsdaten anmelden")'),
  ]);

  log('  Looking for username field...');
  const usernameInput = await findFirstVisible(
    page,
    [
      root => root.locator('#vrNetKey'),
      root => root.locator('[data-automation-id="vrNetKey-input"]'),
      root => root.locator('input[name="vrNetKeyFormControl"]'),
      root => root.locator('#vvrnKey'),
      root => root.locator('[data-automation-id="vvrnKey-input"]'),
      root => root.locator('input[name="vvrnKeyFormControl"]'),
      root => root.getByLabel(/NetKey|Alias|Benutzer|User|Login/i),
      root => root.getByRole('textbox', { name: /NetKey|Alias|Benutzer|User|Login/i }),
      root => root.locator('input[autocomplete="username"]'),
      root => root.locator('input[type="text"]'),
    ],
    'username',
    CONFIG.loginTimeoutMs,
    log
  );
  await usernameInput.fill(CONFIG.username);
  log('  ✓ Username filled');

  log('  Looking for PIN field...');
  const passwordInput = await findFirstVisible(
    page,
    [
      root => root.locator('input#pin'),
      root => root.locator('input[type="password"]'),
    ],
    'pin',
    CONFIG.loginTimeoutMs,
    log
  );
  await passwordInput.fill(CONFIG.password);
  log('  ✓ PIN filled');

  log('  Looking for submit button...');
  const submitButton = await findFirstVisible(
    page,
    [
      root => root.locator('[data-automation-id="sign-in-button"]'),
      root => root.locator('button:has-text("Anmelden")'),
      root => root.getByRole('button', { name: /Anmelden|Login|Einloggen|Weiter/i }),
      root => root.locator('button[type="submit"]'),
    ],
    'login button',
    CONFIG.loginTimeoutMs,
    log
  );
  await submitButton.click();
  log('  ✓ Submit clicked');
}

async function hasVisibleLoginForm(page) {
  const loginFieldSelectors = [
    root => root.locator('#vrNetKey'),
    root => root.locator('#vvrnKey'),
    root => root.locator('input[name="vrNetKeyFormControl"]'),
    root => root.locator('input[name="vvrnKeyFormControl"]'),
    root => root.locator('input#pin'),
    root => root.locator('input[type="password"]'),
  ];

  const visibleElement = await findFirstCurrentlyVisible(page, loginFieldSelectors);
  return visibleElement !== null;
}

async function captureDebugArtifacts(page, label, options, log) {
  const { onScreenshot, onHtmlSnapshot } = options || {};

  if (!page || isPageClosed(page)) {
    log(`⚠️  Cannot capture ${label} artifacts: page is closed or crashed`);
    return;
  }

  ensureDir(CONFIG.debugDir);

  const screenshotPath = path.join(CONFIG.debugDir, createDebugFileName(label, 'png'));
  try {
    await page.screenshot({ path: screenshotPath, fullPage: true, timeout: 15000 });
    log(`📸 Captured screenshot: ${screenshotPath}`);
    if (onScreenshot) onScreenshot(screenshotPath);
  } catch (error) {
    log(`⚠️  Failed to capture screenshot: ${error.message}`);
  }

  const htmlPath = path.join(CONFIG.debugDir, createDebugFileName(label, 'html'));
  try {
    const html = await page.content();
    fs.writeFileSync(htmlPath, html, 'utf-8');
    log(`🧾 Saved HTML snapshot: ${htmlPath}`);
    if (onHtmlSnapshot) onHtmlSnapshot(htmlPath);
  } catch (error) {
    log(`⚠️  Failed to capture HTML snapshot: ${error.message}`);
  }
}

async function waitForLikelyTwoFaChallenge(page, timeoutMs, log) {
  const twoFaSelectors = [
    root => root.locator('text=/Freigabe.*(SecureGo|App)|(SecureGo|App).*Freigabe/i'),
    root => root.locator('text=/Push.*(SecureGo|App)|(SecureGo|App).*Push/i'),
    root => root.locator('text=/TAN\\s*(eingeben|Eingabe)|Scan\\s*QR/i'),
    root => root.locator('[data-automation-id*="tan" i], [data-automation-id*="securego" i], [data-automation-id*="push" i]'),
  ];

  const deadline = Date.now() + timeoutMs;
  let loggedIgnoredMatch = false;
  let nextProgressAt = Date.now() + CONFIG.waitProgressIntervalMs;

  while (Date.now() < deadline) {
    if (isPageClosed(page)) {
      throw new Error('Page closed while waiting for 2FA challenge');
    }

    const visibleTwoFa = await findFirstCurrentlyVisible(page, twoFaSelectors);
    if (visibleTwoFa) {
      const loginFormVisible = await hasVisibleLoginForm(page);
      if (!loginFormVisible) {
        const roots = getRoots(page);
        const rootInfo = roots.length > 1 ? ` (frame: ${getRootUrl(visibleTwoFa.root)})` : '';
        log(`  Found 2FA challenge using selector #${visibleTwoFa.selectorIndex + 1}${rootInfo}`);
        return { type: '2fa' };
      }

      if (!loggedIgnoredMatch) {
        log('  Ignoring early 2FA text while login form is still visible');
        loggedIgnoredMatch = true;
      }
    }

    if (Date.now() >= nextProgressAt) {
      let currentUrl = 'unknown';
      try {
        currentUrl = page.url();
      } catch {
        // ignore
      }
      log(`  Still waiting for account overview or explicit 2FA challenge (url: ${currentUrl})`);
      nextProgressAt = Date.now() + CONFIG.waitProgressIntervalMs;
    }

    await page.waitForLoadState('domcontentloaded', { timeout: 300 }).catch(() => undefined);
  }

  throw new Error('Login timeout - check credentials or 2FA');
}

async function waitForLoginOutcome(page, accountSelectors, timeoutMs, log) {
  const accountPromise = findFirstVisible(page, accountSelectors, 'BFS Komfort account', timeoutMs, log).then(accountElement => ({
    type: 'account',
    accountElement,
  }));
  const twoFaPromise = waitForLikelyTwoFaChallenge(page, timeoutMs, log);

  try {
    return await Promise.any([accountPromise, twoFaPromise]);
  } catch {
    throw new Error('Login timeout - check credentials or 2FA');
  }
}

async function createBrowserContext() {
  const launchArgs = [
    '--no-sandbox',
    '--disable-setuid-sandbox',
    '--disable-dev-shm-usage',
    '--disable-gpu',
    '--disable-blink-features=AutomationControlled',
  ];

  if (CONFIG.usePersistentContext) {
    ensureDir(CONFIG.userDataDir);
    const context = await chromium.launchPersistentContext(CONFIG.userDataDir, {
      headless: CONFIG.headless,
      acceptDownloads: true,
      viewport: { width: 1280, height: 720 },
      userAgent: CONFIG.userAgent,
      locale: 'de-DE',
      args: launchArgs,
    });
    return { browser: null, context };
  }

  const browser = await chromium.launch({
    headless: CONFIG.headless,
    args: launchArgs,
  });
  const context = await browser.newContext({
    acceptDownloads: true,
    viewport: { width: 1280, height: 720 },
    userAgent: CONFIG.userAgent,
    locale: 'de-DE',
  });
  return { browser, context };
}

async function closeBrowserContext(context, browser) {
  try {
    if (context) await context.close();
  } finally {
    if (browser) await browser.close().catch(() => undefined);
  }
}

async function downloadCSV(options = {}) {
  const { onStatus, onLog, onScreenshot, onHtmlSnapshot, signal } = options;
  const log = createLogger(onLog);

  log('🚀 Starting banking sync...');

  if (!CONFIG.username || !CONFIG.password) {
    throw new Error('BANK_USERNAME and BANK_PASSWORD required');
  }

  // Create abort controller for this run
  abortController = new AbortController();
  const localSignal = signal || abortController.signal;

  // Set global timeout
  const globalTimeoutId = setTimeout(() => {
    log(`⏰ Global timeout (${CONFIG.globalTimeoutMs / 1000}s) exceeded, aborting...`);
    if (abortController) {
      abortController.abort();
    }
  }, CONFIG.globalTimeoutMs);

  ensureDir(CONFIG.downloadDir);
  const { browser, context } = await createBrowserContext();

  // Store globally for cancellation
  browserInstance = browser;
  browserContext = context;

  const page = await context.newPage();
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
  });

  // Helper to check if aborted
  const checkAborted = () => {
    if (localSignal.aborted) {
      throw new Error('Sync cancelled by user');
    }
  };

  try {
    // Check for cancellation before major operations
    checkAborted();
    // 1. Navigate and login
    log('📱 Navigating to login...');
    await page.goto(CONFIG.bankUrl, { waitUntil: 'domcontentloaded' });
    await dismissCookieBanner(page);

    log('🔑 Entering credentials...');
    await fillCredentials(page, log);

    // 2. Wait for login or 2FA
    log('⏳ Waiting for login/2FA...');

    const accountSelector = [
      // Primary: Find by IBAN data attribute
      root => root.locator('[data-e2e-konto-business-ident="DE33370205000003321400"] button.konto-item-action'),
      root => root.locator('[data-e2e-konto-business-ident="DE33370205000003321400"]'),
      // Fallback: First konto-list-item with clickable area
      root => root.locator('app-konto-item').first().locator('button.konto-item-action'),
      root => root.locator('app-konto-list-item').first().locator('.konto-list-item'),
      root => root.locator('app-konto-item').first(),
      // Last resort: first account list item
      root => root.locator('app-konto-list-item').first(),
    ];

    let accountElement;
    const loginOutcome = await waitForLoginOutcome(page, accountSelector, CONFIG.loginOutcomeTimeoutMs, log);
    if (loginOutcome.type === '2fa') {
      if (onStatus) onStatus('waiting_for_2fa');
      log('⚠️  2FA required - please approve in SecureGo Plus app');
      accountElement = await findFirstVisible(page, accountSelector, 'BFS Komfort account', CONFIG.twoFaTimeoutMs, log);
    } else {
      accountElement = loginOutcome.accountElement;
      log('  Found account element');
    }

    if (onStatus) onStatus('running');
    log('✅ Logged in successfully');

    // 3. Navigate to transactions
    log('📊 Navigating to transactions...');
    try {
      await accountElement.click();
    } catch (error) {
      log('  Account click intercepted, retrying with force click...');
      await accountElement.click({ force: true });
    }

    // 4. Download CSV
    log('💾 Downloading CSV...');
    const openExportButton = await findFirstVisible(
      page,
      [
        root => root.getByRole('button', { name: 'Exportieren: Modal öffnen zum' }),
        root => root.getByRole('button', { name: /^Exportieren$/ }),
      ],
      'export open button',
      30000,
      log
    );
    await openExportButton.click();

    const csvOption = await findFirstVisible(
      page,
      [root => root.locator('label').filter({ hasText: 'CSV' }), root => root.getByText('CSV', { exact: true })],
      'CSV option',
      30000,
      log
    );
    await csvOption.click();

    const downloadPromise = page.waitForEvent('download');
    const confirmExportButton = await findFirstVisible(
      page,
      [root => root.getByRole('button', { name: /^Exportieren$/ })],
      'export confirm button',
      30000,
      log
    );
    await confirmExportButton.click();
    const download = await downloadPromise;

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const fileName = `sozialbank_${timestamp}_${download.suggestedFilename()}`;
    const targetPath = path.join(CONFIG.downloadDir, fileName);
    await download.saveAs(targetPath);

    log(`✅ Downloaded ${fs.statSync(targetPath).size} bytes to ${targetPath}`);
    clearTimeout(globalTimeoutId);
    await closeBrowserContext(context, browser);
    browserContext = null;
    browserInstance = null;
    abortController = null;
    return targetPath;
  } catch (error) {
    clearTimeout(globalTimeoutId);
    
    // Check if cancelled
    if (localSignal.aborted) {
      log('⚠️ Sync was cancelled');
    }
    
    try {
      log(`📍 Current URL at error: ${page.url()}`);
    } catch {
      // ignore
    }
    try {
      const frameInfo = getRoots(page).map(getRootUrl).join(', ');
      log(`🧭 Frames at error: ${frameInfo}`);
    } catch {
      // ignore
    }

    await captureDebugArtifacts(page, 'error_state', { onScreenshot, onHtmlSnapshot }, log);
    log(`❌ Error: ${error.message}`);
    await closeBrowserContext(context, browser);
    browserContext = null;
    browserInstance = null;
    abortController = null;
    throw error;
  }
}

async function uploadToAPI(csvPath, options = {}) {
  const { onLog } = options;
  const log = createLogger(onLog);
  log('📤 Uploading to API...');

  if (!CONFIG.apiToken) {
    throw new Error('CRON_API_TOKEN required');
  }

  const fileBuffer = fs.readFileSync(csvPath);
  const form = new FormData();
  form.append('file', new Blob([fileBuffer], { type: 'text/csv' }), path.basename(csvPath));

  const response = await fetch(CONFIG.apiUrl.replace(/\/$/, '') + '/import/upload', {
    method: 'POST',
    headers: { 'X-Import-Token': CONFIG.apiToken },
    body: form,
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`API upload failed: ${response.status} ${error}`);
  }

  const result = await response.json();
  log(`✅ Upload successful: ${JSON.stringify(result)}`);
  return result;
}

async function main() {
  const isTest = process.argv.includes('--test');

  try {
    const csvPath = await downloadCSV();

    if (isTest) {
      const csvContent = fs.readFileSync(csvPath, 'utf-8');
      console.log('\n🧪 Test mode - CSV preview:');
      console.log(csvContent.substring(0, 500) + '...');
      console.log('\n✅ Test successful');
      return;
    }

    await uploadToAPI(csvPath);
    console.log('\n🎉 Banking sync completed!');
  } catch (error) {
    console.error('\n❌ Banking sync failed:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { downloadCSV, uploadToAPI, cancelSync, getAbortController };
