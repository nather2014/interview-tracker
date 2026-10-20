const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
(async () => {
 const browser = await chromium.launch({headless:true});
 try {
  for (const width of [320,1280]) {
   const page = await browser.newPage({viewport:{width,height:850}});
   const errors = []; page.on('pageerror', e => errors.push(e.message));
   await page.route('http://bulk.test/**', route => {
    const pathname = new URL(route.request().url()).pathname;
    if (pathname.startsWith('/api/')) return route.fulfill({json:pathname === '/api/filters' ? {fields:[],roles:[],companies:[],topics:[],years:[]} : {items:[],total:0,pageSize:12}});
    const name = pathname === '/' ? 'index.html' : pathname.slice(1);
    return route.fulfill({body:fs.readFileSync(path.join(__dirname,'../web',name)),contentType:name.endsWith('.js')?'text/javascript':name.endsWith('.css')?'text/css':'text/html'});
   });
   await page.goto('http://bulk.test/'); await page.locator('#share').click();
   await page.locator('#bulk-entry summary').click();
   await page.locator('#bulk-topic').fill('Java');
   const list = '[6:00 pm, 27/8/2026] +91 77690 64333: Hello sharing questions👇\n[6:00 pm, 27/8/2026] +91 77690 64333: Deloitte Internal Project interview Questions -\n\n' + Array.from({length:30},(_,i)=>`${i+1}.${i<9?' ':''}Question ${i+1}?${i===4?'\nContinuation of question five.':''}`).join('\n');
   await page.locator('#bulk-questions').fill(list); await page.locator('#import-questions').click();
   assert.equal(await page.locator('[data-question]').count(),30);
   assert.equal(await page.locator('[data-question]').nth(4).inputValue(),'Question 5?\nContinuation of question five.');
   assert.equal(await page.locator('[data-question]').nth(29).inputValue(),'Question 30?');
   assert.equal(await page.locator('[data-topic]').nth(29).inputValue(),'Java');
   await page.locator('#bulk-questions').fill(Array.from({length:21},(_,i)=>`${i+1}) Extra?`).join('\n'));
   await page.locator('#import-questions').click(); assert.match(await page.locator('#bulk-status').textContent(),/up to 50/);
   assert.equal(await page.locator('[data-question]').count(),30);
   await page.locator('#bulk-questions').fill('1. '+ 'x'.repeat(2001)); await page.locator('#import-questions').click();
   assert.match(await page.locator('#bulk-status').textContent(),/2,000/);
   await page.locator('#bulk-questions').fill('• Plain question?\n- Another question?'); await page.locator('#import-questions').click();
   assert.equal(await page.locator('[data-question]').count(),32);
   assert.equal(await page.locator('[data-question]').nth(30).inputValue(),'Plain question?');
   assert.ok(await page.evaluate(()=>document.querySelector('#submission').scrollWidth <= document.querySelector('#submission').clientWidth+1));
   assert.deepEqual(errors,[]); await page.close();
  }
  console.log('Bulk entry passed: WhatsApp headers, 30 numbered questions, continuation lines, topics, append, limits, bullets, mobile and desktop.');
 } finally {await browser.close();}
})().catch(error => {console.error(error);process.exit(1);});
