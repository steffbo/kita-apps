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
  twoFaTimeoutMs: Number(process.env.TWO_FA_TIMEOUT_SECONDS || 600) * 1000,
  loginTimeoutMs: Number(process.env.LOGIN_TIMEOUT_SECONDS || 30) * 1000,
  userAgent:
    process.env.USER_AGENT ||
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
};

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
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

async function findFirstVisible(page, builders, label, timeoutMs = 10000, log = null) {
  const roots = getRoots(page);
  for (const root of roots) {
    for (let i = 0; i < builders.length; i++) {
      const build = builders[i];
      let locator;
      try {
        locator = build(root);
      } catch {
        continue;
      }

      try {
        const deadline = Date.now() + timeoutMs;

        while (Date.now() < deadline) {
          const count = await locator.count();
          const limit = Math.min(count, 5);
          for (let idx = 0; idx < limit; idx++) {
            const candidate = locator.nth(idx);
            const visible = await candidate.isVisible().catch(() => false);
            if (visible) {
              if (log) {
                const rootInfo = roots.length > 1 ? ` (frame: ${getRootUrl(root)})` : '';
                log(`  Found ${label} using selector #${i + 1}${rootInfo}`);
              }
              return candidate;
            }
          }
          await page.waitForTimeout(200);
        }
      } catch {
        // try next
      }
    }
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

  // Wait for form to be fully interactive
  await page.waitForTimeout(500);

  log('  Looking for username field...');
  const usernameInput = await findFirstVisible(
    page,
    [
      root => root.locator('[data-automation-id="vvrnKey-input"]'),
      root => root.locator('input[name="vvrnKeyFormControl"]'),
      root => root.locator('input#vvrnKey'),
      // Fallback for older naming
      root => root.locator('[data-automation-id="vrNetKey-input"]'),
      root => root.locator('input[name="vrNetKeyFormControl"]'),
      root => root.locator('input#vrNetKey'),
      root => root.getByRole('textbox', { name: /NetKey|Alias|Benutzer|User|Login/i }),
      root => root.getByLabel(/NetKey|Alias|Benutzer|User|Login/i),
      root => root.locator('input[autocomplete="username"]'),
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
      root => root.locator('[data-automation-id="pin-input"]'),
      root => root.locator('input[name="pinFormControl"]'),
      root => root.locator('input#pin'),
      root => root.getByLabel(/PIN|Passwort|Password/i),
      root => root.locator('input[autocomplete="current-password"]'),
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

  // Wait for navigation
  await page.waitForTimeout(2000);
}

async function downloadCSV(options = {}) {
  const { onStatus, onLog } = options;
  const log = createLogger(onLog);

  log('🚀 Starting banking sync...');

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

  try {
    // 1. Navigate and login
    log('📱 Navigating to login...');
    await page.goto(CONFIG.bankUrl);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    await dismissCookieBanner(page);

    log('🔑 Entering credentials...');
    await fillCredentials(page, log);

    // 2. Wait for login or 2FA
    log('⏳ Waiting for login/2FA...');
    
    // Wait for account list to load
    await page.waitForTimeout(2000);
    
    const accountSelector = [
      // Primary: Find by IBAN data attribute
      root => root.locator('[data-e2e-konto-business-ident="DE33370205000003321400"]'),
      // Fallback: First konto-list-item with clickable area
      root => root.locator('app-konto-list-item').first().locator('.konto-list-item'),
      root => root.locator('app-konto-item').first(),
      // Last resort: first account list item
      root => root.locator('app-konto-list-item').first(),
    ];
    
    let accountElement;
    try {
      accountElement = await findFirstVisible(page, accountSelector, 'BFS Komfort account', 60000);
      log('  Found account element');
    } catch (error) {
      const secureGoVisible = await page
        .locator('text=/SecureGo|TAN|Freigabe|2FA/i')
        .first()
        .isVisible()
        .catch(() => false);
      if (secureGoVisible) {
        if (onStatus) onStatus('waiting_for_2fa');
        log('⚠️  2FA required - please approve in SecureGo Plus app');
        accountElement = await findFirstVisible(page, accountSelector, 'BFS Komfort account', CONFIG.twoFaTimeoutMs);
      } else {
        throw new Error('Login timeout - check credentials or 2FA');
      }
    }

    if (onStatus) onStatus('running');
    log('✅ Logged in successfully');

    // 3. Navigate to transactions
    log('📊 Navigating to transactions...');
    await accountElement.click();

    // 4. Download CSV
    log('💾 Downloading CSV...');
    await page.getByRole('button', { name: 'Exportieren: Modal öffnen zum' }).click();
    await page.locator('label').filter({ hasText: 'CSV' }).click();
    const downloadPromise = page.waitForEvent('download');
    await page.getByRole('button', { name: 'Exportieren' }).click();
    const download = await downloadPromise;

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const fileName = `sozialbank_${timestamp}_${download.suggestedFilename()}`;
    const targetPath = path.join(CONFIG.downloadDir, fileName);
    await download.saveAs(targetPath);

    log(`✅ Downloaded ${fs.statSync(targetPath).size} bytes to ${targetPath}`);
    await context.close();
    return targetPath;
  } catch (error) {
    log(`❌ Error: ${error.message}`);
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

module.exports = { downloadCSV, uploadToAPI };
