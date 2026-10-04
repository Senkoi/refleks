import { memo } from "react";
import TrainingHelp from "./TrainingHelp";
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
  <section className="training-card"><h2>个人场景锚点 <TrainingHelp label="个人场景锚点"><p>每张练过的图分别建立近期表现基准。采用近 45 天、最近 6 个训练块，每块最多 12 局；先取块内中位数，再取各块中位数，不使用 PB。</p><p>只合并同图、同版本和相同设置的有效完整对局。至少 3 局、跨 2 天、成绩为正且波动比例不超过 15% 时标记跨日稳定；超过 14 天未练则需要更新。</p><p>新成绩会更新基准；更换版本、灵敏度、FOV 或单局时长会重新分组。波动是描述性统计。稳定锚点用于同族试练，候选关系还需通过本地机制与布局配置对照。未绑定本地版本的历史会单独标注。</p></TrainingHelp></h2>
   {!known.length ? <p className="training-muted">暂无锚点 · 完成有效对局后自动建立</p> :
   <div className="training-table-wrap"><table><thead><tr><th>场景</th><th>状态</th><th>中位成绩 / 波动</th><th>速度 / 精度</th><th>依据</th></tr></thead><tbody>{known.map(a=><tr key={a.scenario}>
    <td>{a.scenario}</td><td>{anchorStatuses[a.status] ?? a.status}</td><td>{a.medianScore.toFixed(1)} / {a.scoreMAD.toFixed(1)}</td>
    <td>{a.hitsPerSecond != null && <small>{a.hitsPerSecond.toFixed(2)} 命中/秒</small>}{a.accuracy != null && <small>{(a.accuracy*100).toFixed(1)}% 命中率</small>}</td>
    <td><small>{a.samples} 局 · {a.sessions} 块 · {a.days} 天</small><small>{a.evidence === "local_execution_context" ? "已绑定本地版本" : "历史未绑定版本"}</small></td>
   </tr>)}</tbody></table></div>}
  </section>
  <section className="training-card"><h2>日常试练与低频观察 <TrainingHelp label="试练与观察"><p>两局试练使用个人历史参照，不自动增加前后测。固定观察优先复用主线，需要补充时采用一局熟悉、三局测量。</p><p>每次计划最多一个锚图；新增观察同分类七天最多一次、所有分类合计最多两次，新增时间不超过三分钟及总预算的 10%。预算不足时等待。</p><p>自然练到同图时记录实际间隔，没有三天补测期限。这些局数与频率是待验证的默认值；跨日变化与 PB 增长不能证明某个变体带来了收益。</p></TrainingHelp></h2><div className="training-metrics"><div><strong>2 局</strong><span>日常试练</span></div><div><strong>1 + 3 局</strong><span>熟悉 + 固定测量</span></div><div><strong>≤ 2 次/周</strong><span>新增固定观察</span></div></div></section>
  {[...evaluations].sort((a,b)=>b.createdAt-a.createdAt).slice(0,12).map(e=><section className="training-card" key={e.id}>
   <div className="training-section-title"><div className="training-inline-title"><h3>{e.scenario}</h3><TrainingHelp label={`${e.scenario}观察依据`}><p>{e.reason}</p><p>期间练习为已记录的 CSV 数据；其他游戏与睡眠情况未知。成绩变化不归因于单个变体。</p></TrainingHelp></div><span className="training-badge">{evaluationStatuses[e.status] ?? e.status}</span></div>
   <p>补充 {e.extraRuns} 局 · {(e.extraSeconds/60).toFixed(1)} 分钟</p>
   {e.result && <p>中位成绩 {e.result.score.toFixed(1)} · 三局分布：{e.result.scores?.map(v=>v.toFixed(1)).join(" / ") ?? "旧记录未保存"}</p>}
   {e.previous && <p>同图{e.intervalKind === "same_day" ? "当日" : e.intervalKind === "longer_interval" ? "较长间隔" : "跨日"}表现变化：{e.change == null ? "基准为零，保留原始成绩" : change(e.change)} · 实际间隔 {e.intervalHours?.toFixed(1)} 小时 · {e.comparableDays} 个可比训练日</p>}
   {e.previous && <p className="training-muted">期间已记录练习 {(e.exposure.recordedSeconds/60).toFixed(1)} 分钟，同图 {(e.exposure.sameSceneSeconds/60).toFixed(1)} 分钟，同类 {(e.exposure.sameThemeSeconds/60).toFixed(1)} 分钟；试练 {(e.exposure.trialSeconds/60).toFixed(1)} 分钟{e.exposure.trialScenarios?.length ? `（${e.exposure.trialScenarios.join("、")}）` : ""}。</p>}
  </section>)}
  {!recent.length && <section className="training-card"><p>暂无试练记录 <TrainingHelp label="试练条件">完成主线基线且有合适的本地同族变体后，会在预算内安排试练。无需额外下载或上传 SCE。</TrainingHelp></p></section>}
  {recent.map(st=><section className="training-card" key={st.id}>
   <div className="training-section-title"><div className="training-inline-title"><h3>{st.trainingScenario}</h3><TrainingHelp label={`${st.trainingScenario}试练`}><p>{st.protocolId === "daily_trial_v2" ? "历史参照与变体分数不直接换算进步；本次仅观察当前适配，不绑定迁移测试。" : !st.transferScenario ? "旧协议没有独立保留图，本次只记录同图保持。" : "旧协议保留图用于迁移观察。"}</p></TrainingHelp></div><span className="training-badge">{studyStatuses[st.status] ?? st.status}</span></div>
   <p>训练锚点：{st.anchorScenario}{st.transferScenario && <><br/>迁移保留图：{st.transferScenario}</>}</p>
   <p>{st.protocolId === "daily_trial_v2" ? "历史中位参照" : "旧协议前测"} {st.baseline?.score.toFixed(1) ?? "待记录"} · 试练 {st.trial?.score.toFixed(1) ?? "待记录"}{st.protocolId !== "daily_trial_v2" && <> · 复测 {st.retest?.score.toFixed(1) ?? "待记录"}</>}</p>
   {st.retentionChange != null && <p>同图保持变化：{change(st.retentionChange)}</p>}
   {st.transferChange != null && <p>保留图迁移变化：{change(st.transferChange)}</p>}
   {st.transferContaminated && <p className="training-muted">保留图在两次测量之间被单独练习，已取消本次迁移判断。</p>}
   {st.dueAt != null && <p className="training-muted">复测窗口：{date(st.dueAt)} ～ {date(st.expiresAt ?? st.dueAt)}</p>}
   {st.trial && <div className="training-actions"><span>可选反馈</span>{[["easy","偏易"],["suitable","合适"],["hard","偏难"]].map(([value,label])=><button key={value} aria-pressed={st.feedback===value} disabled={busy} onClick={()=>onFeedback(st.id,value)}>{st.feedback===value ? "✓ " : ""}{label}</button>)}</div>}
  </section>)}
 </>;
});
