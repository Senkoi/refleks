import { memo } from "react";
import type { AnchorEvaluation, PersonalAnchor, TrainingStudy } from "./api";

const anchorStatuses: Record<string,string> = { stable: "跨日稳定", provisional: "暂定", variable: "波动较大", stale: "需要更新" };
const studyStatuses: Record<string,string> = { planned: "等待执行", observed: "两局观察完成", waiting: "旧协议：等待隔日复测", due: "旧协议：复测已到期", transfer_due: "等待迁移复测", measured: "旧协议：复测完成", expired: "旧协议：复测窗口已过", incomplete: "试练记录不足或不可比", baseline_missing: "缺少可比前测", invalidated: "场景版本变化", settings_changed: "测量设置变化" };
const evaluationStatuses: Record<string,string> = { planned: "等待固定观察", observed: "已记录，等待可比观测", compared: "可比变化已记录", incomparable: "记录不足或条件不可比", not_started: "未开始 · 已替换" };
const date = (v: number) => new Date(v).toLocaleString("zh-CN");
studyStatuses.not_started = "未开始 · 已替换";
const change = (v: number) => (v >= 0 ? "+" : "") + (v*100).toFixed(1) + "%";

export default memo(function PersonalEvidence({ anchors, studies, evaluations, busy, onFeedback }: {
 anchors: PersonalAnchor[]; studies: TrainingStudy[]; evaluations: AnchorEvaluation[]; busy: boolean; onFeedback: (id:string,value:string)=>void;
}) {
 const recent = [...studies].sort((a,b)=>b.createdAt-a.createdAt).slice(0,12);
 const known = [...anchors].sort((a,b)=>b.lastPlayed-a.lastPlayed).slice(0,30);
 return <>
  <section className="training-card"><h2>个人场景锚点</h2><p>同图、同版本与设置的训练块分别汇总，再取中位成绩。波动值为描述性统计；暂定和过期记录不会触发同族进阶试练。</p>
   {!known.length ? <p className="training-muted">完成有效对局后自动建立锚点。至少三局、跨两个训练日且波动受控时，启用同族试练。</p> :
   <div className="training-table-wrap"><table><thead><tr><th>场景</th><th>状态</th><th>中位成绩 / 波动</th><th>速度 / 精度</th><th>依据</th></tr></thead><tbody>{known.map(a=><tr key={a.scenario}>
    <td>{a.scenario}</td><td>{anchorStatuses[a.status] ?? a.status}</td><td>{a.medianScore.toFixed(1)} / {a.scoreMAD.toFixed(1)}</td>
    <td>{a.hitsPerSecond != null && <small>{a.hitsPerSecond.toFixed(2)} 命中/秒</small>}{a.accuracy != null && <small>{(a.accuracy*100).toFixed(1)}% 命中率</small>}</td>
    <td><small>{a.samples} 局 · {a.sessions} 块 · {a.days} 天</small><small>{a.evidence === "local_execution_context" ? "记录了执行时本地文件上下文" : "同图历史；未绑定本地文件版本"}</small></td>
   </tr>)}</tbody></table></div>}
  </section>
  <section className="training-card"><h2>日常试练与低频观察</h2><p>两局试练使用个人历史参照，不自动增加前后测，也不阻断后续探索。固定观察优先复用主线；需要补充时采用一局熟悉、三局测量。</p><p>每次计划最多一个锚图；新增评估同分类七天最多一次、所有分类合计最多两次，新增时间不超过三分钟及总预算的 10%。放不下就等待，不改变训练分类。</p><p className="training-muted">自然练到同图时记录实际间隔，没有三天补测期限。局数与频率是待验证的默认值；跨日变化与 PB 增长不能证明某个变体带来了收益。</p></section>
  {[...evaluations].sort((a,b)=>b.createdAt-a.createdAt).slice(0,12).map(e=><section className="training-card" key={e.id}>
   <div className="training-section-title"><h3>{e.scenario}</h3><span className="training-badge">{evaluationStatuses[e.status] ?? e.status}</span></div>
   <p>{e.reason} · 新增 {e.extraRuns} 局 / {(e.extraSeconds/60).toFixed(1)} 分钟</p>
   {e.result && <p>中位成绩 {e.result.score.toFixed(1)} · 三局分布：{e.result.scores?.map(v=>v.toFixed(1)).join(" / ") ?? "旧记录未保存"}</p>}
   {e.previous && <p>同图{e.intervalKind === "same_day" ? "当日" : e.intervalKind === "longer_interval" ? "较长间隔" : "跨日"}表现变化：{e.change == null ? "基准为零，保留原始成绩" : change(e.change)} · 实际间隔 {e.intervalHours?.toFixed(1)} 小时 · {e.comparableDays} 个可比训练日</p>}
   {e.previous && <p className="training-muted">期间已记录练习 {(e.exposure.recordedSeconds/60).toFixed(1)} 分钟，同图 {(e.exposure.sameSceneSeconds/60).toFixed(1)} 分钟，同类 {(e.exposure.sameThemeSeconds/60).toFixed(1)} 分钟；试练 {(e.exposure.trialSeconds/60).toFixed(1)} 分钟{e.exposure.trialScenarios?.length ? `（${e.exposure.trialScenarios.join("、")}）` : ""}。这是共同观察窗口，不将变化归因于单个变体；无 CSV 的练习、其他游戏与睡眠情况未知。</p>}
  </section>)}
  {!recent.length && <section className="training-card"><p>完成主线基线且有合适的本地同族变体后，会在预算内安排试练。无需额外下载或上传 SCE。</p></section>}
  {recent.map(st=><section className="training-card" key={st.id}>
   <div className="training-section-title"><h3>{st.trainingScenario}</h3><span className="training-badge">{studyStatuses[st.status] ?? st.status}</span></div>
   <p>训练锚点：{st.anchorScenario}{st.transferScenario && <><br/>迁移保留图：{st.transferScenario}</>}</p>
   <p>{st.protocolId === "daily_trial_v2" ? "历史中位参照" : "旧协议前测"} {st.baseline?.score.toFixed(1) ?? "待记录"} · 试练 {st.trial?.score.toFixed(1) ?? "待记录"}{st.protocolId !== "daily_trial_v2" && <> · 复测 {st.retest?.score.toFixed(1) ?? "待记录"}</>}</p>
   {st.retentionChange != null && <p>同图保持变化：{change(st.retentionChange)}</p>}
   {st.transferChange != null && <p>保留图迁移变化：{change(st.transferChange)}</p>}
   {st.transferContaminated && <p className="training-muted">保留图在两次测量之间被单独练习，已取消本次迁移判断。</p>}
   {st.protocolId === "daily_trial_v2" ? <p className="training-muted">历史参照与变体分数不直接换算进步；本次仅观察当前适配，不绑定迁移测试。</p> : !st.transferScenario && <p className="training-muted">旧协议没有独立保留图，本次只记录同图保持。</p>}
   {st.dueAt != null && <p className="training-muted">复测窗口：{date(st.dueAt)} ～ {date(st.expiresAt ?? st.dueAt)}</p>}
   {st.trial && <div className="training-actions"><span>可选反馈</span>{[["easy","偏易"],["suitable","合适"],["hard","偏难"]].map(([value,label])=><button key={value} aria-pressed={st.feedback===value} disabled={busy} onClick={()=>onFeedback(st.id,value)}>{st.feedback===value ? "✓ " : ""}{label}</button>)}</div>}
  </section>)}
 </>;
});
