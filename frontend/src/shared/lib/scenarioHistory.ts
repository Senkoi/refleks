export type ScenarioHistoryPoint = {
  at: number; score: number; accuracy: number; version: string; gameVersion: string;
  sensScale: string; sens: number; vertSens: number; dpi: number;
  fov: number; fovScale: string; seconds: number; targetScale: number; timeScale: number;
};
export type ScenarioHistoryGroup = { key: string; label: string; points: ScenarioHistoryPoint[] };
export const scenarioKey = (name: string) => name.trim().toLowerCase();

// Keep version and measurement conditions separate; history is not an ability estimate.
export function groupScenarioHistory(points: ScenarioHistoryPoint[]): ScenarioHistoryGroup[] {
  const versions = new Map<string, number>();
  const groups = new Map<string, ScenarioHistoryGroup>();
  const ordered = [...points].filter(p => Number.isFinite(p.at) && p.at > 0 && Number.isFinite(p.score)).sort((a,b) => a.at-b.at);
  for (const point of ordered) {
    const version = JSON.stringify([point.version, point.gameVersion]);
    if (!versions.has(version)) versions.set(version, versions.size+1);
    const key = JSON.stringify([version, point.sensScale, point.sens, point.vertSens, point.dpi, point.fov, point.fovScale, Math.round(point.seconds), point.targetScale, point.timeScale]);
    if (!groups.has(key)) {
      const label = [point.version ? `版本 ${versions.get(version)}` : "未标记版本", point.sens > 0 ? `灵敏度 ${point.sens} ${point.sensScale}` : "灵敏度未记录", point.vertSens !== point.sens ? `垂直 ${point.vertSens}` : "", point.dpi > 0 ? `${point.dpi} DPI` : "", point.fov > 0 ? `视野 ${point.fov} ${point.fovScale}` : "", point.seconds > 0 ? `${Math.round(point.seconds)} 秒` : "", point.targetScale > 0 && point.targetScale !== 1 ? `目标 ×${point.targetScale}` : "", point.timeScale > 0 && point.timeScale !== 1 ? `速度 ×${point.timeScale}` : ""].filter(Boolean).join(" · ");
      groups.set(key, { key, label, points: [] });
    }
    groups.get(key)!.points.push(point);
  }
  return [...groups.values()].sort((a,b) => b.points[b.points.length-1].at-a.points[a.points.length-1].at);
}

type Bridge = Record<string, (...args: unknown[]) => Promise<unknown>>;
async function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  const bridge = (window as unknown as {go?: {main?: {App?: Bridge}}}).go?.main?.App;
  if (!bridge?.[method]) throw new Error("请在 瞄瞄 桌面应用中查看本机训练历史。");
  return await bridge[method](...args) as T;
}
export async function getScenarioTrainingHistory(name: string) { return await invoke<ScenarioHistoryPoint[] | null>("GetScenarioTrainingHistory", name) ?? []; }
export async function getPlayedScenarioNames() { return await invoke<string[] | null>("GetPlayedScenarioNames") ?? []; }
