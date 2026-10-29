import type { DocumentStatus } from '../api/contracts';
export type WorkflowAction='submit'|'approve'|'reject'|'retry'|'verify';
export const isActive=(status:DocumentStatus):boolean=>['generation_queued','generating','signing_queued','signing'].includes(status);
export function allowedActions(status:DocumentStatus):readonly WorkflowAction[]{switch(status){case'draft':return['submit'];case'pending_approval':return['approve','reject'];case'signed':return['verify'];case'generation_failed':case'signing_failed':return['retry'];default:return[];}}
export const statusLabel=(status:DocumentStatus):string=>status.replaceAll('_',' ');
