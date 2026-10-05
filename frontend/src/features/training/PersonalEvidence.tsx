import { memo } from "react";
import TrainingHelp from "./TrainingHelp";
import AnchorTrend from "./AnchorTrend";
import { ScenarioHistoryLink } from "@/shared/components/ScenarioHistoryLink";
import type { AnchorEvaluation, PersonalAnchor, TrainingStudy } from "./api";

const anchorStatuses: Record<string,string> = { stable: "跨日稳定", provisional: "暂定", variable: "波动较大", stale: "需要更新" };
const studyStatuses: Record<string,string> = { planned: "等待执行", observed: "两局观察完成", waiting: "早期记录：等待隔日复测", due: "早期记录：复测已到期", transfer_due: "等待迁移复测", measured: "早期记录：复测完成", expired: "早期记录：复测时间已过", incomplete: "试练记录不足或不可比", baseline_missing: "缺少可比前测", invalidated: "场景版本变化", settings_changed: "测量设置变化" };
const evaluationStatuses: Record<string,string> = { planned: "等待固定观察", observed: "已记录，等待可比观测", compared: "可比变化已记录", incomparable: "记录不足或条件不可比", not_started: "未开始 · 已替换" };
const date = (v: number) => new Date(v).toLocaleString("zh-CN");
studyStatuses.not_started = "未开始 · 已替换";
const change = (v: number) => (v >= 0 ? "+" : "") + (v*100).toFixed(1) + "%";
function observationHelp(reason: string) {
 if (reason.includes("波动")) return "最近这张图的成绩起伏较大，回到熟悉的图练几局，帮助确认你当前通常能达到的水平。";
 if (reason.includes("长期未练")) return "你有一段时间没练这张图了，这次会补充成绩，让训练参考更贴近现在的表现。";
 if (reason.includes("持续下降")) return "这张图在几个训练日里持续下降，应用安排一次固定观察，帮助确认这次变化。一次低分不会立即降低训练档位。";
 if (reason.includes("探索")) return "你已在不同训练日尝试过一些新练习。这次回到熟悉的图，看当前表现是否发生变化。";
 return "本次列表里已经有这张熟悉的图，我会顺便记录连续完整对局，帮助更新你的训练参考。";
}

