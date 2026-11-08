const {chromium}=require('playwright');
const fs=require('fs'),assert=require('assert');
(async()=>{const browser=await chromium.launch({headless:true,...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{}),args:['--no-sandbox']});try{
 const page=await browser.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 let submitted=null,compareMode=false;await page.route('http://orca.test/**',async route=>{
 const u=new URL(route.request().url());if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(require('path').join(__dirname,'../assets/desktop.html'),'utf8')});
 if(u.pathname==='/api/queue'&&route.request().method()==='POST'){submitted=route.request().postDataJSON();return route.fulfill({json:{ok:true}})}
 if(u.pathname==='/api/queue')return route.fulfill({json:{jobs:compareMode?[{id:'j1',label:'Test molecule',state:'Completed',request:{stages:[{name:'S0 Opt',input:'first.inp',source:'mol1.xyz',opt:true}]}}]:[],paused:false,batchEta:'약 2분 ~ 약 5분'}});
 if(u.pathname==='/api/queue/results')return route.fulfill({json:{Stages:[{Stage:{name:'S0 Opt',source:'mol1.xyz'},Analysis:{Status:'Completed',Method:'PBE0 def2-SVP',Geometry:'mol1.xyz',Electronic:-100.123,Roots:[{Kind:'S',Root:1,EV:3.14},{Kind:'T',Root:1,EV:2.71}],Populations:{Mulliken:[{Index:4,Element:'C',Spin:0.125},{Index:5,Element:'N',Spin:-0.25}]}}},{Stage:{name:'Unrun stage',source:'first.xyz'},Analysis:{Status:'Not calculated'}}]}});
 if(u.pathname==='/api/queue/preflight')return route.fulfill({json:{ok:true,reports:[]}});
 if(u.pathname==='/api/status')return route.fulfill({json:{config:{orcaExe:'/tmp/orca',workRoot:'/tmp/jobs'}}});
 return route.fulfill({json:{ok:true}});
 });await page.goto('http://orca.test/?session=mock');
 await page.evaluate(()=>{active='W2';$('cores').value='2';$('maxcore').value='100';});
 const xyz='3\nwater\nO 0 0 0\nH 0.757 0.586 0\nH -0.757 0.586 0\n';
 await page.locator('#batch-files').setInputFiles([1,2,3,4].map(n=>({name:'mol'+n+'.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)})));
 await page.waitForFunction(()=>batchMolecules.length===4);
 assert.equal(await page.locator('#batch-molecules tr').count(),4);
 const payload=await page.evaluate(()=>batchPayload());assert.equal(payload.jobs.length,4);for(const [i,r]of payload.jobs.entries()){assert.equal(r.stages.length,2);assert(r.files.some(f=>f.name==='mol'+(i+1)+'.xyz'));assert.equal(r.stages[1].source,r.stages[0].input.replace('.inp','.xyz'));}
 assert.equal(await page.evaluate(()=>loadedXYZ),'');
 await page.locator('#batch-concurrency').fill('2');await page.locator('#batch-start').click();await page.waitForFunction(()=>document.querySelector('#batch-status').textContent.includes('등록했습니다'));assert.equal(submitted.jobs.length,4);assert.equal(submitted.limits.concurrency,2);
 // Exclusion/deletion and custom INP editing survive rendering/queue submission.
 await page.locator('#batch-molecules tr').first().locator('input[type=checkbox]').uncheck();assert.equal(await page.evaluate(()=>batchPayload().jobs.length),3);
 await page.locator('#batch-molecules tr').last().getByText('삭제',{exact:true}).click();assert.equal(await page.evaluate(()=>batchMolecules.length),3);
 const custom='! PBE0 def2-SVP\n%maxcore 100\n* xyzfile 0 1 mol1.xyz\n';
 await page.locator('#batch-files').setInputFiles([{name:'custom.inp',mimeType:'text/plain',buffer:Buffer.from(custom)}]);await page.waitForFunction(()=>batchMolecules[0].preset==='Imported INP');assert.equal(await page.evaluate(()=>batchMolecules[0].request.stages[0].input),'custom.inp');

 const chained=[{name:'next.inp',mimeType:'text/plain',buffer:Buffer.from('! PBE0 def2-SVP Opt\n%maxcore 100\n* xyzfile 0 3 first.xyz\n')},{name:'first.inp',mimeType:'text/plain',buffer:Buffer.from('! PBE0 def2-SVP Opt\n%maxcore 100\n* xyzfile 0 1 mol2.xyz\n')}];
 await page.locator('#batch-files').setInputFiles(chained);await page.waitForFunction(()=>batchMolecules[1].request.stages.length===2&&batchMolecules[1].request.stages[0].input==='first.inp');
 assert.equal(await page.evaluate(()=>batchMolecules[1].request.stages[1].source),'first.xyz');
 await page.locator('#batch-drop').dispatchEvent('drop',{dataTransfer:await page.evaluateHandle(xyz=>{const dt=new DataTransfer();dt.items.add(new File([xyz],'drop.xyz',{type:'text/plain'}));return dt},xyz)});await page.waitForFunction(()=>batchMolecules.some(m=>m.name==='drop.xyz'));

 compareMode=true;await page.locator('#batch-compare-refresh').click();await page.waitForFunction(()=>batchComparisonRows.length===2);
 const comparison=await page.locator('#batch-comparison-body').innerText();assert(comparison.includes('3.14'));assert(comparison.includes('2.71'));assert(comparison.includes('0.125 (Mulliken; ORCA 4 C)'));assert(comparison.includes('-0.25 (Mulliken; ORCA 5 N)'));assert(comparison.includes('Not calculated'));
 const csv=await page.evaluate(()=>batchComparisonCSV());assert(csv.includes('PBE0 def2-SVP'));assert(csv.includes('mol1.xyz'));assert(csv.includes('3.14'));assert(!comparison.includes('C5'));
 await page.locator('#batch-atom-a').fill('999');await page.locator('#batch-compare-refresh').click();await page.waitForFunction(()=>batchComparisonRows[0][8]==='Unavailable');
 await page.evaluate(()=>batchPoll());assert((await page.locator('#batch-queue-info').innerText()).includes('약 2분 ~ 약 5분'));
 assert.deepEqual(errors,[]);console.log('PASS browser: four XYZ W2 snapshots/dependency isolation, concurrency2 enqueue, exclusion/deletion, bulk INP import, no JS errors');
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
