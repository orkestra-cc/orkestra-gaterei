import { baseApi } from './baseApi';
import type {
  ModelConfig,
  CreateModelRequest,
  UpdateModelRequest,
  TestModelResult,
  RagDocument,
  RagQueryRequest,
  RagQueryResponse,
} from '../../types/rag';

export const ragApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // --- Models ---

    createModel: builder.mutation<ModelConfig, CreateModelRequest>({
      query: (body) => ({
        url: '/v1/rag/models',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['RagModel'],
    }),

    listModels: builder.query<{ models: ModelConfig[] }, { type?: string } | void>({
      query: (params) => {
        const qs = params?.type ? `?type=${params.type}` : '';
        return `/v1/rag/models${qs}`;
      },
      providesTags: ['RagModel'],
    }),

    getModel: builder.query<ModelConfig, string>({
      query: (uuid) => `/v1/rag/models/${uuid}`,
      providesTags: ['RagModel'],
    }),

    updateModel: builder.mutation<ModelConfig, { uuid: string; body: UpdateModelRequest }>({
      query: ({ uuid, body }) => ({
        url: `/v1/rag/models/${uuid}`,
        method: 'PATCH',
        body,
      }),
      invalidatesTags: ['RagModel'],
    }),

    deleteModel: builder.mutation<{ message: string }, string>({
      query: (uuid) => ({
        url: `/v1/rag/models/${uuid}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['RagModel'],
    }),

    setDefaultModel: builder.mutation<{ message: string }, string>({
      query: (uuid) => ({
        url: `/v1/rag/models/${uuid}/default`,
        method: 'POST',
      }),
      invalidatesTags: ['RagModel'],
    }),

    testModel: builder.mutation<TestModelResult, string>({
      query: (uuid) => ({
        url: `/v1/rag/models/${uuid}/test`,
        method: 'POST',
      }),
    }),

    fetchProviderModels: builder.mutation<
      { models: { id: string; ownedBy?: string }[] },
      { provider: string; baseUrl: string; apiKey?: string }
    >({
      query: (body) => ({
        url: '/v1/rag/models/fetch',
        method: 'POST',
        body,
      }),
    }),

    // --- Documents ---

    uploadDocument: builder.mutation<RagDocument, FormData>({
      query: (formData) => ({
        url: '/v1/rag/documents',
        method: 'POST',
        body: formData,
        formData: true,
      }),
      invalidatesTags: ['RagDocument'],
    }),

    listDocuments: builder.query<{ documents: RagDocument[] }, { status?: string; isoStandard?: string } | void>({
      query: (params) => {
        const searchParams = new URLSearchParams();
        if (params?.status) searchParams.append('status', params.status);
        if (params?.isoStandard) searchParams.append('isoStandard', params.isoStandard);
        const qs = searchParams.toString();
        return `/v1/rag/documents${qs ? `?${qs}` : ''}`;
      },
      providesTags: ['RagDocument'],
    }),

    getDocument: builder.query<RagDocument, string>({
      query: (uuid) => `/v1/rag/documents/${uuid}`,
      providesTags: ['RagDocument'],
    }),

    deleteDocument: builder.mutation<{ message: string }, string>({
      query: (uuid) => ({
        url: `/v1/rag/documents/${uuid}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['RagDocument'],
    }),

    // --- RAG Query ---

    ragQuery: builder.mutation<RagQueryResponse, RagQueryRequest>({
      query: (body) => ({
        url: '/v1/rag/query',
        method: 'POST',
        body,
      }),
    }),
  }),
});

export const {
  useCreateModelMutation,
  useListModelsQuery,
  useGetModelQuery,
  useUpdateModelMutation,
  useDeleteModelMutation,
  useSetDefaultModelMutation,
  useTestModelMutation,
  useFetchProviderModelsMutation,
  useUploadDocumentMutation,
  useListDocumentsQuery,
  useGetDocumentQuery,
  useDeleteDocumentMutation,
  useRagQueryMutation,
} = ragApi;
