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
  dateRangeDays: Number(process.env.DATE_RANGE_DAYS || 90),
  twoFaTimeoutMs: Number(process.env.TWO_FA_TIMEOUT_SECONDS || 600) * 1000,
  screenshotDir: process.env.SCREENSHOT_DIR || process.env.DOWNLOAD_DIR || path.resolve(__dirname, 'output'),
  debugScreenshots: process.env.DEBUG_SCREENSHOTS === 'true',
  loginTimeoutMs: Number(process.env.LOGIN_TIMEOUT_SECONDS || 30) * 1000,
  traceEnabled: process.env.DEBUG_TRACE === 'true',
  traceDir:
    process.env.TRACE_DIR || process.env.SCREENSHOT_DIR || process.env.DOWNLOAD_DIR || path.resolve(__dirname, 'output'),
  userAgent:
    process.env.USER_AGENT ||
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
};

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function joinUrl(base, suffix) {
  return `${base.replace(/\/$/, '')}${suffix}`;
}

function createLogger(onLog) {
  return message => {
    console.log(message);
    if (onLog) {
      onLog(message);
    }
  };
}

function getRootUrl(root) {
  try {
    if (typeof root.url === 'function') {
      return root.url();
    }
  } catch (error) {
    return 'unknown';
  }
  return 'unknown';
}

function getRoots(page) {
  const frames = page.frames();
  if (!frames.length) return [page];
  return [page, ...frames];
}

async function findFirstVisible(page, builders, label, timeoutMs = 10000) {
  const roots = getRoots(page);
  for (const root of roots) {
    for (const build of builders) {
      let locator;
      try {
        locator = build(root);
      } catch (error) {
        continue;
      }
      const candidate = locator.first();
      try {
        await candidate.waitFor({ state: 'visible', timeout: timeoutMs });
        return candidate;
      } catch (error) {
        // try next
      }
    }
  }
  const frameInfo = roots.map(root => getRootUrl(root)).join(', ');
  throw new Error(`Could not find visible element for ${label}. Frames: ${frameInfo}`);
}

async function clickIfVisible(page, builders, label) {
  try {
    const element = await findFirstVisible(page, builders, label, 2000);
    await element.click().catch(() => undefined);
    return true;
  } catch (error) {
    return false;
  }
}

async function dismissCookieBanner(page, log) {
  const buttons = [
    root => root.getByRole('button', { name: /Alle akzeptieren|Akzeptieren|Zustimmen|Accept all|Accept/i }),
    root => root.locator('button:has-text("Alle akzeptieren")'),
    root => root.locator('button:has-text("Akzeptieren")'),
    root => root.locator('button:has-text("Zustimmen")'),
  ];

  try {
    const button = await findFirstVisible(page, buttons, 'cookie banner', 1500);
    await button.click().catch(() => undefined);
    log('🍪 Cookie banner dismissed');
  } catch (error) {
    // Ignore if not present
  }
}

async function fillCredentials(page, log) {
  await clickIfVisible(
    page,
    [root => root.getByRole('tab', { name: /Zugangsdaten/i }), root => root.locator('button:has-text("Mit Zugangsdaten anmelden")')],
    'login tab'
  );

  const usernameCandidates = [
    root => root.locator('[data-automation-id="vvrnKey-input"]'),
    root => root.locator('input[name="vvrnKeyFormControl"]'),
    root => root.locator('input#vvrnKey'),
    root => root.getByRole('textbox', { name: /NetKey|Alias|Benutzer|User|Login/i }),
    root => root.getByLabel(/NetKey|Alias|Benutzer|User|Login/i),
    root => root.locator('input[autocomplete="username"]'),
    root => root.locator('input[name*="user" i], input[name*="login" i]'),
    root => root.locator('input[type="text"]'),
  ];

  const passwordCandidates = [
    root => root.locator('[data-automation-id="pin-input"]'),
    root => root.locator('input[name="pinFormControl"]'),
    root => root.locator('input#pin'),
    root => root.getByLabel(/PIN|Passwort|Password/i),
    root => root.getByRole('textbox', { name: /PIN|Passwort|Password/i }),
    root => root.locator('input[autocomplete="current-password"]'),
    root => root.locator('input[name*="pin" i], input[name*="password" i]'),
    root => root.locator('input[type="password"]'),
  ];

  const submitCandidates = [
    root => root.locator('[data-automation-id="sign-in-button"]'),
    root => root.locator('app-signin-button button'),
    root => root.locator('button:has-text("Anmelden")'),
    root => root.getByRole('button', { name: /Log in|Login|Anmelden|Einloggen|Weiter/i }),
    root => root.locator('button[type="submit"]'),
    root => root.locator('input[type="submit"]'),
  ];

  const usernameInput = await findFirstVisible(page, usernameCandidates, 'username', CONFIG.loginTimeoutMs);
  await usernameInput.fill(CONFIG.username);

  const passwordInput = await findFirstVisible(page, passwordCandidates, 'pin', CONFIG.loginTimeoutMs);
  await passwordInput.fill(CONFIG.password);

  const submitButton = await findFirstVisible(page, submitCandidates, 'login button', CONFIG.loginTimeoutMs);
  await submitButton.click();
}

