import type { Block, PlanHistorySummary } from "./contracts.generated";
export const trainingStatuses:Record<string,string>={draft:"待开始",running:"训练中",ready:"下一练习待开始",paused:"已暂停",waiting:"练习到时 · 等待本局结束",completed:"已结束"};
export const trainingEndReasons:Record<string,string>={plan_complete:"计划已练完",items_processed:"练习项已处理 · 有未完成内容",time_budget:"时间预算已到",manual:"主动结束",legacy_unknown:"已结束 · 旧记录未注明原因"};
export const trainingStatusLabel=(p:{status:string;endReason?:string})=>p.status==="completed"?trainingEndReasons[p.endReason??"legacy_unknown"]??"已结束":trainingStatuses[p.status]??p.status;
export function completedPractice(b:Block,mode:string){
 if(b.runs<=0 || b.recorded<=0)return false;
 return mode==="playlist"?b.runs>=Math.max(1,b.playCount):["threshold","measured","time_limit","list_complete"].includes(b.outcome);
}
export const progressRatio=(p:PlanHistorySummary)=>p.blockCount>0?Math.min(1,p.completedBlocks/p.blockCount):0;
export const trainingClock=(seconds:number)=>{const s=Math.max(0,Math.floor(seconds));return `${Math.floor(s/60)}:${String(s%60).padStart(2,"0")}`;};
