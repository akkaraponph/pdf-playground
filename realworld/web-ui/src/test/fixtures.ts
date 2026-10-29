import type { DocumentRecord } from '../api/contracts';
export function documentFixture(overrides:Partial<DocumentRecord>={}):DocumentRecord {return {id:'0123456789abcdef0123456789abcdef',title:'Vendor contract',filename:'contract.pdf',status:'draft',revision:1,created_at:'2026-10-28T10:00:00Z',updated_at:'2026-10-28T10:00:00Z',events:[],...overrides};}
