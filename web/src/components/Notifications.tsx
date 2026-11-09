import {createContext,useContext,useEffect,useState,type ReactNode} from 'react';
import {Check,X} from 'lucide-react';

const NotifyContext=createContext<(message:string)=>void>(()=>{});
export const useNotify=()=>useContext(NotifyContext);
export default function Notifications({children}:{children:ReactNode}){
 const [notice,setNotice]=useState<{message:string}|null>(null);
 useEffect(()=>{if(!notice)return;const timer=window.setTimeout(()=>setNotice(null),5000);return()=>window.clearTimeout(timer)},[notice]);
 return <NotifyContext.Provider value={message=>setNotice({message})}>{children}<div className="toast-region" role="status" aria-live="polite" aria-atomic="true">{notice&&<div className="success-toast"><Check size={20} aria-hidden="true"/><div><strong>{notice.message}</strong></div><button type="button" aria-label="Dismiss notification" onClick={()=>setNotice(null)}><X size={17}/></button></div>}</div></NotifyContext.Provider>
}
