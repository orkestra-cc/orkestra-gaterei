import { baseApi } from './baseApi';
import { authApi } from './authApi';
import { setAccessToken } from '../slices/authSlice';
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
  tenantId: string;
  email: string;
  roles: string[];
  invitedBy: string;
  createdAt: string;
  expiresAt: string;
  // Raw token — returned only on create, never on list. Treat as
  // unrecoverable after the first response is discarded.
  token?: string;
  acceptedAt?: string | null;
}

export interface Role {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  permissions: string[];
  isSystem: boolean;
  isActive: boolean;
}

export interface UpdateRoleInput {
  name?: string;
  description?: string;
  permissions?: string[];
  isActive?: boolean;
}

export interface Binding {
  id: string;
  userUUID: string;
  tenantId: string;
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

export interface UpdateOrgAdminInput {
  name?: string;
  slug?: string;
  settings?: Record<string, string>;
}

export interface AdminOrgListItem extends Org {
  memberCount: number;
  deletedAt?: string | null;
}

export interface MembershipRecord {
  id: string;
  userUUID: string;
  tenantId: string;
  roles: string[];
  isOwner: boolean;
  invitedBy?: string;
  joinedAt: string;
  expiresAt?: string | null;
}

export const tenantApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // --- Org lifecycle ---
    listMyOrgs: builder.query<{ memberships: Membership[] }, void>({
      query: () => ({ url: '/v1/tenants', method: 'GET' }),
      providesTags: ['Membership']
    }),

