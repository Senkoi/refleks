const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const ts = require('typescript');
const vm = require('node:vm');
const output = ts.transpileModule(fs.readFileSync('src/features/training/catalog.ts', 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
const sandbox = { exports: {} };
vm.runInNewContext(output, sandbox);
const { filterCatalog, catalogPage } = sandbox.exports;
const rows = [
  { name: 'verified precision', evaluation: { fileStatus: 'verified', hasDifficultyEvidence: true, hasPrecisionReference: true } },
  { name: 'verified benchmark', evaluation: { fileStatus: 'verified', hasDifficultyEvidence: true, hasBenchmarkReference: true } },
  { name: 'manual advanced', difficulty: 'advanced', difficultySource: 'manual', evaluation: { fileStatus: 'verified', difficultyStatus: 'unfitted', hasDifficultyEvidence: false } },
  { name: 'parsed incomplete', evaluation: { fileStatus: 'parsed', hasDifficultyEvidence: false } },
  { name: 'conflict', evaluation: { fileStatus: 'conflict', hasDifficultyEvidence: false } },
  { name: 'deleted', evaluation: { fileStatus: 'missing', hasDifficultyEvidence: false } },
];
const names = xs => Array.from(xs, x => x.name);
test('downloaded and assessed intersection excludes guesses, unresolved files and stale downloads', () => {
  assert.deepEqual(names(filterCatalog(rows, '', 'verified', 'evidence')), ['verified precision', 'verified benchmark']);
  assert.deepEqual(names(filterCatalog(rows, 'BENCHMARK', 'verified', 'benchmark')), ['verified benchmark']);
  assert.equal(filterCatalog(rows, '', 'all', 'calibrated').length, 0);
  assert.deepEqual(names(filterCatalog(rows, '', 'issues', 'all')), ['parsed incomplete', 'conflict']);
});
test('all catalog entries remain reachable after the first hundred and after filtering shrinks a page', () => {
  const many = Array.from({ length: 251 }, (_, i) => ({ name: String(i) }));
  const found = [0, 1, 2].flatMap(p => Array.from(catalogPage(many, p).rows));
  assert.deepEqual(found.map(x => x.name), many.map(x => x.name));
  assert.equal(catalogPage(rows, 2).page, 0);
  assert.equal(catalogPage([], 9).pages, 1);
});
