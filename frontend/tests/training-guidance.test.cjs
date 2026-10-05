const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const ts=require('typescript');
const vm=require('node:vm');
const React=require('react');
const {renderToStaticMarkup}=require('react-dom/server');
function load(path,dependencies={},globals={}) {
 const sandbox={exports:{},require:name=>dependencies[name]??require(name),...globals};
 vm.runInNewContext(ts.transpileModule(fs.readFileSync(path,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020,jsx:ts.JsxEmit.ReactJSX}}).outputText,sandbox);
 return sandbox.exports;
}
const widget=({title,children})=>React.createElement('section',null,title,children);
const link=({to,children,...props})=>React.createElement('a',{href:to,...props},children);
const {trainingStatusLabel}=load('src/features/training/progress.ts');
const exploration=load('src/features/training/ExplorationDetails.tsx',{
 '@/shared/components/ui/popover':{Popover:({children})=>React.createElement('div',null,children),PopoverTrigger:({children})=>children,PopoverContent:({children})=>React.createElement('aside',null,children)},
});
const current={mode:'current',planId:'fixed-plan',status:'paused',theme:'static',reason:'继续当前固定列表',items:[{scenario:'Workbench B',role:'practice',runs:2,origin:'作者主线'}]};
function advice(state) {
 return load('src/features/training/TrainingAdviceWidget.tsx',{
  'react-router-dom':{Link:link},'@/shared/components':{Widget:widget},
  '@/shared/components/ScenarioHistoryLink':{ScenarioHistoryLink:({name})=>React.createElement('button',null,name)},
  './TrainingProgressProvider':{useTrainingProgress:()=>state},'./progress':{trainingStatusLabel},'./ExplorationDetails':exploration,
 });
}
test('overview displays the fixed workbench plan instead of independently recommending benchmark scenes',()=>{
 const training=advice({guidance:current,loading:false,error:'',guidanceError:''});
 const overview=load('src/features/overview/components/BenchmarkOverviewWidget.tsx',{
  'react-router-dom':{Link:link},'@/features/training/TrainingAdviceWidget':training,
  '@/features/benchmarks/hooks/useBenchmarkDetailProgress':{useBenchmarkDetailProgress:()=>({progress:{overallRank:1,ranks:[{name:'Gold'}],categories:[{scenarios:[{name:'Unrelated benchmark A'}]}]},difficultyIndex:0})},
  '@/shared/hooks':{useBenchmarks:()=>({selectedBenchmark:'Voltaic S5',getBenchmarkByName:()=>({benchmarkName:'Voltaic S5',difficulties:[{difficultyName:'Novice'}]})})},
  '@/shared/lib/navigation':{benchmarkPath:()=>'/benchmarks/s5'},
 });
 const html=renderToStaticMarkup(React.createElement(overview.BenchmarkOverviewWidget));
 assert.match(html,/本次训练安排/);assert.match(html,/Workbench B/);assert.match(html,/已暂停/);
 assert.doesNotMatch(html,/Unrelated benchmark A/);
 assert.match(html,/href="\/training"/);assert.match(html,/基准成绩：Voltaic S5/);assert.match(html,/Gold/);
});
test('next advice is explicitly a preview and tolerates null collections',()=>{
 const {TrainingAdviceContent}=advice({});
 const html=renderToStaticMarkup(React.createElement(TrainingAdviceContent,{guidance:{mode:'next',theme:'smooth',reason:'近期覆盖较少',items:null,exploration:{status:'no_candidates',selected:null,reasons:null,limitSeconds:162,usedSeconds:0}}}));
 assert.match(html,/方向预览/);assert.match(html,/到工作台生成列表/);assert.match(html,/探索暂未加入/);
 assert.doesNotMatch(html,/启动场景/);
});
test('exploration explanation renders whole-run and evidence exclusions in user language',()=>{
 const html=renderToStaticMarkup(React.createElement(exploration.ExplorationDetails,{report:{status:'budget_limited',selected:[],limitSeconds:27,usedSeconds:0,reasons:[{code:'whole_run_budget',count:1,scenes:['Long scene']},{code:'practice_evidence',count:1,scenes:['Foundation']}]}}));
 assert.match(html,/整局预算不足/);assert.match(html,/剩余额度容不下完整试练/);assert.match(html,/三局可比完整记录/);assert.match(html,/Long scene/);
});
test('active plan polling stays lightweight and an in-flight evidence event refreshes next guidance',async()=>{
 let state,cleanup,evidenceEvent;
 let progress={current:{status:'running'},guidance:current,recentPlans:[]};
 let guidanceReads=0,deferred;
 const timers=new Map();let nextTimer=0;
 const listeners=new Map();
 const globals={Date,Event:class{},window:{runtime:{EventsOnMultiple:true},addEventListener:(name,fn)=>listeners.set(name,fn),removeEventListener:name=>listeners.delete(name),dispatchEvent:()=>{}},setTimeout:fn=>{const id=++nextTimer;timers.set(id,fn);return id;},clearTimeout:id=>timers.delete(id)};
 const hooks={...React,useState:initial=>{state=initial;return [state,update=>{state=typeof update==='function'?update(state):update;}];},useEffect:fn=>{cleanup=fn();}};
 const provider=load('src/features/training/TrainingProgressProvider.tsx',{
  'react':hooks,'@wails/runtime':{EventsOn:(name,fn)=>{if(name==='benchmark:progress:updated')evidenceEvent=fn;return ()=>{};}},
  './api':{readTrainingProgress:async()=>progress,readTrainingGuidance:()=>{guidanceReads++;return deferred?deferred.promise:Promise.resolve({mode:'next',items:[],reason:'next'});}},
 },globals);
 provider.TrainingProgressProvider({children:null});
 const flush=async()=>{for(let i=0;i<8;i++)await Promise.resolve();};
 await flush();assert.equal(guidanceReads,0);assert.equal(state.guidance.planId,'fixed-plan');
 progress={current:null,recentPlans:[]};
 deferred={};deferred.promise=new Promise(resolve=>deferred.resolve=resolve);
 evidenceEvent();await flush();assert.equal(guidanceReads,1);
 evidenceEvent();deferred.resolve({mode:'next',items:[],reason:'older response'});deferred=null;await flush();
 const timer=[...timers.values()].at(-1);timer();await flush();
 assert.equal(guidanceReads,2);assert.equal(state.guidance.reason,'next');cleanup();
});
