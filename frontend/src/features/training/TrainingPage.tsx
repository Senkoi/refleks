import { useCallback, useEffect, useRef, useState } from "react";
import { ArrowRight, Check, Clock3, Compass, Download, Play, RefreshCw, Search, SlidersHorizontal, Target, Upload } from "lucide-react";
import { openURL } from "@/shared/lib/api";
import { call, readState, type Preferences, type Scenario, type State } from "./api";
import "./training.css";

const labels: Record<string, string> = { auto: "自动 · 按弱项与周覆盖", static: "静态点击", dynamic: "动态点击", smooth: "平滑追踪", reactive: "反应追踪", switching: "目标切换", unknown: "待分类" };
const themes: Record<string, string> = { static: "静态点击", dynamic: "动态点击", smooth: "精确追踪", reactive: "反应追踪", switching_speed: "速度切换", switching_evasive: "规避切换" };
const difficultySources: Record<string, string> = { manual: "手动确认", benchmark: "benchmark 等级", name: "名称推断", playlist: "列表等级推断", unknown: "缺少可靠依据" };
const fitLabels: Record<string, string> = { challenging: "对你偏难", suitable: "对你合适", comfortable: "对你偏易", unknown: "个人适配未知" };
const roles: Record<string, string> = { warmup: "热身", practice: "专项练习", explore: "探索", benchmark: "参考测量" };
const statuses: Record<string, string> = { draft: "待开始", running: "训练中", ready: "下一模块待开始", paused: "已暂停", waiting: "模块到时 · 等待本局结束", completed: "已结束" };
const outcomes: Record<string,string> = { pending: "待完成", list_complete: "列表次数完成", threshold: "阈值达标", measured: "已测量", time_limit: "模块到时", session_limit: "总时长到达", skipped: "已跳过", missed: "游戏已进入后续关卡" };
const initial: Preferences = { planningPolicy: "curriculum", minutes: 30, executionMode: "playlist", focus: "auto", difficulty: "any", benchmark: "", variety: .25, thresholdRatio: .9, autoAdvance: false, autoDiscover: true };
function clock(seconds: number) { const s = Math.max(0, Math.floor(seconds)); return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`; }
function external(url: string) { if (/^https:\/\//i.test(url)) openURL(url); }

export default function TrainingPage() {
  const [state, setState] = useState<State | null>(null);
  const [prefs, setPrefs] = useState(initial);
  const [tab, setTab] = useState("plan");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [url, setURL] = useState("");
  const [filter, setFilter] = useState("");
  const [editing, setEditing] = useState<Scenario | null>(null);
  const hydrated = useRef(false);
  const mounted = useRef(true);
  const refresh = useCallback(async () => {
    const s = await readState();
    if (!mounted.current) return;
    setState(s);
    if (!hydrated.current) { setPrefs({ ...s.preferences, planningPolicy: "curriculum", executionMode: "playlist", autoAdvance: false }); hydrated.current = true; }
  }, []);
  useEffect(() => {
    mounted.current = true;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => { try { await refresh(); } catch (e) { if (mounted.current) setError(String(e)); } finally { if (mounted.current) timer = setTimeout(poll, 2000); } };
    void poll();
    return () => { mounted.current = false; clearTimeout(timer); };
  }, [refresh]);
  async function perform(name: string, fn: () => Promise<unknown>, message = "") {
    if (busy) return; setBusy(name); setError(""); setNotice("");
    try { await fn(); await refresh(); if (mounted.current && message) setNotice(message); }
    catch (e) { if (mounted.current) setError(String(e)); }
    finally { if (mounted.current) setBusy(""); }
  }
  const plan = state?.plan;
  const active = !!plan && ["running", "ready", "paused", "waiting"].includes(plan.status);
  const benchmarks = [...new Set(state?.catalog.flatMap(s => s.benchmarks?.map(b => b.name) ?? (s.benchmark ? [s.benchmark] : [])) ?? [])].sort();
  const catalog = (state?.catalog ?? []).filter(s => `${s.name} ${s.skill} ${s.sources?.map(x => x.title).join(" ")}`.toLowerCase().includes(filter.toLowerCase()));
  const enabled = state?.catalog.filter(s => s.enabled).length ?? 0;
  const current = plan?.blocks[plan.index];
  const localStatus = (name: string) => { const status = state?.catalog.find(s => s.name.toLowerCase() === name.toLowerCase())?.localAssessment?.status; if (status === "file_parsed_model_unfitted") return "本地 SCE 已解析；机制证据已缓存，难度尚未标定。"; if (status === "ambiguous_local_versions") return "本地存在同名不同版本 SCE；机制与难度暂不采用。"; if (status === "invalid_local_file") return "本地 SCE 尚不能解析，保持待评估。"; return "等待游戏保存稳定的本地 SCE；首次可直接训练，不额外下载场景。"; };
  const pset = <K extends keyof Preferences>(key: K, value: Preferences[K]) => setPrefs(p => ({ ...p, [key]: value }));

  return <div className="training-page">
    <header className="training-header">
      <div><div className="training-eyebrow">REFLEK’S / ADAPTIVE TRAINING · ALPHA</div><h1>训练工作台</h1><p>从优秀关卡中选出今天的练习，让每一次训练有目标，也有变化。</p></div>
      <div className="training-library-count"><Compass size={20} /><strong>{enabled}</strong><span>可编排关卡</span></div>
    </header>
    <nav className="training-tabs" aria-label="训练模块">
      {[["plan", "当次计划"], ["discover", "发现内容"], ["catalog", "关卡库"]].map(([id,title]) => <button key={id} className={tab === id ? "selected" : ""} onClick={() => setTab(id)}>{title}</button>)}
    </nav>
    {error && <div role="alert" className="training-alert error">{error}</div>}
    {state?.error && <div role="alert" className="training-alert error">{state.error}</div>}
    {notice && <div role="status" className="training-alert">{notice}</div>}
    {busy && <div role="status" className="training-alert"><RefreshCw size={14} className="animate-spin" /> {busy}…{busy === "发现内容" && " 正在读取多个公开来源，可能需要约 2 分钟。"}</div>}

    {tab === "plan" && <>
      <div className="training-columns">
        <section className="training-card training-config">
          <h2><SlidersHorizontal size={18} /> 今天怎么练</h2>
          <fieldset disabled={active || !!busy}>
            <label>可用时间 <span>包含切换与休息</span><div className="training-number"><input aria-label="可用时间" type="number" min={5} max={120} value={prefs.minutes} onChange={e => pset("minutes", Number(e.target.value))} /><span>分钟</span></div></label>
            <label>VDIM 模板<select value={prefs.curriculumId ?? ""} onChange={e => pset("curriculumId", e.target.value)}><option value="">自动 · 按训练重点与周覆盖</option>{state?.curricula?.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}</select><small>自动读取游戏已保存的 VDIM JSON/PLO；保留原场景、顺序与次数。</small></label>
            <label>训练重点<select value={prefs.focus} onChange={e => pset("focus", e.target.value)}>{Object.entries(labels).filter(([k]) => k !== "unknown").map(([k,v]) => <option key={k} value={k}>{v}</option>)}</select></label>
            <label>难度<select value={prefs.difficulty} onChange={e => pset("difficulty", e.target.value)}><option value="any">全部难度</option><option value="novice">Novice</option><option value="intermediate">Intermediate</option><option value="advanced">Advanced</option></select></label>
            <fieldset className="training-benchmark-options"><legend>可轮换的 benchmark</legend><small>用于能力评估与成绩参考；模板中的测量场景保持原位，不额外插入。</small><div className="training-benchmark-list">{benchmarks.map(b => <label className="training-checkbox" key={b}><input type="checkbox" checked={(prefs.benchmarks ?? (prefs.benchmark ? [prefs.benchmark] : [])).includes(b)} onChange={e => setPrefs(p => ({ ...p, benchmark: "", benchmarks: e.target.checked ? [...(p.benchmarks ?? (p.benchmark ? [p.benchmark] : [])), b] : (p.benchmarks ?? (p.benchmark ? [p.benchmark] : [])).filter(x => x !== b) }))} />{b}</label>)}</div>{benchmarks.length === 0 && <small>可在「发现内容」同步 benchmark，帮助解释成绩。</small>}</fieldset>
            <label>变化偏好 <strong>{Math.round(prefs.variety * 100)}%</strong><input aria-label="变化偏好" type="range" min={0} max={.5} step={.05} value={prefs.variety} onChange={e => pset("variety", Number(e.target.value))} /><small>额外探索的时长上限；首次不探索，后续保留原模板，只使用剩余预算。</small></label>
            <label className="training-checkbox"><input type="checkbox" checked={prefs.autoDiscover} onChange={e => pset("autoDiscover", e.target.checked)} /> 应用运行时每周自动发现内容</label>
            <button className="training-primary" disabled={!state?.curricula?.length || !!busy} onClick={() => perform("生成计划", async () => { await call("GenerateTrainingPlan", JSON.stringify({ ...prefs, planningPolicy: "curriculum", executionMode: "playlist", autoAdvance: false })); const path = await call<string>("InstallTrainingPlaylist"); setNotice(`列表已安装：${path}。自动生成只使用一个固定槽位。重启 KovaaK’s 后，在 Local Playlists 中选择 Refleks Adaptive Current 列表。`); })}><Target size={16} />生成并安装本次列表 <ArrowRight size={16} /></button>
          </fieldset>
          {!state?.curricula?.length && <p className="training-muted">先从游戏保存的 VDIM JSON/PLO 导入模板；也会自动读取本地 VDIM 列表。</p>}
        </section>
        <section className="training-main">
          <div className="training-card">
            <div className="training-section-title"><h2><Clock3 size={18} /> {plan ? statuses[plan.status] : "准备好，再开始"}</h2>{plan && <span className="training-badge">{plan.preferences.minutes} 分钟预算</span>}</div>
            {plan ? <>
              {plan.curriculumName && <p className="training-muted">原模板：{plan.curriculumName} · 基础场景、顺序与次数保留；额外探索列在模板之后。</p>}
              {plan.theme && <p className="training-muted">本次 VDIM 专项：{themes[plan.theme] ?? plan.theme} · 按近七天记录自动轮换；列表依然由完成记录校正进度。</p>}
              <div className="training-metrics"><div><strong>{clock(Math.max(0, plan.preferences.minutes * 60 - plan.elapsed))}</strong><span>剩余时间</span></div><div><strong>{clock(plan.recorded)}</strong><span>已记录练习</span></div><div><strong>{plan.blocks.filter(b => b.outcome !== "pending").length} / {plan.blocks.length}</strong><span>训练模块</span></div></div>
              <div className="training-progress"><span style={{ width: `${Math.min(100, plan.elapsed / (plan.preferences.minutes * 60) * 100)}%` }} /></div>
              <div className="training-actions">
                <button disabled={!!busy} onClick={() => perform("测试提醒", () => call("TestTrainingReminder"), "10 秒后显示提醒。现在切回 KovaaK’s，检查游戏画面和声音。")}>测试游戏内提醒</button>
                {plan.preferences.executionMode === "playlist" && <button disabled={!!busy} onClick={() => perform("安装列表", async () => { const path = await call<string>("InstallTrainingPlaylist"); setNotice(`列表已安装：${path}。自动生成只使用一个固定槽位。重启 KovaaK’s 后从 Local Playlists 打开。`); })}>安装到 KovaaK’s</button>}
                {["draft","ready","paused"].includes(plan.status) && <button className="training-primary" disabled={!!busy} onClick={() => perform(plan.preferences.executionMode === "playlist" ? "开始计时" : "启动关卡", () => call("TrainingAction", "start"))}><Play size={15} />{plan.preferences.executionMode === "playlist" ? (plan.status === "paused" ? "继续列表计时" : "开始列表计时") : (plan.status === "paused" ? "继续并启动当前关卡" : "开始当前模块")}</button>}
                {["running","ready","waiting"].includes(plan.status) && <button disabled={!!busy} onClick={() => perform("暂停", () => call("TrainingAction", "pause"))}>暂停计时</button>}
                {["running","paused","waiting"].includes(plan.status) && <button disabled={!!busy} onClick={() => perform("跳过模块", () => call("TrainingAction", "next"))}>{plan.status === "waiting" ? "本局已结束，继续" : "跳过模块"}</button>}
                {active && <button disabled={!!busy} onClick={() => perform("结束训练", () => call("TrainingAction", "finish"))}>结束本次</button>}
                <button disabled={!!busy} onClick={() => perform("导出列表", () => call("ExportTrainingPlaylist"), "已导出列表，重复次数按模块时长估算。")}><Download size={15} />导出列表</button>
              </div>
              {current && active && <p className="training-current">当前记录模块：{current.scenario.name} · 计时 {clock(plan.blockElapsed)} / {clock(current.budget)}。下一关的完成记录出现后会校正模块与起始时间；到时提醒不会中断游戏。</p>}
              {plan.reminder && <div role="alert" className="training-alert">{plan.reminder}</div>}
              <p className="training-muted">{plan.preferences.executionMode === "playlist" ? "在 KovaaK’s 的 Local Playlists 中运行已安装列表。模块和总时间按计时提醒，完成的对局自动计入；即使中途反复重开，计时也会继续。列表切关仍由游戏执行。无边框窗口可显示浮层，全屏独占可能只听到声音。" : "完成并写出的对局自动计入。切图和休息消耗总预算，暂停按钮暂停工作台计时。"}</p>
            </> : <div className="training-empty"><Target size={35} /><p>选择时间与训练重点，生成一份可直接执行的计划。</p><small>每个模块都带有选图理由、练习提示和时间上限。</small></div>}
          </div>
          {plan?.blocks.map((b,i) => <article key={`${plan.id}-${i}`} className={`training-card training-block ${i === plan.index && active ? "current" : ""}`}>
            <div className="training-block-number">{b.outcome !== "pending" ? <Check size={17} /> : String(i + 1).padStart(2,"0")}</div>
            <div className="training-block-body"><div className="training-section-title"><span className="training-eyebrow">{roles[b.role]} · {labels[b.scenario.skill]}</span><span className="training-badge">{plan.preferences.executionMode === "playlist" ? `${b.playCount || 1} 局 · 约 ${clock(b.budget)}` : `上限 ${clock(b.budget)}`}</span></div><h3>{b.scenario.name}</h3>{b.anchorScenario && <small>探索目标来源：{b.anchorScenario}</small>}<p>{localStatus(b.scenario.name)}</p>{b.benchmark && <small>本次测量：{b.benchmark}</small>}<p>{b.cue}</p>{b.difficultyEvidence?.benchmarks?.map((m, i) => <small key={i}>参考：{m.name} · {m.category} / {m.group}{m.benchmarkId ? ` · ID ${m.benchmarkId}` : ""}</small>)}{b.difficultyEvidence && <p>难度 {b.difficultyEvidence.level} · {difficultySources[b.difficultyEvidence.source] ?? "来源未知"} · {fitLabels[b.difficultyEvidence.fit] ?? "个人适配未知"}{b.difficultyEvidence.samples >= 3 && `（${b.difficultyEvidence.samples} 局可比记录）`}</p>}{b.timing && <p>单局约 {clock(b.timing.seconds)} · {b.timing.source === "history" ? `同版本/设置的 ${b.timing.samples} 条历史记录中位数` : b.timing.source === "default" ? "默认估计，历史不足" : "关卡库预估，历史不足"} · 近 24 小时已练 {clock(b.timing.recentSeconds)}</p>}<details><summary>为什么选这张图</summary><p>{b.reason}</p>{b.scenario.sources?.map((s,i) => <button className="training-link" key={i} onClick={() => external(s.url)}>{s.title}</button>)}</details><div className="training-block-footer"><span>{b.target > 0 ? `目标 ≥ ${b.target.toFixed(1)}` : b.role === "benchmark" ? "固定完成一次" : "不设分数门槛"}</span><span>{b.runs} 局 · {clock(b.recorded)} · {outcomes[b.outcome]} <button className="training-link" onClick={() => setEditing({ ...b.scenario })}>评价关卡</button></span></div></div>
          </article>)}
          {!!plan?.warnings.length && <div className="training-alert"><ul>{plan.warnings.map(w => <li key={w}>{w}</li>)}</ul></div>}
        </section>
      </div>
      <section className="training-card"><h2>近 7 天能力覆盖</h2><div className="training-skill-grid">{state?.skills.map(s => <div key={s.skill}><strong>{labels[s.skill]}</strong><b>{s.minutes.toFixed(1)} <small>分钟</small></b><span>{s.samples} 条记录</span><p>{s.evidence}</p></div>)}</div></section>
      {!!state?.history.length && <section className="training-card"><h2>近期计划</h2>{state.history.slice(-5).reverse().map(p => <p key={p.id}>{new Date(p.created).toLocaleString()} · {p.preferences.minutes} 分钟预算 · 已记录 {clock(p.recorded)} · {statuses[p.status]}</p>)}</section>}
    </>}

    {tab === "discover" && <>
      <section className="training-card"><h2><Compass size={18} /> 发现新的训练内容</h2><p>从公开训练列表提取真实关卡；保存来源，并将尚不能解析的网页与分享码作为候选线索。</p><div className="training-actions"><button className="training-primary" disabled={!!busy} onClick={() => perform("发现内容", () => call("DiscoverTrainingContent"), "发现完成；分类由名称推断，未知分类不会自动进入计划。 ")}><Search size={16} />扫描公开来源{state?.searchConfigured ? "与全网" : ""}</button><button disabled={!!busy} onClick={() => perform("同步 benchmark", () => call("ImportTrainingBenchmarks"), "已导入可用 benchmark 定义。 ")}><RefreshCw size={16} />同步 benchmark</button><button disabled={!!busy} onClick={() => perform("导入本地列表", () => call("ImportTrainingPlaylist"))}><Upload size={16} />导入 JSON / PLO</button></div><div className="training-source-input"><input aria-label="训练来源网址" placeholder="粘贴公开训练列表或 JSON 地址" value={url} onChange={e => setURL(e.target.value)} /><button disabled={!!busy || !url.trim()} onClick={() => perform("读取来源", () => call("ImportTrainingSource", url.trim()))}>读取来源</button></div>
        <div className="training-search-status"><span className={`training-dot ${state?.searchConfigured ? "configured" : ""}`} />{state?.searchConfigured ? "已检测到全网搜索配置" : "公开资源扫描可用 · 全网搜索待配置"}</div>
        {!state?.searchConfigured && <p className="training-muted">当前扫描公开 GitHub 仓库及 Ridd、4BK 资源。更广的网页搜索可使用你自己的 Brave Search API：设置环境变量 <code>REFLEKS_BRAVE_API_KEY</code> 后重启应用。搜索只发送训练主题，不上传成绩或录像。</p>}
        <p className="training-muted">最近发现：{state?.discovery.updated ? new Date(state.discovery.updated).toLocaleString() : "尚未扫描"} · 新增 {state?.discovery.imported ?? 0} 张。每次限量读取，防止搜索占用训练时间。</p>
      </section>
      {!!state?.discovery.warnings.length && <div className="training-alert"><ul>{state.discovery.warnings.map((w,i) => <li key={i}>{w}</li>)}</ul></div>}
      <section className="training-card"><h2>候选来源与分享码</h2>{!state?.discovery.candidates.length && <p className="training-muted">扫描后在这里查看作者页面、分享码和待解析内容。</p>}{state?.discovery.candidates.map((c,i) => <div key={i} className="training-candidate"><button className="training-link" onClick={() => external(c.url)}>{c.title}</button><p>{c.description}</p><div className="training-code-list">{c.sharecodes?.slice(0,12).map((code,j) => <code key={j}>{code}</code>)}</div>{(c.sharecodes?.length ?? 0)>12 && <small>另有 {c.sharecodes.length-12} 个分享码，查看来源获取完整内容。</small>}</div>)}</section>
      <section className="training-card"><h2>训练方法依据</h2><p>按 VDIM 的六类专项轮换，并采用 MattyOW 的目标成绩与相关关卡编排思路，加入单模块和总时长上限。这是应用的改造方案，阈值百分比不是他的统一处方。</p><div className="training-actions"><button onClick={() => external("https://www.youtube.com/watch?v=ZEH4CfytNyo")}>VDIM 作者介绍</button><button onClick={() => external("https://rawinput.net/resources/threshold")}>Score Threshold</button><button onClick={() => external("https://rawinput.net/resources/speedmatching")}>Speed Matching</button><button onClick={() => external("https://rawinput.net/resources/purreactivity")}>Reactive Tracking</button></div><p className="training-muted">当前版本不提供新的 AI 轨迹诊断；可继续在 History 查看原有轨迹与录像。具体动作判断需要目标数据和时间同步。</p></section>
    </>}

    {tab === "catalog" && <section className="training-card"><div className="training-section-title"><h2>关卡库 <span className="training-muted">{state?.catalog.length ?? 0}</span></h2><input aria-label="搜索关卡" placeholder="搜索名称、分类或来源" value={filter} onChange={e => setFilter(e.target.value)} /></div><p className="training-muted">Benchmark 保留原生版本、难度组、类别和阈值；原生难度组作为参考，不直接等同于场景机制难度。可修正训练能力、难度与家族。名称不同不保证训练内容不同。新图是否仍可下载，需要在 KovaaK’s 中确认。</p><div className="training-table-wrap"><table><thead><tr><th>关卡 / 来源</th><th>能力</th><th>难度</th><th>标签依据</th><th>启用</th><th /></tr></thead><tbody>{catalog.slice(0,100).map(s => <tr key={s.name}><td><strong>{s.name}</strong><small>{s.sources?.[0]?.title}</small></td><td>{labels[s.skill]}</td><td>{s.difficulty} <small>{difficultySources[s.difficultySource ?? (s.classification === "benchmark" ? "benchmark" : s.classification === "manual" ? "manual" : s.difficulty === "unknown" ? "unknown" : "name")]}</small></td><td>{({ inferred: "名称推断", manual: "手动确认", benchmark: "参考分类" } as Record<string,string>)[s.classification]}</td><td>{s.enabled ? "是" : "否"}</td><td><button onClick={() => setEditing({ ...s })}>编辑</button></td></tr>)}</tbody></table></div>{catalog.length>100 && <p className="training-muted">显示前 100 项，请搜索缩小范围。</p>}{!catalog.length && <div className="training-empty">暂无匹配关卡。先发现或导入训练列表。</div>}</section>}

    {editing && <div className="training-modal-backdrop" onClick={() => setEditing(null)}><section role="dialog" aria-modal="true" aria-labelledby="edit-scenario-title" className="training-card training-modal" onClick={e => e.stopPropagation()}><h2 id="edit-scenario-title">编辑关卡</h2><p>{editing.name}</p><label>训练能力<select value={editing.skill} onChange={e => setEditing({ ...editing, skill: e.target.value })}>{Object.entries(labels).filter(([k]) => k !== "auto").map(([k,v]) => <option key={k} value={k}>{v}</option>)}</select></label><label>难度<select value={editing.difficulty} onChange={e => setEditing({ ...editing, difficulty: e.target.value })}>{["unknown","novice","intermediate","advanced"].map(d => <option key={d}>{d}</option>)}</select></label><label>关卡家族<input value={editing.family} onChange={e => setEditing({ ...editing, family: e.target.value })} /></label><label>已确认的原版关卡<input value={editing.variantOf ?? ""} onChange={e => setEditing({ ...editing, variantOf: e.target.value })} placeholder="留空表示尚未确认变体关系" /></label><label>相关 benchmark（每行一套）<textarea value={(editing.relatedBenchmarks ?? []).join("\n")} onChange={e => setEditing({ ...editing, relatedBenchmarks: e.target.value.split("\n").map(x => x.trim()).filter(Boolean) })} placeholder="这里表示相关能力，不表示直接变体" /></label><small>测量关卡所属体系由 benchmark 同步；相关能力和直接变体由你确认后标注。</small><label>预估单局秒数<input type="number" min={10} max={3600} value={editing.seconds} onChange={e => setEditing({ ...editing, seconds: Number(e.target.value) })} /></label><label>使用感受<select value={editing.preference ?? "neutral"} onChange={e => setEditing({ ...editing, preference: e.target.value })}><option value="liked">喜欢</option><option value="neutral">一般</option><option value="disliked">不喜欢</option></select></label><label>体感难度<select value={editing.personalDifficulty ?? "suitable"} onChange={e => setEditing({ ...editing, personalDifficulty: e.target.value })}><option value="easy">偏易</option><option value="suitable">合适</option><option value="hard">偏难</option></select></label><label className="training-checkbox"><input type="checkbox" checked={editing.enabled} onChange={e => setEditing({ ...editing, enabled: e.target.checked })} />允许用于计划</label><div className="training-actions"><button onClick={() => setEditing(null)}>取消</button><button className="training-primary" disabled={!!busy} onClick={() => perform("保存分类", async () => { await call("UpdateTrainingScenario", JSON.stringify(editing)); setEditing(null); })}>保存</button></div></section></div>}
  </div>;
}
