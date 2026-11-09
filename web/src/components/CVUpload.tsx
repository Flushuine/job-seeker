import {useEffect,useRef,useState,type FormEvent} from 'react';
import {useMutation,useQuery,useQueryClient} from '@tanstack/react-query';
import {Upload,FileText,RefreshCw,CheckCircle2,AlertCircle} from 'lucide-react';
import {api} from '../api';
import type {CVStatus} from '../types';

export default function CVUploadPanel(){
 const client=useQueryClient();const input=useRef<HTMLInputElement>(null);const refreshed=useRef('');
 const [file,setFile]=useState<File|null>(null);const [error,setError]=useState('');
 const status=useQuery({queryKey:['cv'],queryFn:()=>api<CVStatus>('/profile/cv'),refetchInterval:q=>['queued','running'].includes(q.state.data?.task?.status||'')?2000:false});
 const upload=useMutation({mutationFn:(file:File)=>{const body=new FormData();body.append('file',file);return api<CVStatus>('/profile/cv',{method:'POST',body})},onSuccess:data=>{client.setQueryData(['cv'],data);setFile(null);if(input.current)input.current.value=''}});
 const doc=status.data?.upload;const task=status.data?.task;const busy=task?.status==='queued'||task?.status==='running';
 useEffect(()=>{
  if(task?.status==='completed'&&doc?.applied_at&&refreshed.current!==doc.id){
   refreshed.current=doc.id;
   for(const key of ['profile','jobs','job','stats'])void client.invalidateQueries({queryKey:[key]});
  }
 },[task?.status,doc?.applied_at,doc?.id,client]);
 function submit(e:FormEvent){e.preventDefault();setError('');if(!file)return;if(file.size>5*1024*1024){setError('Choose a CV up to 5 MB.');return};upload.mutate(file)}
 return <section className="panel form-panel cv-upload-panel">
  <div className="section-heading"><div><h2><FileText size={19}/>Personalise from your CV</h2><p>Upload a new CV to refresh your professional profile and job matches automatically.</p></div><span className="badge">On-device analysis</span></div>
  <form onSubmit={submit} className="cv-upload-form">
   <label>CV document (PDF or TXT, up to 5 MB)<input ref={input} type="file" accept=".pdf,.txt,application/pdf,text/plain" onChange={e=>{setFile(e.target.files?.[0]||null);setError('');upload.reset()}} required/></label>
   <button className="button primary" disabled={!file||upload.isPending}>{upload.isPending?<RefreshCw size={15} className="spin"/>:<Upload size={15}/>} {upload.isPending?'Uploading…':'Upload and personalise'}</button>
  </form>
  <p className="muted small">Updates your summary and CV skills, plus name, experience and location when explicitly stated. Your job preferences stay unchanged. A newer upload takes priority. PDF files need selectable text; scanned CVs need a TXT/OCR export.</p>
  {(error||upload.error||status.error)&&<div className="error" role="alert"><AlertCircle size={15}/>{error||upload.error?.message||status.error?.message}</div>}
  {doc&&<div className="cv-upload-status" aria-live="polite">
   <div className="section-heading"><strong>{doc.filename}</strong><span className={'badge '+(doc.applied_at?'green':'')}>{busy?(task?.status==='queued'?'Queued':'Analysing…'):doc.applied_at?'Profile updated':task?.status||'Uploaded'}</span></div>
   <small className="muted">Uploaded {new Date(doc.created_at).toLocaleString()} · {doc.model}</small>
   {busy&&<p className="muted small"><RefreshCw className="spin" size={13}/> Your current profile stays available during local analysis.</p>}
   {task?.status==='failed'&&<div className="error" role="alert">{task.error} Your previous profile has been kept. Fix the issue and upload again to retry.</div>}
   {doc.applied_at&&<p className="cv-success"><CheckCircle2 size={15}/>Profile and matching refreshed from this CV.</p>}
   {doc.analysis&&<><p className="small">{doc.analysis.summary}</p><div className="chips">{doc.analysis.skills.map(skill=><span className="badge" key={skill}>{skill}</span>)}</div>{doc.analysis.unknowns.length>0&&<details><summary>Extraction notes and unknown information</summary><ul className="text-list">{doc.analysis.unknowns.map((note,i)=><li key={i}>{note}</li>)}</ul></details>}</>}
  </div>}
 </section>
}
