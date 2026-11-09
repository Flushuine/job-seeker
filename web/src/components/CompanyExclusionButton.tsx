import {useMutation,useQueryClient} from '@tanstack/react-query';
import {Flag,RefreshCw} from 'lucide-react';
import {Link} from 'react-router-dom';
import {api} from '../api';
import {useNotify} from './Notifications';
import type {Job,Preferences} from '../types';

export default function CompanyExclusionButton({job}:{job:Job}){
 const client=useQueryClient();const notify=useNotify();
 const excluded=job.eligibility?.exclusions.includes('Excluded company');
 const exclude=useMutation({mutationFn:()=>api<Preferences>('/companies/'+job.company_id+'/exclude',{method:'POST'}),onSuccess:async pref=>{
  client.setQueryData(['preferences'],pref);
  notify(`${job.company_name} excluded. All its jobs are hidden from default results. Undo in Settings → Job preferences.`);
  await Promise.all(['jobs','job','stats'].map(key=>client.invalidateQueries({queryKey:[key]})));
 }});
 return <div className="company-exclusion-control">{excluded?<Link className="text-link" to="/settings"><Flag size={14}/>Company excluded · Settings</Link>:<button type="button" className="text-link" disabled={exclude.isPending} aria-label={'Exclude company '+job.company_name} title={'Hide all '+job.company_name+' jobs. Undo in Settings → Job preferences.'} onClick={()=>exclude.mutate()}>{exclude.isPending?<RefreshCw size={14} className="spin"/>:<Flag size={14}/>} {exclude.isPending?'Excluding…':'Exclude company'}</button>}{exclude.error&&<small role="alert" className="application-action-error">{exclude.error.message}</small>}</div>
}