    createOrg: builder.mutation<Org, CreateOrgInput>({
      query: (body) => ({ url: '/v1/tenants', method: 'POST', body }),
      invalidatesTags: ['Membership', 'Org'],
      // The caller's current JWT was issued before this tenant existed, so
      // its `mbr` claim does not list the new membership — every subsequent
      // X-Tenant-ID request for this tenant would be rejected by the
      // tenant-resolution middleware. Refresh the access token via /v1/auth/session
      // (which uses the HttpOnly refresh cookie) so the re-issued JWT carries
      // the freshly-created membership. Critical for the first-install
      // wizard, where the admin has no memberships at all when step 2 issues
      // its token.
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          await queryFulfilled;
        } catch {
          return;
        }
        try {
          const session = await dispatch(
            authApi.endpoints.getSession.initiate(undefined, { forceRefetch: true })
          ).unwrap();
          if (session?.accessToken) {
            dispatch(
              setAccessToken({
                accessToken: session.accessToken,
                expiresIn: session.expiresIn,
              })
            );
          }
        } catch {
          // Non-fatal: the user can still recover with a full page reload.
        }
      },
    }),

    getOrg: builder.query<Org, string>({
      query: (tenantId) => ({ url: `/v1/tenants/${tenantId}`, method: 'GET' }),
      providesTags: (_, __, id) => [{ type: 'Org', id }]
    }),

    updatePlan: builder.mutation<Org, { tenantId: string; body: UpdatePlanInput }>({
      query: ({ tenantId, body }) => ({ url: `/v1/tenants/${tenantId}/plan`, method: 'PATCH', body }),
      invalidatesTags: (_, __, { tenantId }) => [{ type: 'Org', id: tenantId }]
    }),

    // --- Members + invites ---
    listMembers: builder.query<{ members: Binding[] }, string>({
      query: (tenantId) => ({ url: `/v1/tenants/${tenantId}/members`, method: 'GET' }),
      providesTags: (_, __, tenantId) => [{ type: 'Membership', id: tenantId }]
    }),

    createInvite: builder.mutation<Invite, { tenantId: string; body: CreateInviteInput }>({
      query: ({ tenantId, body }) => ({ url: `/v1/tenants/${tenantId}/invites`, method: 'POST', body }),
      invalidatesTags: ['Membership']
    }),

    acceptInvite: builder.mutation<Org, { token: string }>({
      query: (body) => ({ url: '/v1/tenants/accept-invite', method: 'POST', body }),
      invalidatesTags: ['Membership']
    }),

    // --- Authz ---
    listPermissions: builder.query<{ permissions: Permission[] }, void>({
      query: () => ({ url: '/v1/authz/permissions', method: 'GET' }),
      providesTags: ['Permission']
    }),

    listRoles: builder.query<{ roles: Role[] }, string>({
      query: (tenantId) => ({ url: `/v1/tenants/${tenantId}/authz/roles`, method: 'GET' }),
      providesTags: ['Role']
    }),

    createRole: builder.mutation<Role, { tenantId: string; body: Omit<Role, 'id' | 'tenantId' | 'isSystem' | 'isActive'> }>({
      query: ({ tenantId, body }) => ({ url: `/v1/tenants/${tenantId}/authz/roles`, method: 'POST', body }),
      invalidatesTags: ['Role']
    }),

    updateRole: builder.mutation<Role, { tenantId: string; roleId: string; body: UpdateRoleInput }>({
      query: ({ tenantId, roleId, body }) => ({
        url: `/v1/tenants/${tenantId}/authz/roles/${roleId}`,
        method: 'PATCH',
        body,
      }),
      // Flipping isActive or editing permissions changes what every bound
      // user receives, so drop the effective-permissions cache as well.
      invalidatesTags: ['Role', 'EffectivePermissions'],
    }),

    deleteRole: builder.mutation<void, { tenantId: string; roleId: string }>({
      query: ({ tenantId, roleId }) => ({ url: `/v1/tenants/${tenantId}/authz/roles/${roleId}`, method: 'DELETE' }),
      // Cascades bindings on the backend — drop Binding + EffectivePermissions too.
      invalidatesTags: ['Role', 'Binding', 'EffectivePermissions'],
    }),

    listBindings: builder.query<{ bindings: Binding[] }, string>({
      query: (tenantId) => ({ url: `/v1/tenants/${tenantId}/authz/bindings`, method: 'GET' }),
      providesTags: ['Binding']
    }),

    createBinding: builder.mutation<Binding, { tenantId: string; body: { userUUID: string; roleId: string; expiresAt?: string } }>({
      query: ({ tenantId, body }) => ({ url: `/v1/tenants/${tenantId}/authz/bindings`, method: 'POST', body }),
      invalidatesTags: ['Binding', 'EffectivePermissions']
    }),

    deleteBinding: builder.mutation<void, { tenantId: string; bindingId: string }>({
      query: ({ tenantId, bindingId }) => ({ url: `/v1/tenants/${tenantId}/authz/bindings/${bindingId}`, method: 'DELETE' }),
      invalidatesTags: ['Binding', 'EffectivePermissions']
    }),

    getEffectivePermissions: builder.query<EffectivePermissions, string>({
      query: (tenantId) => ({ url: `/v1/tenants/${tenantId}/authz/me`, method: 'GET' }),
      providesTags: (_, __, tenantId) => [{ type: 'EffectivePermissions', id: tenantId }]
    }),

    // --- Platform admin tenant management (system.tenants.admin) ---
    listAllOrgsAdmin: builder.query<{ tenants: AdminOrgListItem[] }, { includeDeleted?: boolean } | void>({
      query: (arg) => ({
        url: '/v1/admin/tenants',
        method: 'GET',
        params: arg && arg.includeDeleted ? { includeDeleted: true } : undefined,
      }),
      providesTags: (result) =>
        result
          ? [
              { type: 'AdminOrg', id: 'LIST' },
              ...result.tenants.map((o) => ({ type: 'AdminOrg' as const, id: o.id })),
            ]
          : [{ type: 'AdminOrg', id: 'LIST' }],
    }),

    getOrgAdmin: builder.query<Org, string>({
      query: (tenantId) => ({ url: `/v1/admin/tenants/${tenantId}`, method: 'GET' }),
      providesTags: (_, __, id) => [{ type: 'AdminOrg', id }],
    }),

    updateOrgAdmin: builder.mutation<Org, { tenantId: string; body: UpdateOrgAdminInput }>({
      query: ({ tenantId, body }) => ({ url: `/v1/admin/tenants/${tenantId}`, method: 'PATCH', body }),
      invalidatesTags: (_, __, { tenantId }) => [
        { type: 'AdminOrg', id: tenantId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    deleteOrgAdmin: builder.mutation<void, string>({
      query: (tenantId) => ({ url: `/v1/admin/tenants/${tenantId}`, method: 'DELETE' }),
      invalidatesTags: (_, __, tenantId) => [
        { type: 'AdminOrg', id: tenantId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    updateOrgPlanAdmin: builder.mutation<Org, { tenantId: string; body: UpdatePlanInput }>({
      query: ({ tenantId, body }) => ({ url: `/v1/admin/tenants/${tenantId}/plan`, method: 'PATCH', body }),
      invalidatesTags: (_, __, { tenantId }) => [
        { type: 'AdminOrg', id: tenantId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    listOrgMembersAdmin: builder.query<{ members: MembershipRecord[] }, string>({
      query: (tenantId) => ({ url: `/v1/admin/tenants/${tenantId}/members`, method: 'GET' }),
      providesTags: (_, __, tenantId) => [{ type: 'Membership', id: tenantId }],
    }),

    removeOrgMemberAdmin: builder.mutation<void, { tenantId: string; userUUID: string }>({
      query: ({ tenantId, userUUID }) => ({
        url: `/v1/admin/tenants/${tenantId}/members/${userUUID}`,
        method: 'DELETE',
      }),
      invalidatesTags: (_, __, { tenantId }) => [
        { type: 'Membership', id: tenantId },
        { type: 'AdminOrg', id: tenantId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    listOrgInvitesAdmin: builder.query<
      { invites: Invite[] },
      { tenantId: string; includeAccepted?: boolean }
    >({
      query: ({ tenantId, includeAccepted }) => ({
        url: `/v1/admin/tenants/${tenantId}/invites`,
        method: 'GET',
        params: includeAccepted ? { includeAccepted: true } : undefined,
      }),
      providesTags: (_, __, { tenantId }) => [{ type: 'OrgInvite', id: tenantId }],
    }),

    createOrgInviteAdmin: builder.mutation<Invite, { tenantId: string; body: CreateInviteInput }>({
      query: ({ tenantId, body }) => ({ url: `/v1/admin/tenants/${tenantId}/invites`, method: 'POST', body }),
      invalidatesTags: (_, __, { tenantId }) => [{ type: 'OrgInvite', id: tenantId }],
    }),

    revokeOrgInviteAdmin: builder.mutation<void, { tenantId: string; inviteId: string }>({
      query: ({ tenantId, inviteId }) => ({
        url: `/v1/admin/tenants/${tenantId}/invites/${inviteId}`,
        method: 'DELETE',
      }),
      invalidatesTags: (_, __, { tenantId }) => [{ type: 'OrgInvite', id: tenantId }],
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
  useUpdateRoleMutation,
  useDeleteRoleMutation,
  useListBindingsQuery,
  useCreateBindingMutation,
  useDeleteBindingMutation,
  useGetEffectivePermissionsQuery,
  useListAllOrgsAdminQuery,
  useGetOrgAdminQuery,
  useUpdateOrgAdminMutation,
  useDeleteOrgAdminMutation,
  useUpdateOrgPlanAdminMutation,
  useListOrgMembersAdminQuery,
  useRemoveOrgMemberAdminMutation,
  useListOrgInvitesAdminQuery,
  useCreateOrgInviteAdminMutation,
  useRevokeOrgInviteAdminMutation,
} = tenantApi;
