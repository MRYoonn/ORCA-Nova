const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');
(async()=>{
 const browser=await chromium.launch({headless:true,args:['--no-sandbox'],...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{})});
 try{
  const page=await browser.newPage();
  const errors=[];let lastDoc=null,lastComp=null,calls=0;
  page.on('pageerror',e=>errors.push(e.message));
  await page.route('http://orca.test/**',route=>{
   const u=new URL(route.request().url());
   if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'../assets/desktop.html'),'utf8')});
   if(u.pathname==='/api/si/analyze'){
    calls++;lastDoc=route.request().postDataJSON();
    return route.fulfill({json:{
     name:lastDoc.name,doi:lastDoc.doi,pages:2,
     evidence:[
      {field:'functional',value:'M06-2X',page:25,excerpt:'Computational Details: M06-2X optimized geometry'},
      {field:'basis',value:'6-31+G(d,p)',page:25,excerpt:'Gaussian basis needs ORCA manual validation'},
      {field:'solvation',value:'CPCM',page:25,excerpt:'CPCM(acetonitrile) method'}
     ],
     fields:[
      {field:'functional',values:['M06-2X'],status:'candidate'},
      {field:'basis',values:['6-31+G(d,p)'],status:'unsupported-direct'},
      {field:'solvation',values:['CPCM'],status:'candidate'},
      {field:'task',values:['Opt','SP'],status:'ambiguous'}],
     differences:[{field:'functional',si:'M06-2X',inp:'PBE0',status:'review'}],
     warnings:['전체 SI 검증 미완료'],confirmed:false
    }});
   }
   if(u.pathname==='/api/si/composite'){
    lastComp=route.request().postDataJSON();
    return route.fulfill({json:{
     formula:'G_composite = E_high(SP) + [G_low(Opt+Freq) - E_low(Opt+Freq)]',
     verifiedInputs:true,lowMethod:'PBE0 def2-SVP',highMethod:'PBE0 def2-TZVPP',
     charge:0,multiplicity:3,atoms:3,lowElectronicEh:-151.55083691,
     lowGibbsEh:-151.54618270,highElectronicEh:-151.777,
     correctionEh:0.00465421,compositeGibbsEh:-151.77234579,
     warnings:['표준상태 미적용']
    }});
   }
   if(u.pathname==='/api/queue')return route.fulfill({json:{jobs:[],paused:false}});
   if(u.pathname==='/api/status')return route.fulfill({json:{config:{orcaExe:'/tmp/orca',workRoot:'/tmp/jobs'}}});
   return route.fulfill({json:{ok:true}});
  });
  await page.goto('http://orca.test/?session=x');
  await page.locator('#si-panel summary').first().click();
  assert(await page.locator('#si-text').isVisible());
  await page.locator('#si-doi').fill('10.1002/anie.9565676');
  await page.locator('#si-text').fill('Computational Details. Geometry optimization with M06-2X and 6-31+G(d,p) CPCM(acetonitrile).');
  await page.locator('#si-analyze').click();
  await page.waitForFunction(()=>!!siCurrent);
  assert.equal(lastDoc.doi,'10.1002/anie.9565676');
  assert(lastDoc.text.includes('M06-2X'));
  assert.equal(await page.locator('#si-results input.si-choice').count(),5);
  assert(await page.locator('#si-results input.si-choice[data-value="6-31+G(d,p)"]').isDisabled());
  assert(!(await page.locator('#si-results input.si-choice[data-value="M06-2X"]').isChecked()));
  assert((await page.locator('#si-results').innerText()).includes('Page 25'));
  assert((await page.locator('#si-results').innerText()).includes('review'));
  await page.locator('#si-results input.si-choice[data-value="M06-2X"]').check();
  await page.locator('#si-results input.si-choice[data-value="CPCM"]').check();
  page.once('dialog',d=>d.accept());
  await page.locator('#si-apply').click();
  assert.equal(await page.locator('#functional').inputValue(),'M06-2X');
  assert.equal(await page.locator('#solvmodel').inputValue(),'CPCM');
  // Explicit PDF mode uses base64 and never loads any remote source.
  await page.locator('#si-file').setInputFiles({name:'local_si.pdf',mimeType:'application/pdf',buffer:Buffer.from('%PDF-1.4\nlocal test')});
  await page.locator('#si-analyze').click();
  await page.waitForFunction(()=>siCurrent?.name==='local_si.pdf');
  assert.equal(lastDoc.name,'local_si.pdf');
  assert.equal(lastDoc.pdfBase64,Buffer.from('%PDF-1.4\nlocal test').toString('base64'));
  assert.equal(lastDoc.text,undefined);
  assert.equal(calls,2);
  await page.locator('#si-composite summary').click();
  const contents={
    'si-low-inp': '! PBE0 def2-SVP Opt Freq',
    'si-low-out': 'SCF CONVERGED',
    'si-low-xyz': '3\nwater\nO 0 0 0\nH 0 0 1\nH 0 1 0',
    'si-high-inp': '! PBE0 def2-TZVPP SP',
    'si-high-out': 'FINAL SINGLE POINT ENERGY -151.777',
    'si-high-xyz': '3\nwater\nO 0 0 0\nH 0 0 1\nH 0 1 0'
  };
  for(const [id,content] of Object.entries(contents))await page.locator('#'+id).setInputFiles({name:id+'.txt',mimeType:'text/plain',buffer:Buffer.from(content)});
  await page.locator('#si-stationary').selectOption('ts');
  await page.locator('#si-compute').click();
  await page.waitForFunction(()=>siCompositeReport.includes('Composite G'));
  assert.equal(lastComp.stationaryPoint,'ts');
  assert(lastComp.lowXYZ.includes('water'));
  assert((await page.locator('#si-composite-result').innerText()).includes('-151.772345790 Eh'));
  assert((await page.locator('#si-composite-result').innerText()).includes('ΔG가 아닙니다'));
  assert.deepEqual(errors,[]);
  console.log('PASS: offline SI text/PDF evidence, user opt-in, non-supported basis blocked, composite API and no JS errors');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
