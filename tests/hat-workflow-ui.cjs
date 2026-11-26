const { chromium } = require('playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');

(async () => {
  const browser = await chromium.launch({ headless: true, ...(process.env.CHROMIUM_PATH ? {executablePath:process.env.CHROMIUM_PATH} : {}) });
  try {
    const page = await browser.newPage();
    const errors = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('dialog', d => d.accept());
    const html = fs.readFileSync(path.join(__dirname, '../assets/desktop.html'), 'utf8');
    const reactant = '3\nReactant\nC 0 0 0\nC 1.5 0 0\nH 0 1 0\n';
    const product = '3\nProduct\nC 0 0 0\nC 1.5 0 0\nH 1.5 1 0\n';
    let checked;
    await page.route('**/*', route => {
      const url = new URL(route.request().url());
      if (url.pathname === '/') return route.fulfill({ contentType: 'text/html', body: html });
      if (url.pathname === '/api/hat/mapping') {
        checked = route.request().postDataJSON();
        return route.fulfill({json: {valid:true,changes:[{a:0,b:2,formed:false},{a:1,b:2,formed:true}],
          atoms:['C','C','H'],reactant:[[0,0,0],[1.5,0,0],[0,1,0]],product:[[0,0,0],[1.5,0,0],[1.5,1,0]],
          bonds:[[0,1],[0,2]],aligned:false,swapCandidates:[]}});
      }
      return route.fulfill({json:{}});
    });
    await page.goto('http://localhost/?session=browser-qa');
    const audit = await page.evaluate(() => {
      const defaultValues = ['functional','basis','dispersion','solvmodel','solvent','scf','mult','dft-grid'].map(id=>[id,$(id).value]);
      const originalActive=active;
      const records=[];
      const all = () => stages().flatMap((s,i)=>s[1]?[buildInput(i,false)]:[]);
      for (const item of DATA) {
        active=item.id;
        for(const [id,value] of defaultValues)$(id).value=value;
        records.push(...all().map(input=>({id:item.id,input})));
      }
      for (const preset of PRESETS) {
        for(const [id,value] of defaultValues)$(id).value=value;
        applyPreset(preset.id);
        records.push(...all().map(input=>({id:preset.id,input})));
      }
      active=originalActive;
      for(const [id,value] of defaultValues)$(id).value=value;
      generate(true);
      return {records, table:ORCA6_FUNCTIONALS, calculationCount:DATA.length, presetCount:PRESETS.length};
    });
    const starts = new Set([...Object.values(audit.table).map(x=>x.toLowerCase()),'goat']);
    for(const {id,input} of audit.records){
      const line = input.split('\n').find(row=>row.startsWith('! '));
      assert(line, id+' lacks ORCA simple input');
      const tokens = line.slice(2).trim().toLowerCase().split(/\s+/);
      assert(starts.has(tokens[0]) || ((tokens[0]==='uks'||tokens[0]==='rks')&&starts.has(tokens[1])), id+' unknown functional '+line);
      assert(!tokens.includes('m06-2x'), id+' uses the literature alias in executable input');
      assert.equal(new Set(tokens).size,tokens.length,id+' duplicate keyword: '+line);
      assert.equal(input.split('\n').filter(l=>l.startsWith('* xyzfile ')).length,1,id+' invalid XYZ reference count');
    }
    assert(audit.calculationCount>0 && audit.presetCount>0);
    assert(audit.records.some(r=>r.input.includes('def2-QZVPPD')));
    assert(audit.records.some(r=>r.input.includes(' NEB-TS')));
    for(const functional of ['M06-2X','r2SCAN-3c','wB97X-D3']){
      const inp=await page.evaluate(fn=>{active='A2';$('functional').value=fn;return buildInput(0,false)},functional);
      assert(inp.includes(funcKeyword(functional)));
    }

    await page.locator('#xyz-file').setInputFiles({name:'reactant.xyz',mimeType:'text/plain',buffer:Buffer.from(reactant)});
    await page.waitForFunction(()=>loadedXYZ==='reactant.xyz');
    await page.locator('[data-preset="hat-neb"]').click();
    await page.evaluate(raw=>desktopXYZ.set('product.xyz',raw), product);
    await page.locator('#hat-h').fill('2');
    await page.locator('#hat-donor').fill('0');
    await page.locator('#hat-acceptor').fill('1');
    await page.locator('#hat-check-mapping').click();
    await page.waitForFunction(()=>document.querySelector('#hat-review').textContent.includes('단일 H 이동과 일치'));
    assert.deepEqual(checked.mapping,{h:2,donor:0,acceptor:1});
    assert(await page.locator('#hat-preview').isVisible());
    const stage1 = await page.locator('#editor').inputValue();
    await page.locator('#editor').fill(stage1.replace('M062X','M06-2X')+'\n# manual R');
    await page.locator('#workflow-box [data-stage="1"]').click();
    await page.locator('#editor').fill((await page.locator('#editor').inputValue())+'\n# manual P');
    await page.locator('#workflow-box [data-stage="0"]').click();
    assert((await page.locator('#editor').inputValue()).includes('# manual R'));
    const req = await page.evaluate(()=>desktopRunData(true));
    assert(req.files.find(f=>f.name===req.stages[0].input).content.includes('M062X'));
    assert(req.files.find(f=>f.name===req.stages[1].input).content.includes('# manual P'));
    assert(req.stages[3].source.endsWith('_NEB-TS_converged.xyz'));
    assert(req.files.find(f=>f.name===req.stages[3].input).content.includes('Direction both'));
    assert(req.files.find(f=>f.name===req.stages[2].input).content.includes('DEFGRID3'));
    await page.locator('#hat-reset-edits').click();
    assert(!(await page.locator('#editor').inputValue()).includes('# manual R'));
    await page.locator('#hat-run-mode').selectOption('step3');
    await page.locator('#hat-restart-folder').fill('C:\\jobs\\previous');
    assert.equal((await page.evaluate(()=>desktopRunData(true))).hatRestart.mode,'step3');
    assert.deepEqual(errors,[]);
    console.log(`PASS ${audit.calculationCount} calculations, ${audit.presetCount} presets, ${audit.records.length} generated INPs; D9 mapping, stage edits and restart mode`);
  } finally {
    await browser.close();
  }
})().catch(error=>{console.error(error);process.exit(1)});
function funcKeyword(label){return label==='M06-2X'?'M062X':label;}
