import useAdvanceTable from './useAdvanceTable';
import SubtleBadge from 'components/common/SubtleBadge';
import { useGetCompanyLookupsQuery } from 'store/api/companyApi';
import type { CompanyLookup } from 'types/company';
import { ACTIVITY_STATUS_COLORS, ACTIVITY_STATUS_LABELS } from 'types/company';
import type { BadgeColor } from 'components/common/SubtleBadge';
import { formatItalianDate } from 'types/billing';

const useCompanyLookupTable = (options?: any) => {
  const {
    data: lookupsResponse,
    isLoading,
    error,
  } = useGetCompanyLookupsQuery({ pageSize: 100 });

  const lookups = lookupsResponse?.lookups || [];

  const columns = [
    {
      accessorKey: 'companyName',
      header: 'Azienda',
      meta: {
        headerProps: { className: 'ps-2 text-900', style: { height: '46px' } },
        cellProps: { className: 'py-2 white-space-nowrap pe-3 pe-xxl-4 ps-2' },
      },
      cell: ({ row: { original } }: { row: { original: CompanyLookup } }) => (
        <div>
          <h6 className="mb-0 text-900">{original.companyName}</h6>
          <small className="text-muted font-monospace">{original.vatCode}</small>
        </div>
      ),
    },
    {
      accessorKey: 'taxCode',
      header: 'Codice Fiscale',
      meta: {
        headerProps: { className: 'text-900' },
        cellProps: { className: 'py-2 pe-4' },
      },
      cell: ({ row: { original } }: { row: { original: CompanyLookup } }) => (
        <span className="font-monospace text-900">{original.taxCode}</span>
      ),
    },
    {
      accessorKey: 'address.town',
      header: 'Sede',
      meta: {
        headerProps: { className: 'text-900' },
        cellProps: { className: 'py-2 pe-4' },
      },
      cell: ({ row: { original } }: { row: { original: CompanyLookup } }) => (
        <div>
          <div className="text-900">{original.address.town}</div>
          {original.address.province && (
            <small className="text-muted">({original.address.province})</small>
          )}
        </div>
      ),
    },
    {
      accessorKey: 'activityStatus',
      header: 'Stato',
      meta: {
        headerProps: { className: 'text-900' },
        cellProps: { className: 'fs-9 pe-4' },
      },
      cell: ({ row: { original } }: { row: { original: CompanyLookup } }) => {
        const color = (ACTIVITY_STATUS_COLORS[original.activityStatus] || 'secondary') as BadgeColor;
        const label = ACTIVITY_STATUS_LABELS[original.activityStatus] || original.activityStatus;
        return <SubtleBadge bg={color}>{label}</SubtleBadge>;
      },
    },
    {
      accessorKey: 'sdiCode',
      header: 'Codice SDI',
      meta: {
        headerProps: { className: 'text-900' },
        cellProps: { className: 'py-2 pe-4' },
      },
      cell: ({ row: { original } }: { row: { original: CompanyLookup } }) =>
        original.sdiCode ? (
          <span className="font-monospace text-900">{original.sdiCode}</span>
        ) : (
          <span className="text-muted">-</span>
        ),
    },
    {
      accessorKey: 'updatedAt',
      header: 'Ultimo Aggiornamento',
      meta: {
        headerProps: { className: 'text-900' },
        cellProps: { className: 'py-2 pe-4' },
      },
      cell: ({ row: { original } }: { row: { original: CompanyLookup } }) => (
        <span className="text-900">{formatItalianDate(original.updatedAt)}</span>
      ),
    },
  ];

  const table = useAdvanceTable({
    columns,
    data: lookups,
    isLoading,
    error,
    ...options,
  });

  return table;
};

export default useCompanyLookupTable;
