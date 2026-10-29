import { DocumentPageSchema, DocumentSchema, VerificationSchema, type ContractInput, type DocumentPage, type DocumentRecord, type VerificationReport } from './contracts';

export type ApiError = {kind:'http';status:number;message:string;requestId?:string}|{kind:'network'|'invalid-response';message:string}|{kind:'aborted'};
export type Result<T,E=ApiError> = {ok:true;value:T}|{ok:false;error:E};
type Options = {signal?:AbortSignal|undefined};
export type Decision = {action:'submit'|'retry'}|{action:'approve'|'reject';reason:string};
export interface DocumentApi {
 list(options:{limit:number;offset:number}&Options):Promise<Result<DocumentPage>>;
 get(options:{id:string}&Options):Promise<Result<DocumentRecord>>;
 generate(options:{contract:ContractInput}&Options):Promise<Result<DocumentRecord>>;
 upload(options:{title:string;file:File}&Options):Promise<Result<DocumentRecord>>;
 transition(options:{id:string;decision:Decision}&Options):Promise<Result<DocumentRecord>>;
 download(options:{id:string;version:'original'|'signed'}&Options):Promise<Result<Blob>>;
 verify(options:{id:string}&Options):Promise<Result<VerificationReport>>;
}
type Request = {path:string;method?:string;body?:BodyInit|undefined;signal?:AbortSignal|undefined;json?:boolean|undefined};

export function createApiClient({basePath='/api/v1'}: {basePath?:string} = {}):DocumentApi {
 async function send<T>({path,method='GET',body,signal,json=true}:Request,parse:(data:unknown)=>T|undefined):Promise<Result<T>> {
  const headers=new Headers();
  if(json&&body)headers.set('Content-Type','application/json');
  let response:Response;
  try { response=await fetch(`${basePath}${path}`,{method,headers,...(body?{body}:{}),...(signal?{signal}:{})}); }
  catch(error){return signal?.aborted?{ok:false,error:{kind:'aborted'}}:{ok:false,error:{kind:'network',message:error instanceof Error?error.message:'Cannot reach the document service'}};}
  if(!response.ok){let message=`Request failed (${response.status})`;let requestId:string|undefined;
   try{const data:unknown=await response.json();if(typeof data==='object'&&data!==null){const record=data as Record<string,unknown>;if(typeof record.error==='string')message=record.error;if(typeof record.request_id==='string')requestId=record.request_id;}}catch{/* proxy errors may be HTML */}
   return {ok:false,error:{kind:'http',status:response.status,message,...(requestId?{requestId}:{})}};
  }
  try{const data:unknown=json?await response.json():await response.blob();const value=parse(data);return value===undefined?{ok:false,error:{kind:'invalid-response',message:'The document service returned an unexpected response.'}}:{ok:true,value};}
  catch{return {ok:false,error:{kind:'invalid-response',message:'The document service returned an unreadable response.'}};}
 }
 const record=(data:unknown)=>{const parsed=DocumentSchema.safeParse(data);return parsed.success?parsed.data:undefined;};
 const decisionRoute=(decision:Decision)=>decision.action==='submit'?'submit':decision.action==='retry'?'retry':decision.action;
 return {
  list:({limit,offset,signal})=>send({path:`/documents?limit=${limit}&offset=${offset}`,signal},data=>{const parsed=DocumentPageSchema.safeParse(data);return parsed.success?parsed.data:undefined;}),
  get:({id,signal})=>send({path:`/documents/${encodeURIComponent(id)}`,signal},record),
  generate:({contract,signal})=>send({path:'/documents/generate',method:'POST',body:JSON.stringify(contract),signal},record),
  upload:({title,file,signal})=>{const form=new FormData();form.set('title',title);form.set('file',file);return send({path:'/documents/upload',method:'POST',body:form,signal,json:false},record);},
  transition:({id,decision,signal})=>send({path:`/documents/${encodeURIComponent(id)}/${decisionRoute(decision)}`,method:'POST',body:decision.action==='approve'||decision.action==='reject'?JSON.stringify({reason:decision.reason}):'{}',signal},record),
  download:({id,version,signal})=>send({path:`/documents/${encodeURIComponent(id)}/download?version=${version}`,signal,json:false},data=>data instanceof Blob&&data.type.includes('pdf')?data:undefined),
  verify:({id,signal})=>send({path:`/documents/${encodeURIComponent(id)}/verify`,signal},data=>{const parsed=VerificationSchema.safeParse(data);return parsed.success?parsed.data:undefined;})
 };
}