async function isPageValid(page) {
  try {
    await page.evaluate(() => document.title);
    return true;
  } catch {
    return false;
  }
}

async function captureArtifacts(page, log, onScreenshot, onHtmlSnapshot, label) {
  const isValid = await isPageValid(page).catch(() => false);
  if (!isValid) {
    log(`⚠️  Cannot capture ${label} artifacts: page is closed or crashed`);
    return;
  }

  try {
    ensureDir(CONFIG.screenshotDir);
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const screenshotPath = path.join(CONFIG.screenshotDir, `${label}_${timestamp}.png`);
    await page.screenshot({ path: screenshotPath, fullPage: false, timeout: 10000, animations: 'disabled' });
    log(`📸 Saved screenshot to ${screenshotPath}`);
    if (onScreenshot) {
      onScreenshot(screenshotPath);
    }
  } catch (screenshotError) {
    log(`⚠️  Failed to capture screenshot: ${screenshotError.message}`);
  }

  try {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const htmlPath = path.join(CONFIG.screenshotDir, `${label}_${timestamp}.html`);
    const content = await page.content();
    fs.writeFileSync(htmlPath, content, 'utf-8');
    log(`🧾 Saved HTML snapshot to ${htmlPath}`);
    if (onHtmlSnapshot) {
      onHtmlSnapshot(htmlPath);
    }
  } catch (htmlError) {
    log(`⚠️  Failed to capture HTML snapshot: ${htmlError.message}`);
  }
}

async function startTracing(context, log) {
  if (!CONFIG.traceEnabled) return false;
  try {
    ensureDir(CONFIG.traceDir);
    await context.tracing.start({ screenshots: true, snapshots: true, sources: false });
    log('🧵 Tracing enabled');
    return true;
  } catch (error) {
    log(`⚠️  Failed to start tracing: ${error.message}`);
    return false;
  }
}

async function stopTracing(context, log, onTrace, label, save) {
  if (!CONFIG.traceEnabled) return false;
  try {
    if (!save) {
      await context.tracing.stop();
      return false;
    }
    ensureDir(CONFIG.traceDir);
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const tracePath = path.join(CONFIG.traceDir, `${label}_${timestamp}.zip`);
    await context.tracing.stop({ path: tracePath });
    log(`🧵 Saved trace to ${tracePath}`);
    if (onTrace) {
      onTrace(tracePath);
    }
    return true;
  } catch (error) {
    log(`⚠️  Failed to stop tracing: ${error.message}`);
    return false;
  }
}

