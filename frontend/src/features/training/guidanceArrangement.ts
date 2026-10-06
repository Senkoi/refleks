import type { TrainingGuidance } from "./contracts.generated";

// Guidance is a bounded preview, not the complete plan. Absence means only
// that a scene is not in this preview; it is never a negative recommendation.
export function guidanceArrangements(
  guidance: TrainingGuidance | null,
  stale = false,
) {
  const result = new Map<
    string,
    { mode: "current" | "next"; runs: number; roles: string[] }
  >();
  if (
    stale ||
    !guidance ||
    (guidance.mode !== "current" && guidance.mode !== "next")
  )
    return result;
  for (const item of guidance.items ?? []) {
    const key = item.scenario.trim().toLowerCase();
    if (!key || item.runs <= 0) continue;
    const existing = result.get(key);
    if (existing) {
      existing.runs += item.runs;
      if (!existing.roles.includes(item.role)) existing.roles.push(item.role);
    } else
      result.set(key, {
        mode: guidance.mode,
        runs: item.runs,
        roles: [item.role],
      });
  }
  return result;
}
