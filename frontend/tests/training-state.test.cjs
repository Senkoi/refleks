const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const ts = require('typescript');
const vm = require('node:vm');
const output = ts.transpileModule(fs.readFileSync('src/features/training/api.ts', 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
const sandbox = { exports: {} };
vm.runInNewContext(output, sandbox);
const { mergeLiveState } = sandbox.exports;

test('clock updates retain catalog, ability evidence and original block configuration', () => {
  const scenario = { name: 'A', localAssessment: { measurements: [{ field: 'size', value: 1 }] } };
  const state = { catalog: [scenario], playerLevels: [{ trainingTier: 'intermediate' }], plan: { id: 'session', status: 'running', blocks: [{ scenario, playCount: 2, role: 'practice', recorded: 0 }] } };
  const live = { revision: 1, initializing: false, notice: '', error: '', plan: { id: 'session', status: 'paused', elapsed: 60, blocks: [{ recorded: 60, runs: 1, outcome: 'pending' }] } };
  const next = mergeLiveState(state, live);
  assert.equal(next.catalog, state.catalog);
  assert.equal(next.playerLevels, state.playerLevels);
  assert.equal(next.plan.blocks[0].scenario, scenario);
  assert.equal(next.plan.blocks[0].playCount, 2);
  assert.equal(next.plan.blocks[0].runs, 1);
  assert.equal(next.plan.status, 'paused');
});

test('a stale clock response cannot overwrite a newly generated plan', () => {
  const state = { plan: { id: 'new-session', blocks: [] } };
  assert.equal(mergeLiveState(state, { plan: { id: 'old-session', blocks: [] } }), state);
  assert.equal(mergeLiveState(state, { plan: null }), state);
});


test('shared fixed observation displays original main time once and only added runs as assessment', () => {
  const { planTimeAllocation } = sandbox.exports;
  const blocks = [{ role: 'practice', budget: 240, playCount: 4, timing: { seconds: 60 }, assessment: { mainPlayCount: 2, extraRuns: 2 } }, { role: 'explore', budget: 120 }];
  const time = planTimeAllocation(blocks);
  assert.equal(time.main, 120);
  assert.equal(time.assessment, 120);
  assert.equal(time.explore, 120);
  assert.equal(Object.values(time).reduce((a,b)=>a+b,0), 360);
});

test('legacy separate tests and passive main observations retain their actual time budgets', () => {
  const time = sandbox.exports.planTimeAllocation([{role:'assessment',budget:180}, {role:'practice',budget:240,playCount:4,assessment:{extraRuns:0}}, {role:'challenge',budget:60}]);
  assert.equal(time.assessment,180);
  assert.equal(time.main,240);
  assert.equal(time.challenge,60);
});
