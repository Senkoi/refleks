import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { EventsOn } from "@wails/runtime";
import { readTrainingProgress } from "./api";
import type { TrainingProgressDTO } from "./contracts.generated";

type ProgressState = {data: TrainingProgressDTO | null; error: string; loading: boolean};
const Context = createContext<ProgressState>({data:null,error:"",loading:true});
export const notifyTrainingChanged = () => window.dispatchEvent(new Event("aimmeow:training-changed"));

// One lightweight poll supplies overview and history. Catalogs/ability and
// detailed execution blocks remain on-demand in the workbench.
export function TrainingProgressProvider({children}:{children:ReactNode}) {
  const [state,setState]=useState<ProgressState>({data:null,error:"",loading:true});
  useEffect(() => {
    let alive=true, inFlight=false, dirty=false;
    let timer:ReturnType<typeof setTimeout>;
    let running=false;
    const refresh=async() => {
      if(!alive) return;
      if(inFlight) {dirty=true;return;}
      clearTimeout(timer);inFlight=true;
      try {
        const data=await readTrainingProgress();
        if(!alive) return;
        data.recentPlans ??=[];
        running=!!data.current && ["running","ready","waiting"].includes(data.current.status);
        setState({data,error:"",loading:false});
      } catch(e) {if(alive)setState(old=>({...old,error:String(e),loading:false}));}
      finally {
        inFlight=false;
        if(alive) {const delay=dirty?0:running?2000:10000;dirty=false;timer=setTimeout(()=>void refresh(),delay);}
      }
    };
    const request=()=>void refresh();
    const off=(window as unknown as {runtime?:{EventsOnMultiple?:unknown}}).runtime?.EventsOnMultiple ? EventsOn("runs:added",request) : ()=>{};
    window.addEventListener("aimmeow:training-changed",request);
    window.addEventListener("focus",request);
    request();
    return ()=>{alive=false;clearTimeout(timer);off();window.removeEventListener("aimmeow:training-changed",request);window.removeEventListener("focus",request);};
  },[]);
  return <Context.Provider value={state}>{children}</Context.Provider>;
}
export const useTrainingProgress=()=>useContext(Context);
