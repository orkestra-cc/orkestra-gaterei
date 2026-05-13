// Password-reconfirm coordination between the RTK Query base query and
// the global PasswordConfirmModal. Mirror of stepUp.ts — the base query
// calls requestPasswordConfirm() when it hits a 401 with
// code="password_confirm_required" (sent by the backend's RequireStepUp
// middleware when the user has no MFA factor enrolled). The modal drives
// the user through /v1/auth/operator/me/password-confirm and calls
// completePasswordConfirm(true) on success / false on cancel.
//
// Lives as module-level state — not Redux — for the same reason as
// stepUp.ts: the base query's async flow can await a plain Promise
// without subscribing to a store from inside a fetch closure. Concurrent
// 401s share the same pending promise so a single reconfirm replays
// every paused request.

type Listener = (open: boolean) => void;

const listeners = new Set<Listener>();
let pending: Promise<boolean> | null = null;
let resolver: ((verified: boolean) => void) | null = null;

/**
 * Open the password-reconfirm modal (or join the existing one) and
 * return a promise that resolves with true when the user reconfirms,
 * false if they cancel.
 */
export function requestPasswordConfirm(): Promise<boolean> {
  if (!pending) {
    pending = new Promise<boolean>(res => {
      resolver = res;
    });
    listeners.forEach(l => l(true));
  }
  return pending;
}

/**
 * Called by PasswordConfirmModal after the user reconfirms (true) or
 * cancels (false). Resolves every waiting caller with the same outcome.
 */
export function completePasswordConfirm(verified: boolean) {
  const r = resolver;
  resolver = null;
  pending = null;
  listeners.forEach(l => l(false));
  r?.(verified);
}

/**
 * Subscribe to modal open/close events. Returns an unsubscribe function.
 */
export function subscribePasswordConfirm(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
