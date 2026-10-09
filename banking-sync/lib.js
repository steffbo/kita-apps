const path = require('path');

function createConfig(env = process.env, baseDir = __dirname) {
  const downloadDir = env.DOWNLOAD_DIR || path.resolve(baseDir, 'output');

  return {
    bankUrl:
      env.BANK_URL ||
      'https://www.sozialbank-onlinebanking.de/services_cloud/portal/',
    username: env.BANK_USERNAME,
    password: env.BANK_PASSWORD,
    apiUrl: env.API_URL || 'http://localhost:8081/api/fees/v1',
    apiToken: env.CRON_API_TOKEN,
    uptimeKumaPushUrl: env.UPTIME_KUMA_PUSH_URL || '',
    headless: env.HEADLESS !== 'false',
    downloadDir,
    userDataDir: env.USER_DATA_DIR || path.resolve(baseDir, 'profile'),
    usePersistentContext: env.USE_PERSISTENT_CONTEXT === 'true',
    twoFaTimeoutMs: Number(env.TWO_FA_TIMEOUT_SECONDS || 600) * 1000,
    loginTimeoutMs: Number(env.LOGIN_TIMEOUT_SECONDS || 30) * 1000,
    loginOutcomeTimeoutMs: Number(env.LOGIN_OUTCOME_TIMEOUT_SECONDS || 45) * 1000,
    waitProgressIntervalMs: Number(env.WAIT_PROGRESS_INTERVAL_SECONDS || 10) * 1000,
    debugDir: env.DEBUG_DIR || path.join(downloadDir, 'debug'),
    userAgent:
      env.USER_AGENT ||
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6_1) AppleWebKit/537.36 ' +
        '(KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
    globalTimeoutMs: Number(env.GLOBAL_TIMEOUT_SECONDS || 900) * 1000,
    uploadTimeoutMs: Number(env.UPLOAD_TIMEOUT_SECONDS || 120) * 1000,
    downloadTimeoutMs: Number(env.DOWNLOAD_TIMEOUT_SECONDS || 120) * 1000,
    downloadRetryAttempts: Number(env.DOWNLOAD_RETRY_ATTEMPTS || 3),
    downloadRetryDelayMs: Number(env.DOWNLOAD_RETRY_DELAY_SECONDS || 30) * 1000,
    browserCloseTimeoutMs: Number(env.BROWSER_CLOSE_TIMEOUT_SECONDS || 20) * 1000,
    uptimeKumaPingTimeoutMs: Number(env.UPTIME_KUMA_TIMEOUT_SECONDS || 10) * 1000,
  };
}

function getDownloadRetryPolicy(config) {
  const attempts = config.downloadRetryAttempts;
  const delayMs = config.downloadRetryDelayMs;

  return {
    attempts: !Number.isFinite(attempts) || attempts < 1 ? 1 : Math.floor(attempts),
    delayMs: !Number.isFinite(delayMs) || delayMs < 0 ? 0 : delayMs,
  };
}

function shouldRetryDownload(error) {
  const message = error && error.message ? error.message : String(error);
  return !/Sync cancelled by user|BANK_USERNAME and BANK_PASSWORD required/i.test(message);
}

function appendQueryParams(rawUrl, params) {
  const url = new URL(rawUrl);
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      url.searchParams.set(key, value);
    }
  }
  return url.toString();
}

function createUploadRequest(config, body, signal) {
  return {
    url: `${config.apiUrl.replace(/\/$/, '')}/import/upload`,
    options: {
      method: 'POST',
      headers: { 'X-Import-Token': config.apiToken },
      body,
      ...(signal ? { signal } : {}),
    },
  };
}

async function readUploadResponse(response) {
  if (!response.ok) {
    const error = await response.text();
    throw new Error(`API upload failed: ${response.status} ${error}`);
  }
  return response.json();
}

function formatTimestamp(date = new Date()) {
  return date.toISOString().replace(/[:.]/g, '-');
}

module.exports = {
  appendQueryParams,
  createConfig,
  createUploadRequest,
  formatTimestamp,
  getDownloadRetryPolicy,
  readUploadResponse,
  shouldRetryDownload,
};
