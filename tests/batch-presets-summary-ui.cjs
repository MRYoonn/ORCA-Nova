// Regression for per-XYZ Batch presets and readable, scientifically cautious OUT summaries.
const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');
const xyz='3\nwater\nO 0 0 0\nH 0.757 0.586 0\nH -0.757 0.586 0\n';
(async()=>{
 const browser=await chromium.launch({headless:true,...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{}),args:['--no-sandbox']});
 try{
  const page=await browser.newPage(),errors=[];
  page.on('pageerror',e=>errors.push(e.message));
  await page.route('http://orca.test/**',route=>{
   const u=new URL(route.request().url());
   if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'../assets/desktop.html'),'utf8')});
   if(u.pathname==='/api/queue')return route.fulfill({json:{jobs:[],paused:false}});
   if(u.pathname==='/api/status')return route.fulfill({json:{config:{orcaExe:'/tmp/orca',workRoot:'/tmp/jobs'}}});
   return route.fulfill({json:{ok:true}});
  });
  await page.goto('http://orca.test/?session=x');
  await page.evaluate(()=>{active='A4';$('cores').value='2';$('maxcore').value='500';});
  await page.locator('#batch-files').setInputFiles(['monomer','dimer','control'].map(name=>({name:name+'.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)})));
  await page.waitForFunction(()=>batchMolecules.length===3);
  assert.deepEqual(await page.evaluate(()=>batchMolecules.map(x=>x.presetId)),['A4','A4','A4']);
  await page.locator('#batch-molecules tr').nth(0).locator('.batch-preset-select').selectOption('W2');
  assert.deepEqual(await page.evaluate(()=>batchMolecules.map(x=>x.presetId)),['W2','A4','A4']);
  assert.deepEqual(await page.evaluate(()=>batchMolecules.map(x=>x.request.stages.length)),[2,1,1]);
  assert.equal(await page.evaluate(()=>active),'A4');
  await page.locator('#batch-molecules tr').nth(1).locator('.batch-preset-select').selectOption('A2');
  assert.deepEqual(await page.evaluate(()=>batchMolecules.map(x=>x.presetId)),['W2','A2','A4']);
  const check=await page.evaluate(()=>batchPayload().jobs.map(x=>({label:x.label,stages:x.stages.map(z=>z.name),input:x.files.filter(f=>f.name.endsWith('.inp')).map(f=>f.content)})));
  assert(check[0].label.endsWith('_W2'));
  assert(check[1].label.endsWith('_A2'));
  assert(check[2].label.endsWith('_A4'));
  assert(check[0].input.some(x=>x.includes('* xyzfile 0 3')),'triplet stage must preserve multiplicity');
  const editor=page.locator('#batch-molecules tr').nth(0).locator('.batch-inputs textarea').first();
  await editor.fill('! PBE0 def2-SVP\n%maxcore 500\n* xyzfile 0 1 monomer.xyz\n');
  assert(await page.evaluate(()=>batchMolecules[0].custom));
  page.once('dialog',d=>d.dismiss());
  await page.locator('#batch-molecules tr').nth(0).locator('.batch-preset-select').selectOption('B4');
  assert.equal(await page.evaluate(()=>batchMolecules[0].presetId),'W2');
  assert(await page.evaluate(()=>batchMolecules[0].custom));
  assert.equal(await page.locator('#batch-molecules tr').nth(0).locator('.batch-preset-select').inputValue(),'W2');
  page.once('dialog',d=>d.accept());
  await page.locator('#batch-molecules tr').nth(0).locator('.batch-preset-select').selectOption('B4');
  assert.equal(await page.evaluate(()=>batchMolecules[0].presetId),'B4');
  assert(!(await page.evaluate(()=>batchMolecules[0].custom)));
  assert.deepEqual(await page.evaluate(()=>batchMolecules.map(x=>x.presetId)),['B4','A2','A4']);
  // Imported INP must remain untouched unless the user explicitly chooses a new preset.
  const imported='! PBE0 D4 def2-SVP\n%maxcore 500\n* xyzfile 0 1 dimer.xyz\n';
  await page.locator('#batch-files').setInputFiles([{name:'imported.inp',mimeType:'text/plain',buffer:Buffer.from(imported)}]);
  await page.waitForFunction(()=>batchMolecules[1].preset==='Imported INP');
  assert.equal(await page.locator('#batch-molecules tr').nth(1).locator('.batch-preset-select').inputValue(),'__imported__');
  assert((await page.evaluate(()=>batchMolecules[1].request.files.find(f=>f.name==='imported.inp').content))===imported);
  page.once('dialog',d=>d.accept());
  await page.locator('#batch-molecules tr').nth(1).locator('.batch-preset-select').selectOption('A1');
  assert.equal(await page.evaluate(()=>batchMolecules[1].presetId),'A1');
  assert(!(await page.evaluate(()=>batchMolecules[1].custom)));
  assert.equal(await page.evaluate(()=>batchMolecules[2].presetId),'A4');
  // Readable result cards: exact OUT values, no guessing or scientific state conflation.
  const data={sources:[{name:'triplet.out',analysis:{
   Status:'Completed',Normal:true,SCFConverged:true,Optimized:true,Method:'PBE0 D4 def2-SVP Opt CPCM(acetonitrile)',
   Geometry:'triplet.xyz',Electronic:-123.45678912,Gibbs:-123.34567891,Enthalpy:-123.399991,ZPE:0.0511,Multiplicity:3,Charge:0,S2:2.01234,ImaginaryCount:0,
   Roots:[{Kind:'singlet',Root:1,EV:3.20,WavelengthNM:387.45,OscillatorStrength:0.12}],
   Populations:{Mulliken:[{Index:5,Element:'C',Spin:-0.4},{Index:2,Element:'C',Spin:0.8}]},
   Warnings:['Check <img src=x onerror=alert(1)>']
  },pes:[],warnings:[]}]};
  await page.evaluate(x=>vizSetModel(x),data);
  const summary=page.locator('#viz-summary');
  assert((await summary.innerText()).includes('계산 정상 종료'));
  assert((await summary.innerText()).includes('-123.45678912 Eh'));
  assert((await summary.innerText()).includes('Gibbs Free Energy'));
  assert((await summary.innerText()).includes('2.01234'));
  assert((await summary.innerText()).includes('TD-DFT Vertical'));
  assert((await summary.innerText()).includes('0-based'));
  assert.equal(await summary.locator('img').count(),0);
  assert.equal(await summary.locator('.viz-report-energy').count(),4);
  assert.equal(await summary.locator('details').count(),2);
  assert.equal(await summary.locator('pre').first().isVisible(),false); // advanced JSON hidden
  const spinRows=await summary.locator('section').last().locator('tr').allInnerTexts();
  assert(spinRows.length>=3);
  assert(spinRows[1].includes('2'),'absolute-spin sorted atom index 2 first');
  await page.evaluate(()=>vizSetModel({sources:[{name:'missing.out',analysis:{Status:'Failed',Normal:false,Electronic:null,Gibbs:null,Roots:[],Populations:{}},warnings:[]}]}));
  const missing=await summary.innerText();
  assert(missing.includes('계산 실패 / 미완료'));
  assert(missing.includes('Not calculated'));
  assert(!missing.includes('-123.45678912'));
  assert.deepEqual(errors,[]);
  console.log('PASS Batch per-molecule preset isolation/imported INP guard and readable scientific results with safe unavailable states');
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
