import { Link } from "react-router-dom";
import { Widget } from "@/shared/components";
import { ScenarioHistoryLink } from "@/shared/components/ScenarioHistoryLink";
import { useTrainingProgress } from "./TrainingProgressProvider";
import { trainingStatusLabel } from "./progress";
import { ExplorationDetails } from "./ExplorationDetails";
import type { TrainingGuidance } from "./contracts.generated";

const themes:Record<string,string>={static:"静态点击",dynamic:"动态点击",smooth:"精确追踪",reactive:"反应追踪",switching_speed:"速度切换",switching_evasive:"规避切换"};
const roles:Record<string,string>={warmup:"热身",practice:"专项练习",benchmark:"基准测量",assessment:"复测",explore:"探索试练",challenge:"进阶挑战"};

export function TrainingAdviceContent({guidance}:{guidance:TrainingGuidance}) {
 const current=guidance.mode==="current";
 return <>
  <div className="flex flex-wrap items-center justify-between gap-2"><strong>{guidance.theme?themes[guidance.theme]??guidance.theme:"训练安排"}{current&&guidance.status&&<span className="ml-2 text-xs font-normal text-surface-muted-foreground">{trainingStatusLabel({status:guidance.status})}</span>}</strong><Link to="/training" className="text-xs text-primary hover:underline">{current?"打开本次训练":"到工作台生成列表"}</Link></div>
  <p className="mt-2 text-xs text-surface-muted-foreground">{guidance.reason}</p>
  {(guidance.items??[]).length>0&&<ol className="mt-3 grid gap-2 sm:grid-cols-2">{guidance.items.map((item,i)=><li key={`${i}-${item.scenario}`} className="min-w-0 rounded border border-border p-3"><div className="flex items-start justify-between gap-2"><ScenarioHistoryLink name={item.scenario}/><span className="shrink-0 text-xs text-surface-muted-foreground">{roles[item.role]??item.role} · {item.runs} 局{current?"剩余":""}</span></div><p className="mt-1 truncate text-[11px] text-surface-muted-foreground" title={item.origin}>{item.origin}</p></li>)}</ol>}
  {guidance.exploration&&<div className="mt-3"><ExplorationDetails report={guidance.exploration}/></div>}
  {guidance.mode==="next"&&<p className="mt-2 text-[11px] text-surface-muted-foreground">这是下次训练的方向预览，生成后再按固定列表练习。</p>}
 </>;
}

export function TrainingAdviceWidget() {
 const {guidance,guidanceError,error,loading}=useTrainingProgress();
 const title=guidance?.mode==="current"?"本次训练安排":"下一步训练建议";
 return <Widget title={title}>
  {(guidanceError||error)&&<p role="alert" className="mb-2 text-xs text-destructive">训练建议暂未刷新，请在工作台查看。</p>}
  {guidance?<TrainingAdviceContent guidance={guidance}/>:<p role="status" className="text-xs text-surface-muted-foreground">{loading?"正在读取训练安排…":"暂时无法读取训练建议。"}</p>}
 </Widget>;
}
