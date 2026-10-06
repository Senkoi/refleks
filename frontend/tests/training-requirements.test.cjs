const { test, after, beforeEach, afterEach } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const esbuild = require('esbuild');
const { JSDOM } = require('jsdom');

// Exercise the client renderer, real hooks and real help popovers; replace only
// the desktop bridge, including deferred replies and errors.
const dom = new JSDOM('<!doctype html><html><body></body></html>', {
  url: 'http://localhost/', pretendToBeVisual: true,
});
for (const key of ['window', 'document', 'HTMLElement', 'HTMLInputElement', 'HTMLButtonElement',
  'Element', 'Node', 'NodeFilter', 'MutationObserver', 'CustomEvent', 'Event', 'MouseEvent',
  'FocusEvent', 'getComputedStyle']) global[key] = dom.window[key];
global.requestAnimationFrame = dom.window.requestAnimationFrame.bind(dom.window);
global.cancelAnimationFrame = dom.window.cancelAnimationFrame.bind(dom.window);
global.IS_REACT_ACT_ENVIRONMENT = true;

const cache = path.resolve('node_modules/.cache');
fs.mkdirSync(cache, { recursive: true });
const output = fs.mkdtempSync(path.join(cache, 'requirements-test-'));
esbuild.buildSync({
  entryPoints: {
    panel: path.resolve('src/features/training/Requirements.tsx'),
    guide: path.resolve('src/features/training/requirementsGuide.ts'),
  },
  bundle: true, packages: 'external', platform: 'node', format: 'cjs', jsx: 'automatic',
  alias: { '@': path.resolve('src'), '@wails': path.resolve('wailsjs') },
  define: { 'import.meta.env.VITE_AIMMEOW_BUILD': JSON.stringify('test-build') },
  outdir: output, outExtension: { '.js': '.cjs' },
});
const React = require('react');
const { createRoot } = require('react-dom/client');
const { RequirementDetails, RequirementCompare } = require(path.join(output, 'panel.cjs'));
const { requirementGuide } = require(path.join(output, 'guide.cjs'));
const { act } = React;
let host, root, requests, comparisons;
const waitForToggle = () => new Promise(resolve => setTimeout(resolve, 10));

