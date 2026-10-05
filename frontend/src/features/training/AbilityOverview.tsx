import { memo, useId, useState } from "react";
import { ChevronDown } from "lucide-react";
import TrainingHelp from "./TrainingHelp";
import type { State } from "./api";

const themes: Record<string, string> = { static: "静态点击", dynamic: "动态点击", smooth: "精确追踪", reactive: "反应追踪", switching_speed: "速度切换", switching_evasive: "规避切换" };
const usable = new Set(["inferred", "estimated", "partial"]);

export default memo(function AbilityOverview({ levels = [], tiers = {}, coverage = [] }: { levels?: State["playerLevels"]; tiers?: State["templateTiers"]; coverage?: State["demandCoverage"] }) {
  const [expanded, setExpanded] = useState(false);
  const detailId = useId();
  return <section className="training-card training-levels">
    <div className="training-section-title">
      <h2>自动能力评估 <TrainingHelp label="自动能力评估"><p>我会先看你近期练得怎么样，再帮你挑合适的内容；记录不够时，也会参考已有的 Benchmark 成绩喵。</p><p>“训练参考”是选练习模板用的档位，可以在某些场景还没测完时给出。“本组参考段位”需要这组所有必需场景都有有效成绩；两者含义不同。</p><p>个人水平按不同任务和测试组分别记录；缺少同步的目标与准心轨迹时，不估算变向反应时间、平滑程度或微调耗时。</p><p>还没测齐也不用着急喵，我会参考已有成绩；实在没有依据才暂用 Novice 模板。这里的估计不会改动你的官方段位。</p></TrainingHelp></h2>
      <button type="button" className="training-level-toggle" aria-expanded={expanded} aria-controls={detailId} onClick={() => setExpanded(v => !v)}>{expanded ? "收起成绩详情" : "查看成绩详情"}<ChevronDown size={15} aria-hidden="true" /></button>
    </div>
    <div className="training-ability-summary">{Object.entries(themes).map(([theme, label]) => {
      const known = levels.filter(l => l.theme === theme && usable.has(l.status));
      return <div className="training-ability-item" key={theme}><strong>{label}</strong><span>{tiers[theme] || "Novice"}{!known.length && <small> · 暂用</small>}</span><small>{known.length ? `${known.length} 组已有成绩` : "等待有效成绩"}</small></div>;
    })}</div>
    <details className="training-requirements"><summary>有记录的场景需求覆盖</summary><p className="training-muted">这里列出有近期稳定表现、且绑定本地文件上下文的配置范围，不是反应时间或能力极限。仅有名称的历史成绩继续用于任务评估，不会绑定到当前 SCE。</p>{coverage.length ? <table><thead><tr><th>任务</th><th>配置描述</th><th>覆盖范围</th><th>场景数</th></tr></thead><tbody>{coverage.map(c=><tr key={`${c.theme}/${c.key}`}><td>{themes[c.theme] ?? c.theme}</td><td>{({precision_radius:"配置半径精度",speed_width_pressure:"速度 / 宽度压力",acceleration_width_pressure:"加速度 / 宽度压力",constant_scoring_slot_scarcity:"恒定计分槽位稀缺度"} as Record<string,string>)[c.key] ?? c.key}</td><td>{c.min.toFixed(3)}–{c.max.toFixed(3)}</td><td>{c.scenes.length}</td></tr>)}</tbody></table> : <p>尚无满足稳定性和文件绑定条件的需求覆盖；原生 benchmark 与历史成绩评估仍可使用。</p>}</details>
    <div id={detailId} hidden={!expanded} className="training-level-detail">
      <div className="training-level-glossary">
        <span>成绩时间 <TrainingHelp label="成绩时间（窗口）"><p>“近 7 天”表示这次参考了最近七天的成绩，不是要求你连续训练七天。</p><p>每张图先看近 7 天，未找到至少 3 局可比较的完整成绩时，再看近 14、30、45 天。这里显示的是本组用到的最长时间范围。</p><p>若使用了没有达成日期的 Benchmark 成绩，会标记“含已有测试成绩”；这些成绩不会被算成近期训练局数。范围内的成绩不会仅因时间更早就被扣分。</p></TrainingHelp></span>
        <span>有效场景 <TrainingHelp label="有效场景（覆盖）"><p>“2 / 3 张”表示这一测试组需要三张图，目前有两张具备可用于评估的成绩。它不是训练次数，也不是整个场景库的完成率。</p><p>一张图有足够的近期完整对局，或有可用的已有 Benchmark 成绩，才会计入。缺少成绩的图不会当成零分。</p></TrainingHelp></span>
        <span>参考段位 <TrainingHelp label="本组参考段位（定级）"><p>等这一组所有必需场景都有有效成绩，才会给出整组参考段位。整组以其中较低的场景段位为准，避免用一张强项图代表整个能力。</p><p>“待补齐”表示还缺场景成绩；“低于首档”表示场景已测齐，但至少一张还没有达到这组最低段位的分数线。它们不是同一种情况。</p><p>含没有日期的已有测试成绩时，段位会标记“暂定”。你可以继续正常训练，新的完整成绩会逐步更新评估。</p></TrainingHelp></span>
      </div>
      <div className="training-level-scroll" role="region" aria-label="各能力成绩详情" tabIndex={expanded ? 0 : -1}>{Object.entries(themes).map(([theme, label]) => {
        const rows = levels.filter(l => l.theme === theme);
        return <section className="training-level-group" key={theme}><h3>{label}</h3>{!rows.length ? <p className="training-muted">我还在等你的测试成绩喵</p> : rows.map(l => {
          const full = l.status === "inferred" || l.status === "estimated";
          const rank = !full ? "待补齐" : l.rank === "unranked" ? "低于首档" : l.rank || "待确认";
          const hasBenchmark = l.source === "benchmark" || l.source === "mixed" || l.status === "estimated";
          return <div className="training-level-row" key={`${l.system}/${l.nativeDifficulty}/${l.category}/${l.group}`}>
            <div className="training-level-row-title"><strong>{l.system || "Benchmark"} · {l.nativeDifficulty}</strong><span>{l.group || l.category}</span></div>
            <dl><div><dt>本组参考段位</dt><dd>{l.status === "estimated" && "暂定 · "}{rank}</dd></div><div><dt>训练参考</dt><dd>{l.trainingTier || "等待成绩"}</dd></div><div><dt>有效场景</dt><dd>{l.scenarios} / {l.required} 张</dd></div><div><dt>成绩时间</dt><dd>{l.windowDays ? `近 ${l.windowDays} 天` : hasBenchmark ? "达成日期未知" : "等待成绩"}{hasBenchmark && l.windowDays ? " · 含已有测试成绩" : ""}</dd></div><div><dt>近期有效对局</dt><dd>{l.samples} 局</dd></div></dl>
          </div>;
        })}</section>;
      })}</div>
    </div>
  </section>;
});