async function downloadCSV(options = {}) {
  const { onStatus, onLog, onScreenshot, onHtmlSnapshot, onTrace } = options;
  const log = createLogger(onLog);

  log('🚀 Starting banking sync...');
  log(`   URL: ${CONFIG.bankUrl}`);

  if (!CONFIG.username || !CONFIG.password) {
    throw new Error('BANK_USERNAME and BANK_PASSWORD required');
  }

  ensureDir(CONFIG.downloadDir);
  ensureDir(CONFIG.userDataDir);

  const context = await chromium.launchPersistentContext(CONFIG.userDataDir, {
    headless: CONFIG.headless,
    acceptDownloads: true,
    viewport: { width: 1280, height: 720 },
    userAgent: CONFIG.userAgent,
    locale: 'de-DE',
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--disable-blink-features=AutomationControlled',
    ],
  });

  const page = await context.newPage();
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
  });

  let traceStarted = false;
  let traceSaved = false;

  try {
    traceStarted = await startTracing(context, log);

    // 1. Login page (recorded via playwright codegen)
    log('📱 Navigating to login...');
    await page.goto(CONFIG.bankUrl);
    await page.waitForLoadState('domcontentloaded');
    await dismissCookieBanner(page, log);
    if (CONFIG.debugScreenshots) {
      await captureArtifacts(page, log, onScreenshot, onHtmlSnapshot, 'login_page');
    }

    // Fill credentials
    log('🔑 Entering credentials...');
    await captureArtifacts(page, log, onScreenshot, onHtmlSnapshot, 'before_credentials');
    await fillCredentials(page, log);

    // 2. Wait for login or 2FA
    log('⏳ Waiting for login/2FA...');
    const transactionsButton = page.getByRole('button', { name: 'Umsätze von BFS Komfort' });
    try {
      await transactionsButton.waitFor({ state: 'visible', timeout: 60000 });
    } catch (error) {
      const secureGoVisible = await page
        .locator('text=/SecureGo|TAN|Freigabe|2FA/i')
        .first()
        .isVisible()
        .catch(() => false);
      if (secureGoVisible) {
        if (onStatus) {
          onStatus('waiting_for_2fa');
        }
        log('⚠️  2FA required - please approve in SecureGo Plus app');
        await transactionsButton.waitFor({ state: 'visible', timeout: CONFIG.twoFaTimeoutMs });
      } else {
        throw new Error('Login timeout - check credentials or 2FA');
      }
    }

    if (onStatus) {
      onStatus('running');
    }
    log('✅ Logged in successfully');

    // 3. Navigate to transactions (recorded)
    log('📊 Navigating to transactions...');
    await transactionsButton.click();

    // 4. Download CSV (recorded)
    log('💾 Downloading CSV...');
    await page.getByRole('button', { name: 'Exportieren: Modal öffnen zum' }).click();
    await page.locator('label').filter({ hasText: 'CSV' }).click();
    const downloadPromise = page.waitForEvent('download');
    await page.getByRole('button', { name: 'Exportieren' }).click();
    const download = await downloadPromise;

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const suggestedName = download.suggestedFilename();
    const fileName = `sozialbank_${timestamp}_${suggestedName}`;
    const targetPath = path.join(CONFIG.downloadDir, fileName);
    await download.saveAs(targetPath);

    const fileSize = fs.statSync(targetPath).size;
    log(`✅ Downloaded ${fileSize} bytes to ${targetPath}`);

    if (traceStarted) {
      await stopTracing(context, log, onTrace, 'trace_success', false);
      traceStarted = false;
    }

    await context.close();

    return targetPath;
  } catch (error) {
    log(`❌ Error during sync: ${error.message}`);
    
    // Try to capture current URL for debugging
    try {
      const currentUrl = page.url();
      log(`📍 Current URL at error: ${currentUrl}`);
    } catch {
      log('📍 Could not get current URL (page may be closed)');
    }
    
    // Capture artifacts before stopping trace
    await captureArtifacts(page, log, onScreenshot, onHtmlSnapshot, 'error_state');
    
    // Stop tracing and save on error
    if (traceStarted) {
      traceSaved = await stopTracing(context, log, onTrace, 'trace_error', true);
      traceStarted = false;
    }
    
    await context.close();
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

  const response = await fetch(joinUrl(CONFIG.apiUrl, '/import/upload'), {
    method: 'POST',
    headers: {
      'X-Import-Token': CONFIG.apiToken,
    },
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
      console.log('\n🧪 Test mode - CSV content preview:');
      console.log(csvContent.substring(0, 500) + '...');
      console.log('\n✅ Test successful - ready for production');
      return;
    }

    await uploadToAPI(csvPath);
    console.log('\n🎉 Banking sync completed successfully!');
  } catch (error) {
    console.error('\n❌ Banking sync failed:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = { downloadCSV, uploadToAPI };
