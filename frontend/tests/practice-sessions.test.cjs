const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const ts=require('typescript');
const vm=require('node:vm');
function load(path,dependencies={}) {
 const sandbox={exports:{},require:name=>dependencies[name]??{},Date,Set,Map};
 vm.runInNewContext(ts.transpileModule(fs.readFileSync(path,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020,jsx:ts.JsxEmit.ReactJSX}}).outputText,sandbox);
 return sandbox.exports;
}
const {materializeSessions}=load('src/shared/lib/practiceSessions.ts');
const record=(id)=>({filePath:id,stats:{summary:{datePlayed:'2026-10-05T12:00:00Z'}}});
test('merged old training intervals retain every name and note, with explicit new edits taking precedence',()=>{
 const group={id:'session-a',startedAt:1000,endedAt:61000,runIds:['a','b'],legacyIds:['session-100','session-200']};
 const old={'session-100':{name:'上午',notes:'第一段'},'session-200':{name:'加练',notes:'第二段'}};
 let s=materializeSessions([record('a'),record('b')],[group],old)[0];
 assert.equal(s.name,'上午 / 加练');assert.equal(s.notes,'第一段\n\n第二段');assert.equal(Date.parse(s.end)-Date.parse(s.start),60000);
 assert.equal(s.items[0].filePath,'b');
 s=materializeSessions([record('a'),record('b')],[group],{...old,'session-a':{name:'统一名称',notes:'',mergedAliases:group.legacyIds}})[0];
 assert.equal(s.name,'统一名称');assert.equal(s.notes,'');
});
const {completedPractice,trainingStatusLabel}=load('src/features/training/progress.ts');
test('skipped, timed-out and generated-only practice never inflate completion',()=>{
 for(const outcome of ['pending','skipped','missed','session_limit'])assert.equal(completedPractice({runs:0,recorded:0,playCount:2,outcome},'playlist'),false);
 assert.equal(completedPractice({runs:1,recorded:60,playCount:2,outcome:'skipped'},'playlist'),false);
 assert.equal(completedPractice({runs:2,recorded:120,playCount:2,outcome:'list_complete'},'playlist'),true);
 assert.match(trainingStatusLabel({status:'completed',endReason:'time_budget'}),/预算/);
 assert.match(trainingStatusLabel({status:'completed',endReason:'manual'}),/主动/);
});

test('changing a gap also retains notes saved under newer canonical interval IDs',()=>{
 const group={id:'session-a',startedAt:1000,endedAt:61000,runIds:['a','b'],legacyIds:['session-a','session-b']};
 const notes={'session-a':{name:'一',notes:'一段',mergedAliases:['session-a']},'session-b':{name:'二',notes:'二段',mergedAliases:['session-b']}};
 const s=materializeSessions([record('a'),record('b')],[group],notes)[0];
 assert.equal(s.name,'一 / 二');assert.equal(s.notes,'一段\n\n二段');
});

test('overview renders actual completion, separate durations, end reason and exact plan history link',()=>{
 const React=require('react');
 const {renderToStaticMarkup}=require('react-dom/server');
 const progress=load('src/features/training/progress.ts');
 const {TrainingPlanProgress}=load('src/features/overview/components/sessionWidgets/TrainingPlanProgress.tsx',{
  'react/jsx-runtime':require('react/jsx-runtime'),
  'react-router-dom':{Link:({to,children,...props})=>React.createElement('a',{href:to,...props},children)},
  '@/shared/components':{Widget:({title,children})=>React.createElement('section',null,title,children)},
  '@/features/training/progress':progress,
 });
 const plan={id:'plan/a',status:'completed',endReason:'time_budget',completedBlocks:1,blockCount:4,runs:3,targetRuns:8,elapsed:600,minutes:10,recorded:180};
 const html=renderToStaticMarkup(React.createElement(TrainingPlanProgress,{plan,stale:true}));
 assert.match(html,/时间预算已到/);
 assert.match(html,/aria-valuenow="25"/);
 assert.match(html,/3 \/ 8/);
 assert.match(html,/10:00 \/ 10:00/);
 assert.match(html,/已记录练习时长<\/dt><dd>3:00/);
 assert.match(html,/href="\/history\?plan=plan%2Fa"/);
 assert.match(html,/进度暂未刷新/);
});

test('pathless history records use the same stable IDs as plan attribution',()=>{
 const r={fileName:'run.csv',stats:{summary:{datePlayed:'2026-10-05T12:00:00Z',scenario:'map'}}};
 const {runId}=load('src/shared/lib/practiceSessions.ts');
 const {buildHistoryRuns}=load('src/features/history/lib/historyModels.ts',{
  '@/shared/lib/practiceSessions':{runId},
  '@/shared/lib':{getScenarioName:r=>r.stats.summary.scenario,readRunTimestamp:()=>0,readRunScore:()=>0,readRunAccuracy:()=>0,readRunDurationMs:()=>0},
 });
 const runs=buildHistoryRuns([{id:'session-a',items:[r]}]);
 assert.equal(runs[0].id,runId(r));
});
