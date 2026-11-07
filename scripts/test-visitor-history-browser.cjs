const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');

(async () => {
  const browser = await chromium.launch({headless:true});
  try {
    for (const width of [320, 1280]) {
      const page = await browser.newPage({viewport:{width,height:850}});
      const errors = [];
      let unavailable = false;
      page.on('pageerror', error => errors.push(error.message));
      await page.route('http://history.test/**', route => {
        const url = new URL(route.request().url());
        if (url.pathname.startsWith('/api/admin/')) {
          assert.equal(route.request().headers().authorization, 'Bearer test-token');
          if (url.pathname.endsWith('/session')) return route.fulfill({json:{authenticated:true}});
          if (unavailable) return route.fulfill({status:500,json:{message:'Unavailable'}});
          const older = url.searchParams.get('historyPage') === '1';
          return route.fulfill({json:{online:2,today:8,historyPage:older?1:0,hasMoreHistory:!older,
            history:older?[{day:'2026-09-01',visitors:0}]:[{day:'2026-10-01',visitors:8}]}});
        }
        if (url.pathname === '/api/interviews') return route.fulfill({json:{items:[],total:0,pageSize:12}});
        const name = url.pathname.slice(1);
        return route.fulfill({body:fs.readFileSync(path.join(__dirname,'../web',name)),
          contentType:name.endsWith('.js')?'text/javascript':name.endsWith('.css')?'text/css':'text/html'});
      });
      await page.goto('http://history.test/admin.html');
      assert.equal(await page.locator('#admin-content').isVisible(), false);
      await page.locator('#admin-token').fill('test-token');
      await page.getByRole('button',{name:'Sign in',exact:true}).click();
      await page.locator('#visitor-history-body').filter({hasText:'2026-10-01'}).waitFor();
      assert.equal(await page.locator('#visitors-today').textContent(), '8');
      assert.equal(await page.locator('#visitor-history-newer').isDisabled(), true);
      await page.locator('#visitor-history-older').click();
      await page.locator('#visitor-history-body').filter({hasText:'2026-09-01'}).waitFor();
      assert.equal(await page.locator('#visitor-history-body td').last().textContent(), '0');
      assert.equal(await page.locator('#visitor-history-older').isDisabled(), true);
      await page.locator('#visitor-history-newer').click();
      await page.locator('#visitor-history-body').filter({hasText:'2026-10-01'}).waitFor();
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      unavailable = true;
      await page.locator('#visitor-history-older').click();
      await page.locator('#visitor-history-empty').filter({hasText:'History unavailable'}).waitFor();
      assert.equal(await page.locator('#visitor-history-body tr').count(), 0);
      await page.getByRole('button',{name:'Sign out',exact:true}).click();
      assert.equal(await page.locator('#admin-content').isVisible(), false);
      assert.deepEqual(errors, []);
      await page.close();
    }
    console.log('Visitor history browser checks passed: sign-in, totals, pagination, zero days, errors, sign-out, mobile and desktop.');
  } finally { await browser.close(); }
})().catch(error => {console.error(error);process.exit(1);});
