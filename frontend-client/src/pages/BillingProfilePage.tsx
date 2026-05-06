import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import {
  type BillingProfile,
  type UpsertBillingProfileInput,
  getBillingProfile,
  putBillingProfile,
} from '@/api/billingProfile';

// Tier-2 self-service billing profile editor. Form maps to the wire
// shape on backend/internal/addons/clientbilling/handlers/me_handler.go.
// Backend Validate (services/customer_service.go) enforces:
//   - country non-empty (in either case)
//   - isCompany=true  → legalName required
//   - isCompany=false → at least one of firstName/lastName
// We mirror those rules client-side so 400s only happen on real edge
// cases, not the empty-field path. The ?next= query param is honored
// post-save so this page can chain into the subscribe flow when the
// payments handler returns 409 ("complete your billing profile").
export function BillingProfilePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [params] = useSearchParams();
  const next = params.get('next');

  const profile = useQuery({
    queryKey: ['billing-profile'],
    queryFn: ({ signal }) => getBillingProfile(signal),
  });

  const [isCompany, setIsCompany] = useState<boolean>(false);
  const [legalName, setLegalName] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [vatNumber, setVatNumber] = useState('');
  const [fiscalCode, setFiscalCode] = useState('');
  const [country, setCountry] = useState('IT');
  const [addressLine1, setAddressLine1] = useState('');
  const [addressLine2, setAddressLine2] = useState('');
  const [city, setCity] = useState('');
  const [postalCode, setPostalCode] = useState('');
  const [province, setProvince] = useState('');
  const [hydrated, setHydrated] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [savedFlash, setSavedFlash] = useState(false);

  // Hydrate the form once the GET resolves. We keep the local state the
  // source of truth from then on so the user's in-progress edits aren't
  // wiped by a refetch.
  useEffect(() => {
    if (!profile.data || hydrated) return;
    const p = profile.data;
    setIsCompany(p.isCompany);
    setLegalName(p.legalName ?? '');
    setFirstName(p.firstName ?? '');
    setLastName(p.lastName ?? '');
    setEmail(p.email ?? '');
    setVatNumber(p.vatNumber ?? '');
    setFiscalCode(p.fiscalCode ?? '');
    setCountry(p.country?.trim() ? p.country : 'IT');
    setAddressLine1(p.addressLine1 ?? '');
    setAddressLine2(p.addressLine2 ?? '');
    setCity(p.city ?? '');
    setPostalCode(p.postalCode ?? '');
    setProvince(p.province ?? '');
    setHydrated(true);
  }, [profile.data, hydrated]);

  const saveMutation = useMutation<BillingProfile, Error, UpsertBillingProfileInput>({
    mutationFn: putBillingProfile,
    onSuccess: (saved) => {
      queryClient.setQueryData(['billing-profile'], saved);
      if (next) {
        navigate(next, { replace: true });
        return;
      }
      setSavedFlash(true);
      window.setTimeout(() => setSavedFlash(false), 3000);
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    if (isCompany) {
      if (!legalName.trim()) {
        setValidationError(t('billing.errorLegalName'));
        return;
      }
    } else if (!firstName.trim() && !lastName.trim()) {
      setValidationError(t('billing.errorPersonName'));
      return;
    }
    if (!country.trim()) {
      setValidationError(t('billing.errorCountry'));
      return;
    }

    saveMutation.mutate({
      isCompany,
      legalName: legalName.trim(),
      firstName: firstName.trim(),
      lastName: lastName.trim(),
      email: email.trim(),
      vatNumber: vatNumber.trim(),
      fiscalCode: fiscalCode.trim(),
      country: country.trim().toUpperCase(),
      addressLine1: addressLine1.trim(),
      addressLine2: addressLine2.trim(),
      city: city.trim(),
      postalCode: postalCode.trim(),
      province: province.trim(),
    });
  }

  const stripeBadge = useMemo(() => {
    if (!profile.data?.hasStripe) return null;
    return (
      <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700">
        {t('billing.stripeLinked')}
      </span>
    );
  }, [profile.data?.hasStripe, t]);

  if (profile.isLoading) {
    return (
      <section className="mx-auto max-w-2xl px-6 py-16">
        <p className="text-slate-500">{t('loading')}</p>
      </section>
    );
  }

  if (profile.isError) {
    return (
      <section className="mx-auto max-w-2xl px-6 py-16 text-center">
        <h1 className="mb-3 text-3xl font-semibold tracking-tight">{t('billing.errorTitle')}</h1>
        <p className="mb-8 text-slate-600">{t('error.generic')}</p>
        <Link
          to="/account"
          className="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
        >
          {t('account.back')}
        </Link>
      </section>
    );
  }

  return (
    <section className="mx-auto max-w-2xl px-6 py-16">
      <Link to="/account" className="mb-6 inline-block text-sm text-slate-600 hover:underline">
        ← {t('account.back')}
      </Link>

      <header className="mb-8 flex items-start justify-between gap-4">
        <div>
          <h1 className="mb-2 text-3xl font-semibold tracking-tight">{t('billing.title')}</h1>
          <p className="text-slate-600">{t('billing.subtitle')}</p>
        </div>
        {stripeBadge}
      </header>

      {next && (
        <p className="mb-6 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-800">
          {t('billing.nextHint')}
        </p>
      )}

      <form onSubmit={handleSubmit} noValidate className="space-y-6">
        <fieldset className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
          <legend className="px-2 text-sm font-medium text-slate-700">
            {t('billing.typeLegend')}
          </legend>
          <div className="flex flex-col gap-2 sm:flex-row sm:gap-6">
            <label className="inline-flex items-center gap-2 text-sm text-slate-700">
              <input
                type="radio"
                name="kind"
                checked={!isCompany}
                onChange={() => setIsCompany(false)}
                className="h-4 w-4 border-slate-300 text-slate-900 focus:ring-slate-500"
              />
              {t('billing.typePerson')}
            </label>
            <label className="inline-flex items-center gap-2 text-sm text-slate-700">
              <input
                type="radio"
                name="kind"
                checked={isCompany}
                onChange={() => setIsCompany(true)}
                className="h-4 w-4 border-slate-300 text-slate-900 focus:ring-slate-500"
              />
              {t('billing.typeCompany')}
            </label>
          </div>
        </fieldset>

        <fieldset className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
          <legend className="px-2 text-sm font-medium text-slate-700">
            {t('billing.identityLegend')}
          </legend>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {isCompany ? (
              <Field
                label={t('billing.legalName')}
                value={legalName}
                onChange={setLegalName}
                required
                colSpan={2}
              />
            ) : (
              <>
                <Field
                  label={t('billing.firstName')}
                  value={firstName}
                  onChange={setFirstName}
                  required
                />
                <Field
                  label={t('billing.lastName')}
                  value={lastName}
                  onChange={setLastName}
                  required
                />
              </>
            )}
            <Field
              label={t('billing.email')}
              value={email}
              onChange={setEmail}
              type="email"
              colSpan={2}
              hint={t('billing.emailHint')}
            />
            {isCompany && (
              <Field
                label={t('billing.vatNumber')}
                value={vatNumber}
                onChange={setVatNumber}
                hint={t('billing.vatHint')}
              />
            )}
            <Field
              label={t('billing.fiscalCode')}
              value={fiscalCode}
              onChange={setFiscalCode}
              hint={isCompany ? t('billing.fiscalHintCompany') : t('billing.fiscalHintPerson')}
            />
          </div>
        </fieldset>

        <fieldset className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
          <legend className="px-2 text-sm font-medium text-slate-700">
            {t('billing.addressLegend')}
          </legend>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field
              label={t('billing.country')}
              value={country}
              onChange={setCountry}
              required
              hint={t('billing.countryHint')}
              maxLength={2}
            />
            <Field
              label={t('billing.addressLine1')}
              value={addressLine1}
              onChange={setAddressLine1}
              colSpan={2}
            />
            <Field
              label={t('billing.addressLine2')}
              value={addressLine2}
              onChange={setAddressLine2}
              colSpan={2}
            />
            <Field label={t('billing.city')} value={city} onChange={setCity} />
            <Field
              label={t('billing.postalCode')}
              value={postalCode}
              onChange={setPostalCode}
            />
            <Field
              label={t('billing.province')}
              value={province}
              onChange={setProvince}
              hint={t('billing.provinceHint')}
            />
          </div>
        </fieldset>

        {validationError && (
          <p
            className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700"
            role="alert"
          >
            {validationError}
          </p>
        )}
        {saveMutation.isError && !validationError && (
          <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">
            {saveMutation.error.message}
          </p>
        )}
        {savedFlash && (
          <p className="rounded-md bg-emerald-50 px-3 py-2 text-sm text-emerald-800" role="status">
            {t('billing.saved')}
          </p>
        )}

        <div className="flex flex-col gap-3 sm:flex-row-reverse">
          <button
            type="submit"
            disabled={saveMutation.isPending}
            className="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2.5 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:bg-slate-400 sm:w-auto"
          >
            {saveMutation.isPending ? t('billing.submitting') : t('billing.submit')}
          </button>
          <Link
            to="/account"
            className="inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 sm:w-auto"
          >
            {t('subscribe.cancel')}
          </Link>
        </div>
      </form>
    </section>
  );
}

function Field({
  label,
  value,
  onChange,
  type = 'text',
  required,
  hint,
  colSpan,
  maxLength,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  required?: boolean;
  hint?: string;
  colSpan?: 1 | 2;
  maxLength?: number;
}) {
  const span = colSpan === 2 ? 'sm:col-span-2' : '';
  const id = useMemo(() => `f-${label.replace(/\s+/g, '-').toLowerCase()}`, [label]);
  return (
    <div className={span}>
      <label htmlFor={id} className="mb-1 block text-sm font-medium text-slate-700">
        {label}
        {required && <span className="ml-1 text-red-600">*</span>}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        maxLength={maxLength}
        onChange={(e) => onChange(e.target.value)}
        className="block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
      />
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  );
}
