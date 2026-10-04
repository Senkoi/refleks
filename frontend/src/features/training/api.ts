export type Scenario = {
  name: string; skill: string; technique?: string; family: string; difficulty: string; difficultySource?: string; seconds: number;
  benchmark: string; thresholds?: number[]; benchmarks?: { name: string; benchmarkId?: number; system?: string; nativeDifficulty?: string; category?: string; group?: string; thresholds?: number[]; ranks?: string[] }[];
  relatedBenchmarks?: string[]; variantOf?: string; preference?: string; personalDifficulty?: string;
  classification: string; enabled: boolean;
  evaluation?: { fileStatus: string; difficultyStatus: string; hasDifficultyEvidence: boolean; hasPrecisionReference: boolean; hasBenchmarkReference: boolean; fit: string; samples: number };
  localAssessment?: { status: string; fileSHA256?: string; observedAt?: string; issues?: string[]; measurements?: { profile?: string; field: string; value: number; unit: string; line: number }[]; precisionComparisons?: { reference: string; profile: string; radiusRatio: number; precisionDelta: number }[] };
  mechanics?: { fileSHA256: string; tags: string[]; status: string; role: string; geometryStatus: string; angularSize: number | null; transitionAngle: number | null };
  sources: { url: string; title: string; retrieved: string }[];
};
export type Preferences = {
  planningPolicy?: "curriculum" | "legacy"; curriculumId?: string;
  minutes: number; executionMode: "playlist" | "adaptive"; focus: string; difficulty: string; benchmark: string; benchmarks?: string[];
  variety: number; thresholdRatio: number; autoAdvance: boolean; autoDiscover: boolean;
};
export type Block = {
	assessment?: { id: string; protocolId: string; mainPlayCount: number; extraRuns: number };
  personalization?: { source: string; anchor: PersonalAnchor; relation?: { direction: string; uniform: boolean; maxDelta: number; profiles: { profile: string; radiusRatio: number; precisionDelta: number }[] }; prediction?: { status: string; expectedScore?: number; samples: number; days: number; validationMAE?: number; baselineMAE?: number } };
  measurement?: { studyId: string; phase: string };
  timing?: { seconds: number; source: string; samples: number; recentSeconds: number; weeklySeconds: number };
  difficultyEvidence?: { level: string; source: string; fit: string; samples: number; windowDays?: number; trend?: string; trendSessions?: number; recentScore?: number; benchmarks?: Scenario["benchmarks"] };
  anchorScenario?: string; scenario: Scenario; role: string; budget: number; playCount: number; sourcePlayCount?: number; target: number; reason: string; benchmark?: string;
  cue: string; recorded: number; runs: number; best: number; outcome: string;
};
export type Plan = {
  selectionReason?: string;
	tierReason?: string;
  playerTier?: string; templateTier?: string; curriculumId?: string; curriculumName?: string; curriculumStart?: number; curriculumEnd?: number; curriculumTotal?: number; id: string; created: string; theme?: string; preferences: Preferences; blocks: Block[]; warnings: string[];
  status: string; index: number; elapsed: number; recorded: number; blockElapsed: number;
  reminder?: string;
};

export function planTimeAllocation(blocks: Block[]) {
 const time = { main: 0, assessment: 0, challenge: 0, explore: 0 };
 for (const b of blocks) {
  if (b.role === "assessment") time.assessment += b.budget;
  else if (b.role === "challenge") time.challenge += b.budget;
  else if (b.role === "explore") time.explore += b.budget;
  else {
   const extra = Math.min(b.budget, (b.assessment?.extraRuns ?? 0) * (b.timing?.seconds ?? b.budget / Math.max(1,b.playCount)));
   time.main += b.budget - extra;
   time.assessment += extra;
  }
 }
 return time;
}
export type State = {
	anchorEvaluations?: AnchorEvaluation[];
  personalAnchors?: PersonalAnchor[];
  trainingStudies?: TrainingStudy[];
  revision?: number;
  themePriorities?: Record<string,{ priority: number; minutes: number; level?: number; evidence: string }>;
  templateTiers?: Record<string, string>;
  initializing?: boolean; notice?: string;
  playerLevels?: { trainingTier?: string; ability?: number; source?: string; windowDays?: number; lastPlayed?: string; theme: string; category?: string; group?: string; system: string; nativeDifficulty: string; rank?: string; tier: string; status: string; scenarios: number; required: number; samples: number; evidence: string }[];
  curricula?: { id: string; name: string; theme: string; tier?: string; rows: { scenarioName: string; playCount: number }[] }[];
  version: number; catalog: Scenario[]; preferences: Preferences; plan: Plan | null;
  history: Plan[]; searchConfigured: boolean; error: string;
  skills: { skill: string; priority: number; minutes: number; samples: number; evidence: string }[];
  discovery: { updated: string; imported: number; warnings: string[];
    candidates: { title: string; url: string; description: string; sharecodes: string[] }[] };
};

