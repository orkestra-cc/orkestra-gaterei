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
  orgId: string;
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
  orgId: string;
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
  orgId: string;
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
      query: () => ({ url: '/v1/orgs', method: 'GET' }),
      providesTags: ['Membership']
    }),

    createOrg: builder.mutation<Org, CreateOrgInput>({
      query: (body) => ({ url: '/v1/orgs', method: 'POST', body }),
      invalidatesTags: ['Membership', 'Org'],
      // The caller's current JWT was issued before this org existed, so its
      // `mbr` claim does not list the new membership — every subsequent
      // X-Org-ID request for this org would be rejected by resolveCurrentOrg
      // in the auth middleware. Refresh the access token via /v1/auth/session
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

    createRole: builder.mutation<Role, { orgId: string; body: Omit<Role, 'id' | 'orgId' | 'isSystem' | 'isActive'> }>({
      query: ({ orgId, body }) => ({ url: `/v1/orgs/${orgId}/authz/roles`, method: 'POST', body }),
      invalidatesTags: ['Role']
    }),

    updateRole: builder.mutation<Role, { orgId: string; roleId: string; body: UpdateRoleInput }>({
      query: ({ orgId, roleId, body }) => ({
        url: `/v1/orgs/${orgId}/authz/roles/${roleId}`,
        method: 'PATCH',
        body,
      }),
      // Flipping isActive or editing permissions changes what every bound
      // user receives, so drop the effective-permissions cache as well.
      invalidatesTags: ['Role', 'EffectivePermissions'],
    }),

    deleteRole: builder.mutation<void, { orgId: string; roleId: string }>({
      query: ({ orgId, roleId }) => ({ url: `/v1/orgs/${orgId}/authz/roles/${roleId}`, method: 'DELETE' }),
      // Cascades bindings on the backend — drop Binding + EffectivePermissions too.
      invalidatesTags: ['Role', 'Binding', 'EffectivePermissions'],
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

    // --- Platform admin tenant management (system.tenants.admin) ---
    listAllOrgsAdmin: builder.query<{ orgs: AdminOrgListItem[] }, { includeDeleted?: boolean } | void>({
      query: (arg) => ({
        url: '/v1/admin/orgs',
        method: 'GET',
        params: arg && arg.includeDeleted ? { includeDeleted: true } : undefined,
      }),
      providesTags: (result) =>
        result
          ? [
              { type: 'AdminOrg', id: 'LIST' },
              ...result.orgs.map((o) => ({ type: 'AdminOrg' as const, id: o.id })),
            ]
          : [{ type: 'AdminOrg', id: 'LIST' }],
    }),

    getOrgAdmin: builder.query<Org, string>({
      query: (orgId) => ({ url: `/v1/admin/orgs/${orgId}`, method: 'GET' }),
      providesTags: (_, __, id) => [{ type: 'AdminOrg', id }],
    }),

    updateOrgAdmin: builder.mutation<Org, { orgId: string; body: UpdateOrgAdminInput }>({
      query: ({ orgId, body }) => ({ url: `/v1/admin/orgs/${orgId}`, method: 'PATCH', body }),
      invalidatesTags: (_, __, { orgId }) => [
        { type: 'AdminOrg', id: orgId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    deleteOrgAdmin: builder.mutation<void, string>({
      query: (orgId) => ({ url: `/v1/admin/orgs/${orgId}`, method: 'DELETE' }),
      invalidatesTags: (_, __, orgId) => [
        { type: 'AdminOrg', id: orgId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    updateOrgPlanAdmin: builder.mutation<Org, { orgId: string; body: UpdatePlanInput }>({
      query: ({ orgId, body }) => ({ url: `/v1/admin/orgs/${orgId}/plan`, method: 'PATCH', body }),
      invalidatesTags: (_, __, { orgId }) => [
        { type: 'AdminOrg', id: orgId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    listOrgMembersAdmin: builder.query<{ members: MembershipRecord[] }, string>({
      query: (orgId) => ({ url: `/v1/admin/orgs/${orgId}/members`, method: 'GET' }),
      providesTags: (_, __, orgId) => [{ type: 'Membership', id: orgId }],
    }),

    removeOrgMemberAdmin: builder.mutation<void, { orgId: string; userUUID: string }>({
      query: ({ orgId, userUUID }) => ({
        url: `/v1/admin/orgs/${orgId}/members/${userUUID}`,
        method: 'DELETE',
      }),
      invalidatesTags: (_, __, { orgId }) => [
        { type: 'Membership', id: orgId },
        { type: 'AdminOrg', id: orgId },
        { type: 'AdminOrg', id: 'LIST' },
      ],
    }),

    listOrgInvitesAdmin: builder.query<
      { invites: Invite[] },
      { orgId: string; includeAccepted?: boolean }
    >({
      query: ({ orgId, includeAccepted }) => ({
        url: `/v1/admin/orgs/${orgId}/invites`,
        method: 'GET',
        params: includeAccepted ? { includeAccepted: true } : undefined,
      }),
      providesTags: (_, __, { orgId }) => [{ type: 'OrgInvite', id: orgId }],
    }),

    createOrgInviteAdmin: builder.mutation<Invite, { orgId: string; body: CreateInviteInput }>({
      query: ({ orgId, body }) => ({ url: `/v1/admin/orgs/${orgId}/invites`, method: 'POST', body }),
      invalidatesTags: (_, __, { orgId }) => [{ type: 'OrgInvite', id: orgId }],
    }),

    revokeOrgInviteAdmin: builder.mutation<void, { orgId: string; inviteId: string }>({
      query: ({ orgId, inviteId }) => ({
        url: `/v1/admin/orgs/${orgId}/invites/${inviteId}`,
        method: 'DELETE',
      }),
      invalidatesTags: (_, __, { orgId }) => [{ type: 'OrgInvite', id: orgId }],
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
