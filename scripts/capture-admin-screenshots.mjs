import puppeteer from 'puppeteer-core';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.join(__dirname, '../docs/screenshots');
const chromePath = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

const tabs = [
  ['overview', 'How it works'],
  ['clients', 'Clients'],
  ['segments', 'Segments'],
  ['rules', 'Rules'],
  ['runs', 'Campaign runs'],
];

const wait = (ms) => new Promise((r) => setTimeout(r, ms));

async function main() {
  fs.mkdirSync(outDir, { recursive: true });

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: 'new',
    defaultViewport: { width: 1280, height: 900 },
  });

  const page = await browser.newPage();
  await page.goto('http://127.0.0.1:8080/admin/', { waitUntil: 'networkidle0', timeout: 15000 });

  for (const [id] of tabs) {
    await page.click(`button[data-tab="${id}"]`);
    await wait(600);
    await page.screenshot({ path: path.join(outDir, `${id}.png`), fullPage: true });
  }

  await page.click('button[data-tab="overview"]');
  await page.click('#runCampaign');
  await wait(2500);
  await page.screenshot({ path: path.join(outDir, 'campaign-results.png'), fullPage: true });

  await browser.close();
  console.log('Screenshots saved to', outDir);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
