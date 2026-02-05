const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const { downloadCSV, uploadToAPI } = require('./sync');

const CONFIG = {
  port: Number(process.env.PORT || 3333),
  syncToken: process.env.SYNC_API_TOKEN || '',
  stateDir: process.env.STATE_DIR || path.resolve(__dirname, 'state'),
  maxLogLines: Number(process.env.LOG_LINES || 200),
};

const STATE_FILE = path.join(CONFIG.stateDir, 'status.json');

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function nowIso() {
  return new Date().toISOString();
}

const defaultState = {
  status: 'idle',
  runId: null,
  startedAt: null,
  finishedAt: null,
  lastError: null,
  lastMessage: null,
  downloadPath: null,
  uploadResult: null,
  lastScreenshot: null,
  lastHtmlSnapshot: null,
  logs: [],
  updatedAt: nowIso(),
};

function loadState() {
  try {
    if (fs.existsSync(STATE_FILE)) {
      const raw = fs.readFileSync(STATE_FILE, 'utf-8');
      const parsed = JSON.parse(raw);
      return { ...defaultState, ...parsed };
    }
  } catch (error) {
    console.warn('Failed to load state file:', error.message);
  }
  return { ...defaultState };
}

function saveState() {
  const payload = JSON.stringify(state, null, 2);
  fs.writeFileSync(STATE_FILE, payload);
}

function updateState(partial) {
  state = {
    ...state,
    ...partial,
    updatedAt: nowIso(),
  };
  saveState();
  return state;
}

function appendLog(message) {
  const entry = `${nowIso()} ${message}`;
  const logs = [...(state.logs || []), entry].slice(-CONFIG.maxLogLines);
  updateState({ logs, lastMessage: message });
}

function recoverStaleInProgressState() {
  if (state.status !== 'running' && state.status !== 'waiting_for_2fa') {
    return;
  }

  const recoveredAt = nowIso();
  const reason = 'Runner restarted while sync was in progress; state recovered.';
  const logs = [...(state.logs || []), `${recoveredAt} ⚠️ ${reason}`].slice(-CONFIG.maxLogLines);

  state = {
    ...state,
    status: 'error',
    finishedAt: state.finishedAt || recoveredAt,
    lastError: reason,
    lastMessage: reason,
    logs,
    updatedAt: recoveredAt,
  };
  saveState();
}

function isAuthorized(req) {
  if (!CONFIG.syncToken) {
    return true;
  }
  const token = req.headers['x-sync-token'];
  return token === CONFIG.syncToken;
}

function respondJson(res, status, payload) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(payload));
}

async function readJson(req) {
  return new Promise(resolve => {
    let data = '';
    req.on('data', chunk => {
      data += chunk;
    });
    req.on('end', () => {
      if (!data) return resolve(null);
      try {
        resolve(JSON.parse(data));
      } catch {
        resolve(null);
      }
    });
  });
}

let state;
let running = false;

ensureDir(CONFIG.stateDir);
state = loadState();
recoverStaleInProgressState();

async function runSync({ test = false } = {}) {
  if (running) {
    return state;
  }

  const runId = crypto.randomUUID();
  running = true;
  updateState({
    status: 'running',
    runId,
    startedAt: nowIso(),
    finishedAt: null,
    lastError: null,
    lastMessage: null,
    downloadPath: null,
    uploadResult: null,
    lastScreenshot: null,
    lastHtmlSnapshot: null,
  });

  try {
    const csvPath = await downloadCSV({
      onStatus: status => {
        if (status === 'waiting_for_2fa') {
          updateState({ status: 'waiting_for_2fa' });
        }
        if (status === 'running') {
          updateState({ status: 'running' });
        }
      },
      onLog: appendLog,
      onScreenshot: screenshotPath => {
        updateState({ lastScreenshot: screenshotPath });
      },
      onHtmlSnapshot: htmlPath => {
        updateState({ lastHtmlSnapshot: htmlPath });
      },
    });

    updateState({ downloadPath: csvPath });

    if (!test) {
      const result = await uploadToAPI(csvPath, { onLog: appendLog });
      updateState({ uploadResult: result });
    }

    updateState({ status: 'success', finishedAt: nowIso() });
  } catch (error) {
    updateState({
      status: 'error',
      finishedAt: nowIso(),
      lastError: error && error.message ? error.message : String(error),
    });
  } finally {
    running = false;
  }

  return state;
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', 'http://localhost');
  const pathname = url.pathname;

  if (pathname === '/health' && req.method === 'GET') {
    return respondJson(res, 200, { status: 'ok' });
  }

  if (!isAuthorized(req)) {
    return respondJson(res, 401, { error: 'unauthorized' });
  }

  if (pathname === '/status' && req.method === 'GET') {
    return respondJson(res, 200, state);
  }

  if (pathname === '/run' && req.method === 'POST') {
    if (running || state.status === 'waiting_for_2fa') {
      return respondJson(res, 409, { error: 'run already in progress', status: state });
    }

    const body = await readJson(req);
    const testParam = url.searchParams.get('test');
    const test =
      (typeof body?.test === 'boolean' && body.test) ||
      (typeof testParam === 'string' && ['1', 'true', 'yes'].includes(testParam.toLowerCase()));

    void runSync({ test });
    return respondJson(res, 200, state);
  }

  respondJson(res, 404, { error: 'not found' });
});

server.listen(CONFIG.port, () => {
  if (!CONFIG.syncToken) {
    console.warn('SYNC_API_TOKEN not set. Runner endpoints are unauthenticated.');
  }
  console.log(`Banking sync runner listening on :${CONFIG.port}`);
});
