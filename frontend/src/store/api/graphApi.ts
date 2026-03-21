import { baseApi } from './baseApi';
import type {
  QueryResult,
  GraphData,
  DatabaseInfo,
  SchemaInfo,
  GDSProjection,
  AlgorithmInfo,
  AlgorithmRequest,
  CreateProjectionRequest,
  VectorIndex,
  VectorSearchRequest,
  CreateVectorIndexRequest,
  ExecuteQueryRequest,
  BrowseNodesParams,
  BrowseRelationshipsParams,
  NodeNeighborsParams,
} from '../../types/graph';

// Helper to build query params
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const buildQueryParams = (params: Record<string, any>): string => {
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      if (Array.isArray(value)) {
        value.forEach((v) => searchParams.append(key, String(v)));
      } else {
        searchParams.append(key, String(value));
      }
    }
  });
  return searchParams.toString();
};

export const graphApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // --- Core ---

    graphHealth: builder.query<{ status: string; uri: string }, void>({
      query: () => '/v1/graph/health',
    }),

    listDatabases: builder.query<{ databases: DatabaseInfo[] }, void>({
      query: () => '/v1/graph/databases',
      providesTags: ['GraphSchema'],
    }),

    getSchema: builder.query<SchemaInfo, { database?: string } | void>({
      query: (params) => {
        const qs = params ? buildQueryParams(params) : '';
        return `/v1/graph/schema${qs ? `?${qs}` : ''}`;
      },
      providesTags: ['GraphSchema'],
    }),

    // --- Query ---

    executeQuery: builder.mutation<QueryResult, ExecuteQueryRequest>({
      query: (body) => ({
        url: '/v1/graph/query',
        method: 'POST',
        body,
      }),
      invalidatesTags: (_result, _error, arg) =>
        arg.readOnly ? [] : ['GraphQuery', 'GraphSchema'],
    }),

    // --- Browse ---

    browseNodes: builder.query<QueryResult, BrowseNodesParams | void>({
      query: (params) => {
        const qs = params ? buildQueryParams(params) : '';
        return `/v1/graph/nodes${qs ? `?${qs}` : ''}`;
      },
      providesTags: ['GraphQuery'],
    }),

    browseRelationships: builder.query<QueryResult, BrowseRelationshipsParams | void>({
      query: (params) => {
        const qs = params ? buildQueryParams(params) : '';
        return `/v1/graph/relationships${qs ? `?${qs}` : ''}`;
      },
      providesTags: ['GraphQuery'],
    }),

    getNodeNeighbors: builder.query<GraphData, NodeNeighborsParams>({
      query: ({ nodeId, ...params }) => {
        const qs = buildQueryParams(params);
        return `/v1/graph/nodes/${nodeId}/neighbors${qs ? `?${qs}` : ''}`;
      },
    }),

    // --- GDS ---

    listProjections: builder.query<{ projections: GDSProjection[] }, { database?: string } | void>({
      query: (params) => {
        const qs = params ? buildQueryParams(params) : '';
        return `/v1/graph/gds/projections${qs ? `?${qs}` : ''}`;
      },
      providesTags: ['GDSProjection'],
    }),

    createProjection: builder.mutation<GDSProjection, CreateProjectionRequest>({
      query: (body) => ({
        url: '/v1/graph/gds/projections',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['GDSProjection'],
    }),

    dropProjection: builder.mutation<{ message: string }, { name: string; database?: string }>({
      query: ({ name, database }) => {
        const qs = database ? `?database=${database}` : '';
        return {
          url: `/v1/graph/gds/projections/${name}${qs}`,
          method: 'DELETE',
        };
      },
      invalidatesTags: ['GDSProjection'],
    }),

    runAlgorithm: builder.mutation<QueryResult, AlgorithmRequest>({
      query: (body) => ({
        url: '/v1/graph/gds/algorithms',
        method: 'POST',
        body,
      }),
    }),

    listAlgorithms: builder.query<{ algorithms: AlgorithmInfo[] }, void>({
      query: () => '/v1/graph/gds/algorithms',
    }),

    // --- Vector ---

    vectorSearch: builder.mutation<QueryResult, VectorSearchRequest>({
      query: (body) => ({
        url: '/v1/graph/vector/search',
        method: 'POST',
        body,
      }),
    }),

    listVectorIndexes: builder.query<{ indexes: VectorIndex[] }, { database?: string } | void>({
      query: (params) => {
        const qs = params ? buildQueryParams(params) : '';
        return `/v1/graph/vector/indexes${qs ? `?${qs}` : ''}`;
      },
      providesTags: ['VectorIndex'],
    }),

    createVectorIndex: builder.mutation<{ message: string }, CreateVectorIndexRequest>({
      query: (body) => ({
        url: '/v1/graph/vector/indexes',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['VectorIndex'],
    }),

    dropVectorIndex: builder.mutation<{ message: string }, { name: string; database?: string }>({
      query: ({ name, database }) => {
        const qs = database ? `?database=${database}` : '';
        return {
          url: `/v1/graph/vector/indexes/${name}${qs}`,
          method: 'DELETE',
        };
      },
      invalidatesTags: ['VectorIndex'],
    }),
  }),
});

export const {
  useGraphHealthQuery,
  useListDatabasesQuery,
  useGetSchemaQuery,
  useLazyGetSchemaQuery,
  useExecuteQueryMutation,
  useBrowseNodesQuery,
  useLazyBrowseNodesQuery,
  useBrowseRelationshipsQuery,
  useLazyBrowseRelationshipsQuery,
  useGetNodeNeighborsQuery,
  useLazyGetNodeNeighborsQuery,
  useListProjectionsQuery,
  useCreateProjectionMutation,
  useDropProjectionMutation,
  useRunAlgorithmMutation,
  useListAlgorithmsQuery,
  useVectorSearchMutation,
  useListVectorIndexesQuery,
  useCreateVectorIndexMutation,
  useDropVectorIndexMutation,
} = graphApi;
