// Shared type definitions for the compliance module. Mirrors the shapes
// the backend emits from `GET /v1/admin/audit-events` (see
// backend/internal/addons/compliance/handlers/admin_handler.go).

export type AuditActorType = 'user' | 'system' | 'anonymous';

export type AuditOutcome = 'success' | 'failure' | 'denied';

export interface AuditEvent {
  uuid: string;
  tenantId?: string;
  tenantKind?: string;
  actorUserId?: string;
  actorEmail?: string;
  actorType: AuditActorType;
  action: string;
  resourceType?: string;
  resourceId?: string;
  outcome: AuditOutcome;
  ipAddress?: string;
  userAgent?: string;
  metadata?: Record<string, unknown>;
  timestamp: string;
}

export interface ListAuditEventsParams {
  tenantId?: string;
  actorUserId?: string;
  /** Exact action match, e.g. `auth.login.succeeded` */
  action?: string;
  /** Action family prefix, e.g. `auth.` to match every `auth.*` event */
  actionPrefix?: string;
  resourceType?: string;
  resourceId?: string;
  outcome?: AuditOutcome;
  /** RFC3339 lower bound (inclusive) */
  since?: string;
  /** RFC3339 upper bound (inclusive) */
  until?: string;
  /** 1–500, default 50 */
  limit?: number;
  /** 0-based, default 0 */
  offset?: number;
}

export interface ListAuditEventsResponse {
  items: AuditEvent[];
  total: number;
  limit: number;
  offset: number;
}
