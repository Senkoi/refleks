import { memo } from "react";
import type { PersonalAnchor, TrainingStudy } from "./api";

const anchorStatuses: Record<string,string> = { stable: "跨日稳定", provisional: "暂定", variable: "波动较大", stale: "需要更新" };
const studyStatuses: Record<string,string> = { planned: "等待前测与试练", waiting: "等待隔日复测", due: "复测已到期", transfer_due: "等待迁移复测", measured: "复测完成", expired: "复测窗口已过", incomplete: "试练未完成", baseline_missing: "缺少可比前测", invalidated: "场景版本变化", settings_changed: "测量设置变化" };
const date = (v: number) => new Date(v).toLocaleString("zh-CN");
studyStatuses.not_started = "未开始 · 已替换";
const change = (v: number) => (v >= 0 ? "+" : "") + (v*100).toFixed(1) + "%";

export default memo(function PersonalEvidence({ anchors, studies, busy, onFeedback }: {
 anchors: PersonalAnchor[]; studies: TrainingStudy[]; busy: boolean; onFeedback: (id:string,value:string)=>void;
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
  <section className="training-card"><h2>试练与跨日复测</h2><p>前测与复测均为一局熟悉、两局测量。试练完成 24～72 小时后安排固定锚图复测；设置或场景版本变化时停止比较。</p><p className="training-muted">保持变化比较同一锚图，迁移变化比较独立保留图。单次变化与 PB 增长不能证明调度策略带来了长期收益。</p></section>
  {!recent.length && <section className="training-card"><p>完成主线基线且有可比的本地同族变体后，会在预算内安排试练与复测。无需额外下载或上传 SCE。</p></section>}
  {recent.map(st=><section className="training-card" key={st.id}>
   <div className="training-section-title"><h3>{st.trainingScenario}</h3><span className="training-badge">{studyStatuses[st.status] ?? st.status}</span></div>
   <p>训练锚点：{st.anchorScenario}{st.transferScenario && <><br/>迁移保留图：{st.transferScenario}</>}</p>
   <p>前测 {st.baseline?.score.toFixed(1) ?? "待记录"} · 试练 {st.trial?.score.toFixed(1) ?? "待记录"} · 复测 {st.retest?.score.toFixed(1) ?? "待记录"}</p>
   {st.retentionChange != null && <p>同图保持变化：{change(st.retentionChange)}</p>}
   {st.transferChange != null && <p>保留图迁移变化：{change(st.transferChange)}</p>}
   {st.transferContaminated && <p className="training-muted">保留图在两次测量之间被单独练习，已取消本次迁移判断。</p>}
   {!st.transferScenario && <p className="training-muted">尚无合适的独立保留图，本次只记录同图保持。</p>}
   {st.dueAt != null && <p className="training-muted">复测窗口：{date(st.dueAt)} ～ {date(st.expiresAt ?? st.dueAt)}</p>}
   {st.trial && <div className="training-actions"><span>可选反馈</span>{[["easy","偏易"],["suitable","合适"],["hard","偏难"]].map(([value,label])=><button key={value} aria-pressed={st.feedback===value} disabled={busy} onClick={()=>onFeedback(st.id,value)}>{st.feedback===value ? "✓ " : ""}{label}</button>)}</div>}
  </section>)}
 </>;
});
