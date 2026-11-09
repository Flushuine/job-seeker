import {useEffect,useId,useLayoutEffect,useRef,useState,type CSSProperties,type KeyboardEvent} from 'react';
import {createPortal} from 'react-dom';
import {Check,ChevronDown} from 'lucide-react';

export type SelectOption={value:string|number;label:string;disabled?:boolean};
export function nextOption(options:SelectOption[],current:number,direction:1|-1){
 for(let step=1;step<=options.length;step++){const index=(current+direction*step+options.length)%options.length;if(!options[index].disabled)return index}
 return -1;
}
// Scroll only the menu; scrollIntoView also moves the document for portalled options.
export function revealOption(list:HTMLElement,option:HTMLElement){
 const bounds=list.getBoundingClientRect();const item=option.getBoundingClientRect();
 const top=bounds.top+list.clientTop;const bottom=top+list.clientHeight;
 if(item.top<top)list.scrollTop+=item.top-top;
 else if(item.bottom>bottom)list.scrollTop+=item.bottom-bottom;
}
export default function Select({options,value,defaultValue,onChange,name,disabled=false,'aria-label':label}:{options:SelectOption[];value?:string|number;defaultValue?:string|number;onChange?:(value:string)=>void;name?:string;disabled?:boolean;'aria-label':string}){
 const [local,setLocal]=useState(String(defaultValue??options[0]?.value??''));const selected=String(value??local);
 const [open,setOpen]=useState(false);const [active,setActive]=useState(-1);const [position,setPosition]=useState<CSSProperties>({position:'fixed',visibility:'hidden'});
 const root=useRef<HTMLDivElement>(null);const trigger=useRef<HTMLButtonElement>(null);const menu=useRef<HTMLDivElement>(null);const search=useRef({text:'',time:0});const id=useId();
 function show(){if(disabled||options.every(option=>option.disabled))return;const index=options.findIndex(option=>String(option.value)===selected&&!option.disabled);setActive(index>=0?index:nextOption(options,-1,1));setOpen(true)}
 function choose(index:number){const option=options[index];if(!option||option.disabled)return;setLocal(String(option.value));onChange?.(String(option.value));setOpen(false);trigger.current?.focus({preventScroll:true})}
 useEffect(()=>{if(disabled)setOpen(false)},[disabled]);
 useEffect(()=>{const form=root.current?.closest('form');const reset=()=>{setLocal(String(defaultValue??options[0]?.value??''));setOpen(false)};form?.addEventListener('reset',reset);return()=>form?.removeEventListener('reset',reset)},[defaultValue,options]);
 useLayoutEffect(()=>{
  if(!open)return;
  function place(){const rect=trigger.current?.getBoundingClientRect();if(!rect)return;const below=window.innerHeight-rect.bottom-14;const above=rect.top-14;const upwards=below<Math.min(240,options.length*44)&&above>below;const maxHeight=Math.max(44,Math.min(280,upwards?above:below));const width=Math.min(Math.max(rect.width,200),window.innerWidth-16);setPosition({position:'fixed',left:Math.max(8,Math.min(rect.left,window.innerWidth-width-8)),width,maxHeight,...(upwards?{bottom:window.innerHeight-rect.top+6}:{top:rect.bottom+6})})}
  function outside(event:PointerEvent){if(!root.current?.contains(event.target as Node)&&!menu.current?.contains(event.target as Node))setOpen(false)}
  function focusOutside(event:FocusEvent){if(!root.current?.contains(event.target as Node)&&!menu.current?.contains(event.target as Node))setOpen(false)}
  place();window.addEventListener('resize',place);window.addEventListener('scroll',place,true);document.addEventListener('pointerdown',outside);document.addEventListener('focusin',focusOutside);
  return()=>{window.removeEventListener('resize',place);window.removeEventListener('scroll',place,true);document.removeEventListener('pointerdown',outside);document.removeEventListener('focusin',focusOutside)}
 },[open,options.length]);
 useEffect(()=>{const option=document.getElementById(id+'-'+active);if(open&&menu.current&&option)revealOption(menu.current,option)},[active,open,id]);
 function keyDown(event:KeyboardEvent<HTMLButtonElement>){
  if(event.key==='Tab'){setOpen(false);return}
  if(event.key==='Escape'){if(open){event.preventDefault();event.stopPropagation();setOpen(false)}return}
  if(['ArrowDown','ArrowUp','Home','End','Enter',' '].includes(event.key)){
   event.preventDefault();
   if(!open){show();if(event.key==='Home')setActive(nextOption(options,-1,1));if(event.key==='End')setActive(nextOption(options,0,-1));return}
   if(event.key==='Enter'||event.key===' '){choose(active);return}
   if(event.key==='Home')setActive(nextOption(options,-1,1));else if(event.key==='End')setActive(nextOption(options,0,-1));else setActive(nextOption(options,active,event.key==='ArrowDown'?1:-1));
   return;
  }
  if(event.key.length===1&&!event.ctrlKey&&!event.metaKey&&!event.altKey){event.preventDefault();const now=Date.now();search.current={text:(now-search.current.time<700?search.current.text:'')+event.key.toLowerCase(),time:now};if(!open)show();const query=search.current.text;const index=options.findIndex(option=>!option.disabled&&option.label.toLowerCase().startsWith(query));if(index>=0)setActive(index)}
 }
 const current=options.find(option=>String(option.value)===selected);
 return <div className="custom-select" ref={root}>
  {name&&<input type="hidden" name={name} value={selected} disabled={disabled}/>}
  <button ref={trigger} type="button" className="select-trigger" role="combobox" aria-label={label} aria-haspopup="listbox" aria-expanded={open} aria-controls={open?id:undefined} aria-activedescendant={open&&active>=0?id+'-'+active:undefined} disabled={disabled} onClick={()=>open?setOpen(false):show()} onKeyDown={keyDown}><span>{current?.label||'Choose an option'}</span><ChevronDown size={17} aria-hidden="true"/></button>
  {open&&createPortal(<div ref={menu} id={id} role="listbox" aria-label={label} className="select-menu" style={position}>{options.map((option,index)=><div key={String(option.value)} id={id+'-'+index} role="option" aria-selected={String(option.value)===selected} aria-disabled={option.disabled||undefined} className={'select-option '+(active===index?'highlighted ':'')+(String(option.value)===selected?'selected':'')} onPointerMove={()=>{if(!option.disabled)setActive(index)}} onMouseDown={event=>event.preventDefault()} onClick={()=>choose(index)}><span>{option.label}</span>{String(option.value)===selected&&<Check size={16} aria-hidden="true"/>}</div>)}</div>,document.body)}
 </div>
}
