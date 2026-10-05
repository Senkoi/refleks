import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { readPlanRunIDs } from "@/features/training/api";
import { useTrainingProgress } from "@/features/training/TrainingProgressProvider";
export function useTrainingPlanFilter(){
 const [params,setParams]=useSearchParams();
 const planId=params.get("plan")??"";
 const {data}=useTrainingProgress();
 const current=data?.current;
 const plans=[...(current?[current]:[]),...(data?.recentPlans??[]).slice().reverse()].filter((p,i,all)=>all.findIndex(x=>x.id===p.id)===i);
 const [state,setState]=useState<{id:string;ids:Set<string>;loading:boolean;error:string}>({id:"",ids:new Set(),loading:false,error:""});
 const summary=plans.find(p=>p.id===planId);
 const recorded=summary?.recorded??0,runs=summary?.runs??0;
 useEffect(()=>{
  let cancelled=false;
  if(!planId){setState({id:"",ids:new Set(),loading:false,error:""});return;}
  setState(old=>({id:planId,ids:old.id===planId?old.ids:new Set(),loading:true,error:""}));
  void readPlanRunIDs(planId).then(ids=>{if(!cancelled)setState({id:planId,ids:new Set(ids??[]),loading:false,error:""});}).catch(e=>{if(!cancelled)setState({id:planId,ids:new Set(),loading:false,error:String(e)});});
  return ()=>{cancelled=true;};
 },[planId,recorded,runs]);
 const setPlanId=(id:string)=>setParams(old=>{const next=new URLSearchParams(old);if(id)next.set("plan",id);else next.delete("plan");return next;});
 return {planId,setPlanId,plans,planRunIds:state.id===planId?state.ids:new Set<string>(),planLoading:!!planId&&(state.id!==planId||state.loading),planError:state.error};
}
