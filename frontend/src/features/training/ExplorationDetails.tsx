import { CircleHelp } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/components/ui/popover";
import type { ExplorationReport } from "./contracts.generated";

const reasons:Record<string,string>={
 practice_evidence:"相关练习尚不足三局可比完整记录",goal_evidence:"同目标的依据不足",audience_tier:"目标档位超出当前范围",feedback:"你的反馈暂不适合",declining:"近期表现持续下降",difficulty:"当前难度偏高",benchmark_alignment:"基准分类或难度不匹配",whole_run_budget:"剩余额度容不下完整试练",budget_or_priority:"共用预算已优先分配给其他变式",
};
export function ExplorationDetails({report}:{report:ExplorationReport}) {
 const selected=report.selected??[];
 const label=report.status==="disabled"?"探索已关闭":selected.length?`本次安排 ${selected.length} 个变式`:report.status==="budget_limited"?"探索暂未加入 · 整局预算不足":"探索暂未加入";
 return <div className="flex items-center gap-2 text-xs text-surface-muted-foreground">
  <span>{label}</span>
  <Popover><PopoverTrigger asChild><button type="button" className="inline-flex rounded p-1 text-surface-muted-foreground hover:text-primary" aria-label="探索安排说明"><CircleHelp size={15}/></button></PopoverTrigger>
   <PopoverContent className="z-50 max-h-80 w-80 overflow-auto text-xs" side="bottom" align="start">
    <p>相关主线有三局可比完整记录后，可以尝试同目标的新场景。普通探索与同家族试练共用额度；尝试结果用于下一份列表。</p>
    <p className="mt-2">探索额度 {Math.floor(report.limitSeconds/60)} 分 {report.limitSeconds%60} 秒 · 已安排 {report.usedSeconds} 秒</p>
    {selected.length>0&&<p className="mt-2">变式：{selected.join("、")}</p>}
    {(report.reasons??[]).length>0&&<ul className="mt-2 space-y-2">{report.reasons.map(r=><li key={r.code}>{reasons[r.code]??"暂未满足安排条件"}（{r.count} 项）{(r.scenes??[]).length>0&&<span className="block opacity-75">{r.scenes.join("、")}</span>}</li>)}</ul>}
    {report.status==="disabled"&&<p className="mt-2">可以在工作台开启同目标探索，下次生成时生效。</p>}
   </PopoverContent>
  </Popover>
 </div>;
}
