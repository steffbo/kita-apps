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

async function findFirstVisible(locators, label, timeoutMs = 2000) {
  for (const locator of locators) {
    const candidate = locator.first();
    try {
      await candidate.waitFor({ state: 'visible', timeout: timeoutMs });
      return candidate;
    } catch (error) {
      // try next
    }
  }
  throw new Error(`Could not find visible element for ${label}`);
}

async function dismissCookieBanner(page, log) {
  const buttons = [
    page.getByRole('button', { name: /Alle akzeptieren|Akzeptieren|Zustimmen|Accept all|Accept/i }),
    page.locator('button:has-text("Alle akzeptieren")'),
    page.locator('button:has-text("Akzeptieren")'),
    page.locator('button:has-text("Zustimmen")'),
  ];

  try {
    const button = await findFirstVisible(buttons, 'cookie banner', 1500);
    await button.click().catch(() => undefined);
    log('🍪 Cookie banner dismissed');
  } catch (error) {
    // Ignore if not present
  }
}

async function fillCredentials(page, log) {
  const usernameCandidates = [
    page.locator('[data-automation-id="vvrnKey-input"]'),
    page.locator('input[name="vvrnKeyFormControl"]'),
    page.locator('input#vvrnKey'),
    page.getByRole('textbox', { name: /NetKey|Alias|Benutzer|User|Login/i }),
    page.getByLabel(/NetKey|Alias|Benutzer|User|Login/i),
    page.locator('input[autocomplete="username"]'),
    page.locator('input[name*="user" i], input[name*="login" i]'),
    page.locator('input[type="text"]'),
  ];

  const passwordCandidates = [
    page.locator('[data-automation-id="pin-input"]'),
    page.locator('input[name="pinFormControl"]'),
    page.locator('input#pin'),
    page.getByLabel(/PIN|Passwort|Password/i),
    page.getByRole('textbox', { name: /PIN|Passwort|Password/i }),
    page.locator('input[autocomplete="current-password"]'),
    page.locator('input[name*="pin" i], input[name*="password" i]'),
    page.locator('input[type="password"]'),
  ];

  const submitCandidates = [
    page.locator('[data-automation-id="sign-in-button"]'),
    page.locator('app-signin-button button'),
    page.locator('button:has-text("Anmelden")'),
    page.getByRole('button', { name: /Log in|Login|Anmelden|Einloggen|Weiter/i }),
    page.locator('button[type="submit"]'),
    page.locator('input[type="submit"]'),
  ];

  const usernameInput = await findFirstVisible(usernameCandidates, 'username');
  await usernameInput.fill(CONFIG.username);

  const passwordInput = await findFirstVisible(passwordCandidates, 'pin');
  await passwordInput.fill(CONFIG.password);

  const submitButton = await findFirstVisible(submitCandidates, 'login button');
  await submitButton.click();
}

async function downloadCSV(options = {}) {
  const { onStatus, onLog } = options;
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
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const page = await context.newPage();

  try {
    // 1. Login page (recorded via playwright codegen)
    log('📱 Navigating to login...');
    await page.goto(CONFIG.bankUrl);
    await page.waitForLoadState('domcontentloaded');
    await dismissCookieBanner(page, log);

    // Fill credentials
    log('🔑 Entering credentials...');
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

    await context.close();

    return targetPath;
  } catch (error) {
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
