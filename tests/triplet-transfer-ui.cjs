// Triplet Geometry Transfer UI regression: review/approve before Batch,
 // multi-target isolation, role mapping, spin comparison and existing W2 intact.
const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');
const xyz='8\nsynthetic UI-only XYZ\nC 0 0 0\nN 1.3 0 0\nC 1.9 1 0\nC 1.2 2.2 0\nC -0.1 2.2 0\nN -0.7 1.1 0\nS -1.7 0 0\nO 3 1.1 0\n';
const roles={C2:0,N3:1,C4:2,C5:3,C6:4,N1:5,S2:6,O4:7};
(async()=>{
 const browser=await chromium.launch({headless:true,args:['--no-sandbox'],...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{})});
 try{
  const page=await browser.newPage({viewport:{width:1440,height:940},acceptDownloads:true});
  const errors=[],prepared=[],queued=[],dialogs=[];
  let jobs=[];
  page.on('pageerror',e=>errors.push(e.message));
  page.on('dialog',d=>{dialogs.push(d.message());d.accept()});
  await page.route('http://orca.test/**',route=>{
   const r=route.request(),u=new URL(r.url()),url=u.pathname;
   if(url==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'../assets/desktop.html'),'utf8')});
   if(url==='/api/status')return route.fulfill({json:{config:{orcaExe:'C:\\ORCA\\orca.exe',workRoot:'C:\\jobs'},update:{channel:'stable',current:'1.6.1'}}});
   if(url==='/api/update/check')return route.fulfill({json:{channel:'stable',current:'1.6.1',available:false}});
   if(url==='/api/config')return route.fulfill({json:{ok:true}});
   if(url==='/api/triplet/prepare'){
    const payload=r.postDataJSON();prepared.push(payload);
    const targets=payload.targets.map(x=>({
     name:x.name,referenceRoles:payload.referenceRoles||roles,targetRoles:x.roles||roles,
     transferredXYZ:x.xyz,warnings:[],
     request:payload.confirmed?{label:'TGT_'+x.name,triplet:{confirmed:true,roles:roles,targetName:x.name},files:[{name:'transferred_initial.xyz',content:x.xyz}],stages:[{input:'step1_constrained.inp',source:'transferred_initial.xyz',opt:true},{input:'step2_triplet.inp',source:'step1_constrained.xyz',opt:true},{input:'step3_spin.inp',source:'step2_triplet.xyz',opt:false}]}:null
    }));
    return route.fulfill({json:{items:targets,warnings:['mock'],needsConfirmation:!payload.confirmed}})
   }
   if(url==='/api/queue/preflight'){const data=r.postDataJSON();queued.push({type:'preflight',payload:data});return route.fulfill({json:{ok:true,reports:data.jobs.map(()=>({ok:true}))}})}
   if(url==='/api/queue'&&r.method()==='POST'){const data=r.postDataJSON();queued.push({type:'run',payload:data});jobs=data.jobs.map((job,i)=>({id:'T'+i,label:job.label,request:job,state:'Completed',job:{stageName:'Spin Density'}}));return route.fulfill({json:{ok:true}})}
   if(url==='/api/queue')return route.fulfill({json:{jobs,limits:{concurrency:2,cpu:16,ramMB:56000},paused:false}});
   if(url==='/api/triplet/result'){
    return route.fulfill({json:{
     target:'targetA.xyz',method:'UKS B3LYP D4 def2-TZVP TightSCF SP SMD(acetonitrile)',charge:1,multiplicity:3,
     converged:true,scfConverged:true,optimizationConverged:true,minimumVerified:false,frequencyChecked:false,
     energyEh:-530.105,s2:2.019,roles,spin:[
      {role:'C2',index:0,element:'C',mulliken:0.15,loewdin:0.13,hirshfeld:0.17},
      {role:'C5',index:3,element:'C',mulliken:0.84,loewdin:0.81,hirshfeld:0.79},
      {role:'C6',index:4,element:'C',mulliken:0.32,loewdin:0.34,hirshfeld:0.31}
     ],bonds:{'C2-N1':1.35,'N1-C6':1.40,'C5-C6':1.42,'C2-S2':1.78},
     cubeAvailable:true,gbwAvailable:true,finalXYZ:xyz,warnings:['Frequency 미계산: 최소점 미검증'],somoCandidates:[{orbitalIndex:41,alphaOccupation:1,betaOccupation:0,alphaEnergyEh:-0.25}]
    }});
   }
   if(url==='/api/triplet/compare'){
    return route.fulfill({json:{sameTarget:true,deltaEnergyKcal:-2.5,excitationKcal:null,warnings:['최저 T1 상태는 별도 확인'],
      original:{spin:[{role:'C2',mulliken:0.1},{role:'C5',mulliken:0.6},{role:'C6',mulliken:0.3}],energyEh:-530.101,s2:2.02,bonds:{'C2-N1':1.33,'C5-C6':1.41}},
      transfer:{spin:[{role:'C2',mulliken:0.15},{role:'C5',mulliken:0.84},{role:'C6',mulliken:0.32}],energyEh:-530.105,s2:2.019,bonds:{'C2-N1':1.35,'C5-C6':1.42}}
    }});
   }
   return route.fulfill({json:{ok:true}});
  });
  await page.goto('http://orca.test/?session=x');
  assert(!(await page.locator('#triplet-panel').isVisible()));
  const w2Before=await page.evaluate(()=>calc('W2').name),w5Before=await page.evaluate(()=>calc('W5').name);
  await page.locator('[data-preset="triplet-transfer"]').click();
  assert(await page.locator('#triplet-panel').isVisible());
  assert.equal(await page.locator('#trip-fixed input:checked').count(),6);
  assert.equal(await page.locator('#trip-frequency').isChecked(),false);
  await page.locator('#trip-ref').setInputFiles({name:'NH_S-Me_T1.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)});
  await page.locator('#trip-target').setInputFiles(['targetA.xyz','targetB.xyz'].map(name=>({name,mimeType:'text/plain',buffer:Buffer.from(xyz)})));
  await page.locator('#trip-preview').click();
  await page.waitForFunction(()=>tripState.draft?.items?.length===2);
  assert.equal(prepared.length,1);
  assert.equal(prepared[0].confirmed,false);
  assert.equal(await page.locator('.triplet-mapping-card').count(),2);
  assert.equal(await page.locator('.triplet-mapping-card .trip-ref-role').count(),24);
  assert.equal(await page.locator('#trip-run').isDisabled(),false);
  await page.locator('.trip-target-role[data-index="0"][data-role="C5"]').fill('5');
  assert(await page.locator('#trip-run').isDisabled(),'editing roles invalidates approval');
  await page.locator('#trip-review').click();
  await page.waitForFunction(()=>tripState.draft?.items?.length===2);
  assert.equal(prepared.at(-1).targets[0].roles.C5,5,'user-edited target chemical mapping must be posted');
  assert.equal(prepared.at(-1).confirmed,false);
  await page.locator('#trip-budget-cpu').fill('16');
  await page.locator('#trip-budget-ram').fill('56000');
  await page.locator('#trip-concurrency').fill('2');
  await page.locator('#trip-review').click();
  await page.waitForFunction(()=>tripState.draft?.items?.length===2);
  await page.locator('#trip-run').click();
  await page.waitForFunction(()=>queued.some(x=>x.type==='run'));
  assert(dialogs.some(x=>x.includes('C2/C5/C6')),'user confirmation was skipped');
  const submitted=queued.find(x=>x.type==='run').payload;
  assert.equal(submitted.jobs.length,2);
  assert.equal(submitted.limits.concurrency,2);
  assert.equal(submitted.limits.ramMB,56000);
  assert(submitted.jobs.every(x=>x.triplet?.confirmed&&x.stages.length===3));
  await page.locator('#trip-results-refresh').click();
  assert.equal(await page.locator('#trip-result-job option').count(),2);
  await page.locator('#trip-result-load').click();
  await page.waitForFunction(()=>tripState.lastResult?.spin?.length===3);
  const text=await page.locator('#trip-result-view').innerText();
  assert(text.includes('C5')&&text.includes('0.84000'));
  assert(text.includes('Frequency 미계산'));
  await page.locator('.triplet-original summary').click();
  for(const id of ['trip-original-out','trip-original-inp','trip-original-xyz']){
   await page.locator('#'+id).setInputFiles({name:id+'.txt',mimeType:'text/plain',buffer:Buffer.from(id==='trip-original-xyz'?xyz:'synthetic input')});
  }
  await page.locator('#trip-compare').click();
  await page.waitForFunction(()=>tripState.lastComparison?.sameTarget===true);
  assert((await page.locator('#trip-comparison').innerText()).includes('-2.5000'));
  assert(await page.locator('#trip-export').isEnabled());
  assert.equal(await page.evaluate(()=>calc('W2').name),w2Before);
  assert.equal(await page.evaluate(()=>calc('W5').name),w5Before);
  assert.deepEqual(errors,[]);
  console.log('PASS: Triplet preset visibility, mapped XYZ approval, two-target Queue, controlled budgets, spin report, W2/W5 unchanged');
 }finally{await browser.close()}
})().catch(err=>{console.error(err);process.exit(1)});
