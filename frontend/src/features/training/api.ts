import { GetTrainingWorkbench, GetTrainingExecution, CreateTrainingPlan, CompareTrainingScenes, GetTrainingSceneRequirements } from "@wails/go/main/App";
import type { Block, Preferences, WorkbenchDTO as State, LiveState, Plan, ScenarioComparison } from "./contracts.generated";
export type { Block, Preferences, Plan, Scenario, PersonalAnchor, TrainingStudy, AnchorEvaluation, LiveState, ScenarioComparison } from "./contracts.generated";
export type { WorkbenchDTO as State } from "./contracts.generated";

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
// Separate primitive-only bridge: Wails discovers the exported Go methods at build time.
type Bridge = Record<string, (...args: string[]) => Promise<unknown>>;
export async function call<T>(method: string, ...args: string[]): Promise<T> {
  const bridge = (window as unknown as { go?: { main?: { App?: Bridge } } }).go?.main?.App;
  if (!bridge?.[method]) throw new Error("请在 瞄瞄 桌面应用中打开训练模块；浏览器预览无法读取本机训练数据。");
  return await bridge[method](...args) as T;
}
export async function readState(): Promise<State> {
  desktopBridge();
  const s = await GetTrainingWorkbench() as unknown as State;
  s.catalog ??= []; s.skills ??= []; s.playerLevels ??= []; s.recentPlans ??= [];
  s.discovery.candidates ??= []; s.discovery.warnings ??= [];
  return s;
}


function desktopBridge() {
 const bridge=(window as unknown as {go?:{main?:{App?:Bridge}}}).go?.main?.App;
 if (!bridge) throw new Error("请在 瞄瞄 桌面应用中打开训练模块；浏览器预览无法读取本机训练数据。");
 return bridge;
}
export async function readLiveState():Promise<LiveState>{ desktopBridge();return await GetTrainingExecution() as unknown as LiveState; }
export async function generatePlan(preferences:Preferences):Promise<Plan>{ desktopBridge();return await CreateTrainingPlan({preferences} as Parameters<typeof CreateTrainingPlan>[0]) as unknown as Plan; }
export async function compareScenes(anchor:string,candidate:string):Promise<ScenarioComparison>{desktopBridge();return await CompareTrainingScenes(anchor,candidate) as unknown as ScenarioComparison;}
export function mergeLiveState(state: State, live: LiveState): State {
 if ((state.plan?.id ?? null)!==(live.plan?.id ?? null)) return state;
 return { ...state, initializing: live.initializing, notice: live.notice, error: live.error,
  plan: state.plan && live.plan ? { ...state.plan, ...live.plan, blocks: state.plan.blocks.map((b,i)=>({ ...b, ...live.plan!.blocks[i] })) } : null };
}

export async function readRequirements(name:string,hash:string):Promise<import("./contracts.generated").Descriptor>{desktopBridge();return await GetTrainingSceneRequirements(name,hash) as unknown as import("./contracts.generated").Descriptor;}
