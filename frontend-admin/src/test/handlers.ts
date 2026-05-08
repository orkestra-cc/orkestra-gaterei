import { http, HttpResponse } from 'msw';
import type { BillingStats } from 'types/billing';

// Wildcard host so handlers match regardless of how baseApi resolves
// VITE_BACKEND_URL (e.g. console.localhost:3000 in dev, anything in tests).
export const url = (path: string) => `*${path}`;

// Default empty BillingStats — every counter at zero. Most tests can use
// this as-is; tests that need a populated response override it via
// server.use(billingStatsHandler({...})).
export const emptyBillingStats: BillingStats = {
  issuedTotal: 0,
  issuedDraft: 0,
  issuedSent: 0,
  issuedDelivered: 0,
  issuedRejected: 0,
  issuedAmount: 0,
  receivedTotal: 0,
  receivedPending: 0,
  receivedAccepted: 0,
  receivedRejected: 0,
  receivedAmount: 0,
  unprocessedNotifications: 0,
  pendingActions: 0,
  weeklyData: [],
  periodStart: '2000-01-01T00:00:00Z',
  periodEnd: '2099-12-31T23:59:59Z',
};

// Captured params from the most recent /v1/billing/stats request — used by
// regression tests that need to assert what fromDate/toDate the SPA sent.
// Reset between tests via resetCapturedRequests() in setup.ts.
export const capturedRequests = {
  billingStatsParams: null as URLSearchParams | null,
};

export const resetCapturedRequests = () => {
  capturedRequests.billingStatsParams = null;
};

export const billingStatsHandler = (
  body: BillingStats = emptyBillingStats,
) =>
  http.get(url('/v1/billing/stats'), ({ request }) => {
    capturedRequests.billingStatsParams = new URL(request.url).searchParams;
    return HttpResponse.json(body);
  });

// Default handlers used by every test unless overridden. Keep this list
// small — only stub endpoints the harness itself depends on, plus any
// chatty endpoints that components fire on mount (none yet).
export const defaultHandlers = [billingStatsHandler()];
