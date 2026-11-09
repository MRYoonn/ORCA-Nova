// Dependency-free syntax and provenance guard for embedded HTML UI scripts.
// Does not replace real Chromium / Windows WebView2 behavioral testing.
const fs = require('fs');
const vm = require('vm');
const assert = require('assert');
const html = fs.readFileSync('assets/desktop.html','utf8');
const scripts=[...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/gi)].map(m=>m[1]);
assert(scripts.length>=3,'embedded scripts unexpectedly missing');
for (let i=0;i<scripts.length;i++){
  try {new vm.Script(scripts[i],{filename:`desktop-inline-${i}.js`})}
  catch(e){throw new Error(`Inline script ${i+1}: ${e.message}`)}
}
for (const id of ['viz-profile-kind','viz-profile-energy','viz-role-a','viz-role-b',
  'viz-role-c','viz-profile-confirm','viz-profile-run','viz-profile-result']){
  assert(html.includes(`id="${id}"`),`missing energy profile control ${id}`);
}
assert(html.includes("D3ZERO"),'Hartmann published D3zero keyword missing');
assert(html.includes("def2-QZVPPD"),'Hartmann published SP basis missing');
assert(html.includes("10.1002/anie.9565676"),'DOI provenance missing');
assert(html.includes("data-class=\"${p.tag.includes(\"추천\")"),'preset class styling missing');
console.log(`Inline JS parsed: ${scripts.length} scripts; profile controls and DOI present.`);
