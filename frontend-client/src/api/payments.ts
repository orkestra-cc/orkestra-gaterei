// Tier-2 self-service payment wrappers. Hand-typed against
// backend/internal/addons/payments/handlers/client_handler.go and
// client_routes.go. Phase 4 only needs setup-mode Checkout — the
// subscription is created with Status=active and NextBillingAt=now,
// and the renewal job (default 1h cadence) generates the first invoice
// and charges the saved card off-session. Payment-mode Checkout is
// for paying a pending invoice from the Phase 5 dashboard.
import { apiBaseURL } from '@/api/client';
import { getAccessToken } from '@/auth/tokenStore';

export interface CheckoutSessionResponse {
  sessionId: string;
  url: string;
  amountCents?: number;
  currency?: string;
  invoiceUuid?: string;
  invoiceNumber?: string;
}

export interface PaymentApiError extends Error {
  status: number;
  code?: string;
}

function err(message: string, status: number, code?: string): PaymentApiError {
  const e = new Error(message) as PaymentApiError;
  e.status = status;
  if (code) e.code = code;
  return e;
}

async function readError(res: Response, fallback: string): Promise<PaymentApiError> {
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

// Owner is polymorphic per the post-onboarding refactor: tenant-owned
// when the user picks an admin-attached organization, user-owned for
// the personal default. The backend defaults `ownerKind` to "user" when
// the body omits both ownerUuid and tenantUuid, so a personal subscribe
// only needs the success/cancel URLs.
export interface CreateSetupCheckoutInput {
  ownerKind?: 'user' | 'tenant';
  ownerUuid?: string;
  tenantUuid?: string;
  successUrl: string;
  cancelUrl: string;
}

export async function createSetupCheckoutSession(
  input: CreateSetupCheckoutInput,
): Promise<CheckoutSessionResponse> {
  const res = await authedJson('/v1/me/payments/setup-checkout-session', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  if (!res.ok) throw await readError(res, 'Failed to open checkout');
  return (await res.json()) as CheckoutSessionResponse;
}

export interface CreatePaymentCheckoutInput {
  subscriptionUuid: string;
  successUrl: string;
  cancelUrl: string;
}

export async function createPaymentCheckoutSession(
  input: CreatePaymentCheckoutInput,
): Promise<CheckoutSessionResponse> {
  const res = await authedJson('/v1/me/payments/checkout-session', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  if (!res.ok) throw await readError(res, 'Failed to open checkout');
  return (await res.json()) as CheckoutSessionResponse;
}
