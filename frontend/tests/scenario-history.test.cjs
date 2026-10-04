const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const ts = require('typescript');
const vm = require('node:vm');
const output = ts.transpileModule(fs.readFileSync('src/shared/lib/scenarioHistory.ts','utf8'), {compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020}}).outputText;
const sandbox={exports:{}};
vm.runInNewContext(output,sandbox);
const {groupScenarioHistory,scenarioKey}=sandbox.exports;
const point={at:1,score:100,accuracy:.9,version:'v1',gameVersion:'1',sensScale:'cm/360',sens:30,vertSens:30,dpi:800,fov:103,fovScale:'source',seconds:60,targetScale:1,timeScale:1};

test('history sorts chronologically, keeps zero and single scores, and never mutates the input',()=>{
 const input=[{...point,at:3,score:0},{...point,at:1},{...point,at:2,score:200},{...point,at:4,score:NaN}];
 const groups=groupScenarioHistory(input);
 assert.equal(groups.length,1);
 assert.deepEqual(Array.from(groups[0].points,p=>p.score),[100,200,0]);
 assert.equal(input[0].at,3);
 assert.equal(groupScenarioHistory([point])[0].points.length,1);
 assert.equal(groupScenarioHistory([]).length,0);
 assert.equal(scenarioKey('  MAP X  '),'map x');
});

test('versions and each measurement condition stay separate with the most recent condition first',()=>{
 for(const [field,value] of Object.entries({version:'v2',gameVersion:'2',sensScale:'Valorant',sens:40,vertSens:20,dpi:400,fov:90,fovScale:'Quake',seconds:30,targetScale:.8,timeScale:.5})){
  const groups=groupScenarioHistory([point,{...point,at:2,[field]:value}]);
  assert.equal(groups.length,2,field);
  assert.equal(groups[0].points[0].at,2,field);
 }
});
