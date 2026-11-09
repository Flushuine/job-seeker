import {Clock} from 'lucide-react';
import type {Job} from '../types';

export default function JobWorkHours({job,compact=false}:{job:Job;compact?:boolean}){
 const hours=job.work_hours;
 if(!hours)return <div className="job-work-hours"><Clock size={16}/><span>Working hours in your time zone are not available.</span></div>;
 return <div className={'job-work-hours '+(compact?'compact':'')}>
  <Clock size={16} aria-hidden="true"/>
  <div><strong>{hours.status==='stated'?`Posted time window · ${hours.target_label}`:hours.status==='example'?`Possible hours · ${hours.target_label}`:'Your local working hours are unknown'}</strong>{hours.target_zone&&<small>Based on your CV/profile location: {hours.target_location} · {hours.target_zone}</small>}
   {hours.examples.map(example=><div className="work-hours-example" key={example.zone+example.local_hours}><b>{example.your_hours} {hours.target_label}</b><span>{example.local_hours} · {example.zone}</span>{!compact&&<small>{example.difference}</small>}</div>)}
   {hours.status==='example'&&<span className="badge">Example only · assumes 09:00–17:00 local</span>}
   <p>{compact?(hours.status==='example'?'Hours unconfirmed. Named zones follow today’s daylight saving rules; confirm the schedule with the recruiter.':hours.status==='stated'?'Posted time window; confirm whether it describes a shift or required overlap.':hours.note):hours.note}</p>
   {!compact&&hours.source&&<p className="work-hours-source">Posting excerpt: “{hours.source}”</p>}
  </div>
 </div>
}
