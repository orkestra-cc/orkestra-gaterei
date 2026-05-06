// Tier-2 self-service billing-profile wrappers. Hand-typed against
// backend/internal/addons/clientbilling/handlers/me_handler.go's
// BillingProfile + GetBillingProfile / PutBillingProfile. The backend
// returns a 200 with all-blank fields when no profile exists yet — the
// SPA detects "missing profile" by checking that the persisted payload
// has no identity fields filled in (see hasBillingProfile below) instead
// of relying on a 404, which the contract deliberately avoids.
import { apiBaseURL } from '@/api/client';
import { getAccessToken } from '@/auth/tokenStore';

export interface BillingProfile {
  legalName?: string;
  firstName?: string;
  lastName?: string;
  email?: string;
  vatNumber?: string;
  fiscalCode?: string;
  country?: string;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  postalCode?: string;
  province?: string;
  isCompany: boolean;
  hasStripe: boolean;
}

export interface BillingProfileApiError extends Error {
  status: number;
  code?: string;
}

function err(message: string, status: number, code?: string): BillingProfileApiError {
  const e = new Error(message) as BillingProfileApiError;
  e.status = status;
  if (code) e.code = code;
  return e;
}

async function readError(res: Response, fallback: string): Promise<BillingProfileApiError> {
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

export async function getBillingProfile(signal?: AbortSignal): Promise<BillingProfile> {
  const res = await authedJson('/v1/me/billing-profile', { method: 'GET', signal });
  if (!res.ok) throw await readError(res, 'Failed to load billing profile');
  return (await res.json()) as BillingProfile;
}

export interface UpsertBillingProfileInput {
  legalName?: string;
  firstName?: string;
  lastName?: string;
  email?: string;
  vatNumber?: string;
  fiscalCode?: string;
  country?: string;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  postalCode?: string;
  province?: string;
  isCompany: boolean;
}

export async function putBillingProfile(
  input: UpsertBillingProfileInput,
): Promise<BillingProfile> {
  const res = await authedJson('/v1/me/billing-profile', {
    method: 'PUT',
    body: JSON.stringify(input),
  });
  if (!res.ok) throw await readError(res, 'Failed to save billing profile');
  return (await res.json()) as BillingProfile;
}

// hasBillingProfile mirrors the backend's Upsert invariant: legalName is
// mandatory when isCompany=true, firstName OR lastName when isCompany=false,
// and country is mandatory in either case. The GET endpoint returns a 200
// with all-blank fields when no row exists yet, so we infer "no profile"
// from those required fields being empty.
export function hasBillingProfile(p: BillingProfile | null | undefined): boolean {
  if (!p) return false;
  if (!p.country?.trim()) return false;
  if (p.isCompany) {
    return !!p.legalName?.trim();
  }
  return !!(p.firstName?.trim() || p.lastName?.trim());
}
