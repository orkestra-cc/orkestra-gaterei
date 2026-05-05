// Tier-2 self-service subscription wrappers. Hand-typed against
// backend/internal/addons/subscriptions/handlers/subscription_handler.go
// (SelfSubscribeInput / SubscriptionResponse / Get) and routes.go's
// /v1/me/subscriptions mounts. Codegen will sharpen later.
import { apiBaseURL } from '@/api/client';
import { getAccessToken } from '@/auth/tokenStore';

export type SubscriptionStatus =
  | 'active'
  | 'past_due'
  | 'suspended'
  | 'cancelled'
  | 'expired';

export interface Subscription {
  uuid: string;
  tenantUuid: string;
  serviceUuid: string;
  tierCode: string;
  status: SubscriptionStatus;
  startedAt: string;
  currentPeriodStart: string;
  currentPeriodEnd: string;
  nextBillingAt?: string;
  cancelledAt?: string;
}

export interface SubscriptionApiError extends Error {
  status: number;
  code?: string;
}

function err(message: string, status: number, code?: string): SubscriptionApiError {
  const e = new Error(message) as SubscriptionApiError;
  e.status = status;
  if (code) e.code = code;
  return e;
}

async function readError(res: Response, fallback: string): Promise<SubscriptionApiError> {
  const body = (await res.json().catch(() => ({}))) as {
    detail?: string;
    title?: string;
    code?: string;
  };
  return err(body.detail ?? body.title ?? fallback, res.status, body.code);
}

async function authedJson(path: string, init?: RequestInit): Promise<Response> {
  const token = getAccessToken();
  return fetch(`${apiBaseURL}${path}`, {
    credentials: 'include',
    ...init,
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {}),
    },
  });
}

export interface SelfSubscribeInput {
  tenantUuid: string;
  serviceCode: string;
  tierCode: string;
}

export async function selfSubscribe(input: SelfSubscribeInput): Promise<Subscription> {
  const res = await authedJson('/v1/me/subscriptions', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  if (!res.ok) throw await readError(res, 'Failed to create subscription');
  return (await res.json()) as Subscription;
}

export async function getMySubscription(uuid: string, signal?: AbortSignal): Promise<Subscription> {
  const res = await authedJson(`/v1/me/subscriptions/${encodeURIComponent(uuid)}`, {
    method: 'GET',
    signal,
  });
  if (!res.ok) throw await readError(res, 'Failed to load subscription');
  return (await res.json()) as Subscription;
}
