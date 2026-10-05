import { useEffect, useRef, useState } from "react";
import { compareScenes, readRequirements, type Scenario, type ScenarioComparison } from "./api";
import type { Descriptor, Fact } from "./contracts.generated";
import TrainingHelp from "./TrainingHelp";

const axisLabels: Record<string,string> = {
 precision:"精度", motion_and_direction_changes:"运动与变向", spatial_selection_and_transfer:"空间选择与转移",
 kill_and_lifetime_windows:"击杀与生存窗口", error_cost_and_score_strategy:"失误代价与计分", phase_and_external_mechanisms:"阶段与外部机制",
};
const factLabels: Record<string,string> = {
 MainBBRadius:"目标配置半径", MainBBHeight:"目标配置高度", MaxSpeed:"速度上限", Acceleration:"加速度上限", NoDodging:"关闭自主 Dodge",
 Gravity:"重力", MinLRTimeChange:"左右变向最短间隔", MaxLRTimeChange:"左右变向最长间隔", MinFBTimeChange:"前后变向最短间隔", MaxFBTimeChange:"前后变向最长间隔",
 ToggleLeftRight:"左右变向开关", ToggleForwardBack:"前后变向开关", DodgeProfileWeights:"Dodge 条目权重",
 MaxHealth:"生命值", HealthRegenPerSec:"每秒生命变化", HealthRegenDelay:"生命变化延迟", MinRespawnDelay:"最短重生延迟", MaxRespawnDelay:"最长重生延迟",
 SpawnOffsetMin:"生成偏移下界", SpawnOffsetMax:"生成偏移上界", scoring_slots_min:"计分槽位下界", scoring_slots_max:"计分槽位上界", angular_size_and_transfer:"目标视角大小与转移角度",
 ideal_body_hits_to_kill:"理想身体命中次数", ideal_post_first_hit_kill_seconds:"首命中后最短击杀时间", self_decay_seconds:"无外伤时的自衰减时间",
 full_refund_reload_at_q50:"命中率 50% 时补弹次数 / 千发",full_refund_reload_at_q75:"命中率 75% 时补弹次数 / 千发",full_refund_reload_at_q90:"命中率 90% 时补弹次数 / 千发",
 MagazineMax:"弹匣容量", AmmoPerShot:"每发耗弹", AmmoReloadedOnKill:"击杀补弹", ReloadTimeFromEmpty:"空弹匣装填时间", TimeBetweenShots:"最短射击间隔", DamagePerShot:"每发伤害",
 Timescale:"配置时钟倍率", MapScale:"地图缩放声明", TimeDilationBaseMultiplier:"目标速度基础倍率", TargetSizeBaseMultiplier:"目标尺寸基础倍率", IsTimeDilationActive:"自适应速度开关", IsTargetSizeActive:"自适应尺寸开关",
 BotMaxLives:"角色生命次数",BotTeams:"角色队伍", DisableScoring:"禁用计分",NoAiming:"禁用自主瞄准",
 precision_radius:"配置半径精度需求",speed_width_pressure:"速度 / 目标宽度压力",acceleration_width_pressure:"加速度 / 目标宽度压力",constant_scoring_slot_scarcity:"恒定计分槽位稀缺度",
};
const statusLabels: Record<string,string>={declared:"配置声明",calculated_under_explicit_conditions:"条件推算",runtime_verified:"运行验证",unknown:"未知"};
const kindLabels: Record<string,string>={strict_precision:"严格精度对照",native_family_order:"同一 benchmark 的家族级别对照",estimated_direction:"配置需求方向估计",partial_axes:"部分维度可对照",incompatible:"任务不兼容"};
const directions: Record<string,string>={higher_precision:"精度要求更高",lower_precision:"精度要求更低",higher_declared_demand:"配置需求增加",lower_declared_demand:"配置需求减少",higher_native_tier:"原生级别更高",lower_native_tier:"原生级别更低",unchanged:"支持的维度未变化",uncertain:"无法给出整体方向"};
function conditionText(value:string) {
 const rules: [string,string][] = [
  ["Absent legacy", "旧格式未声明蓄力或网格命中开关；此计算仅假设它们未启用，游戏默认值仍未验证。"],
  ["No external damage", "无外部伤害、治疗或免疫，负生命恢复从出生立即开始；不是实测阶段长度。"],
  ["One projectile", "每次点击仅一发 hitscan，无连发、蓄力或附加射击；不免疫且不恢复生命。"],
  ["All body hits", "每次身体命中都造成声明的完整伤害，无减伤、增益或外部交互。"],
  ["Successful shots", "以声明的最短射击间隔连续命中；不含首次找目标、装填和反应时间。"],
  ["One hit kills", "一发命中即击杀且补满弹匣；命中独立且概率固定，缺弹时自动补满；未模拟装填和射速的重叠。"],
  ["Resolved phase", "按已解析阶段取计分槽位下界，未确认资格不计入下界。"],
  ["Candidate eligibility", "计分资格范围来自候选声明，不是运行时同时存在的目标数。"],
 ];
 return rules.find(([prefix])=>value.startsWith(prefix))?.[1] ?? value;
}
function reasonText(value:string) {
 const labels:Record<string,string>={missing_descriptor:"缺少所选版本的需求描述",task_mismatch_or_unknown:"任务类型不同或尚未确认",training_goal_mismatch:"训练目标不同",unresolved_task:"未确认任务分类",unsupported_descriptor_version:"需求模型版本不兼容",external_helper_influence_not_modeled:"存在尚未建模的辅助角色作用",unknown_or_phase_dependent_scoring_slots:"计分资格未知或随阶段变化",unverified_scoring_eligibility:"旧格式未明确声明计分资格",unsupported_hitbox_shape:"命中形状暂不支持整体比较",unknown_dodge_gate:"自主运动开关未知",ability_motion_not_modeled:"能力产生的运动尚未建模",gravity_not_modeled:"重力作用尚未纳入运动模型",unknown_or_adaptive_modifiers:"自适应开关启用或未声明",target_profile_mismatch:"目标角色集合不同",height_change_outside_radius_basis:"目标高度变化超出当前比较基底",unverified_shape_projection_for_radius_change:"尺寸改变后的形状投影尚未验证",unresolved_phase:"轮换阶段未完全解析"};
 if(value.startsWith("incomplete_fixed_basis:"))return `缺少固定比较项：${factLabels[value.split(":")[1]] ?? value}`;
 return labels[value] ?? value;
}
function display(f:Fact){return f.value!=null ? Number(f.value.toFixed(4)).toString() : f.status==="declared" ? (({true:"是",false:"否"} as Record<string,string>)[f.text ?? ""] ?? f.text ?? "空声明") : "未知";}
function units(f:Fact){if(f.unit==="configured seconds")return "配置秒";if(f.unit==="reloads per 1000 shots")return "次 / 千发";if(f.unit==="hits")return "次";if(f.unit==="slots")return "槽位";return "";}
function FactRow({fact:f}:{fact:Fact}){
 return <li><span>{factLabels[f.key] ?? f.key}：<strong>{display(f)} {units(f)}</strong></span><small>{statusLabels[f.status] ?? f.status}{f.sources?.[0] && ` · ${f.sources[0].profile || "场景"}，第 ${f.sources[0].line} 行`}</small>{!!f.conditions?.length && <TrainingHelp label={`${f.key}推算条件`}><p>该值在列出的假设下计算，不能视为实测轨迹、反应时间或实际阶段时长。</p>{f.conditions.map((condition,i)=><p key={i}>{conditionText(condition)}</p>)}</TrainingHelp>}</li>;
}
export function RequirementDetails({scenario:s}:{scenario:Scenario}){
 const [open,setOpen]=useState(false),[detail,setDetail]=useState<Descriptor|null>(null),[error,setError]=useState("");
 const summary=s.localAssessment?.requirements;
 useEffect(()=>{
  if(!open || !summary)return;
  let active=true;setDetail(null);setError("");
  void readRequirements(s.name,summary.fileSHA256).then(value=>{if(active)setDetail(value);},e=>{if(active)setError(String(e));});
  return ()=>{active=false;};
 },[open,s.name,summary?.fileSHA256]);
 const d=detail ?? summary;
 if(!d)return null;
 const targets=d.targets ?? [],helpers=d.helpers ?? [];
 const count=d.scoringMin!=null && d.scoringMax!=null ? d.scoringMin===d.scoringMax ? String(d.scoringMin) : `${d.scoringMin}–${d.scoringMax}` : "未知";
 return <details className="training-requirements" onToggle={e=>setOpen(e.currentTarget.open)}><summary>查看六维场景需求</summary>{open && <div>
  <p>计分候选角色 {targets.length} 种 · 辅助角色 {helpers.length} 种 · 计分槽位 {count}</p>
  <p className="training-muted">槽位为轮换阶段的候选范围；配置速度上限不等于实际速度。未知值不会按 0 参与排序。</p>
  {!!d.slots?.length && <details><summary>生成与轮换条目</summary>{d.slots.map((slot,i)=><p key={i}>{i+1}. {slot.reference} → {(slot.candidates ?? []).join(" → ") || "未解析"}</p>)}</details>}
  {error && <p role="alert">{error}</p>}{!detail && !error && <p>正在读取此版本的详细需求…</p>}
  {(d.axes ?? []).map(axis=><details key={axis.key}><summary>{axisLabels[axis.key] ?? axis.key}</summary><ul>{(axis.facts ?? []).map((f,i)=><FactRow key={i} fact={f}/>)}</ul></details>)}
  {targets.map(t=><div key={t.bot}><small>{t.bot} · {t.dodgeGate==="off" ? "自主 Dodge 关闭" : t.dodgeGate==="candidate_on" ? "自主 Dodge 已配置" : "自主 Dodge 未知"}</small>{t.motionModels?.map((m,i)=><p key={i}>{m.axis==="LR"?"左右":"前后"}间隔 {m.dwellMin}–{m.dwellMax} 配置秒；一维周期假设下，中点间隔的速度峰值 {m.midpointEnvelope?.peakSpeed.toFixed(3)}、位移范围 {m.midpointEnvelope?.centerExcursion.toFixed(3)}。<TrainingHelp label="变向联合推算">速度、加速度和间隔共同决定此理想模型的运动包络。间隔更短也可能变成更慢、更小的抖动，所以不把变向频率单独当作难度系数。这不是游戏目标的实测路径。</TrainingHelp></p>)}</div>)}
  {d.map && <p>地图声明：生成点 {d.map.counts ? (d.map.counts.SpawnPoint ?? 0)+(d.map.counts.PlayerSpawn ?? 0) : "未知"} · 生成体积 {d.map.counts ? d.map.counts.SpawnVolume ?? 0 : "未知"} · 伤害区域 {d.map.counts ? d.map.counts.Hurt ?? 0 : "未知"}。尚未验证碰撞、视线、相机与体积尺度。</p>}
  {d.hazards?.map((h,i)=><p key={i}>{h.character} 连续处于伤害区域的条件存活范围：{h.minSeconds}–{h.maxSeconds} 配置秒；假设伤害生效、无免疫或外部治疗，不能据此确认阶段长度。</p>)}
 </div>}</details>;
}
export function RequirementCompare({catalog}:{catalog:Scenario[]}){
 const [open,setOpen]=useState(false),[a,setA]=useState(""),[b,setB]=useState("");
 const [result,setResult]=useState<ScenarioComparison|null>(null),[error,setError]=useState(""),[busy,setBusy]=useState(false);const request=useRef(0);
 const available=catalog.filter(s=>s.localAssessment?.status==="file_parsed_model_unfitted" && s.localAssessment.requirements);
 const select=(side:"a"|"b",value:string)=>{request.current++;setResult(null);setError("");setBusy(false);if(side==="a")setA(value);else setB(value);};
 const run=async()=>{const id=++request.current;setBusy(true);setError("");try{const value=await compareScenes(a,b);if(request.current===id)setResult(value);}catch(e){if(request.current===id)setError(String(e));}finally{if(request.current===id)setBusy(false);}};
 return <details className="training-requirements" onToggle={e=>setOpen(e.currentTarget.open)}><summary>比较两张图的需求</summary>{open && <div><div className="training-actions">
  <label>参考图<select aria-label="需求参考图" value={a} onChange={e=>select("a",e.target.value)}><option value="">选择已解析场景</option>{available.map(s=><option key={s.name} value={s.name}>{s.name}</option>)}</select></label>
  <label>候选图<select aria-label="需求候选图" value={b} onChange={e=>select("b",e.target.value)}><option value="">选择已解析场景</option>{available.filter(s=>s.name!==a).map(s=><option key={s.name} value={s.name}>{s.name}</option>)}</select></label>
  <button disabled={!a||!b||a===b||busy} onClick={()=>void run()}>{busy?"比较中…":"比较"}</button></div>
  {error && <p role="alert">{error}</p>}{result && <div><p><strong>{kindLabels[result.result.kind] ?? result.result.kind}</strong> · {directions[result.result.direction] ?? result.result.direction}</p>
   <table><thead><tr><th>配置维度</th><th>参考图</th><th>候选图</th><th>差值</th></tr></thead><tbody>{(result.result.axes ?? []).map(axis=><tr key={axis.key}><td>{factLabels[axis.key] ?? axis.key}</td><td>{axis.a?.toFixed(4) ?? "未知"}</td><td>{axis.b?.toFixed(4) ?? "未知"}</td><td>{axis.delta?.toFixed(4) ?? "未知"}</td></tr>)}</tbody></table>
   <p>这些是条件配置描述，不是成功率或总难度分。{result.result.neighborDistance!=null ? "完整、可比的需求只在原有候选范围内辅助排序，不触发自动升级。" : "存在缺失或机制差异，此次不用于整体排序。"}</p>
   {!!result.result.unknown?.length && <details><summary>无法排序的原因</summary><ul>{result.result.unknown.map(reason=><li key={reason}>{reasonText(reason)}</li>)}</ul></details>}
   {!!result.result.changedMechanisms?.length && <p>发生变化的机制：{result.result.changedMechanisms.map(m=>(({map_geometry:"地图几何",configuration_outside_supported_caps:"支持范围以外的配置"} as Record<string,string>)[m] ?? m)).join("、")}</p>}
  </div>}
 </div>}</details>;
}