beforeEach(() => {
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
  requests = [];
  comparisons = [];
  window.go = { main: { App: {
    GetTrainingSceneRequirements: (name, hash) => new Promise((resolve, reject) => requests.push({ name, hash, resolve, reject })),
    CompareTrainingScenes: (a, b) => new Promise((resolve, reject) => comparisons.push({ a, b, resolve, reject })),
  } } };
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
after(() => {
  dom.window.close();
  fs.rmSync(output, { recursive: true, force: true });
});

function descriptor(values = {}) {
  return { fileSHA256: 'v1', targets: null, helpers: null, axes: null, slots: null,
    scoringMin: null, scoringMax: null, hazards: null, ...values };
}
function scenario(requirements = descriptor(), name = 'Fixture') {
  return { name, localAssessment: { status: 'file_parsed_model_unfitted', requirements } };
}
async function renderDetails(s) {
  await act(async () => root.render(React.createElement(React.StrictMode, null,
    React.createElement(RequirementDetails, { scenario: s }))));
}
async function click(element) {
  assert.ok(element, 'click target exists');
  await act(async () => { element.click(); await waitForToggle(); });
}
async function respond(request, value) {
  await act(async () => { request.resolve(value); });
}
function summary(text) {
  return [...host.querySelectorAll('summary')].find(element => element.textContent === text);
}

test('a real click loads nullable collections and shows six readable requirements before raw configuration', async () => {
  await renderDetails(scenario());
  assert.equal(requests.length, 0);
  await click(summary('查看场景训练要求'));
  assert.match(host.textContent, /正在读取此版本的训练要求/);
  assert.deepEqual([requests[0].name, requests[0].hash], ['Fixture', 'v1']);
  await respond(requests[0], descriptor({
    scoringMin: 0, scoringMax: 0,
    axes: [null, { key: 'precision', facts: null }, { key: 'kill_and_lifetime_windows', facts: [null,
      { key: 'ideal_body_hits_to_kill', value: null, status: 'unknown' },
      { key: 'DamagePerShot', value: 0, status: 'declared', sources: null, conditions: null },
    ] }], map: { counts: null },
  }));
  assert.deepEqual([...host.querySelectorAll('h4')].map(element => element.textContent),
    ['瞄准精度', '追踪控制', '切换选择', '击杀节奏', '失误代价', '特殊机制']);
  assert.match(host.textContent, /计分目标名额：0 个/);
  assert.doesNotMatch(host.textContent, /每发伤害/);
  await click(summary('查看详细配置与计算条件'));
  assert.match(host.textContent, /理想身体命中次数：未知/);
  assert.match(host.textContent, /每发伤害：0/);
  assert.doesNotMatch(host.textContent, /暂时无法展示/);
});

test('the actual help popover explains evidence labels and does not imply a personal measurement', async () => {
  await renderDetails(scenario());
  await click(summary('查看场景训练要求'));
  await respond(requests[0], descriptor());
  await click(host.querySelector('button[aria-label="场景训练要求说明"]'));
  const help = document.querySelector('[role="dialog"]');
  assert.ok(help);
  assert.match(help.textContent, /“条件推算”表示只在列出的条件成立时适用/);
  assert.match(help.textContent, /暂不换算总难度分，也不测量你的反应时间/);
  await click(help.querySelector('button[aria-label="关闭说明"]'));
});

test('a failed request gives a readable error and closing then opening retries the real request', async () => {
  await renderDetails(scenario());
  await click(summary('查看场景训练要求'));
  await act(async () => requests[0].reject(new Error('场景文件已变化')));
  assert.match(host.querySelector('[role="alert"]').textContent, /请刷新场景库后重试/);
  assert.match(host.textContent, /场景文件已变化/);
  assert.match(host.textContent, /界面构建：test-build/);
  assert.equal(host.querySelector('[role="status"]'), null);
  await click(summary('查看场景训练要求'));
  await click(summary('查看场景训练要求'));
  assert.equal(requests.length, 2);
  await respond(requests[1], descriptor());
  assert.equal(host.querySelector('[role="alert"]'), null);
  assert.equal(host.querySelectorAll('h4').length, 6);
});

test('a late response for another scenario cannot replace the currently selected scenario', async () => {
  await renderDetails(scenario(descriptor(), 'A'));
  await click(summary('查看场景训练要求'));
  await renderDetails(scenario(descriptor({ fileSHA256: 'v2' }), 'B'));
  assert.deepEqual(requests.map(request => [request.name, request.hash]), [['A', 'v1'], ['B', 'v2']]);
  await respond(requests[0], descriptor({ scoringMin: 99, scoringMax: 99 }));
  assert.match(host.textContent, /正在读取/);
  assert.doesNotMatch(host.textContent, /99 个/);
  await respond(requests[1], descriptor({ fileSHA256: 'v2', scoringMin: 2, scoringMax: 2 }));
  assert.match(host.textContent, /计分目标名额：2 个/);
});

test('changing the hash for the same scenario clears old content and rejects mismatched file versions', async () => {
  await renderDetails(scenario());
  await click(summary('查看场景训练要求'));
  await respond(requests[0], descriptor({ scoringMin: 7, scoringMax: 7 }));
  assert.match(host.textContent, /7 个/);
  await renderDetails(scenario(descriptor({ fileSHA256: 'v2' })));
  assert.match(host.textContent, /正在读取/);
  assert.doesNotMatch(host.textContent, /7 个/);
  await respond(requests[1], descriptor());
  assert.match(host.querySelector('[role="alert"]').textContent, /场景文件版本不一致/);
  assert.equal(host.querySelectorAll('h4').length, 0);
});

test('a request that completes after closing does not leak into the next opening', async () => {
  await renderDetails(scenario());
  await click(summary('查看场景训练要求'));
  await click(summary('查看场景训练要求'));
  await respond(requests[0], descriptor({ scoringMin: 99, scoringMax: 99 }));
  await click(summary('查看场景训练要求'));
  assert.match(host.textContent, /正在读取/);
  assert.doesNotMatch(host.textContent, /99 个/);
  await respond(requests[1], descriptor());
  assert.equal(host.querySelectorAll('h4').length, 6);
});

test('detailed motion and fact help tolerate missing envelopes and null list members', async () => {
  await renderDetails(scenario());
  await click(summary('查看场景训练要求'));
  await respond(requests[0], descriptor({
    targets: [null, { bot: 'Target', dodgeGate: 'candidate_on', facts: null, windows: null,
      abilities: null, motionModels: [null, { axis: 'LR', dwellMin: null, dwellMax: 0, midpointEnvelope: null }] }],
    helpers: [null], slots: [null, { reference: 'Target', candidates: null }],
    axes: [{ key: 'kill_and_lifetime_windows', facts: [{ key: 'self_decay_seconds', value: 2,
      status: 'calculated_under_explicit_conditions', unit: 'configured seconds', sources: [null],
      conditions: [null, 'No external damage'] }] }],
    hazards: [null, { character: 'Target', minSeconds: null, maxSeconds: 0 }],
  }));
  await click(summary('查看详细配置与计算条件'));
  assert.match(host.textContent, /左右间隔 未知–0.000/);
  assert.match(host.textContent, /速度峰值 未知/);
  await click(host.querySelector('button[aria-label="self_decay_seconds推算条件说明"]'));
  assert.match(document.querySelector('[role="dialog"]').textContent, /不是实测阶段长度/);
});

test('a real comparison click accepts null result collections from Go', async () => {
  await act(async () => root.render(React.createElement(RequirementCompare, {
    catalog: [scenario(descriptor(), 'A'), scenario(descriptor(), 'B')],
  })));
  await click(summary('比较两张图的需求'));
  for (const [label, value] of [['需求参考图', 'A'], ['需求候选图', 'B']]) {
    await act(async () => {
      const select = host.querySelector(`select[aria-label="${label}"]`);
      select.value = value;
      select.dispatchEvent(new Event('change', { bubbles: true }));
    });
  }
  await click([...host.querySelectorAll('button')].find(element => element.textContent === '比较'));
  assert.deepEqual([comparisons[0].a, comparisons[0].b], ['A', 'B']);
  await respond(comparisons[0], { result: { kind: 'incompatible', direction: 'uncertain',
    axes: null, unknown: null, changedMechanisms: null, neighborDistance: null } });
  assert.match(host.textContent, /任务不兼容/);
  assert.match(host.textContent, /无法给出整体方向/);
});

test('an unexpected configuration render error stays inside the panel and can be retried', async () => {
  let clicks = 0;
  await act(async () => root.render(React.createElement(React.Fragment, null,
    React.createElement('button', { onClick: () => clicks++ }, '工作台操作'),
    React.createElement(RequirementDetails, { scenario: scenario() }))));
  await click(summary('查看场景训练要求'));
  await respond(requests[0], descriptor({ targets: [{ bot: { invalid: true }, facts: null, windows: null }] }));
  const originalError = console.error;
  const errors = [];
  console.error = (...args) => errors.push(args.map(String).join(' '));
  try {
    await click(summary('查看详细配置与计算条件'));
  } finally {
    console.error = originalError;
  }
  assert.ok(errors.length, 'React reported the malformed child');
  assert.match(host.querySelector('[role="alert"]').textContent, /此场景的训练要求暂时无法展示/);
  assert.match(host.textContent, /Fixture.*界面构建：test-build/s);
  await click([...host.querySelectorAll('button')].find(element => element.textContent === '工作台操作'));
  assert.equal(clicks, 1);
  await click(summary('查看场景训练要求'));
  await click(summary('查看场景训练要求'));
  await respond(requests[1], descriptor());
  assert.equal(host.querySelector('[role="alert"]'), null);
  assert.equal(host.querySelectorAll('h4').length, 6);
});

test('the guide keeps partial target data unknown and distinguishes configuration from measured behavior', () => {
  const guide = requirementGuide(descriptor({ scoringMin: 0, scoringMax: 0,
    targets: [
      { dodgeGate: 'candidate_on', facts: [{ key: 'MainBBRadius', value: 2 }], windows: [{ key: 'ideal_body_hits_to_kill', value: 1 }] },
      { facts: null, windows: null },
    ],
  }));
  assert.equal(guide.find(item => item.key === 'precision').status, '待确认');
  assert.equal(guide.find(item => item.title === '追踪控制').status, '待确认');
  assert.equal(guide.find(item => item.title === '击杀节奏').status, '待确认');
  assert.equal(guide.find(item => item.title === '切换选择').status, '已读取配置');
  assert.match(guide.find(item => item.title === '切换选择').observations.join(''), /名额不等于同时可见的目标数/);
  assert.doesNotMatch(guide.find(item => item.title === '击杀节奏').observations.join(''), /身体命中 1 次/);
});

test('known rules yield conditional kill timing, while absent rules do not imply no mechanisms', () => {
  const guide = requirementGuide(descriptor({
    targets: [{ dodgeGate: 'off', facts: [{ key: 'MainBBRadius', value: 2 }],
      windows: [{ key: 'ideal_body_hits_to_kill', value: 3 }, { key: 'self_decay_seconds', value: 1.5 }] }],
    axes: [{ facts: [{ key: 'ScoreMultAccuracy', status: 'declared', text: 'true' }] }],
  }));
  assert.equal(guide.find(item => item.title === '瞄准精度').status, '已读取配置');
  const timing = guide.find(item => item.title === '击杀节奏');
  assert.equal(timing.status, '条件推算');
  assert.match(timing.observations.join(''), /身体命中 3 次/);
  assert.match(timing.observations.join(''), /不包含找目标、反应和微调耗时/);
  assert.equal(guide.find(item => item.title === '失误代价').status, '已读取配置');
  assert.match(guide.find(item => item.title === '特殊机制').observations.join(''), /缺少声明不代表机制不存在/);
});
