import { baseApi } from './baseApi';
import type { LogLevelsView, SetLevelBody } from '../../types/observability';

// observabilityApi wraps the platform-admin endpoints for runtime log-
// level mutation (ADR-0005 Phase F). Administrator-only on the backend.
// All mutations return the fresh LogLevelsView so the table re-renders
// without a separate refetch — the backend View() is cheap (in-memory).

export const observabilityApi = baseApi.injectEndpoints({
  endpoints: builder => ({
    getLogLevels: builder.query<LogLevelsView, void>({
      query: () => ({
        url: '/v1/admin/observability/log-levels',
        method: 'GET'
      }),
      providesTags: [{ type: 'LogLevels' as const, id: 'SNAPSHOT' }]
    }),

    setGlobalLogLevel: builder.mutation<LogLevelsView, SetLevelBody>({
      query: body => ({
        url: '/v1/admin/observability/log-levels/global',
        method: 'PUT',
        body
      }),
      invalidatesTags: [{ type: 'LogLevels' as const, id: 'SNAPSHOT' }]
    }),

    setModuleLogLevel: builder.mutation<
      LogLevelsView,
      { module: string } & SetLevelBody
    >({
      query: ({ module, level }) => ({
        url: `/v1/admin/observability/log-levels/${encodeURIComponent(module)}`,
        method: 'PUT',
        body: { level }
      }),
      invalidatesTags: [{ type: 'LogLevels' as const, id: 'SNAPSHOT' }]
    }),

    unsetModuleLogLevel: builder.mutation<LogLevelsView, { module: string }>({
      query: ({ module }) => ({
        url: `/v1/admin/observability/log-levels/${encodeURIComponent(module)}`,
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'LogLevels' as const, id: 'SNAPSHOT' }]
    }),

    resetLogLevels: builder.mutation<LogLevelsView, void>({
      query: () => ({
        url: '/v1/admin/observability/log-levels/reset',
        method: 'POST'
      }),
      invalidatesTags: [{ type: 'LogLevels' as const, id: 'SNAPSHOT' }]
    })
  })
});

export const {
  useGetLogLevelsQuery,
  useSetGlobalLogLevelMutation,
  useSetModuleLogLevelMutation,
  useUnsetModuleLogLevelMutation,
  useResetLogLevelsMutation
} = observabilityApi;
