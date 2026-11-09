// Browser regression for the offline 3D XYZ viewer (Playwright 1.56.1).
const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');
const xyz='3\nWater\nO 0.0000 0.0000 0.0000\nH 0.7586 0.0000 0.5043\nH -0.7586 0.0000 0.5043\n';
(async()=>{
 const browser=await chromium.launch({headless:true,...(process.env.CHROMIUM_PATH?{executablePath:process.env.CHROMIUM_PATH}:{}),args:['--no-sandbox']});
 try{
  const page=await browser.newPage({viewport:{width:1280,height:900}}),errors=[];
  page.on('pageerror',e=>errors.push(e.message));
  await page.route('http://orca.test/**',route=>{
   const u=new URL(route.request().url());
   if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'../assets/desktop.html'),'utf8')});
   if(u.pathname==='/api/queue')return route.fulfill({json:{jobs:[],paused:false}});
   if(u.pathname==='/api/status')return route.fulfill({json:{config:{orcaExe:'/tmp/orca',workRoot:'/tmp/jobs'},version:'dev'}});
   return route.fulfill({json:{ok:true}});
  });
  await page.goto('http://orca.test/?session=mock');
  assert(!(await page.locator('#xyz3d-primary').isEnabled()));
  await page.locator('#xyz-file').setInputFiles({name:'water.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)});
  await page.waitForFunction(()=>xyzOriginal.length>0);
  assert(await page.locator('#xyz3d-primary').isEnabled());
  await page.locator('#xyz3d-primary').click();
  await page.locator('#xyz3d-dialog').waitFor({state:'visible'});
  assert((await page.locator('#xyz3d-info').innerText()).includes('3 atoms'));
  let data=await page.evaluate(()=>({atoms:xyz3dView.atoms.length,bonds:xyz3dView.bonds.length,indices:xyz3dView.atoms.map(a=>a.index),width:$('xyz3d-canvas').width,points:xyz3dView.points.length}));
  assert.equal(data.atoms,3);assert.equal(data.bonds,2);assert.deepEqual(data.indices,[0,1,2]);assert(data.width>0);assert.equal(data.points,3);
  // Atom index uses the first XYZ coordinate line as ORCA index 0.
  const pt=await page.evaluate(()=>xyz3dView.points[0]);
  const canvas=await page.locator('#xyz3d-canvas').boundingBox();
  await page.mouse.click(canvas.x+pt.x,canvas.y+pt.y);
  assert((await page.locator('#xyz3d-selected').innerText()).includes('ORCA index 0'));
  await page.locator('#xyz3d-mark-c5').click();
  assert.equal(await page.evaluate(()=>xyz3dView.marks.C5),0);
  await page.evaluate(()=>xyz3DSelect(1));
  await page.locator('#xyz3d-mark-c6').click();
  assert.deepEqual(await page.evaluate(()=>xyz3dView.marks),{C5:0,C6:1});
  const before=await page.evaluate(()=>({yaw:xyz3dView.yaw,zoom:xyz3dView.zoom}));
  await page.mouse.move(canvas.x+100,canvas.y+110);
  await page.mouse.down();await page.mouse.move(canvas.x+170,canvas.y+140,{steps:6});await page.mouse.up();
  assert.notEqual(await page.evaluate(()=>xyz3dView.yaw),before.yaw);
  await page.mouse.move(canvas.x+200,canvas.y+180);await page.mouse.wheel(0,-400);
  assert((await page.evaluate(()=>xyz3dView.zoom))>before.zoom);
  await page.locator('#xyz3d-reset').click();
  assert.equal(await page.evaluate(()=>xyz3dView.zoom),1);
  await page.locator('#xyz3d-indices').uncheck();
  assert.equal(await page.locator('#xyz3d-indices').isChecked(),false);
  await page.locator('#xyz3d-close').click();
  // Additional XYZ and Batch row both open the matching molecule copy.
  await page.locator('#desktop-extra-xyz').setInputFiles({name:'extra.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)});
  await page.waitForFunction(()=>desktopXYZ.has('extra.xyz'));
  await page.locator('#desktop-extra-list .xyz-extra-item button').first().click();
  assert((await page.locator('#xyz3d-title').innerText()).includes('extra.xyz'));
  await page.locator('#xyz3d-close').click();
  await page.locator('#batch-files').setInputFiles({name:'batch.xyz',mimeType:'text/plain',buffer:Buffer.from(xyz)});
  await page.waitForFunction(()=>batchMolecules.length===1);
  await page.locator('#batch-molecules .batch-inputs button').first().click();
  assert((await page.locator('#xyz3d-title').innerText()).includes('batch.xyz'));
  await page.locator('#xyz3d-close').click();
  const failures=await page.evaluate(()=>{
   const bad=['2\nmissing\nH 0 0 0\n','1\nbad\nH NaN 0 0\n','1\nbad\nQx 0 0 0\n'];
   return bad.map(input=>{try{xyz3DParse(input);return false}catch(e){return true}});
  });
  assert.deepEqual(failures,[true,true,true]);
  await page.locator('#xyz-remove').click();
  assert(!(await page.locator('#xyz3d-primary').isEnabled()));
  assert.equal(await page.evaluate(()=>batchMolecules.length),1);
  assert.deepEqual(errors,[]);
  console.log('PASS: offline XYZ 3D: primary/extra/batch, strict parse, bonds, rotation, zoom, atom 0-based indices, manual C5/C6, deletion isolation');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
