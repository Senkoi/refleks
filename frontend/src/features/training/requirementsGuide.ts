import type { Descriptor, Fact, Target } from "./contracts.generated";

export type RequirementGuide = {
  key: string;
  title: string;
  meaning: string;
  observations: string[];
  status: "已读取配置" | "条件推算" | "待确认";
};

// Go nil slices and older stored descriptors can both arrive as null.
export function requirementList<T>(value: readonly T[] | null | undefined): T[] {
  return Array.isArray(value) ? value.filter(item => item != null) : [];
}

function numeric(fact: Fact | undefined): number | null {
  return fact && typeof fact.value === "number" && Number.isFinite(fact.value) ? fact.value : null;
}

function targetValues(targets: Target[], key: string, windows = false): number[] | null {
  if (!targets.length) return null;
  const values = targets.map(target => numeric(requirementList(windows ? target.windows : target.facts).find(fact => fact.key === key)));
  return values.every(value => value != null) ? values as number[] : null;
}

function range(values: number[]): string {
  const low = Math.min(...values), high = Math.max(...values);
  const format = (value: number) => String(Number(value.toFixed(3)));
  return low === high ? format(low) : `${format(low)}–${format(high)}`;
}

export function requirementGuide(descriptor: Descriptor): RequirementGuide[] {
  const targets = requirementList(descriptor.targets);
  const helpers = requirementList(descriptor.helpers);
  const axes = requirementList(descriptor.axes);
  const facts = axes.flatMap(axis => requirementList(axis.facts));
  const radii = targetValues(targets, "MainBBRadius");
  const sizesKnown = !!radii && radii.every(value => value > 0);
  const hits = targetValues(targets, "ideal_body_hits_to_kill", true);
  const decay = targetValues(targets, "self_decay_seconds", true);
  const enabled = (key: string) => facts.some(fact => fact.key === key && fact.status === "declared" && fact.text === "true");
  const motion: string[] = [];
  if (targets.some(target => target.dodgeGate === "candidate_on")) motion.push("目标配置包含自主运动与变向。");
  else if (targets.length && targets.every(target => target.dodgeGate === "off")) motion.push("自主变向开关关闭；其他运动仍需结合配置判断。");
  else motion.push("目标的运动方式尚未确认。");
  if (targets.some(target => requirementList(target.abilities).length)) motion.push("还有能力控制角色行为，常规速度参数不能说明完整路线。");
  else motion.push("速度上限和变向间隔已列在详细配置中，实际运动轨迹尚未记录。");

  const low = descriptor.scoringMin, high = descriptor.scoringMax;
  const slotsKnown = typeof low === "number" && Number.isFinite(low) && typeof high === "number" && Number.isFinite(high);
  const slots = slotsKnown ? `${low === high ? low : `${low}–${high}`} 个` : "尚未确认";
  const kill: string[] = [];
  if (hits) kill.push(`按伤害与血量的计算条件，身体命中 ${range(hits)} 次可击杀。`);
  if (decay) kill.push(`无外部伤害或治疗时，目标自行衰减时间约 ${range(decay)} 秒（条件推算）。`);
  if (!kill.length) kill.push("击杀需要多少次命中、目标会存在多久，目前还不能可靠推算。");
  else kill.push("这些时间不包含找目标、反应和微调耗时。");

  const cost: string[] = [];
  if (enabled("ScoreMultAccuracy") || enabled("MultSqrtAcc")) cost.push("计分配置启用了准确率相关规则。");
  if (facts.some(fact => ["MagazineMax", "ReloadTimeFromEmpty"].includes(fact.key) && numeric(fact) != null)) cost.push("已读取弹药或装填设置，可查看详细配置。");
  const costKnown = cost.length > 0;
  if (!costKnown) cost.push("具体扣分和装填代价尚未确认，不能判断乱点的成本。");

  const special: string[] = [];
  if (helpers.length) special.push("含非计分角色，它们可能影响场景流程。");
  if (requirementList(descriptor.slots).some(slot => requirementList(slot.candidates).length > 1)) special.push("生成配置含角色轮换条目。");
  if (enabled("IsTimeDilationActive")) special.push("启用了自适应速度开关。");
  if (enabled("IsTargetSizeActive")) special.push("启用了自适应尺寸开关。");
  if ((descriptor.map?.counts?.Hurt ?? 0) > 0) special.push("地图声明了伤害区域，实际作用还需确认。");
  const specialKnown = special.length > 0;
  if (!specialKnown) special.push("目前未形成可靠的特殊机制结论；缺少声明不代表机制不存在。");

  return [
    { key: "precision", title: "瞄准精度", meaning: "能否把准心准确放进目标的命中区域。", observations: [sizesKnown ? "已读取命中区域的配置尺寸。" : "命中尺寸信息不足。", "屏幕上目标看起来有多大，还需结合距离和视角；目前不标高低难度。"], status: sizesKnown ? "已读取配置" : "待确认" },
    { key: "motion_and_direction_changes", title: "追踪控制", meaning: "目标移动或变向时，能否持续跟住它。", observations: motion, status: targets.length && targets.every(target => ["off", "candidate_on"].includes(target.dodgeGate)) ? "已读取配置" : "待确认" },
    { key: "spatial_selection_and_transfer", title: "切换选择", meaning: "选中下一目标，并把准心转移过去。", observations: [`计分目标名额：${slots}（生成配置）。`, "名额不等于同时可见的目标数；转移距离和目标离准心的位置尚未确认。"], status: slotsKnown ? "已读取配置" : "待确认" },
    { key: "kill_and_lifetime_windows", title: "击杀节奏", meaning: "需要连中多少次，以及要在多久内处理目标。", observations: kill, status: hits || decay ? "条件推算" : "待确认" },
    { key: "error_cost_and_score_strategy", title: "失误代价", meaning: "打空会怎样影响得分和后续输出。", observations: cost, status: costKnown ? "已读取配置" : "待确认" },
    { key: "phase_and_external_mechanisms", title: "特殊机制", meaning: "目标轮换、辅助角色或规则变化怎样影响练习。", observations: special, status: specialKnown ? "已读取配置" : "待确认" },
  ];
}
