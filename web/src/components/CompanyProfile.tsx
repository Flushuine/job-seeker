import {useState,type FormEvent} from 'react';
import {useMutation,useQueryClient} from '@tanstack/react-query';
import {Building2,ExternalLink,Globe2,MapPin,Pencil,UserRound,X,Check} from 'lucide-react';
import {api,json} from '../api';
import type {Company,CompanyFact,CompanyProfile} from '../types';

const fields=[
 ['summary','Company overview',Building2],
 ['legal_name','Legal entity',Building2],
 ['origin_country','Origin / incorporation country',Globe2],
 ['headquarters','Headquarters',MapPin],
 ['ceo','CEO / co-CEOs',UserRound],
 ['ceo_nationality','CEO nationality / citizenship',Globe2],
] as const;
const emptyFact=():CompanyFact=>({value:'',source_url:'',source_name:'',notes:''});
const recorded=(v:string)=>new Date(v).toLocaleDateString(undefined,{day:'numeric',month:'short',year:'numeric'});

export default function CompanyProfilePanel({company}:{company:Company}){
 const [editing,setEditing]=useState(false);
 const profile=company.profile||{};
 return <section className="panel detail-section company-profile">
  <div className="section-heading"><div><h2>Company profile</h2><p>Company origins and leadership, with sources.</p></div><button className="button" type="button" onClick={()=>setEditing(!editing)}>{editing?<X size={15}/>:<Pencil size={15}/>} {editing?'Cancel':'Edit profile'}</button></div>
  {editing?<ProfileEditor key={company.updated_at} company={company} onSaved={()=>setEditing(false)}/>:<dl className="company-facts">{fields.map(([key,label,Icon])=>{
   const fact=profile[key];
   return <div className={'company-fact '+(key==='summary'?'company-fact-wide':'')} key={key}>
    <dt><Icon size={15}/>{label}</dt>
    <dd>{fact?.value?<><p className="fact-value">{fact.value}</p><a className="text-link" href={fact.source_url} target="_blank" rel="noreferrer">{fact.source_name}<ExternalLink size={12}/></a>{fact.recorded_at&&<small>Recorded {recorded(fact.recorded_at)}</small>}{fact.notes&&<p className="fact-notes">{fact.notes}</p>}</>:<><p className="fact-unknown">Unknown</p><small>No sourced information recorded.</small></>}</dd>
   </div>
  })}</dl>}
  <p className="muted small company-profile-note">Company origin and headquarters are separate from job location. Nationality is recorded only when explicitly sourced; it does not affect job ranking. Saved sources are not automatically fetched or verified.</p>
 </section>
}

function ProfileEditor({company,onSaved}:{company:Company;onSaved:()=>void}){
 const client=useQueryClient();
 const [draft,setDraft]=useState<CompanyProfile>(()=>Object.fromEntries(fields.map(([key])=>[key,{...emptyFact(),...company.profile?.[key]}])));
 const save=useMutation({mutationFn:()=>api<Company>('/companies/'+company.id+'/profile',json('PATCH',draft)),onSuccess:()=>{client.invalidateQueries({queryKey:['company',company.id]});client.invalidateQueries({queryKey:['companies']});onSaved()}});
 function update(key:keyof CompanyProfile,part:Partial<CompanyFact>){setDraft(current=>{
  const next={...current,[key]:{...emptyFact(),...current[key],...part}};
  if(key==='ceo'&&part.value!==undefined&&part.value!==current.ceo?.value){next.ceo_nationality=emptyFact()}
  return next;
 })}
 function submit(e:FormEvent){e.preventDefault();save.mutate()}
 return <form className="company-profile-editor" onSubmit={submit}>
  <p className="muted small">Use an official company page, filing, or reliable public biography. Leave unsupported fields empty. For co-CEOs, name each person and specify whose nationality the source establishes.</p>
  {fields.map(([key,label])=>{
   const fact=draft[key]||emptyFact();const hasValue=!!fact.value.trim();
   return <fieldset className="company-fact-editor" key={key}><legend>{label}</legend>
    <label>{label}{key==='summary'?<textarea value={fact.value} rows={3} maxLength={2000} onChange={e=>update(key,{value:e.target.value})}/>:<input value={fact.value} maxLength={300} disabled={key==='ceo_nationality'&&!draft.ceo?.value.trim()} onChange={e=>update(key,{value:e.target.value})} placeholder="Unknown until sourced"/>}</label>
    {hasValue&&<><div className="two-col"><label>Source URL<input type="url" value={fact.source_url} required maxLength={2048} placeholder="https://company.com/about" onChange={e=>update(key,{source_url:e.target.value})}/></label><label>Source name<input value={fact.source_name} required maxLength={120} placeholder="Company leadership page" onChange={e=>update(key,{source_name:e.target.value})}/></label></div><label>Context or source limitations (optional)<textarea value={fact.notes} rows={2} maxLength={600} placeholder="For example: nationality reported in a dated corporate biography." onChange={e=>update(key,{notes:e.target.value})}/></label></>}
   </fieldset>
  })}
  {save.error&&<div className="error" role="alert">{save.error.message}</div>}
  <button className="button primary" disabled={save.isPending}><Check size={15}/>{save.isPending?'Saving…':'Save company profile'}</button>
 </form>
}
