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
