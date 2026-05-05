import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import {
  clearAccessToken,
  getAccessToken,
  refreshAccessToken,
  setAccessToken,
  subscribe,
} from '@/auth/tokenStore';
import { apiBaseURL } from '@/api/client';
import { AuthContext, type AuthState } from '@/auth/authContext';

interface AuthProviderProps {
  children: ReactNode;
}

// AuthProvider mirrors the in-memory token store into React state so any
// component can re-render when the user logs in or out. On mount it does
// one optimistic refresh attempt — if the refresh cookie is still valid
// (returning user, page reload), the SPA boots authenticated. Phase 3
// will wire the actual login form against /v1/auth/client/login; this
// provider only owns the lifecycle, not the UI.
export function AuthProvider({ children }: AuthProviderProps) {
  const [token, setToken] = useState<string | null>(getAccessToken());

  useEffect(() => subscribe(setToken), []);

  useEffect(() => {
    // One-shot bootstrap refresh — silently no-op if there is no
    // refresh cookie yet (anonymous visitor on /catalog).
    void refreshAccessToken(apiBaseURL);
  }, []);

  const signIn = useCallback((next: string) => {
    setAccessToken(next);
  }, []);

  const signOut = useCallback(async () => {
    try {
      await fetch(`${apiBaseURL}/v1/auth/client/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } finally {
      clearAccessToken();
    }
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      accessToken: token,
      isAuthenticated: token !== null,
      signIn,
      signOut,
    }),
    [token, signIn, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
