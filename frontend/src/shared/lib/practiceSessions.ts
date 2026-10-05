import { GroupPracticeSessions } from "@wails/go/main/App";
import type { Session } from "../types/domain";
import type { RunRecord } from "../types/ipc";
import type { SessionGroup } from "@/features/training/contracts.generated";

export const runId = (r: RunRecord) => r.filePath || `${r.fileName}|${r.stats.summary.datePlayed}|${r.stats.summary.scenario}`;

export async function groupPracticeSessions(items: RunRecord[], gapMinutes: number): Promise<SessionGroup[]> {
  const entries = items.map(r => ({runId: runId(r), endedAt: Date.parse(r.stats.summary.datePlayed), seconds: r.stats.summary.duration})).filter(r => Number.isFinite(r.endedAt) && Number.isFinite(r.seconds));
  const bridge = (window as unknown as {go?: {main?: {App?: {GroupPracticeSessions?: (request: string) => Promise<string>}}}}).go?.main?.App;
  if (!bridge?.GroupPracticeSessions) throw new Error("训练时段暂时无法读取，请重新打开瞄瞄。");
  return JSON.parse(await GroupPracticeSessions(JSON.stringify({entries, gapMinutes}))) as SessionGroup[];
}

// Legacy names/notes are read through aliases until the user next saves them.
// Merging two old intervals must retain both notes, rather than overwrite one.
export function materializeSessions(items: RunRecord[], groups: SessionGroup[], notes: Record<string, {name: string; notes: string; mergedAliases?:string[]}>): Session[] {
  const index = new Map(items.map(r => [runId(r), r]));
  return groups.map(g => {
    const exact=notes[g.id];
    const consumed=new Set(exact?.mergedAliases ?? []);
    const other=[...new Set(g.legacyIds)].filter(id=>id!==g.id&&!consumed.has(id)).map(id=>notes[id]).filter(Boolean);
    const combined=[...(exact?[exact]:[]),...other];
    const saved={name:[...new Set(combined.map(n=>n.name).filter(Boolean))].join(" / "),notes:[...new Set(combined.map(n=>n.notes).filter(Boolean))].join("\n\n")};
    return {id:g.id, legacyIds:g.legacyIds, start:new Date(g.startedAt).toISOString(), end:new Date(g.endedAt).toISOString(), items:g.runIds.map(id => index.get(id)).filter((r): r is RunRecord => !!r).reverse(), name:saved.name, notes:saved.notes};
  }).filter(s => s.items.length > 0).reverse();
}
