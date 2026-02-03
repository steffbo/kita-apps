const { chromium } = require('playwright');
const { Blob } = require('buffer');
const fs = require('fs');
const path = require('path');

// Configuration from environment
const CONFIG = {
  bankUrl: process.env.BANK_URL || 'https://www.sozialbank-onlinebanking.de/services_cloud/portal/',
  username: process.env.BANK_USERNAME,
  password: process.env.BANK_PASSWORD,
  apiUrl: process.env.API_URL || 'http://localhost:8081/api/fees/v1',
  apiToken: process.env.CRON_API_TOKEN,
  headless: process.env.HEADLESS !== 'false',
  downloadDir: process.env.DOWNLOAD_DIR || path.resolve(__dirname, 'output'),
  userDataDir: process.env.USER_DATA_DIR || path.resolve(__dirname, 'profile'),
  dateRangeDays: Number(process.env.DATE_RANGE_DAYS || 90),
};

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function joinUrl(base, suffix) {
  return `${base.replace(/\/$/, '')}${suffix}`;
}

async function downloadCSV() {
  console.log('🚀 Starting banking sync...');
  console.log(`   URL: ${CONFIG.bankUrl}`);

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
    // 1. Login page
    console.log('📱 Navigating to login...');
    await page.goto(CONFIG.bankUrl, { waitUntil: 'networkidle' });

    // Wait for login form (adjust selectors based on actual page)
    await page.waitForSelector('input[type="text"], input[name="username"], #username', { timeout: 10000 });

    // Fill credentials
    console.log('🔑 Entering credentials...');
    await page.fill('input[type="text"], input[name="username"], #username', CONFIG.username);
    await page.fill('input[type="password"], input[name="password"], #password', CONFIG.password);

    // Submit login
    await page.click('button[type="submit"], input[type="submit"], .login-button');

    // 2. Handle 2FA if needed (first time or security check)
    console.log('⏳ Waiting for login/2FA...');
    try {
      // Wait for either dashboard OR 2FA prompt
      await Promise.race([
        page.waitForSelector('.dashboard, .account-overview, [data-testid="dashboard"]', { timeout: 30000 }),
        page.waitForSelector('.tan-prompt, .securego, [data-testid="2fa"]', { timeout: 30000 }),
      ]);

      // Check if we're on 2FA page
      const is2FA = await page.$('.tan-prompt, .securego, [data-testid="2fa"]');
      if (is2FA) {
        console.log('⚠️  2FA required - please approve in SecureGo Plus app');
        // Wait for user to approve (or timeout)
        await page.waitForSelector('.dashboard, .account-overview', { timeout: 120000 });
      }
    } catch (e) {
      throw new Error('Login timeout - check credentials or 2FA');
    }

    console.log('✅ Logged in successfully');

    // 3. Navigate to transactions
    console.log('📊 Navigating to transactions...');
    await page.click('a[href*="transaction"], a[href*="umsatz"], .menu-transactions');
    await page.waitForLoadState('networkidle');

    // 4. Set date range (last N days)
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - CONFIG.dateRangeDays);

    const startStr = startDate.toISOString().split('T')[0];
    const endStr = endDate.toISOString().split('T')[0];

    console.log(`📅 Setting date range: ${startStr} to ${endStr}`);

    // Fill date range (adjust selectors)
    await page.fill('input[name="startDate"], input[name="from"]', startStr);
    await page.fill('input[name="endDate"], input[name="to"]', endStr);

    // Search/Apply
    await page.click('button[type="submit"], .search-button, .apply-filter');
    await page.waitForLoadState('networkidle');

    // 5. Download CSV
    console.log('💾 Downloading CSV...');

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.click('button:has-text("CSV"), a:has-text("CSV"), .export-csv'),
    ]);

    const fileName = `sozialbank_${startStr}_to_${endStr}.csv`;
    const targetPath = path.join(CONFIG.downloadDir, fileName);
    await download.saveAs(targetPath);

    const fileSize = fs.statSync(targetPath).size;
    console.log(`✅ Downloaded ${fileSize} bytes to ${targetPath}`);

    await context.close();

    return targetPath;
  } catch (error) {
    await context.close();
    throw error;
  }
}

async function uploadToAPI(csvPath) {
  console.log('📤 Uploading to API...');

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
  console.log('✅ Upload successful:', result);
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
