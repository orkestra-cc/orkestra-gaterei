/**
 * Runtime environment configuration for the operator console.
 *
 * Values come from `window.__ORKESTRA_CONFIG__`, populated at container start
 * by the nginx entrypoint (or in dev by `public/config.js`). Build-time
 * `import.meta.env.VITE_*` is consulted only as a fallback for `npm run
 * dev`/`vite build` invocations that bypass `public/config.js` (Vitest,
 * SSR scratch builds, etc.).
 *
 * Use the exported `config` singleton everywhere — never reach for
 * `import.meta.env.VITE_*` from new code. The point of moving to runtime
 * config was to make a single published image work in dev / staging /
 * prod without rebuilding.
 */

export type Environment = 'development' | 'staging' | 'production';

interface RuntimeConfig {
  apiUrl?: string;
  wsUrl?: string;
  env?: Environment | string;
  debug?: boolean;
}

declare global {
  interface Window {
    __ORKESTRA_CONFIG__?: RuntimeConfig;
  }
}

interface EnvironmentConfig {
  /** Current environment name */
  env: Environment;
  /** Backend API URL */
  apiUrl: string;
  /** WebSocket URL */
  wsUrl: string;
  /** Debug mode enabled */
  debug: boolean;
  /** True if running in production */
  isProduction: boolean;
  /** True if running in staging */
  isStaging: boolean;
  /** True if running in development */
  isDevelopment: boolean;
  /** True for staging and production (production-like behavior) */
  isProductionLike: boolean;
}

function readRuntime(): RuntimeConfig {
  if (typeof window === 'undefined') return {};
  return window.__ORKESTRA_CONFIG__ ?? {};
}

function pickEnv(value: unknown): Environment {
  if (value === 'staging' || value === 'production') return value;
  return 'development';
}

function pickString(
  ...candidates: Array<string | undefined>
): string | undefined {
  for (const c of candidates) {
    if (typeof c === 'string' && c.length > 0) return c;
  }
  return undefined;
}

function createConfig(): EnvironmentConfig {
  const runtime = readRuntime();
  const env = pickEnv(runtime.env ?? import.meta.env.VITE_ENV);

  const apiUrl =
    pickString(
      runtime.apiUrl,
      import.meta.env.VITE_API_URL,
      import.meta.env.VITE_BACKEND_URL
    ) ?? 'http://console.localhost:3000';

  const wsUrl =
    pickString(runtime.wsUrl, import.meta.env.VITE_WS_URL) ??
    'ws://console.localhost:3000/ws';

  const debug =
    typeof runtime.debug === 'boolean'
      ? runtime.debug
      : import.meta.env.VITE_DEBUG === 'true';

  return {
    env,
    apiUrl,
    wsUrl,
    debug,
    isProduction: env === 'production',
    isStaging: env === 'staging',
    isDevelopment: env === 'development',
    isProductionLike: env === 'production' || env === 'staging'
  };
}

/** Environment configuration singleton */
export const config = createConfig();

/** Check if running in development environment. */
export function isDevelopment(): boolean {
  return config.isDevelopment;
}

/** Check if running in production environment. */
export function isProduction(): boolean {
  return config.isProduction;
}

/** Check if running in staging environment. */
export function isStaging(): boolean {
  return config.isStaging;
}

/**
 * Check if running in a production-like environment (staging or production).
 * Use this for security and behavior that should match production.
 */
export function isProductionLike(): boolean {
  return config.isProductionLike;
}

/** Get the current environment name. */
export function getEnv(): Environment {
  return config.env;
}

export default config;
