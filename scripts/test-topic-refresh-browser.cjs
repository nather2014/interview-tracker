const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
(async()=>{
 const browser=await chromium.launch({headless:true});
 try {for(const width of [320,1280]){
  const page=await browser.newPage({viewport:{width,height:850}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
  let topic='Original topic',queries=[];
  await page.route('http://topics.test/**',route=>{
   const u=new URL(route.request().url());
   if(u.pathname==='/api/filters')return route.fulfill({json:{fields:['Technology'],roles:['Developer'],companies:['Example'],topics:[topic],years:[2026]}});
   if(u.pathname==='/api/visits')return route.fulfill({status:204});
   if(u.pathname==='/api/interviews'){
    queries.push(u.searchParams.get('topic'));
    const matches=!u.searchParams.get('topic')||u.searchParams.get('topic')===topic;
    return route.fulfill({json:{total:matches?1:0,pageSize:12,items:matches?[{id:1,field:'Technology',company:'Example',role:'Developer',experience:'3–5',interviewType:'Online',interviewDate:'2026-01-15',questions:[{question:'Example question?',topic}]}]:[]}});
   }
   const name=u.pathname==='/'?'index.html':u.pathname.slice(1);
   return route.fulfill({body:fs.readFileSync(path.join(__dirname,'../web',name)),contentType:name.endsWith('.js')?'text/javascript':name.endsWith('.css')?'text/css':'text/html'});
  });
  await page.goto('http://topics.test/');await page.locator('#filters [name="topic"] option').filter({hasText:'Original topic'}).waitFor({state:'attached'});
  await page.locator('#filters [name="topic"]').selectOption('Original topic');await page.waitForFunction(()=>document.querySelector('#result-count').textContent==='1 interview report');
  topic='Edited topic';await page.evaluate(()=>window.dispatchEvent(new Event('focus')));
  await page.locator('#filters [name="topic"] option').filter({hasText:'Edited topic'}).waitFor({state:'attached'});
  assert.equal(await page.locator('#filters [name="topic"] option[value="Original topic"]').count(),0);
  assert.equal(await page.locator('#filters [name="topic"]').inputValue(),'');
  await page.locator('.question-topic').filter({hasText:'Edited topic'}).waitFor();
  await page.locator('#filters [name="topic"]').selectOption('Edited topic');
  await page.waitForResponse(r=>r.url().includes('/api/interviews?')&&new URL(r.url()).searchParams.get('topic')==='Edited topic');
  topic='Newest topic';await page.locator('#filters [name="search"]').fill('Example');
  await page.locator('#filters [name="topic"] option').filter({hasText:'Newest topic'}).waitFor({state:'attached'});
  await page.locator('.question-topic').filter({hasText:'Newest topic'}).waitFor();assert.equal(queries.at(-1),null);
  // Updating filter options must never overwrite submission metadata.
  await page.locator('#share').click();assert.equal(await page.locator('#submission [name="field"] option').count(),11);await page.locator('#close').click();
  topic='Cross-tab topic';await page.evaluate(()=>window.dispatchEvent(new StorageEvent('storage',{key:'interview-reports-updated',newValue:'changed'})));
  await page.locator('#filters [name="topic"] option').filter({hasText:'Cross-tab topic'}).waitFor({state:'attached'});
  await page.locator('.question-topic').filter({hasText:'Cross-tab topic'}).waitFor();assert.deepEqual(errors,[]);await page.close();
 }
 console.log('Topic refresh passed: return to page, new/removed topics, stale selections, filtering, cross-tab updates and form isolation on mobile/desktop.');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
