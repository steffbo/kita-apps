const { describe, expect, test } = require('bun:test');
const {
  appendQueryParams,
  createConfig,
  createUploadRequest,
  formatTimestamp,
  getDownloadRetryPolicy,
  readUploadResponse,
  shouldRetryDownload,
} = require('./lib');

describe('createConfig', () => {
  test('provides the existing defaults', () => {
    const config = createConfig({}, '/app');

    expect(config.bankUrl).toBe('https://www.sozialbank-onlinebanking.de/services_cloud/portal/');
    expect(config.apiUrl).toBe('http://localhost:8081/api/fees/v1');
    expect(config.downloadDir).toBe('/app/output');
    expect(config.debugDir).toBe('/app/output/debug');
    expect(config.userDataDir).toBe('/app/profile');
    expect(config.headless).toBe(true);
    expect(config.downloadRetryAttempts).toBe(3);
    expect(config.downloadRetryDelayMs).toBe(30000);
    expect(config.uptimeKumaPingTimeoutMs).toBe(10000);
  });

  test('parses environment overrides with the existing conversions', () => {
    const config = createConfig(
      {
        BANK_URL: 'https://bank.example/',
        BANK_USERNAME: 'user',
        BANK_PASSWORD: 'secret',
        API_URL: 'https://api.example/v1',
        CRON_API_TOKEN: 'token',
        UPTIME_KUMA_PUSH_URL: 'https://status.example/push',
        HEADLESS: 'false',
        DOWNLOAD_DIR: '/data/downloads',
        USER_DATA_DIR: '/data/profile',
        USE_PERSISTENT_CONTEXT: 'true',
        TWO_FA_TIMEOUT_SECONDS: '12',
        LOGIN_TIMEOUT_SECONDS: '13',
        LOGIN_OUTCOME_TIMEOUT_SECONDS: '14',
        WAIT_PROGRESS_INTERVAL_SECONDS: '15',
        DEBUG_DIR: '/data/debug',
        USER_AGENT: 'custom agent',
        GLOBAL_TIMEOUT_SECONDS: '16',
        UPLOAD_TIMEOUT_SECONDS: '17',
        DOWNLOAD_TIMEOUT_SECONDS: '18',
        DOWNLOAD_RETRY_ATTEMPTS: '4',
        DOWNLOAD_RETRY_DELAY_SECONDS: '19',
        BROWSER_CLOSE_TIMEOUT_SECONDS: '20',
        UPTIME_KUMA_TIMEOUT_SECONDS: '21',
      },
      '/app'
    );

    expect(config).toMatchObject({
      bankUrl: 'https://bank.example/',
      username: 'user',
      password: 'secret',
      apiUrl: 'https://api.example/v1',
      apiToken: 'token',
      uptimeKumaPushUrl: 'https://status.example/push',
      headless: false,
      downloadDir: '/data/downloads',
      userDataDir: '/data/profile',
      usePersistentContext: true,
      twoFaTimeoutMs: 12000,
      loginTimeoutMs: 13000,
      loginOutcomeTimeoutMs: 14000,
      waitProgressIntervalMs: 15000,
      debugDir: '/data/debug',
      userAgent: 'custom agent',
      globalTimeoutMs: 16000,
      uploadTimeoutMs: 17000,
      downloadTimeoutMs: 18000,
      downloadRetryAttempts: 4,
      downloadRetryDelayMs: 19000,
      browserCloseTimeoutMs: 20000,
      uptimeKumaPingTimeoutMs: 21000,
    });
  });
});

describe('download retry policy', () => {
  test('normalizes invalid attempt counts and delays', () => {
    expect(getDownloadRetryPolicy({ downloadRetryAttempts: 0, downloadRetryDelayMs: -1 })).toEqual({
      attempts: 1,
      delayMs: 0,
    });
    expect(getDownloadRetryPolicy({ downloadRetryAttempts: 2.8, downloadRetryDelayMs: 0 })).toEqual({
      attempts: 2,
      delayMs: 0,
    });
    expect(
      getDownloadRetryPolicy({ downloadRetryAttempts: Number.NaN, downloadRetryDelayMs: Number.NaN })
    ).toEqual({ attempts: 1, delayMs: 0 });
  });

  test('does not retry cancellation or missing credentials', () => {
    expect(shouldRetryDownload(new Error('Sync cancelled by user'))).toBe(false);
    expect(shouldRetryDownload(new Error('BANK_USERNAME and BANK_PASSWORD required'))).toBe(false);
    expect(shouldRetryDownload(new Error('temporary browser failure'))).toBe(true);
  });
});

describe('upload helpers', () => {
  test('builds the upload request and appends query parameters', () => {
    const request = createUploadRequest(
      { apiUrl: 'https://api.example/v1/', apiToken: 'token' },
      'form-data',
      'abort-signal'
    );

    expect(request).toEqual({
      url: 'https://api.example/v1/import/upload',
      options: {
        method: 'POST',
        headers: { 'X-Import-Token': 'token' },
        body: 'form-data',
        signal: 'abort-signal',
      },
    });
    expect(appendQueryParams('https://status.example/push?status=old', {
      status: 'up',
      msg: 'Banking sync completed',
      ping: '',
      skip: null,
    })).toBe('https://status.example/push?status=up&msg=Banking+sync+completed');
  });

  test('returns successful JSON and reports API error responses', async () => {
    await expect(
      readUploadResponse({ ok: true, json: async () => ({ imported: 3 }) })
    ).resolves.toEqual({ imported: 3 });
    await expect(
      readUploadResponse({ ok: false, status: 422, text: async () => 'invalid CSV' })
    ).rejects.toThrow('API upload failed: 422 invalid CSV');
  });
});

test('formats ISO timestamps as filesystem-safe names', () => {
  expect(formatTimestamp(new Date('2026-10-09T12:34:56.789Z'))).toBe('2026-10-09T12-34-56-789Z');
});
