const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const ts = require('typescript');
const vm = require('node:vm');
const React = require('react');
const { renderToStaticMarkup } = require('react-dom/server');

function renderPanel(name, props, states) {
  let index = 0;
  const hooks = {
    ...React,
    useState: initial => [index < states.length ? states[index++] : initial, () => {}],
    // Opening first renders the workbench summary before the async request resolves.
    useEffect: () => {},
  };
  const source = fs.readFileSync('src/features/training/Requirements.tsx', 'utf8');
  const output = ts.transpileModule(source, { compilerOptions: {
    module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020,
    jsx: ts.JsxEmit.ReactJSX,
  } }).outputText;
  const sandbox = { exports: {}, require: path => {
    if (path === 'react') return hooks;
    if (path === './api') return {};
    if (path === './TrainingHelp') return { default: ({ children }) => React.createElement('span', null, children) };
    return require(path);
  } };
  vm.runInNewContext(output, sandbox);
  return renderToStaticMarkup(React.createElement(sandbox.exports[name], props));
}

function scenario(requirements) {
  return { name: 'Fixture', localAssessment: { requirements } };
}

test('opening requirements renders the lazy summary when Go sends null collections', () => {
  const summary = { fileSHA256: 'v1', targets: [{ bot: 'target' }], helpers: null, axes: null, slots: null, scoringMin: null, scoringMax: null };
  const html = renderPanel('RequirementDetails', { scenario: scenario(summary) }, [true, null, '']);
  assert.match(html, /计分候选角色 1 种 · 辅助角色 0 种 · 计分槽位 未知/);
  assert.match(html, /正在读取此版本的详细需求/);
});

test('loaded requirements render empty axes and unknown values without losing real facts', () => {
  const detail = { fileSHA256: 'v1', targets: null, helpers: null, scoringMin: 0, scoringMax: 0,
    axes: [{ key: 'precision', facts: null }, { key: 'kill_and_lifetime_windows', facts: [
      { key: 'ideal_body_hits_to_kill', value: null, status: 'unknown' },
      { key: 'DamagePerShot', value: 0, status: 'declared', unit: '' },
    ] }], map: { counts: null },
  };
  const html = renderPanel('RequirementDetails', { scenario: scenario(detail) }, [true, detail, '']);
  assert.match(html, /精度/);
  assert.match(html, /理想身体命中次数/);
  assert.match(html, /未知/);
  assert.match(html, /每发伤害：<strong>0/);
  assert.match(html, /计分槽位 0/);
});

test('a failed detail request remains a readable panel instead of crashing the page', () => {
  const summary = { fileSHA256: 'v1', targets: null, helpers: null, axes: null };
  const html = renderPanel('RequirementDetails', { scenario: scenario(summary) }, [true, null, '请重新扫描或刷新关卡库']);
  assert.match(html, /role="alert"/);
  assert.match(html, /请重新扫描或刷新关卡库/);
  assert.doesNotMatch(html, /正在读取/);
});

test('an incompatible comparison renders null result collections from Go', () => {
  const result = { result: { kind: 'incompatible', direction: 'uncertain', axes: null, unknown: null, changedMechanisms: null, neighborDistance: null } };
  const html = renderPanel('RequirementCompare', { catalog: [] }, [true, 'A', 'B', result, '', false]);
  assert.match(html, /任务不兼容/);
  assert.match(html, /无法给出整体方向/);
});
