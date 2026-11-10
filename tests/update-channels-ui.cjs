const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');

(async()=>{
 const browser=await chromium.launch({headless:true,args:['--no-sandbox']});
 try {
  const page=await browser.newPage();
  const errors=[],calls=[],prompts=[];
  const states={
   stable:{channel:'stable',current:'1.5.1',available:true,latest:'v1.6.0',releaseUrl:'https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.0',message:'v1.6.0 사용 가능'},
   beta:{channel:'beta',current:'1.5.1',available:true,prerelease:true,latest:'v1.6.0-rc.9',releaseUrl:'https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.0-rc.9',message:'v1.6.0-rc.9 사용 가능'},
  };
  let channel='stable',installed=0;
  page.on('pageerror',e=>errors.push(e.message));
  page.on('dialog',async d=>{prompts.push(d.message());await d.dismiss()});
  await page.route('http://orca.test/**',route=>{
   const r=route.request(),url=new URL(r.url());
   if(url.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'../assets/desktop.html'),'utf8')});
   if(url.pathname==='/api/status')return route.fulfill({json:{version:'1.5.1',config:{workRoot:'/tmp/jobs'},update:{...states[channel],available:false,message:'업데이트 확인 전'}}});
   if(url.pathname==='/api/update/check'){calls.push('check:'+channel);return route.fulfill({json:states[channel]})}
   if(url.pathname==='/api/update/channel'){
    let data=r.postDataJSON();calls.push('channel:'+data.channel);
    assert.equal(r.headers()['x-orca-session'],'x','channel endpoint should use session token');
    channel=data.channel;
    return route.fulfill({json:{channel,current:'1.5.1',available:false,message:channel+' 선택 완료'}})
   }
   if(url.pathname==='/api/update/install'){installed++;return route.fulfill({json:{ok:true,message:'설치 완료'}})}
   if(url.pathname==='/api/queue')return route.fulfill({json:{jobs:[],paused:false}});
   return route.fulfill({json:{ok:true}});
  });
  await page.goto('http://orca.test/?session=x');
  await page.waitForFunction(()=>document.querySelector('#update-release-link')?.textContent.includes('v1.6.0'));
  assert.equal(await page.locator('#update-channel').inputValue(),'stable');
  assert.equal(await page.locator('#update-version').textContent(),'v1.5.1');
  assert.equal(await page.locator('#update-install').isVisible(),true);
  assert.equal(await page.locator('#update-release-link').getAttribute('href'),'https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.0');
  await page.locator('#update-channel').selectOption('beta');
  await page.waitForFunction(()=>document.querySelector('#update-release-link')?.textContent.includes('rc.9'));
  assert(calls.includes('channel:beta')&&calls.includes('check:beta'));
  assert.equal(await page.locator('#update-channel').inputValue(),'beta');
  assert.equal(await page.locator('#update-release-link').getAttribute('href'),'https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.0-rc.9');
  await page.locator('#update-install').click();
  assert.equal(installed,0,'dismissed install confirmation must not invoke updater');
  assert(prompts.some(text=>text.includes('GitHub 릴리스 노트')&&text.includes('v1.6.0-rc.9')),'install prompt should name candidate and user backups');
  await page.locator('#update-channel').selectOption('stable');
  await page.waitForFunction(()=>document.querySelector('#update-release-link')?.textContent.includes('v1.6.0')&&!document.querySelector('#update-release-link')?.textContent.includes('rc.'));
  assert.equal(await page.locator('#update-channel').inputValue(),'stable');
  assert.equal(installed,0);
  assert.deepEqual(errors,[]);
  console.log('PASS: Stable/Beta UI channel persistence, latest stable/beta offers, release notes and approve-before-install');
 } finally {await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
