import { baseApi } from './baseApi';
import type {
  RagDocument,
  RagQueryRequest,
  RagQueryResponse,
} from '../../types/rag';

export const ragApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
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
  useUploadDocumentMutation,
  useListDocumentsQuery,
  useGetDocumentQuery,
  useDeleteDocumentMutation,
  useRagQueryMutation,
} = ragApi;
