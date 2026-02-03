const { Blob } = require('buffer');
const fs = require('fs');
const path = require('path');

const CONFIG = {
  apiUrl: process.env.API_URL || 'http://localhost:8081/api/fees/v1',
  apiToken: process.env.CRON_API_TOKEN,
};

function joinUrl(base, suffix) {
  return `${base.replace(/\/$/, '')}${suffix}`;
}

function getFilePath() {
  const argPath = process.argv.find(arg => arg.startsWith('--file='));
  if (argPath) {
    return argPath.slice('--file='.length);
  }
  return process.argv[2] || process.env.CSV_PATH;
}

async function uploadFile(filePath) {
  if (!CONFIG.apiToken) {
    throw new Error('CRON_API_TOKEN required');
  }

  if (!filePath) {
    throw new Error('CSV path required. Usage: bun upload.js --file=/path/to/file.csv');
  }

  const resolvedPath = path.resolve(filePath);
  if (!fs.existsSync(resolvedPath)) {
    throw new Error(`File not found: ${resolvedPath}`);
  }

  const fileBuffer = fs.readFileSync(resolvedPath);
  const form = new FormData();
  form.append('file', new Blob([fileBuffer], { type: 'text/csv' }), path.basename(resolvedPath));

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
  try {
    const filePath = getFilePath();
    await uploadFile(filePath);
  } catch (error) {
    console.error('\n❌ Upload failed:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { uploadFile };
