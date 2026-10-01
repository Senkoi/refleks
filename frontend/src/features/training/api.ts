export type Scenario = {
  name: string; skill: string; family: string; difficulty: string; seconds: number;
  benchmark: string; thresholds?: number[]; benchmarks?: { name: string; thresholds?: number[] }[];
  relatedBenchmarks?: string[]; variantOf?: string; preference?: string; personalDifficulty?: string;
  classification: string; enabled: boolean;
  sources: { url: string; title: string; retrieved: string }[];
};
export type Preferences = {
  minutes: number; executionMode: "playlist" | "adaptive"; focus: string; difficulty: string; benchmark: string; benchmarks?: string[];
  variety: number; thresholdRatio: number; autoAdvance: boolean; autoDiscover: boolean;
};
export type Block = {
  timing?: { seconds: number; source: string; samples: number; recentSeconds: number; weeklySeconds: number };
  scenario: Scenario; role: string; budget: number; playCount: number; target: number; reason: string; benchmark?: string;
  cue: string; recorded: number; runs: number; best: number; outcome: string;
};
export type Plan = {
  id: string; created: string; preferences: Preferences; blocks: Block[]; warnings: string[];
  status: string; index: number; elapsed: number; recorded: number; blockElapsed: number;
  reminder?: string;
};
export type State = {
  version: number; catalog: Scenario[]; preferences: Preferences; plan: Plan | null;
  history: Plan[]; searchConfigured: boolean; error: string;
  skills: { skill: string; priority: number; minutes: number; samples: number; evidence: string }[];
  discovery: { updated: string; imported: number; warnings: string[];
    candidates: { title: string; url: string; description: string; sharecodes: string[] }[] };
};

// Separate primitive-only bridge: Wails discovers the exported Go methods at build time.
type Bridge = Record<string, (...args: string[]) => Promise<unknown>>;
export async function call<T>(method: string, ...args: string[]): Promise<T> {
  const bridge = (window as unknown as { go?: { main?: { App?: Bridge } } }).go?.main?.App;
  if (!bridge?.[method]) throw new Error("请在 RefleK’s 桌面应用中打开训练模块；浏览器预览无法读取本机训练数据。");
  return await bridge[method](...args) as T;
}
export async function readState(): Promise<State> {
  const s = JSON.parse(await call<string>("GetTrainingState")) as State;
  s.catalog ??= []; s.skills ??= []; s.history ??= [];
  s.discovery.candidates ??= []; s.discovery.warnings ??= [];
  return s;
}
