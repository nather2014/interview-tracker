const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('fs');
const path = require('path');
const assert = require('assert');
(async () => {
 const browser = await chromium.launch({headless:true});
 for (const width of [320,1280]) {
  const page = await browser.newPage({viewport:{width,height:900}});
  const errors=[]; page.on('pageerror',error=>errors.push(error.message));
  let mode='data', lastQuery='';
  await page.route('http://insights.test/**', async route => {
   const url = new URL(route.request().url());
   if (url.pathname === '/api/filters') return route.fulfill({json:{companies:['Example'],fields:['Technology'],roles:['Developer'],years:[2026]}});
   if (url.pathname === '/api/visits') return route.fulfill({status:204});
   if (url.pathname === '/api/insights') {
    lastQuery=url.search;
    if (mode==='error') return route.fulfill({status:500});
    return route.fulfill({json: mode==='empty' ? {reports:0,questions:0,companies:0,undated:0,untagged:0,topics:[],companyBreakdown:[],months:[]} : {reports:3,questions:4,companies:1,undated:1,untagged:1,topics:[{label:'Java <script>',count:3,reports:2}],companyBreakdown:[{label:'Example',count:3}],months:[{label:'2026-01',count:1},{label:'2026-03',count:1}]}});
   }
   const file=path.join(__dirname,'../web',url.pathname);
   return route.fulfill({body:fs.readFileSync(file),contentType:file.endsWith('.js')?'text/javascript':file.endsWith('.css')?'text/css':'text/html'});
  });
  await page.goto('http://insights.test/insights.html');
  await page.waitForFunction(()=>document.querySelector('#total-reports').textContent==='3');
  assert.equal(await page.locator('#month-chart .insight-bar-row').count(),3);
  assert.equal(await page.locator('#topic-chart script').count(),0);
  assert(await page.locator('#topic-chart').innerText().then(t=>t.includes('Java <script>')));
  await page.selectOption('[name=company]','Example');
  await page.waitForFunction(()=>document.querySelector('#insight-status').textContent==='Based on 3 submitted reports.');
  assert(lastQuery.includes('company=Example'));
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  mode='empty'; await page.selectOption('[name=year]','2026'); await page.locator('#insight-empty').waitFor({state:'visible'});
  mode='error'; await page.getByRole('button',{name:'Clear filters'}).click(); await page.locator('#insight-retry').waitFor({state:'visible'});
  mode='data'; await page.locator('#insight-retry').click(); await page.locator('#insight-results').waitFor({state:'visible'});
  assert.deepEqual(errors,[]);
  await page.close();
 }
 await browser.close(); console.log('Insights browser checks passed at 320px and 1280px.');
})().catch(error=>{console.error(error);process.exit(1)});
