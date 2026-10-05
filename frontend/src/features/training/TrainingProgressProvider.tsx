import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { EventsOn } from "@wails/runtime";
import { readTrainingProgress, readTrainingGuidance } from "./api";
import type { TrainingProgressDTO, TrainingGuidance } from "./contracts.generated";

type ProgressState = {data: TrainingProgressDTO | null; guidance:TrainingGuidance|null; guidanceError:string; error: string; loading: boolean};
const initial:ProgressState={data:null,guidance:null,guidanceError:"",error:"",loading:true};
const Context = createContext<ProgressState>(initial);
export const notifyTrainingChanged = () => window.dispatchEvent(new Event("aimmeow:training-changed"));

// One lightweight poll supplies overview and history. Catalogs/ability and
// detailed execution blocks remain on-demand in the workbench.
export function TrainingProgressProvider({children}:{children:ReactNode}) {
  const [state,setState]=useState<ProgressState>(initial);
  useEffect(() => {
    let alive=true, inFlight=false, dirty=false;
    let timer:ReturnType<typeof setTimeout>;
    let running=false;
    let guidance:TrainingGuidance|null=null, guidanceAt=0;
    const refresh=async() => {
      if(!alive) return;
      if(inFlight) {dirty=true;return;}
      clearTimeout(timer);inFlight=true;
      try {
        const data=await readTrainingProgress();
        if(!alive) return;
        data.recentPlans ??=[];
        let guidanceError="";
        if(data.guidance) guidance=data.guidance;
        else if(!guidance || guidance.mode==="current" || Date.now()-guidanceAt>=30000) {
          try { guidance=await readTrainingGuidance();guidanceAt=Date.now(); }
          catch(e) { guidance=null;guidanceError=String(e); }
        }
        if(!alive) return;
        running=!!data.current && ["running","ready","waiting"].includes(data.current.status);
        setState({data,guidance,guidanceError,error:"",loading:false});
      } catch(e) {if(alive)setState(old=>({...old,error:String(e),loading:false}));}
      finally {
        inFlight=false;
        if(alive) {if(dirty)guidanceAt=0;const delay=dirty?0:running?2000:10000;dirty=false;timer=setTimeout(()=>void refresh(),delay);}
      }
    };
    const request=()=>{guidanceAt=0;void refresh();};
    const off=(window as unknown as {runtime?:{EventsOnMultiple?:unknown}}).runtime?.EventsOnMultiple ? EventsOn("runs:added",request) : ()=>{};
    const offBenchmark=(window as unknown as {runtime?:{EventsOnMultiple?:unknown}}).runtime?.EventsOnMultiple ? EventsOn("benchmark:progress:updated",request) : ()=>{};
    window.addEventListener("aimmeow:training-changed",request);
    window.addEventListener("focus",request);
    request();
    return ()=>{alive=false;clearTimeout(timer);off();offBenchmark();window.removeEventListener("aimmeow:training-changed",request);window.removeEventListener("focus",request);};
  },[]);
  return <Context.Provider value={state}>{children}</Context.Provider>;
}
export const useTrainingProgress=()=>useContext(Context);
