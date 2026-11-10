// UX regression: row-major two-column presets and default-folded calculation library.
// Independent of ORCA and any research materials.
const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');
(async()=>{
 const browser=await chromium.launch({headless:true,...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{}),args:['--no-sandbox']});
 try{
  const page=await browser.newPage({viewport:{width:1600,height:930},deviceScaleFactor:1}),errors=[];
  page.on('pageerror',e=>errors.push(e.message));
  await page.route('http://orca.test/**',route=>{
   const u=new URL(route.request().url());
   if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'../assets/desktop.html'),'utf8')});
   if(u.pathname==='/api/status')return route.fulfill({json:{config:{workRoot:'/tmp/jobs'},update:{channel:'stable',current:'1.6.0',available:false}}});
   if(u.pathname==='/api/queue')return route.fulfill({json:{jobs:[],paused:true}});
   if(u.pathname==='/api/update/check')return route.fulfill({json:{channel:'stable',current:'1.6.0',available:false}});
   return route.fulfill({json:{ok:true}});
  });
  await page.goto('http://orca.test/?session=test');
  const wb=page.locator('#workbench'),toggle=page.locator('#library-toggle');
  assert(await wb.evaluate(el=>el.classList.contains('library-collapsed')),'library must be folded by default');
  assert.equal(await toggle.getAttribute('aria-expanded'),'false');
  assert(!(await page.locator('#library-content').isVisible()),'library lists should not occupy space initially');
  assert.equal(await page.locator('#calc-nav details.nav-category').count(),6);
  assert.equal(await page.locator('#calc-nav details[open]').count(),0,'all calculation groups should start collapsed');
  const groupNames=await page.locator('#preset-grid .preset-group').evaluateAll(nodes=>nodes.map(el=>el.dataset.kind));
  assert.deepEqual(groupNames,['literature','recommended','preset'],'preset groups must be literature > recommended > preset');
  const uiOrder=await page.locator('#preset-grid .preset-item').evaluateAll(els=>els.map(el=>el.dataset.preset));
  const expected=await page.evaluate(()=>PRESET_CATEGORIES.flatMap(cat=>PRESETS.filter(p=>presetClass(p)===cat.id).map(p=>p.id)));
  assert.deepEqual(uiOrder,expected);
  assert.equal(new Set(uiOrder).size,await page.evaluate(()=>PRESETS.length),'preset duplication or omission');
  assert.equal(await page.locator('#preset-grid .preset-group[data-kind="literature"] .preset-item').count(),2);
  // CSS grid must be row-major, not column-major.
  for(const group of await page.locator('#preset-grid .preset-group').all()){
   const cards=group.locator('.preset-item');if(await cards.count()<2)continue;
   const a=await cards.nth(0).boundingBox(),b=await cards.nth(1).boundingBox();
   assert(Math.abs(a.y-b.y)<2 && a.x<b.x,'2-column row-major layout incorrect');
   if(await cards.count()>2){
    const c=await cards.nth(2).boundingBox();
    assert(c.y>a.y+10&&Math.abs(c.x-a.x)<2,'third card should begin the next row, left');
   }
  }
  const config=await page.locator('.config-panel').boundingBox();
  assert(config.width>640,'preset/configuration panel is still too narrow on desktop '+config.width);
  const titleHeights=await page.locator('#preset-grid .preset-title').evaluateAll(nodes=>nodes.map(n=>({name:n.textContent,actual:n.getBoundingClientRect().height,line:parseFloat(getComputedStyle(n).lineHeight)})));
  assert(titleHeights.every(x=>x.actual<x.line*1.4),'preset title unexpectedly wraps on 1600px desktop: '+JSON.stringify(titleHeights.filter(x=>x.actual>=x.line*1.4)));
  await fs.promises.mkdir(path.join(__dirname,'../ui-screenshots'),{recursive:true});
  await page.screenshot({path:path.join(__dirname,'../ui-screenshots/preset-default-1600.png')});
  // Original Method selection must remain functional.
  await page.locator('[data-preset="glorius-hartmann-opt"]').click();
  assert.equal(await page.evaluate(()=>active),'A4');
  assert.equal(await page.locator('#functional').inputValue(),'M06-2X');
  assert.equal(await page.locator('[data-preset="glorius-hartmann-opt"]').getAttribute('data-picked'),'true');
  assert.equal(await toggle.getAttribute('aria-expanded'),'false','selecting preset must not expand library');
  // Expand sidebar, then expand only W category.
  await toggle.click();
  assert(await page.locator('#library-content').isVisible());
  assert.equal(await toggle.getAttribute('aria-expanded'),'true');
  assert.equal(await page.locator('#calc-nav details[open]').count(),0);
  await page.locator('#calc-nav details[data-group="W"] > summary').click();
  assert(await page.locator('#calc-nav details[data-group="W"]').evaluate(x=>x.open));
  await page.locator('#calc-nav [data-id="W2"]').click();
  assert.equal(await page.evaluate(()=>active),'W2');
  assert(await page.locator('#calc-nav details[data-group="W"]').evaluate(x=>x.open),'manual group open must survive renderNav');
  // Search automatically reveals groups temporarily.
  await page.locator('#search').fill('Gibbs');
  assert((await page.locator('#calc-nav details[open]').count())>=1);
  await page.locator('#search').fill('');
  assert(await page.locator('#calc-nav details[data-group="W"]').evaluate(x=>x.open));
  assert.equal(await page.locator('#calc-nav details[open]').count(),1,'search must not permanently expand other groups');
  await toggle.click();
  assert(!(await page.locator('#library-content').isVisible()));
  assert(await page.locator('#preset-grid').isVisible());
  await page.setViewportSize({width:1280,height:900});
  const desktop=await page.locator('.config-panel').boundingBox();
  assert(desktop.width>=590,'at 1280px the closed library should leave configuration usable: '+desktop.width);
  await page.screenshot({path:path.join(__dirname,'../ui-screenshots/preset-default-1280.png')});
  await page.setViewportSize({width:780,height:900});
  assert(await toggle.isVisible(),'library must remain accessible in narrow viewport');
  assert.equal(await page.locator('#preset-grid .preset-group').count(),3);
  await toggle.click();
  assert(await page.locator('#search').isVisible(),'library should be usable on narrow screen');
  assert.deepEqual(errors,[]);
  console.log('PASS: preset literature→recommended→preset, two-column row-major, unclipped wide titles, folded sidebar and individually collapsible library groups');
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
