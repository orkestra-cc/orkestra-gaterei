import { useEffect, useRef } from 'react';
import { moduleCatalog } from 'modules';
import { useGetModulesQuery } from 'store/api/moduleApi';

/**
 * Injects RTK Query API slices for all enabled modules.
 * Call once at app root (e.g. App.tsx).
 *
 * For admin users: only injects slices for modules the backend reports as
 * enabled/running. For non-admin users: the query fails with 403, so this
 * hook is a no-op — pages import their API slices directly as a fallback.
 */
export function useModuleApiInjection() {
  const { data: modules } = useGetModulesQuery();
  const injectedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!modules) return;

    const enabled = modules
      .filter((m) => m.enabled && m.status === 'running')
      .map((m) => m.moduleName);

    for (const name of enabled) {
      if (injectedRef.current.has(name)) continue;
      const manifest = moduleCatalog[name];
      if (manifest?.injectApi) {
        manifest.injectApi();
        injectedRef.current.add(name);
      }
    }
  }, [modules]);
}
