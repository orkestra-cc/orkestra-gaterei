import { baseApi } from './baseApi';
import type {
  Template,
  TemplateListItem,
  TemplateListResponse,
  TemplateListParams,
  CreateTemplateInput,
  UpdateTemplateInput,
  DuplicateTemplateInput,
  GeneratePDFInput,
  GeneratePDFResponse,
  PreviewHTMLInput,
  PreviewHTMLFromContentInput,
  PreviewHTMLResponse,
  GeneratedDocumentMeta,
  ServiceStatusResponse,
  TemplateVariablesResponse,
  TemplateType,
} from '../../types/documents';

// API slice for documents module
export const documentsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // ============================================
    // Template Endpoints
    // ============================================

    // Get all templates with optional filters
    getTemplates: builder.query<TemplateListResponse, TemplateListParams>({
      query: (params = {}) => {
        const queryParams = new URLSearchParams();
        if (params.page) queryParams.append('page', params.page.toString());
        if (params.pageSize) queryParams.append('pageSize', params.pageSize.toString());
        if (params.type) queryParams.append('type', params.type);
        if (params.isDefault !== undefined) queryParams.append('isDefault', params.isDefault.toString());
        if (params.isBuiltIn !== undefined) queryParams.append('isBuiltIn', params.isBuiltIn.toString());
        if (params.isActive !== undefined) queryParams.append('isActive', params.isActive.toString());
        if (params.search) queryParams.append('search', params.search);

        const queryString = queryParams.toString();
        return {
          url: `/api/v1/documents/templates${queryString ? `?${queryString}` : ''}`,
          method: 'GET',
        };
      },
      providesTags: (result) =>
        result?.templates
          ? [
              ...result.templates.map(({ id }) => ({ type: 'DocumentTemplate' as const, id })),
              { type: 'DocumentTemplate' as const, id: 'LIST' },
            ]
          : [{ type: 'DocumentTemplate' as const, id: 'LIST' }],
    }),

    // Get a single template by ID
    getTemplate: builder.query<Template, string>({
      query: (id) => ({
        url: `/api/v1/documents/templates/${id}`,
        method: 'GET',
      }),
      transformResponse: (response: { template: Template }) => response.template,
      providesTags: (result, error, id) => [{ type: 'DocumentTemplate' as const, id }],
    }),

    // Get template variables for a type
    getTemplateVariables: builder.query<TemplateVariablesResponse, TemplateType>({
      query: (type) => ({
        url: `/api/v1/documents/templates/variables/${type}`,
        method: 'GET',
      }),
      transformResponse: (response: { variables: TemplateVariablesResponse }) => response.variables,
    }),

    // Create a new template
    createTemplate: builder.mutation<Template, CreateTemplateInput>({
      query: (data) => ({
        url: '/api/v1/documents/templates',
        method: 'POST',
        body: { template: data },
      }),
      transformResponse: (response: { template: Template }) => response.template,
      invalidatesTags: [{ type: 'DocumentTemplate', id: 'LIST' }],
    }),

    // Update an existing template
    updateTemplate: builder.mutation<Template, { id: string; data: UpdateTemplateInput }>({
      query: ({ id, data }) => ({
        url: `/api/v1/documents/templates/${id}`,
        method: 'PATCH',
        body: { template: data },
      }),
      transformResponse: (response: { template: Template }) => response.template,
      invalidatesTags: (result, error, { id }) => [
        { type: 'DocumentTemplate', id },
        { type: 'DocumentTemplate', id: 'LIST' },
      ],
    }),

    // Delete a template
    deleteTemplate: builder.mutation<{ success: boolean; message: string }, string>({
      query: (id) => ({
        url: `/api/v1/documents/templates/${id}`,
        method: 'DELETE',
      }),
      invalidatesTags: (result, error, id) => [
        { type: 'DocumentTemplate', id },
        { type: 'DocumentTemplate', id: 'LIST' },
      ],
    }),

    // Set a template as default
    setDefaultTemplate: builder.mutation<{ success: boolean; message: string }, string>({
      query: (id) => ({
        url: `/api/v1/documents/templates/${id}/default`,
        method: 'POST',
      }),
      invalidatesTags: [{ type: 'DocumentTemplate', id: 'LIST' }],
    }),

    // Duplicate a template
    duplicateTemplate: builder.mutation<Template, { id: string; data: DuplicateTemplateInput }>({
      query: ({ id, data }) => ({
        url: `/api/v1/documents/templates/${id}/duplicate`,
        method: 'POST',
        body: data,
      }),
      transformResponse: (response: { template: Template }) => response.template,
      invalidatesTags: [{ type: 'DocumentTemplate', id: 'LIST' }],
    }),

    // ============================================
    // PDF Generation Endpoints
    // ============================================

    // Generate a PDF
    generatePDF: builder.mutation<GeneratedDocumentMeta, GeneratePDFInput>({
      query: (data) => ({
        url: '/api/v1/documents/generate',
        method: 'POST',
        body: { input: data },
      }),
      transformResponse: (response: GeneratePDFResponse) => response.document,
      invalidatesTags: [{ type: 'GeneratedDocument', id: 'LIST' }],
    }),

    // Preview HTML from template
    previewHTML: builder.mutation<string, PreviewHTMLInput>({
      query: (data) => ({
        url: '/api/v1/documents/preview',
        method: 'POST',
        body: { input: data },
      }),
      transformResponse: (response: PreviewHTMLResponse) => response.html,
    }),

    // Preview HTML from raw content
    previewHTMLFromContent: builder.mutation<string, PreviewHTMLFromContentInput>({
      query: (data) => ({
        url: '/api/v1/documents/preview/content',
        method: 'POST',
        body: { input: data },
      }),
      transformResponse: (response: PreviewHTMLResponse) => response.html,
    }),

    // Get document metadata
    getDocument: builder.query<GeneratedDocumentMeta, string>({
      query: (id) => ({
        url: `/api/v1/documents/${id}`,
        method: 'GET',
      }),
      transformResponse: (response: { document: GeneratedDocumentMeta }) => response.document,
      providesTags: (result, error, id) => [{ type: 'GeneratedDocument' as const, id }],
    }),

    // Get service status
    getDocumentsServiceStatus: builder.query<ServiceStatusResponse, void>({
      query: () => ({
        url: '/api/v1/documents/status',
        method: 'GET',
      }),
    }),
  }),
});

// Export auto-generated hooks
export const {
  // Template queries
  useGetTemplatesQuery,
  useGetTemplateQuery,
  useGetTemplateVariablesQuery,
  useLazyGetTemplatesQuery,
  useLazyGetTemplateQuery,

  // Template mutations
  useCreateTemplateMutation,
  useUpdateTemplateMutation,
  useDeleteTemplateMutation,
  useSetDefaultTemplateMutation,
  useDuplicateTemplateMutation,

  // PDF generation
  useGeneratePDFMutation,
  usePreviewHTMLMutation,
  usePreviewHTMLFromContentMutation,

  // Document queries
  useGetDocumentQuery,
  useGetDocumentsServiceStatusQuery,
} = documentsApi;
