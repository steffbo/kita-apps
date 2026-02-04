import { test as base, expect } from '@playwright/test';
import fs from 'node:fs/promises';
import path from 'node:path';

const coverageEnabled = process.env.PW_COVERAGE === '1';
const coverageDir = process.env.PW_COVERAGE_DIR ?? 'coverage/beitraege/raw';

export const test = base.extend({
  page: async ({ page }, use, testInfo) => {
    if (coverageEnabled && page.coverage) {
      try {
        await page.coverage.startJSCoverage({ resetOnNavigation: false });
      } catch {
        // Ignore if coverage is unsupported in the current browser.
      }
    }

    await use(page);

    if (coverageEnabled && page.coverage) {
      try {
        const coverage = await page.coverage.stopJSCoverage();
        const safeId = testInfo.testId.replace(/[^a-zA-Z0-9_-]/g, '_');
        const fileName = `${testInfo.project.name}-${safeId}.json`;
        const filePath = path.join(coverageDir, fileName);
        await fs.mkdir(coverageDir, { recursive: true });
        await fs.writeFile(filePath, JSON.stringify(coverage));
      } catch {
        // Ignore if coverage is unsupported or already stopped.
      }
    }
  },
});

export { expect };
