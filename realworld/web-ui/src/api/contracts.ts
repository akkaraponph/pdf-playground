import { z } from 'zod';

export const StatusSchema = z.enum(['generation_queued','generating','generation_failed','draft','pending_approval','signing_queued','signing','signing_failed','signed','rejected']);
export type DocumentStatus = z.infer<typeof StatusSchema>;
export const EventSchema = z.object({ action:z.string(), actor:z.string(), at:z.string(), detail:z.string().optional() });
export const ContractSchema = z.object({ title:z.string(), customer:z.string(), amount_cents:z.number().int(), currency:z.enum(['USD','THB','EUR']) });
export const DocumentSchema = z.object({ id:z.string(), title:z.string(), filename:z.string(), status:StatusSchema, revision:z.number().int(), contract:ContractSchema.optional(), reason:z.string().optional(), approved_by:z.string().optional(), last_error:z.string().optional(), original_sha256:z.string().optional(), signed_sha256:z.string().optional(), created_at:z.string(), updated_at:z.string(), events:z.array(EventSchema) });
export type DocumentRecord = z.infer<typeof DocumentSchema>;
export const DocumentPageSchema = z.object({ items:z.array(DocumentSchema), total:z.number().int(), offset:z.number().int(), limit:z.number().int() });
export type DocumentPage = z.infer<typeof DocumentPageSchema>;
export type ContractInput = z.infer<typeof ContractSchema>;
export const VerificationSchema = z.object({ valid:z.boolean(), signatures:z.number().int(), trusted:z.boolean(), revocation_checked:z.boolean(), timestamp_checked:z.boolean() });
export type VerificationReport = z.infer<typeof VerificationSchema>;
