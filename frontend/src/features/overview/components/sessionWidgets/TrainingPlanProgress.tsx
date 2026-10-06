import { Link } from "react-router-dom";
import { Widget } from "@/shared/components";
import type { PlanHistorySummary } from "@/features/training/contracts.generated";
import { progressRatio,trainingClock,trainingStatusLabel } from "@/features/training/progress";

export function TrainingPlanProgress({plan,stale=false}:{plan:PlanHistorySummary;stale?:boolean}){
 return <Widget title="本次训练">
  <div className="flex items-center justify-between gap-2"><strong className="text-sm">{trainingStatusLabel(plan)}</strong><Link to="/training" className="text-xs text-primary hover:underline">打开工作台</Link></div>
  {stale&&<p role="status" className="mt-2 text-xs text-surface-muted-foreground">进度暂未刷新，可在工作台查看。</p>}
  <div className="mt-4 text-2xl font-semibold">{plan.completedBlocks}<span className="text-sm font-normal text-surface-muted-foreground"> / {plan.blockCount} 练习完成</span></div>
  <div className="my-3 h-2 overflow-hidden rounded bg-surface-muted-soft" role="progressbar" aria-label="计划练习完成率" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(progressRatio(plan)*100)}><div className="h-full bg-primary transition-all" style={{width:`${progressRatio(plan)*100}%`}}/></div>
  <dl className="space-y-1 text-xs text-surface-muted-foreground"><div className="flex justify-between"><dt>实际对局 / 目标局数</dt><dd>{plan.runs} / {plan.targetRuns}</dd></div><div className="flex justify-between"><dt>计时 / 时间预算</dt><dd>{trainingClock(plan.elapsed)} / {trainingClock(plan.minutes*60)}</dd></div><div className="flex justify-between"><dt>已记录练习时长</dt><dd>{trainingClock(plan.recorded)}</dd></div></dl>
  <Link to={`/history?plan=${encodeURIComponent(plan.id)}`} className="mt-3 inline-block text-xs text-primary hover:underline">查看本次对局</Link>
 </Widget>;
}
