import { baseApi } from './baseApi';

// --- Types ---

interface SkillRequest {
  url: string;
  locale?: string;
  context?: string;
}

interface SkillResponse {
  skill: string;
  result: any;
  inputTokens: number;
  outputTokens: number;
  latencyMs: number;
  modelUsed: string;
}

interface ProspectRequest {
  url: string;
  locale?: string;
}

interface ProspectJobResponse {
  jobId: string;
  streamUrl: string;
}

interface QuickProspectResponse {
  score: number;
  grade: string;
  companyName: string;
  summary: string;
  findings: any;
  inputTokens: number;
  outputTokens: number;
  latencyMs: number;
}

interface Job {
  uuid: string;
  createdBy: string;
  companyUrl: string;
  locale: string;
  status: string;
  phases: Array<{ name: string; status: string; startedAt?: string; completedAt?: string }>;
  agentResults?: any[];
  totalScore?: number;
  grade?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

interface JobListResponse {
  jobs: Job[];
  total: number;
  page: number;
  pageSize: number;
}

// --- API Slice ---

export const salesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Individual skills
    runSkill: builder.mutation<SkillResponse, { skill: string } & SkillRequest>({
      query: ({ skill, ...body }) => ({
        url: `v1/sales/${skill}`,
        method: 'POST',
        body,
      }),
    }),

    // Full async prospect
    createProspectJob: builder.mutation<ProspectJobResponse, ProspectRequest>({
      query: (body) => ({
        url: 'v1/sales/prospect',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['Sales'],
    }),

    // Quick sync prospect
    runQuickProspect: builder.mutation<QuickProspectResponse, ProspectRequest>({
      query: (body) => ({
        url: 'v1/sales/prospect/quick',
        method: 'POST',
        body,
      }),
    }),

    // Jobs
    listSalesJobs: builder.query<JobListResponse, { page?: number; pageSize?: number; status?: string }>({
      query: (params) => ({
        url: 'v1/sales/jobs',
        params,
      }),
      providesTags: ['Sales'],
    }),

    getSalesJob: builder.query<Job, string>({
      query: (uuid) => `v1/sales/jobs/${uuid}`,
      providesTags: ['Sales'],
    }),

    cancelSalesJob: builder.mutation<void, string>({
      query: (uuid) => ({
        url: `v1/sales/jobs/${uuid}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['Sales'],
    }),

    retrySalesJob: builder.mutation<{ jobId: string; streamUrl: string }, string>({
      query: (uuid) => ({
        url: `v1/sales/jobs/${uuid}/retry`,
        method: 'POST',
      }),
      invalidatesTags: ['Sales'],
    }),

    // Reports
    listSalesReports: builder.query<ReportListResponse, { page?: number; pageSize?: number }>({
      query: (params) => ({
        url: 'v1/sales/reports',
        params,
      }),
      providesTags: ['Sales'],
    }),

    getSalesReport: builder.query<Report, string>({
      query: (uuid) => `v1/sales/reports/${uuid}`,
      providesTags: ['Sales'],
    }),

    generateSalesReport: builder.mutation<Report, string>({
      query: (jobUuid) => ({
        url: `v1/sales/reports/generate/${jobUuid}`,
        method: 'POST',
      }),
      invalidatesTags: ['Sales'],
    }),

    // Settings
    getSalesSettings: builder.query<SalesSettings, void>({
      query: () => 'v1/sales/settings',
      providesTags: ['Sales'],
    }),

    updateSalesSettings: builder.mutation<SalesSettings, Partial<SalesSettings>>({
      query: (body) => ({
        url: 'v1/sales/settings',
        method: 'PATCH',
        body,
      }),
      invalidatesTags: ['Sales'],
    }),
  }),
});

export interface Report {
  uuid: string;
  jobUuid: string;
  createdBy: string;
  companyUrl: string;
  companyName: string;
  score: number;
  grade: string;
  contentMd?: string;
  agentData?: any;
  createdAt: string;
}

export interface ReportListResponse {
  reports: Report[];
  total: number;
  page: number;
  pageSize: number;
}

export interface SalesSettings {
  uuid: string;
  userUuid: string;
  modelUuid?: string;
  temperature?: number;
  maxTokens?: number;
  locale?: string;
  createdAt?: string;
  updatedAt?: string;
}

export const {
  useRunSkillMutation,
  useCreateProspectJobMutation,
  useRunQuickProspectMutation,
  useListSalesJobsQuery,
  useGetSalesJobQuery,
  useCancelSalesJobMutation,
  useRetrySalesJobMutation,
  useListSalesReportsQuery,
  useGetSalesReportQuery,
  useGenerateSalesReportMutation,
  useGetSalesSettingsQuery,
  useUpdateSalesSettingsMutation,
} = salesApi;
