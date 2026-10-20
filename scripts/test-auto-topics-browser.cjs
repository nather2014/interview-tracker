const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
(async()=>{
 const browser=await chromium.launch({headless:true});
 try {for(const width of [320,1280]){
  const page=await browser.newPage({viewport:{width,height:850}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
  let unavailable=false,delay=false,posted=null;
  await page.route('http://auto.test/**',async route=>{
   const req=route.request(),u=new URL(req.url());
   if(u.pathname==='/api/topics/suggest'){
    if(unavailable)return route.fulfill({status:503,json:{message:'Automatic topics are unavailable. Enter topics manually.'}});
    if(delay)await new Promise(resolve=>setTimeout(resolve,200));
    return route.fulfill({json:{suggestions:req.postDataJSON().questions.map(q=>({topic:q.includes('Kafka')?'Kafka':q.includes('segregation')?'Java OOP':'SQL',needsReview:q.includes('segregation'),method:'rules'}))}});
   }
   if(u.pathname==='/api/filters')return route.fulfill({json:{fields:[],roles:[],companies:[],topics:[],years:[]}});
   if(u.pathname==='/api/visits')return route.fulfill({status:204});
   if(u.pathname==='/api/interviews'){
    if(req.method()==='POST'){posted=req.postDataJSON();return route.fulfill({status:201,json:{id:1}});}
    return route.fulfill({json:{items:[],total:0,pageSize:12}});
   }
   const name=u.pathname==='/'?'index.html':u.pathname.slice(1);
   return route.fulfill({body:fs.readFileSync(path.join(__dirname,'../web',name)),contentType:name.endsWith('.js')?'text/javascript':name.endsWith('.css')?'text/css':'text/html'});
  });
  await page.goto('http://auto.test/');await page.locator('#share').click();await page.locator('#bulk-entry summary').click();
  await page.locator('#bulk-questions').fill('1. Have you used Kafka?\n2. Write a salary query.\n3. What is segregation in Java?');
  await page.locator('#import-questions').click();await page.locator('#topic-status').filter({hasText:'3 topics assigned'}).waitFor();
  assert.deepEqual(await page.locator('[data-topic]').evaluateAll(inputs=>inputs.map(input=>input.value)),['Kafka','SQL','Java OOP']);
  assert.match(await page.locator('.question-field').nth(2).locator('.topic-hint').textContent(),/please review/);
  await page.locator('[data-topic]').first().fill('Messaging');
  await page.locator('[data-question]').first().fill('Kafka consumer groups?');await page.locator('#suggest-topics').click();assert.equal(await page.locator('[data-topic]').first().inputValue(),'Messaging');
  await page.locator('[data-question]').nth(1).fill('Kafka partitions?');await page.locator('#topic-status').filter({hasText:'1 topic assigned'}).waitFor();
  assert.equal(await page.locator('[data-topic]').nth(1).inputValue(),'Kafka');
  await page.locator('#add-question').click();unavailable=true;await page.locator('[data-question]').last().fill('What is another query?');
  await page.locator('#topic-status').filter({hasText:'Enter topics manually'}).waitFor();assert.equal(await page.locator('[data-topic]').last().inputValue(),'');
  await page.locator('[data-topic]').last().fill('Manual fallback');unavailable=false;delay=true;
  await page.locator('#add-question').click();await page.locator('[data-question]').last().fill('Write a query?');
  const request=page.waitForRequest(r=>r.url().endsWith('/api/topics/suggest'));await page.locator('#suggest-topics').click();await request;
  await page.locator('[data-topic]').last().fill('My override');await page.waitForTimeout(300);
  assert.equal(await page.locator('[data-topic]').last().inputValue(),'My override');
  const form=page.locator('#submission-form');await form.locator('[name="field"]').selectOption('Technology');await form.locator('[name="company"]').fill('Auto test');await form.locator('[name="role"]').fill('Developer');await form.locator('[name="experience"]').selectOption('3–5');await form.locator('[name="interviewDate"]').fill('2026-01-15');
  await page.locator('#publish').click();await page.locator('#notice').filter({hasText:'Interview saved.'}).waitFor();
  assert.equal(posted.questions[0].topicSource,'manual');assert.equal(posted.questions[1].topicSource,'auto');assert.equal(posted.questions[2].topicSource,'auto');assert.equal(posted.questions[3].topicSource,'manual');assert.equal(posted.questions[4].topic,'My override');
  await page.locator('#share').click();
  await form.locator('[name="company"]').fill('Company only');
  assert.equal(await form.evaluate(el=>el.checkValidity()),true);
  await page.locator('#publish').click();await page.locator('#submission').waitFor({state:'hidden'});
  assert.equal(posted.company,'Company only');assert.equal(posted.interviewDate,'');assert.equal(posted.interviewType,'');assert.deepEqual(posted.questions,[]);
  await page.locator('#share').click();await page.locator('#auto-topics').uncheck();
  await form.locator('[name="company"]').fill('Question without topic');
  await page.locator('[data-question]').fill('Explain your development process.');
  await page.locator('#publish').click();await page.locator('#submission').waitFor({state:'hidden'});
  assert.equal(posted.questions.length,1);assert.equal(posted.questions[0].topic,'');
  assert.deepEqual(errors,[]);await page.close();
 }
 console.log('Automatic topics passed: mixed bulk lists, typed questions, manual overrides, review flags, service failure, in-flight edits and saved topic sources on mobile/desktop.');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
