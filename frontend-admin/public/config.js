// Runtime config — replaces build-time VITE_API_URL baking.
//
// Dev (Vite serves /public/* as-is): the values below are the dev defaults.
// Prod (nginx): the entrypoint at `docker-entrypoint.d/10-write-config.sh`
// overwrites this file from container env vars (ORKESTRA_API_URL,
// ORKESTRA_WS_URL, ORKESTRA_ENV, ORKESTRA_DEBUG) before nginx starts.
//
// Adding a new field: declare it on RuntimeConfig in
// src/config/environment.ts, read it via the `config` singleton, and add
// the corresponding env-var fallback in the entrypoint script. Never
// reach for `import.meta.env.VITE_*` from new code — those bake at build.
window.__ORKESTRA_CONFIG__ = {
  apiUrl: 'http://console.localhost:3000',
  wsUrl: 'ws://console.localhost:3000/ws',
  env: 'development',
  debug: true
};
