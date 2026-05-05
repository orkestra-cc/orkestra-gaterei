import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { listPublicCatalog, findServiceByCode } from '@/api/catalog';
import { selfSubscribe, type Subscription } from '@/api/subscriptions';
import { createSetupCheckoutSession } from '@/api/payments';
import { useOwnedTenants } from '@/auth/memberships';
import { formatPrice } from '@/lib/format';

// Subscribe orchestration page (Phase 4):
//   1. Read service+tier from query string (set by /catalog/:code).
//   2. Resolve a tenant the caller owns (auto-select when there is only
//      one, picker when several). The backend re-checks ownership on
//      every /v1/me/* call so this is a UX hint, not a security gate.
//   3. POST /v1/me/subscriptions — the backend creates the subscription
//      with Status=active and NextBillingAt=now (entitlements granted
//      synchronously by the entitlement syncer).
//   4. POST /v1/me/payments/setup-checkout-session — opens hosted
//      Stripe Checkout in setup mode so the card is saved without
//      charging. The renewal job (default 1h cadence) generates the
//      first invoice and charges the saved card off-session via the
//      same metadata stamp the existing webhook reconciler matches on.
//   5. Redirect to Stripe via window.location.href. Return URL is
//      /subscribe/return?sub={uuid}&result=success|cancel.
//
// Design note: payment-mode Checkout requires a pending invoice to
// already exist (planner returns ErrCheckoutNoPendingInvoice → 409).
// At cold-subscribe time no invoice exists yet, so payment-mode is
// not viable here — it's the right tool for paying outstanding
// invoices from the dashboard (Phase 5). Setup mode covers the cold
// path without backend changes.
export function SubscribePage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const serviceCode = params.get('service') ?? '';
  const tierCode = params.get('tier') ?? '';

  const ownedTenants = useOwnedTenants();
  const [selectedTenant, setSelectedTenant] = useState<string>('');

  const catalog = useQuery({
    queryKey: ['catalog'],
    queryFn: ({ signal }) => listPublicCatalog(signal),
    staleTime: 60_000,
  });
  const service = useMemo(
    () => findServiceByCode(catalog.data?.items, serviceCode),
    [catalog.data, serviceCode],
  );
  const tier = service?.pricingTiers.find((t) => t.code === tierCode);
  const language = i18n.resolvedLanguage ?? 'it';

  // Auto-select the only owned tenant once memberships resolve. Skip
  // when the user has manually changed the picker so a token rotation
  // mid-session doesn't yank the selection back.
  useEffect(() => {
    if (selectedTenant) return;
    if (ownedTenants.length === 1) setSelectedTenant(ownedTenants[0].tenantUuid);
  }, [ownedTenants, selectedTenant]);

  const subscribeMutation = useMutation<
    { sub: Subscription; checkoutUrl: string },
    Error,
    { tenantUuid: string }
  >({
    mutationFn: async ({ tenantUuid }) => {
      if (!serviceCode || !tierCode) throw new Error(t('subscribe.errorMissingTier'));
      const sub = await selfSubscribe({ tenantUuid, serviceCode, tierCode });
      const origin = window.location.origin;
      const successUrl = `${origin}/subscribe/return?sub=${encodeURIComponent(sub.uuid)}&result=success`;
      const cancelUrl = `${origin}/subscribe/return?sub=${encodeURIComponent(sub.uuid)}&result=cancel`;
      const session = await createSetupCheckoutSession({
        tenantUuid,
        successUrl,
        cancelUrl,
      });
      return { sub, checkoutUrl: session.url };
    },
    onSuccess: ({ checkoutUrl }) => {
      // Stripe-hosted Checkout — full-page navigation. The browser
      // returns to /subscribe/return after the user enters card
      // details (success) or hits cancel.
      window.location.href = checkoutUrl;
    },
  });

  function handleSubscribe() {
    if (!selectedTenant) return;
    subscribeMutation.mutate({ tenantUuid: selectedTenant });
  }

  if (!serviceCode || !tierCode) {
    return (
      <section className="mx-auto max-w-2xl px-6 py-16 text-center">
        <h1 className="mb-3 text-3xl font-semibold tracking-tight">
          {t('subscribe.errorTitle')}
        </h1>
        <p className="mb-8 text-slate-600">{t('subscribe.errorMissingTier')}</p>
        <Link
          to="/catalog"
          className="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
        >
          {t('subscribe.browseCatalog')}
        </Link>
      </section>
    );
  }

  if (catalog.isLoading) {
    return (
      <section className="mx-auto max-w-2xl px-6 py-16">
        <p className="text-slate-500">{t('loading')}</p>
      </section>
    );
  }

  if (!service || !tier) {
    return (
      <section className="mx-auto max-w-2xl px-6 py-16 text-center">
        <h1 className="mb-3 text-3xl font-semibold tracking-tight">
          {t('subscribe.errorTitle')}
        </h1>
        <p className="mb-8 text-slate-600">{t('subscribe.errorTierGone')}</p>
        <Link
          to="/catalog"
          className="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
        >
          {t('subscribe.browseCatalog')}
        </Link>
      </section>
    );
  }

  if (ownedTenants.length === 0) {
    // Tier-2 self-service implicitly assumes single-owner tenants —
    // a fresh signup flow always provisions one. If the user landed
    // here with no owned tenant something is off (their tenant was
    // archived, or they signed up out-of-band). Surface a clear path
    // back through the onboarding form rather than a blank screen.
    return (
      <section className="mx-auto max-w-2xl px-6 py-16 text-center">
        <h1 className="mb-3 text-3xl font-semibold tracking-tight">
          {t('subscribe.noTenantTitle')}
        </h1>
        <p className="mb-8 text-slate-600">{t('subscribe.noTenantBody')}</p>
        <Link
          to={`/signup?service=${encodeURIComponent(serviceCode)}&tier=${encodeURIComponent(tierCode)}`}
          className="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
        >
          {t('subscribe.createOrg')}
        </Link>
      </section>
    );
  }

  const price = formatPrice(tier.amountCents, tier.currency, language);
  const setupFee = service.setupFeeCents
    ? formatPrice(service.setupFeeCents, tier.currency, language)
    : null;

  return (
    <section className="mx-auto max-w-xl px-6 py-16">
      <Link to={`/catalog/${encodeURIComponent(serviceCode)}`} className="mb-8 inline-block text-sm text-slate-600 hover:underline">
        ← {t('subscribe.back')}
      </Link>

      <h1 className="mb-2 text-3xl font-semibold tracking-tight">{t('subscribe.title')}</h1>
      <p className="mb-10 text-slate-600">{t('subscribe.subtitle')}</p>

      <div className="mb-8 rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
        <p className="mb-1 text-xs font-medium uppercase tracking-wider text-slate-500">
          {service.category ?? t('subscribe.servicePlan')}
        </p>
        <h2 className="mb-1 text-xl font-semibold text-slate-900">{service.name}</h2>
        <p className="mb-4 text-sm text-slate-600">{tier.code}</p>
        <p className="text-2xl font-semibold tracking-tight text-slate-900">
          {price} <span className="text-base font-normal text-slate-500">{t(`cycle.${tier.cycle}`)}</span>
        </p>
        {setupFee && (
          <p className="mt-2 text-sm text-slate-500">
            {t('subscribe.plusSetup', { fee: setupFee })}
          </p>
        )}
      </div>

      {ownedTenants.length > 1 && (
        <div className="mb-6">
          <label htmlFor="tenant" className="mb-1 block text-sm font-medium text-slate-700">
            {t('subscribe.tenantLabel')}
          </label>
          <select
            id="tenant"
            value={selectedTenant}
            onChange={(e) => setSelectedTenant(e.target.value)}
            className="block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
          >
            <option value="">{t('subscribe.tenantPlaceholder')}</option>
            {ownedTenants.map((m) => (
              <option key={m.tenantUuid} value={m.tenantUuid}>
                {m.tenantUuid}
              </option>
            ))}
          </select>
          <p className="mt-1 text-xs text-slate-500">{t('subscribe.tenantHint')}</p>
        </div>
      )}

      <p className="mb-6 text-sm text-slate-600">{t('subscribe.cardNotice')}</p>

      {subscribeMutation.isError && (
        <p className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">
          {subscribeMutation.error.message}
        </p>
      )}

      <button
        type="button"
        onClick={handleSubscribe}
        disabled={subscribeMutation.isPending || !selectedTenant}
        className="inline-flex w-full items-center justify-center rounded-md bg-slate-900 px-4 py-2.5 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:bg-slate-400"
      >
        {subscribeMutation.isPending ? t('subscribe.submitting') : t('subscribe.submit')}
      </button>

      <button
        type="button"
        onClick={() => navigate(-1)}
        className="mt-3 inline-flex w-full items-center justify-center rounded-md px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900"
      >
        {t('subscribe.cancel')}
      </button>
    </section>
  );
}
