// Runtime config — see frontend-admin/public/config.js for the rationale.
//
// Dev (Vite): these are the defaults.
// Prod (nginx): the entrypoint at /docker-entrypoint.d/10-write-config.sh
// rewrites this file from ORKESTRA_API_BASE / ORKESTRA_STRIPE_PUBLISHABLE_KEY
// before nginx starts.
window.__ORKESTRA_CONFIG__ = {
  apiBase: "http://api.localhost:3000",
  stripePublishableKey: "",
};
