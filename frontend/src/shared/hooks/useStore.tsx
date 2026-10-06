import type { ReactNode } from "react";
import { createContext, useCallback, useContext, useEffect, useMemo, useReducer, useState } from "react";
import { saveSessionNote } from "../lib/api";
import { groupPracticeSessions, materializeSessions } from "../lib/practiceSessions";
import { DEFAULT_SESSION_GAP_MINUTES, type SessionGroup } from "@/features/training/contracts.generated";
import type { Session } from "../types/domain";
import type { RunRecord } from "../types/ipc";

type State = {
  items: RunRecord[];
  groups: SessionGroup[];
  sessionGapMinutes: number;
  sessionNotes: Record<string, {name: string; notes: string; mergedAliases?:string[]}>;
  sessionGrouping: boolean;
  sessionError: string;
  runHydration: {loading: boolean; loaded: number; total: number; complete: boolean};
};
type Action =
  | {type: "set"; items: RunRecord[]}
  | {type: "setGap"; minutes: number}
  | {type: "grouped"; groups: SessionGroup[]}
  | {type: "groupError"; error: string}
  | {type: "setSessionNotes"; notes: State["sessionNotes"]}
  | {type: "updateRunHydration"; hydration: Partial<State["runHydration"]>}
  | {type: "updateSessionNote"; id: string; name: string; notes: string; mergedAliases?:string[]};
const initial: State = {
  items: [], groups: [], sessionGapMinutes: DEFAULT_SESSION_GAP_MINUTES, sessionNotes: {}, sessionGrouping: false, sessionError: "",
  runHydration: {loading:false, loaded:0, total:0, complete:false},
};
function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "set": return {...state, items: action.items, sessionGrouping:true, sessionError:""};
    case "setGap": return {...state, sessionGapMinutes: Number.isFinite(action.minutes) && action.minutes>0 ? Math.max(1,Math.floor(action.minutes)) : DEFAULT_SESSION_GAP_MINUTES, sessionGrouping:true};
    case "grouped": return {...state, groups:action.groups, sessionGrouping:false, sessionError:""};
    case "groupError": return {...state, sessionGrouping:false, sessionError:action.error};
    case "setSessionNotes": return {...state, sessionNotes:action.notes};
    case "updateRunHydration": return {...state, runHydration:{...state.runHydration,...action.hydration}};
    case "updateSessionNote": return {...state, sessionNotes:{...state.sessionNotes,[action.id]:{name:action.name,notes:action.notes,mergedAliases:action.mergedAliases}}};
  }
}
type Ctx = State & {
  sessions: Session[];
  setRuns: (items: RunRecord[]) => void;
  setSessionGap: (minutes: number) => void;
  setSessionNotes: (notes: State["sessionNotes"]) => void;
  updateRunHydration: (hydration: Partial<State["runHydration"]>) => void;
  saveSessionNote: (id: string, name: string, notes: string) => Promise<void>;
  isInSession: boolean;
};
const StoreCtx = createContext<Ctx | null>(null);
export function StoreProvider({children}: {children: ReactNode}) {
  const [state,dispatch] = useReducer(reducer,initial);
  const [now,setNow] = useState(Date.now);
  const setRuns = useCallback((items:RunRecord[]) => dispatch({type:"set",items}),[]);
  const setSessionGap = useCallback((minutes:number) => dispatch({type:"setGap",minutes}),[]);
  const setSessionNotes = useCallback((notes:State["sessionNotes"]) => dispatch({type:"setSessionNotes",notes}),[]);
  const updateRunHydration = useCallback((hydration:Partial<State["runHydration"]>) => dispatch({type:"updateRunHydration",hydration}),[]);
  const saveSessionNoteAction = useCallback(async(id:string,name:string,notes:string) => {
    const mergedAliases=state.groups.find(g=>g.id===id)?.legacyIds??[];
    await saveSessionNote(id,name,notes,mergedAliases);
    dispatch({type:"updateSessionNote",id,name,notes,mergedAliases});
  },[state.groups]);
  useEffect(() => {
    let cancelled=false;
    if (!state.items.length) { dispatch({type:"grouped",groups:[]}); return; }
    void groupPracticeSessions(state.items,state.sessionGapMinutes).then(groups => {
      if (!cancelled) dispatch({type:"grouped",groups});
    }).catch(error => { if (!cancelled) dispatch({type:"groupError",error:String(error)}); });
    return () => {cancelled=true;};
  },[state.items,state.sessionGapMinutes]);
  const sessions = useMemo(() => materializeSessions(state.items,state.groups,state.sessionNotes),[state.items,state.groups,state.sessionNotes]);
  const lastEnd = sessions[0] ? Date.parse(sessions[0].end) : 0;
  // Recompute when idle expires, even if no new CSV or settings event arrives.
  useEffect(() => {
    setNow(Date.now());
    if (!lastEnd) return;
    const timer=setInterval(() => setNow(Date.now()),15000);
    return () => clearInterval(timer);
  },[lastEnd]);
  const isInSession=lastEnd>0 && now-lastEnd>=0 && now-lastEnd<state.sessionGapMinutes*60000;
  const value=useMemo<Ctx>(() => ({...state,sessions,setRuns,setSessionGap,setSessionNotes,updateRunHydration,saveSessionNote:saveSessionNoteAction,isInSession}),[state,sessions,setRuns,setSessionGap,setSessionNotes,updateRunHydration,saveSessionNoteAction,isInSession]);
  return <StoreCtx.Provider value={value}>{children}</StoreCtx.Provider>;
}
export function useStore<T>(selector:(s:Ctx)=>T):T {
  const ctx=useContext(StoreCtx);
  if(!ctx) throw new Error("StoreProvider missing");
  return selector(ctx);
}
