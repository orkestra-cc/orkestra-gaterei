import { baseApi } from './baseApi';
import type {
  ListAuditEventsParams,
  ListAuditEventsResponse,
} from '../../types/compliance';

// complianceApi wraps the platform-admin compliance endpoints. The backend
// gates every call behind `system.compliance.audit.read` (super_admin /
// administrator / developer inherit it). Endpoints here are the thin
// read-only projection — writes happen inside other modules via the
// iface.AuditSink backend pathway, not from the client.

export const complianceApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listAuditEvents: builder.query<ListAuditEventsResponse, ListAuditEventsParams | void>({
      query: (params) => {
        const clean = params
          ? Object.fromEntries(
              Object.entries(params).filter(
                ([, value]) => value !== undefined && value !== '' && value !== null,
              ),
            )
          : undefined;
        return {
          url: '/v1/admin/audit-events',
          method: 'GET',
          params: clean,
        };
      },
      providesTags: (result) =>
        result
          ? [
              { type: 'AuditEvent' as const, id: 'LIST' },
              ...result.items.map((ev) => ({ type: 'AuditEvent' as const, id: ev.uuid })),
            ]
          : [{ type: 'AuditEvent' as const, id: 'LIST' }],
    }),
  }),
  overrideExisting: false,
});

export const { useListAuditEventsQuery } = complianceApi;
