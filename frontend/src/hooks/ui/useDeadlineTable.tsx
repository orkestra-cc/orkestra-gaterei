
import useAdvanceTable from './useAdvanceTable';
import SubtleBadge from 'components/common/SubtleBadge';
import { DeadlineItem, DeadlineStatus, EntityType, DeadlineType } from 'store/api/reportsApi';
import dayjs from 'dayjs';

// Map deadline types to readable labels
const deadlineTypeLabels: Record<DeadlineType, string> = {
  revision: 'Revisione',
  scheduled_revision: 'Revisione Programmata',
  insurance: 'Assicurazione',
  car_tax: 'Bollo Auto',
  license: 'Patente',
  driver_card: 'Carta Conducente',
  cqc: 'CQC',
  adr: 'ADR',
  tachograph: 'Tachigrafo',
  medical_check: 'Visita Medica',
};

// Map entity types to readable labels
const entityTypeLabels: Record<EntityType, string> = {
  vehicle: 'Veicolo',
  user: 'Utente',
  medical: 'Visita Medica',
};

// Map status to badge colors
const statusColors: Record<DeadlineStatus, string> = {
  expired: 'danger',
  warning: 'warning',
  ok: 'success',
};

// Map status to readable labels
const statusLabels: Record<DeadlineStatus, string> = {
  expired: 'Scaduto',
  warning: 'In Scadenza',
  ok: 'OK',
};

const columns = [
  {
    accessorKey: 'entityType',
    header: 'Tipo',
    meta: {
      headerProps: { className: 'text-900', style: { width: '120px' } },
      cellProps: {
        className: 'py-2 pe-3'
      }
    },
    cell: ({ row: { original } }: { row: { original: DeadlineItem } }) => {
      return (
        <span className="fw-semibold">
          {entityTypeLabels[original.entityType]}
        </span>
      );
    }
  },
  {
    accessorKey: 'entityName',
    header: 'Nome',
    meta: {
      headerProps: {
        style: { minWidth: '14.625rem' },
        className: 'text-900'
      },
      cellProps: {
        className: 'py-2 pe-4'
      }
    },
    cell: ({ row: { original } }: { row: { original: DeadlineItem } }) => {
      return (
        <div>
          <div className="fw-semibold">{original.entityName}</div>
          {original.notes && (
            <small className="text-muted">{original.notes}</small>
          )}
        </div>
      );
    }
  },
  {
    accessorKey: 'deadlineType',
    header: 'Scadenza',
    meta: {
      headerProps: { className: 'text-900', style: { width: '180px' } },
      cellProps: {
        className: 'py-2 pe-4'
      }
    },
    cell: ({ row: { original } }: { row: { original: DeadlineItem } }) => {
      return (
        <span>
          {deadlineTypeLabels[original.deadlineType]}
        </span>
      );
    }
  },
  {
    accessorKey: 'expiryDate',
    header: 'Data Scadenza',
    meta: {
      headerProps: { className: 'text-900', style: { width: '140px' } },
      cellProps: {
        className: 'py-2 pe-4'
      }
    },
    cell: ({ row: { original } }: { row: { original: DeadlineItem } }) => {
      return (
        <span>
          {dayjs(original.expiryDate).format('DD/MM/YYYY')}
        </span>
      );
    }
  },
  {
    accessorKey: 'daysUntilExpiry',
    header: 'Giorni Rimanenti',
    meta: {
      headerProps: { className: 'text-900 text-center', style: { width: '140px' } },
      cellProps: {
        className: 'py-2 pe-4 text-center'
      }
    },
    cell: ({ row: { original } }: { row: { original: DeadlineItem } }) => {
      const days = original.daysUntilExpiry;
      let className = '';
      let text = '';

      if (days < 0) {
        className = 'text-danger fw-bold';
        text = `${Math.abs(days)} giorni fa`;
      } else if (days === 0) {
        className = 'text-danger fw-bold';
        text = 'Oggi';
      } else if (days <= 30) {
        className = 'text-warning fw-semibold';
        text = `${days} giorni`;
      } else {
        className = 'text-success';
        text = `${days} giorni`;
      }

      return <span className={className}>{text}</span>;
    }
  },
  {
    accessorKey: 'status',
    header: 'Stato',
    meta: {
      headerProps: { className: 'text-900', style: { width: '120px' } },
      cellProps: {
        className: 'py-2 pe-4'
      }
    },
    cell: ({ row: { original } }: { row: { original: DeadlineItem } }) => {
      return (
        <SubtleBadge bg={statusColors[original.status] as any}>
          {statusLabels[original.status]}
        </SubtleBadge>
      );
    }
  }
];

interface UseDeadlineTableOptions {
  data: DeadlineItem[];
  selection?: boolean;
  sortable?: boolean;
  pagination?: boolean;
  perPage?: number;
  selectionColumnWidth?: number;
}

const useDeadlineTable = (options: UseDeadlineTableOptions) => {
  const { data, ...restOptions } = options;

  const table = useAdvanceTable({
    columns,
    data: data || [],
    ...restOptions
  });

  return table;
};

export default useDeadlineTable;