export type PersonalAnchor = {
 scenario: string; theme: string; status: string; evidence: string; medianScore: number; scoreMAD: number;
 accuracy?: number; hitsPerSecond?: number; samples: number; sessions: number; days: number; lastPlayed: number;
 points?: { at: number; score: number; samples: number }[];
};
type MeasurementResult = { score: number; samples: number; at: number; accuracy?: number; hitsPerSecond?: number; scores?: number[]; protocolId?: string };
export type AnchorEvaluation = {
 id: string; planId: string; theme: string; scenario: string; protocolId: string; status: string; reason: string;
 createdAt: number; startedAt?: number; extraRuns: number; extraSeconds: number; result?: MeasurementResult; previous?: MeasurementResult;
 change?: number; intervalHours?: number; intervalKind?: string; comparableDays: number;
 exposure: { recordedSeconds: number; sameSceneSeconds: number; sameThemeSeconds: number; trialSeconds: number; trialScenarios?: string[] };
};
export type TrainingStudy = {
	protocolId?: string;
 transferContaminated?: boolean;
 id: string; theme: string; anchorScenario: string; trainingScenario: string; transferScenario?: string;
 status: string; feedback?: string; createdAt: number; dueAt?: number; expiresAt?: number;
 baseline?: MeasurementResult; trial?: MeasurementResult; retest?: MeasurementResult;
 transferBaseline?: MeasurementResult; transferRetest?: MeasurementResult; retentionChange?: number; transferChange?: number;
};

// Separate primitive-only bridge: Wails discovers the exported Go methods at build time.
type Bridge = Record<string, (...args: string[]) => Promise<unknown>>;
export async function call<T>(method: string, ...args: string[]): Promise<T> {
  const bridge = (window as unknown as { go?: { main?: { App?: Bridge } } }).go?.main?.App;
  if (!bridge?.[method]) throw new Error("请在 瞄瞄 桌面应用中打开训练模块；浏览器预览无法读取本机训练数据。");
  return await bridge[method](...args) as T;
}
export async function readState(): Promise<State> {
  const s = JSON.parse(await call<string>("GetTrainingState")) as State;
  s.catalog ??= []; s.skills ??= []; s.history ??= [];
  s.discovery.candidates ??= []; s.discovery.warnings ??= [];
  return s;
}

export type LiveState = {
 revision: number; initializing: boolean; notice: string; error: string;
 plan: (Pick<Plan,"id"|"status"|"index"|"elapsed"|"recorded"|"blockElapsed"|"reminder"> & { blocks: Pick<Block,"recorded"|"runs"|"best"|"outcome"|"target"|"reason">[] }) | null;
};
export async function readLiveState(): Promise<LiveState> { return JSON.parse(await call<string>("GetTrainingLiveState")) as LiveState; }
export function mergeLiveState(state: State, live: LiveState): State {
 if ((state.plan?.id ?? null)!==(live.plan?.id ?? null)) return state;
 return { ...state, initializing: live.initializing, notice: live.notice, error: live.error,
  plan: state.plan && live.plan ? { ...state.plan, ...live.plan, blocks: state.plan.blocks.map((b,i)=>({ ...b, ...live.plan!.blocks[i] })) } : null };
}
