const assert = require('node:assert/strict'), fs = require('node:fs'), path = require('node:path');
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
(async () => {
 const browser = await chromium.launch({headless:true});
 try {
  for (const width of [320,1280]) {
   const page = await browser.newPage({viewport:{width,height:850}}), errors=[];
   page.on('pageerror', e=>errors.push(e.message));
   let report={id:42,company:'Original company',role:'Developer',field:'',experience:'3–5',interviewType:'Online',interviewDate:'2026-01-15',location:'Remote',questions:[{question:'Original question?',topic:'Java'},{question:'Remove this question?',topic:'SQL'}]}, fail=true, writes=0;
   await page.route('http://edit.test/**', route=>{
    const req=route.request(), url=new URL(req.url());
    if(url.pathname.startsWith('/api/admin/')) {
     assert.equal(req.headers().authorization,'Bearer test-token');
     if(req.method()==='PUT') {
      assert.equal(url.pathname,'/api/admin/interviews/42'); writes++;
      if(fail) {fail=false;return route.fulfill({status:500,json:{message:'Temporary save failure'}});}
      report={...req.postDataJSON(),id:42}; return route.fulfill({json:{id:42}});
     }
     return route.fulfill({json:url.pathname.endsWith('session')?{authenticated:true}:{online:0,today:0}});
    }
    if(url.pathname==='/api/interviews')return route.fulfill({json:{items:[report],total:1,pageSize:12}});
    const name=url.pathname.slice(1);
    return route.fulfill({body:fs.readFileSync(path.join(__dirname,'../web',name)),contentType:name.endsWith('.js')?'text/javascript':name.endsWith('.css')?'text/css':'text/html'});
   });
   await page.goto('http://edit.test/admin.html'); await page.locator('#admin-token').fill('test-token'); await page.getByRole('button',{name:'Sign in',exact:true}).click();
   await page.getByRole('button',{name:'Edit report',exact:true}).click();
   assert.equal(await page.locator('#submission [name="company"]').inputValue(),'Original company');
   assert.equal(await page.locator('[data-question]').count(),2);
   await page.locator('#close').click(); assert.equal(writes,0);
   await page.getByRole('button',{name:'Edit report',exact:true}).click();
   await page.locator('#submission [name="company"]').fill('Edited company');
   await page.locator('[data-question]').first().fill('Edited question?');
   await page.locator('.question-field').nth(1).getByRole('button').click();
   await page.locator('#bulk-entry summary').click(); await page.locator('#bulk-topic').fill('Spring');
   await page.locator('#bulk-questions').fill('1. Added question?');await page.locator('#import-questions').click();
   await page.locator('#publish').click(); await page.locator('#form-error').filter({hasText:'Temporary save failure'}).waitFor();
   assert.equal(await page.locator('[data-question]').first().inputValue(),'Edited question?');
   await page.locator('#publish').click(); await page.locator('#notice').filter({hasText:'Report updated.'}).waitFor();
   assert.equal(report.id,42);assert.equal(report.company,'Edited company'); assert.equal(report.field,'');assert.equal(report.questions.length,2);
   assert.equal(report.questions[1].question,'Added question?'); assert.equal(report.questions[1].topic,'Spring');
   await page.getByRole('button',{name:'Edit report',exact:true}).click();
   assert.equal(await page.locator('[data-question]').first().inputValue(),'Edited question?');
   assert.ok(await page.evaluate(()=>document.querySelector('#submission').scrollWidth<=document.querySelector('#submission').clientWidth+1));
   assert.deepEqual(errors,[]);await page.close();
  }
  console.log('Edit browser checks passed: prefill, cancel, metadata, question changes/removal/bulk addition, failed-save retry, saved identity, legacy field, phone and desktop.');
 } finally {await browser.close();}
})().catch(e=>{console.error(e);process.exit(1);});
