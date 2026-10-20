const assert=require('node:assert/strict');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
(async()=>{
 const browser=await chromium.launch({headless:true});
 try {for(const width of [320,1280]){
  const page=await browser.newPage({viewport:{width,height:850}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto(process.env.TEST_BASE_URL||'https://interviewtracker.online');
  await page.locator('#result-count').filter({hasText:/interview report/}).waitFor();
  for(const [term,expected] of [['hashmp',/hashmap/i],['syncronisation',/synchronisation/i],['sprng securty',/spring security/i],['flatmp',/flatMap/i]]){
   const loaded=page.waitForResponse(r=>r.url().includes('/api/interviews?')&&new URL(r.url()).searchParams.get('search')===term);
   await page.locator('#filters [name="search"]').fill(term);const response=await loaded;assert.equal(response.status(),200);
   const body=await response.json();assert.ok(body.total>0,term+' found no results');
   await page.locator('#result-count').filter({hasText:/interview report/}).waitFor();
   assert.match(await page.locator('#reports').textContent(),expected);
  }
  await page.locator('#filters [name="topic"]').selectOption('Kafka');
  await page.getByText('No matching interviews yet.',{exact:true}).waitFor();
  await page.getByRole('button',{name:'Clear filters',exact:true}).click();
  await page.locator('#result-count').filter({hasText:/interview report/}).waitFor();
  assert.equal(await page.locator('#filters [name="search"]').inputValue(),'');
  assert.deepEqual(errors,[]);await page.close();
 }
 console.log('Live fuzzy search passed on mobile and desktop: HashMap, synchronisation, Spring Security, flatMap typos, combined topics and reset.');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
