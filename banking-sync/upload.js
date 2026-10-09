const { Blob } = require('buffer');
const fs = require('fs');
const path = require('path');
const { createConfig, createUploadRequest, readUploadResponse } = require('./lib');

const CONFIG = createConfig();

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

  const request = createUploadRequest(CONFIG, form);
  const response = await fetch(request.url, request.options);
  const result = await readUploadResponse(response);
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
