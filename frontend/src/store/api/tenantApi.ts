import { baseApi } from './baseApi';
import type { Membership, EffectivePermissions } from '../slices/tenantSlice';

export interface Org {
  id: string;
  name: string;
  slug: string;
  ownerUserUUID: string;
  plan: string;
  features: string[];
  settings?: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

export interface CreateOrgInput {
  name: string;
  slug: string;
  plan?: string;
}

export interface UpdatePlanInput {
  plan: string;
  features: string[];
}

export interface CreateInviteInput {
  email: string;
  roles: string[];
}

export interface Invite {
  id: string;
  orgId: string;
  email: string;
  roles: string[];
  invitedBy: string;
  createdAt: string;
  expiresAt: string;
}

export interface Role {
  id: string;
  orgId: string;
  name: string;
  description: string;
  permissions: string[];
  isSystem: boolean;
}

export interface Binding {
  id: string;
  userUUID: string;
  orgId: string;
  roleId: string;
  roleName: string;
  grantedBy?: string;
  grantedAt: string;
  expiresAt?: string | null;
}

export interface Permission {
  key: string;
  module: string;
  description: string;
  system: boolean;
}

export const tenantApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // --- Org lifecycle ---
    listMyOrgs: builder.query<{ memberships: Membership[] }, void>({
      query: () => ({ url: '/v1/orgs', method: 'GET' }),
      providesTags: ['Membership']
    }),

    createOrg: builder.mutation<Org, CreateOrgInput>({
      query: (body) => ({ url: '/v1/orgs', method: 'POST', body }),
      invalidatesTags: ['Membership', 'Org']
    }),

    getOrg: builder.query<Org, string>({
      query: (orgId) => ({ url: `/v1/orgs/${orgId}`, method: 'GET' }),
      providesTags: (_, __, id) => [{ type: 'Org', id }]
    }),

    updatePlan: builder.mutation<Org, { orgId: string; body: UpdatePlanInput }>({
      query: ({ orgId, body }) => ({ url: `/v1/orgs/${orgId}/plan`, method: 'PATCH', body }),
      invalidatesTags: (_, __, { orgId }) => [{ type: 'Org', id: orgId }]
    }),

    // --- Members + invites ---
    listMembers: builder.query<{ members: Binding[] }, string>({
      query: (orgId) => ({ url: `/v1/orgs/${orgId}/members`, method: 'GET' }),
      providesTags: (_, __, orgId) => [{ type: 'Membership', id: orgId }]
    }),

    createInvite: builder.mutation<Invite, { orgId: string; body: CreateInviteInput }>({
      query: ({ orgId, body }) => ({ url: `/v1/orgs/${orgId}/invites`, method: 'POST', body }),
      invalidatesTags: ['Membership']
    }),

    acceptInvite: builder.mutation<Org, { token: string }>({
      query: (body) => ({ url: '/v1/orgs/accept-invite', method: 'POST', body }),
      invalidatesTags: ['Membership']
    }),

    // --- Authz ---
    listPermissions: builder.query<{ permissions: Permission[] }, void>({
      query: () => ({ url: '/v1/authz/permissions', method: 'GET' }),
      providesTags: ['Permission']
    }),

    listRoles: builder.query<{ roles: Role[] }, string>({
      query: (orgId) => ({ url: `/v1/orgs/${orgId}/authz/roles`, method: 'GET' }),
      providesTags: ['Role']
    }),

    createRole: builder.mutation<Role, { orgId: string; body: Omit<Role, 'id' | 'orgId' | 'isSystem'> }>({
      query: ({ orgId, body }) => ({ url: `/v1/orgs/${orgId}/authz/roles`, method: 'POST', body }),
      invalidatesTags: ['Role']
    }),

    deleteRole: builder.mutation<void, { orgId: string; roleId: string }>({
      query: ({ orgId, roleId }) => ({ url: `/v1/orgs/${orgId}/authz/roles/${roleId}`, method: 'DELETE' }),
      invalidatesTags: ['Role']
    }),

    listBindings: builder.query<{ bindings: Binding[] }, string>({
      query: (orgId) => ({ url: `/v1/orgs/${orgId}/authz/bindings`, method: 'GET' }),
      providesTags: ['Binding']
    }),

    createBinding: builder.mutation<Binding, { orgId: string; body: { userUUID: string; roleId: string; expiresAt?: string } }>({
      query: ({ orgId, body }) => ({ url: `/v1/orgs/${orgId}/authz/bindings`, method: 'POST', body }),
      invalidatesTags: ['Binding', 'EffectivePermissions']
    }),

    deleteBinding: builder.mutation<void, { orgId: string; bindingId: string }>({
      query: ({ orgId, bindingId }) => ({ url: `/v1/orgs/${orgId}/authz/bindings/${bindingId}`, method: 'DELETE' }),
      invalidatesTags: ['Binding', 'EffectivePermissions']
    }),

    getEffectivePermissions: builder.query<EffectivePermissions, string>({
      query: (orgId) => ({ url: `/v1/orgs/${orgId}/authz/me`, method: 'GET' }),
      providesTags: (_, __, orgId) => [{ type: 'EffectivePermissions', id: orgId }]
    }),
  }),
  overrideExisting: false,
});

export const {
  useListMyOrgsQuery,
  useCreateOrgMutation,
  useGetOrgQuery,
  useUpdatePlanMutation,
  useListMembersQuery,
  useCreateInviteMutation,
  useAcceptInviteMutation,
  useListPermissionsQuery,
  useListRolesQuery,
  useCreateRoleMutation,
  useDeleteRoleMutation,
  useListBindingsQuery,
  useCreateBindingMutation,
  useDeleteBindingMutation,
  useGetEffectivePermissionsQuery,
} = tenantApi;
