const assert=require('node:assert/strict'),fs=require('node:fs');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const base=process.env.TEST_BASE_URL||'http://localhost:8091';
if(!process.env.TEST_REPORT_IDS_FILE)throw Error('Set TEST_REPORT_IDS_FILE to record temporary reports for cleanup.');
(async()=>{const browser=await chromium.launch({headless:true});try{
 for(const width of [320,1280]){
  const context=await browser.newContext({viewport:{width,height:850}}),page=await context.newPage(),errors=[];
  page.on('pageerror',e=>errors.push(e.message));
  await page.goto(base);await page.locator('#result-count').filter({hasText:/interview report/}).waitFor();
  assert.equal(await page.locator('#filters [name="field"]').count(),1);assert.equal(await page.locator('#filters [name="role"]').count(),1);
  await page.locator('#share').click();const form=page.locator('#submission-form');
  const marker='Browser verification '+width+' '+Date.now();
  await form.locator('[name="field"]').selectOption('Finance');await form.locator('[name="company"]').fill(marker);
  await form.locator('[name="role"]').fill('Accountant '+marker);await form.locator('[name="experience"]').selectOption('3–5');
  await form.locator('[name="interviewDate"]').fill('2026-01-15');await form.locator('[name="location"]').fill('Remote');
  await form.locator('[data-question]').fill('How do you prepare a balance sheet?');await form.locator('[data-topic]').fill('Accounting');
  const response=page.waitForResponse(r=>r.url().endsWith('/api/interviews')&&r.request().method()==='POST');
  await form.locator('#publish').click();const posted=await response;assert.equal(posted.status(),201);const data=await posted.json();
  fs.appendFileSync(process.env.TEST_REPORT_IDS_FILE,JSON.stringify({id:data.id,company:marker})+'\n');
  await page.locator('#notice').filter({hasText:'now published'}).waitFor();
  await page.locator('#filters [name="role"] option').filter({hasText:'Accountant '+marker}).waitFor({state:'attached'});
  await page.locator('#filters [name="field"]').selectOption('Finance');await page.locator('#filters [name="role"]').selectOption('Accountant '+marker);
  await page.locator('#result-count').filter({hasText:/^1 interview report$/}).waitFor();
  assert.match(await page.locator('.report').textContent(),/Finance/);assert.match(await page.locator('.report').textContent(),/balance sheet/);
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'page overflows');
  await page.locator('#filters [name="search"]').fill('no-such-question-'+marker);await page.getByText('No matching interviews yet.',{exact:true}).waitFor();
  await page.getByRole('button',{name:'Clear filters',exact:true}).click();await page.locator('#result-count').filter({hasText:/interview report/}).waitFor();
  assert.equal(await page.locator('#filters [name="field"]').inputValue(),'');assert.equal(await page.locator('#filters [name="role"]').inputValue(),'');
  assert.deepEqual(errors,[]);await context.close();
 }
 console.log('General interview browser checks passed: Finance posting, role/field filters, search, reset and phone/desktop layouts.');
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
