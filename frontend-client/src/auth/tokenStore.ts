// Module-scoped in-memory access token. Stored outside React state so the
// API client middleware can read it synchronously without dragging React
// context into the fetch path. The refresh cookie is httpOnly and lives
// on the API origin (Domain=api.localhost in dev, app.orkestra.cc in
// staging) — the SPA never sees it directly; it just calls the refresh
// endpoint and stores the resulting access token here.

let accessToken: string | null = null;
const subscribers = new Set<(token: string | null) => void>();

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
  for (const fn of subscribers) fn(token);
}

export function clearAccessToken(): void {
  setAccessToken(null);
}

export function subscribe(fn: (token: string | null) => void): () => void {
  subscribers.add(fn);
  return () => {
    subscribers.delete(fn);
  };
}

// In-flight refresh promise — coalesces concurrent 401s into a single
// /v1/auth/client/refresh-cookie call so a burst of parallel requests
// can't trigger N refresh attempts.
let inflightRefresh: Promise<string | null> | null = null;

export async function refreshAccessToken(apiBase: string): Promise<string | null> {
  if (inflightRefresh) return inflightRefresh;
  inflightRefresh = (async () => {
    try {
      const res = await fetch(`${apiBase}/v1/auth/client/refresh-cookie`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        clearAccessToken();
        return null;
      }
      // The refresh-cookie response shape comes from the backend's
      // RefreshCookieResponse — we read accessToken from the body.
      // Codegen will sharpen the type once src/api/openapi.gen.ts has
      // the operation typed; for now we accept either { accessToken }
      // or { token } until the contract is locked in Phase 3.
      const body = (await res.json().catch(() => ({}))) as {
        accessToken?: string;
        token?: string;
      };
      const fresh = body.accessToken ?? body.token ?? null;
      setAccessToken(fresh);
      return fresh;
    } finally {
      inflightRefresh = null;
    }
  })();
  return inflightRefresh;
}
