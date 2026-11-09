import {useMutation,useQuery,useQueryClient} from '@tanstack/react-query';
import {Link} from 'react-router-dom';
import {Bookmark,Check,ArrowUpRight,RefreshCw} from 'lucide-react';
import {api,json} from '../api';
import type {Application,Job,JobPage} from '../types';

export default function JobApplicationControls({job}:{job:Job}){
 const client=useQueryClient();
 // All cards share this query key, so application status uses one cached request.
 const applications=useQuery({queryKey:['applications'],queryFn:()=>api<Application[]>('/applications')});
 const application=applications.data?.find(a=>a.job_id===job.id);
 const hasApplied=!!application&&application.stage!=='Saved';
 const statusReady=applications.isSuccess;
 const mark=useMutation({
  mutationFn:(stage:'Saved'|'Applied')=>{
   if(application&&application.stage!=='Saved')throw new Error('This job is already tracked. Update its stage in Application Tracker.');
   return application
    ?api<Application>('/applications/'+application.id,json('PATCH',{stage,notes:application.notes}))
    :api<Application>('/applications',json('POST',{job_id:job.id,stage,notes:''}));
  },
  onSuccess:async(saved)=>{
   client.setQueryData<Application[]>(['applications'],current=>[saved,...(current||[]).filter(a=>a.job_id!==saved.job_id)]);
   if(saved.stage!=='Saved'){
    for(const [key,page] of client.getQueriesData<JobPage>({queryKey:['jobs']})){
     const includeApplied=new URLSearchParams(String(key[1]||'')).get('include_applied')==='true';
     if(page&&!includeApplied&&page.items.some(j=>j.id===job.id))client.setQueryData(key,{...page,items:page.items.filter(j=>j.id!==job.id),total:Math.max(0,page.total-1)});
    }
   }
   await Promise.all([
    client.invalidateQueries({queryKey:['jobs']}),
    client.invalidateQueries({queryKey:['applications']}),
    client.invalidateQueries({queryKey:['stats']}),
    client.invalidateQueries({queryKey:['job',job.id]}),
   ]);
  },
 });
 const disabled=!statusReady||mark.isPending;
 return <div className="job-application-controls">
  {application&&<span className={'badge application-flag '+(hasApplied?(['Rejected','Withdrawn'].includes(application.stage)?'warn':'green'):'')} aria-label={'Application status: '+application.stage}>{hasApplied?<Check size={12}/>:<Bookmark size={12}/>} {application.stage}</span>}
  {hasApplied?<Link className="text-link" to="/applications">View application <ArrowUpRight size={13}/></Link>:<div className="job-application-buttons">
   <button type="button" className="icon-button save-button" title={application?'Already saved':'Save job'} aria-label={(application?'Saved: ':'Save ')+job.title} disabled={disabled||!!application} onClick={()=>mark.mutate('Saved')}>{application?<Check size={17}/>:<Bookmark size={17}/>}</button>
   <button type="button" className="button mark-applied" disabled={disabled} title="Record that you have already submitted an application" onClick={()=>mark.mutate('Applied')}><Check size={14}/>{mark.isPending&&mark.variables==='Applied'?'Marking…':'Mark applied'}</button>
  </div>}
  {applications.isPending&&<small className="muted">Loading application status…</small>}
  {applications.isError&&<button type="button" className="text-link" onClick={()=>applications.refetch()} title={applications.error.message}><RefreshCw size={12}/>Retry application status</button>}
  {mark.error&&<small className="application-action-error" role="alert">{mark.error.message}</small>}
 </div>
}
