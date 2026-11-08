const {chromium}=require('playwright');
const {spawn}=require('child_process'),fs=require('fs'),assert=require('assert');
(async()=>{const root=fs.mkdtempSync('/tmp/orca-e2e-');fs.mkdirSync(root+'/jobs');
const child=spawn(process.env.ORCA_NOVA_BIN||require('path').join(__dirname,'../bin/orca-nova'),[],{env:{...process.env,ORCA_NOVA_TEST_MODE:'1',XDG_CONFIG_HOME:root+'/config'}});
let browser;try{const url=await new Promise((resolve,reject)=>{child.stdout.once('data',b=>resolve(b.toString().trim().replace('TEST_URL=','')));child.once('exit',c=>reject(Error('server exit '+c)))});const token=new URL(url).searchParams.get('session'),base=new URL(url).origin;
async function api(path,body){const r=await fetch(base+'/api/'+path+'?session='+token,{method:body?'POST':'GET',headers:{'Content-Type':'application/json','X-ORCA-Session':token},body:body?JSON.stringify(body):undefined});if(!r.ok)throw Error(await r.text());return r.json()}
await api('config',{orcaExe:require('path').join(__dirname,'mock_orca.py'),workRoot:root+'/jobs'});
browser=await chromium.launch({headless:true,...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{}),args:['--no-sandbox']});const page=await browser.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto(url);
await page.evaluate(()=>{active='W2';$('cores').value='1';$('maxcore').value='100'});
const xyz='3\nwater\nO 0 0 0\nH .757 .586 0\nH -.757 .586 0\n';await page.locator('#batch-files').setInputFiles([1,2,3,4].map(n=>({name:'mol'+n+'.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)})));await page.waitForFunction(()=>batchMolecules.length===4);
await page.locator('#batch-concurrency').fill('2');await page.locator('#batch-cpu').fill('2');await page.locator('#batch-ram').fill('1024');await page.locator('#batch-start').click();await page.waitForFunction(()=>$('batch-status').textContent.includes('등록했습니다'));
let q=await api('queue');assert.equal(q.jobs.filter(j=>j.state==='Running').length,2);assert.equal(q.jobs.filter(j=>j.state==='Pending').length,2);
const deadline=Date.now()+20000;do{q=await api('queue');if(q.jobs.every(j=>['Completed','Failed'].includes(j.state)))break;await new Promise(r=>setTimeout(r,100))}while(Date.now()<deadline);
assert.deepEqual(q.jobs.map(j=>j.state),['Completed','Failed','Completed','Completed']);const old=q.jobs[1].job.folder;fs.writeFileSync(root+'/jobs/allow_retry','yes');await api('queue/action',{id:q.jobs[1].id,action:'retry'});
do{q=await api('queue');if(q.jobs[1].state==='Completed')break;await new Promise(r=>setTimeout(r,100))}while(Date.now()<deadline);assert(q.jobs.every(j=>j.state==='Completed'));assert.notEqual(old,q.jobs[1].job.folder);
for(const j of q.jobs){assert(fs.existsSync(j.job.folder+'/analysis.txt'));assert(fs.existsSync(j.job.folder+'/results.json'))}assert(fs.existsSync(root+'/jobs/batch_results.csv'));assert(fs.existsSync(root+'/jobs/batch_summary.txt'));assert.equal(new Set(q.jobs.map(j=>j.job.folder)).size,4);assert.deepEqual(errors,[]);
console.log('PASS real HTTP + Chromium: 4 XYZ W2, two concurrent, isolated folders, B failure/retry, TXT/JSON/CSV/summary, zero JS errors');console.log('Evidence directory: '+root);
}finally{if(browser)await browser.close();child.kill();}})().catch(e=>{console.error(e);process.exit(1)});
