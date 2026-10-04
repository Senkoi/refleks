import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ChartNoAxesCombined } from "lucide-react";
import { EventsOn } from "@wails/runtime/runtime";
import { getPlayedScenarioNames, scenarioKey } from "../lib/scenarioHistory";
import { ScenarioTrainingHistoryModal } from "./ScenarioTrainingHistoryModal";
import "./scenarioHistory.css";

type Selection = { scenario: string; supplementary?: ReactNode; trigger?: HTMLElement };
const HistoryContext = createContext<{ played: Set<string>; open: (selection: Selection) => void } | null>(null);

export function ScenarioHistoryProvider({ children }: { children: ReactNode }) {
  const [played, setPlayed] = useState<Set<string>>(new Set());
  const [selected, setSelected] = useState<Selection | null>(null);
  const request = useRef(0);
  const refresh = useCallback(async () => {
    const id = ++request.current;
    try { const names = await getPlayedScenarioNames(); if (id === request.current) setPlayed(new Set(names.map(scenarioKey))); }
    catch { /* Optional history entries must not block the rest of the app. */ }
  }, []);
  useEffect(() => {
    void refresh();
    if (!(window as unknown as {runtime?: {EventsOnMultiple?: unknown}}).runtime?.EventsOnMultiple) return () => { request.current++; };
    let timer: ReturnType<typeof setTimeout>;
    const schedule = () => { clearTimeout(timer); timer = setTimeout(() => { void refresh(); }, 250); };
    const off = EventsOn("runs:added", schedule);
    const offWatcher = EventsOn("runs:watcher:started", schedule);
    return () => { request.current++; clearTimeout(timer); off(); offWatcher(); };
  }, [refresh]);
  const value = useMemo(() => ({ played, open: setSelected }), [played]);
  return <HistoryContext.Provider value={value}>{children}{selected && <ScenarioTrainingHistoryModal key={selected.scenario} scenario={selected.scenario} supplementary={selected.supplementary} onClose={() => { setSelected(null); requestAnimationFrame(() => { if (selected.trigger?.isConnected) selected.trigger.focus(); }); }} />}</HistoryContext.Provider>;
}

export function ScenarioHistoryLink({ name, known = false, supplementary, iconOnly = false, className = "" }: {
  name: string; known?: boolean; supplementary?: ReactNode; iconOnly?: boolean; className?: string;
}) {
  const history = useContext(HistoryContext);
  const available = !!history && (known || history.played.has(scenarioKey(name)));
  if (!available) return <span className={className}>{name}</span>;
  return <button type="button" className={`scenario-history-link ${className}`} title="查看训练历史" aria-label={`查看 ${name} 的训练历史`} aria-haspopup="dialog" onKeyDown={e => e.stopPropagation()} onClick={e => { e.stopPropagation(); history.open({scenario:name,supplementary,trigger:e.currentTarget}); }}>{!iconOnly && <span>{name}</span>}<ChartNoAxesCombined size={13} aria-hidden="true" /></button>;
}
