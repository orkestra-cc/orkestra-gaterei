import { useTranslation } from 'react-i18next';

// Phase 1 placeholder. Phase 2 wires this to GET /v1/public/catalog/services
// via the codegen'd OpenAPI client (src/api/client.ts).
export function CatalogPage() {
  const { t } = useTranslation();
  return (
    <section className="mx-auto max-w-6xl px-6 py-16">
      <h2 className="mb-2 text-3xl font-semibold tracking-tight">
        {t('nav.catalog')}
      </h2>
      <p className="text-slate-600">
        {/* Intentionally untranslated — placeholder for Phase 2. */}
        Service catalog will land in Phase 2 (anonymous catalog + signup).
      </p>
    </section>
  );
}
