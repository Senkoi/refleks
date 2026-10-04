import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AIMMEOW_MASCOT } from "@/assets";
import { ArrowRight, ChartNoAxesCombined, Check, Clock3, Compass, Download, Library, Play, RefreshCw, Search, SlidersHorizontal, Target, Upload } from "lucide-react";
import { openURL } from "@/shared/lib/api";
import { call, readState, readLiveState, mergeLiveState, planTimeAllocation, type Block, type Preferences, type Scenario, type State } from "./api";
import { catalogPage, filterCatalog, fileLabels, evidenceLabels, type FileFilter, type DifficultyFilter } from "./catalog";
import "./training.css";
import PersonalEvidence from "./PersonalEvidence";
import TrainingHelp from "./TrainingHelp";
import AbilityOverview from "./AbilityOverview";
import { ScenarioHistoryLink } from "@/shared/components/ScenarioHistoryLink";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/components/ui/tabs";

const labels: Record<string, string> = { auto: "自动 · 平衡弱项与练习量", static: "静态点击", dynamic: "动态点击", smooth: "平滑追踪", reactive: "反应追踪", switching: "目标切换", unknown: "待分类" };
const themes: Record<string, string> = { static: "静态点击", dynamic: "动态点击", smooth: "精确追踪", reactive: "反应追踪", switching_speed: "速度切换", switching_evasive: "规避切换" };
const difficultySources: Record<string, string> = { manual: "手动确认", benchmark: "benchmark 等级", name: "名称推断", playlist: "列表等级推断", unknown: "缺少可靠依据" };
const fitLabels: Record<string, string> = { challenging: "对你偏难", suitable: "对你合适", comfortable: "对你偏易", unknown: "个人适配未知" };
const roles: Record<string, string> = { warmup: "热身", practice: "专项练习", explore: "探索", challenge: "进阶挑战", benchmark: "参考测量", assessment: "固定前测 / 复测" };
const statuses: Record<string, string> = { draft: "待开始", running: "训练中", ready: "下一模块待开始", paused: "已暂停", waiting: "模块到时 · 等待本局结束", completed: "已结束" };
const outcomes: Record<string,string> = { pending: "待完成", list_complete: "列表次数完成", threshold: "阈值达标", measured: "已测量", time_limit: "模块到时", session_limit: "总时长到达", skipped: "已跳过", missed: "游戏已进入后续关卡" };
const initial: Preferences = { planningPolicy: "curriculum", minutes: 30, executionMode: "playlist", focus: "auto", difficulty: "any", benchmark: "", variety: .1, thresholdRatio: .9, autoAdvance: false, autoDiscover: true };
function clock(seconds: number) { const s = Math.max(0, Math.floor(seconds)); return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`; }
function external(url: string) { if (/^https:\/\//i.test(url)) openURL(url); }
function selectionHelp(b: Block) {
  if (b.role === "warmup") return "先用这张图进入训练状态。按列表次数完成，把注意力放在动作和控制上。";
  if (b.role === "challenge") return "你近期的表现帮帮瞄瞄尝试更有挑战的练习。按安排的次数完成即可，一次低分不会让你的训练档位立即回退。";
  if (b.role === "explore") return "用少量训练时间尝试当前专项的另一种练习。完成安排的次数后继续列表；这次表现和你的反馈会帮助选择下一次内容。";
  if (b.role === "benchmark") return "这张图用于对照基准测试的分数要求。完成一次即可，不需要反复刷到目标分数。";
  if (b.role === "assessment") return "回到熟悉的图，看看当前表现是否有变化。按安排的次数完成即可，低分也会保留。";
  return "这张图来自本次 VDIM 练习顺序。按列表次数完成，我会结合单局时长和最近练习量安排时间。";
}


const FileDetails = memo(function FileDetails({ scenario: s }: { scenario: Scenario }) {
 return <>
  {!!s.localAssessment?.measurements?.length && <div className="training-file-values">{s.localAssessment.measurements.map((m,i)=><span key={i}>{m.profile || "场景"} · {m.field} = {m.value}</span>)}</div>}
  {!!s.localAssessment?.precisionComparisons?.length && <div className="training-file-values">{s.localAssessment.precisionComparisons.map((m,i)=><span key={i}>相对 <ScenarioHistoryLink name={m.reference} /> / {m.profile}：半径 ×{m.radiusRatio.toFixed(3)} · 精度差 {m.precisionDelta.toFixed(3)}</span>)}<TrainingHelp label="同族精度对照">这些图的玩法和布局相同，主要区别是目标大小。目标更小通常需要更精确的瞄准；这里只比较这一项，不能据此判断整张图对你有多难。</TrainingHelp></div>}
 </>;
});
const CatalogRow = memo(function CatalogRow({scenario:s,localText,onEdit}: {scenario:Scenario;localText:string;onEdit:(s:Scenario)=>void}) {
 return <tr><td><strong><ScenarioHistoryLink name={s.name} /></strong><small>{s.sources?.[0]?.title}</small></td><td>{labels[s.skill]}</td><td>{s.benchmarks?.length ? s.benchmarks.map(m => <small key={`${m.name}/${m.benchmarkId}`}>{m.system || m.name} / {m.nativeDifficulty || "原生难度组未提供"}</small>) : <small>尚无 benchmark 难度锚点</small>}<small>{fileLabels[s.evaluation?.fileStatus ?? "missing"]}</small><small>{evidenceLabels[s.evaluation?.difficultyStatus ?? "unfitted"]}</small>{s.evaluation?.samples && s.evaluation.samples >= 3 ? <small>{fitLabels[s.evaluation.fit] ?? "个人适配未知"} · {s.evaluation.samples} 局可比记录</small> : null}<TrainingHelp label={`${s.name}本地文件`}>{localText}</TrainingHelp><FileDetails scenario={s}/><small>等级标签：{s.difficulty} · {difficultySources[s.difficultySource ?? "unknown"]}</small></td><td>{({ inferred: "名称推断", manual: "手动确认", benchmark: "参考分类", sce_description: "SCE 作者说明" } as Record<string,string>)[s.classification]}</td><td>{s.enabled ? "是" : "否"}</td><td><button onClick={() => onEdit({ ...s })}>编辑</button></td></tr>;
});

export default function TrainingPage() {
  const [state, setState] = useState<State | null>(null);
  const [prefs, setPrefs] = useState(initial);
  const [tab, setTab] = useState("plan");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [notice, setNotice] = useState("");
  const [url, setURL] = useState("");
  const [filter, setFilter] = useState("");
  const [fileFilter, setFileFilter] = useState<FileFilter>("all");
  const [difficultyFilter, setDifficultyFilter] = useState<DifficultyFilter>("all");
  const [page, setPage] = useState(0);
  useEffect(() => setPage(0), [filter, fileFilter, difficultyFilter]);
  const [editing, setEditing] = useState<Scenario | null>(null);
  const hydrated = useRef(false);
  const mounted = useRef(true);
  const stateRef = useRef<State | null>(null);
  const lastFullAt = useRef(0);
  const lastLive = useRef("");
  const fullRequest = useRef(0);
  const refresh = useCallback(async () => {
    const request=++fullRequest.current;
    const s = await readState();
    if (!mounted.current || request!==fullRequest.current) return;
    stateRef.current=s; lastFullAt.current=Date.now();
    setState(s);
    setLoadError("");
    if (!s.initializing && !hydrated.current) { setPrefs({ ...s.preferences, variety: Math.min(s.preferences.variety, .1), planningPolicy: "curriculum", executionMode: "playlist", autoAdvance: false, curriculumId: "", difficulty: "any", benchmark: "", benchmarks: [] }); hydrated.current = true; }
  }, []);
  useEffect(() => {
    mounted.current = true;
    let timer: ReturnType<typeof setTimeout>;
    let cancelled=false;
    const poll = async () => { try {
      if (!stateRef.current) await refresh();
      else {
        const live=await readLiveState();
        if (cancelled || !mounted.current) return;
        if (live.revision!==stateRef.current.revision || (live.plan?.id ?? null)!==(stateRef.current.plan?.id ?? null) || Date.now()-lastFullAt.current>=30000 || live.initializing!==stateRef.current.initializing) await refresh();
        else {
          const key=JSON.stringify(live);
          if (key!==lastLive.current) { lastLive.current=key; const next=mergeLiveState(stateRef.current,live);stateRef.current=next;if(mounted.current)setState(next); }
        }
      }
      if (!cancelled && mounted.current) setLoadError("");
    } catch (e) { if (!cancelled && mounted.current) setLoadError(String(e)); } finally { if (!cancelled && mounted.current) timer = setTimeout(poll, 2000); } };
    void poll();
    return () => { cancelled=true; mounted.current = false; fullRequest.current++; clearTimeout(timer); };
  }, [refresh]);
  const perform = useCallback(async (name: string, fn: () => Promise<unknown>, message = "") => {
    if (busy) return; setBusy(name); setError(""); setNotice("");
    try { await fn(); await refresh(); if (mounted.current && message) setNotice(message); }
    catch (e) { if (mounted.current) setError(String(e)); }
    finally { if (mounted.current) setBusy(""); }
  }, [busy, refresh]);
  const recordFeedback = useCallback((id: string, value: string) => {
    void perform("保存试练反馈", () => call("RecordTrainingTrialFeedback", id, value), "收到你的感受啦喵，我会用它安排后面的练习。");
  }, [perform]);
  const plan = state?.initializing ? null : state?.plan;
  const allocation = useMemo(() => planTimeAllocation(plan?.blocks ?? []), [plan?.blocks]);
  const active = !!plan && ["running", "ready", "paused", "waiting"].includes(plan.status);

  const catalog = useMemo(()=>filterCatalog(state?.catalog ?? [], filter, fileFilter, difficultyFilter),[state?.catalog,filter,fileFilter,difficultyFilter]);
  const visible = catalogPage(catalog, page);
  const enabled = useMemo(()=>state?.catalog.filter(s => s.enabled).length ?? 0,[state?.catalog]);
  const byName=useMemo(()=>new Map((state?.catalog ?? []).map(s=>[s.name.toLowerCase(),s])),[state?.catalog]);
  const current = plan?.blocks[plan.index];
  const localStatus = (name: string) => { const status = byName.get(name.toLowerCase())?.localAssessment?.status; if (status === "file_parsed_model_unfitted") return "已经读到这张图的本地场景文件，可以查看目标大小等信息。对你是否合适，还要结合实际训练成绩判断。"; if (status === "ambiguous_local_versions") return "电脑里找到了多个同名但内容不同的版本。我暂时无法确认使用哪一个，因此不会用这些文件判断难度；你仍可在游戏里正常练习。"; if (status === "local_file_unavailable") return "之前读到的场景文件现在找不到了，旧的文件判断暂时不再使用。你再次加载这张图后，我会重新检查。"; if (status === "invalid_local_file") return "暂时无法读取这张图的场景文件。你可以正常训练，我仍会参考已经记录的成绩。"; return "还没有读到这张图的本地场景文件。可以先在游戏里练习；游戏保存文件后，我会自动检查，不需要你另行上传或下载。"; };
  const pset = <K extends keyof Preferences>(key: K, value: Preferences[K]) => setPrefs(p => ({ ...p, [key]: value }));

  const settingsLocked = active || !!busy || !!state?.initializing;
  return <Tabs value={tab} onValueChange={setTab} className="training-page">
    <header className="training-header">
      <div className="training-brand"><img src={AIMMEOW_MASCOT} alt="" aria-hidden="true" width={88} height={88} /><div><div className="training-eyebrow">瞄瞄 / AIMMEOW</div><h1>训练工作台</h1></div></div>
      <div className="training-library-count"><Compass size={20} /><strong>{enabled}</strong><span>可编排关卡</span></div>
    </header>
    <TabsList className="training-tabs" aria-label="训练工作台页面">
      {[{ id: "plan", title: "当次计划", icon: Target }, { id: "evidence", title: "训练评估", icon: ChartNoAxesCombined }, { id: "discover", title: "发现内容", icon: Compass }, { id: "catalog", title: "关卡库", icon: Library }].map(({ id, title, icon: Icon }) => <TabsTrigger key={id} value={id}><Icon size={16} aria-hidden="true" /><span>{title}</span></TabsTrigger>)}
    </TabsList>
    {loadError && <div role="alert" className="training-alert error">{loadError}</div>}
    {state?.initializing && <div role="status" className="training-alert">正在读取已有 benchmark 成绩与近 45 天训练历史，完成后自动生成并安装列表…</div>}
    {!state?.initializing && state?.notice && <div role="status" className="training-alert">{state.notice}</div>}
    {error && <div role="alert" className="training-alert error">{error}</div>}
    {state?.error && <div role="alert" className="training-alert error">{state.error}</div>}
    {notice && <div role="status" className="training-alert">{notice}</div>}
    {busy && <div role="status" className="training-alert"><RefreshCw size={14} className="animate-spin" /> {busy}…{busy === "发现内容" && " 正在读取多个公开来源，可能需要约 2 分钟。"}</div>}

    <TabsContent value="evidence"><PersonalEvidence anchors={state?.personalAnchors ?? []} studies={state?.trainingStudies ?? []} evaluations={state?.anchorEvaluations ?? []} busy={!!busy} onFeedback={recordFeedback} /></TabsContent>
    <TabsContent value="plan">
      <div className="training-columns">
        <section className="training-card training-config">
          <h2><SlidersHorizontal size={18} /> 今天怎么练 <TrainingHelp label="计划编排"><p>告诉我这次能练多久，我会结合已有成绩和最近练得较少的能力来安排。记录还少的话，我们先从基础模板开始喵。</p><p>上次还没练完、又没超过 24 小时，我会先帮你接上。想练某一项也能指定；交给我时，我会平衡各类练习。</p><p>列表生成后，我们就按顺序练。我会把新成绩留给下一次安排，让你安心练完这一份。</p></TrainingHelp></h2>
          <fieldset>
            <label>可用时间 <TrainingHelp label="可用时间">这是本次训练能用的总时间，包含切图和休息。我会据此安排关卡与局数；需要离开时可以暂停计时。</TrainingHelp><div className="training-number"><input aria-label="可用时间" disabled={settingsLocked} type="number" min={5} max={120} value={prefs.minutes} onChange={e => pset("minutes", Number(e.target.value))} /><span>分钟</span></div></label>
            <div className="training-muted">VDIM · {state?.curricula?.length ?? 0} 套模板</div>
            <label>训练重点 <TrainingHelp label="训练重点">选择你今天想重点练的能力，或保留“自动”，让应用结合已有成绩和近期训练量安排。未完成的列表在 24 小时内会优先续接。</TrainingHelp><select disabled={settingsLocked} value={prefs.focus} onChange={e => pset("focus", e.target.value)}>{Object.entries(labels).filter(([k]) => k !== "unknown").map(([k,v]) => <option key={k} value={k}>{v}</option>)}</select></label>
            <label>探索比例 <TrainingHelp label="探索比例">决定本次愿意留多少练习时间尝试新图，最多 10%。设为 0 就不安排额外探索。首次会先练基础列表，之后再加入相关的新练习；进阶挑战由近期表现另行安排，不占这个探索比例。</TrainingHelp><strong>{Math.round(prefs.variety * 100)}%</strong><input aria-label="探索比例" disabled={settingsLocked} type="range" min={0} max={.1} step={.01} value={prefs.variety} onChange={e => pset("variety", Number(e.target.value))} /></label>
            <label className="training-checkbox"><input type="checkbox" disabled={settingsLocked} checked={prefs.autoDiscover} onChange={e => pset("autoDiscover", e.target.checked)} /> 应用运行时每周自动发现内容 <TrainingHelp label="自动发现内容">打开后，应用运行期间约每周检查一次新的公开训练资源。发现内容先加入关卡库或候选来源，再按训练目标筛选；不会直接改掉当前列表。你也可以在“发现内容”页面手动扫描。</TrainingHelp></label>
            <button className="training-primary" disabled={!state || !!busy || active || !!state?.initializing} onClick={() => perform("生成计划", async () => { await call("GenerateTrainingPlan", JSON.stringify({ ...prefs, curriculumId: "", difficulty: "any", benchmark: "", benchmarks: [], planningPolicy: "curriculum", executionMode: "playlist", autoAdvance: false })); const path = await call<string>("InstallTrainingPlaylist"); setNotice(`本次列表放好啦喵：${path}。后续会复用这个位置。重启 KovaaK’s 后，在 Local Playlists 中打开刚安装的本次列表。`); })}><Target size={16} />生成并安装本次列表 <ArrowRight size={16} /></button>
          </fieldset>
          {!state?.curricula?.length && <p className="training-muted">正在读取自动初始化的 VDIM 模板；若初始化失败，生成时会显示具体原因。</p>}
        </section>
        <section className="training-main">
          <AbilityOverview levels={state?.playerLevels} tiers={state?.templateTiers} />

          <div className="training-card">
            <div className="training-section-title"><h2><Clock3 size={18} /> {plan ? statuses[plan.status] : "准备好，再开始"} <TrainingHelp label="计划执行"><p>{plan?.preferences.executionMode === "playlist" ? "在 KovaaK’s 的 Local Playlists 中打开刚安装的本次列表，按列表训练；在这里开始计时。完整对局的成绩会自动记录，游戏负责切换关卡。重新开始、切图和休息时计时仍会继续，需要离开时请暂停。提醒不会中断游戏；无边框窗口通常能看到提示，全屏独占时可能只有声音。" : "完成对局后，成绩会自动记录。切图和休息也计入本次时间；需要离开时，请用暂停按钮停下工作台计时。"}</p>{plan && <><p>{plan.theme ? `本次重点是${themes[plan.theme] ?? plan.theme}，训练模板参考 ${plan.templateTier || plan.playerTier || "Novice"} 档。` : "我会按你的可用时间安排这次练习。"}</p><p>24 小时内会优先继续未完成的列表。一个模块练几局，取决于单局时长、最近练习量和本次可用时间；训练档位只是选择练习的参考，不代表官方段位。</p></>}</TrainingHelp></h2>{plan && <span className="training-badge">{plan.preferences.minutes} 分钟预算</span>}</div>
            {plan ? <>
              {plan.curriculumName && <p className="training-muted">原模板：{plan.curriculumName} · 第 {(plan.curriculumStart ?? 0) + 1}–{plan.curriculumEnd ?? plan.blocks.length} / {plan.curriculumTotal ?? plan.blocks.length} 行</p>}
              {plan.theme && <p className="training-muted">本次 VDIM 专项：{themes[plan.theme] ?? plan.theme}{plan.playerTier && ` · 模板选择参考 ${plan.playerTier}`}{plan.templateTier && ` · 模板档位 ${plan.templateTier}`}</p>}
              <p className="training-muted">本次安排：主线 {clock(allocation.main)} · 评估补充 {clock(allocation.assessment)} · 进阶挑战 {clock(allocation.challenge)} · 探索试练 {clock(allocation.explore)}</p>
              <div className="training-metrics"><div><strong>{clock(Math.max(0, plan.preferences.minutes * 60 - plan.elapsed))}</strong><span>剩余时间</span></div><div><strong>{clock(plan.recorded)}</strong><span>已记录练习</span></div><div><strong>{plan.blocks.filter(b => b.outcome !== "pending").length} / {plan.blocks.length}</strong><span>训练模块</span></div></div>
              <div className="training-progress"><span style={{ width: `${Math.min(100, plan.elapsed / (plan.preferences.minutes * 60) * 100)}%` }} /></div>
              <div className="training-actions">
                <button disabled={!!busy} onClick={() => perform("测试提醒", () => call("TestTrainingReminder"), "10 秒后显示提醒。现在切回 KovaaK’s，检查游戏画面和声音。")}>测试游戏内提醒</button>
                {plan.preferences.executionMode === "playlist" && <button disabled={!!busy} onClick={() => perform("安装列表", async () => { const path = await call<string>("InstallTrainingPlaylist"); setNotice(`本次列表放好啦喵：${path}。后续会复用这个位置。重启 KovaaK’s 后从 Local Playlists 打开。`); })}>安装到 KovaaK’s</button>}
                {["draft","ready","paused"].includes(plan.status) && <button className="training-primary" disabled={!!busy} onClick={() => perform(plan.preferences.executionMode === "playlist" ? "开始计时" : "启动关卡", () => call("TrainingAction", "start"))}><Play size={15} />{plan.preferences.executionMode === "playlist" ? (plan.status === "paused" ? "继续列表计时" : "开始列表计时") : (plan.status === "paused" ? "继续并启动当前关卡" : "开始当前模块")}</button>}
                {["running","ready","waiting"].includes(plan.status) && <button disabled={!!busy} onClick={() => perform("暂停", () => call("TrainingAction", "pause"))}>暂停计时</button>}
                {["running","paused","waiting"].includes(plan.status) && <button disabled={!!busy} onClick={() => perform("跳过模块", () => call("TrainingAction", "next"))}>{plan.status === "waiting" ? "本局已结束，继续" : "跳过模块"}</button>}
                {active && <button disabled={!!busy} onClick={() => perform("结束训练", () => call("TrainingAction", "finish"))}>结束本次</button>}
                <button disabled={!!busy} onClick={() => perform("导出列表", () => call("ExportTrainingPlaylist"), "已导出列表，重复次数与本次计划一致。")}><Download size={15} />导出列表</button>
              </div>
              {current && active && <p className="training-current">当前记录模块：<ScenarioHistoryLink name={current.scenario.name} /> · 计时 {clock(plan.blockElapsed)} / {clock(current.budget)}</p>}
              {plan.reminder && <div role="alert" className="training-alert">{plan.reminder}</div>}

            </> : <div className="training-empty"><Target size={35} /><p>告诉我时间和重点，我们就可以开始一起练啦喵。</p></div>}
          </div>
          {plan?.blocks.map((b,i) => <article key={`${plan.id}-${i}`} className={`training-card training-block ${i === plan.index && active ? "current" : ""}`}>
            <div className="training-block-number">{b.outcome !== "pending" ? <Check size={17} /> : String(i + 1).padStart(2,"0")}</div>
            <div className="training-block-body">
              <div className="training-section-title">
                <span className="training-eyebrow">{roles[b.role]} · {labels[b.scenario.skill]}</span>
                <span className="training-badge">{plan.preferences.executionMode === "playlist" ? `${b.playCount || 1} 局 · 约 ${clock(b.budget)}` : `上限 ${clock(b.budget)}`}</span>
              </div>
              <div className="training-inline-title"><h3><ScenarioHistoryLink name={b.scenario.name} /></h3><TrainingHelp label={`${b.scenario.name}选图与评估`}>
                <p>{selectionHelp(b)}</p>
                {b.personalization?.relation && <p>这张图与参照图的玩法、布局相同，主要改变了目标大小：{b.personalization.relation.profiles.map(p=>`${p.profile} 为原来的 ${(p.radiusRatio*100).toFixed(0)}%`).join("；")}。可以用来尝试不同的精度要求；两张图的分数不能直接相减当成进步。</p>}
                {b.personalization?.prediction && <p>{b.personalization.prediction.status === "local_backtest" ? `根据你之前的类似练习，预计成绩约 ${b.personalization.prediction.expectedScore?.toFixed(1)}；这只是参考，不是必须达到的目标。` : "目前还不能可靠预测你在这张图上的分数。先按安排的次数试练，我会记录表现，帮助选择下一次内容。"}</p>}
                <p>{localStatus(b.scenario.name)}</p>
                {b.benchmark && <p>相关基准测试：{b.benchmark}</p>}
                {b.difficultyEvidence && <p>场景难度标签：{({unknown:"尚未确认",novice:"基础",intermediate:"中等",advanced:"进阶"} as Record<string,string>)[b.difficultyEvidence.level] ?? b.difficultyEvidence.level} · {difficultySources[b.difficultyEvidence.source] ?? "来源未知"}。这里的标签与“对你合适 / 偏难”是两种判断，个人感受以实际成绩和反馈为参考。</p>}
                <ul className="training-references">{Array.from(new Map((b.difficultyEvidence?.benchmarks ?? []).map(m => [`${m.name}/${m.benchmarkId}/${m.category}/${m.group}`, m])).values()).map((m, i) => <li key={i}>{m.name} · {[m.category, m.group].filter(Boolean).join(" / ")}</li>)}</ul>
                {b.timing && <p>单局约 {clock(b.timing.seconds)} · {b.timing.source === "history" ? `参考你的 ${b.timing.samples} 局记录` : b.timing.source === "default" ? "还没有足够记录，暂用默认时长" : "还没有足够记录，暂用关卡时长"} · 近 24 小时已练 {clock(b.timing.recentSeconds)}</p>}
                <div className="training-reference-links">{b.scenario.sources?.map((s,i) => /^https:\/\//i.test(s.url) && <button className="training-link" key={i} onClick={() => external(s.url)}>{s.title}</button>)}</div>
</TrainingHelp></div>
              <p>{b.cue}</p>
              {b.personalization && <p className="training-fit">个人锚点：<ScenarioHistoryLink name={b.personalization.anchor.scenario} known /> · 中位 {b.personalization.anchor.medianScore.toFixed(1)} · {b.personalization.anchor.days} 天 / {b.personalization.anchor.samples} 局</p>}
              {b.difficultyEvidence && <p className="training-fit">{fitLabels[b.difficultyEvidence.fit] ?? "个人适配未知"}{b.difficultyEvidence.samples >= 3 && ` · 近 ${b.difficultyEvidence.windowDays ?? 45} 天 / ${b.difficultyEvidence.samples} 局`}{b.anchorScenario && <> · 训练目标：<ScenarioHistoryLink name={b.anchorScenario} /></>}</p>}
              {b.difficultyEvidence?.trend && b.difficultyEvidence.trend !== "insufficient" && <p>训练表现：{({improving:"持续改善",stable:"近期稳定",declining:"多次训练下降"} as Record<string,string>)[b.difficultyEvidence.trend] ?? "待观察"} · {b.difficultyEvidence.trendSessions} 次可比训练</p>}
              <div className="training-block-footer">
                <span>{b.assessment ? `一局熟悉 · 三局测量；主线 ${b.assessment.mainPlayCount} 局，补充 ${b.assessment.extraRuns} 局` : b.target > 0 ? `目标 ≥ ${b.target.toFixed(1)}` : b.role === "assessment" ? "早期安排：一局熟悉 · 两局测量" : b.role === "benchmark" ? "固定完成一次" : "不设分数门槛"}</span>
                <span>{b.runs} 局 · {clock(b.recorded)} · {outcomes[b.outcome]}</span>
                <button className="training-link" onClick={() => setEditing({ ...b.scenario })}>评价关卡</button>
              </div>
            </div>
          </article>)}
          {!!plan?.warnings.length && <div className="training-alert"><ul>{plan.warnings.map(w => <li key={w}>{w}</li>)}</ul></div>}
        </section>
      </div>
      <section className="training-card"><h2>近 7 天训练分布</h2><div className="training-skill-grid">{state?.skills.map(s => <div key={s.skill}><strong>{labels[s.skill]}</strong><b>{s.minutes.toFixed(1)} <small>分钟</small></b><span>{s.samples} 条记录</span><TrainingHelp label={`${labels[s.skill]}训练时间`}>这里汇总最近七天已经记录的对局时间，帮助你看出哪些能力练得较多、哪些练得较少。它反映练习量，不表示能力高低；我会把它作为安排下一次训练的参考。</TrainingHelp></div>)}</div></section>
      {!!state?.history.length && <section className="training-card"><h2>近期计划</h2>{state.history.slice(-5).reverse().map(p => <p key={p.id}>{new Date(p.created).toLocaleString()} · {p.preferences.minutes} 分钟预算 · 已记录 {clock(p.recorded)} · {statuses[p.status]}</p>)}</section>}
    </TabsContent>

    <TabsContent value="discover">
      <section className="training-card"><h2><Compass size={18} /> 发现新的训练内容 <TrainingHelp label="发现内容">查找公开社区训练列表，把能够识别的关卡加入关卡库，再按你的训练目标筛选。作者页面和暂时读不出关卡的分享码会留在“候选来源”，方便你查看。当前扫描的是社区资源，尚未接入创意工坊热门榜。</TrainingHelp></h2><div className="training-actions"><button className="training-primary" disabled={!!busy} onClick={() => perform("发现内容", () => call("DiscoverTrainingContent"), "发现完成；分类由名称推断，未知分类不会自动进入计划。 ")}><Search size={16} />扫描公开来源{state?.searchConfigured ? "与全网" : ""}</button><button disabled={!!busy} onClick={() => perform("同步 benchmark", () => call("ImportTrainingBenchmarks"), "已导入可用 benchmark 定义。 ")}><RefreshCw size={16} />同步 benchmark</button><button disabled={!!busy} onClick={() => perform("导入本地列表", () => call("ImportTrainingPlaylist"))}><Upload size={16} />导入 JSON / PLO</button></div><div className="training-source-input"><input aria-label="训练来源网址" placeholder="粘贴公开训练列表或 JSON 地址" value={url} onChange={e => setURL(e.target.value)} /><button disabled={!!busy || !url.trim()} onClick={() => perform("读取来源", () => call("ImportTrainingSource", url.trim()))}>读取来源</button></div>
        <div className="training-search-status"><span className={`training-dot ${state?.searchConfigured ? "configured" : ""}`} />{state?.searchConfigured ? "已检测到全网搜索配置" : "公开资源扫描可用 · 全网搜索待配置"}<TrainingHelp label="全网搜索"><p>不配置额外服务也能扫描 Ridd、4BK 和公开 GitHub 训练资源。全网搜索用于查找更多作者页面和训练列表，不是正常训练的必需项。</p><p>搜索只发送训练主题，不会上传你的成绩或录像。</p><p>如果你已有 Brave Search API Key，可在启动应用前设置 <code>AIMMEOW_BRAVE_API_KEY</code>，然后重启。未配置时，社区资源扫描仍可正常使用。</p></TrainingHelp></div>

        <p className="training-muted">最近发现：{state?.discovery.updated ? new Date(state.discovery.updated).toLocaleString() : "尚未扫描"} · 新增 {state?.discovery.imported ?? 0} 张</p>
      </section>
      {!!state?.discovery.warnings.length && <div className="training-alert"><ul>{state.discovery.warnings.map((w,i) => <li key={i}>{w}</li>)}</ul></div>}
      <section className="training-card"><h2>候选来源与分享码</h2>{!state?.discovery.candidates.length && <p className="training-muted">我找到的新资源会放在这里，作者页面和分享码都帮你留好喵。</p>}{state?.discovery.candidates.map((c,i) => <div key={i} className="training-candidate"><button className="training-link" onClick={() => external(c.url)}>{c.title}</button><p>{c.description}</p><div className="training-code-list">{c.sharecodes?.slice(0,12).map((code,j) => <code key={j}>{code}</code>)}</div>{(c.sharecodes?.length ?? 0)>12 && <small>另有 {c.sharecodes.length-12} 个分享码，查看来源获取完整内容。</small>}</div>)}</section>
      <section className="training-card"><h2>训练方法依据 <TrainingHelp label="训练方法"><p>VDIM 提供六类专项练习的顺序和搭配。我会参考这些安排，再结合你的可用时间和成绩，选择适合的一段并加入少量变化。</p><p>下面的链接介绍训练思路，可以按兴趣阅读，不需要先学完才能使用工作台。想回看具体对局时，可去 History 查看已有轨迹与录像。</p></TrainingHelp></h2><div className="training-actions"><button onClick={() => external("https://www.youtube.com/watch?v=ZEH4CfytNyo")}>VDIM 作者介绍</button><button onClick={() => external("https://rawinput.net/resources/threshold")}>Score Threshold</button><button onClick={() => external("https://rawinput.net/resources/speedmatching")}>Speed Matching</button><button onClick={() => external("https://rawinput.net/resources/purreactivity")}>Reactive Tracking</button></div></section>
    </TabsContent>

    <TabsContent value="catalog"><section className="training-card"><div className="training-section-title"><h2>关卡库 <TrainingHelp label="关卡库评估"><p>我把可以安排的关卡都收在这里啦。按名称、能力或来源找找，还能看看下载状态和你以前的评价。</p><p>“配置验证通过”表示本地文件能够读取、必要内容齐全，还需要游戏实际运行才能确认可玩。“同家族精度对照”主要比较相似图的目标大小；“Benchmark 参考”表示它有基准测试的分数要求。</p><p>目前还不能给所有关卡一个统一的难度分。如果显示“尚无难度评估依据”，可以先正常训练，再用记录的成绩和你的反馈判断是否合适。新图能否下载，以游戏内结果为准。</p></TrainingHelp><span className="training-muted">{catalog.length} / {state?.catalog.length ?? 0}</span></h2><input aria-label="搜索关卡" placeholder="搜索名称、分类或来源" value={filter} onChange={e => setFilter(e.target.value)} /></div><div className="training-actions"><button disabled={!!busy} onClick={() => perform("扫描本地场景", () => call("RefreshTrainingLocal"), "本地扫描已执行；新下载文件稳定后会在约 15 秒内自动评估。")}>扫描已下载场景</button><span className="training-muted">已解析 {(state?.catalog ?? []).filter(s => s.localAssessment?.status === "file_parsed_model_unfitted").length} 张 SCE</span></div><div className="training-catalog-filters"><label>本地文件<select aria-label="筛选本地文件" value={fileFilter} onChange={e => setFileFilter(e.target.value as FileFilter)}><option value="all">全部文件状态</option><option value="verified">已下载 · 配置验证通过</option><option value="parsed">已解析（含待验证）</option><option value="issues">待验证 / 异常</option><option value="missing">本地文件不可用</option></select></label><label>难度依据<select aria-label="筛选难度依据" value={difficultyFilter} onChange={e => setDifficultyFilter(e.target.value as DifficultyFilter)}><option value="all">全部评估状态</option><option value="evidence">有难度评估依据</option><option value="precision">同家族精度对照</option><option value="benchmark">原生 benchmark 参考</option><option value="calibrated">总难度已标定</option><option value="unfitted">尚无难度评估依据</option></select></label><button onClick={() => { setFileFilter("verified"); setDifficultyFilter("evidence"); }}>已下载且有难度依据</button><button onClick={() => { setFileFilter("all"); setDifficultyFilter("all"); setFilter(""); }}>清除筛选</button></div><div className="training-table-wrap"><table><thead><tr><th>关卡 / 来源</th><th>能力</th><th>难度参考 / 文件评估</th><th>标签依据</th><th>启用</th><th /></tr></thead><tbody>{visible.rows.map(s => <CatalogRow key={s.name} scenario={s} localText={localStatus(s.name)} onEdit={setEditing}/>)}</tbody></table></div>{catalog.length > 0 && <div className="training-actions"><button disabled={visible.page === 0} onClick={() => setPage(visible.page - 1)}>上一页</button><span>第 {visible.page + 1} / {visible.pages} 页 · 共 {catalog.length} 项</span><button disabled={visible.page + 1 >= visible.pages} onClick={() => setPage(visible.page + 1)}>下一页</button></div>}{!catalog.length && <div className="training-empty">暂无符合筛选条件的关卡。可清除筛选或扫描已下载场景；总难度尚未标定时，该筛选结果为空。</div>}</section></TabsContent>

    {editing && <div className="training-modal-backdrop" onClick={() => setEditing(null)}><section role="dialog" aria-modal="true" aria-labelledby="edit-scenario-title" className="training-card training-modal" onClick={e => e.stopPropagation()}><h2 id="edit-scenario-title">编辑关卡</h2><p><ScenarioHistoryLink name={editing.name} /></p><label>训练能力<select value={editing.skill} onChange={e => setEditing({ ...editing, skill: e.target.value })}>{Object.entries(labels).filter(([k]) => k !== "auto").map(([k,v]) => <option key={k} value={k}>{v}</option>)}</select></label><label>难度<select value={editing.difficulty} onChange={e => setEditing({ ...editing, difficulty: e.target.value })}>{["unknown","novice","intermediate","advanced"].map(d => <option key={d}>{d}</option>)}</select></label><label>关卡家族<input value={editing.family} onChange={e => setEditing({ ...editing, family: e.target.value })} /></label><label>已确认的原版关卡<input value={editing.variantOf ?? ""} onChange={e => setEditing({ ...editing, variantOf: e.target.value })} placeholder="留空表示尚未确认变体关系" /></label>{editing.variantOf && <p>原版关卡：<ScenarioHistoryLink name={editing.variantOf} /></p>}<label>相关 benchmark（每行一套） <TrainingHelp label="相关 benchmark">填写与这张图训练目标相关的 Benchmark，每行一个。这样可以补充它适合练什么；不会把它当成该测试的正式测量图，也不会改变测试分数线。若要注明它是谁的直接变体，请填写上面的“已确认的原版关卡”。</TrainingHelp><textarea value={(editing.relatedBenchmarks ?? []).join("\n")} onChange={e => setEditing({ ...editing, relatedBenchmarks: e.target.value.split("\n").map(x => x.trim()).filter(Boolean) })} placeholder="这里表示相关能力，不表示直接变体" /></label><label>预估单局秒数<input type="number" min={10} max={3600} value={editing.seconds} onChange={e => setEditing({ ...editing, seconds: Number(e.target.value) })} /></label><label>使用感受<select value={editing.preference ?? "neutral"} onChange={e => setEditing({ ...editing, preference: e.target.value })}><option value="liked">喜欢</option><option value="neutral">一般</option><option value="disliked">不喜欢</option></select></label><label>体感难度<select value={editing.personalDifficulty ?? ""} onChange={e => setEditing({ ...editing, personalDifficulty: e.target.value })}><option value="">未评价</option><option value="easy">偏易</option><option value="suitable">合适</option><option value="hard">偏难</option></select></label><label className="training-checkbox"><input type="checkbox" checked={editing.enabled} onChange={e => setEditing({ ...editing, enabled: e.target.checked })} />允许用于计划</label><div className="training-actions"><button onClick={() => setEditing(null)}>取消</button><button className="training-primary" disabled={!!busy} onClick={() => perform("保存分类", async () => { await call("UpdateTrainingScenario", JSON.stringify(editing)); setEditing(null); })}>保存</button></div></section></div>}
  </Tabs>;
}