export default memo(function PersonalEvidence({ anchors, studies, evaluations, busy, onFeedback }: {
 anchors: PersonalAnchor[]; studies: TrainingStudy[]; evaluations: AnchorEvaluation[]; busy: boolean; onFeedback: (id:string,value:string)=>void;
}) {
 const recent = [...studies].sort((a,b)=>b.createdAt-a.createdAt).slice(0,12);
 const known = [...anchors].sort((a,b)=>b.lastPlayed-a.lastPlayed).slice(0,30);
 return <>
  <section className="training-card"><h2>个人参照成绩 <TrainingHelp label="个人参照成绩"><p>我把你在熟悉图上的近期通常表现叫作“参照成绩”，用它来挑相近的练习。偶尔一局高分不会把参照成绩直接拉高喵。</p><p>点击地图名称可查看训练历史和近期参照成绩折线。参照成绩折线上的每个点代表一个训练时段的中间成绩（中位数），虚线是当前基准。我取近 45 天最近 6 个训练时段，每次最多 12 局，再取这些中间成绩的中位数，减少偶然高分或低分的影响。</p><p>“暂定”表示记录还少；“波动较大”表示近期表现不够一致；“跨日稳定”需要至少 3 局、跨 2 天，且成绩波动较小。超过 14 天未练会提示“需要更新”。继续正常练习，新成绩就会自动更新参照成绩。</p><p>更换场景版本、灵敏度、视野或单局时长后，会分开记录，避免把不同条件的分数混在一起。“历史未绑定版本”表示旧记录无法确认使用了哪个场景版本。</p></TrainingHelp></h2>
   {!known.length ? <p className="training-muted">先练几局，我会帮你建立参照成绩喵</p> :
   <div className="training-anchor-grid">{known.map(a=><article className="training-anchor-card" key={a.scenario}>
    <div className="training-section-title"><h3><ScenarioHistoryLink name={a.scenario} known supplementary={<AnchorTrend anchor={a} />} /></h3><span className="training-badge">{anchorStatuses[a.status] ?? a.status}</span></div>
    <div className="training-anchor-stats"><strong>{a.medianScore.toFixed(1)} <small>当前基准</small></strong><span>波动 {a.scoreMAD.toFixed(1)}</span>{a.hitsPerSecond != null && <span>{a.hitsPerSecond.toFixed(2)} 命中/秒</span>}{a.accuracy != null && <span>{(a.accuracy*100).toFixed(1)}% 命中率</span>}</div>
    <div className="training-anchor-meta"><span>{a.samples} 局 · {a.sessions} 次练习 · {a.days} 天</span><span>{a.evidence === "local_execution_context" ? "已确认场景版本" : "历史未绑定版本"}</span></div>
   </article>)}</div>}
  </section>
  <section className="training-card"><h2>日常试练与低频观察 <TrainingHelp label="试练与观察"><p>我们用两局尝尝新练习，看看它对你偏易、合适还是偏难。练完告诉我感受，我下次就更会挑啦喵。</p><p>我会偶尔带你回到熟悉的图，先热身一局，再记下三局成绩。能用本次列表里的练习，我就尽量不额外占你的时间。</p><p>每份计划最多观察一张熟悉图；需要额外补局时，同类每周最多一次、合计最多两次，补充时间不超过 3 分钟，也不超过本次总时间的 10%。时间不够就留待以后，无需为了赶期限补测。</p><p>我会帮你找变化，但只练一次就拿了高分，还不能确定是哪张图带来的进步喵。</p></TrainingHelp></h2><div className="training-metrics"><div><strong>2 局</strong><span>日常试练</span></div><div><strong>1 + 3 局</strong><span>熟悉 + 固定测量</span></div><div><strong>≤ 2 次/周</strong><span>新增固定观察</span></div></div></section>
  {[...evaluations].sort((a,b)=>b.createdAt-a.createdAt).slice(0,12).map(e=><section className="training-card" key={e.id}>
   <div className="training-section-title"><div className="training-inline-title"><h3><ScenarioHistoryLink name={e.scenario} known={!!e.result || !!e.previous} /></h3><TrainingHelp label={`${e.scenario}观察依据`}><p>{observationHelp(e.reason)}</p><p>按列表次数练完即可，不必刷到高分。下面会对照同图、相同条件的记录；条件不同或记录不足时，会提示暂时无法比较。</p><p>“期间已记录练习”只包含我读到的对局。状态和其他练习也可能影响成绩，因此变化不直接算作某一张试练图的效果。</p></TrainingHelp></div><span className="training-badge">{evaluationStatuses[e.status] ?? e.status}</span></div>
   <p>补充 {e.extraRuns} 局 · {(e.extraSeconds/60).toFixed(1)} 分钟</p>
   {e.result && <p>中位成绩 {e.result.score.toFixed(1)} · 三局分布：{e.result.scores?.map(v=>v.toFixed(1)).join(" / ") ?? "旧记录未保存"}</p>}
   {e.previous && <p>同图{e.intervalKind === "same_day" ? "当日" : e.intervalKind === "longer_interval" ? "较长间隔" : "跨日"}表现变化：{e.change == null ? "基准为零，保留原始成绩" : change(e.change)} · 实际间隔 {e.intervalHours?.toFixed(1)} 小时 · {e.comparableDays} 个可比训练日</p>}
   {e.previous && <p className="training-muted">期间已记录练习 {(e.exposure.recordedSeconds/60).toFixed(1)} 分钟，同图 {(e.exposure.sameSceneSeconds/60).toFixed(1)} 分钟，同类 {(e.exposure.sameThemeSeconds/60).toFixed(1)} 分钟；试练 {(e.exposure.trialSeconds/60).toFixed(1)} 分钟{!!e.exposure.trialScenarios?.length && <>（{e.exposure.trialScenarios.map((name,i) => <span key={name}>{i > 0 && "、"}<ScenarioHistoryLink name={name} known /></span>)}）</>}。</p>}
  </section>)}
  {!recent.length && <section className="training-card"><p>暂无试练记录 <TrainingHelp label="试练条件">先按主线列表正常练习，积累熟悉图的成绩。应用找到玩法相近、可以确认目标大小差异的新图后，会在探索时间内安排试练。你不需要另行上传场景文件；探索比例为 0 时不会安排额外试练。</TrainingHelp></p></section>}
  {recent.map(st=><section className="training-card" key={st.id}>
   <div className="training-section-title"><div className="training-inline-title"><h3><ScenarioHistoryLink name={st.trainingScenario} known={!!st.trial} /></h3><TrainingHelp label={`${st.trainingScenario}试练`}><p>{st.protocolId === "daily_trial_v2" ? "参照成绩来自你熟悉的图，试练成绩来自这张新图。两张图的分数不能直接比较高低来判断进步；这次主要看看新图是否适合你，不会要求额外复测。" : !st.transferScenario ? "这是一条早期试练记录，会比较熟悉图在试练前后的表现。它没有安排另一张图，因此不提供其他场景的表现变化。" : "这是一条早期试练记录。除了熟悉图，还安排了另一张测试图，看看变化能否体现在其他场景；若两次测试之间单独练过那张图，就不作这项判断。"}</p><p>练完可以告诉我“偏易、合适、偏难”，让我下次挑得更准。懒得填也没关系，接着练就好喵。</p></TrainingHelp></div><span className="training-badge">{studyStatuses[st.status] ?? st.status}</span></div>
   <p>参照场景：<ScenarioHistoryLink name={st.anchorScenario} known={!!st.baseline} />{st.transferScenario && <><br/>迁移保留图：<ScenarioHistoryLink name={st.transferScenario} /></>}</p>
   <p>{st.protocolId === "daily_trial_v2" ? "熟悉图参考成绩" : "试练前成绩"} {st.baseline?.score.toFixed(1) ?? "待记录"} · 试练 {st.trial?.score.toFixed(1) ?? "待记录"}{st.protocolId !== "daily_trial_v2" && <> · 复测 {st.retest?.score.toFixed(1) ?? "待记录"}</>}</p>
   {st.retentionChange != null && <p>同图保持变化：{change(st.retentionChange)}</p>}
   {st.transferChange != null && <p>保留图迁移变化：{change(st.transferChange)}</p>}
   {st.transferContaminated && <p className="training-muted">保留图在两次测量之间被单独练习，已取消本次迁移判断。</p>}
   {st.dueAt != null && <p className="training-muted">复测窗口：{date(st.dueAt)} ～ {date(st.expiresAt ?? st.dueAt)}</p>}
   {st.trial && <div className="training-actions"><span>可选反馈</span>{[["easy","偏易"],["suitable","合适"],["hard","偏难"]].map(([value,label])=><button key={value} aria-pressed={st.feedback===value} disabled={busy} onClick={()=>onFeedback(st.id,value)}>{st.feedback===value ? "✓ " : ""}{label}</button>)}</div>}
  </section>)}
 </>;
});
