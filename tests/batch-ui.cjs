const {chromium}=require('playwright');
const fs=require('fs'),assert=require('assert');
(async()=>{const browser=await chromium.launch({headless:true,...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{}),args:['--no-sandbox']});try{
 const page=await browser.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 let submitted=null,compareMode=false,txtRequests=0,autoProbe=false,config={orcaExe:'/tmp/orca',workRoot:'/tmp/jobs'},singleRun=null,queueFixture=null,singleJob=null,singleETA=null,revealRequests=[],revealError=false;await page.route('http://orca.test/**',async route=>{
 const u=new URL(route.request().url());if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(require('path').join(__dirname,'../assets/desktop.html'),'utf8')});
 if(u.pathname==='/api/queue/monitor')return route.fulfill({json:{cpuPercent:37.5,ramUsedMB:24576,ramTotalMB:65536,ramAvailableMB:40960}});
 if(u.pathname==='/api/queue/resources')return route.fulfill({json:autoProbe?{supported:true,cpu:16,totalMB:65536,availableMB:62500,budgetMB:50000}:{supported:false}});
 if(u.pathname==='/api/queue/export'&&u.searchParams.get('format')==='txt'){txtRequests++;return route.fulfill({contentType:'text/plain',body:'ORCA Nova | Batch Research Summary\n'})}
 if(u.pathname==='/api/queue'&&route.request().method()==='POST'){submitted=route.request().postDataJSON();return route.fulfill({json:{ok:true}})}
 if(u.pathname==='/api/queue')return route.fulfill({json:{jobs:queueFixture|| (compareMode?[{id:'j1',label:'Test molecule',state:'Completed',request:{stages:[{name:'S0 Opt',input:'first.inp',source:'mol1.xyz',opt:true}]}}]:[]),paused:false,limits:{concurrency:2,cpu:4,ramMB:8192},batchEta:'약 2분 ~ 약 5분'}});
 if(u.pathname==='/api/queue/results')return route.fulfill({json:{Stages:[{Stage:{name:'S0 Opt',source:'mol1.xyz'},Analysis:{Status:'Completed',Method:'PBE0 def2-SVP',Geometry:'mol1.xyz',Electronic:-100.123,Roots:[{Kind:'S',Root:1,EV:3.14},{Kind:'T',Root:1,EV:2.71}],Populations:{Mulliken:[{Index:4,Element:'C',Spin:0.125},{Index:5,Element:'N',Spin:-0.25}]}}},{Stage:{name:'Unrun stage',source:'first.xyz'},Analysis:{Status:'Not calculated'}}]}});
 if(u.pathname==='/api/queue/preflight')return route.fulfill({json:{ok:true,reports:[]}});
 if(u.pathname==='/api/status')return route.fulfill({json:{config,job:singleJob,eta:singleETA}});
 if(['/api/reveal','/api/queue/action'].includes(u.pathname)){revealRequests.push({path:u.pathname,body:route.request().postDataJSON()});return revealError?route.fulfill({status:400,json:{error:'결과 폴더를 찾을 수 없습니다'}}):route.fulfill({json:{ok:true}});}
 if(u.pathname==='/api/config'){config=route.request().postDataJSON();return route.fulfill({json:{ok:true}})}
 if(u.pathname==='/api/preflight')return route.fulfill({json:{ok:true,stages:[],errors:[],warnings:[],allocation:route.request().postDataJSON().autoCPU?{cpu:4,ramMB:16000}:null}});
 if(u.pathname==='/api/run'){singleRun=route.request().postDataJSON();return route.fulfill({json:{ok:true,folder:'/tmp/jobs/standalone',allocation:{cpu:4,ramMB:16000}}})}
 return route.fulfill({json:{ok:true}});
 });await page.goto('http://orca.test/?session=mock');await page.locator('#nav-batch').click();
 await page.waitForFunction(()=>document.querySelector('#live-cpu').textContent==='37.5%');
 assert.equal(await page.locator('#live-ram').innerText(),'24.0 / 64.0 GB');
 assert.equal(await page.locator('#live-ram-bar').evaluate(el=>el.value),37.5);
 await page.evaluate(()=>{active='W2';$('cores').value='2';$('maxcore').value='100';});
 const xyz='3\nwater\nO 0 0 0\nH 0.757 0.586 0\nH -0.757 0.586 0\n';
 await page.locator('#batch-files').setInputFiles([1,2,3,4].map(n=>({name:'mol'+n+'.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)})));
 await page.waitForFunction(()=>batchMolecules.length===4);
 assert.equal(await page.locator('#batch-molecules tr').count(),4);
 const payload=await page.evaluate(()=>batchPayload());assert.equal(payload.jobs.length,4);for(const [i,r]of payload.jobs.entries()){assert.equal(r.stages.length,2);assert(r.files.some(f=>f.name==='mol'+(i+1)+'.xyz'));assert.equal(r.stages[1].source,r.stages[0].input.replace('.inp','.xyz'));}
 assert.equal(await page.evaluate(()=>loadedXYZ),'');
 await page.locator('#batch-concurrency').fill('2');await page.locator('#batch-start').click();await page.waitForFunction(()=>document.querySelector('#batch-status').textContent.includes('등록했습니다'));assert.equal(submitted.jobs.length,4);assert.equal(submitted.limits.concurrency,2);
 // Exclusion/deletion and custom INP editing survive rendering/queue submission.
 await page.locator('#batch-molecules tr').first().locator('input[aria-label="Batch 실행 포함"]').uncheck();assert.equal(await page.evaluate(()=>batchPayload().jobs.length),3);
 await page.locator('#batch-molecules tr').last().getByText('등록 삭제',{exact:true}).click();assert.equal(await page.evaluate(()=>batchMolecules.length),3);
 const custom='! PBE0 def2-SVP\n%maxcore 100\n* xyzfile 0 1 mol1.xyz\n';
 await page.locator('#batch-files').setInputFiles([{name:'custom.inp',mimeType:'text/plain',buffer:Buffer.from(custom)}]);await page.waitForFunction(()=>batchMolecules[0].preset==='Imported INP');assert.equal(await page.evaluate(()=>batchMolecules[0].request.stages[0].input),'custom.inp');

 const chained=[{name:'next.inp',mimeType:'text/plain',buffer:Buffer.from('! PBE0 def2-SVP Opt\n%maxcore 100\n* xyzfile 0 3 first.xyz\n')},{name:'first.inp',mimeType:'text/plain',buffer:Buffer.from('! PBE0 def2-SVP Opt\n%maxcore 100\n* xyzfile 0 1 mol2.xyz\n')}];
 await page.locator('#batch-files').setInputFiles(chained);await page.waitForFunction(()=>batchMolecules[1].request.stages.length===2&&batchMolecules[1].request.stages[0].input==='first.inp');
 assert.equal(await page.evaluate(()=>batchMolecules[1].request.stages[1].source),'first.xyz');
 await page.locator('#batch-drop').dispatchEvent('drop',{dataTransfer:await page.evaluateHandle(xyz=>{const dt=new DataTransfer();dt.items.add(new File([xyz],'drop.xyz',{type:'text/plain'}));return dt},xyz)});await page.waitForFunction(()=>batchMolecules.some(m=>m.name==='drop.xyz'));

 compareMode=true;await page.locator('.batch-comparison summary').click();await page.locator('#batch-compare-refresh').click();await page.waitForFunction(()=>batchComparisonRows.length===2);
 const comparison=await page.locator('#batch-comparison-body').innerText();assert(comparison.includes('3.14'));assert(comparison.includes('2.71'));assert(comparison.includes('0.125 (Mulliken; ORCA 4 C)'));assert(comparison.includes('-0.25 (Mulliken; ORCA 5 N)'));assert(comparison.includes('Not calculated'));
 const csv=await page.evaluate(()=>batchComparisonCSV());assert(csv.includes('PBE0 def2-SVP'));assert(csv.includes('mol1.xyz'));assert(csv.includes('3.14'));assert(!comparison.includes('C5'));
 await page.locator('#batch-atom-a').fill('999');await page.locator('#batch-compare-refresh').click();await page.waitForFunction(()=>batchComparisonRows[0][8]==='Unavailable');
 await page.evaluate(()=>batchPoll());assert((await page.locator('#batch-queue-info').innerText()).includes('약 2분 ~ 약 5분'));
 assert((await page.locator('#batch-queue-info').innerText()).includes('예약 0/4 CPU, 0/8192 MB RAM'));
 assert(await page.locator('#batch-txt').isVisible(),'full Batch TXT export should be reachable');
 await page.locator('#batch-txt').click();
 assert.equal(txtRequests,1,'TXT export must request the dedicated authenticated format');
 assert(await page.locator('.preset-pill[data-class="recommended"]').count()>0);
 assert(await page.locator('.preset-pill[data-class="preset"]').count()>0);
 const badgeTitle=await page.locator('.preset-pill[data-class="recommended"]').first().getAttribute('title');
 assert(badgeTitle.includes('계산법 검증이나 논문 재현이 아닙니다'));
 const methodTypography=await page.evaluate(()=>({
  functional:getComputedStyle(document.querySelector('#picker-functional .picker-button')).fontSize,
  basis:getComputedStyle(document.querySelector('#picker-basis .picker-button')).fontSize
 }));
 assert.equal(methodTypography.functional,'13px');
 assert.equal(methodTypography.basis,'13px');

 // New UI: two/four parallel presets reveal resource budget restrictions.
 await page.locator('#batch-two-parallel').click();
 assert.equal(await page.locator('#batch-concurrency').inputValue(),'2');
 assert((await page.locator('#batch-budget-note').innerText()).includes('CPU'));
 await page.locator('#batch-four-parallel').click();
 assert.equal(await page.locator('#batch-concurrency').inputValue(),'4');

 // Selection is distinct from the checkbox that includes a molecule in a run.
 const originalCount=await page.evaluate(()=>batchMolecules.length);
 await page.locator('#batch-molecules tr').nth(0).locator('input[aria-label="목록 선택"]').check();
 await page.locator('#batch-molecules tr').nth(1).locator('input[aria-label="목록 선택"]').check();
 assert((await page.locator('#batch-selected-count').innerText()).includes('2개 선택'));
 page.once('dialog',d=>d.accept());
 await page.locator('#batch-remove-selected').click();
 assert.equal(await page.evaluate(()=>batchMolecules.length),originalCount-2);
 assert(!(await page.locator('#batch-remove-selected').isEnabled()));
 // Users can clear all remaining registrations without touching submitted queue.
 page.once('dialog',d=>d.accept());
 await page.locator('#batch-clear-all').click();
 assert.equal(await page.evaluate(()=>batchMolecules.length),0);
 assert.equal(submitted.jobs.length,4); // Already submitted jobs remain untouched.

 // Primary and additional XYZ can transfer into Batch and be removed independently.
 await page.locator('#nav-calculation').click();
 await page.locator('#xyz-file').setInputFiles({name:'primary.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)});
 await page.locator('#desktop-extra-xyz').setInputFiles([{name:'extra.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)}]);
 await page.waitForFunction(()=>desktopXYZ.has('extra.xyz'));
 assert(await page.locator('#xyz-remove').isEnabled());
 assert.equal(await page.locator('#desktop-extra-list .xyz-extra-item').count(),1);
 await page.locator('#desktop-to-batch').click();
 await page.waitForFunction(()=>batchMolecules.length===2);
 assert.equal(await page.evaluate(()=>batchMolecules.map(m=>m.name).sort().join(',')),'extra.xyz,primary.xyz');
 assert(await page.locator('#page-batch').isVisible());await page.locator('#nav-calculation').click();
 await page.locator('#desktop-extra-list .xyz-extra-item button[aria-label$="추가 XYZ 등록 해제"]').click();
 assert.equal(await page.evaluate(()=>desktopXYZ.size),0);
 await page.locator('#xyz-remove').click();
 assert.equal(await page.evaluate(()=>loadedXYZ),'');
 assert.equal(await page.evaluate(()=>xyzOriginal),'');
 assert.equal(await page.evaluate(()=>batchMolecules.length),2); // Registration copies persist.
 await page.locator('#desktop-extra-xyz').setInputFiles([{name:'extra2.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)}]);
 await page.waitForFunction(()=>xyzOriginal.length>0); // FileReader async after input event
 page.once('dialog',d=>d.accept());
 await page.locator('#desktop-clear-xyz').click();
 assert.equal(await page.evaluate(()=>xyzOriginal),'');
 assert.equal(await page.evaluate(()=>desktopXYZ.size),0);
 assert.equal(await page.evaluate(()=>batchMolecules.length),2);

 // Automatic mode hides manual controls and distinguishes generated from edited INPs.
 autoProbe=true;
 await page.locator('#nav-batch').click();
 await page.evaluate(()=>batchLoadResourceSystem());
 assert(await page.locator('#batch-auto').isChecked());
 assert(!(await page.locator('#batch-manual-options').isVisible()));
 await page.locator('#batch-cpu').fill('16');
 let autoPayload=await page.evaluate(()=>batchPayload());
 assert.equal(autoPayload.limits.cpu,16);
 assert.deepEqual(autoPayload.auto.eligible,[true,true]);
 const detail=page.locator('#batch-molecules tr').first().locator('details').first();
 await detail.locator('summary').click();
 const text=detail.locator('textarea');
 await text.fill((await text.inputValue())+'\n# user edit\n');
 autoPayload=await page.evaluate(()=>batchPayload());
 assert.deepEqual(autoPayload.auto.eligible,[false,true]);
 await page.locator('#batch-start').click();
 await page.waitForFunction(()=>document.querySelector('#batch-status').textContent.includes('등록했습니다'));
 assert.deepEqual(submitted.auto.eligible,[false,true]);
 await page.locator('#batch-auto').uncheck();
 assert(await page.locator('#batch-manual-options').isVisible());
 assert(!(await page.evaluate(()=>batchPayload())).auto);
  // Live queue filtering and explicit append selection are independent of source registration.
 queueFixture=[{id:'running1',label:'T1 running',state:'Running',cpu:2,ramMB:4000,autoResources:true,autoEligible:true,eta:{elapsedSeconds:125,stageEta:'약 2분 ~ 약 3분',workflowEta:'약 4분 ~ 약 7분',estimateBasis:'유사 계산 기록 3개',optimizationCycle:5}},{id:'pending1',label:'T2 waiting',state:'Pending',waitReason:'가용 물리 RAM 대기'},{id:'done1',label:'SP done',state:'Completed',job:{folder:'/tmp/jobs/SP done'}},{id:'done2',label:'Opt done',state:'Completed',job:{folder:'/tmp/jobs/Opt done'}},{id:'failed1',label:'Opt failed',state:'Failed'}];
 await page.evaluate(()=>batchPoll());
 await page.waitForFunction(()=>document.querySelector('#batch-queue').textContent.includes('가용 물리 RAM 대기'));
 assert(await page.locator('#batch-eta-overview').isVisible());assert.equal(await page.locator('#batch-eta-total').innerText(),'약 2분 ~ 약 5분');
 await page.locator('#queue-filter').selectOption('active');assert.equal(await page.locator('#batch-queue tr').count(),2);
 assert((await page.locator('#batch-queue tr').first().innerText()).includes('Job 남은 시간: 약 4분 ~ 약 7분'));
 assert((await page.locator('#batch-queue tr').first().innerText()).includes('현재 단계: 약 2분 ~ 약 3분'));
 await page.locator('#queue-search').fill('T2');assert.equal(await page.locator('#batch-queue tr').count(),1);
 await page.locator('#queue-search').fill('');await page.locator('#queue-filter').selectOption('failed');assert((await page.locator('#batch-queue').innerText()).includes('Opt failed'));
 await page.locator('#batch-append').check();assert.equal(await page.evaluate(()=>batchPayload().append),true);
 assert((await page.locator('#batch-start').innerText()).includes('Queue에 추가'));
 // Folder buttons are visible on completion and use the exact backend job identity.
 await page.locator('#queue-filter').selectOption('completed');
 assert.equal(await page.locator('#batch-queue tr').count(),2);
 await page.getByRole('button',{name:'SP done 결과 폴더 열기',exact:true}).click();
 await page.waitForFunction(()=>true);
 for(let i=0;i<30&&!revealRequests.length;i++)await new Promise(r=>setTimeout(r,20));
 assert.deepEqual(revealRequests.at(-1),{path:'/api/queue/action',body:{action:'reveal',id:'done1'}});
 await page.evaluate(()=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async value=>{window.copiedFolder=value}}}));
 await page.getByRole('button',{name:'Opt done 결과 폴더 경로 복사',exact:true}).click();
 assert.equal(await page.evaluate(()=>window.copiedFolder),'/tmp/jobs/Opt done');
 await page.locator('#nav-results').click();
 await page.locator('#completed-folder-select').selectOption('done1');
 assert.equal(await page.locator('#completed-folder-path').innerText(),'/tmp/jobs/SP done');
 await page.evaluate(()=>batchPoll());
 assert.equal(await page.locator('#completed-folder-select').inputValue(),'done1');
 await page.locator('#completed-folder-copy').click();
 assert.equal(await page.evaluate(()=>window.copiedFolder),'/tmp/jobs/SP done');
 await page.locator('#completed-folder-open').click();
 for(let i=0;i<30&&revealRequests.length<2;i++)await new Promise(r=>setTimeout(r,20));
 assert.equal(revealRequests.at(-1).body.id,'done1');
 await page.locator('#nav-calculation').click();
 assert(await page.locator('#desktop-reveal-quick').isDisabled());
 singleJob={folder:'/tmp/jobs/standalone',state:'running',stageCount:1,stageIndex:0,stageName:'Opt',message:'실행 중'};
 singleETA={elapsedSeconds:3671,stageEta:'약 3분 ~ 약 8분',workflowEta:'약 3분 ~ 약 8분',estimateBasis:'유사 설정 완료 기록 2개',optimizationCycle:12};
 await page.evaluate(()=>desktopPoll());
 assert(await page.locator('#quick-eta-overview').isVisible());
 assert.equal(await page.locator('#quick-eta-elapsed').innerText(),'1시간 1분 11초');
 assert.equal(await page.locator('#quick-eta-stage').innerText(),singleETA.stageEta);
 assert.equal(await page.locator('#quick-eta-workflow').innerText(),singleETA.workflowEta);
 await page.locator('#nav-results').click();assert.equal(await page.locator('#eta-stage').innerText(),singleETA.stageEta);
 await page.locator('#nav-calculation').click();
 singleETA={elapsedSeconds:3700,stageEta:'추정 중 · 유사 계산 기록 없음',workflowEta:'추정 중'};
 await page.evaluate(()=>desktopPoll());assert.equal(await page.locator('#quick-eta-workflow').innerText(),'추정 중');
 singleJob={folder:'/tmp/jobs/standalone',state:'completed',stageCount:1,stageIndex:0,stageName:'Opt',message:'완료'};
 singleETA={elapsedSeconds:3800,stageEta:'계산 종료',workflowEta:'계산 종료'};
 await page.evaluate(()=>desktopPoll());
 assert(await page.locator('#desktop-reveal-quick').isEnabled());
 assert.equal(await page.locator('#quick-eta-state').innerText(),'정상 종료');assert.equal(await page.locator('#quick-eta-workflow').innerText(),'계산 종료');
 await page.locator('#desktop-reveal-quick').click();
 for(let i=0;i<30&&revealRequests.length<3;i++)await new Promise(r=>setTimeout(r,20));
 assert.equal(revealRequests.at(-1).path,'/api/reveal');
 await page.locator('#nav-results').click();
 assert.equal(await page.locator('#single-result-path').innerText(),singleJob.folder);
 revealError=true;
 await page.locator('#desktop-reveal').click();
 await page.waitForFunction(()=>document.querySelector('#toast').textContent.includes('결과 폴더 열기 실패'));
 revealError=false;singleJob=null;await page.evaluate(()=>desktopPoll());
 await page.locator('#nav-calculation').click();
 await page.locator('#completed-folder-shortcut').click();
 assert(await page.locator('#completed-folder-select').isVisible());
 // Editing paths must survive status polling and switching workspace.
 await page.locator('#nav-settings').click();
 await page.locator('#desktop-work-root').fill('/tmp/custom-path');
 await page.locator('#desktop-orca-path').focus();
 await page.evaluate(()=>desktopPoll());
 assert.equal(await page.locator('#desktop-work-root').inputValue(),'/tmp/custom-path');
 assert((await page.locator('#settings-dirty').innerText()).includes('저장하지 않은'));
 await page.locator('#nav-calculation').click();await page.evaluate(()=>desktopPoll());
 await page.locator('#nav-settings').click();assert.equal(await page.locator('#desktop-work-root').inputValue(),'/tmp/custom-path');
 await page.locator('#desktop-save-config').click();await page.waitForFunction(()=>document.querySelector('#settings-dirty').textContent==='경로 저장 완료');
 assert.equal(config.workRoot,'/tmp/custom-path');
 // Standalone mode: resource preview, actual request, and manual INP protection.
 await page.locator('#nav-calculation').click();
 await page.evaluate(()=>singleLoadResourceSystem());
 assert(await page.locator('#single-auto').isChecked());
 assert(!(await page.locator('#single-manual-maxcore').isVisible()));
 await page.locator('#cores').fill('4');
 await page.evaluate(()=>selectedCalc('A1'));
 await page.locator('#xyz-file').setInputFiles({name:'standalone.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)});
 await page.waitForFunction(()=>loadedXYZ==='standalone.xyz');
 assert((await page.locator('#editor').inputValue()).includes('nprocs 4'));
 assert.equal(await page.evaluate(()=>desktopRunData(false).autoCPU),4);
 await page.locator('#desktop-preflight').click();await page.waitForFunction(()=>document.querySelector('#single-preflight-report').textContent.includes('자동 배분 예상'));
 await page.locator('#desktop-run').click();await page.waitForFunction(()=>document.querySelector('#single-run-allocation').textContent.includes('4 CPU'));
 assert.equal(singleRun.autoCPU,4);
 assert.equal(config.autoCPU,4);
 await page.locator('#cores').fill('0');
 assert(await page.evaluate(()=>{try{desktopRunData(false);return false}catch(e){return e.message.includes('CPU')}}),'zero CPU must not silently fall back to Manual');
 await page.locator('#cores').fill('4');
 const edited=(await page.locator('#editor').inputValue())+'\n# researcher manual edit';
 await page.locator('#editor').fill(edited);
 assert(!(await page.evaluate(()=>desktopRunData(false))).autoCPU);
 assert.equal(await page.locator('#editor').inputValue(),edited);
 assert(await page.locator('#single-manual-maxcore').isVisible());
 // Global preference follows all Auto resource panels, but Manual values stay independent.
 await page.evaluate(()=>tripLoadResourceSystem());
 assert.equal(await page.locator('#trip-budget-cpu').inputValue(),'4');
 await page.locator('#nav-batch').click();await page.locator('#batch-auto').check();
 assert.equal(await page.locator('#batch-cpu').inputValue(),'4');
 await page.locator('#batch-cpu').fill('6');
 assert.equal(await page.locator('#cores').inputValue(),'6');
 assert.equal(await page.locator('#trip-budget-cpu').inputValue(),'6');
 assert.equal(await page.locator('#editor').inputValue(),edited,'global CPU change must preserve edited INP');
 await page.locator('#batch-auto').uncheck();await page.locator('#batch-cpu').fill('3');
 assert.equal(await page.locator('#cores').inputValue(),'6');
 await page.locator('#batch-auto').check();assert.equal(await page.locator('#batch-cpu').inputValue(),'6');
 await page.locator('#nav-settings').click();await page.locator('#desktop-save-config').click();
 await page.waitForFunction(()=>document.querySelector('#auto-cpu-setting-status').textContent.includes('저장됨'));
 assert.equal(config.autoCPU,6);
 await page.reload();
 await page.waitForFunction(()=>['single-auto','batch-auto','trip-auto'].every(id=>document.getElementById(id).checked)&&['cores','batch-cpu','trip-budget-cpu'].every(id=>document.getElementById(id).value==='6'));
 assert.deepEqual(errors,[]);console.log('PASS browser: multi XYZ batch, deletion, transfers, parallel budget, live queue untouched and no JS errors');
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
